# Tasks: Resource budgets and capacity checks

**Input**: `specs/030-resource-budgets/` (spec, plan, research R1–R7, data-model, contracts, quickstart)
**Tests**: included. The project checks every change with `go vet ./... && go test ./...` and
`python3 -m unittest discover pilot/images/tests` (plan "Testing"); each phase adds its tests first.

Phases run in order; each is one slice. Phase 2 is the MVP that unblocks documcp's US1
(quickstart §1, §4).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel (different files, no dependency on an unfinished task)
- **[Story]**: the user story from spec.md

---

## Phase 1: Foundational — the budget type and resolver (blocks every story)

**Purpose**: one place that turns the catalog plus the profile into a per-role budget.

- [ ] T001 Add `worker` and `author` blocks (`cpu`, `memory`, `deadline_s`) to the toolchain catalog type and to every entry in `pilot/images/toolchains.json` and `operatorDefaults` in `internal/images/toolchains.go`, with today's values: worker 2 / 2Gi / 1200, author 2 / 2Gi / 1200 (author uses `worker.Deadline` today, `internal/cli/author.go:219`) (worker fields done in Phase 2, defaults in `images.DefaultWorker`; author block with Phase 3)
- [ ] T002 Create `internal/budget/budget.go`: `type Role struct{CPU, Memory string; DeadlineS, CommandTimeoutS, SoftMarginS int}`, `type Budget struct{Worker, Grader, Prefetch, Author Role; Raised []string; Source string}` matching data-model.md ("prefetch always equals grader's CPU, memory and job deadline"; `soft_margin_s` fixed at 120) (`images.Budget` with the worker role done in Phase 2, in `internal/images/budget.go` to avoid an import cycle; grader, prefetch and author roles with Phase 3)
- [ ] T003 Implement `budget.Resolve(tc images.Toolchain, t config.Target) (Budget, error)` in `internal/budget/budget.go`: catalog values, raised by the profile; refuse a value below the floor with `TARGET_INVALID: <KEY> <value> is below the catalog's <floor> for <toolchain>`; refuse `FORGE_COMMAND_TIMEOUT_S` > resolved grader deadline; refuse `FORGE_WORKER_DEADLINE_S` ≤ 180 (contracts/profile-keys.md) (worker part done: `images.Resolve`, floors in `images.CheckTarget`; grader and author with Phase 3)
- [ ] T004 [P] Unit tests in `internal/budget/budget_test.go`: defaults equal today's values; each raise applied and listed in `Raised`; below-floor, malformed and timeout>deadline refused; operator-supplied toolchain floors; prefetch mirrors grader (worker part done in `internal/images/budget_test.go`; the rest with Phase 3)
- [x] T005 Write the resolved budget to `run.json` (`budget`) at run start, before the mode approval, and read it back on resume instead of re-resolving (FR-006), in `internal/pipeline/item.go` and `internal/evidence/`; emit event `budget_resolved` (data-model.md "Events")
- [x] T006 [P] Test in `internal/pipeline/item_test.go`: a resumed run whose profile lost its raises keeps the budget from `run.json`

**Checkpoint**: `go test ./internal/budget/ ./internal/pipeline/` green; behaviour unchanged without keys.

---

## Phase 2: User Story 6 + worker budget — attempts that run out of time keep their work (Priority: P1) 🎯 MVP

**Goal**: a deadline-stopped attempt keeps its change, log and usage; the worker's deadline, CPU
and memory can be raised.

**Independent Test**: quickstart §1 — an attempt with `FORGE_WORKER_DEADLINE_S=300` and longer
work fails `deadline` with `report.json` (`stopped_by: "deadline"`, non-null `spend`) and the
run's Cost/Tokens filled.

### Tests for this phase

- [x] T007 [P] [US6] Python test `pilot/images/tests/test_watchdog_deadline.py`: with `soft_deadline_s` in the past, `watchdog.py` SIGTERMs the harness pid, SIGKILLs it after 15 s if still alive, and writes `/tmp/spend.json` with `"stopped_by_deadline": true` (contracts/worker-soft-deadline.md)
- [x] T008 [P] [US6] Go test in `internal/worker/worker_test.go`: a fake runner returning `Complete` with a frame whose `report.json` has `error: "deadline"` → attempt `failed/deadline` with `spend`, `changed_paths`, `report.json` and `change.bundle` kept; a hard `DeadlineExceeded` without a frame → `failed/deadline`, `spend: null`, note `no_frame`
- [x] T009 [P] [US1] Go test in `internal/worker/worker_test.go`: the rendered worker Job carries the budget's CPU, memory and `activeDeadlineSeconds`, and `meta.json` has `soft_deadline_s = deadline_s − 120`

### Implementation

