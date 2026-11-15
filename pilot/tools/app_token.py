"""Mint a GitHub App installation token for the pilot (004 research R5). Child process only.

    app_token.py <key.pem> <app_id> <installation_id> <owner/repo> <out.env>

The private key is read only by `openssl` (never by this script). The JWT and the installation
token exist only in this child: the token is written to <out.env> (mode 600) as
    PILOT_SCM_TOKEN=...   PILOT_SCM_KIND=app   PILOT_SCM_LOGIN=<slug>[bot]
Prints one JSON line without any secret: {ok, login, expires_at, repositories, permissions, problems}.
The token is limited to the one repository with contents, pull requests and issues write.
"""
from __future__ import annotations

import base64
import json
import os
import subprocess
import sys
import time
import urllib.error
import urllib.request

API = "https://api.github.com"
PERMISSIONS = {"contents": "write", "pull_requests": "write", "issues": "write"}


def b64url(b: bytes) -> str:
    return base64.urlsafe_b64encode(b).rstrip(b"=").decode()


def jwt(key_path: str, app_id: str, now: int | None = None, sign=None) -> str:
    now = int(now if now is not None else time.time())
    header = b64url(json.dumps({"alg": "RS256", "typ": "JWT"}, separators=(",", ":")).encode())
    payload = b64url(json.dumps({"iat": now - 60, "exp": now + 540, "iss": str(app_id)}, separators=(",", ":")).encode())
    signing_input = f"{header}.{payload}".encode()
    sign = sign or (lambda data: subprocess.run(["openssl", "dgst", "-sha256", "-sign", key_path], input=data,
                                                capture_output=True, check=True).stdout)
    return f"{header}.{payload}.{b64url(sign(signing_input))}"


def call(method: str, path: str, bearer: str, body=None):
    data = json.dumps(body).encode() if body is not None else None
    r = urllib.request.Request(API + path, data=data, method=method, headers={
        "Authorization": f"Bearer {bearer}", "Accept": "application/vnd.github+json",
        "X-GitHub-Api-Version": "2022-11-28", "User-Agent": "specfoundry-pilot-app-token"})
    try:
        with urllib.request.urlopen(r, timeout=30) as resp:
            return resp.status, json.loads(resp.read() or b"null")
    except urllib.error.HTTPError as e:
        return e.code, None


def token_request(repo: str) -> dict:
    return {"repositories": [repo.split("/", 1)[1]], "permissions": dict(PERMISSIONS)}


def mint(key_path, app_id, installation_id, repo, out_env, call=call, sign=None) -> dict:
    problems = []
    try:
        j = jwt(key_path, app_id, sign=sign)
    except (subprocess.CalledProcessError, OSError) as e:
        return {"ok": False, "problems": [f"cannot sign with the App key ({type(e).__name__}); is it the App's private key?"]}
    st, app = call("GET", "/app", j)
    if st != 200:
        return {"ok": False, "problems": [f"GitHub rejected the App JWT (HTTP {st}): check PILOT_APP_ID and the key"]}
    login = f"{app['slug']}[bot]"
    st, tok = call("POST", f"/app/installations/{int(installation_id)}/access_tokens", j, token_request(repo))
    if st != 201:
        return {"ok": False, "login": login, "problems": [
            f"cannot mint an installation token for {repo} (HTTP {st}): is the App installed on that repository "
            "with Contents, Pull requests and Issues: read and write?"]}
    perms = tok.get("permissions") or {}
    for k in PERMISSIONS:
        if perms.get(k) != "write":
            problems.append(f"the App lacks {k}: write on {repo}")
    repos = [r.get("full_name") for r in tok.get("repositories") or []]
    if repos and repos != [repo]:
        problems.append(f"token covers {repos}, expected only {repo}")
    fd = os.open(out_env, os.O_WRONLY | os.O_CREAT | os.O_TRUNC, 0o600)
    with os.fdopen(fd, "w") as f:
        f.write(f"PILOT_SCM_TOKEN={tok['token']}\nPILOT_SCM_KIND=app\nPILOT_SCM_LOGIN={login}\n")
    return {"ok": not problems, "login": login, "expires_at": tok.get("expires_at"), "repositories": repos,
            "permissions": perms, "problems": problems}


def main(argv) -> int:
    if len(argv) != 6:
        print(json.dumps({"ok": False, "problems": ["usage: app_token.py <key> <app_id> <installation_id> <repo> <out.env>"]}))
        return 2
    res = mint(*argv[1:])
    print(json.dumps(res))
    return 0 if res["ok"] else 1


if __name__ == "__main__":
    sys.exit(main(sys.argv))
