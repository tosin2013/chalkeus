"""grade.py slice runs (026, specs/026-slice-runs/contracts/grading.md): G3 per slice, G8 earlier
slices kept, the slice baseline, and a whole-item run staying exactly as before (US1)."""
import importlib.util
import os
import shutil
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
TASKS = "specs/002-shout/tasks.md"

# A stand-in for `spec status --json` 0.4.0: it reads the item's tasks.md under --root and reports
# per-phase counts the way spec-status does (a phase is a `## Phase` section; `story` from
# "User Story N"). FAKE_SPEC_NO_PHASES=1 drops the field, like spec-status 0.3.0.
FAKE_SPEC = r'''#!/usr/bin/env python3
import json, os, re, sys
root = sys.argv[sys.argv.index("--root") + 1]
phases, checked, total, cur = [], 0, 0, None
for n, line in enumerate(open(os.path.join(root, "specs/002-shout/tasks.md")).read().splitlines(), 1):
    if line.startswith("#"):
        cur = None
        if line.startswith("## Phase"):
            m = re.search(r"User Story (\d+)", line)
            cur = {"heading": line[3:], "line": n, "story": int(m.group(1)) if m else None, "checked": 0, "total": 0}
            phases.append(cur)
        continue
    m = re.match(r"- \[( |x|X)\] ", line)
    if m:
        total += 1
        done = m.group(1) != " "
        checked += done
        if cur is not None:
            cur["total"] += 1
            cur["checked"] += done
tasks = {"checked": checked, "total": total, "percent": (100 * checked // total) if total else None}
if not os.environ.get("FAKE_SPEC_NO_PHASES"):
    tasks["phases"] = phases
print(json.dumps({"items": [{"id": "002", "tasks": tasks}]}))
'''

TASKS_BASE = """# Tasks

## Phase 1: Setup

- [ ] T001 setup

## Phase 2: User Story 1 - greet loudly

- [ ] T002 [US1] add shout

## Phase 3: User Story 2 - whisper

- [ ] T003 [US2] add whisper

## Phase 4: Polish

- [ ] T004 docs

## Notes

- [ ] T005 outside any phase
"""

US1 = ["Phase 1: Setup", "Phase 2: User Story 1 - greet loudly"]
US2 = ["Phase 3: User Story 2 - whisper", "Phase 4: Polish"]

HIDDEN_US1 = "import unittest, app\nclass H1(unittest.TestCase):\n    def test_shout(self):\n        self.assertEqual(app.shout('a'), 'A!')\n"
HIDDEN_US2 = "import unittest, app\nclass H2(unittest.TestCase):\n    def test_whisper(self):\n        self.assertEqual(app.whisper('A'), 'a...')\n"
APP_BASE = "def greet(n):\n    return 'hi ' + n\n"
APP_US1 = APP_BASE + "\ndef shout(n):\n    return n.upper() + '!'\n"
APP_US2 = APP_US1 + "\ndef whisper(n):\n    return n.lower() + '...'\n"


def sh(cwd, *args):
    subprocess.run([*GIT, *args], cwd=cwd, check=True, capture_output=True)


def out(cwd, *args):
    return subprocess.run(["git", *args], cwd=cwd, check=True, capture_output=True, text=True).stdout.strip()


def write(root, rel, body):
    p = Path(root, rel)
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(body)


def tick(text, *ids):
    for t in ids:
        text = text.replace(f"- [ ] {t} ", f"- [X] {t} ")
    return text


