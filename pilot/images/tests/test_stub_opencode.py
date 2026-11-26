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


    def test_a_rule_can_name_the_attempt(self):
        # 029: the deps scenario edits the manifest on attempt 1 and fixes the code after it.
        target = IMAGES.parents[1] / "test" / "integration" / "fixtures" / "deps"
        prompt = "# Task list (`specs/001-greetlib-2/tasks.md`)\n"
        for n, want in ((1, "edit-manifest"), (2, "fix-code"), (None, "fix-code")):
            with self.subTest(attempt=n):
                d = Path(tempfile.mkdtemp())
                shutil.copytree(target, d, dirs_exist_ok=True)
                (d / "requirements.txt").write_text("greetlib==2.0\n")  # the approved bump, as the worker gets it
                meta = d.parent / (d.name + "-meta.json")
                if n is not None:
                    meta.write_text(json.dumps({"attempt": n}))
                env = dict(os.environ, PILOT_STUB_DIR=str(SCEN), PILOT_STUB_SCENARIO="", PILOT_STUB_META=str(meta))
                r = subprocess.run([str(STUB), "run", prompt], cwd=d, env=env, capture_output=True, text=True)
                self.assertEqual(r.returncode, 0, r.stderr)
                manifest_edited = (d / "requirements.txt").read_text() == "greetlib==1.0\n"
                code_fixed = "Hello, " in (d / "src/greet/__init__.py").read_text()
                self.assertEqual((manifest_edited, code_fixed), (want == "edit-manifest", want == "fix-code"))
                self.assertEqual((d / "specs/001-greetlib-2/tasks.md").read_text().count("- [X]"), 2)


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


class TestStubSlices(unittest.TestCase):
    """026: in a slice run the stub ticks only the slice's phases, as a real worker must."""

    def test_tick_only_the_slice(self):
        import importlib.machinery
        import importlib.util
        loader = importlib.machinery.SourceFileLoader("stub", str(STUB))
        spec = importlib.util.spec_from_loader("stub", loader)
        stub = importlib.util.module_from_spec(spec)
        loader.exec_module(stub)
        prompt = ("# Task list (`specs/001-x/tasks.md`)\n\n...\n\n# This run: slice US2\n\nBuild only slice US2: ...\n\n"
                  "- Phase 3: User Story 2 - b\n- Phase 4: Polish\n")
        self.assertEqual(stub.slice_phases(prompt), ["Phase 3: User Story 2 - b", "Phase 4: Polish"])
        self.assertEqual(stub.slice_phases("# Task list (`x`)\n- [ ] a\n"), [])
        tasks = ("# Tasks\n\n## Phase 1: Setup\n\n- [ ] T001 a\n\n## Phase 3: User Story 2 - b\n\n- [ ] T002 b\n"
                 "### Tests\n- [ ] T003 c\n\n## Phase 4: Polish\n\n- [ ] T004 d\n\n## Notes\n\n- [ ] T005 e\n")
        got = stub.tick_phases(tasks, stub.slice_phases(prompt))
        self.assertIn("- [ ] T001 a", got)
        self.assertIn("- [X] T002 b", got)
        self.assertIn("- [X] T003 c", got)  # a ### heading stays inside its phase, as in spec-status
        self.assertIn("- [X] T004 d", got)
        self.assertIn("- [ ] T005 e", got)