- [x] T010 [US6] `pilot/images/worker/watchdog.py`: read `soft_deadline_s` from `/in/meta.json` (absent → no soft deadline); stop the harness when it passes (SIGTERM, then SIGKILL after 15 s); add `stopped_by_deadline` to `/tmp/spend.json`
- [x] T011 [US6] `pilot/images/worker/entrypoint.sh`: after the harness exits, set `ERROR="deadline"` when `stopped_by_deadline`; keep the normal copy-back, commit, bundle and `finish()`; add `"stopped_by": "deadline" | "spend_cap" | null` to `report.json`
- [x] T012 [US1] Add profile keys `FORGE_WORKER_CPU`, `FORGE_WORKER_MEMORY`, `FORGE_WORKER_DEADLINE_S` (with `PILOT_*` fallbacks and the 027 deprecation warning) to `internal/config/targets.go`, validated as CPU quantity / memory quantity / whole seconds
- [x] T013 [US1] Replace `const Deadline` uses in `internal/worker/worker.go` (Job `DEADLINE`, `Runner.Wait` timeout) and `internal/pipeline/pipeline.go:111` (`worker_deadline_s`) with the resolved worker budget; pass `soft_deadline_s` in `meta.json`
- [x] T014 [US1] Template the worker's limits: `cpu: "${CPU}"`, `memory: ${MEMORY}` in `pilot/manifests/worker-job.yaml.tmpl` and `pilot/manifests/worker-job-item.yaml.tmpl`, filled from the budget in `internal/worker/worker.go`
- [x] T015 [US6] `internal/worker/worker.go` `RunAttempt`: when `report.error == "deadline"`, record `failed/deadline` but keep the decoded files and `att["spend"]`; for a hard deadline with no frame record note `no_frame`; never grade or publish either (FR-017)
- [x] T016 [US6] Count a deadline-stopped attempt's spend toward the run's spend and token caps (wherever `spend_cap` attempts are summed, `internal/pipeline/`), and state in the retry decision text (terminal and GitHub): `stopped at the deadline after <s> s; <tokens> tokens, $<cost>; changed: <n> file(s)`
- [x] T017 [US6] Rebuild path: confirm the worker image change goes through the image workflow (C-09) and that an older worker image (no `soft_deadline_s` support) keeps today's behaviour; note it in `pilot/images/worker/` header comments

**Checkpoint**: quickstart §1 passes on the 025 harness; documcp can run US1 with `FORGE_WORKER_DEADLINE_S=2400` (quickstart §4).

---

## Phase 3: User Story 1 — the rest of the raise-only budget (Priority: P1)

**Goal**: grader CPU, per-command timeout and author budget are raisable; prefetch follows the grader.

**Independent Test**: quickstart §2 — raised keys appear in the Jobs' manifests and `run.json`;
removing them returns new runs to the floors.

