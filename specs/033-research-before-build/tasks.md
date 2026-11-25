# Tasks: Research before build

**Input**: `specs/033-research-before-build/` (spec, plan, research R1–R9, data-model, contracts,
quickstart)
**Tests**: included: the spec's success criteria are replay and parity tests (SC-001, SC-002).
Check every phase with `go vet ./... && go test ./...`.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel (different files, no dependency on an unfinished task)
- **[Story]**: the user story the task belongs to

---

## Phase 1: Setup

- [ ] T001 Copy the replay inputs into `internal/diagnose/testdata/documcp-20261125T163307Z-001-1/`: `report.json`, `meta.json`, `proxy-denials.log` and the `# Task list` section of `prompt.md` from `specs/026-slice-runs/evidence/20261125T163307Z/attempts/001-1/`, plus that attempt's record and `readiness.allowed_paths` from the run's `run.json` (as `attempt.json`); check none holds a key, token or the model host (AGENTS.md)
- [ ] T002 [P] Copy the fixture's A/1 equivalents from `specs/016-go-pilot/evidence/20261117T001332Z/` into `internal/diagnose/testdata/fixture-A-1/` (report, grade.json, proxy-denials.log, task list, attempt record)

---

## Phase 2: Foundational (shared by US1 and US3)

- [ ] T003 Create `internal/diagnose/wording.go`: `ResearchPhrases` (research R3: `locate`, `find out`, `find the`, `find which`, `investigate`, `identify the cause`, `identify which`, `figure out`, `determine why`, `determine which`, `diagnose`, `research`, `explore`, `spike`) and `ResearchTask(desc string) (phrase string, ok bool)`: case-insensitive, matched at the start of the description after the task ID and `[P]`/`[USn]` markers
- [ ] T004 [P] Create `internal/diagnose/catalogue.go`: `KnownReasons` (research R4: `deadline`, `spend_cap`, `grade_fail`, `harness_error`, `no_result`, `bad_frame`, `oversize`, `grader_error`, `grader_timeout`, `clone_failed`, `checkout_failed`, `workspace_failed`, `bundle_failed`, `bad_meta`, `induction_failed`) and `Known(reason string) bool`
- [ ] T005 [P] Unit tests in `internal/diagnose/wording_test.go`: documcp's old T003 ("Locate the suites …") and T004a ("Find the test that writes …") match; T001, T002, T005, T006 and "Set X; find and replace Y" don't; markers are skipped

**Checkpoint**: wording and catalogue ready for the diagnosis and doctor.

---

## Phase 3: User Story 1 - A failed attempt says why, and what to do next (Priority: P1) 🎯 MVP

**Goal**: every failed attempt gets a deterministic diagnosis shown with the retry question, in
the summary and on the tracking issue.

**Independent Test**: `go test ./internal/diagnose/ -run Replay`: documcp attempt 1 →
`back_to_planning` naming T003, `package-lock.json` and one `github.com` refusal by the target
proxy; fixture A/1 → `retry` with no egress signal.

### Tests for User Story 1

- [ ] T006 [P] [US1] Replay tests in `internal/diagnose/diagnose_test.go` on both testdata folders: recommendation, triggers, routes, signal kinds and sources as in contracts/diagnosis.md; the documcp text block matches the contract's example; two runs give byte-identical JSON (FR-006)
- [ ] T007 [P] [US1] Rule tests in `internal/diagnose/rules_test.go`: deadline with no trigger and allowance < 5 → `raise_budget`; allowance unknown → `retry`; unknown reason → `back_to_planning` with route `plan_research`; egress refusals alone → no trigger; egress plus `clone_failed` → `environment_need`; lock file inside the allowed paths → no signal

### Implementation for User Story 1

