# Tasks: Step mode, the laptop manager loop

**Input**: `specs/021-step-mode/` (spec.md, plan.md, research.md, contracts/step.md,
quickstart.md)

**Tests**: included. Each code task has unit tests with fakes. One live run closes the feature
(SC-001).

## Phase 1: Setup

- [X] T001 Add profile keys `PILOT_PROPOSE_LABEL` (default "pilot/go") and `PILOT_STEP_INTERVAL_S` (default 120, at least 30) to `config.Target` in `internal/config/targets.go`, with tests; document them in `specs/014-any-project/contracts/target-profile.md`

## Phase 2: Foundational

- [X] T002 Implement `ghloop.ProposedBy(ctx, repo, issue, label)` in `internal/ghloop/labels.go`: whether the label is on the issue now, and the login of its last `labeled` event (issue events API, all pages); tests with the fake GitHub server
- [X] T003 [P] Add `WatchEnv.Only` to `internal/author/watch.go` so a pass drafts only that item; test
- [X] T004 [P] Record `run_started_by <who>` from `PILOT_STARTED_BY` in `RunItem` (`internal/pipeline/item.go`), with the value checked against a fixed list (`step`); the audit is unchanged; test
- [X] T005 Write `step.Next` in `internal/step/next.go`: `ItemState` and `Action` per contracts/step.md's decision table (FR-004); a table test covering every row, including "another item in flight" and the restart cases of SC-003

## Phase 3: User Story 1 — a proposed item is carried to verified (P1) 🎯 MVP

- [X] T006 [US1] Write the gatherer in `internal/step/gather.go`: candidate items from `spec status` (active, startable, dependencies satisfied, in roadmap order); for each, the proposal (T002), readiness with `SkipHidden` (hidden files), open `author/<NNN>-` PRs, the newest item run for target and item (from the evidence root and the decisions log's run folders) with its state, and the target and companion commits; tests with fakes
- [X] T007 [US1] Write the manager in `internal/step/manager.go`:
  - each pass: sync item issues (004), gather, then act on `Next` for the first item in flight;
  - **draft:** `author.Pass` with `Only`;
  - **doctor:** run the pilot binary's `doctor --baseline` and keep its log; on failure, comment the FAIL and WARN lines;
  - **run:** the pilot binary's `run`, with `PILOT_STARTED_BY=step`;
  - append each action to `step/<target>/decisions.jsonl`;
  - post each comment once, with its marker (research R4, R5).

  Tests with a fake subprocess runner and a fake GitHub writer.
- [X] T008 [US1] Add `pilot step --target <name> [--once]` in `internal/cli/step.go`, per contracts/step.md: the refusals, polling, `-vv` by default, one line per decision, and exit codes; CLI test

## Phase 4: User Story 2 — only an approver's label proposes work (P1)

- [X] T009 [US2] Wire `ProposedBy` into the gatherer: proposed only if the label is present and its last adder is in `PILOT_APPROVERS`; print the reason once per item when it isn't; tests for a non-approver, an approver, a non-approver re-adding after an approver, and removal before the run (SC-002)

## Phase 5: User Story 3 — stop, restart, failures (P2)

- [ ] T010 [US3] Make doctor and run failures stop the item: post one comment, and retry doctor only when the target's or the companion's commit changes; a failed run waits for a relabel (a newer `labeled` event); on SIGINT or SIGTERM finish the current subprocess or cancel drafting, record `step_stopped`, and exit; tests (SC-003, SC-004)

## Phase 6: Polish and validation

- [ ] T011 [P] Docs:
  - `docs/targets.md`, a new section "Step mode: leave it running": setup, the label, what it does and never does;
  - `.claude/skills/pilot/SKILL.md`: the human starts `pilot step` in their own terminal;
  - `docs/troubleshooting.md`: the refusals and the comments it posts.

  Run the docs check.
- [ ] T012 Live validation with the owner (quickstart): Stockroom 009, labelled, carried to `verified` with only GitHub answers; after `down`, audit 11/11; `decisions.jsonl` recorded. Record it in `specs/021-step-mode/evidence/README.md` (SC-001)
- [ ] T013 Privacy check of the new records, then the 021 debrief

## Dependencies

- T001 → T002–T005 → US1 (T006–T008) → US2 (T009) → US3 (T010) → T011–T013.
- T003 and T004 can run in parallel with T002 and T005.
- **MVP:** T001–T009 (an approver's label carries an item to verified).
