"""Grader (runs in a fresh `pilot-g-<task>-<n>` namespace with no secrets and no egress; R7).

Inputs  /in/meta.json {run_id, task, attempt, item_id, base_revision, allowed_paths}
        /in/base.bundle, /in/change.bundle (may be absent)
        /grading/test_hidden_<item_id>.py  (grading material: never mounted in a worker)
Output  stdout frame with grade.json (contracts/grade-report.schema.json)

Item runs (014, specs/014-any-project/contracts/grading.md) add optional meta fields:
  mode               "attempt" (default) | "baseline": no change bundle; G4 must pass, G5 must fail
  test_command       run with `sh -c` from the project root for G4 and G5 (else pytest, as the fixture)
  hidden_mode        "overlay": copy /grading/overlay/ over the project, then run test_command again
  command_timeout_s  per-command timeout (default 180)
  test_paths         globs of the project's test files (the target's PILOT_TEST_PATHS). When set,
                     two more checks run on an attempt:
                       G6_tests_changed        the change adds or edits at least one test file
                       G7_existing_tests_kept  the base's versions of the test files the change
                                               edited or deleted still pass against the changed
                                               code (an agent can't weaken a test to pass)
  tests_may_change   globs of existing tests the item's spec says may change (skipped by G7)
  author_check       true for `pilot author check` (018): the baseline's hidden-test output is
                     returned in a separate frame file, hidden_output.txt, for the test author's
                     own terminal; grade.json keeps only the withheld summary
Without them, grading is exactly the fixture's.
"""
from __future__ import annotations

import fnmatch
import json
import os
import shutil
import subprocess
import sys
import tempfile

CHECKS = ["G1_bundle_applies", "G2_allowed_paths", "G3_tasks_checked", "G4_visible_tests", "G5_hidden_tests"]
TEST_CHECKS = ["G6_tests_changed", "G7_existing_tests_kept"]
AUTHOR_OUTPUT: list = []  # author_check only: the hidden tests' output, never put in grade.json


def matches(path, globs):
    return any(fnmatch.fnmatch(path, g) for g in globs or [])


AUTHOR_OUTPUT_LIMIT = 200_000  # 018: the test author sees enough to tell why each of its tests fails


def run_full(cmd, cwd=None, timeout=180):
    """For output the grader parses (file lists, spec status JSON): the whole stdout, never cut.
    run() keeps only an output's tail for evidence; parsing a tail silently drops data (a large
    project's spec status JSON, or files in a large change)."""
    try:
        r = subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout)
        return r.returncode, r.stdout, r.stderr[-1000:]
    except subprocess.TimeoutExpired:
        return 124, "", "timeout"


def run(cmd, cwd=None, env=None, timeout=180, limit=3000):
    try:
        r = subprocess.run(cmd, cwd=cwd, env=env, capture_output=True, text=True, timeout=timeout)
        return r.returncode, (r.stdout + r.stderr)[-limit:]
    except subprocess.TimeoutExpired:
        return 124, "timeout"


