#!/usr/bin/env python3
"""Record a target's GitHub API responses for internal/debt's tests (029 T005).

Read-only: every call is `gh api` GET. Bodies are trimmed to the lines the parser reads, and
lockfiles to the packages the PRs bump plus those with install scripts, so the test data stays
small and holds no release notes. No token is written.

  python3 internal/debt/testdata/record.py tosin2013/documcp internal/debt/testdata/documcp-20261124
"""
import base64
import json
import os
import re
import subprocess
import sys


def gh(path, raw=False):
    args = ["gh", "api", "-X", "GET", path]
    if raw:
        args += ["-H", "Accept: application/vnd.github.raw+json"]
    p = subprocess.run(args, capture_output=True, text=True)
    if p.returncode != 0:
        m = re.search(r"HTTP (\d+)", p.stderr)
        return int(m.group(1)) if m else 500, None
    return 200, (p.stdout if raw else json.loads(p.stdout))


def trim_body(body):
    keep = []
    for line in (body or "").splitlines():
        if line.startswith(("Bumps ", "Updates `", "| ")):
            keep.append(line)
    return "\n".join(keep)


def trim_lock(text, names):
    lock = json.loads(text)
    pk = lock.get("packages", {})
    kept = {k: v for k, v in pk.items()
            if k == "" or v.get("hasInstallScript") or k.split("node_modules/")[-1] in names}
    for v in kept.values():
        for drop in ("resolved", "integrity", "dependencies", "devDependencies", "peerDependencies",
                     "optionalDependencies", "engines", "funding", "bin", "license"):
            v.pop(drop, None)
    return json.dumps({"name": lock.get("name"), "lockfileVersion": lock.get("lockfileVersion"),
                       "packages": kept}, indent=1, sort_keys=True)


def main():
    repo, out = sys.argv[1], sys.argv[2]
    r = {}

    def put(path, status, body):
        r["GET " + path] = {"status": status, "body": body}

    st, info = gh(f"/repos/{repo}")
    put(f"/repos/{repo}", st, {"default_branch": info["default_branch"], "full_name": info["full_name"]})
    branch = info["default_branch"]
    st, b = gh(f"/repos/{repo}/branches/{branch}")
    sha = b["commit"]["sha"]
    put(f"/repos/{repo}/branches/{branch}", st, {"name": branch, "commit": {"sha": sha}})
    st, prs = gh(f"/repos/{repo}/pulls?per_page=100&state=open")
    slim, names, dirs = [], set(), {"/"}
    for p in prs:
        body = trim_body(p.get("body"))
        slim.append({"number": p["number"], "html_url": p["html_url"], "title": p["title"], "body": body,
                     "user": {"login": p["user"]["login"], "type": p["user"]["type"]},
                     "head": {"ref": p["head"]["ref"], "sha": p["head"]["sha"]}})
        for m in re.finditer(r"bump (\S+) from", p["title"], re.I):
            names.add(m.group(1))
        for m in re.finditer(r"Updates `([^`]+)`", body):
            names.add(m.group(1))
        m = re.search(r" in (/\S*)$", p["title"])
        if m:
            dirs.add(m.group(1))
    put(f"/repos/{repo}/pulls?per_page=100&state=open", st, slim)
    for s in [sha] + [p["head"]["sha"] for p in slim]:
        st, runs = gh(f"/repos/{repo}/commits/{s}/check-runs?per_page=100")
        if runs is not None:
            runs = {"total_count": runs["total_count"], "check_runs": [
                {"name": c["name"], "status": c["status"], "conclusion": c["conclusion"]} for c in runs["check_runs"]]}
        put(f"/repos/{repo}/commits/{s}/check-runs?per_page=100", st, runs)
        st, stat = gh(f"/repos/{repo}/commits/{s}/status?per_page=100")
        if stat is not None:
            stat = {"state": stat["state"], "statuses": [{"context": x["context"], "state": x["state"]} for x in stat["statuses"]]}
        put(f"/repos/{repo}/commits/{s}/status?per_page=100", st, stat)
    for d in sorted(dirs):
        for f in ("package.json", "package-lock.json", "jest.config.js"):
            path = (d.strip("/") + "/" + f).lstrip("/")
            st, text = gh(f"/repos/{repo}/contents/{path}?ref={sha}", raw=True)
            if text is not None and f == "package-lock.json":
                text = trim_lock(text, names)
            put(f"/repos/{repo}/contents/{path}?ref={sha}", st, text)
    st, alerts = gh(f"/repos/{repo}/dependabot/alerts?per_page=100&state=open")
    if alerts is not None:
        alerts = [{"number": a["number"], "security_advisory": {"ghsa_id": a["security_advisory"]["ghsa_id"]},
                   "security_vulnerability": {"package": {"name": a["security_vulnerability"]["package"]["name"]},
                                              "first_patched_version": a["security_vulnerability"].get("first_patched_version")}}
                  for a in alerts]
    put(f"/repos/{repo}/dependabot/alerts?per_page=100&state=open", st, alerts)
    os.makedirs(out, exist_ok=True)
    with open(os.path.join(out, "responses.json"), "w") as fh:
        json.dump({"repo": repo, "default_commit": sha, "responses": r}, fh, indent=1, sort_keys=True)
        fh.write("\n")
    print(f"{len(r)} responses, {len(slim)} open PRs, default {sha[:7]}")


if __name__ == "__main__":
    main()
