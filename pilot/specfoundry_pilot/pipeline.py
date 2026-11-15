"""`pilot run` / `pilot resume`: one governed cycle for fixture tasks A then B.

Every human decision point fails closed: anything but `y` means "no". With no terminal, the run
pauses at the decision instead, and `pilot approve` (see approvals.py) can approve that one step.
  mode confirmation (FR-023)        -> exit 5, no worker staged (no terminal: paused(awaiting-mode))
  retry confirmation (FR-017)       -> paused(awaiting-retry), exit 6
  merge of A's PR (FR-018a)          -> paused(awaiting-merge), exit 0 / resume exit 4
  reconcile approval (FR-018)        -> exit 5, roadmap unchanged
"""
from __future__ import annotations

import datetime as _dt
import json
import re
import shutil
import subprocess

from . import approvals, evidence, fixture, gate, grader, images, paths, publisher, reconcile, worker
from .runner.jobs import JobsRunner

try:
    from . import checks  # US3 live checks
except ImportError:  # pragma: no cover
    checks = None

EXIT_OK, EXIT_FAIL, EXIT_ERROR, EXIT_AWAIT_MERGE, EXIT_DECLINED, EXIT_AWAIT_RETRY = 0, 1, 2, 4, 5, 6


def mode_of(cfg) -> dict:
    return {"name": "step-by-step", "retry_limit": cfg.retry_limit, "spend_cap_usd": float(cfg["PILOT_SPEND_CAP_USD"]),
            "token_cap": int(cfg["PILOT_TOKEN_CAP"]), "worker_deadline_s": worker.DEADLINE}


def confirm_mode(run, cfg) -> tuple[bool, str]:
    mode = mode_of(cfg)
    run.state["proposed_mode"] = mode
    run.save()
    print(f"Run mode for this pilot: {json.dumps(mode, sort_keys=True)}\n"
          "  one attempt at a time; every retry needs your confirmation; risk rating is out of scope in M1.")
    ok, ch = approvals.decide(run, "mode", "Confirm this mode and start staging workers? [y/N] ")
    approvals.note(run, "mode", None, None, ok, ch)
    github_terminal(cfg, run, "mode", ok, ch)
    if not ok:
        run.event("mode", None, None, "mode_declined", json.dumps(mode, sort_keys=True))
        return False, ch
    run.state["mode"] = mode
    run.event("mode", None, None, "mode_confirmed", json.dumps(mode, sort_keys=True))
    return True, ch


def confirm_retry(run, task: str, n: int, total: int, cfg=None) -> tuple[bool, str]:
    ok, ch = approvals.decide(run, "retry", f"Retry {task} (attempt {n} of {total})? [y/N] ", task=task, n=n)
    approvals.note(run, "retry", task, n, ok, ch)
    github_terminal(cfg, run, "retry", ok, ch)
    run.event("retry", task, n, "retry_confirmed" if ok else "retry_declined", "")
    return ok, ch


def github_on(cfg) -> bool:
    return bool(cfg is not None and getattr(cfg, "github_approvals", False))


def github_post(cfg, run) -> None:
    """004: with GitHub approvals on, post the pending decision on the run's tracking issue."""
    if not github_on(cfg):
        return
    from . import ghapprove, ghsync
    p = approvals.pending_step(run)
    if not p or p["step"] not in ("mode", "retry"):
        return
    try:
        ghapprove.post_decision(cfg, run, p)
        issue = ghapprove._tracking(run)["issue"]
        print(f"  or approve on GitHub: https://github.com/{cfg.repo}/issues/{issue} "
              f"(reply `/approve {p['step']}`; allowed: {', '.join(cfg.approvers)})")
    except ghsync.GitHubLoopError as e:
        print(f"  (could not post the decision on GitHub: {e})")


def github_terminal(cfg, run, step: str, ok: bool, ch: str) -> None:
    if github_on(cfg) and ch == "tty":
        from . import ghapprove
        ghapprove.mark_answered(cfg, run, step, ok)


