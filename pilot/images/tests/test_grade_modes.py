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
