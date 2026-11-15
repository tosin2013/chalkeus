"""Every committed successful run record still passes the audit (012 SC-004, research R6).

Records made before 012 have no approval-channel events: they must pass unchanged and report
no channel rather than a guessed one. A record copied before `pilot down` ran for it (012's early
run 20261114T213046Z) has no teardown.jsonl; it is audited as non-final, so AU07 is not required.
"""
import json
import unittest
from pathlib import Path

from specfoundry_pilot import audit

ROOT = Path(__file__).resolve().parents[2]


def committed_successful_runs() -> list[Path]:
    out = []
    for rj in sorted(ROOT.glob("specs/*/evidence/*/run.json")):
        if json.loads(rj.read_text()).get("state") == "succeeded":
            out.append(rj.parent)
    return out


def has_channels(d: Path) -> bool:
    ev = d / "events.jsonl"
    return ev.exists() and '"approval_channel"' in ev.read_text()


class TestCommittedEvidence(unittest.TestCase):
    def test_at_least_three_records(self):
        self.assertGreaterEqual(len(committed_successful_runs()), 3)

    def test_each_record_passes(self):
        for d in committed_successful_runs():
            with self.subTest(run=str(d.relative_to(ROOT))):
                rows = audit.audit_dir(d, final=(d / "teardown.jsonl").exists())
                failed = [f"{r['id']}: {r['detail']}" for r in rows if r["result"] != "pass"]
                self.assertEqual(failed, [])
                if not has_channels(d):
                    for r in rows:
                        if r["id"] in ("AU09", "AU10"):
                            self.assertIn("channels=[]", r["detail"])


if __name__ == "__main__":
    unittest.main()
