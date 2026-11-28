"""publish.py on each Git host (037 T030/T031, contracts/publisher.md), against a local stub."""
import importlib.util
import json
import os
import threading
import unittest
import urllib.parse
from http.server import BaseHTTPRequestHandler, HTTPServer
from pathlib import Path
from unittest import mock

_spec = importlib.util.spec_from_file_location("pilot_publish_hosts", Path(__file__).resolve().parents[1] / "tools/publish.py")
publish_mod = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(publish_mod)

TOKEN = "ATCTT3xFfGN0-secret-token"
BOT = {"uuid": "{bot-uuid}", "nickname": "chalkeus-bot", "display_name": "chalkeus-bot"}


class Stub(BaseHTTPRequestHandler):
    """A Bitbucket /2.0 stub: lists leave drafts out unless the query asks for draft=true."""
    prs: list = []
    seen: list = []

    def log_message(self, *a):
        pass

    def _send(self, code, body):
        raw = json.dumps(body).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(raw)))
        self.end_headers()
        self.wfile.write(raw)

    def do_GET(self):
        Stub.seen.append(("GET", self.path, self.headers.get("Authorization")))
        u = urllib.parse.urlparse(self.path)
        q = urllib.parse.parse_qs(u.query).get("q", [""])[0]
        want_draft = "draft=true" in q
        head = q.split('source.branch.name="')[1].split('"')[0] if "source.branch.name" in q else None
        vals = [p for p in Stub.prs if p["state"] == "OPEN" and p["draft"] == want_draft and (head is None or p["source"]["branch"]["name"] == head)]
        self._send(200, {"values": vals})

    def do_POST(self):
        body = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
        Stub.seen.append(("POST", self.path, body))
        pr = {"id": 40 + len(Stub.prs), "state": "OPEN", "draft": body["draft"], "title": body["title"],
              "source": body["source"], "destination": body["destination"], "author": BOT,
              "links": {"html": {"href": f"https://bitbucket.org/ws/repo/pull-requests/{40 + len(Stub.prs)}"}}}
        Stub.prs.append(pr)
        self._send(201, pr)


SPEC = {"repo": "ws/repo", "head": "pilot/20261128T000000Z/006", "base": "main", "base_revision": "abc",
        "title": "feat(006): x", "body": "b", "run_id": "r", "task": "006"}


class TestBitbucket(unittest.TestCase):
    def setUp(self):
        Stub.prs, Stub.seen = [], []
        self.srv = HTTPServer(("127.0.0.1", 0), Stub)
        threading.Thread(target=self.srv.serve_forever, daemon=True).start()
        self.api = f"http://127.0.0.1:{self.srv.server_port}/2.0"
        self.spec = {**SPEC, "scm": {"kind": "bitbucket", "api": self.api, "git_base": "https://bitbucket.org/"}}

    def tearDown(self):
        self.srv.shutdown()
        self.srv.server_close()

    def test_opens_one_native_draft_and_reports_the_bot(self):
        pushes = []
        r = publish_mod.publish(publish_mod.client(self.spec, TOKEN), pushes.append, self.spec)
        posts = [s for s in Stub.seen if s[0] == "POST"]
        self.assertEqual(len(posts), 1)
        self.assertTrue(posts[0][2]["draft"])
        self.assertEqual(posts[0][2]["source"]["branch"]["name"], SPEC["head"])
        self.assertEqual((r["created"], r["draft"], r["author_id"], r["token_login"], r["author"]),
                         (True, True, "{bot-uuid}", "chalkeus-bot", "chalkeus-bot"))
        self.assertEqual(len(pushes), 1)
        self.assertTrue(all(s[2] == "Bearer " + TOKEN for s in Stub.seen if s[0] == "GET"))

    def test_reuses_an_open_draft_found_only_with_draft_true(self):
        c = publish_mod.client(self.spec, TOKEN)
        publish_mod.publish(c, lambda s: None, self.spec)
        pushes = []
        r = publish_mod.publish(c, pushes.append, self.spec)
        self.assertEqual((r["created"], pushes, len(Stub.prs)), (False, [], 1))
        self.assertTrue(any("draft%3Dtrue" in s[1] or "draft=true" in urllib.parse.unquote_plus(s[1]) for s in Stub.seen if s[0] == "GET"))

    def test_push_uses_x_token_auth_and_never_prints_the_token(self):
        calls = []

        def fake_run(args, cwd=None, env=None, check=None, stdout=None, stderr=None):
            if args[1] == "clone":
                os.makedirs(args[-1], exist_ok=True)
                calls.append((args, open(env["GIT_ASKPASS"]).read(), env.get("GIT_SSL_CAINFO")))
            else:
                calls.append((args, None, None))
        spec = {**self.spec, "scm": {**self.spec["scm"], "ca_file": "/etc/chalkeus/scm-ca.pem"}}
        with mock.patch.object(publish_mod.subprocess, "run", side_effect=fake_run), mock.patch.dict(os.environ, {"PILOT_SCM_TOKEN": TOKEN}):
            publish_mod.git_push(spec, bundle="/dev/null")
        clone, askpass, ca = calls[0]
        self.assertIn("https://bitbucket.org/ws/repo.git", clone)
        self.assertIn("echo x-token-auth", askpass)
        self.assertNotIn(TOKEN, askpass)
        self.assertEqual(ca, "/etc/chalkeus/scm-ca.pem")
        self.assertNotIn(TOKEN, json.dumps([c[0] for c in calls]))

    def test_github_is_unchanged_without_a_scm_block(self):
        self.assertIsInstance(publish_mod.client(SPEC, TOKEN), publish_mod.GitHub)
        self.assertIsInstance(publish_mod.client({**SPEC, "scm": {"kind": "github"}}, TOKEN), publish_mod.GitHub)
        with self.assertRaises(ValueError):
            publish_mod.client({**SPEC, "scm": {"kind": "gitea"}}, TOKEN)

    def test_no_merge_code_path(self):
        src = (Path(__file__).resolve().parents[1] / "tools/publish.py").read_text()
        for banned in ("/merge", "merge_method", "/approve", "/decline"):
            self.assertNotIn(banned, src)


if __name__ == "__main__":
    unittest.main()