- [ ] T018 [P] [US1] Go tests in `internal/pipeline/item_test.go` and `internal/author/job_test.go`: grader and prefetch Jobs carry the raised CPU; the grader's `command_timeout_s` and the worker prompt's stated timeout equal the resolved value; the author Job carries the author budget
- [ ] T019 [US1] Add keys `FORGE_GRADER_CPU`, `FORGE_COMMAND_TIMEOUT_S`, `FORGE_AUTHOR_CPU`, `FORGE_AUTHOR_MEMORY`, `FORGE_AUTHOR_DEADLINE_S` in `internal/config/targets.go` (existing `FORGE_GRADER_MEMORY` and `FORGE_GRADER_DEADLINE_S` move to the resolver)
- [ ] T020 [US1] Use the resolved grader budget for grader and prefetch Jobs and `commandTimeout()` in `internal/pipeline/item.go` (replacing lines 486–526's catalog-plus-override logic)
- [ ] T021 [US1] Template the author Job's limits (`pilot/manifests/author-job.yaml.tmpl`) and take its deadline from the author budget in `internal/author/job.go` and `internal/cli/author.go`; draft checks (`author check`) use the grader budget
- [ ] T022 [US1] Record the budget in the draft's `draft.json` (`internal/author/`), resolved at `author check` start
- [ ] T023 [P] [US1] `./chalkeus forge targets` prints each target's raised keys (`internal/cli/targets.go`)

**Checkpoint**: quickstart §2 passes.

---

## Phase 4: User Story 2 — capacity check before a run (Priority: P1)

**Goal**: doctor, `run` and `author check` say whether the cluster can give the largest role its limits.

**Independent Test**: quickstart §3.1 — `FORGE_GRADER_CPU=64` on kind fails doctor with the local
message; `forge run` refuses to start.

- [ ] T024 [P] [US2] Go tests in `internal/doctor/capacity_test.go` with fake node/pod/quota/limitrange lists: pass with headroom; local fail message; remote fail message; quota fail naming the quota and namespace; unreadable nodes → warn (contracts/capacity.md, verbatim message shapes)
- [ ] T025 [US2] Implement `capacity.Check(c *k8s.Client, kubeContext string, b budget.Budget) Result` in `internal/doctor/capacity.go`: largest role by limits vs the largest schedulable node's allocatable minus requests of Chalkeus's running pods; `ResourceQuota` `limits.cpu`/`limits.memory` and `LimitRange` max where readable; local when the context is `paths.Context`, else remote (research R3)
- [ ] T026 [US2] Add the `== capacity` area to `internal/doctor/doctor.go` for `--target`
- [ ] T027 [US2] Call `capacity.Check` before staging in `forge run` (`internal/pipeline/`) and `forge author check` (`internal/author/check.go`); a fail refuses with exit 2 and `CAPACITY_INSUFFICIENT`; record `capacity` in `run.json`/`draft.json` and event `capacity_checked`

**Checkpoint**: quickstart §3.1 passes.

---

## Phase 5: User Story 3 — unschedulable pods stop the run (Priority: P2)

**Goal**: a pod that can't be scheduled for lack of CPU or memory stops the run within 2 minutes, without counting an attempt.

**Independent Test**: quickstart §3.2.

- [ ] T028 [P] [US3] Go test in `internal/runner/runner_test.go`: a pod with `PodScheduled=False`, reason `Unschedulable`, message containing `Insufficient cpu` for over 120 s → `Wait` returns `Unschedulable`; a pod scheduled after 30 s → normal outcome
- [ ] T029 [US3] `internal/runner/runner.go` `Wait`: detect the unschedulable pod (120 s wait limit, not a profile key) and return `Unschedulable`
- [ ] T030 [US3] Handle `Unschedulable` in `internal/worker/worker.go`, `internal/grader/`, the prefetch and `internal/author/`: delete the namespace, emit `capacity_stop` (role, pod reason, waited seconds), stop the run with `CAPACITY_UNSCHEDULABLE` and the contracts/capacity.md message; do not advance the attempt counter or record spend

**Checkpoint**: quickstart §3.2 passes.

---

## Phase 6: User Story 4 — budget visible and recorded (Priority: P2)

**Goal**: the owner sees the budget when approving the mode; the audit checks it.

**Independent Test**: approving a raised-budget run shows each role with raised values marked; `run.json.budget` matches.

- [ ] T031 [P] [US4] Go tests in `internal/pipeline/pipeline_test.go` and `internal/audit/audit_test.go`: the mode text lists every role's CPU, memory and times and marks raised keys; "catalog budget" when none; the audit fails when `run.json.budget` differs from the approved mode's, and skips evidence without `budget`
- [ ] T032 [US4] Mode summary in `internal/pipeline/pipeline.go` gains `budget`; terminal mode text and the GitHub decision comment (`internal/ghloop/`) print it
- [ ] T033 [US4] New audit check in `internal/audit/audit.go` comparing `run.json.budget` with the approved mode (runs recorded after this feature only, FR-022-style compatibility)

**Checkpoint**: SC-005 holds on a fixture run.

---

## Phase 7: User Story 5 — sizing docs (Priority: P3)

**Goal**: one docs section on sizing a target.

**Independent Test**: `docs/targets.md` has the catalog table, every key and floor, the capacity messages, and slow-vs-never-exits with the documcp example; the docs check passes.

- [ ] T034 [US5] Add "Sizing a target" to `docs/targets.md`: catalog budgets per toolchain and role, each key with its floor (contracts/profile-keys.md), the capacity messages (contracts/capacity.md), deadline stops and what their evidence holds, and how to tell a slow test command from one that never exits (Jest in band on 2 CPUs; `--forceExit`; documcp)
- [ ] T035 [P] [US5] Add `CAPACITY_INSUFFICIENT`, `CAPACITY_UNSCHEDULABLE` and the deadline-stop retry text to `docs/troubleshooting.md`

---

## Phase 8: Polish

- [ ] T036 Run `go vet ./... && go test ./...` and `python3 -m unittest discover pilot/images/tests`; fix what they find
- [ ] T037 Run quickstart §1–§3 on the 025 integration harness and §4 live on documcp; record the evidence under `specs/030-resource-budgets/evidence/`

---

## Dependencies & Execution Order

- Phase 1 blocks all others.
- Phase 2 (MVP) needs Phase 1 only. Phase 3 needs Phase 1 (and reuses Phase 2's template work).
- Phase 4 needs Phase 1 (the budget) and is independent of Phases 2–3.
- Phase 5 is independent of Phase 4 but shares its message text (contracts/capacity.md).
- Phase 6 needs Phases 1–3 (all roles resolved). Phase 7 after Phases 2–5. Phase 8 last.

## Parallel Opportunities

- Within Phase 1: T004 and T006 alongside each other once T002–T003 and T005 exist.
- Within Phase 2: T007, T008, T009 (tests) together; then T010–T011 (image) in parallel with T012–T014 (Go).
- Phases 4 and 5 can proceed in parallel after Phase 1.

## Implementation Strategy

- **MVP**: Phases 1–2. Merge, let the image workflow publish the new worker image, then set
  `FORGE_WORKER_DEADLINE_S=2400` for documcp and start a new US1 run (026 T048).
- Then Phases 3–7 as separate PRs, in order.
