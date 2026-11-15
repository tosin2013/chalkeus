"""004 US3: the status-change PR — exactly one heading edit, merged by an allowed human (R6)."""
import json
import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest import mock

from specfoundry_pilot import approvals, evidence, reconcile

ROADMAP = """# r

### 001 — Slug function  [status: planned]

- x

### 002 — Unique slugs  [status: planned]
"""
EDITED = reconcile.status_edit(ROADMAP, "001")


class TestCheckStatusPR(unittest.TestCase):
    def chk(self, view=None, head=EDITED, changed=None, approvers=("alice",), publisher="pub-bot"):
        view = view or {"state": "MERGED", "mergedBy": {"login": "alice"}}
        return reconcile.check_status_pr(view, ROADMAP, head, changed if changed is not None else [reconcile.ROADMAP_REL],
                                         "001", list(approvers), publisher)

    def test_exact_edit_by_approver(self):
        self.assertEqual(self.chk(), (True, None))

    def test_not_merged(self):
        self.assertEqual(self.chk(view={"state": "OPEN"}), (False, "NOT_MERGED"))

    def test_extra_file_or_extra_change_is_edited(self):
        self.assertEqual(self.chk(changed=[reconcile.ROADMAP_REL, "README.md"])[1], "STATUS_PR_EDITED")
        self.assertEqual(self.chk(head=EDITED.replace("- x", "- y"))[1], "STATUS_PR_EDITED")
        self.assertEqual(self.chk(head=reconcile.status_edit(ROADMAP, "002"))[1], "STATUS_PR_EDITED")

    def test_merged_by_publisher_or_non_approver(self):
        self.assertEqual(self.chk(view={"state": "MERGED", "mergedBy": {"login": "pub-bot"}}, approvers=("alice", "pub-bot"))[1],
                         "STATUS_PR_MERGED_BY_PUBLISHER")
        self.assertEqual(self.chk(view={"state": "MERGED", "mergedBy": {"login": "bob"}})[1], "STATUS_PR_NOT_APPROVER")


class TestStatusBundle(unittest.TestCase):
    """The bundle built by for_fixture's status_bundle holds exactly one commit with the edit."""
    def test_bundle_has_only_the_edit(self):
        with tempfile.TemporaryDirectory() as t:
            repo = Path(t) / "repo"
            (repo / ".specify/memory").mkdir(parents=True)
            (repo / reconcile.ROADMAP_REL).write_text(ROADMAP)
            g = lambda *a: subprocess.run(["git", *a], cwd=repo, check=True, capture_output=True, text=True)  # noqa: E731
            g("init", "-q", "-b", "main")
            g("-c", "user.email=t@e", "-c", "user.name=t", "add", ".")
            g("-c", "user.email=t@e", "-c", "user.name=t", "commit", "-qm", "base")
            base = g("rev-parse", "HEAD").stdout.strip()
            run = mock.Mock(id="R1", dir=Path(t))
            real = subprocess.run
            with mock.patch.object(reconcile.fixture, "fresh_clone", return_value=(repo, base)), \
                 mock.patch.object(reconcile.fixture, "git", side_effect=lambda *a, cwd=None, check=True: real(
                     ["git", "-c", "user.email=t@e", "-c", "user.name=t", *a], cwd=cwd, check=check, capture_output=True, text=True)), \
                 mock.patch.object(reconcile.subprocess, "run", side_effect=lambda *a, **k: subprocess.CompletedProcess(a, 0, "", "")):
                rec_obj, _ = reconcile.for_fixture(mock.Mock(repo="o/r", branch="main"), run, "A", Path(t), base)
                b, bundle = rec_obj.status_bundle(EDITED)
            self.assertEqual(b, base)
            v = subprocess.run(["git", "bundle", "list-heads", str(bundle)], capture_output=True, text=True)
            self.assertEqual(len(v.stdout.splitlines()), 1)
            diff = g("diff", "--name-only", base, "HEAD").stdout.split()
            self.assertEqual(diff, [reconcile.ROADMAP_REL])


class TestApprovalNote(unittest.TestCase):
    def test_merger_recorded_as_github_approver(self):
        with tempfile.TemporaryDirectory() as t:
            r = evidence.Run(Path(t) / "R")
            r.state = {"run_id": "R", "state": "in-progress"}
            r.save()
            r.last_approval = {"approver": "old", "comment_id": 5}
            approvals.note(r, "reconcile", "A", None, True, "github", approver="alice", extra="pr=9")
            ev = [json.loads(x) for x in (r.dir / "events.jsonl").read_text().splitlines()][-1]
            self.assertEqual(ev["detail"], "step=reconcile approved=true channel=github approver=alice pr=9")


if __name__ == "__main__":
    unittest.main()
