"""Reconcile after a human merge: check reality, propose `planned -> verified`, ask, apply.

Deterministic pilot equivalent of the spec-status `reconcile` skill (research R9). The
status edit is pushed with the operator's own git credentials (a human action under
constitution V), never with a worker, grader or publisher credential.
"""
from __future__ import annotations

import json
import re
import shutil
import subprocess
from dataclasses import dataclass
from pathlib import Path
from typing import Callable

from . import evidence, fixture, paths

ROADMAP_REL = ".specify/memory/roadmap.md"


def status_edit(roadmap_text: str, item_id: str) -> str:
    pat = re.compile(rf"^(### {re.escape(item_id)} — .*?\[status: )planned(\])[ \t]*$", re.M)
    hits = pat.findall(roadmap_text)
    if len(hits) != 1:
        raise ValueError(f"expected exactly one planned heading for {item_id}, found {len(hits)}")
    return pat.sub(lambda m: f"{m.group(1)}verified{m.group(2)}", roadmap_text, count=1)


@dataclass
class Reconciler:
    pr_view: Callable[[int], dict]
    tasks_percent: Callable[[str], int | None]
    default_file: Callable[[str], bytes | None]
    graded_file: Callable[[str], bytes | None]
    roadmap_text: Callable[[], str]
    push: Callable[[str], str]
    ask: Callable[[str], str]
    status_bundle: Callable[[str], tuple[str, Path]] | None = None  # 004: (base_rev, bundle) for a status PR

    def check(self, task: str, item_id: str, pr_number: int, changed_paths: list[str]) -> tuple[int, dict, str | None]:
        """The reality checks only (merged, tasks 100%, tree matches graded). Returns
        (0, rec, new_roadmap_text) when a proposal can be made, else (code, rec, None)."""
        rec = {"task": task, "item_id": item_id, "pr": pr_number, "pr_merged": False, "merge_commit": None,
               "tasks_percent": None, "tree_matches_graded": None, "mismatched_paths": [],
               "proposed": None, "approved": False, "applied_commit": None, "reason": None, "at": evidence.now()}
        pr = self.pr_view(pr_number)
        rec["pr_merged"] = bool(pr.get("state") == "MERGED" and pr.get("mergedAt"))
        rec["merge_commit"] = (pr.get("mergeCommit") or {}).get("oid")
        if not rec["pr_merged"]:
            rec["reason"] = "NOT_MERGED"
            return 4, rec, None
        rec["tasks_percent"] = self.tasks_percent(item_id)
        if rec["tasks_percent"] != 100:
            rec["reason"] = "TASKS_INCOMPLETE"
            return 1, rec, None
        mism = [p for p in changed_paths if self.default_file(p) != self.graded_file(p)]
        rec["tree_matches_graded"] = not mism
        rec["mismatched_paths"] = mism
        if mism:
            rec["reason"] = "TREE_MISMATCH"
            return 1, rec, None
        old = self.roadmap_text()
        new = status_edit(old, item_id)
        diff = [f"- {a}\n+ {b}" for a, b in zip(old.splitlines(), new.splitlines()) if a != b]
        rec["proposed"] = "planned→verified"
        print(f"Reconcile proposal for task {task} (item {item_id}), PR #{pr_number} merged "
              f"({rec['merge_commit']}), tasks 100%, graded tree matches:\n" + "\n".join(diff))
        return 0, rec, new

    def run(self, task: str, item_id: str, pr_number: int, changed_paths: list[str]) -> tuple[int, dict]:
        code, rec, new = self.check(task, item_id, pr_number, changed_paths)
        if code != 0:
            return code, rec
        ans = (self.ask(f"Apply '{item_id}: planned → verified' to the fixture roadmap? [y/N] ") or "").strip().lower()
        if ans != "y":
            rec["reason"] = "DECLINED"
            return 5, rec
        rec["approved"] = True
        rec["applied_commit"] = self.push(new)
        return 0, rec


