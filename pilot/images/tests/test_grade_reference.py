"""grade.py reference mode (041 T008, specs/041-proven-hidden-tests/contracts/commands.md): the
reference patch applies, the project's tests pass with it, and each hidden test file fails on the
base and passes on the reference in every run."""
import subprocess
import unittest
from pathlib import Path

from test_grade_modes import Modes, grade, sh, write

SHOUT = "def greet(n):\n    return 'hi ' + n\n\n\ndef shout(n):\n    return n.upper() + '!'\n"


class Reference(Modes):
    def setUp(self):
        super().setUp()
        grade.AUTHOR_OUTPUT.clear()
        self.reference(SHOUT)

    def reference(self, body, path="app.py"):
        """Write the reference patch: body for path, as a diff against the base."""
        write(self.repo, path, body)
        diff = subprocess.run(["git", "diff", "--", path], cwd=self.repo, capture_output=True, text=True, check=True).stdout
        sh(self.repo, "checkout", "--", path) if path == "app.py" else Path(self.repo, path).unlink()
        write(self.grading, "reference.patch", diff)

    def ref(self, **kw):
        return self.run_grade(self.meta(mode="reference", attempt=0, **kw))

    def check(self, r, cid):
        return next(c for c in r["checks"] if c["id"] == cid)

    def test_a_good_test_is_proven(self):
        r = self.ref()
        self.assertEqual(r["result"], "pass", r)
        self.assertEqual(r["mode"], "reference")
        self.assertIsNone(r["failure"])
        self.assertEqual(r["repeats"], 3)
        self.assertEqual(r["files"], [{"path": "tests/test_hidden.py", "base": "fail",
                                       "reference": ["pass", "pass", "pass"], "verdict": "proven"}])

    def test_a_test_that_does_not_load_fails_on_the_reference(self):
        # documcp 004: a test that can't compile "fails on the base" too; the reference catches it.
        write(self.grading, "overlay/tests/test_hidden.py", "import unittest, app\nthis is not python\n")
        r = self.ref()
        self.assertEqual(r["result"], "fail")
        self.assertEqual(r["failure"], "DRAFT_FAILS_ON_REFERENCE")
        self.assertEqual(r["files"][0]["verdict"], "fails_on_reference")
        self.assertIn("tests/test_hidden.py: fails_on_reference", self.check(r, "R3_hidden_files")["detail"])

    def test_a_test_that_passes_on_the_base_is_named(self):
        write(self.grading, "overlay/tests/test_hidden.py",
              "import unittest, app\nclass H(unittest.TestCase):\n    def test_greet(self):\n        self.assertTrue(app.greet)\n")
        r = self.ref()
        self.assertEqual(r["failure"], "DRAFT_PASSES_ON_BASE")
        self.assertEqual(r["files"][0]["verdict"], "passes_on_base")

    def test_each_file_is_judged_on_its_own(self):
        write(self.grading, "overlay/tests/test_hidden2.py", "import unittest\nbroken(\n")
        r = self.ref()
        self.assertEqual(r["failure"], "DRAFT_FAILS_ON_REFERENCE")
        self.assertEqual({f["path"]: f["verdict"] for f in r["files"]},
                         {"tests/test_hidden.py": "proven", "tests/test_hidden2.py": "fails_on_reference"})

    def test_a_flaky_test(self):
        # Passes on the reference only every other run (a counter file outside the tree).
        counter = Path(self.tmp, "count")
        counter.write_text("0")
        write(self.grading, "overlay/tests/test_hidden.py",
              "import unittest, app, pathlib\nc = pathlib.Path(%r)\nn = int(c.read_text()) + 1\nc.write_text(str(n))\n"
              "class H(unittest.TestCase):\n    def test_shout(self):\n        self.assertEqual(app.shout('a'), 'A!')\n"
              "        self.assertEqual(n %% 2, 0)\n" % str(counter))
        r = self.ref()
        self.assertEqual(r["failure"], "DRAFT_FLAKY", r["files"])
        self.assertEqual(r["files"][0]["verdict"], "flaky")

    def test_a_broken_reference(self):
        self.reference("def greet(n):\n    return 'HELLO'\n\n\ndef shout(n):\n    return n.upper() + '!'\n")
        r = self.ref()
        self.assertEqual(r["failure"], "REFERENCE_BROKEN")
        self.assertEqual(self.check(r, "R3_hidden_files")["result"], "skipped")
        self.assertEqual(r["files"], [])

    def test_a_reference_that_does_not_apply(self):
        write(self.grading, "reference.patch", "--- a/nope.py\n+++ b/nope.py\n@@ -1 +1 @@\n-x\n+y\n")
        r = self.ref()
        self.assertEqual(r["failure"], "REFERENCE_DOESNT_APPLY")
        self.assertEqual(self.check(r, "R2_reference_suite_passes")["result"], "skipped")

    def test_no_reference_staged(self):
        Path(self.grading, "reference.patch").unlink()
        r = self.ref()
        self.assertEqual(r["failure"], "REFERENCE_DOESNT_APPLY")

    def test_repeats_never_below_two(self):
        self.assertEqual(self.ref(repeats=1)["repeats"], 2)
        self.assertEqual(len(self.ref(repeats=5)["files"][0]["reference"]), 5)

    def test_focused_command(self):
        r = self.ref(hidden_test_command="python3 -m unittest -q {files}")
        self.assertEqual(r["result"], "pass", r)

    def test_helpers_and_the_map_are_not_tests(self):
        write(self.grading, "overlay/tests/helper_data.py", "X = 1\n")
        write(self.grading, "overlay/.acceptance-map.json", "{}")
        r = self.ref(test_paths=["tests/test_*.py"])
        self.assertEqual([f["path"] for f in r["files"]], ["tests/test_hidden.py"])
        self.assertEqual(r["result"], "pass", r)

    def test_hidden_output_only_for_the_author(self):
        r = self.ref()
        self.assertEqual(grade.AUTHOR_OUTPUT, [])
        self.assertNotIn("shout", str(r["checks"]))
        self.ref(author_check=True)
        self.assertEqual(len(grade.AUTHOR_OUTPUT), 4)  # 1 base run + 3 reference runs
        self.assertIn("on the base (must fail)", grade.AUTHOR_OUTPUT[0])


if __name__ == "__main__":
    unittest.main()