def mode_not_confirmed(run, ch: str, cfg=None) -> int:
    if ch == "none":
        _pause(run, "awaiting-mode")
        print(f"paused: mode not confirmed. {approvals.hint(run, 'mode')}")
        github_post(cfg, run)
        return EXIT_DECLINED
    run.set_run_state("failed", "MODE_NOT_CONFIRMED")
    evidence.write_summary(run)
    return EXIT_DECLINED


def gate_task(run, cfg, task: str) -> dict:
    seq = run.next_gate_seq()
    d = gate.evaluate_fresh(cfg, task, seq=seq, out_dir=run.dir / "gate")
    t = run.task(task)
    t.setdefault("gate_seqs", []).append(seq)
    if d["decision"] == "allow":
        t["base_revision"] = d["input"]["fixture_commit"]
    run.save()
    run.event("gate", task, None, f"gate_{d['decision']}", f"seq={seq} " + ",".join(d["reasons"]))
    print(f"  gate {task}: {d['decision']} {d['reasons']}")
    return d


def _clone_at(cfg, rev: str):
    path, _ = fixture.fresh_clone(cfg.repo, cfg.branch, depth=None)
    subprocess.run(["git", "-C", str(path), "checkout", "--quiet", rev], check=True)
    return path


def _pause(run, state: str, **pending) -> None:
    run.state["pending"] = pending
    run.state["paused_at"] = evidence.now()
    run.set_run_state(f"paused({state})")
    evidence.write_summary(run)


def _fail(run, reason: str) -> int:
    run.set_run_state("failed", reason)
    evidence.write_summary(run)
    print(f"run {run.id} FAILED: {reason}")
    return EXIT_FAIL


def run_task(run, cfg, imgs, runner, task: str, start_n: int = 1) -> int:
    """Attempts (confirmed retries), grading, publishing, then the post-publish step."""
    t = run.task(task)
    total = 1 + cfg.retry_limit
    n = start_n
    while True:
        t["state"] = "attempting"
        run.save()
        clone = _clone_at(cfg, t["base_revision"])
        try:
            att = worker.run_attempt(run, cfg, imgs, runner, task, n, clone, t["base_revision"], checks=checks)
        finally:
            shutil.rmtree(clone, ignore_errors=True)
        if att["state"] == "collected":
            g = grader.grade(run, imgs, runner, task, n)
            att["result"] = g["result"]
            att["state"] = "passed" if g["result"] == "pass" else "failed"
            if g["result"] != "pass":
                att["reason"] = g.get("failure") or "grade_fail"
        else:
            att["result"] = "fail"
        t["attempts"] = [a for a in t.get("attempts", []) if a["n"] != n] + [att]
        run.save()
        if att["state"] == "passed":
            t["passing_attempt"] = n
            break
        if task == "A":
            d = gate_task(run, cfg, "B")
            if d["decision"] != "refuse":
                return _fail(run, "B_NOT_BLOCKED")
        if n >= total:
            return _fail(run, f"{task}_RETRIES_EXHAUSTED")
        ok, ch = confirm_retry(run, task, n + 1, total, cfg)
        if not ok:
            _pause(run, "awaiting-retry", task=task, n=n + 1)
            print(f"paused: retry of {task} not confirmed. "
                  + (approvals.hint(run, "retry") if ch == "none" else "`pilot resume` asks again."))
            if ch == "none":
                github_post(cfg, run)
            return EXIT_AWAIT_RETRY
        n += 1
    return publish_task(run, cfg, imgs, runner, task)


