"""Publisher (runs in the `pilot-publish` namespace only; research R8, FR-013-FR-016).

Inputs  /in/pr.json   {repo, head, base, base_revision, title, body, run_id, task, scm?}
        /in/change.bundle
        env PILOT_SCM_TOKEN (one repository only), HTTPS_PROXY (scm-egress)
Output  stdout frame with result.json {number, url, head, draft, author, author_id?, token_login, created}

037: pr.json's optional "scm" block {kind, api, git_base, ca_file?} names the Git host; absent
means GitHub with today's behaviour. Bitbucket Cloud opens a native draft pull request with a
repository access token (git user x-token-auth); the token can't read its own identity, so the
PR's author is the publisher identity.

There is deliberately no merge code path in this file.
"""
from __future__ import annotations

import json
import os
import ssl
import subprocess
import sys
import tempfile
import urllib.parse
import urllib.request

# 025: the integration tests point this at a fake GitHub (GITHUB_API, test mode only).
API = (os.environ.get("GITHUB_API") or "https://api.github.com").rstrip("/")


class GitHub:
    def __init__(self, token: str, api: str = API):
        self._token = token
        self.api = api

    def _req(self, method: str, path: str, body=None):
        data = json.dumps(body).encode() if body is not None else None
        req = urllib.request.Request(self.api + path, data=data, method=method, headers={
            "Authorization": f"Bearer {self._token}", "Accept": "application/vnd.github+json",
            "X-GitHub-Api-Version": "2022-11-28", "User-Agent": "specfoundry-pilot-publisher"})
        if data is not None:
            req.add_header("Content-Type", "application/json")
        with urllib.request.urlopen(req, timeout=60) as r:
            return json.loads(r.read() or b"null")

    def user_login(self) -> str:
        return self._req("GET", "/user")["login"]

    def open_prs(self, repo: str, owner: str, head: str) -> list:
        q = urllib.parse.urlencode({"head": f"{owner}:{head}", "state": "open"})
        return self._req("GET", f"/repos/{repo}/pulls?{q}")

    def create_draft_pr(self, repo: str, head: str, base: str, title: str, body: str) -> dict:
        return self._req("POST", f"/repos/{repo}/pulls",
                         {"title": title, "head": head, "base": base, "body": body, "draft": True})


class Bitbucket:
    """Bitbucket Cloud /2.0 with a repository access token (037 contracts/publisher.md). Pull
    requests come back in the GitHub shape publish() reads, plus author_id (the bot's {uuid})."""

    def __init__(self, token: str, api: str, ca_file: str = ""):
        self._token = token
        self.api = api.rstrip("/")
        self._ctx = ssl.create_default_context(cafile=ca_file) if ca_file else None

    def _req(self, method: str, path: str, body=None):
        data = json.dumps(body).encode() if body is not None else None
        req = urllib.request.Request(self.api + path, data=data, method=method, headers={
            "Authorization": f"Bearer {self._token}", "Accept": "application/json", "User-Agent": "chalkeus-publisher"})
        if data is not None:
            req.add_header("Content-Type", "application/json")
        with urllib.request.urlopen(req, timeout=60, context=self._ctx) as r:
            return json.loads(r.read() or b"null")

    def user_login(self) -> str:
        return ""  # GET /user refuses repository access tokens; the PR's author is the identity

    @staticmethod
    def _shape(pr: dict) -> dict:
        a = pr.get("author") or {}
        return {"number": pr["id"], "html_url": pr["links"]["html"]["href"], "head": {"ref": pr["source"]["branch"]["name"]},
                "draft": bool(pr.get("draft")), "user": {"login": a.get("nickname") or a.get("display_name") or ""},
                "author_id": a.get("uuid") or ""}

    def open_prs(self, repo: str, owner: str, head: str) -> list:
        seen, out = set(), []
        for q in (f'source.branch.name="{head}" AND state="OPEN"', f'source.branch.name="{head}" AND state="OPEN" AND draft=true'):
            page = self._req("GET", f"/repositories/{repo}/pullrequests?" + urllib.parse.urlencode({"q": q, "pagelen": 50}))
            for pr in (page or {}).get("values", []):
                if pr["id"] not in seen:
                    seen.add(pr["id"])
                    out.append(self._shape(pr))
        return out

    def create_draft_pr(self, repo: str, head: str, base: str, title: str, body: str) -> dict:
        return self._shape(self._req("POST", f"/repositories/{repo}/pullrequests", {
            "title": title, "description": body, "draft": True, "close_source_branch": False,
            "source": {"branch": {"name": head}}, "destination": {"branch": {"name": base}}}))