def check_status_pr(view: dict, base_text: str, head_text: str | None, changed: list[str], item_id: str,
                    approvers: list[str], publisher: str | None) -> tuple[bool, str | None]:
    """004 R6: is a merged status-change PR exactly the proposed edit, merged by an allowed human?
    `view` has state, mergedBy; `head_text` is the roadmap at the PR head; `changed` the files the
    PR head changes against its base. Returns (ok, reason)."""
    if view.get("state") != "MERGED":
        return False, "NOT_MERGED"
    if changed != [ROADMAP_REL] or head_text != status_edit(base_text, item_id):
        return False, "STATUS_PR_EDITED"
    merger = ((view.get("mergedBy") or {}).get("login") or "")
    if publisher and merger.lower() == publisher.lower():
        return False, "STATUS_PR_MERGED_BY_PUBLISHER"
    if merger.lower() not in {a.lower() for a in approvers}:
        return False, "STATUS_PR_NOT_APPROVER"
    return True, None


def ask_tty(question: str) -> str:
    """Ask on the controlling terminal. No terminal (or any error) -> "" (decline, fail closed).

    A tty is not seekable, so it must be opened separately for writing and reading
    (text mode "r+" raises io.UnsupportedOperation).
    """
    try:
        with open("/dev/tty", "w") as out, open("/dev/tty", "r") as inp:
            out.write(question)
            out.flush()
            return inp.readline()
    except (OSError, ValueError):
        return ""


def for_fixture(cfg, run, task: str, attempt_dir: Path, base_rev: str, ask=ask_tty) -> tuple[Reconciler, Callable[[], None]]:
    """Build a Reconciler bound to a fresh full clone of the fixture default branch."""
    clone, _ = fixture.fresh_clone(cfg.repo, cfg.branch, depth=None)
    graded = Path(run.dir / f".graded-{task}")
    shutil.rmtree(graded, ignore_errors=True)
    base = base_rev
    subprocess.run(["git", "clone", "--quiet", str(clone), str(graded)], check=True)
    subprocess.run(["git", "-C", str(graded), "checkout", "--quiet", base], check=True)
    subprocess.run(["git", "-C", str(graded), "fetch", "--quiet", str(attempt_dir / "change.bundle"),
                    "HEAD:refs/pilot/graded"], check=True)
    subprocess.run(["git", "-C", str(graded), "checkout", "--quiet", "refs/pilot/graded"], check=True)

    def read(root: Path, p: str):
        f = root / p
        return f.read_bytes() if f.is_file() else None

    def tasks_percent(item_id):
        r = subprocess.run(["spec", "status", "--json", "--root", str(clone)], capture_output=True, text=True)
        doc = json.loads(r.stdout)
        it = next(i for i in doc["items"] if i["id"] == item_id)
        return (it.get("tasks") or {}).get("percent")

    def pr_view(n):
        out = fixture.gh("pr", "view", str(n), "-R", cfg.repo, "--json", "state,mergedAt,mergeCommit").stdout
        return json.loads(out)

    def push(text):
        (clone / ROADMAP_REL).write_text(text)
        fixture.git("add", ROADMAP_REL, cwd=clone)
        fixture.git("commit", "--quiet", "-m", f"pilot: {paths.TASKS[task]} verified (run {run.id})", cwd=clone)
        fixture.git("push", "--quiet", fixture.url(cfg.repo), f"HEAD:refs/heads/{cfg.branch}", cwd=clone)
        return fixture.git("rev-parse", "HEAD", cwd=clone).stdout.strip()

    def status_bundle(text):
        """Commit exactly the status edit on top of the default branch and bundle it (004 R6)."""
        base = fixture.git("rev-parse", "HEAD", cwd=clone).stdout.strip()
        (clone / ROADMAP_REL).write_text(text)
        fixture.git("add", ROADMAP_REL, cwd=clone)
        fixture.git("commit", "--quiet", "-m", f"pilot: {paths.TASKS[task]} verified (run {run.id})", cwd=clone)
        out = run.dir / f".status-{task}.bundle"
        fixture.git("bundle", "create", "--quiet", str(out), f"{base}..HEAD", cwd=clone)
        return base, out

    r = Reconciler(pr_view=pr_view, tasks_percent=tasks_percent, default_file=lambda p: read(clone, p),
                   graded_file=lambda p: read(graded, p), roadmap_text=lambda: (clone / ROADMAP_REL).read_text(),
                   push=push, ask=ask, status_bundle=status_bundle)

    def cleanup():
        shutil.rmtree(clone, ignore_errors=True)
        shutil.rmtree(graded, ignore_errors=True)

    return r, cleanup