class SliceGrading(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.mkdtemp()
        bindir = Path(self.tmp, "bin")
        bindir.mkdir()
        fake = bindir / "spec"
        fake.write_text(FAKE_SPEC)
        fake.chmod(fake.stat().st_mode | stat.S_IEXEC)
        self.path = os.environ["PATH"]
        os.environ["PATH"] = f"{bindir}:{self.path}"
        os.environ.pop("FAKE_SPEC_NO_PHASES", None)
        self.repo = Path(self.tmp, "repo")
        self.repo.mkdir()
        sh(self.repo, "init", "-q", "-b", "main")
        write(self.repo, "app.py", APP_BASE)
        write(self.repo, "tests/__init__.py", "")
        write(self.repo, "tests/test_visible.py",
              "import unittest, app\nclass T(unittest.TestCase):\n    def test_greet(self):\n        self.assertEqual(app.greet('a'), 'hi a')\n")
        write(self.repo, TASKS, TASKS_BASE)
        sh(self.repo, "add", "-A")
        sh(self.repo, "commit", "-qm", "base")
        self.inp = Path(self.tmp, "in")
        self.inp.mkdir()
        self.grading = Path(self.tmp, "grading")
        self.rebase()

    def tearDown(self):
        os.environ["PATH"] = self.path
        os.environ.pop("FAKE_SPEC_NO_PHASES", None)

    def rebase(self):
        """The current HEAD becomes the run's base revision."""
        self.base = out(self.repo, "rev-parse", "HEAD")
        sh(self.repo, "bundle", "create", "-q", str(self.inp / "base.bundle"), "main")

    def merge_us1(self):
        """US1 already merged: its code, its ticks and (in the companion) its hidden tests."""
        write(self.repo, "app.py", APP_US1)
        write(self.repo, TASKS, tick(TASKS_BASE, "T001", "T002"))
        sh(self.repo, "commit", "-qam", "US1 merged")
        self.rebase()

    def change(self, app=None, tasks=None, extra=None):
        if app is not None:
            write(self.repo, "app.py", app)
        if tasks is not None:
            write(self.repo, TASKS, tasks)
        for rel, body in (extra or {}).items():
            write(self.repo, rel, body)
        sh(self.repo, "add", "-A")
        sh(self.repo, "commit", "-qm", "pilot: change")
        sh(self.repo, "bundle", "create", "-q", str(self.inp / "change.bundle"), f"{self.base}..HEAD")

    def meta(self, sl=None, **kw):
        m = {"run_id": "r", "task": "002", "attempt": 1, "item_id": "002", "base_revision": self.base,
             "allowed_paths": ["app.py", TASKS], "test_command": TEST_CMD, "hidden_mode": "overlay",
             "command_timeout_s": 60}
        if sl:
            m["slice"] = sl
        m.update(kw)
        return m

    def run_grade(self, meta):
        return grade.grade(meta, inp=str(self.inp), grading=str(self.grading), work=tempfile.mkdtemp(dir=self.tmp))

    @staticmethod
    def results(report):
        return {c["id"]: c["result"] for c in report["checks"]}

    @staticmethod
    def detail(report, cid):
        return next(c["detail"] for c in report["checks"] if c["id"] == cid)

    # --- US1: a whole-item run is graded exactly as before (T011) ---

    def test_whole_item_unchanged(self):
        write(self.grading, "overlay/tests/test_h1.py", HIDDEN_US1)
        write(self.grading, "overlay/tests/test_h2.py", HIDDEN_US2)
        self.change(app=APP_US2, tasks=tick(TASKS_BASE, "T001", "T002", "T003", "T004", "T005"))
        r = self.run_grade(self.meta())
        self.assertEqual(r["result"], "pass", r)
        self.assertEqual(list(self.results(r)), grade.CHECKS)
        self.assertTrue(self.detail(r, "G3_tasks_checked").startswith("tasks percent=100"))
        self.assertNotIn("G8_earlier_slices_kept", self.results(r))

    # --- US3: a slice stays inside its own boundaries (T016) ---

    def us1(self):
        return {"id": "US1", "phases": US1, "earlier": []}

    def test_slice_g3_passes_on_its_own_ticks(self):
        write(self.grading, "overlay/tests/test_h1.py", HIDDEN_US1)
        self.change(app=APP_US1, tasks=tick(TASKS_BASE, "T001", "T002"))
        r = self.run_grade(self.meta(self.us1()))
        self.assertEqual(r["result"], "pass", r)
        self.assertIn("2 phase(s) fully ticked", self.detail(r, "G3_tasks_checked"))

    def test_slice_g3_fails_when_another_slice_is_ticked(self):
        write(self.grading, "overlay/tests/test_h1.py", HIDDEN_US1)
        self.change(app=APP_US1, tasks=tick(TASKS_BASE, "T001", "T002", "T003"))
        r = self.run_grade(self.meta(self.us1()))
        self.assertEqual(self.results(r)["G3_tasks_checked"], "fail")
        self.assertIn("another slice's phase changed", self.detail(r, "G3_tasks_checked"))
        self.assertEqual(self.results(r)["G4_visible_tests"], "skipped")

    def test_slice_g3_fails_when_its_task_is_unticked(self):
        self.change(app=APP_US1, tasks=tick(TASKS_BASE, "T002"))
        r = self.run_grade(self.meta(self.us1()))
        self.assertEqual(self.results(r)["G3_tasks_checked"], "fail")
        self.assertIn("not fully ticked: 'Phase 1: Setup' 0/1", self.detail(r, "G3_tasks_checked"))

    def test_slice_g3_fails_when_outside_tasks_change(self):
        self.change(app=APP_US1, tasks=tick(TASKS_BASE, "T001", "T002", "T005"))
        r = self.run_grade(self.meta(self.us1()))
        self.assertIn("tasks outside phases changed", self.detail(r, "G3_tasks_checked"))

    def test_slice_g3_fails_on_a_renamed_heading(self):
        renamed = tick(TASKS_BASE, "T001", "T002").replace("Phase 3: User Story 2 - whisper", "Phase 3: User Story 2 - murmur")
        self.change(app=APP_US1, tasks=renamed)
        r = self.run_grade(self.meta(self.us1()))
        self.assertIn("heading changed", self.detail(r, "G3_tasks_checked"))

    def test_slice_g3_fails_closed_without_phases(self):
        os.environ["FAKE_SPEC_NO_PHASES"] = "1"
        self.change(app=APP_US1, tasks=tick(TASKS_BASE, "T001", "T002"))
        r = self.run_grade(self.meta(self.us1()))
        self.assertEqual(self.results(r)["G3_tasks_checked"], "fail")
        self.assertIn("needs spec-status 0.4.0+", self.detail(r, "G3_tasks_checked"))

    def test_slice_g2_uses_the_slice_paths(self):
        self.change(app=APP_US1, tasks=tick(TASKS_BASE, "T001", "T002"), extra={"other.py": "x = 1\n"})
        r = self.run_grade(self.meta(self.us1()))
        self.assertEqual(self.results(r)["G2_allowed_paths"], "fail")


if __name__ == "__main__":
    unittest.main()
