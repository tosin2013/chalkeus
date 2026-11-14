"""Human decisions with or without a terminal.

With a terminal, the pilot asks there, as before (channel `tty`).

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
004) will be the stronger channel. The audit labels each confirmation with its channel so a
reviewer can tell them apart.
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


def record(run, step: str) -> tuple[int, str]:
    """`pilot approve`: approve exactly the pending step. Returns (exit code, message)."""
    p = pending_step(run)
    if p is None:
        return 2, f"run {run.id} is not paused at a decision (state={run.state.get('state')})"
    if p["step"] == "merge":
        return 4, ("the run is waiting for you to merge A's PR on GitHub; merging is never approved "
                   "here. Merge it, run `pilot/pilot resume`, then approve the reconcile proposal it shows.")
    if p["step"] != step:
        return 2, f"run {run.id} is waiting for '{p['step']}', not '{step}': {describe(p)}"
    a = {k: v for k, v in p.items() if k != "mode"} | {"channel": "harness", "at": evidence.now()}
    _file(run).parent.mkdir(parents=True, exist_ok=True)
    evidence.write_json(_file(run), a)
    run.event("approval", p["task"], p["n"], "approval_recorded", f"step={step} channel=harness")
    return 0, f"approved: {describe(p)}. Now run: pilot/pilot resume --run {run.id}"


def take(run, step: str, task=None, n=None) -> bool:
    """Use the recorded approval for this exact step, once. Anything else: no."""
    f = _file(run)
    if not f.exists():
        return False
    try:
        a = json.loads(f.read_text())
    finally:
        f.unlink(missing_ok=True)
    if a.get("step") != step or a.get("task") != task or a.get("n") != n:
        return False
    if step == "reconcile":
        rec_path = run.dir / "reconcile" / "A.json"
        rec = json.loads(rec_path.read_text()) if rec_path.exists() else {}
        if rec.get("merge_commit") != a.get("merge_commit"):
            return False
    return True


def decide(run, step: str, question: str, task=None, n=None, ask_tty=None) -> tuple[bool, str]:
    """Ask on the terminal if there is one; otherwise use a recorded approval for this step.

    Returns (approved, channel) with channel in {"tty", "harness", "none"}.
    """
    from . import reconcile
    ask_tty = ask_tty or reconcile.ask_tty
    if has_tty():
        return ask_tty(question).strip().lower() == "y", "tty"
    if take(run, step, task, n):
        print(f"{question}y  (approved with `pilot/pilot approve`)")
        return True, "harness"
    return False, "none"


def note(run, step: str, task, n, approved: bool, channel: str) -> None:
    run.event("approval", task, n, "approval_channel", f"step={step} approved={str(approved).lower()} channel={channel}")


def hint(run, step: str) -> str:
    return (f"no terminal: approve with `pilot/pilot approve --run {run.id} {step}` "
            f"(your agent harness asks you first), then `pilot/pilot resume --run {run.id}`")
