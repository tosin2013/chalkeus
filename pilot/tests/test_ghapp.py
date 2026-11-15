"""004 US4: GitHub App token minting (research R5) with a throwaway key; no network."""
import base64
import datetime as dt
import importlib.util
import json
import os
import stat
import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest import mock

from specfoundry_pilot import config, ghapp

_spec = importlib.util.spec_from_file_location("app_token", Path(__file__).resolve().parents[1] / "tools/app_token.py")
at = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(at)

REPO = "o/x-pilot-fixture"


def unb64(s):
    return base64.urlsafe_b64decode(s + "=" * (-len(s) % 4))


class FakeAPI:
    def __init__(self, perms=None, repos=None, mint_status=201):
        self.calls = []
        self.perms = perms or {"contents": "write", "pull_requests": "write", "issues": "write", "metadata": "read"}
        self.repos = repos if repos is not None else [{"full_name": REPO}]
        self.mint_status = mint_status

    def __call__(self, method, path, bearer, body=None):
        self.calls.append((method, path, body))
        if path == "/app":
            return 200, {"slug": "specfoundry-pilot"}
        if path.endswith("/access_tokens"):
            if self.mint_status != 201:
                return self.mint_status, None
            return 201, {"token": "ghs_SECRETSECRETSECRET1234", "expires_at": "2026-11-16T11:00:00Z",
                         "permissions": self.perms, "repositories": self.repos}
        raise AssertionError(path)


class TestMintChild(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.tmp = tempfile.TemporaryDirectory()
        cls.key = Path(cls.tmp.name) / "k.pem"
        subprocess.run(["openssl", "genrsa", "-out", str(cls.key), "2048"], check=True, capture_output=True)
        subprocess.run(["openssl", "rsa", "-in", str(cls.key), "-pubout", "-out", str(cls.key) + ".pub"],
                       check=True, capture_output=True)

    @classmethod
    def tearDownClass(cls):
        cls.tmp.cleanup()

    def test_jwt_claims_and_signature(self):
        tok = at.jwt(str(self.key), "123", now=1_000_000)
        h, p, s = tok.split(".")
        self.assertEqual(json.loads(unb64(h)), {"alg": "RS256", "typ": "JWT"})
        self.assertEqual(json.loads(unb64(p)), {"iat": 1_000_000 - 60, "exp": 1_000_000 + 540, "iss": "123"})
        sig = Path(self.tmp.name) / "sig"
        sig.write_bytes(unb64(s))
        v = subprocess.run(["openssl", "dgst", "-sha256", "-verify", str(self.key) + ".pub", "-signature", str(sig)],
                           input=f"{h}.{p}".encode(), capture_output=True)
        self.assertEqual(v.returncode, 0, v.stderr)

    def test_token_request_is_repo_scoped_with_three_permissions(self):
        self.assertEqual(at.token_request(REPO), {"repositories": ["x-pilot-fixture"], "permissions":
                                                  {"contents": "write", "pull_requests": "write", "issues": "write"}})

    def test_mint_writes_mode_600_and_prints_no_secret(self):
        out = Path(self.tmp.name) / "t.env"
        res = at.mint(str(self.key), "123", "456", REPO, str(out), call=FakeAPI())
        self.assertTrue(res["ok"], res)
        self.assertEqual(res["login"], "specfoundry-pilot[bot]")
        self.assertNotIn("ghs_", json.dumps(res))
        self.assertEqual(stat.S_IMODE(out.stat().st_mode), 0o600)
        self.assertIn("PILOT_SCM_KIND=app", out.read_text())

    def test_mint_problems(self):
        out = Path(self.tmp.name) / "t2.env"
        self.assertFalse(at.mint(str(self.key), "1", "2", REPO, str(out), call=FakeAPI(mint_status=404))["ok"])
        r = at.mint(str(self.key), "1", "2", REPO, str(out), call=FakeAPI(perms={"contents": "write"}))
        self.assertFalse(r["ok"])
        self.assertIn("issues", " ".join(r["problems"]))
        r = at.mint(str(self.key), "1", "2", REPO, str(out), call=FakeAPI(repos=[{"full_name": REPO}, {"full_name": "o/other"}]))
        self.assertFalse(r["ok"])

    def test_bad_key_is_reported(self):
        bad = Path(self.tmp.name) / "bad.pem"
        bad.write_text("not a key")
        self.assertFalse(at.mint(str(bad), "1", "2", REPO, str(Path(self.tmp.name) / "x.env"), call=FakeAPI())["ok"])


class TestHostSide(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        d = Path(self.tmp.name)
        v = config.validate_values({"PILOT_FIXTURE_REPO": REPO, "PILOT_MODEL": "p/m", "PILOT_MODEL_EGRESS_HOSTS": "h",
                                    "PILOT_PUBLISHER_CREDENTIAL": "app", "PILOT_APP_ID": "1", "PILOT_APP_INSTALLATION_ID": "2"})
        self.cfg = config.Config(values=v, config_dir=d, model_dir=d / "opencode", publisher_env=d / "publisher.env")

    def test_fresh_margin(self):
        now = dt.datetime(2026, 11, 16, 10, 0, tzinfo=dt.timezone.utc)
        self.assertTrue(ghapp._fresh({"expires_at": "2026-11-16T10:30:00Z"}, now))
        self.assertFalse(ghapp._fresh({"expires_at": "2026-11-16T10:04:00Z"}, now))
        self.assertFalse(ghapp._fresh({}, now))

    def test_current_env_mints_when_missing_and_reuses_when_fresh(self):
        calls = []

        def fake_mint(cfg, key=None, out=None):
            calls.append(1)
            ghapp.env_path(cfg).write_text("PILOT_SCM_TOKEN=x\n")
            exp = (dt.datetime.now(dt.timezone.utc) + dt.timedelta(hours=1)).strftime("%Y-%m-%dT%H:%M:%SZ")
            ghapp.info_path(cfg).write_text(json.dumps({"login": "b[bot]", "expires_at": exp}))
            return {"ok": True}
        with mock.patch.object(ghapp, "mint", side_effect=fake_mint):
            self.assertEqual(ghapp.current_env(self.cfg), ghapp.env_path(self.cfg))
            ghapp.current_env(self.cfg)
            self.assertEqual(len(calls), 1)
            self.assertEqual(ghapp.bot_login(self.cfg), "b[bot]")

    def test_mint_failure_fails_closed(self):
        from specfoundry_pilot.ghsync import GitHubLoopError
        with mock.patch.object(ghapp, "mint", return_value={"ok": False, "problems": ["nope"]}):
            with self.assertRaises(GitHubLoopError):
                ghapp.current_env(self.cfg)

    def test_missing_key_is_a_config_error(self):
        with self.assertRaises(config.ConfigError):
            ghapp.mint(self.cfg)

    def test_discard(self):
        ghapp.env_path(self.cfg).write_text("x")
        ghapp.info_path(self.cfg).write_text("{}")
        ghapp.discard(self.cfg)
        self.assertFalse(ghapp.env_path(self.cfg).exists() or ghapp.info_path(self.cfg).exists())


if __name__ == "__main__":
    unittest.main()
