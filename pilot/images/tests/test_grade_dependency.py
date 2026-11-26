"""grade.py for dependency items (029 T011, specs/029-brownfield-intake/contracts/grading.md):
`fixed_paths` in G2, and the `dependency` baseline rule."""
import unittest
from pathlib import Path

from test_grade_modes import Modes, SHOUT, sh, write


class FixedPaths(Modes):
    def setUp(self):
        super().setUp()
        # The approved manifest change is part of the base the attempt starts from.
        write(self.repo, "requirements.txt", "requests==2.32.3\n")
        sh(self.repo, "add", "-A")
        sh(self.repo, "commit", "-qm", "chore(deps): bump requests")
        self.base = sh_out(self.repo)
        sh(self.repo, "bundle", "create", "-q", str(self.inp / "base.bundle"), "main")

    def change_files(self, files, delete=(), rename=None):
        for rel, body in files.items():
            write(self.repo, rel, body)
        for rel in delete:
            Path(self.repo, rel).unlink()
        if rename:
            sh(self.repo, "mv", *rename)
        sh(self.repo, "add", "-A")
        sh(self.repo, "commit", "-qm", "pilot: change")
        sh(self.repo, "bundle", "create", "-q", str(self.inp / "change.bundle"), f"{self.base}..HEAD")

    def g2(self, report):
        return next(c for c in report["checks"] if c["id"] == "G2_allowed_paths")

    def test_editing_a_fixed_path_fails_g2_even_under_allow_all(self):
        self.change_files({"app.py": SHOUT, "requirements.txt": "requests==2.31.0\n"})
        r = self.run_grade(self.meta(allowed_paths=["**", "*"], fixed_paths=["requirements.txt"]))
        self.assertEqual(r["result"], "fail")
        self.assertEqual(self.g2(r)["result"], "fail")
        self.assertTrue(self.g2(r)["detail"].startswith("MANIFEST_CHANGED: requirements.txt "), self.g2(r)["detail"])
        self.assertIn("changed=", self.g2(r)["detail"])

    def test_deleting_or_renaming_a_fixed_path_fails_g2(self):
        self.change_files({"app.py": SHOUT}, delete=["requirements.txt"])
        r = self.run_grade(self.meta(allowed_paths=["*"], fixed_paths=["requirements.txt"]))
        self.assertIn("MANIFEST_CHANGED: requirements.txt", self.g2(r)["detail"])

    def test_renaming_a_fixed_path_fails_g2(self):
        self.change_files({"app.py": SHOUT}, rename=("requirements.txt", "reqs.txt"))
        r = self.run_grade(self.meta(allowed_paths=["*"], fixed_paths=["requirements.txt"]))
        self.assertEqual(self.g2(r)["result"], "fail")
        self.assertIn("MANIFEST_CHANGED: requirements.txt", self.g2(r)["detail"])

    def test_a_change_elsewhere_passes_as_before(self):
        self.change_files({"app.py": SHOUT})
        r = self.run_grade(self.meta(fixed_paths=["requirements.txt"]))
        self.assertEqual(r["result"], "pass", r["checks"])
        self.assertNotIn("MANIFEST_CHANGED", self.g2(r)["detail"])

    def test_without_fixed_paths_grades_as_before(self):
        self.change_files({"app.py": SHOUT, "requirements.txt": "requests==2.31.0\n"})
        r = self.run_grade(self.meta(allowed_paths=["*"]))
        self.assertEqual(r["result"], "pass", r["checks"])


class DependencyBaseline(Modes):
    def break_visible(self):
        # The approved bump broke the code: the visible test fails on the base.
        write(self.repo, "app.py", "def greet(n):\n    return 'hello ' + n\n")
        sh(self.repo, "commit", "-qam", "chore(deps): bump greeter")
        self.base = sh_out(self.repo)
        sh(self.repo, "bundle", "create", "-q", str(self.inp / "base.bundle"), "main")

    def check(self, report, cid):
        return next(c for c in report["checks"] if c["id"] == cid)

    def passing_hidden(self):
        write(self.grading, "overlay/tests/test_hidden.py",
              "import unittest, app\nclass H(unittest.TestCase):\n    def test_ok(self):\n        self.assertTrue(callable(app.greet))\n")

    def test_failing_visible_tests_are_the_signal(self):
        # The hidden tests may well pass here (they guard what must not change): not measured.
        self.break_visible()
        self.passing_hidden()
        r = self.run_grade(self.meta(mode="baseline", dependency=True))
        self.assertEqual(r["result"], "pass", r["checks"])
        g4 = self.check(r, "G4_visible_tests")
        self.assertEqual(g4["result"], "fail")
        self.assertTrue(g4["detail"].startswith("with the approved manifest change: "))
        g5 = self.check(r, "G5_hidden_tests")
        self.assertEqual(g5["result"], "skipped")
        self.assertIn("not measured", g5["detail"])

    def test_passing_visible_tests_need_failing_hidden_tests(self):
        r = self.run_grade(self.meta(mode="baseline", dependency=True))
        self.assertEqual(r["result"], "pass", r["checks"])
        self.assertEqual(self.check(r, "G4_visible_tests")["result"], "pass")
        self.assertEqual(self.check(r, "G5_hidden_tests")["result"], "pass")  # hidden tests fail on the base

    def test_nothing_failing_means_no_work_and_fails(self):
        self.passing_hidden()
        r = self.run_grade(self.meta(mode="baseline", dependency=True))
        self.assertEqual(r["result"], "fail")
        self.assertEqual(self.check(r, "G5_hidden_tests")["result"], "fail")
        self.assertIn("carry no signal", self.check(r, "G5_hidden_tests")["detail"])

    def test_without_dependency_failing_visible_tests_still_fail(self):
        self.break_visible()
        r = self.run_grade(self.meta(mode="baseline"))
        self.assertEqual(r["result"], "fail")
        self.assertEqual(self.check(r, "G5_hidden_tests")["result"], "skipped")
        self.assertFalse(self.check(r, "G4_visible_tests")["detail"].startswith("with the approved"))


def sh_out(repo):
    import subprocess
    return subprocess.run(["git", "rev-parse", "HEAD"], cwd=repo, check=True, capture_output=True, text=True).stdout.strip()


del Modes  # imported as a base class; its own tests run in test_grade_modes


if __name__ == "__main__":
    unittest.main()
