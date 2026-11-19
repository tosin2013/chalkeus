"""The worker image's author role (024 R3): workspace in, a framed report.json and draft.tar out.

Runs pilot/images/worker/author.sh with a fake `opencode` on PATH and temporary paths.
"""
import io
import json
import os
import stat
import subprocess
import tarfile
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SCRIPT = ROOT / "worker" / "author.sh"


def decode(raw: bytes):
    import importlib.util
    spec = importlib.util.spec_from_file_location("frame", ROOT / "common" / "frame.py")
    frame = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(frame)
    return frame.decode(raw.decode())


FAKE_OPENCODE = r"""#!/usr/bin/env bash
# fake harness: writes what $FAKE_ACTION says into the workspace, prints one step_finish event
case "$FAKE_ACTION" in
  draft)  mkdir -p draft/src/test && echo 'class T {}' > draft/src/test/T.java && echo '{}' > draft/.acceptance-map.json ;;
  escape) mkdir -p draft && ln -s /etc/passwd draft/link && echo x > draft/ok.txt ;;
  base)   echo hacked > base/README.md 2>/dev/null; mkdir -p draft && echo x > draft/a.txt ;;
  none)   ;;
esac
echo '{"type":"step_finish","part":{"id":"s1","type":"step-finish","cost":0,"tokens":{"input":100,"output":5,"reasoning":0,"cache":{"read":0,"write":0}}}}'
exit "${FAKE_EXIT:-0}"
"""


class TestAuthorRole(unittest.TestCase):
    def run_role(self, action, exit_code=0):
        d = Path(tempfile.mkdtemp())
        inp, work, tmp, lib, bin_ = d / "in", d / "work", d / "tmp", d / "lib", d / "bin"
        for p in (inp, work, tmp, lib, bin_):
            p.mkdir()
        for f in ("frame.py",):
            (lib / f).write_bytes((ROOT / "common" / f).read_bytes())
        (lib / "watchdog.py").write_bytes((ROOT / "worker" / "watchdog.py").read_bytes())
        oc = bin_ / "opencode"
        oc.write_text(FAKE_OPENCODE)
        oc.chmod(oc.stat().st_mode | stat.S_IEXEC)
        (inp / "meta.json").write_text(json.dumps({"role": "author", "draft_id": "D1", "item": "009", "attempt": 1}))
        (inp / "prompt.md").write_text("draft tests")
        buf = io.BytesIO()
        with tarfile.open(fileobj=buf, mode="w") as t:
            for name, data in {"spec.md": b"# spec", "criteria.json": b"{}", "base/README.md": b"base"}.items():
                info = tarfile.TarInfo(name)
                info.size = len(data)
                t.addfile(info, io.BytesIO(data))
        (inp / "workspace.tar").write_bytes(buf.getvalue())
        env = dict(os.environ, PILOT_IN=str(inp), PILOT_WORK=str(work), PILOT_LIB=str(lib), PILOT_TMP=str(tmp),
                   PATH=f"{bin_}:{os.environ['PATH']}", PILOT_MODEL="p/m", FAKE_ACTION=action, FAKE_EXIT=str(exit_code))
        r = subprocess.run(["bash", "-c", f'exec 3>&1; exec 1>>"{tmp}/entry.log" 2>&1; exec "{SCRIPT}"'],
                           env=env, capture_output=True, timeout=60)
        self.assertEqual(r.returncode, 0, r.stderr)
        files, err = decode(r.stdout)
        self.assertIsNone(err, r.stdout[-500:])
        report = json.loads(files["report.json"])
        draft = {}
        if "draft.tar" in files:
            with tarfile.open(fileobj=io.BytesIO(files["draft.tar"])) as t:
                draft = {m.name: m for m in t.getmembers()}
        return report, draft, work

    def test_draft_out(self):
        report, draft, _ = self.run_role("draft")
        self.assertEqual((report["role"], report["draft_id"], report["error"]), ("author", "D1", None))
        self.assertEqual(report["spend"]["tokens"], 105)
        self.assertIn("draft/src/test/T.java", draft)
        self.assertIn("draft/.acceptance-map.json", draft)
        self.assertTrue(all(n.startswith("draft") for n in draft))

    def test_links_never_leave(self):
        _, draft, _ = self.run_role("escape")
        self.assertIn("draft/ok.txt", draft)
        self.assertNotIn("draft/link", draft)

    def test_base_is_read_only(self):
        _, draft, work = self.run_role("base")
        self.assertEqual((work / "task" / "base" / "README.md").read_text(), "base")
        self.assertEqual(sorted(n for n in draft if not draft[n].isdir()), ["draft/a.txt"])

    def test_no_draft_and_harness_error(self):
        report, draft, _ = self.run_role("none", exit_code=3)
        self.assertEqual(report["error"], "harness_error")
        self.assertEqual(draft, {})


if __name__ == "__main__":
    unittest.main()
