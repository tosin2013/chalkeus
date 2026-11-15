"""004 US2: GitHub approvals on the run's tracking issue (research R3, SC-004).

Every rejection rule has a test; only a valid `/approve <step>` from a listed approver with write
access, after the decision post, unedited and not withdrawn, is recorded (channel github).
"""
import json
import tempfile
import unittest
from pathlib import Path
from unittest import mock

from specfoundry_pilot import approvals, evidence, ghapprove
from tests.ghfake import FakeGitHub


class Cfg:
    repo = "o/x-pilot-fixture"
    branch = "main"
    credential = "token"
    approvers = ["alice"]
    github_approvals = True
    github_issues = True


def make_run(d: Path, state="paused(awaiting-retry)", pending=None) -> evidence.Run:
    r = evidence.Run(d)
    r.state = {"run_id": d.name, "state": state, "pending": pending if pending is not None else {"task": "A", "n": 2},
               "tasks": {}}
    r.save()
    return r


class Base(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.gh = FakeGitHub(publisher="pub-bot", permissions={"alice": "write", "bob": "write", "carol": "read"}).patch(self)
        self.run = make_run(Path(self.tmp.name) / "20261116T101500Z")
        self.p = approvals.pending_step(self.run)
        self.d = ghapprove.post_decision(Cfg(), self.run, self.p)
        self.issue = json.loads((self.run.dir / "github/tracking.json").read_text())["issue"]

    def say(self, body, author="alice", **kw):
        return self.gh.add_comment(self.issue, body, author, **kw)

    def find(self):
        return ghapprove.find_approval(Cfg(), self.run, self.p)

    def reasons(self):
        f = self.run.dir / "github/approvals.jsonl"
        return [json.loads(x)["reason"] for x in f.read_text().splitlines()] if f.exists() else []


class TestPosting(Base):
    def test_tracking_issue_and_post_are_idempotent(self):
        again = ghapprove.post_decision(Cfg(), self.run, self.p)
        self.assertEqual(again["comment_id"], self.d["comment_id"])
        self.assertEqual([o for o, _ in self.gh.ops].count("create-issue"), 1)
        body = self.gh.comments[self.issue][0]["body"]
        self.assertIn("<!-- specfoundry:decision run=20261116T101500Z step=retry task=A n=2 -->", body)
        self.assertIn("/approve retry", body)
        self.assertIn("@alice", body)


class TestRules(Base):
    def test_accept_valid(self):
        cid = self.say("/approve retry\nlooks fine")
        self.assertEqual(self.find(), {"approver": "alice", "comment_id": cid})
        self.assertEqual(self.reasons(), [None])

    def test_reject_wrong_step(self):
        self.say("/approve mode")
        self.assertIsNone(self.find())
        self.assertEqual(self.reasons(), ["wrong_step"])

    def test_reject_extra_words_on_first_line(self):
        self.say("/approve retry please")
        self.assertIsNone(self.find())

    def test_reject_not_listed(self):
        self.say("/approve retry", author="bob")
        self.assertIsNone(self.find())
        self.assertEqual(self.reasons(), ["not_an_approver"])

    def test_reject_no_write_access_now(self):
        self.gh.permissions["alice"] = "read"
        self.say("/approve retry")
        self.assertIsNone(self.find())
        self.assertEqual(self.reasons(), ["no_write_access"])

    def test_reject_publisher_identity(self):
        cfg = Cfg()
        cfg.approvers = ["alice", "pub-bot"]
        self.gh.permissions["pub-bot"] = "admin"
        self.say("/approve retry", author="pub-bot")
        self.assertIsNone(ghapprove.find_approval(cfg, self.run, self.p))
        self.assertEqual(self.reasons(), ["publisher_identity"])

    def test_reject_app_authored(self):
        self.say("/approve retry", app=True)
        self.assertIsNone(self.find())
        self.assertEqual(self.reasons(), ["app_authored"])

    def test_reject_edited(self):
        self.say("/approve retry", edited=True)
        self.assertIsNone(self.find())
        self.assertEqual(self.reasons(), ["edited"])

    def test_reject_before_decision_post(self):
        self.gh.comments[self.issue].insert(0, {"id": 1, "body": "/approve retry", "author": "alice",
                                                "created_at": "t", "updated_at": "t", "app": False})
        self.assertIsNone(self.find())
        self.assertEqual(self.reasons(), ["before_decision"])

    def test_reject_withdrawn(self):
        self.say("/approve retry")
        self.say("/withdraw retry")
        self.assertIsNone(self.find())
        self.assertEqual(self.reasons(), ["withdrawn"])

    def test_withdraw_by_someone_else_does_not_cancel(self):
        cid = self.say("/approve retry")
        self.say("/withdraw retry", author="bob")
        self.assertEqual(self.find()["comment_id"], cid)

    def test_duplicate_earliest_wins(self):
        first = self.say("/approve retry")
        self.say("/approve retry")
        self.assertEqual(self.find()["comment_id"], first)
        self.assertEqual(self.reasons(), [None, "duplicate"])

    def test_other_attempt_post_does_not_count(self):
        # an approval under the post for attempt 2 does not approve attempt 3
        self.say("/approve retry")
        p3 = dict(self.p, n=3)
        self.run.state["pending"] = {"task": "A", "n": 3}
        ghapprove.post_decision(Cfg(), self.run, p3)
        self.assertIsNone(ghapprove.find_approval(Cfg(), self.run, p3))

    def test_github_error_fails_closed(self):
        self.say("/approve retry")
        self.gh.fail = "read"
        self.assertIsNotNone(ghapprove.collect(Cfg(), self.run))  # message, nothing recorded
        self.assertFalse((self.run.dir / "approvals/pending.json").exists())

    def test_log_is_not_duplicated_across_resumes(self):
        self.say("/approve mode")
        self.find()
        self.find()
        self.assertEqual(self.reasons(), ["wrong_step"])


class TestRecording(Base):
    def test_collect_records_github_channel_used_once(self):
        cid = self.say("/approve retry")
        msg = ghapprove.collect(Cfg(), self.run)
        self.assertIn("@alice", msg)
        rec = json.loads((self.run.dir / "approvals/pending.json").read_text())
        self.assertEqual((rec["channel"], rec["approver"], rec["comment_id"]), ("github", "alice", cid))
        with mock.patch.object(approvals, "has_tty", return_value=False):
            ok, ch = approvals.decide(self.run, "retry", "Retry? ", task="A", n=2)
            self.assertEqual((ok, ch), (True, "github"))
            approvals.note(self.run, "retry", "A", 2, ok, ch)
            self.assertEqual(approvals.decide(self.run, "retry", "Retry? ", task="A", n=2), (False, "none"))
        ev = [json.loads(x) for x in (self.run.dir / "events.jsonl").read_text().splitlines()]
        last = [e for e in ev if e["event"] == "approval_channel"][-1]
        self.assertEqual(last["detail"], f"step=retry approved=true channel=github approver=alice comment={cid}")

    def test_terminal_overrides_github_approval(self):
        self.say("/approve retry")
        ghapprove.collect(Cfg(), self.run)
        with mock.patch.object(approvals, "has_tty", return_value=True):
            ok, ch = approvals.decide(self.run, "retry", "Retry? ", task="A", n=2, ask_tty=lambda q: "n")
        self.assertEqual((ok, ch), (False, "tty"))
        self.assertFalse((self.run.dir / "approvals/pending.json").exists())

    def test_reconcile_and_merge_not_collected(self):
        r = make_run(Path(self.tmp.name) / "20261116T111111Z", state="paused(awaiting-merge)", pending={"task": "A"})
        self.assertIsNone(ghapprove.collect(Cfg(), r))


if __name__ == "__main__":
    unittest.main()


class TestWatch(unittest.TestCase):
    def test_resumes_when_ready_and_stops_when_done(self):
        from specfoundry_pilot import pipeline
        states = iter(["paused(awaiting-retry)", "paused(awaiting-retry)", "succeeded", "succeeded"])
        runs = []

        def load(_id):
            r = mock.Mock()
            r.state = {"state": next(states), "pending": {}}
            runs.append(r)
            return r
        ready = iter([False, True])
        slept = []
        with mock.patch.object(pipeline.evidence, "load_run", side_effect=load), \
             mock.patch.object(pipeline, "ready_to_resume", side_effect=lambda c, r: next(ready)), \
             mock.patch.object(pipeline, "resume", return_value=0) as res:
            code = pipeline.watch(Cfg(), "X", interval=1, sleep=slept.append)
        self.assertEqual(code, 0)
        res.assert_called_once()
        self.assertEqual(slept, [1])
