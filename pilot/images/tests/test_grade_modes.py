"""grade.py item-run modes (014 T024, specs/014-any-project/contracts/grading.md): baseline,
test_command and the hidden-test overlay. The fixture path is covered by smoke.sh and
test_grading_reference.py."""
import importlib.util
import json
import os
import stat
import subprocess
import tempfile
import unittest
from pathlib import Path

_spec = importlib.util.spec_from_file_location("grade", Path(__file__).resolve().parents[1] / "tools/grade.py")
grade = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(grade)

GIT = ["git", "-c", "user.name=t", "-c", "user.email=t@invalid"]
TEST_CMD = "python3 -m unittest discover -s tests -q"


def sh(cwd, *args):
    subprocess.run([*GIT, *args], cwd=cwd, check=True, capture_output=True)


def out(cwd, *args):
    return subprocess.run(["git", *args], cwd=cwd, check=True, capture_output=True, text=True).stdout.strip()


def write(root, rel, body):
    p = Path(root, rel)
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(body)


class Modes(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.mkdtemp()
        # A stand-in `spec` that reports item 002's tasks at 100 % (G3), first on PATH.
        bindir = Path(self.tmp, "bin")
        bindir.mkdir()
        fake = bindir / "spec"
        fake.write_text("#!/bin/sh\necho '{\"items\": [{\"id\": \"002\", \"tasks\": {\"percent\": 100}}]}'\n")
        fake.chmod(fake.stat().st_mode | stat.S_IEXEC)
        self.path = os.environ["PATH"]
        os.environ["PATH"] = f"{bindir}:{self.path}"
        # The project at its base revision: greet() exists, shout() doesn't yet.
        self.repo = Path(self.tmp, "repo")
        self.repo.mkdir()
        sh(self.repo, "init", "-q", "-b", "main")
        write(self.repo, "app.py", "def greet(n):\n    return 'hi ' + n\n")
        write(self.repo, "tests/__init__.py", "")
        write(self.repo, "tests/test_visible.py",
              "import unittest, app\nclass T(unittest.TestCase):\n    def test_greet(self):\n        self.assertEqual(app.greet('a'), 'hi a')\n")
        write(self.repo, "specs/002-shout/tasks.md", "- [X] T001 add shout\n")
        sh(self.repo, "add", "-A")
        sh(self.repo, "commit", "-qm", "base")
        self.base = out(self.repo, "rev-parse", "HEAD")
        self.inp = Path(self.tmp, "in")
        self.inp.mkdir()
        sh(self.repo, "bundle", "create", "-q", str(self.inp / "base.bundle"), "main")
        # Hidden tests mirror the project's layout.
        self.grading = Path(self.tmp, "grading")
        write(self.grading, "overlay/tests/test_hidden.py",
              "import unittest, app\nclass H(unittest.TestCase):\n    def test_shout(self):\n        self.assertEqual(app.shout('a'), 'A!')\n")

    def tearDown(self):
        os.environ["PATH"] = self.path

    def meta(self, **kw):
        m = {"run_id": "r", "task": "002", "attempt": 1, "item_id": "002", "base_revision": self.base,
             "allowed_paths": ["app.py", "specs/002-shout/tasks.md"], "test_command": TEST_CMD,
             "hidden_mode": "overlay", "command_timeout_s": 60}
        m.update(kw)
        return m

    def change(self, body):
        write(self.repo, "app.py", body)
        sh(self.repo, "commit", "-qam", "pilot: change")
        sh(self.repo, "bundle", "create", "-q", str(self.inp / "change.bundle"), f"{self.base}..HEAD")

    def run_grade(self, meta):
        return grade.grade(meta, inp=str(self.inp), grading=str(self.grading), work=tempfile.mkdtemp(dir=self.tmp))

    def results(self, report):
        return {c["id"]: c["result"] for c in report["checks"]}

    def test_baseline_passes_when_hidden_tests_fail_on_base(self):
        r = self.run_grade(self.meta(mode="baseline"))
        self.assertEqual(r["result"], "pass", r)
        self.assertEqual(r["mode"], "baseline")
        self.assertEqual(self.results(r), {"G1_bundle_applies": "pass", "G2_allowed_paths": "skipped",
                                           "G3_tasks_checked": "skipped", "G4_visible_tests": "pass",
                                           "G5_hidden_tests": "pass"})
        self.assertIsInstance(r["test_seconds"], int)  # 033 R7: one run of the test command
        self.assertGreaterEqual(r["test_seconds"], 1)

    def test_baseline_fails_when_hidden_tests_already_pass(self):
        write(self.grading, "overlay/tests/test_hidden.py",
              "import unittest, app\nclass H(unittest.TestCase):\n    def test_greet(self):\n        self.assertEqual(app.greet('b'), 'hi b')\n")
        r = self.run_grade(self.meta(mode="baseline"))
        self.assertEqual(r["result"], "fail")
        self.assertEqual(self.results(r)["G5_hidden_tests"], "fail")
        self.assertIn("carry no signal", next(c["detail"] for c in r["checks"] if c["id"] == "G5_hidden_tests"))

    def test_baseline_fails_when_the_suite_fails_on_base(self):
        write(self.repo, "tests/test_broken.py", "import unittest\nclass B(unittest.TestCase):\n    def test_x(self):\n        self.fail('broken')\n")
        sh(self.repo, "add", "-A")
        sh(self.repo, "commit", "-qm", "broken base")
        sh(self.repo, "bundle", "create", "-q", str(self.inp / "base.bundle"), "main")
        r = self.run_grade(self.meta(mode="baseline", base_revision=out(self.repo, "rev-parse", "HEAD")))
        self.assertEqual(r["result"], "fail")
        self.assertEqual(self.results(r)["G4_visible_tests"], "fail")
        self.assertEqual(self.results(r)["G5_hidden_tests"], "skipped")

    def test_attempt_passes_with_overlay(self):
        self.change("def greet(n):\n    return 'hi ' + n\n\ndef shout(n):\n    return n.upper() + '!'\n")
        r = self.run_grade(self.meta())
        self.assertEqual(r["result"], "pass", r)
        self.assertNotIn("mode", r)

    def test_attempt_fails_hidden(self):
        self.change("def greet(n):\n    return 'hi ' + n\n\ndef shout(n):\n    return n\n")
        r = self.run_grade(self.meta())
        self.assertEqual(r["result"], "fail")
        self.assertEqual(self.results(r)["G4_visible_tests"], "pass")
        self.assertEqual(self.results(r)["G5_hidden_tests"], "fail")

    def test_overlay_wins_over_the_workers_copy(self):
        # The worker writes its own tests/test_hidden.py that always passes; the overlay replaces it.
        write(self.repo, "tests/test_hidden.py", "import unittest\nclass H(unittest.TestCase):\n    def test_ok(self):\n        pass\n")
        sh(self.repo, "add", "-A")
        self.change("def greet(n):\n    return 'hi ' + n\n")
        r = self.run_grade(self.meta(allowed_paths=["app.py", "tests/*", "specs/002-shout/tasks.md"]))
        self.assertEqual(self.results(r)["G5_hidden_tests"], "fail", r)

    def test_missing_overlay_fails_closed(self):
        import shutil
        shutil.rmtree(self.grading / "overlay")
        self.change("def greet(n):\n    return 'hi ' + n\n\ndef shout(n):\n    return n.upper() + '!'\n")
        r = self.run_grade(self.meta())
        self.assertEqual(self.results(r)["G5_hidden_tests"], "fail")

    def test_report_is_json(self):
        json.dumps(self.run_grade(self.meta(mode="baseline")))


if __name__ == "__main__":
    unittest.main()


SHOUT = "def greet(n):\n    return 'hi ' + n\n\ndef shout(n):\n    return n.upper() + '!'\n"
SHOUT_TEST = ("import unittest, app\nclass S(unittest.TestCase):\n    def test_shout(self):\n"
              "        self.assertEqual(app.shout('b'), 'B!')\n")


class TestRules(Modes):
    """G6 and G7 (PILOT_TEST_PATHS): the change must add tests and keep the existing ones."""

    def change_files(self, files, delete=()):
        for rel, body in files.items():
            write(self.repo, rel, body)
        for rel in delete:
            Path(self.repo, rel).unlink()
        sh(self.repo, "add", "-A")
        sh(self.repo, "commit", "-qm", "pilot: change")
        sh(self.repo, "bundle", "create", "-q", str(self.inp / "change.bundle"), f"{self.base}..HEAD")

    def tmeta(self, **kw):
        return self.meta(allowed_paths=["app.py", "tests/*", "specs/002-shout/tasks.md"], test_paths=["tests/*"], **kw)

    def test_fixture_and_items_without_test_paths_keep_five_checks(self):
        self.change(SHOUT)
        r = self.run_grade(self.meta())
        self.assertEqual(r["result"], "pass")
        self.assertEqual(len(r["checks"]), 5)

    def test_change_with_a_new_test_passes_all_seven(self):
        self.change_files({"app.py": SHOUT, "tests/test_shout.py": SHOUT_TEST})
        r = self.run_grade(self.tmeta())
        self.assertEqual(r["result"], "pass", r["checks"])
        self.assertEqual(len(r["checks"]), 7)
        self.assertEqual(self.results(r)["G7_existing_tests_kept"], "pass")

    def test_change_without_tests_fails_g6(self):
        self.change(SHOUT)
        r = self.run_grade(self.tmeta())
        self.assertEqual(r["result"], "fail")
        res = self.results(r)
        self.assertEqual(res["G6_tests_changed"], "fail")
        self.assertEqual(res["G4_visible_tests"], "skipped")

    def test_weakened_existing_test_fails_g7(self):
        # The change breaks greet() and edits the existing test to match.
        broken = "def greet(n):\n    return 'yo ' + n\n\ndef shout(n):\n    return n.upper() + '!'\n"
        weak = "import unittest, app\nclass T(unittest.TestCase):\n    def test_greet(self):\n        self.assertEqual(app.greet('a'), 'yo a')\n"
        self.change_files({"app.py": broken, "tests/test_visible.py": weak, "tests/test_shout.py": SHOUT_TEST})
        r = self.run_grade(self.tmeta())
        res = self.results(r)
        self.assertEqual(res["G4_visible_tests"], "pass")
        self.assertEqual(res["G7_existing_tests_kept"], "fail")
        self.assertEqual(res["G5_hidden_tests"], "skipped")
        self.assertEqual(r["result"], "fail")

    def test_deleted_existing_test_is_restored_and_checked(self):
        broken = "def greet(n):\n    return 'yo ' + n\n\ndef shout(n):\n    return n.upper() + '!'\n"
        self.change_files({"app.py": broken, "tests/test_shout.py": SHOUT_TEST}, delete=["tests/test_visible.py"])
        r = self.run_grade(self.tmeta())
        self.assertEqual(self.results(r)["G7_existing_tests_kept"], "fail")

    def test_spec_may_allow_an_existing_test_to_change(self):
        changed = "def greet(n):\n    return 'yo ' + n\n\ndef shout(n):\n    return n.upper() + '!'\n"
        new = "import unittest, app\nclass T(unittest.TestCase):\n    def test_greet(self):\n        self.assertEqual(app.greet('a'), 'yo a')\n"
        self.change_files({"app.py": changed, "tests/test_visible.py": new})
        r = self.run_grade(self.tmeta(tests_may_change=["tests/test_visible.py"]))
        self.assertEqual(r["result"], "pass", r["checks"])
        g7 = next(c["detail"] for c in r["checks"] if c["id"] == "G7_existing_tests_kept")
        self.assertIn("which the spec says may change", g7)
        self.assertIn("tests/test_visible.py", g7)

    def test_extending_an_existing_test_file_passes(self):
        extended = (Path(self.repo, "tests/test_visible.py").read_text() +
                    "    def test_shout(self):\n        self.assertEqual(app.shout('c'), 'C!')\n")
        self.change_files({"app.py": SHOUT, "tests/test_visible.py": extended})
        r = self.run_grade(self.tmeta())
        self.assertEqual(r["result"], "pass", r["checks"])



class HiddenOutputWithheld(Modes):
    """C-14: an item run's G5 detail never carries the hidden tests' output (it can show their
    source), in attempts and in the baseline."""

    MARK = "SECRET_EXPECTATION_42"

    def setUp(self):
        super().setUp()
        write(self.grading, "overlay/tests/test_hidden.py",
              "import unittest, app\nclass H(unittest.TestCase):\n    def test_shout(self):\n"
              f"        self.assertEqual(app.shout('a'), 'A!', '{self.MARK}')\n")

    def detail(self, r):
        return next(c["detail"] for c in r["checks"] if c["id"] == "G5_hidden_tests")

    def test_attempt_detail_has_no_output(self):
        self.change("def greet(n):\n    return 'hi ' + n\n\ndef shout(n):\n    return n\n")
        r = self.run_grade(self.meta())
        self.assertEqual(self.results(r)["G5_hidden_tests"], "fail")
        self.assertNotIn(self.MARK, json.dumps(r))
        self.assertIn("output withheld", self.detail(r))

    def test_baseline_detail_has_no_output(self):
        r = self.run_grade(self.meta(mode="baseline"))
        self.assertEqual(r["result"], "pass")
        self.assertNotIn(self.MARK, json.dumps(r))
        self.assertIn("fail on the base revision (expected)", self.detail(r))


class AuthorCheck(HiddenOutputWithheld):
    """018: with author_check, the baseline keeps the hidden output for the author only."""

    def test_output_kept_for_the_author_not_in_the_report(self):
        grade.AUTHOR_OUTPUT.clear()
        r = self.run_grade(self.meta(mode="baseline", author_check=True))
        self.assertEqual(r["result"], "pass")
        self.assertNotIn(self.MARK, json.dumps(r))
        self.assertTrue(grade.AUTHOR_OUTPUT and self.MARK in grade.AUTHOR_OUTPUT[-1])

    def test_no_output_kept_without_the_flag(self):
        grade.AUTHOR_OUTPUT.clear()
        self.run_grade(self.meta(mode="baseline"))
        self.assertEqual(grade.AUTHOR_OUTPUT, [])


class AuthorOutputLength(Modes):
    """018: the author's output isn't cut to the last 3000 characters."""

    def test_long_output_kept_whole(self):
        long_test = ("import unittest\nclass H(unittest.TestCase):\n" +
                     "".join(f"    def test_{i}(self):\n        self.fail('FIRST_MARK' if {i} == 0 else 'x' * 200)\n" for i in range(40)))
        write(self.grading, "overlay/tests/test_hidden.py", long_test)
        grade.AUTHOR_OUTPUT.clear()
        r = self.run_grade(self.meta(mode="baseline", author_check=True))
        self.assertEqual(r["result"], "pass")
        out = grade.AUTHOR_OUTPUT[-1]
        self.assertGreater(len(out), 3000)
        self.assertIn("FIRST_MARK", out)


class ParsedOutputNotCut(Modes):
    """A real project's spec status JSON is far over 3000 characters (cli-schema-mcp: ~10 KB);
    the grader must parse all of it, and see every changed file."""

    def test_large_spec_status_output(self):
        filler = ",".join('{"id": "9%02d", "tasks": {"percent": 0}}' % i for i in range(90))
        fake = Path(self.tmp, "bin", "spec")
        fake.write_text("#!/bin/sh\necho '{\"items\": [" + filler + ", {\"id\": \"002\", \"tasks\": {\"percent\": 100}}]}'\n")
        self.assertGreater(len(filler), 3000)
        self.change(SHOUT)
        r = self.run_grade(self.meta())
        self.assertEqual(self.results(r)["G3_tasks_checked"], "pass", r["checks"])

    def test_every_changed_file_seen(self):
        files = {f"src/pkg/module_with_a_long_name_{i:03d}.py": "x = 1\n" for i in range(120)}
        files["app.py"] = SHOUT
        for rel, body in files.items():
            write(self.repo, rel, body)
        write(self.repo, "aaa_not_allowed.txt", "sneaky\n")
        sh(self.repo, "add", "-A")
        sh(self.repo, "commit", "-qm", "pilot: change")
        sh(self.repo, "bundle", "create", "-q", str(self.inp / "change.bundle"), f"{self.base}..HEAD")
        r = self.run_grade(self.meta(allowed_paths=["app.py", "src/*", "specs/002-shout/tasks.md"]))
        detail = next(c["detail"] for c in r["checks"] if c["id"] == "G2_allowed_paths")
        self.assertEqual(self.results(r)["G2_allowed_paths"], "fail")
        self.assertIn("aaa_not_allowed.txt", detail)
