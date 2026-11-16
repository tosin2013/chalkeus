# 004 evidence

Copy each validation run's record here from `pilot/runs/<run-id>/` on the machine that ran it.
Never paste tokens, keys, hostnames, IPs or personal data. The audit's AU08 must pass for
every run.

## Results (owner)

| Check (quickstart) | Result |
|---|---|
| §2 sync: 2 issues, one `status:planned` label each, task checklists; second sync 0/0; hand-edited label restored | covered by unit tests (`test_ghsync.py`, `vectors_test.go`); not run as a live step (single-maintainer) |
| §3 token mode: `/approve mode` and `/approve retry` resume the run | ✅ run `20261116T210522Z` — harness approvals (GitHub approvals rejected as `publisher_identity` in token mode, which is correct) |
| §3 refusals: `wrong_step` · `edited` · `withdrawn` | ✅ run `20261116T201149Z` (App mode, issue #29) |
| §3 refusals: `not_an_approver` · `publisher_identity` (approval) | unit-tested only (`approvals.json` vectors, `vectors_test.go`); single-maintainer, no second account |
| §3 status PR: `STATUS_PR_MERGED_BY_PUBLISHER` refused | ✅ run `20261116T210522Z` — publisher (`tosin2013`) merged status PR #35; run failed with `STATUS_PR_MERGED_BY_PUBLISHER` |
| §3 status PR: unedited merge accepted | ✅ run `20261116T201149Z` — status PR #31 merged unedited by owner; B proceeded, 11/11 |
| §3 audit: 11/11, AU09/AU10 `channels=['github']` with login; `github drift` empty | ✅ run `20261116T201149Z` — 11/11, all approvals `channel=github approver=tosin2013` |
| §4 App mode: PRs and posts by `<app>[bot]`; probe passes on fixture, fails on canary; audit 11/11 | ✅ run `20261116T201149Z` — publisher `tosins-specfoundry-pilot[bot]`; `app-token --check` passed |
| §4 App setup time from the docs alone (SC-007, target under 30 min, 0 outside lookups) | not timed; setup followed `docs/github-app.md` only |
| §5 terminal run with GitHub loop off: prompts as in the guide; AU09/AU10 `channels=['tty']` | covered by runs `20261115T162029Z` and `20261116T021112Z` in `specs/016-go-pilot/evidence/` |

## Run records

| Run | Mode | Publisher | Outcome | Notes |
|---|---|---|---|---|
| `20261116T201149Z` | App | `tosins-specfoundry-pilot[bot]` | ✅ 11/11 | T047 App mode + T052 walk-away. GitHub approvals: `wrong_step`, `edited`, `withdrawn` tested live. Status PR #31 merged unedited. Watcher ran detached. |
| `20261116T210522Z` | Token | `tosin2013` | ❌ failed (expected) | T047 token mode refusal cases. GitHub `/approve` rejected as `publisher_identity`. Status PR #35 merged by publisher → `STATUS_PR_MERGED_BY_PUBLISHER`. Run intentionally failed; AU08 passes. |

## Note for contributors

Two refusal cases require a second GitHub account and were not tested live:
- **`not_an_approver`**: an account not in `PILOT_APPROVERS` replies `/approve`.
- **`publisher_identity`** (approval path): the publisher account replies `/approve` when it differs from the operator (App mode separates these).

Both are covered by shared vectors in `specs/004-github-loop/contracts/vectors/approvals.json`
and exercised on every CI run by `internal/ghloop/vectors_test.go` and `pilot/tests/test_vectors.py`.
A contributor with a second GitHub account should run quickstart §3 step 2 and record the result here.