def grade(meta: dict, inp: str = "/in", grading: str = "/grading", work: str | None = None) -> dict:
    work = work or tempfile.mkdtemp(prefix="grade-")
    repo = os.path.join(work, "repo")
    results = []

    def add(cid, ok, detail=""):
        results.append({"id": cid, "result": "pass" if ok else "fail", "detail": str(detail)[-4000:]})
        return ok

    base = meta["base_revision"]
    if meta.get("mode") == "baseline":
        return baseline(meta, inp, grading, work, repo)
    change = os.path.join(inp, "change.bundle")
    ok = os.path.exists(change)
    detail = "no change.bundle from worker"
    if ok:
        rc1, o1 = run(["git", "clone", "--quiet", os.path.join(inp, "base.bundle"), repo])
        rc2, o2 = (run(["git", "checkout", "--quiet", base], cwd=repo) if rc1 == 0 else (1, ""))
        rc3, o3 = (run(["git", "bundle", "verify", change], cwd=repo) if rc2 == 0 else (1, ""))
        rc4, o4 = (run(["git", "fetch", "--quiet", change, "HEAD:refs/pilot/graded"], cwd=repo) if rc3 == 0 else (1, ""))
        rc5, o5 = (run(["git", "merge-base", "--is-ancestor", base, "refs/pilot/graded"], cwd=repo) if rc4 == 0 else (1, ""))
        rc6, o6 = (run(["git", "checkout", "--quiet", "refs/pilot/graded"], cwd=repo) if rc5 == 0 else (1, ""))
        ok = rc6 == 0
        detail = " | ".join(x for x in (o1, o2, o3, o4, o5, o6) if x)[:2000] or "fast-forward from base"
    if add("G1_bundle_applies", ok, detail):
        rc, out, _ = run_full(["git", "diff", "--name-only", base, "HEAD"], cwd=repo)
        changed = [p for p in out.splitlines() if p]
        bad = [p for p in changed if not any(fnmatch.fnmatch(p, g) for g in meta["allowed_paths"])]
        if add("G2_allowed_paths", rc == 0 and not bad and bool(changed), f"changed={changed} disallowed={bad}"):
            rc, out, err = run_full(["spec", "status", "--json", "--root", repo])
            pct, why = None, ""
            try:
                items = json.loads(out if rc in (0, 1) else "{}")["items"]
                item = next((i for i in items if i["id"] == meta["item_id"]), None)
                if item is None:
                    why = f" (item {meta['item_id']} not in spec status output)"
                else:
                    pct = (item.get("tasks") or {}).get("percent")
            except Exception as e:  # noqa: BLE001
                why = f" (spec status unreadable: rc={rc} {type(e).__name__}; {err.strip()[:200]})"
            if add("G3_tasks_checked", pct == 100, f"tasks percent={pct}{why}"):
                if meta.get("test_command"):  # an item run (014)
                    if not meta.get("test_paths") or tests_changed(meta, repo, base, add):
                        visible_and_hidden(meta, grading, repo, add, work, base)
                else:  # the fixture, unchanged
                    env = {**os.environ, "PYTHONDONTWRITEBYTECODE": "1"}
                    rc, out = run([sys.executable, "-m", "pytest", "-q", "-p", "no:cacheprovider", "tests/"], cwd=repo, env=env)
                    if add("G4_visible_tests", rc == 0, out):
                        hidden = os.path.join(grading, f"test_hidden_{meta['item_id']}.py")
                        tdir = os.path.join(work, "hidden")
                        os.makedirs(tdir, exist_ok=True)
                        shutil.copy(hidden, tdir)
                        env["PYTHONPATH"] = os.path.join(repo, "src")
                        rc, out = run([sys.executable, "-m", "pytest", "-q", "-p", "no:cacheprovider",
                                       os.path.join(tdir, os.path.basename(hidden))], cwd=tdir, env=env)
                        add("G5_hidden_tests", rc == 0, out)
    expected = CHECKS + (TEST_CHECKS if meta.get("test_command") and meta.get("test_paths") else [])
    done = {r["id"] for r in results}
    for cid in expected:
        if cid not in done:
            results.append({"id": cid, "result": "skipped", "detail": "earlier check failed"})
    passed = all(r["result"] == "pass" for r in results) and len(results) == len(expected)
    return {"version": 1, "run_id": meta["run_id"], "task": meta["task"], "attempt": int(meta["attempt"]),
            "result": "pass" if passed else "fail", "failure": None,
            "grader_image": os.environ.get("GRADER_IMAGE", "unknown"), "checks": results}


def project_tests(meta, repo, limit=3000):
    """Run the project's own test command from the project root (item runs)."""
    return run(["sh", "-c", meta["test_command"]], cwd=repo, timeout=int(meta.get("command_timeout_s") or 180), limit=limit)


def overlay(grading, repo):
    """Copy the item's hidden tests over the project; they mirror its layout, and win."""
    src = os.path.join(grading, "overlay")
    if not os.path.isdir(src) or not os.listdir(src):
        return False, "no hidden tests in /grading/overlay"
    shutil.copytree(src, repo, dirs_exist_ok=True)
    return True, ""


def changes(repo, base):
    """(status, path) pairs of the change against the base: A added, M modified, D deleted."""
    rc, out, _ = run_full(["git", "diff", "--name-status", "--no-renames", base, "HEAD"], cwd=repo)
    pairs = []
    for line in out.splitlines() if rc == 0 else []:
        parts = line.split("\t", 1)
        if len(parts) == 2:
            pairs.append((parts[0][:1], parts[1]))
    return pairs


def tests_changed(meta, repo, base, add):
    """G6: a change must add or edit a test file (the project's rule, PILOT_TEST_PATHS)."""
    tests = [p for st, p in changes(repo, base) if st in ("A", "M") and matches(p, meta["test_paths"])]
    return add("G6_tests_changed", bool(tests),
               f"test files added or edited: {tests}" if tests else
               f"the change adds or edits no test file matching {meta['test_paths']}: add tests for the change")


def existing_tests_kept(meta, repo, work, base, add):
    """G7: put back the base's version of every test file the change edited or deleted (except
    those the spec says may change), and run the project's tests against the changed code."""
    touched = [p for st, p in changes(repo, base)
               if st in ("M", "D") and matches(p, meta["test_paths"]) and not matches(p, meta.get("tests_may_change"))]
    if not touched:
        return add("G7_existing_tests_kept", True, "no existing test file was edited or deleted")
    # A clean checkout of the change (no build output or bytecode left by G4), then the base's
    # versions of the touched test files on top.
    orig = os.path.join(work, "original-tests")
    rc, out = run(["git", "worktree", "add", "--quiet", "--detach", orig, "HEAD"], cwd=repo)
    if rc == 0:
        rc, out = run(["git", "checkout", "--quiet", base, "--", *touched], cwd=orig)
    if rc != 0:
        return add("G7_existing_tests_kept", False, f"could not restore {touched}: {out}")
    rc, out = project_tests(meta, orig)
    run(["git", "worktree", "remove", "--force", orig], cwd=repo)
    return add("G7_existing_tests_kept", rc == 0,
               (f"the original versions of {touched} still pass\n" if rc == 0 else
                f"the original versions of {touched} FAIL against the change: an existing test was "
                "weakened or removed; keep it passing, or list it in the spec under "
                "'## Tests that may change'\n") + out)


