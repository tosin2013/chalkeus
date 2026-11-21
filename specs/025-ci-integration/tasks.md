# Tasks: Integration tests in CI

**Input**: `specs/025-ci-integration/` (spec.md, plan.md, research.md, contracts/test-mode.md,
quickstart.md)

**Tests**: included. Every stage has unit tests; the integration scenarios are the feature's
own test. Stages follow research R6, and each is mergeable on its own.

## Stage 1: Test mode and endpoints (US3)

- [X] T001 [US3] Read `PILOT_TEST_MODE`, `PILOT_TEST_GITHUB_API` and `PILOT_TEST_GIT_BASE` in `internal/config/config.go`: refuse either address without test mode (exit 2, the contract's message); with test mode set `ghloop.API` and `fixture.BaseURL`, and record `test_mode` and both addresses in `config.json`; tests in `internal/config/config_test.go`
- [X] T002 [US3] In test mode, pass `GITHUB_API` and `GIT_BASE` to the publisher and test-author Jobs (`pilot/manifests/publisher-job.yaml.tmpl`, `internal/publisher/`), point the SCM proxy's allowlist at the fake's host (`internal/cluster/infra.go`), and make `pilot/images/tools/publish.py`, `ghops.py` and `check_token.py` read them (defaults unchanged); Python tests in `pilot/images/tests/`
- [X] T003 [US3] Add AU12 "not a test run" to `internal/audit/audit.go`, failing only when `config.json` has `test_mode`; the golden audit is unchanged; test

## Stage 2: The stub harness (US2)

- [X] T004 [US2] Write `pilot/images/worker/stub-opencode`: read `PILOT_STUB_SCENARIO`, copy `stub-scenarios/<name>/files/` into the workspace (worker) or `draft/` (author), print one `step_finish` event, exit with the scenario's code; scenarios `fixture-a1-induced`, `fixture-pass`, `item-pass`, `author-draft`, `author-collide`; Python tests run it with a fake workspace
- [X] T005 [US2] Make `internal/images` copy the stub over `/usr/local/bin/opencode` only for a test build (`--local-build` with `PILOT_TEST_MODE=1`), and pass `PILOT_STUB_SCENARIO` to worker and author Jobs only in test mode (`internal/worker/worker.go`, `internal/author/job.go`); a test that a non-test build never includes the stub
  Done as (2026-11-21): a test build adds a layer on top of the built worker (`pilot-worker-test`, `images.BuildStub`); the scenario is chosen by the stub from the prompt, not passed through the Jobs, because the fixture's manifests must stay byte-identical (the parity test caught it). The fixture scenario is in `pilot/images/stub/scenarios/fixture/`; the item and author scenarios come with their fixtures (stages 4-5).

## Stage 3: The fake GitHub (US1, US2)

- [X] T006 [P] Write `cmd/fakegithub` REST: users, repositories and permissions, branch protection, pulls (list, get, create, edit, files, merge), issues (list, get, create, edit, comments, events), labels, git refs, commits and trees, contents, and the App installation token; state in memory; handler tests with go-github as the client
- [ ] T007 Serve git smart HTTP through `git http-backend` over bare repositories, with pushes updating the refs the REST side reads; test: clone, push a branch, open and merge a PR through REST, clone again
- [ ] T008 Add the `/_test/` owner API (repos from fixture folders, comment, label with an issue event, merge, state) per `contracts/test-mode.md`; tests

## Stage 4: The harness and the fixture scenario (US1, US2)

- [ ] T009 Write `test/integration/` (build tag `integration`): build the pilot and the fake, start the fake on the Docker network's gateway, write a temporary config (pilot.env, the fixture repository, approvers, a model stub), run `pilot up --local-build` in test mode once, and `down` at the end; helpers to run pilot commands and read evidence
- [ ] T010 [US2] Scenario **fixture**: `run` (A/1 fails by induction), `/approve retry` through the fake, A/2 passes, merge A's PR and its status PR, B runs and opens its PR; assert P01–P10, GM01–GM04, the PRs on the fake, and the audit rows (AU12 failing, the others passing except AU07)

## Stage 5: Drafting and step mode (US1, US2)

- [ ] T011 [US2] Scenario **auto-draft**: a test target with auto-draft on; `author watch --once` with `author-draft`, asserting A01–A05, `DRAFT_OK` and a companion PR on the fake; then with `author-collide`, asserting `DRAFT_PATH_IN_SPEC` and no PR
- [ ] T012 [US2] Scenario **step mode**: an approver's label through the fake; `step --once` passes: draft, merge the tests, doctor and run (asserting the run pauses at `paused(awaiting-mode)` with `channel=none`), `/approve mode`, merge the item and status PRs; assert `run_started_by step`, the run succeeds, and `decisions.jsonl` has each step

## Stage 6: The debrief gaps (US2)

- [ ] T013 [US2] Scenarios: a non-approver's label starts nothing; a failing doctor (a target whose tests fail) gives exactly one comment and no run
- [ ] T014 Add a stale-run rule to `internal/step/gather.go`: an `in-progress` run with no watcher and no event for an hour counts as failed (then waits for a relabel); unit test, and a scenario with a crashed run

## Stage 7: CI and the coverage floor (US1, US4)

- [ ] T015 [US1] Add `.github/workflows/integration.yml`: weekly and on PRs touching `cmd/`, `internal/`, `pilot/manifests/`, `pilot/images/` or itself; `ubuntu-24.04`, no secrets, a 45-minute timeout; artifacts on failure
- [ ] T016 [US4] Add `.github/coverage-floor.txt` with today's numbers for `cli`, `pipeline`, `worker`, `checks`, `cluster`, `grader`, and a step in `.github/workflows/go.yml` that fails a package below its floor
- [ ] T017 [US1] SC-001 check: reintroduce each live bug on a scratch branch (images not loaded, the A01 label, a terminal for step mode's subprocesses, the issues rule) and confirm the job fails at the matching scenario; record it in `specs/025-ci-integration/evidence/README.md`
- [ ] T018 Docs (`CONTRIBUTING.md` or `docs/`: running the integration tests locally), privacy check, and the 025 debrief

## Dependencies

- Stage 1 → 2 → 3 → 4 → 5 → 6 → 7. T006 can start alongside Stage 2.
- **MVP:** Stages 1–4 (the fixture scenario on kind with the fake GitHub).
