# Tasks: Measure a target before building, and review its CI (036)

**Input**: `specs/036-measure-and-ci-review/spec.md`

## Phase 1: User Story 1 - Doctor reviews the target's CI (Priority: P1) 🎯 MVP

- [x] T001 [US1] `internal/ci/workflow.go`: parse workflows (gating triggers, tag-only pushes,
  steps after a failure or allowed to fail dropped, matrices), split scripts, resolve package
  scripts, classify commands and actions, read test flags.
- [x] T002 [US1] `internal/ci/review.go`: findings for workflows, default-branch health, test
  command, checks grading doesn't run, runtime; `ErrorLine` and `StepSection` for logs.
- [x] T003 [US1] `internal/ci/github.go`: the `Reader` interface and its GitHub Actions version
  (workflow files, `package.json`, latest completed push runs per workflow, failed job logs).
- [x] T004 [US1] `internal/cli/doctor_ci.go`: `doctorCI` on every `doctor --target`, after the
  roadmap rows; information for other hosts.
- [x] T005 [P] [US1] Tests: `internal/ci/review_test.go` (documcp's CI as testdata),
  `internal/cli/doctor_ci_test.go` (a fake reader).
- [x] T006 [P] [US1] Docs: targets (doctor's CI review), strategies §3, release notes.
- [x] T007 [US1] Run doctor on documcp and check SC-001.

## Phase 2: User Story 2 - `doctor --measure` (Priority: P2)

- [ ] T008 [US2] Spec and plan the measurements (roadmap 036 outcome), then generate their tasks.
