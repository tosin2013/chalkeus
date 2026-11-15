"""004 GitHub approvals (research R3, R4; contracts/github-objects.md).

The pilot posts each pending decision (mode, retry) on the run's tracking issue. An allowed
approver replies `/approve <step>`. On `resume` without a terminal, the pilot reads the comments
and records the first valid approval in 012's pending-step record with channel `github`, so
the existing single-use, bound rules apply unchanged.

A comment counts only if all of these hold:
  - its first line is exactly `/approve <step>`;
  - it was posted after the matching decision post, and never edited;
  - its author is in PILOT_APPROVERS and still has write, maintain or admin on the target repo;
  - it was not made with the publisher credential, nor through any GitHub App;
  - its author did not post a later `/withdraw <step>`.

Every candidate comment is logged with its reason. Any GitHub error means no approval (fail
closed). Limit (FR-009a): an approval binds to a GitHub identity, so anyone who can act as that
identity can approve. Agents must never post these comments.
"""
from __future__ import annotations

import json
import re
import subprocess

from . import approvals, evidence, fixture, ghsync
from .ghsync import GitHubLoopError

ALLOWED_PERMISSIONS = {"admin", "maintain", "write"}
DECISION_MARKER = re.compile(r"<!-- specfoundry:decision run=(\S+) step=(\S+) task=(\S+) n=(\S+) -->")


# ---------------------------------------------------------------- I/O seams (patched in tests)

def read_comments(cfg, number: int) -> list[dict]:
    try:
        out = fixture.gh("api", "--paginate", f"repos/{cfg.repo}/issues/{int(number)}/comments").stdout
    except subprocess.CalledProcessError as e:
        raise GitHubLoopError(f"cannot read comments on #{number}: {(e.stderr or '').strip()[-200:]}")
    raw = json.loads(out.replace("][", ",")) if out.strip() else []
    return [{"id": c["id"], "body": c.get("body") or "", "author": (c.get("user") or {}).get("login", ""),
             "created_at": c["created_at"], "updated_at": c["updated_at"],
             "app": bool(c.get("performed_via_github_app")) or (c.get("user") or {}).get("type") == "Bot"}
            for c in raw]


def read_permission(cfg, login: str) -> str:
    try:
        out = fixture.gh("api", f"repos/{cfg.repo}/collaborators/{login}/permission").stdout
    except subprocess.CalledProcessError:
        return "none"
    d = json.loads(out)
    role = d.get("role_name") or ""
    return "maintain" if role == "maintain" else d.get("permission", "none")


# ---------------------------------------------------------------- tracking issue and posts

def _tracking_file(run):
    return run.dir / "github" / "tracking.json"


def _tracking(run) -> dict:
    f = _tracking_file(run)
    return json.loads(f.read_text()) if f.exists() else {"issue": None, "decisions": []}


def _save(run, t: dict) -> None:
    _tracking_file(run).parent.mkdir(exist_ok=True)
    evidence.write_json(_tracking_file(run), t)


def publisher_login(cfg, run) -> str:
    login = run.state.get("publisher_login")
    if not login:
        login = ghsync.run_ghops(cfg, "whoami", {})["login"]
        run.state["publisher_login"] = login
        run.save()
    return login


def ensure_tracking(cfg, run) -> int:
    t = _tracking(run)
    if t["issue"]:
        return t["issue"]
    body = (f"<!-- specfoundry:run={run.id} -->\nDecisions for pilot run `{run.id}` are posted here. "
            f"Allowed approvers reply with the exact command shown in each post.\n\n"
            f"_The pilot never merges, and a merge is never approved here._\n")
    res = ghsync.run_ghops(cfg, "create-issue", {"repo": cfg.repo, "title": f"pilot run {run.id}", "body": body})
    t["issue"] = res["number"]
    _save(run, t)
    return t["issue"]


def marker(run, p: dict) -> str:
    return f"<!-- specfoundry:decision run={run.id} step={p['step']} task={p.get('task') or '-'} n={p.get('n') or '-'} -->"


def post_decision(cfg, run, p: dict) -> dict:
    """Post the pending decision once (idempotent per run/step/task/n). Returns the decision entry."""
    t = _tracking(run)
    key = marker(run, p)
    for d in t["decisions"]:
        if d["marker"] == key:
            return d
    issue = ensure_tracking(cfg, run)
    t = _tracking(run)
    approvers = ", ".join(f"@{a}" for a in cfg.approvers)
    body = (f"{key}\n**Decision pending: {approvals.describe(p)}.**\n\n"
            f"To approve, reply with exactly:\n\n    /approve {p['step']}\n\n"
            f"Allowed approvers: {approvers}. To cancel your approval before the run resumes: "
            f"`/withdraw {p['step']}`.\n")
    res = ghsync.run_ghops(cfg, "comment", {"repo": cfg.repo, "number": issue, "body": body})
    d = {"marker": key, "step": p["step"], "task": p.get("task"), "n": p.get("n"), "comment_id": res["id"],
         "at": evidence.now()}
    t["decisions"].append(d)
    _save(run, t)
    run.event("github", p.get("task"), p.get("n"), "decision_posted", f"step={p['step']} issue={issue} comment={res['id']}")
    return d