GIT_USERS = {"github": "x-access-token", "bitbucket": "x-token-auth", "gitlab": "oauth2"}


def _summary(pr: dict, created: bool, token_login: str) -> dict:
    s = {"number": pr["number"], "url": pr["html_url"], "head": pr["head"]["ref"], "draft": bool(pr.get("draft")),
         "author": pr["user"]["login"], "token_login": token_login or pr["user"]["login"], "created": created}
    if pr.get("author_id"):
        s["author_id"] = pr["author_id"]
    return s


def publish(gh, push, spec: dict) -> dict:
    """Idempotent: an open PR for the head branch is recorded, never duplicated."""
    repo, head = spec["repo"], spec["head"]
    owner = repo.split("/")[0]
    login = spec.get("publisher_login") or gh.user_login()  # App tokens cannot call GET /user (004)
    existing = gh.open_prs(repo, owner, head)
    if existing:
        return _summary(existing[0], False, login)
    push(spec)
    existing = gh.open_prs(repo, owner, head)  # a concurrent/previous call may have won
    if existing:
        return _summary(existing[0], False, login)
    pr = gh.create_draft_pr(repo, head, spec["base"], spec["title"], spec["body"])
    return _summary(pr, True, login)


def git_push(spec: dict, bundle: str = "/in/change.bundle") -> None:
    """Push base_revision + bundle to the head branch. The token reaches git only via GIT_ASKPASS."""
    host = spec.get("scm") or {}
    kind = host.get("kind") or "github"
    with tempfile.TemporaryDirectory() as t:
        askpass = os.path.join(t, "askpass.sh")
        with open(askpass, "w") as f:
            f.write('#!/bin/sh\ncase "$1" in Username*) echo ' + GIT_USERS[kind] + ';; *) printf %s "$PILOT_SCM_TOKEN";; esac\n')
        os.chmod(askpass, 0o700)
        env = {**os.environ, "GIT_ASKPASS": askpass, "GIT_TERMINAL_PROMPT": "0",
               "https_proxy": os.environ.get("HTTPS_PROXY", ""), "HOME": t}
        if host.get("ca_file"):
            env["GIT_SSL_CAINFO"] = host["ca_file"]
        if kind == "github":
            url = f"{os.environ.get('GIT_BASE') or 'https://github.com/'}{spec['repo']}.git"  # 025: GIT_BASE in test mode
        else:
            url = f"{host['git_base'].rstrip('/')}/{spec['repo']}.git"
        repo = os.path.join(t, "repo")

        def g(*a):
            subprocess.run(["git", *a], cwd=repo if os.path.isdir(repo) else t, env=env, check=True,
                           stdout=subprocess.DEVNULL, stderr=sys.stderr)
        g("clone", "--quiet", "--branch", spec["base"], url, repo)
        g("cat-file", "-e", spec["base_revision"] + "^{commit}")
        g("fetch", "--quiet", bundle, "HEAD:refs/pilot/graded")
        g("push", "--quiet", "--force", "origin", f"refs/pilot/graded:refs/heads/{spec['head']}")


def client(spec: dict, token: str):
    """The host's client from pr.json's scm block (037); absent or github: GitHub as before."""
    host = spec.get("scm") or {}
    kind = host.get("kind") or "github"
    if kind == "github":
        return GitHub(token)
    if kind == "bitbucket":
        return Bitbucket(token, host["api"], host.get("ca_file", ""))
    raise ValueError(f"unsupported Git host {kind!r}")


def main() -> int:
    sys.path.insert(0, "/usr/local/lib/pilot")
    import frame  # noqa: E402  (copied into the image)
    spec = json.load(open("/in/pr.json"))
    result = {"error": None}
    try:
        result = publish(client(spec, os.environ["PILOT_SCM_TOKEN"]), git_push, spec)
        result["error"] = None
    except Exception as e:  # noqa: BLE001
        result = {"error": f"{e.__class__.__name__}: {str(e)[:300]}"}
    out = os.path.join(tempfile.mkdtemp(), "result.json")
    with open(out, "w") as f:
        json.dump(result, f)
    frame.main(["emit", f"result.json={out}"])
    return 0 if not result.get("error") else 1


if __name__ == "__main__":
    sys.exit(main())
