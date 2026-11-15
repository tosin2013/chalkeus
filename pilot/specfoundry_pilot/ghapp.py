"""004 US4: the GitHub App as the publisher credential (research R5).

The App's private key (`~/.config/specfoundry/github-app.pem`, mode 600) is never read into
Python, and is never copied to the cluster. `pilot/tools/app_token.py` mints a 1-hour
installation token in a child process, limited to the target repository with contents, pull
requests and issues write. It writes the token to `app-token.env` (mode 600, next to the key),
which is then used exactly like `publisher.env`. Only non-secret facts (bot login, expiry) are
kept in `app-token.json`.
"""
from __future__ import annotations

import datetime as _dt
import json
import subprocess
from pathlib import Path

from . import paths
from .config import ConfigError, check_mode

MINT = paths.ROOT / "pilot" / "tools" / "app_token.py"
MARGIN_S = 300  # re-mint when less than 5 minutes are left


def env_path(cfg) -> Path:
    return cfg.config_dir / "app-token.env"


def info_path(cfg) -> Path:
    return cfg.config_dir / "app-token.json"


def _info(cfg) -> dict:
    f = info_path(cfg)
    return json.loads(f.read_text()) if f.exists() else {}


def _fresh(info: dict, now=None) -> bool:
    exp = info.get("expires_at")
    if not exp:
        return False
    now = now or _dt.datetime.now(_dt.timezone.utc)
    t = _dt.datetime.strptime(exp, "%Y-%m-%dT%H:%M:%SZ").replace(tzinfo=_dt.timezone.utc)
    return (t - now).total_seconds() > MARGIN_S


def mint(cfg, key: Path | None = None, out: Path | None = None) -> dict:
    key = key or cfg.app_key
    out = out or env_path(cfg)
    if not key.exists():
        raise ConfigError(f"no GitHub App key at {key}: store it with `pilot/pilot app-key`")
    check_mode(key)
    app_id, inst = cfg.values.get("PILOT_APP_ID", ""), cfg.values.get("PILOT_APP_INSTALLATION_ID", "")
    r = subprocess.run(["python3", "-I", str(MINT), str(key), app_id, inst, cfg.repo, str(out)],
                       capture_output=True, text=True, timeout=90)
    try:
        res = json.loads(r.stdout.strip().splitlines()[-1])
    except Exception:  # noqa: BLE001
        res = {"ok": False, "problems": [f"app token mint failed to run: {(r.stderr or '').strip()[-300:]}"]}
    if res.get("ok") and out == env_path(cfg):
        info_path(cfg).write_text(json.dumps({k: res.get(k) for k in ("login", "expires_at", "repositories")}))
    return res


def current_env(cfg) -> Path:
    """The env file with a valid App token, minting a new one when missing or near expiry."""
    if env_path(cfg).exists() and _fresh(_info(cfg)):
        return env_path(cfg)
    res = mint(cfg)
    if not res.get("ok"):
        from .ghsync import GitHubLoopError
        raise GitHubLoopError("GitHub App token: " + "; ".join(res.get("problems") or ["unknown error"]))
    return env_path(cfg)


def bot_login(cfg) -> str:
    current_env(cfg)
    return _info(cfg).get("login") or ""


def discard(cfg) -> None:
    env_path(cfg).unlink(missing_ok=True)
    info_path(cfg).unlink(missing_ok=True)


def check(cfg, key: Path | None = None) -> int:
    """`pilot app-token --check`: mint a token, run the write probe with it, discard it."""
    from . import publisher_token
    tmp = cfg.config_dir / "app-token.check.env"
    try:
        res = mint(cfg, key=key, out=tmp)
        if not res.get("ok"):
            print("GitHub App: NOT OK")
            for p in res.get("problems") or []:
                print(f"  - {p}")
            return 1
        probe = publisher_token.verify(cfg, env_file=tmp, issues=True)
        print(f"GitHub App {res.get('login')}: token expires {res.get('expires_at')}, repositories {res.get('repositories')}")
        publisher_token.report(probe)
        return 0 if probe.get("ok") else 1
    finally:
        tmp.unlink(missing_ok=True)


def store_key(cfg) -> int:
    """`pilot app-key`: copy the PEM from stdin to a staging file (mode 600, never through Python
    memory); keep it only if `check` passes (the 006 F3 rule)."""
    staging = cfg.app_key.with_name(cfg.app_key.name + ".new")
    r = subprocess.run(["install", "-m", "600", "/dev/stdin", str(staging)])
    if r.returncode != 0 or not staging.exists() or staging.stat().st_size == 0:
        staging.unlink(missing_ok=True)
        print("  not saved: no key read from stdin")
        return 1
    if check(cfg, key=staging) != 0:
        staging.unlink(missing_ok=True)
        print("  not saved: " + ("your previous key is unchanged" if cfg.app_key.exists() else "no key stored")
              + ". Fix the App settings and run this again.")
        return 1
    staging.replace(cfg.app_key)
    print(f"  saved: {cfg.app_key}")
    return 0