def mark_answered(cfg, run, step: str, approved: bool) -> None:
    """With a terminal attached, the terminal decided (012 R5): say so under the post (best effort)."""
    t = _tracking(run)
    if not t["issue"]:
        return
    try:
        ghsync.run_ghops(cfg, "comment", {"repo": cfg.repo, "number": t["issue"],
                                   "body": f"Decision `{step}` was answered at the operator's terminal "
                                           f"({'yes' if approved else 'no'}); approvals here no longer apply."})
    except GitHubLoopError:
        pass


# ---------------------------------------------------------------- reading approvals

def evaluate(comments: list[dict], decision: dict, step: str, approvers: list[str], publisher: str,
             permission) -> tuple[dict | None, list[dict]]:
    """Pure rule check. Returns (accepted comment or None, log of every candidate with reason)."""
    log, valid = [], []
    want = f"/approve {step}"
    allowed = {a.lower() for a in approvers}
    cmts = sorted(comments, key=lambda c: c["id"])
    withdrawn_after = {}
    for c in cmts:
        first = (c["body"].strip().splitlines() or [""])[0].strip()
        if first == f"/withdraw {step}" and c["id"] > decision["comment_id"]:
            withdrawn_after.setdefault(c["author"].lower(), []).append(c["id"])
    for c in cmts:
        first = (c["body"].strip().splitlines() or [""])[0].strip()
        if not first.startswith("/approve"):
            continue
        reason = None
        author = c["author"].lower()
        if c["id"] <= decision["comment_id"]:
            reason = "before_decision"
        elif first != want:
            reason = "wrong_step"
        elif c.get("app"):
            reason = "app_authored"
        elif author == (publisher or "").lower():
            reason = "publisher_identity"
        elif c["updated_at"] != c["created_at"]:
            reason = "edited"
        elif author not in allowed:
            reason = "not_an_approver"
        elif any(w > c["id"] for w in withdrawn_after.get(author, [])):
            reason = "withdrawn"
        elif permission(c["author"]) not in ALLOWED_PERMISSIONS:
            reason = "no_write_access"
        elif valid:
            reason = "duplicate"
        entry = {"comment_id": c["id"], "author": c["author"], "created_at": c["created_at"],
                 "accepted": reason is None, "reason": reason}
        log.append(entry)
        if reason is None:
            valid.append(c)
    return (valid[0] if valid else None), log


def find_approval(cfg, run, p: dict) -> dict | None:
    """Read the tracking issue for an approval of pending step `p`. Logs every candidate.
    Raises GitHubLoopError on any GitHub problem (the caller keeps the run paused)."""
    t = _tracking(run)
    d = next((x for x in t["decisions"] if x["marker"] == marker(run, p)), None)
    if not t["issue"] or d is None:
        return None
    comments = read_comments(cfg, t["issue"])
    accepted, log = evaluate(comments, d, p["step"], cfg.approvers, publisher_login(cfg, run),
                             lambda login: read_permission(cfg, login))
    f = run.dir / "github" / "approvals.jsonl"
    seen = {json.loads(x)["comment_id"] for x in f.read_text().splitlines()} if f.exists() else set()
    with f.open("a") as out:
        for e in log:
            if e["comment_id"] in seen:
                continue
            out.write(json.dumps({**e, "step": p["step"], "task": p.get("task"), "n": p.get("n"),
                                  "at": evidence.now()}, sort_keys=True) + "\n")
            if not e["accepted"]:
                run.event("approval", p.get("task"), p.get("n"), "approval_rejected",
                          f"step={p['step']} comment={e['comment_id']} author={e['author']} reason={e['reason']}")
    if accepted is None:
        return None
    return {"approver": accepted["author"], "comment_id": accepted["id"]}


def collect(cfg, run) -> str | None:
    """On resume without a terminal: if a valid GitHub approval exists for the pending step,
    record it (channel github). Returns a message for the operator, or None."""
    p = approvals.pending_step(run)
    if not p or p["step"] not in ("mode", "retry"):
        return None
    try:
        found = find_approval(cfg, run, p)
    except Exception as e:  # noqa: BLE001 - any doubt means no approval (fail closed)
        return f"could not read GitHub approvals ({e}); the run stays paused"
    if not found:
        return None
    code, msg = approvals.record(run, p["step"], channel="github", approver=found["approver"],
                                 comment_id=found["comment_id"])
    return msg if code == 0 else None
