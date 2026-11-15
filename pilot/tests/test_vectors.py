"""016 FR-011: the reference passes every shared vector (specs/004-github-loop/contracts/vectors/)."""
import json
import unittest
from pathlib import Path

from specfoundry_pilot import ghapprove, reconcile

V = Path(__file__).resolve().parents[2] / "specs/004-github-loop/contracts/vectors"


class TestApprovalVectors(unittest.TestCase):
    def test_cases(self):
        doc = json.loads((V / "approvals.json").read_text())
        self.assertGreaterEqual(len(doc["cases"]), 12)
        for case in doc["cases"]:
            with self.subTest(case["name"]):
                i = case["input"]
                perms = i["permissions"]
                acc, log = ghapprove.evaluate(i["comments"], i["decision"], i["step"], i["approvers"],
                                              i["publisher"], lambda u: perms.get(u, "none"))
                self.assertEqual(acc["id"] if acc else None, case["expect"]["accepted"])
                self.assertEqual([{"comment_id": e["comment_id"], "reason": e["reason"]} for e in log],
                                 case["expect"]["log"])


class TestStatusPRVectors(unittest.TestCase):
    def test_cases(self):
        doc = json.loads((V / "status_pr.json").read_text())
        for case in doc["cases"]:
            with self.subTest(case["name"]):
                i = case["input"]
                view = {"state": i["state"], "mergedBy": {"login": i["merged_by"]} if i["merged_by"] else None}
                ok, reason = reconcile.check_status_pr(view, i["base_text"], i["head_text"], i["changed"], i["item"],
                                                       i["approvers"], i["publisher"])
                self.assertEqual({"ok": ok, "reason": reason}, case["expect"])


if __name__ == "__main__":
    unittest.main()