def hidden_summary(rc, out):
    """What an item run's G5 may record: the exit code and the size and hash of the output,
    never the output itself. Test runners print failing assertions and source lines, and grade
    evidence is kept and may be shared, so the hidden tests' contents must not reach it (C-14)."""
    import hashlib
    digest = hashlib.sha256(out.encode()).hexdigest()
    return f"exit={rc} output_lines={len(out.splitlines())} output_sha256={digest} (output withheld: hidden tests)"


def visible_and_hidden(meta, grading, repo, add, work=None, base=None):
    rc, out = project_tests(meta, repo)
    if add("G4_visible_tests", rc == 0, out):
        if meta.get("test_paths") and not existing_tests_kept(meta, repo, work, base, add):
            return
        ok, why = overlay(grading, repo)
        if not ok:
            add("G5_hidden_tests", False, why)
            return
        rc, out = project_tests(meta, repo)
        add("G5_hidden_tests", rc == 0, hidden_summary(rc, out))


def baseline(meta, inp, grading, work, repo):
    """Before any attempt: on the base revision the project's tests must pass and the hidden
    tests must fail, or the hidden tests can't tell a correct change from no change (FR-007)."""
    results = []

    def add(cid, result, detail=""):
        results.append({"id": cid, "result": result, "detail": str(detail)[-4000:]})
        return result == "pass"

    rc1, o1 = run(["git", "clone", "--quiet", os.path.join(inp, "base.bundle"), repo])
    rc2, o2 = (run(["git", "checkout", "--quiet", meta["base_revision"]], cwd=repo) if rc1 == 0 else (1, ""))
    if add("G1_bundle_applies", "pass" if rc2 == 0 else "fail", " | ".join(x for x in (o1, o2) if x)[:2000] or "base checked out"):
        add("G2_allowed_paths", "skipped", "baseline: no change")
        add("G3_tasks_checked", "skipped", "baseline: no change")
        rc, out = project_tests(meta, repo)
        if add("G4_visible_tests", "pass" if rc == 0 else "fail", out):
            ok, why = overlay(grading, repo)
            if not ok:
                add("G5_hidden_tests", "fail", why)
            else:
                rc, out = project_tests(meta, repo, AUTHOR_OUTPUT_LIMIT if meta.get("author_check") else 3000)
                if meta.get("author_check"):
                    AUTHOR_OUTPUT.append(out)
                # Here a pass means the hidden tests FAILED on the base, as they must.
                add("G5_hidden_tests", "pass" if rc != 0 else "fail",
                    ("hidden tests fail on the base revision (expected)\n" if rc != 0 else
                     "hidden tests PASS on the base revision: they carry no signal\n") + hidden_summary(rc, out))
    done = {r["id"] for r in results}
    for cid in CHECKS:
        if cid not in done:
            results.append({"id": cid, "result": "skipped", "detail": "earlier check failed"})
    by = {r["id"]: r["result"] for r in results}
    passed = by["G1_bundle_applies"] == "pass" and by["G4_visible_tests"] == "pass" and by["G5_hidden_tests"] == "pass"
    return {"version": 1, "run_id": meta["run_id"], "task": meta["task"], "attempt": int(meta["attempt"]),
            "mode": "baseline", "result": "pass" if passed else "fail", "failure": None,
            "grader_image": os.environ.get("GRADER_IMAGE", "unknown"), "checks": results}


def main() -> int:
    sys.path.insert(0, "/usr/local/lib/pilot")
    import frame  # noqa: E402
    meta = json.load(open("/in/meta.json"))
    try:
        report = grade(meta)
    except Exception as e:  # noqa: BLE001 - any grader crash is a fail
        report = {"version": 1, "run_id": meta.get("run_id"), "task": meta.get("task"),
                  "attempt": int(meta.get("attempt", 1)), "result": "fail", "failure": "grader_error",
                  "grader_image": os.environ.get("GRADER_IMAGE", "unknown"),
                  "checks": [{"id": c, "result": "skipped", "detail": f"grader crashed: {e}"[:400]} for c in CHECKS]}
    tmp = tempfile.mkdtemp()
    out = os.path.join(tmp, "grade.json")
    with open(out, "w") as f:
        json.dump(report, f)
    files = [f"grade.json={out}"]
    if meta.get("author_check") and AUTHOR_OUTPUT:
        hidden = os.path.join(tmp, "hidden_output.txt")
        with open(hidden, "w") as f:
            f.write(AUTHOR_OUTPUT[-1])
        files.append(f"hidden_output.txt={hidden}")
    frame.main(["emit", *files])
    return 0


if __name__ == "__main__":
    sys.exit(main())
