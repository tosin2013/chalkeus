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


NOTES = IMAGES.parents[1] / "test" / "integration" / "fixtures" / "notes"


class TestStubAuthor(unittest.TestCase):
    """The author role's workspace: spec.md, plan.md, tasks.md at the top, base/, draft/."""

    def run_author(self, item, spec_dir):
        d = Path(tempfile.mkdtemp())
        for f in ("spec.md", "plan.md", "tasks.md"):
            shutil.copy(NOTES / spec_dir / f, d / f)
        (d / "draft").mkdir()
        env = dict(os.environ, PILOT_STUB_DIR=str(SCEN))
        env.pop("PILOT_STUB_SCENARIO", None)
        prompt = f"You are the test author for item {item}. You draft its hidden acceptance tests"
        r = subprocess.run([str(STUB), "run", "--auto", "--model", "x", prompt], cwd=d, env=env, capture_output=True, text=True)
        return r, d

    def test_good_draft_and_its_map(self):
        import hashlib
        r, d = self.run_author("001", "specs/001-word-count")
        self.assertEqual(r.returncode, 0, r.stderr)
        self.assertTrue((d / "draft" / "tests" / "test_word_count_acceptance.py").is_file())
        m = json.loads((d / "draft" / ".acceptance-map.json").read_text())
        self.assertEqual(m["spec_sha256"], hashlib.sha256((d / "spec.md").read_bytes()).hexdigest())
        self.assertEqual(sorted(m["criteria"]), ["US1-AS1", "US1-AS2"])
        self.assertEqual(m["uncovered"], ["FR-001"])
        self.assertIn("[ ]", (d / "tasks.md").read_text(), "the author never ticks tasks")

    def test_colliding_draft_uses_a_path_the_plan_names(self):
        r, d = self.run_author("002", "specs/002-line-count")
        self.assertEqual(r.returncode, 0, r.stderr)
        self.assertTrue((d / "draft" / "tests" / "test_line_count.py").is_file())
        self.assertIn("tests/test_line_count.py", (d / "plan.md").read_text())

    def test_requires_tells_targets_apart(self):
        d = Path(tempfile.mkdtemp())
        (d / "spec.md").write_text("# Spec 001: Something else\n")
        env = dict(os.environ, PILOT_STUB_DIR=str(SCEN))
        env.pop("PILOT_STUB_SCENARIO", None)
        r = subprocess.run([str(STUB), "run", "You are the test author for item 001."], cwd=d, env=env, capture_output=True, text=True)
        self.assertEqual(r.returncode, 4, "no rule matches another target's item 001")
