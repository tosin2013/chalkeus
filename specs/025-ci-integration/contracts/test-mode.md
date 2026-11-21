# Contract: test mode and the fake GitHub

## Settings (environment only)

| Variable | Meaning |
|---|---|
| `PILOT_TEST_MODE=1` | allows the two below; marks every run as a test run |
| `PILOT_TEST_GITHUB_API` | replaces `https://api.github.com` for the pilot, the publisher and the test author |
| `PILOT_TEST_GIT_BASE` | replaces `https://github.com/` for clones and pushes |
| `PILOT_STUB_SCENARIO` | optional override for the stub harness; normally the stub picks the scenario whose rules match the prompt (no Job template carries it, so the fixture's manifests stay byte-identical) |

Either address without `PILOT_TEST_MODE=1`: every command exits 2, with "test endpoints need
PILOT_TEST_MODE=1 (they are for the integration tests only)".

## Evidence

- `config.json` gains `"test_mode": true`, `"test_github_api"` and `"test_git_base"`.
- The audit adds `AU12 not a test run`, failing, when `config.json` has `test_mode`. Records
  without the field audit as before.

## Fake GitHub test API (`/_test/`, test harness only)

| Call | Effect |
|---|---|
| `POST /_test/repos` `{repo, from}` | create a repository from a fixture folder (one commit on `main`) |
| `POST /_test/comment` `{repo, issue, login, body}` | a comment by `login` |
| `POST /_test/label` `{repo, issue, login, label, remove?}` | a label added or removed by `login` (records an issue event) |
| `POST /_test/merge` `{repo, pr, login}` | merge a PR into its base as `login` |
| `GET /_test/state` | repositories, issues, PRs and comments as JSON, for assertions |
