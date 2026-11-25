# T031: privacy check of 031's evidence (2026-11-25)

**Scope:** every file under `specs/031-macos-multiarch/evidence/` (234 files: runs T013, T025 and
its failed predecessor, T026, the isolation suite, notes). Pattern scans, counts only; nothing
matched was printed.

| Looked for | Files |
|---|---|
| GitHub tokens (`ghp_`, `gho_`, `ghs_`, `ghu_`, `ghr_`), fine-grained PATs | 0 |
| API keys (`sk-…`), AWS keys, `Bearer` headers | 0 |
| private keys (`BEGIN … PRIVATE KEY`), JWTs (`eyJ…`) | 0 |
| the model host (from `pilot/.build/model-host.txt`) | 0 |
| the provider key (`~/.config/chalkeus/opencode/*_api_key`), matched literally | 0 |
| email addresses | 0 |
| local home paths (`/Users/…`) | 1, see finding 1 |

Every run's own AU08 ("no secrets in the evidence record") also passed.

**Tests:**
- `go vet ./... && go test ./...`: pass.
- `python3 -m unittest discover -s pilot/images/tests -t pilot/images/tests`: 137 tests.
  - On the Mac, 2 tests in `test_author_role` fail only because macOS's `tar` adds AppleDouble
    `._` files.
  - With `COPYFILE_DISABLE=1`, all 137 pass. CI (Linux) is unaffected.
  - A known follow-up: set it in the test.

## Findings

1. **The test author's draft evidence was committed by mistake** in #210
   (`evidence/drafts/20261125T214011Z/`, 9 files, written there by the separate test-author
   session).
   - Draft evidence holds names, hashes and results, never test contents (docs/targets.md).
   - Its one sensitive item is a local home path, which names the maintainer's macOS account.
   - This change stops tracking it (the files stay on disk). It remains in the history of #210.
2. **`prefetch.json` keeps a 2,000-character tail of the prefetch log**, and for a target run that
   log is a test run that includes the hidden tests. Their file names and failure lines can
   appear there. `companion.json` also names the hidden-test files.
   - This isn't specific to 031: 14 committed `prefetch.json` files across specs follow the same
     pattern.
   - The worker never sees evidence, so runs aren't affected. But a private companion's test names
     and assertion messages reach a public repository when evidence is committed.
   - Proposed follow-up (forge-wide, not 031): redact hidden-test lines from `log_tail`, or keep
     prefetch logs out of committed evidence.
