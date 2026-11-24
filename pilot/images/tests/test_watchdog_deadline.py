"""030 soft deadline (contracts/worker-soft-deadline.md): the watchdog stops the harness before
the Job's deadline, so the entrypoint can still collect the change and usage."""
import importlib.util
import json
import os
import subprocess
import sys
import tempfile
import time
import unittest
from pathlib import Path

_path = Path(__file__).resolve().parents[1] / "worker/watchdog.py"
_spec = importlib.util.spec_from_file_location("watchdog", _path)
wd = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(wd)


class TestSoftDeadlineValue(unittest.TestCase):
    def test_parse(self):
        self.assertEqual(wd.soft_deadline({}), 0)
        self.assertEqual(wd.soft_deadline({"PILOT_SOFT_DEADLINE_S": ""}), 0)
        self.assertEqual(wd.soft_deadline({"PILOT_SOFT_DEADLINE_S": "x"}), 0)
        self.assertEqual(wd.soft_deadline({"PILOT_SOFT_DEADLINE_S": "1080"}), 1080)

    def test_no_timer_without_pid_or_deadline(self):
        m = wd.Meter(1.0, 10)
        self.assertIsNone(wd.arm_deadline(m, 0, 5))
        self.assertIsNone(wd.arm_deadline(m, 123, 0))


class TestSoftDeadlineStopsHarness(unittest.TestCase):
    def run_watchdog(self, harness: str, deadline: str, kill_after_env: str = ""):
        with tempfile.TemporaryDirectory() as tmp:
            h = subprocess.Popen([sys.executable, "-c", harness], stdout=subprocess.PIPE, text=True)
            env = dict(os.environ, PILOT_TMP=tmp, PILOT_SOFT_DEADLINE_S=deadline)
            start = time.monotonic()
            w = subprocess.run([sys.executable, str(_path), str(h.pid)], stdin=h.stdout, env=env, timeout=60)
            h.wait(timeout=30)
            took = time.monotonic() - start
            self.assertEqual(w.returncode, 0)
            return json.load(open(os.path.join(tmp, "spend.json"))), h.returncode, took

    def test_sigterm_at_deadline(self):
        spend, code, took = self.run_watchdog(
            "import json,time\nprint(json.dumps({'type':'step_finish','part':{'id':'a','cost':0.01,"
            "'tokens':{'input':10,'output':1}}}),flush=True)\ntime.sleep(60)", "1")
        self.assertTrue(spend["stopped_by_deadline"])
        self.assertFalse(spend["stopped_by_cap"])
        self.assertEqual(spend["tokens"], 11)  # usage before the stop is kept
        self.assertLess(code, 0)  # killed by a signal
        self.assertLess(took, 20)

    def test_sigkill_when_sigterm_ignored(self):
        spend, code, took = self.run_watchdog(
            "import signal,time\nsignal.signal(signal.SIGTERM, signal.SIG_IGN)\ntime.sleep(60)", "1")
        self.assertTrue(spend["stopped_by_deadline"])
        self.assertEqual(code, -9)
        self.assertLess(took, 30)

    def test_no_deadline_when_harness_finishes_first(self):
        spend, code, _ = self.run_watchdog("print('{}')", "30")
        self.assertFalse(spend["stopped_by_deadline"])
        self.assertEqual(code, 0)


if __name__ == "__main__":
    unittest.main()
