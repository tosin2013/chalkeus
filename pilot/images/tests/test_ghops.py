"""ghops.py (004 GitHub writes) against a fake GitHub API: no token output, errors reported."""
import importlib.util
import json
import unittest
from pathlib import Path

_spec = importlib.util.spec_from_file_location("ghops", Path(__file__).resolve().parents[1] / "tools/ghops.py")
go = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(go)


class FakeAPI:
    def __init__(self, existing_labels=()):
        self.calls = []
        self.labels = set(existing_labels)

    def __call__(self, token, method, path, body=None):
        self.calls.append((method, path, body))
        if path == "/user":
            return 200, {"login": "pub-bot"}
        if method == "GET" and "/labels/" in path:
            name = path.rsplit("/", 1)[1].replace("%3A", ":")
            return (200, {}) if name in self.labels else (404, None)
        if method == "POST" and path.endswith("/labels"):
            return 201, {}
        if method == "POST" and path.endswith("/issues"):
            return 201, {"number": 7, "html_url": "u"}
        if method == "PATCH":
            return 200, {"number": 7}
        if method == "POST" and path.endswith("/comments"):
            return 201, {"id": 99, "html_url": "c"}
        return 500, None


class TestGhops(unittest.TestCase):
    def test_whoami(self):
        self.assertEqual(go.run("whoami", {}, "SECRET", FakeAPI()), {"ok": True, "result": {"login": "pub-bot", "app": False}})

    def test_ensure_labels_only_creates_missing(self):
        api = FakeAPI(existing_labels={"status:planned"})
        r = go.run("ensure-labels", {"repo": "o/r", "labels": [{"name": "status:planned"}, {"name": "status:verified"}]}, "T", api)
        self.assertEqual(r["result"]["created"], ["status:verified"])

    def test_create_comment_close(self):
        api = FakeAPI()
        self.assertEqual(go.run("create-issue", {"repo": "o/r", "title": "t", "body": "b"}, "T", api)["result"]["number"], 7)
        self.assertEqual(go.run("comment", {"repo": "o/r", "number": 7, "body": "x"}, "T", api)["result"]["id"], 99)
        self.assertTrue(go.run("close-issue", {"repo": "o/r", "number": 7}, "T", api)["ok"])

    def test_error_and_no_token_in_output(self):
        r = go.run("create-issue", {"repo": "o/r", "title": "t", "body": "b"}, "SECRET-VALUE", lambda *a, **k: (403, None))
        self.assertFalse(r["ok"])
        self.assertNotIn("SECRET-VALUE", json.dumps(r))

    def test_no_merge_op(self):
        self.assertFalse(any("merge" in k for k in go.OPS))
        self.assertFalse(go.run("merge", {}, "T", FakeAPI())["ok"])