def publish_task(run, cfg, imgs, runner, task: str) -> int:
    t = run.task(task)
    n = t["passing_attempt"]
    t["state"] = "publishing"
    run.save()
    try:
        pr = publisher.publish(run, cfg, imgs, runner, task, n)
    except Exception as e:  # noqa: BLE001 - publisher trouble is resumable, never "verified"
        t["state"] = "publish-failed"
        run.event("publish", task, n, "publish_failed", f"{e.__class__.__name__}: {str(e)[:300]}")
        _pause(run, "awaiting-publish", task=task, n=n)
        print(f"publisher failed for {task}: {e}\n`pilot resume` re-runs it (idempotent).")
        return EXIT_FAIL
    t["state"] = "published"
    t["pr"] = pr
    run.save()
    if checks:
        checks.run_cluster_checks(run, imgs)
    if task == "A":
        d = gate_task(run, cfg, "B")
        if d["decision"] != "refuse":
            return _fail(run, "B_NOT_BLOCKED")
        t["state"] = "awaiting-merge"
        _pause(run, "awaiting-merge", task="A")
        print(f"\nTask A draft PR: {pr['url']}\nReview and merge it on GitHub yourself, then run: pilot/pilot resume")
        return EXIT_OK
    return finish(run, cfg)


def github_refresh(cfg, run, final: bool = False) -> None:
    """004: re-sync labels/bodies after a status change; at the end, record the drift report.
    Best effort here: a GitHub error is recorded, never treated as success, and the drift report
    (SC-001) shows anything left over."""
    if not getattr(cfg, "github_issues", False):
        return
    from . import ghsync
    try:
        ghsync.sync(cfg, run)
        if final:
            rep = ghsync.drift(cfg, run)
            print(f"github drift: {len(rep)} difference(s)" + ("" if not rep else f": {rep}"))
    except ghsync.GitHubLoopError as e:
        run.event("github", None, None, "github_sync_failed", str(e)[:300])
        print(f"warning: github sync failed: {e}")


def finish(run, cfg=None) -> int:
    from . import audit
    if cfg is not None:
        github_refresh(cfg, run, final=True)
    ok = audit.run(run)
    if not ok:
        return _fail(run, "AUDIT")
    run.set_run_state("succeeded")
    evidence.write_summary(run)
    print(f"run {run.id} succeeded. Evidence: {run.dir.relative_to(paths.ROOT)}")
    return EXIT_OK


def run(cfg) -> int:
    imgs = images.load()
    runner = JobsRunner()
    open_prs = fixture.open_pilot_prs(cfg.repo)
    if open_prs:
        print(f"open pilot PRs exist on {cfg.repo}: {[p['url'] for p in open_prs]}\nrun `pilot reset-fixture` first")
        return EXIT_ERROR
    if not paths.FIXTURE_BASE.exists():
        print("no fixture base recorded; run `pilot reset-fixture` first")
        return EXIT_ERROR
    r = evidence.new_run()
    print(f"== run {r.id}")
    r.state["fixture_base"] = paths.FIXTURE_BASE.read_text().strip()
    paths.MODEL_HOST.write_text(cfg.model_hosts[0] + "\n")  # for the P06 probe; never in evidence
    evidence.write_config_json(r, cfg)
    evidence.write_versions(r, imgs)
    r.event("run", None, None, "run_started", cfg.repo)
    if cfg.github_issues:  # 004 US1: mirror the roadmap before anything is staged
        from . import ghsync
        try:
            ghsync.sync(cfg, r)
        except ghsync.GitHubLoopError as e:
            print(f"github sync failed: {e}")
            return _fail(r, "GITHUB_SYNC")
    if checks:
        checks.gate_matrix(r)
        if any(row["result"] != "pass" for row in evidence.read_jsonl(r.dir / "checks/results.jsonl")
               if row["id"].startswith("GM")):
            return _fail(r, "GATE_MATRIX")
    if gate_task(r, cfg, "A")["decision"] != "allow":
        return _fail(r, "A_REFUSED")
    if gate_task(r, cfg, "B")["decision"] != "refuse":
        return _fail(r, "B_NOT_BLOCKED")
    ok, ch = confirm_mode(r, cfg)
    if not ok:
        return mode_not_confirmed(r, ch, cfg)
    r.save()
    return run_task(r, cfg, imgs, runner, "A")


