"""GitHub write operations for the 004 GitHub loop (contracts/github-objects.md).

Runs as a child process with the publisher credential in its own environment only
(PILOT_SCM_TOKEN), the same pattern as check_token.py: the orchestrator never holds the value.
It never prints the token. There is deliberately no merge operation here.

    ghops.py <op> <payload.json>

ops: whoami | ensure-labels | create-issue | edit-issue | comment | close-issue
Prints one JSON line: {"ok": true, "result": {...}} or {"ok": false, "error": "..."}.
"""
from __future__ import annotations

import json
import os
import sys
import urllib.error
import urllib.parse
import urllib.request

API = "https://api.github.com"


class GitHubError(Exception):
    pass


def req(token: str, method: str, path: str, body=None, api: str = API):
    data = json.dumps(body).encode() if body is not None else None
    r = urllib.request.Request(api + path, data=data, method=method, headers={
        "Authorization": f"Bearer {token}", "Accept": "application/vnd.github+json",
        "X-GitHub-Api-Version": "2022-11-28", "User-Agent": "specfoundry-pilot-ghops"})
    try:
        with urllib.request.urlopen(r, timeout=30) as resp:
            return resp.status, json.loads(resp.read() or b"null")
    except urllib.error.HTTPError as e:
        return e.code, None


def _ok(st, want, what):
    if st not in want:
        raise GitHubError(f"{what} failed (HTTP {st})")


def op_whoami(token, p, call):
    if os.environ.get("PILOT_SCM_KIND") == "app":
        return {"login": os.environ.get("PILOT_SCM_LOGIN", ""), "app": True}
    st, u = call(token, "GET", "/user")
    _ok(st, (200,), "whoami")
    return {"login": u["login"], "app": False}


def op_ensure_labels(token, p, call):
    repo = p["repo"]
    created = []
    for lab in p["labels"]:
        st, _ = call(token, "GET", f"/repos/{repo}/labels/{urllib.parse.quote(lab['name'], safe='')}")
        if st == 200:
            continue
        st, _ = call(token, "POST", f"/repos/{repo}/labels", {"name": lab["name"], "color": lab.get("color", "ededed")})
        _ok(st, (201, 422), f"create label {lab['name']}")  # 422: created concurrently
        created.append(lab["name"])
    return {"created": created}


def op_create_issue(token, p, call):
    st, i = call(token, "POST", f"/repos/{p['repo']}/issues",
                 {"title": p["title"], "body": p["body"], "labels": p.get("labels", [])})
    _ok(st, (201,), "create issue")
    return {"number": i["number"], "url": i.get("html_url")}


def op_edit_issue(token, p, call):
    body = {k: p[k] for k in ("title", "body", "labels") if k in p}
    st, i = call(token, "PATCH", f"/repos/{p['repo']}/issues/{int(p['number'])}", body)
    _ok(st, (200,), "edit issue")
    return {"number": i["number"]}


def op_comment(token, p, call):
    st, c = call(token, "POST", f"/repos/{p['repo']}/issues/{int(p['number'])}/comments", {"body": p["body"]})
    _ok(st, (201,), "comment")
    return {"id": c["id"], "url": c.get("html_url")}


def op_close_issue(token, p, call):
    st, i = call(token, "PATCH", f"/repos/{p['repo']}/issues/{int(p['number'])}", {"state": "closed"})
    _ok(st, (200,), "close issue")
    return {"number": i["number"]}


OPS = {"whoami": op_whoami, "ensure-labels": op_ensure_labels, "create-issue": op_create_issue,
       "edit-issue": op_edit_issue, "comment": op_comment, "close-issue": op_close_issue}


def run(op: str, payload: dict, token: str, call=req) -> dict:
    if op not in OPS:
        return {"ok": False, "error": f"unknown op {op}"}
    try:
        return {"ok": True, "result": OPS[op](token, payload, call)}
    except GitHubError as e:
        return {"ok": False, "error": str(e)}
    except (urllib.error.URLError, OSError, KeyError, ValueError) as e:
        return {"ok": False, "error": f"{type(e).__name__}: {e}"}


def main(argv) -> int:
    token = os.environ.get("PILOT_SCM_TOKEN", "")
    if not token:
        print(json.dumps({"ok": False, "error": "PILOT_SCM_TOKEN missing"}))
        return 1
    if len(argv) != 3:
        print(json.dumps({"ok": False, "error": "usage: ghops.py <op> <payload.json>"}))
        return 2
    with open(argv[2]) as f:
        payload = json.load(f)
    res = run(argv[1], payload, token)
    print(json.dumps(res))
    return 0 if res["ok"] else 1


if __name__ == "__main__":
    sys.exit(main(sys.argv))
