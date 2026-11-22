# Evidence: integration tests in CI (025)

Test runs against the fake GitHub are never committed: they audit as test runs (AU12) and live in
the git-ignored `out` folder of `test/integration/`. This record holds the results that close the
feature.

## SC-001: each live bug, put back, fails its scenario (T017, 2026-11-22)

Each bug was reintroduced in a scratch worktree of `025-stage7-floor-close` (a one-line undo of
its fix), and only the matching scenario was run locally with
`go test -tags integration -run <scenario> ./test/integration/`. The baseline is the same code
without any undo.

| Run | Bug put back (fix) | Scenario | Result | How it failed |
|---|---|---|---|---|
| baseline | none | `TestAutoDraft`, `TestStepGaps`, `TestStepMode` | ✅ all pass | |
| images | author watch doesn't load the pulled images (#112) | `TestAutoDraft` | ✅ fails | `attempt 1 produced no draft (harness_error: no worker or tools image)` |
| pod label | A01 looks for `app=worker` (#113) | `TestAutoDraft` | ✅ fails | `FAIL A01 pods=1 containers=0`, `FAIL A02 no author pod found`; no PR |
| path collision | a draft may reuse a path the plan names (#115) | `TestAutoDraft` | ✅ fails | item 002 got `DRAFT_OK` and a second companion PR |
| issues setting | step requires `PILOT_GITHUB_ISSUES=on` (#121) | `TestStepGaps` | ✅ fails | step refused the `gaps` target, whose profile leaves it unset |
| terminal | step's runs inherit the terminal (#122) | `TestStepMode` | ✅ fails | `Confirm this mode and start staging workers? [y/N]` appeared in the terminal; the run ended `failed` with no tracking issue |

To catch the last two, stage 7 changed two scenarios: the `gaps` target leaves
`PILOT_GITHUB_ISSUES` unset, as the live profile did, and step mode's passes run under a
pseudo-terminal (`script`) without `PILOT_NO_TTY`, as an operator runs `pilot step`.

**#124 (a run's last lines missed after it ends), follow-up after the debrief (F2):** the step
scenario now keeps a long-running `pilot step` beside the run and asserts that it relays the run's
last line (`run … is failed; stopped watching`). That assertion failed even **with** #124's fix in
place, and found a related bug: merging the status PR makes the item verified, which drops it
from step mode's candidates while its run is still finishing, so the run's end was never relayed
(#124 fixed only items still proposed). Fixed (`step.finishRelays`: follow such a run until it
ends); with the fix the scenario passes and relays `FAILED: AUDIT` and the last line. In this
scenario the item always becomes verified, so #124's own undo (an item still proposed after its
run, for example a run that fails before its status PR) is still not exercised; it is covered by
the unit test `TestManagerRelaysTheEndOfARun`.

## SC-002: the job on GitHub-hosted runners

`integration` passed on PRs #140 (fixture), #146 (+ auto-draft, step) and #147 (+ gaps), each in
under 20 minutes on `ubuntu-24.04`, with no secrets.

## SC-004: coverage of the glue packages

Unit coverage is held by `.github/coverage-floor.txt` (go workflow). The scenarios' coverage
comes from the test pilot built with `-cover`; these numbers are from the SC-001 baseline round,
**without** the fixture scenario, so a full run is higher:

| Package | Unit tests | Integration scenarios |
|---|---|---|
| `internal/cli` | 22.7% | 48.2% |
| `internal/pipeline` | 21.7% | 57.5% |
| `internal/worker` | 15.5% | 53.1% |
| `internal/checks` | 20.9% | 79.4% |
| `internal/cluster` | 17.7% | 67.7% |
| `internal/grader` | 21.4% | 32.7% |
| `internal/step` | | 64.7% |
| `internal/author` | | 74.2% |

The integration job prints these in its summary on every run. Unit coverage itself did not rise
much in 025; what rose is the share of this code that a test runs, end to end.

## Real bugs the scenarios found while being built

1. The publisher's egress proxy only reaches port 443: correct, so the fake moved to 443.
2. The status-change commit failed silently without a git identity; the pilot now says why.
3. A03 probed an empty model host before any `pilot run`; `pilot up` now records it.
4. doctor failed every `pilot up --local-build` cluster, so step mode could never run there.
5. doctor claimed ungraded hidden tests "already pass" when the project's tests failed first.
6. step's `next:` line described the step just done (or said "nothing is waiting for you") in
   four places.
6a. step stopped relaying a run once its status PR's merge made the item verified, so "succeeded"
   (or the audit's result) never reached step's terminal (found by the F2 assertion).
7. The 024 docs described output and a file (`checks.jsonl`) the code never produced.

The fake GitHub also gained what the pilot needed from GitHub along the way (`pull/<n>/head`
refs, a stand-in model endpoint for doctor's key check).

## Privacy (T018)

025's diff (113 files) has no token or key patterns, no model host and no provider names. The
only credentials are the tests' dummies (`owner-token`, `bot-token`, `not-a-real-key`).