def resume(cfg, run_id: str | None) -> int:
    r = evidence.load_run(run_id)
    if r is None or not str(r.state.get("state", "")).startswith("paused"):
        print(f"no paused run{' ' + run_id if run_id else ''} (state={r.state.get('state') if r else None})")
        return EXIT_ERROR
    imgs = images.load()
    runner = JobsRunner()
    if github_on(cfg) and not approvals.has_tty():  # 004: a valid /approve on GitHub becomes the approval
        from . import ghapprove
        msg = ghapprove.collect(cfg, r)
        if msg:
            print(msg)
    st, pend = r.state["state"], r.state.get("pending") or {}
    r.state["state"] = "in-progress"
    r.save()
    r.event("run", None, None, "run_resumed", st)
    if st == "paused(awaiting-mode)":
        ok, ch = confirm_mode(r, cfg)
        if not ok:
            return mode_not_confirmed(r, ch, cfg)
        r.save()
        return run_task(r, cfg, imgs, runner, "A")
    if st == "paused(awaiting-retry)":
        task, n = pend["task"], int(pend["n"])
        ok, ch = confirm_retry(r, task, n, 1 + cfg.retry_limit, cfg)
        if not ok:
            _pause(r, "awaiting-retry", task=task, n=n)
            if ch == "none":
                print(f"paused: retry of {task} not confirmed. {approvals.hint(r, 'retry')}")
                github_post(cfg, r)
            return EXIT_AWAIT_RETRY
        return run_task(r, cfg, imgs, runner, task, start_n=n)
    if st == "paused(awaiting-publish)":
        return publish_task(r, cfg, imgs, runner, pend["task"])
    if st != "paused(awaiting-merge)":
        print(f"cannot resume from {st}")
        return EXIT_ERROR
    if github_on(cfg):  # 004 R6: the status change is approved by merging a status-change PR
        return reconcile_github(cfg, r, imgs, runner)
    # Human merge happened (or not): reconcile A.
    paused_at = _dt.datetime.strptime(r.state.get("paused_at", evidence.now()), "%Y-%m-%dT%H:%M:%SZ")
    tA = r.task("A")
    n = tA["passing_attempt"]
    channel = {}

    def ask_reconcile(question: str) -> str:
        ok, ch = approvals.decide(r, "reconcile", question, task="A")
        approvals.note(r, "reconcile", "A", None, ok, ch)
        channel["ch"] = ch
        return "y" if ok else ""

    rec_obj, cleanup = reconcile.for_fixture(cfg, r, "A", r.attempt_dir("A", n), tA["base_revision"],
                                             ask=ask_reconcile)
    try:
        code, rec = rec_obj.run("A", paths.TASKS["A"], tA["pr"]["number"],
                                next(a for a in tA["attempts"] if a["n"] == n).get("changed_paths") or [])
    finally:
        cleanup()
    evidence.write_json(r.dir / "reconcile" / "A.json", rec)
    r.event("reconcile", "A", n, f"reconcile_{rec.get('reason') or 'applied'}", rec.get("applied_commit") or "")
    if code == EXIT_AWAIT_MERGE:
        _pause(r, "awaiting-merge", task="A")
        print("A's PR is not merged yet. Merge it on GitHub, then run `pilot resume` again.")
        return EXIT_AWAIT_MERGE
    if code == EXIT_DECLINED:
        _pause(r, "awaiting-merge", task="A")
        if channel.get("ch") == "none":
            print(f"waiting for your approval of the status change; roadmap unchanged. {approvals.hint(r, 'reconcile')}")
        else:
            print("status change declined; roadmap unchanged; B stays blocked.")
        return EXIT_DECLINED
    if code != 0:
        return _fail(r, f"RECONCILE_{rec.get('reason')}")
    wait = int((_dt.datetime.now(_dt.timezone.utc).replace(tzinfo=None) - paused_at).total_seconds())
    r.event("reconcile", "A", None, "merge_wait_seconds", str(wait))
    tA["state"] = "verified"
    r.save()
    r.event("status", "A", None, "task_verified", rec.get("applied_commit") or "")
    github_refresh(cfg, r)
    if gate_task(r, cfg, "B")["decision"] != "allow":
        return _fail(r, "B_STILL_BLOCKED")
    return run_task(r, cfg, imgs, runner, "B")


