"""025 R2: the stub harness that stands in for opencode in integration-test images."""
import json
import os
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path

IMAGES = Path(__file__).resolve().parents[1]
STUB = IMAGES / "stub" / "stub-opencode"
SCEN = IMAGES / "stub" / "scenarios"
FIXTURE = IMAGES.parent / "fixture"
REF = IMAGES / "tests" / "reference" / "slugkit"


class TestStub(unittest.TestCase):
    def run_stub(self, prompt, scenario="fixture", extra=None):
        d = Path(tempfile.mkdtemp())
        shutil.copytree(FIXTURE, d, dirs_exist_ok=True)
        if extra:
            shutil.copytree(SCEN / "fixture" / extra, d, dirs_exist_ok=True)
        env = dict(os.environ, PILOT_STUB_DIR=str(SCEN), PILOT_STUB_SCENARIO=scenario)
        r = subprocess.run([str(STUB), "run", "--auto", "--model", "x", prompt], cwd=d, env=env, capture_output=True, text=True)
        return r, d

    def test_induced_a1_writes_an_incomplete_slugify(self):
        r, d = self.run_stub("# Task list (`specs/001-slugify/tasks.md`)\n")
        self.assertEqual(r.returncode, 0, r.stderr)
        ev = json.loads(r.stdout.strip().splitlines()[-1])
        self.assertEqual((ev["type"], ev["part"]["tokens"]["input"]), ("step_finish", 1000))
        self.assertNotEqual((d / "src/slugkit/slug.py").read_text(), (REF / "slug.py").read_text())
        self.assertEqual((d / "specs/001-slugify/tasks.md").read_text().count("- [X]"), 3)

    def test_full_spec_writes_the_reference(self):
        r, d = self.run_stub("## Acceptance details\n...\n# Task list (`specs/001-slugify/tasks.md`)\n")
        self.assertEqual(r.returncode, 0, r.stderr)
        self.assertEqual((d / "src/slugkit/slug.py").read_text(), (REF / "slug.py").read_text())
        self.assertTrue((d / "tests/test_slug.py").is_file())

    def test_task_b(self):
        r, d = self.run_stub("## Acceptance details\n# Task list (`specs/002-unique-slug/tasks.md`)\n", extra="a-right")
        self.assertEqual(r.returncode, 0, r.stderr)
        self.assertEqual((d / "src/slugkit/unique.py").read_text(), (REF / "unique.py").read_text())
        self.assertIn("- [X]", (d / "specs/002-unique-slug/tasks.md").read_text())
        self.assertNotIn("- [X]", (d / "specs/001-slugify/tasks.md").read_text())

    def test_unknown_scenario_and_no_rule(self):
        self.assertEqual(self.run_stub("x", scenario="nope")[0].returncode, 3)
        self.assertEqual(self.run_stub("a prompt naming no task")[0].returncode, 4)

    def test_scenario_chosen_from_the_prompt(self):
        # No PILOT_STUB_SCENARIO: the rules that match the prompt decide (no Job template change).
        r, d = self.run_stub("## Acceptance details\n# Task list (`specs/001-slugify/tasks.md`)\n", scenario="")
        self.assertEqual(r.returncode, 0, r.stderr)
        self.assertEqual((d / "src/slugkit/slug.py").read_text(), (REF / "slug.py").read_text())


if __name__ == "__main__":
    unittest.main()
