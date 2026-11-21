"""025: the publisher's scripts read GITHUB_API (test mode only; the default is GitHub)."""
import importlib.util
import os
import unittest
from pathlib import Path

TOOLS = Path(__file__).resolve().parents[1] / "tools"


def load(name):
    spec = importlib.util.spec_from_file_location(f"t_{name}", TOOLS / f"{name}.py")
    m = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(m)
    return m


class TestEndpoints(unittest.TestCase):
    def test_default_and_override(self):
        old = os.environ.pop("GITHUB_API", None)
        try:
            for name in ("publish", "ghops", "check_token"):
                self.assertEqual(load(name).API, "https://api.github.com", name)
            os.environ["GITHUB_API"] = "http://172.18.0.1:8090/"
            for name in ("publish", "ghops", "check_token"):
                self.assertEqual(load(name).API, "http://172.18.0.1:8090", name)
            os.environ["GITHUB_API"] = ""
            self.assertEqual(load("publish").API, "https://api.github.com", "an empty value keeps GitHub")
        finally:
            os.environ.pop("GITHUB_API", None)
            if old is not None:
                os.environ["GITHUB_API"] = old


if __name__ == "__main__":
    unittest.main()