- [ ] T008 [US1] Create `internal/diagnose/signals.go`: parse `proxy-denials.log` (tinyproxy `Proxying refused on filtered domain "<host>"`; lines prefixed `[<ns>]` are the target proxy, others the model proxy), dropping the probes (`github.com` via the model proxy, P06; `example.invalid` via the target proxy, P06r; research R5); changed paths outside the allowed paths with `images.IsManifest` for the toolchain (R6); research tasks from the task-list text with their line numbers; the reason against `Known`
- [ ] T009 [US1] Create `internal/diagnose/diagnose.go`: types `Signal{Kind, Detail, Source, Trigger}` and `Diagnosis{Version, Task, Attempt, Ended, Usage, Signals, Triggers, Recommend, Routes, Text}` (data-model.md); `Diagnose(in Input) Diagnosis` with the rule "first match wins: any trigger → back_to_planning; deadline or spend_cap with allowance known and below 5 → raise_budget; otherwise retry"; `Render` producing the text block in contracts/diagnosis.md; no clock, network or model
- [ ] T010 [US1] In `internal/pipeline/pipeline.go` `runTask`, after the attempt record is saved (~:333) and before the retries-exhausted branch: build the `Input` from the attempt dir, the run's allowed paths, the toolchain and the run baseline's `test_seconds` if recorded; write `attempts/<task>-<n>/diagnosis.json`; add `diagnosis: {recommend, triggers}` to the attempt record; emit event `attempt_diagnosed`
- [ ] T011 [US1] In `confirmRetry` (`internal/pipeline/pipeline.go:184-201`), print the diagnosis text and store it as `run.State["decision_note"]` in place of `deadlineNote`'s line (keep `deadlineNote` only for attempts without a diagnosis, e.g. resumed old runs); the GitHub comment picks it up through `ghloop.decisionNote` unchanged
- [ ] T012 [US1] In `internal/evidence/evidence.go` `WriteSummary`, add a `## Diagnosis` section after the Attempts table, only when an attempt has `diagnosis.json`; confirm `TestFixtureModeAndSummaryUnchanged` still passes
- [ ] T013 [US1] For run-level failures before any attempt (`PREFETCH_FAILED`, `TOOLCHAIN_*` in `internal/pipeline/item.go:200-206`), append the route "back to planning: change it in your planning PR" to the failure message; when retries are exhausted, the summary still carries the last diagnosis

**Checkpoint**: US1 complete; the replay tests pass and the fixture summary is unchanged.

---

## Phase 4: User Story 2 - The owner ends a run to go back to planning (Priority: P1)

**Goal**: `replan` ends a run paused for a retry as `stopped(back-to-planning)`, recorded,
watcher stopped, slot freed, audited as an owner decision.

**Independent Test**: on a test run paused at a retry, record `replan` through the harness
channel and through a GitHub comment vector; the state is final, the watcher stopped, the
evidence records who and how, and a new run starts.

### Tests for User Story 2

- [ ] T014 [P] [US2] Add `replan` cases to `specs/004-github-loop/contracts/vectors/approvals.json` (allowed approver while awaiting retry → accepted; not allowed → ignored; while awaiting mode → refused; `/withdraw replan`) and make `internal/ghloop/vectors_test.go` pass
- [ ] T015 [P] [US2] Pipeline test in `internal/pipeline/replan_test.go`: paused(awaiting-retry) + replan record → `stopped(back-to-planning)`, events `approval_channel step=replan` and `run_stopped`, `run.State["stopped"]` has by, channel, at, task and attempt; no attempt n+1; grading namespace drop called; summary written

### Implementation for User Story 2

- [ ] T016 [US2] `internal/cli/commands.go` and `internal/cli/approve.go`: add `replan` to `approve`'s `ValidArgs` and help; output per contracts/cli.md
- [ ] T017 [US2] `internal/decisions/`: accept a `replan` record while `PendingStep` is `retry` (bound to the task and attempt like a retry); `Hint` for a retry names both `approve … retry` and `approve … replan`
- [ ] T018 [US2] `internal/ghloop/approve.go`: `Evaluate`/`Collect`/`FindApproval` accept `/approve replan` and `/withdraw replan` with the same allow-list and permission checks; `PostDecision` for a retry lists both replies (contracts/cli.md)
- [ ] T019 [US2] `internal/pipeline/pipeline.go` `Resume`, branch `paused(awaiting-retry)`: before `confirmRetry`, take a `replan` record; then `SetRunState("stopped(back-to-planning)", …)`, set `run.State["stopped"]`, `dropGrading`, `WriteSummary`, close the tracking-issue decision, stop and release the watcher, print the contract's message and return exit 0
- [ ] T020 [US2] Watcher (`internal/pipeline/github.go` `ReadyToResume`, `Watch`): a `replan` approval makes the run ready to resume (so it ends), and the loop exits on any `stopped(...)` state
- [ ] T021 [P] [US2] Step mode (`internal/step/manager.go:203-213` `runEnded`, `internal/step/gather.go:136-147` `runState`, `waitingFor`): `stopped(...)` counts as ended; add a test
- [ ] T022 [P] [US2] Audit (`internal/audit/audit.go` `itemAU02`, `itemAU03`; `internal/audit/slices.go` `ItemView`): on `stopped(back-to-planning)`, AU02 and AU03 report `n/a  run ended by the owner for planning` and don't count as failures; `ItemView` prints the state; add a test
- [ ] T023 [P] [US2] `forge status` prints the stopped state with who and the channel (`internal/cli/runs.go`)

**Checkpoint**: US2 complete; a paused run can be ended for planning from the terminal or GitHub.

---

## Phase 5: User Story 3 - Doctor warns before a run (Priority: P2)

**Goal**: doctor warns about research tasks, a runtime mismatch, and a low test-run allowance.

**Independent Test**: doctor on a test target whose slice has a research task and a "Node 22"
task against `node24` prints both warnings and still exits 0; with a recorded `test_seconds`,
the baseline section prints the time and, below 5, the allowance warning.

