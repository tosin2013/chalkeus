"""004 US1: roadmap -> item issues, status labels, task checklists; idempotent sync; drift (R1, R8)."""
import json
import tempfile
import unittest
from pathlib import Path
from unittest import mock

from specfoundry_pilot import ghsync
from tests.ghfake import FakeGitHub

DOC = {"items": [
    {"id": "001", "title": "Slug function", "status": "planned", "active": True, "description": "A slugify.",
     "outcome": "It works.", "spec_dir": "specs/001-slugify", "tasks": {"source": "specs/001-slugify/tasks.md"}},
    {"id": "002", "title": "Unique slugs", "status": "planned", "active": True, "description": "Unique.",
     "outcome": "No collisions.", "spec_dir": "specs/002-unique-slug", "tasks": {"source": "specs/002-unique-slug/tasks.md"}},
    {"id": "000", "title": "Old", "status": "abandoned", "active": False},
]}
TEXTS = {"001": "# Tasks\n\n- [x] T001 Implement slugify\n- [ ] T002 Export it\nnot a task\n", "002": None}


class Cfg:
    repo = "o/x-pilot-fixture"
    branch = "main"
    credential = "token"


class TestPure(unittest.TestCase):
    def test_body_has_marker_status_and_checklist(self):
        d = ghsync.desired_state(DOC, TEXTS)
        self.assertEqual(sorted(d), ["001", "002"])  # inactive items are history
        b = d["001"]["body"]
        self.assertTrue(b.startswith("<!-- specfoundry:item=001 -->"))
        self.assertIn("status `planned`", b)
        self.assertIn("- [x] T001 Implement slugify", b)
        self.assertIn("- [ ] T002 Export it", b)
        self.assertNotIn("not a task", b)
        self.assertEqual(d["001"]["label"], "status:planned")
        self.assertNotIn("### Tasks", d["002"]["body"])

    def test_unknown_status_maps_to_undecided(self):
        doc = {"items": [dict(DOC["items"][0], status="done")]}
        self.assertEqual(ghsync.desired_state(doc, {})["001"]["label"], "status:undecided")

    def test_issues_matched_by_marker_not_title(self):
        issues = [{"number": 5, "title": "001 — Slug function", "body": "no marker", "labels": [], "state": "open"}]
        ops = ghsync.plan_ops("o/r", ghsync.desired_state(DOC, TEXTS), issues, set())
        self.assertEqual([o for o, _ in ops].count("create-issue"), 2)


class TestSync(unittest.TestCase):
    def setUp(self):
        self.gh = FakeGitHub().patch(self)
        p = mock.patch.object(ghsync, "_snapshot", lambda cfg: (DOC, TEXTS))
        p.start()
        self.addCleanup(p.stop)

    def test_first_sync_creates_one_issue_per_item_and_labels(self):
        s = ghsync.sync(Cfg())
        self.assertEqual((s["created"], s["changed"]), (2, 0))
        self.assertEqual(sorted(s["issues"]), ["001", "002"])
        self.assertIn("status:verified", self.gh.labels)
        for i in self.gh.issues:
            self.assertEqual([lab for lab in i["labels"] if lab.startswith("status:")], ["status:planned"])

    def test_second_sync_is_a_no_op(self):
        ghsync.sync(Cfg())
        n = len(self.gh.ops)
        s = ghsync.sync(Cfg())
        self.assertEqual((s["created"], s["changed"]), (0, 0))
        self.assertEqual(len(self.gh.ops), n)

    def test_hand_edited_label_is_restored_other_labels_kept(self):
        ghsync.sync(Cfg())
        self.gh.issues[0]["labels"] = ["status:verified", "good first issue"]
        self.assertEqual(ghsync.drift_report(ghsync.desired_state(DOC, TEXTS), self.gh.issues)[0]["kind"], "label_mismatch")
        s = ghsync.sync(Cfg())
        self.assertEqual(s["changed"], 1)
        self.assertEqual(sorted(self.gh.issues[0]["labels"]), ["good first issue", "status:planned"])

    def test_sync_never_closes_or_deletes(self):
        ghsync.sync(Cfg())
        self.gh.issues.append({"number": 99, "title": "x", "body": "<!-- specfoundry:item=777 -->", "labels": [], "state": "open"})
        ghsync.sync(Cfg())
        self.assertNotIn("close-issue", [o for o, _ in self.gh.ops])

    def test_github_error_raises(self):
        self.gh.fail = "read"
        with self.assertRaises(RuntimeError):
            ghsync.sync(Cfg())

    def test_records_evidence_with_run(self):
        with tempfile.TemporaryDirectory() as d:
            run = mock.Mock(dir=Path(d))
            ghsync.sync(Cfg(), run)
            self.assertEqual(json.loads((Path(d) / "github/issues.json").read_text())["issues"], {"001": 1, "002": 2})
            run.event.assert_called_once()


class TestDrift(unittest.TestCase):
    def d(self, **over):
        doc = {"items": [dict(DOC["items"][0], **over), DOC["items"][1]]}
        return ghsync.desired_state(doc, TEXTS)

    def issues(self, state1="open", label1="status:planned"):
        return [{"number": 1, "title": "", "body": "<!-- specfoundry:item=001 -->", "labels": [label1], "state": state1},
                {"number": 2, "title": "", "body": "<!-- specfoundry:item=002 -->", "labels": ["status:planned"], "state": "open"}]

    def kinds(self, rep):
        return sorted(r["kind"] for r in rep)

    def test_no_drift(self):
        self.assertEqual(ghsync.drift_report(self.d(), self.issues()), [])

    def test_missing_orphan_duplicate(self):
        iss = self.issues()[:1] + [{"number": 3, "title": "", "body": "<!-- specfoundry:item=009 -->", "labels": [], "state": "open"},
                                   {"number": 4, "title": "", "body": "<!-- specfoundry:item=001 -->", "labels": [], "state": "open"}]
        self.assertEqual(self.kinds(ghsync.drift_report(self.d(), iss)), ["duplicate_issue", "missing_issue", "orphan_issue"])

    def test_state_vs_status(self):
        self.assertIn("open_but_verified", self.kinds(ghsync.drift_report(self.d(status="verified"), self.issues(label1="status:verified"))))
        self.assertIn("closed_but_not_verified", self.kinds(ghsync.drift_report(self.d(), self.issues(state1="closed"))))

    def test_task_pr_must_close_its_issue(self):
        prs = [{"task": "A", "number": 7, "kind": "task", "closes": None}, {"task": "A", "number": 8, "kind": "status"}]
        rep = ghsync.drift_report(self.d(), self.issues(), prs, {"001": 1})
        self.assertEqual(self.kinds(rep), ["pr_without_closes"])
        prs[0]["closes"] = 1
        self.assertEqual(ghsync.drift_report(self.d(), self.issues(), prs, {"001": 1}), [])


if __name__ == "__main__":
    unittest.main()
