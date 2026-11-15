"""004 audit changes: github channel approvers in AU09/AU10, AU03 with status PRs, key redaction."""
import json
import shutil
import tempfile
import unittest
from pathlib import Path

from specfoundry_pilot import audit
from tests import evidence_builder


def add_events(d: Path, details):
    f = d / "events.jsonl"
    rows = [json.loads(x) for x in f.read_text().splitlines()]
    seq = rows[-1]["seq"] if rows and "seq" in rows[-1] else len(rows)
    for det in details:
        seq += 1
        rows.append({"seq": seq, "at": "t", "stage": "approval", "task": None, "attempt": None,
                     "event": "approval_channel", "detail": det})
    f.write_text("".join(json.dumps(r) + "\n" for r in rows))


class TestAuditGitHub(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.d = evidence_builder.build(Path(self.tmp.name) / "20261116T000000Z")

    def rows(self, d=None):
        return {r["id"]: r for r in audit.audit_dir(d or self.d, secret_files=[], final=True)}

    def copy(self, name):
        d = Path(self.tmp.name) / name
        shutil.copytree(self.d, d)
        return d

    def test_github_approvers_in_detail(self):
        add_events(self.d, ["step=mode approved=true channel=github approver=alice comment=11",
                            "step=retry approved=true channel=github approver=alice comment=12",
                            "step=retry approved=false channel=none"])
        r = self.rows()
        self.assertEqual(r["AU10"]["result"], "pass")
        self.assertIn("channels=['github'] approvers=['alice']", r["AU10"]["detail"])
        self.assertIn("channels=['github'] approvers=['alice']", r["AU09"]["detail"])

    def test_no_approvers_suffix_without_github(self):
        self.assertNotIn("approvers=", self.rows()["AU09"]["detail"])

    def edit_prs(self, d, fn):
        f = d / "prs.json"
        f.write_text(json.dumps(fn(json.loads(f.read_text()))))

    def test_au03_accepts_one_status_pr_per_item(self):
        d = self.copy("status-ok")
        self.edit_prs(d, lambda p: p + [dict(p[0], kind="status", number=99)])
        self.assertEqual(self.rows(d)["AU03"]["result"], "pass")

    def test_au03_rejects_two_status_prs_for_one_item(self):
        d = self.copy("status-two")
        self.edit_prs(d, lambda p: p + [dict(p[0], kind="status", number=98), dict(p[0], kind="status", number=99)])
        self.assertEqual(self.rows(d)["AU03"]["result"], "fail")

    def test_au03_rejects_non_draft_status_pr_or_unknown_kind(self):
        for name, fn in (("nd", lambda p: p + [dict(p[0], kind="status", draft=False)]),
                         ("uk", lambda p: p + [dict(p[0], kind="other")])):
            d = self.copy(name)
            self.edit_prs(d, fn)
            self.assertEqual(self.rows(d)["AU03"]["result"], "fail", name)

    def test_private_key_is_a_redaction_hit(self):
        d = self.copy("key")
        (d / "github").mkdir(exist_ok=True)
        (d / "github" / "leak.txt").write_text("-----BEGIN RSA PRIVATE KEY-----\nabc\n")
        self.assertEqual(self.rows(d)["AU08"]["result"], "fail")


if __name__ == "__main__":
    unittest.main()