- [ ] T024 [US3] In `internal/cli/doctor.go` (item section): read the slice's task lines with `ghloop.Checklist` limited to the slice's phases (`readiness.Result.Slice.Phases`; the whole item without slices) and warn per contracts/cli.md for each unchecked task where `diagnose.ResearchTask` matches, with `tasks.md:<line>`
- [ ] T025 [US3] In `internal/cli/doctor.go`: find runtime versions named in the slice's task lines (`Node NN`, `Python 3.NN`, `Go 1.NN`, `JDK NN`/`Java NN`) and warn when the major version differs from the target toolchain's pinned version (`pilot/pins.env`: `NODE24_VERSION`, `PYTHON313_*`, `GO_VERSION`; JDK from the toolchain name)
- [ ] T026 [P] [US3] `pilot/images/tools/grade.py`: time one run of the project test command in G4 and record `test_seconds` (integer) in G4's detail and the report; test in `pilot/images/tests/` (the tools image is rebuilt by the image workflow's PR, C-09)
- [ ] T027 [US3] Doctor baseline section: read `test_seconds` from the baseline grade (`pipeline.ItemBaseline` result), print the time, compute the allowance `floor((worker deadline − 120) / test_seconds)` with the deadline from `images.Resolve(...).Worker.DeadlineS`, and warn below 5; `INFO … not recorded by this grader image` when absent
- [ ] T028 [P] [US3] Tests in `internal/cli/doctor_test.go` (or the existing doctor test file): research-task warning with line, no warning for plain tasks, runtime mismatch warning, allowance warning and the INFO line; every new line is a warning or info and `Failed()` stays false

**Checkpoint**: US3 complete; documcp's doctor shows the warnings (quickstart §5).

---

## Phase 6: User Story 4 - The docs explain the triggers and the route back (Priority: P3)

- [ ] T029 [US4] Rewrite the deadline entry in `docs/troubleshooting.md` (`#deadline-or-spend-cap`): the configurable `FORGE_WORKER_DEADLINE_S`, three causes (slow model; work larger than the budget; unknown work or an environment need) and the choice for each, pointing to the diagnosis; add an entry for "back to planning" (`replan`) and entries for each reason in `diagnose.KnownReasons` the page lacks (`grade_fail`, `no_result`, `bad_frame`, `oversize`, `grader_timeout`, `grader_error`, `no_frame` and the worker report errors)
- [ ] T030 [P] [US4] `docs/targets.md`: section "Research tasks and environment changes" (anchor `research-tasks`): the research wording list, the allowance threshold, where research and environment changes go, a brownfield example (documcp US1: the leak scan moved into research R3), and how to go back to planning from a diagnosis
- [ ] T031 [P] [US4] `docs/greenfield.md`: the same rule for a new project, with a greenfield example (an unknown framework choice resolved in the plan's research before the tasks)
- [ ] T032 [US4] Docs test in `internal/cli/docs_test.go`: every reason in `diagnose.KnownReasons` appears in `docs/troubleshooting.md`; the research phrases listed in `docs/targets.md` equal `diagnose.ResearchPhrases`
- [ ] T033 [P] [US4] `.claude/skills/forge/SKILL.md`: explain the diagnosis at a retry pause, and the `replan` decision (human only; run `approve … replan` only after the human says yes to it; the harness asks)

**Checkpoint**: all stories complete.

---

## Phase 7: Polish & Cross-Cutting Concerns

- [ ] T034 Run quickstart §1–§2 and record the output in `specs/033-research-before-build/evidence/`; run §3–§5 after the tools image with T026 is built
- [ ] T035 [P] Release notes entry in `docs/release-notes.md`
- [ ] T036 `go vet ./... && go test ./...` and `pilot/images/tests/` pass; `spec status` shows no new error diagnostics

---

## Dependencies & Execution Order

- Setup (T001–T002) → Foundational (T003–T005) → US1 → US2 → US3 → US4 → Polish.
- US2 needs US1 only for the retry-comment text (T018 after T011); its state and audit work is
  independent.
- US3's wording check needs only Foundational; T027 needs T026 and a rebuilt tools image for
  the end-to-end check (unit tests use a fake grade).
- US4's docs test (T032) needs T003 and T004.

## Parallel examples

- After Foundational: T006 and T007 (tests) together; then T008 → T009 → T010–T013.
- US2: T014, T015, T021, T022 and T023 in parallel once T016–T019 are agreed.
- US3: T026 (Python) alongside T024–T025 (Go).
- US4: T030, T031 and T033 in parallel.

## Implementation strategy

- **MVP:** Setup + Foundational + US1. The diagnosis alone turns the documcp failure into a
  "back to planning" recommendation, even before `replan` exists (the owner leaves the run
  paused).
- Then US2 (so the paused documcp run can be ended properly), US3, US4.