def _pr_merged(cfg, number) -> bool:
    try:
        v = json.loads(fixture.gh("pr", "view", str(number), "-R", cfg.repo, "--json", "state").stdout)
    except subprocess.CalledProcessError:
        return False
    return v.get("state") == "MERGED"


def ready_to_resume(cfg, r) -> bool:
    """004 `github watch`: would `resume` make progress now? (approval found, or PR merged)."""
    st = str(r.state.get("state", ""))
    pend = r.state.get("pending") or {}
    if st in ("paused(awaiting-mode)", "paused(awaiting-retry)"):
        from . import ghapprove
        msg = ghapprove.collect(cfg, r)
        if msg:
            print(msg)
        return (r.dir / "approvals" / "pending.json").exists()
    if st == "paused(awaiting-merge)":
        spr = pend.get("status_pr")
        if spr:
            return _pr_merged(cfg, spr)
        pr = (r.task("A").get("pr") or {}).get("number")
        return bool(pr) and _pr_merged(cfg, pr)
    return False


def watch(cfg, run_id: str, interval: int = 15, sleep=None) -> int:
    """Poll GitHub and resume when a valid approval or a merge appears. Ctrl-C leaves it paused."""
    import time
    sleep = sleep or time.sleep
    code = EXIT_ERROR
    while True:
        r = evidence.load_run(run_id)
        if r is None:
            print(f"no run {run_id}")
            return EXIT_ERROR
        st = str(r.state.get("state", ""))
        if not st.startswith("paused"):
            print(f"run {run_id} is {st}; stopped watching")
            return code if code != EXIT_ERROR else (EXIT_OK if st == "succeeded" else EXIT_ERROR)
        if ready_to_resume(cfg, r):
            code = resume(cfg, run_id)
            after = evidence.load_run(run_id)
            if after is not None and after.state.get("state") == st and after.state.get("pending") == r.state.get("pending"):
                sleep(interval)  # no progress (e.g. declined): don't spin
            continue
        sleep(interval)


def reconcile_github(cfg, r, imgs, runner) -> int:
    """004 US3. After A's merge: check reality, open a status-change PR, pause. After the human
    merges it: check it is exactly the proposed edit, merged by an allowed approver, then verify A."""
    pend = r.state.get("pending") or {}
    tA = r.task("A")
    n = tA["passing_attempt"]
    if pend.get("status_pr"):
        return status_pr_merged(cfg, r, imgs, runner, pend)
    rec_obj, cleanup = reconcile.for_fixture(cfg, r, "A", r.attempt_dir("A", n), tA["base_revision"], ask=lambda q: "")
    base = bundle = None
    try:
        code, rec, new = rec_obj.check("A", paths.TASKS["A"], tA["pr"]["number"],
                                       next(a for a in tA["attempts"] if a["n"] == n).get("changed_paths") or [])
        if code == 0:
            base, bundle = rec_obj.status_bundle(new)
    finally:
        cleanup()
    rec["channel"] = "github"
    evidence.write_json(r.dir / "reconcile" / "A.json", rec)
    r.event("reconcile", "A", n, f"reconcile_{rec.get('reason') or 'proposed'}", rec.get("merge_commit") or "")
    if code == EXIT_AWAIT_MERGE:
        _pause(r, "awaiting-merge", task="A")
        print("A's PR is not merged yet. Merge it on GitHub, then run `pilot resume` (or `pilot github watch`).")
        return EXIT_AWAIT_MERGE
    if code != 0:
        return _fail(r, f"RECONCILE_{rec.get('reason')}")
    try:
        pr = publisher.publish_status(r, cfg, imgs, runner, "A", base, bundle)
    except Exception as e:  # noqa: BLE001 - resumable; never "verified"
        r.event("publish", "A", None, "status_pr_failed", f"{e.__class__.__name__}: {str(e)[:300]}")
        _pause(r, "awaiting-merge", task="A")
        print(f"could not open the status-change PR: {e}\n`pilot resume` tries again.")
        return EXIT_FAIL
    _pause(r, "awaiting-merge", task="A", status_pr=pr["number"], status_base=base)
    print(f"\nStatus-change PR: {pr['url']}\nIt changes only item {paths.TASKS['A']} to `verified`. "
          "Merging it on GitHub approves the status change; then run `pilot resume` (or `pilot github watch`).")
    return EXIT_AWAIT_MERGE


