"""Human decisions with or without a terminal.

With a terminal, the pilot asks there, as before (channel `tty`), and any pending approval for
the run is discarded: the terminal answer is the current decision.

Without one (an agent's shell, CI), nothing is ever assumed: the run pauses at the decision
and says so. A human then approves that one pending step with `pilot/pilot approve`, and the
next `resume` uses the approval once (channel `harness`). An approval:
  - is accepted only for the step the run is paused at, after that step was shown
    (mode printed; the failed attempt graded; the reconcile proposal printed);
  - is bound to that step (task, attempt number, merge commit) and used at most once;
  - is recorded in events.jsonl with its channel, and the audit reports the channels.

The `harness` channel is only as strong as the agent harness rule that makes
`pilot/pilot approve` always ask the human (`.claude/settings.json`, `opencode.json`). It
trusts the harness and the agent not to forge the approval file; GitHub approvals (roadmap
004) are the stronger channel: they are recorded here with channel `github` and the approver's
login (see ghapprove.py), under the same single-use, bound rules. The audit labels each
confirmation with its channel so a reviewer can tell them apart.
"""
from __future__ import annotations

import json

from . import evidence

STEPS = ("mode", "retry", "reconcile")


def _file(run):
    return run.dir / "approvals" / "pending.json"


def has_tty() -> bool:
    try:
        with open("/dev/tty", "r"):
            return True
    except OSError:
        return False


def pending_step(run) -> dict | None:
    """The decision the run is paused at, or None."""
    st = str(run.state.get("state", ""))
    pend = run.state.get("pending") or {}
    if st == "paused(awaiting-mode)":
        return {"step": "mode", "task": None, "n": None, "mode": run.state.get("proposed_mode")}
    if st == "paused(awaiting-retry)":
        return {"step": "retry", "task": pend.get("task"), "n": int(pend.get("n"))}
    if st == "paused(awaiting-merge)":
        rec_path = run.dir / "reconcile" / "A.json"
        rec = json.loads(rec_path.read_text()) if rec_path.exists() else {}
        if rec.get("reason") == "DECLINED" and rec.get("proposed") and rec.get("merge_commit"):
            return {"step": "reconcile", "task": "A", "n": None, "merge_commit": rec["merge_commit"],
                    "proposed": rec["proposed"], "item_id": rec.get("item_id")}
        return {"step": "merge", "task": "A", "n": None}
    return None


def describe(p: dict) -> str:
    if p["step"] == "mode":
        return f"start the run in this mode: {json.dumps(p.get('mode'), sort_keys=True)}"
    if p["step"] == "retry":
        return f"retry task {p['task']} (attempt {p['n']})"
    if p["step"] == "reconcile":
        return (f"apply '{p.get('item_id')}: {p.get('proposed')}' to the fixture roadmap "
                f"(merge commit {p['merge_commit']})")
    return "merge"


def record(run, step: str, channel: str = "harness", approver: str | None = None,
           comment_id: int | None = None) -> tuple[int, str]:
    """Approve exactly the pending step (`pilot approve`: channel harness; 004: channel github
    with the approver's login and comment id). Returns (exit code, message)."""
    p = pending_step(run)
    if p is None:
        return 2, f"run {run.id} is not paused at a decision (state={run.state.get('state')})"
    if p["step"] == "merge":
        return 4, ("the run is waiting for you to merge A's PR on GitHub; merging is never approved "
                   "here. Merge it, run `pilot/pilot resume`, then approve the reconcile proposal it shows.")
    if p["step"] != step:
        return 2, f"run {run.id} is waiting for '{p['step']}', not '{step}': {describe(p)}"
    a = {k: v for k, v in p.items() if k != "mode"} | {"channel": channel, "at": evidence.now()}
    extra = ""
    if channel == "github":
        a |= {"approver": approver, "comment_id": comment_id}
        extra = f" approver={approver} comment={comment_id}"
    _file(run).parent.mkdir(parents=True, exist_ok=True)
    evidence.write_json(_file(run), a)
    run.event("approval", p["task"], p["n"], "approval_recorded", f"step={step} channel={channel}{extra}")
    by = f" by @{approver} on GitHub" if channel == "github" else ""
    return 0, f"approved{by}: {describe(p)}. Now run: pilot/pilot resume --run {run.id}"


def take(run, step: str, task=None, n=None) -> bool:
    """Use the recorded approval for this exact step, once. Anything else: no."""
    return take_record(run, step, task, n) is not None


def take_record(run, step: str, task=None, n=None) -> dict | None:
    """As `take`, returning the used record (with its channel) or None."""
    f = _file(run)
    if not f.exists():
        return None
    try:
        a = json.loads(f.read_text())
    finally:
        f.unlink(missing_ok=True)
    if a.get("step") != step or a.get("task") != task or a.get("n") != n:
        return None
    if step == "reconcile":
        rec_path = run.dir / "reconcile" / "A.json"
        rec = json.loads(rec_path.read_text()) if rec_path.exists() else {}
        if rec.get("merge_commit") != a.get("merge_commit"):
            return None
    return a


def decide(run, step: str, question: str, task=None, n=None, ask_tty=None) -> tuple[bool, str]:
    """Ask on the terminal if there is one; otherwise use a recorded approval for this step.

    Returns (approved, channel) with channel in {"tty", "harness", "github", "none"}. The used
    record is kept on `run.last_approval` so `note` can name a GitHub approver.
    """
    run.last_approval = None
    from . import reconcile
    ask_tty = ask_tty or reconcile.ask_tty
    if has_tty():
        try:
            return ask_tty(question).strip().lower() == "y", "tty"
        finally:
            _file(run).unlink(missing_ok=True)  # a terminal answer overrides a pending approval (R5)
    a = take_record(run, step, task, n)
    if a is not None:
        ch = a.get("channel", "harness")
        run.last_approval = a
        how = f"approved on GitHub by @{a.get('approver')}" if ch == "github" else "approved with `pilot/pilot approve`"
        print(f"{question}y  ({how})")
        return True, ch
    return False, "none"


def note(run, step: str, task, n, approved: bool, channel: str, approver: str | None = None,
         extra: str = "") -> None:
    detail = f"step={step} approved={str(approved).lower()} channel={channel}"
    if channel == "github":
        a = {} if approver else (getattr(run, "last_approval", None) or {})
        approver = approver or a.get("approver")
        cid = a.get("comment_id")
        detail += f" approver={approver}" + (f" comment={cid}" if cid else "") + (f" {extra}" if extra else "")
    run.event("approval", task, n, "approval_channel", detail)


def hint(run, step: str) -> str:
    return (f"no terminal: approve with `pilot/pilot approve --run {run.id} {step}` "
            f"(your agent harness asks you first), then `pilot/pilot resume --run {run.id}`")
