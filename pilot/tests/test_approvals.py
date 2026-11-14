"""Approvals without a terminal: one pending step, bound, used once, channel recorded."""
import json
import tempfile
import unittest
from pathlib import Path
from unittest import mock

from specfoundry_pilot import approvals, audit, evidence, pipeline


def make_run(d: Path, state: str, pending=None) -> evidence.Run:
    (d / "reconcile").mkdir(parents=True, exist_ok=True)
    r = evidence.Run(d)
    r.state = {"run_id": d.name, "state": state, "pending": pending or {}, "event_seq": 0,
               "proposed_mode": {"name": "step-by-step"}}
    r.save()
    return r


class Base(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.d = Path(self.tmp.name) / "20990101T000000Z"
        self.d.mkdir()
        self.no_tty = mock.patch.object(approvals, "has_tty", return_value=False)
        self.no_tty.start()

    def tearDown(self):
        self.no_tty.stop()
        self.tmp.cleanup()


class TestRecord(Base):
    def test_not_paused_refused(self):
        r = make_run(self.d, "in-progress")
        self.assertEqual(approvals.record(r, "mode")[0], 2)

    def test_wrong_step_refused(self):
        r = make_run(self.d, "paused(awaiting-retry)", {"task": "A", "n": 2})
        code, msg = approvals.record(r, "mode")
        self.assertEqual(code, 2)
        self.assertIn("retry", msg)

    def test_merge_is_never_approved_here(self):
        r = make_run(self.d, "paused(awaiting-merge)", {"task": "A"})
        code, msg = approvals.record(r, "reconcile")
        self.assertEqual(code, 4)
        self.assertIn("GitHub", msg)

    def test_reconcile_only_after_proposal_shown(self):
        r = make_run(self.d, "paused(awaiting-merge)", {"task": "A"})
        evidence.write_json(self.d / "reconcile" / "A.json",
                            {"reason": "DECLINED", "proposed": "planned→verified", "merge_commit": "abc", "item_id": "001"})
        code, msg = approvals.record(r, "reconcile")
        self.assertEqual(code, 0)
        self.assertIn("abc", msg)


class TestTake(Base):
    def test_used_once(self):
        r = make_run(self.d, "paused(awaiting-mode)")
        approvals.record(r, "mode")
        self.assertTrue(approvals.take(r, "mode"))
        self.assertFalse(approvals.take(r, "mode"))

    def test_bound_to_attempt(self):
        r = make_run(self.d, "paused(awaiting-retry)", {"task": "A", "n": 2})
        approvals.record(r, "retry")
        self.assertFalse(approvals.take(r, "retry", "A", 3))
        self.assertFalse(approvals.take(r, "retry", "A", 2))  # consumed by the mismatch

    def test_reconcile_bound_to_merge_commit(self):
        r = make_run(self.d, "paused(awaiting-merge)", {"task": "A"})
        rec = self.d / "reconcile" / "A.json"
        evidence.write_json(rec, {"reason": "DECLINED", "proposed": "planned→verified", "merge_commit": "abc"})
        approvals.record(r, "reconcile")
        evidence.write_json(rec, {"reason": "DECLINED", "proposed": "planned→verified", "merge_commit": "other"})
        self.assertFalse(approvals.take(r, "reconcile", "A"))


class TestDecide(Base):
    def test_no_tty_no_approval_is_no(self):
        r = make_run(self.d, "paused(awaiting-mode)")
        self.assertEqual(approvals.decide(r, "mode", "Q? "), (False, "none"))

    def test_no_tty_with_approval(self):
        r = make_run(self.d, "paused(awaiting-mode)")
        approvals.record(r, "mode")
        self.assertEqual(approvals.decide(r, "mode", "Q? "), (True, "harness"))

    def test_tty_asks_and_ignores_approval_file(self):
        r = make_run(self.d, "paused(awaiting-mode)")
        approvals.record(r, "mode")
        with mock.patch.object(approvals, "has_tty", return_value=True):
            self.assertEqual(approvals.decide(r, "mode", "Q? ", ask_tty=lambda q: "n"), (False, "tty"))
        self.assertTrue((self.d / "approvals" / "pending.json").exists())


class TestPipelineAndAudit(Base):
    def test_no_terminal_pauses_instead_of_failing(self):
        r = make_run(self.d, "in-progress")
        with mock.patch.object(evidence, "write_summary"):
            self.assertEqual(pipeline.mode_not_confirmed(r, "none"), pipeline.EXIT_DECLINED)
        self.assertEqual(r.state["state"], "paused(awaiting-mode)")

    def test_explicit_no_on_terminal_fails(self):
        r = make_run(self.d, "in-progress")
        with mock.patch.object(evidence, "write_summary"):
            pipeline.mode_not_confirmed(r, "tty")
        self.assertEqual(r.state["state"], "failed")

    def test_audit_reports_channels(self):
        ev = [{"event": "approval_channel", "detail": "step=mode approved=true channel=harness"},
              {"event": "approval_channel", "detail": "step=retry approved=false channel=none"},
              {"event": "approval_channel", "detail": "step=retry approved=true channel=tty"}]
        self.assertEqual(audit._channels(ev), {"mode": ["harness"], "retry": ["tty"]})


if __name__ == "__main__":
    unittest.main()