def status_pr_merged(cfg, r, imgs, runner, pend) -> int:
    from . import ghapprove
    num, base, item = int(pend["status_pr"]), pend["status_base"], paths.TASKS["A"]
    try:
        view = json.loads(fixture.gh("pr", "view", str(num), "-R", cfg.repo, "--json",
                                     "state,mergedBy,headRefOid,mergeCommit").stdout)
    except subprocess.CalledProcessError as e:
        _pause(r, "awaiting-merge", task="A", status_pr=num, status_base=base)
        print(f"could not read status-change PR #{num}: {e}; the run stays paused")
        return EXIT_ERROR
    if view.get("state") != "MERGED":
        _pause(r, "awaiting-merge", task="A", status_pr=num, status_base=base)
        print(f"status-change PR #{num} is not merged yet. Merge it on GitHub to approve, then `pilot resume`.")
        return EXIT_AWAIT_MERGE
    clone, _ = fixture.fresh_clone(cfg.repo, cfg.branch, depth=None)
    try:
        head = view["headRefOid"]
        fixture.git("fetch", "--quiet", "origin", f"pull/{num}/head", cwd=clone)
        changed = fixture.git("diff", "--name-only", base, head, cwd=clone).stdout.split()
        show = lambda rev: fixture.git("show", f"{rev}:{reconcile.ROADMAP_REL}", cwd=clone, check=False).stdout  # noqa: E731
        base_text, head_text = show(base), show(head)
        on_default = re.search(rf"^### {re.escape(item)} — .*\[status: verified\]", show("HEAD"), re.M) is not None
    finally:
        shutil.rmtree(clone, ignore_errors=True)
    ok, reason = reconcile.check_status_pr(view, base_text, head_text, changed, item, cfg.approvers,
                                           ghapprove.publisher_login(cfg, r))
    if ok and not on_default:
        ok, reason = False, "STATUS_NOT_ON_DEFAULT"
    merger = (view.get("mergedBy") or {}).get("login")
    evidence.write_json(r.dir / "reconcile" / "A-status.json",
                        {"pr": num, "base": base, "head": view.get("headRefOid"),
                         "merge_commit": (view.get("mergeCommit") or {}).get("oid"), "merged_by": merger,
                         "changed": changed, "ok": ok, "reason": reason, "at": evidence.now()})
    r.event("reconcile", "A", None, "status_pr_checked", f"pr={num} ok={str(ok).lower()} reason={reason}")
    if not ok:
        approvals.note(r, "reconcile", "A", None, False, "github", approver=merger, extra=f"pr={num} reason={reason}")
        return _fail(r, reason)
    approvals.note(r, "reconcile", "A", None, True, "github", approver=merger, extra=f"pr={num}")
    tA = r.task("A")
    tA["state"] = "verified"
    r.save()
    r.event("status", "A", None, "task_verified", (view.get("mergeCommit") or {}).get("oid") or "")
    github_refresh(cfg, r)
    if gate_task(r, cfg, "B")["decision"] != "allow":
        return _fail(r, "B_STILL_BLOCKED")
    return run_task(r, cfg, imgs, runner, "B")
