# Tasks: Test-author automation

**Input**: `specs/024-test-author-automation/` (spec.md, plan.md, research.md,
contracts/author-watch.md, quickstart.md)

**Tests**: included. Each code task has unit tests, written as fakes the way
`internal/author/flow_test.go` does. One live run closes the feature (US1, SC-001, SC-005).

## Phase 1: Setup

- [ ] T001 Add profile keys `PILOT_AUTHOR_AUTO` (`on`/`off`, default `off`), `PILOT_AUTHOR_INTERVAL_S` (default 120) and `PILOT_AUTHOR_AUTO_ATTEMPTS` (default 2, 1–3) to `config.Target` in `internal/config/targets.go`, with validation tests in `internal/config/targets_test.go`; document them in `specs/014-any-project/contracts/target-profile.md`

## Phase 2: Foundational (blocks every story)

- [ ] T002 Add the author proxy `pilot-egress-a` (clients `specfoundry.io/role: author`, allowlist exactly the model hosts) to `cluster.ApplyAll` in `internal/cluster/infra.go`, recreated when its hosts change; tests in `internal/cluster/cluster_test.go`
- [ ] T003 [P] Add `pilot/manifests/author-ns.yaml.tmpl`: role `author`, default deny, egress only DNS and `pilot-egress-a`, quota pods=1 secrets=1, service account without token; render test in `internal/cluster/cluster_test.go`
- [ ] T004 [P] Add `pilot/manifests/author-job.yaml.tmpl`: the worker image with `meta.json` role `author`, the model-key secret, the deadline and resources of a worker, non-root and read-only root; render test
- [ ] T005 Add the `author` role to `pilot/images/worker/entrypoint.sh`: unpack `/in/workspace.tar` into `/work/task`, run OpenCode with `/in/prompt.md`, emit `report.json` (with `role`, `draft_id`, redacted tail) and `draft.tar` holding only `draft/`; Python tests in `pilot/images/tests/` (a fake `opencode` writing a draft file)
- [ ] T006 [P] Add the author prompt in `internal/prompt/author.go`, built from the test-author skill's rules (research R4), with a feedback section for a retry (code plus the drafted tests' base output); a test in `internal/prompt/author_test.go` that each rule from `.claude/skills/test-author/SKILL.md` §Steps 3–4 is present

## Phase 3: User Story 2 — the unattended author is sandboxed (P1)

**Goal**: an author Job that can only reach the model and holds no credential.
**Independent test**: one Job on kind; A01–A05 pass; the draft arrives through the frame only.

- [ ] T007 [US2] Implement `author.RunJob` in `internal/author/job.go`: create the namespace (T003), stage `workspace.tar` and `prompt.md` the way `worker` stages inputs, run the Job (T004), collect and decode the frame, unpack `draft.tar` into the draft's host workspace (only under `draft/`, no absolute or `..` paths), delete the namespace; record per-Job spend, exit and error in `draft.json` `jobs`; tests with a fake runner in `internal/author/job_test.go`, including a frame with a path escaping `draft/`
- [ ] T008 [US2] Add checks A01–A05 (research R6) in `internal/checks/author.go`, run against the author namespace from a probe twin and written to the draft's `checks.jsonl`; pure-rule tests in `internal/checks/author_test.go`
- [ ] T009 [US2] Make `pilot down` delete leftover `pilot-a-*` namespaces and `pilot-egress-a`, and write teardown row T06, in `internal/cluster/`; test

## Phase 4: User Story 1 — a ready item gets a draft on its own (P1) 🎯 MVP

**Goal**: an item that becomes ready gets a checked companion PR with no operator step.
**Independent test**: `pilot author watch --target <t> --once` with a ready item opens a PR whose check passed.

- [ ] T010 [US1] Implement eligibility (research R2) in `internal/author/watch.go`: active, non-verified items with satisfied dependencies, in roadmap order; author readiness `allow`, `hidden_files` 0, no open `author/<NNN>-` PR, no earlier automatic draft for the same spec SHA-256; tests in `internal/author/watch_test.go` for each skip reason
- [ ] T011 [US1] Implement the draft loop in `internal/author/watch.go`: `Start` (with `trigger: auto`), `RunJob`, `Check`; on `DRAFT_HIDDEN_PASSES` or `DRAFT_SUITE_FAILS` and attempts left, a second `RunJob` with the feedback prompt; on `DRAFT_OK`, `Submit`; otherwise record the failure (FR-007); tests for pass, retry-then-pass, fail-after-attempts and the no-PR-without-pass rule (SC-003)
- [ ] T012 [US1] Add `pilot author watch --target <name> [--once]` in `internal/cli/author.go` per `contracts/author-watch.md` (refusals, polling, one line per decision, exit codes); CLI tests

## Phase 5: User Story 3 — the operator stays in control (P2)

**Goal**: off by default, visible, stoppable.
**Independent test**: with `PILOT_AUTHOR_AUTO` unset nothing starts; stopping the watcher mid-draft opens no PR and leaves no namespace.

- [ ] T013 [US3] Refuse `author watch` unless `PILOT_AUTHOR_AUTO=on`, and make `pilot doctor` show `auto-draft: on|off` per target, in `internal/cli/author.go` and `internal/cli/doctor.go`; tests (SC-004)
- [ ] T014 [US3] On interrupt (SIGINT, SIGTERM), cancel the running Job, delete its namespace, record `watch_stopped` in the draft, and exit without submitting; test with a cancelled context in `internal/author/watch_test.go`

## Phase 6: Polish and validation

- [ ] T015 [P] Docs:
  - `docs/targets.md`: a new section "Drafting hidden tests automatically" (keys, the watcher, what it never does);
  - `.claude/skills/pilot/SKILL.md`: the driving session never starts or reads `author watch`;
  - `.claude/skills/test-author/SKILL.md`: a note that the same rules drive the automatic author;
  - `docs/troubleshooting.md`: entries for the watcher's refusals.

  Run the docs check.
- [ ] T016 Live validation (quickstart), with the owner: Stockroom with auto-draft on; plan a small item; the watcher opens a checked companion PR with A01–A05 passing; the owner merges it; the item runs to `verified`; after `down`, audit 11/11. Record it in `specs/024-test-author-automation/evidence/README.md` (SC-001, SC-002, SC-005)
- [ ] T017 Privacy check of the new records (model host, keys, token patterns, no hidden-test content outside the companion PR), and the 024 debrief

## Dependencies

- T001 → T002–T006 → US2 (T007–T009) → US1 (T010–T012) → US3 (T013–T014) → T015–T017.
- T003, T004 and T006 can run in parallel once T001–T002 are done.
- **MVP:** US2 plus US1 (T001–T012). An item gets a checked PR from a sandboxed author.
