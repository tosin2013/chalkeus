# Tasks: Working with an agent — pilot skill and harness approvals

**Input**: Design documents from `specs/012-agent-approvals/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: included. SC-002 (refusal cases) and SC-004 (old evidence passes) are tested
requirements.

**Starting point**: the first cut merged in PR #11. Tasks close gaps G1–G6 from plan.md, then
validate.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel (different files, no dependency on incomplete tasks)
- **[Story]**: US1–US4 from spec.md

---

## Phase 1: Setup

- [X] T001 Baseline: run `PYTHONPATH=pilot python3 -m unittest discover -s pilot/tests -t pilot` and `python3 pilot/tools/docs_check.py` on the branch; both must pass before any change (134 tests, 0 problems)
- [X] T002 [P] Create `specs/012-agent-approvals/evidence/README.md`: what goes here (the early agent-driven run `20261114T213046Z` and the validation run, each as `<run-id>/` copied from `pilot/runs/`); the rule "never paste secrets, hostnames or IPs"

---

## Phase 2: Foundational (contracts)

- [X] T003 [P] Add one line under the title of `specs/002-m1-gated-pilot/contracts/cli.md`: "Amended by 012: `pilot approve`, pause-without-terminal and the mode pause; see `specs/012-agent-approvals/contracts/cli-approve.md`." Leave the rest of the 002 contract unchanged (historical record)
- [X] T004 [P] Add one line under the title of `specs/002-m1-gated-pilot/contracts/evidence-record.md`: "Amended by 012: `approval_channel` events and the AU09/AU10 channel detail; see `specs/012-agent-approvals/contracts/audit-channels.md`."

**Checkpoint**: the contracts point to the 012 amendments; docs check passes.

---

## Phase 3: User Story 1 — A human and an agent complete a run together (P1) 🎯 MVP

**Goal**: an agent drives the run, explains each pause, and you approve each decision through the harness prompt.

**Independent test**: quickstart.md §3 in one harness.

- [X] T005 [US1] Review `.claude/skills/pilot/SKILL.md` against FR-011, item by item. It must cover:
  - commands and exit codes;
  - explain each decision, then ask;
  - propose `approve` only after a clear yes;
  - never approve for the human, edit approvals or evidence, merge, or print secrets;
  - stop if the harness wouldn't ask.

  Fix any gap. Also check that the "normal run" steps match `pipeline.py`:
  - mode pause, then A/1, then the retry pause;
  - A/2, publish, then the merge pause;
  - reconcile pause, then B, then the audit.
- [X] T006 [P] [US1] Review `AGENTS.md` against FR-011 (pointer to the skill plus the never-rules); keep it to a few lines
- [X] T007 [P] [US1] In `.claude/skills/pilot/SKILL.md`, state that commands must be run exactly as `pilot/pilot …` from the repo root, never by absolute path or another wrapper, because the harness ask rules match that prefix (research R3)

**Checkpoint**: the skill and AGENTS.md cover every FR-011 point.

---

## Phase 4: User Story 2 — Nothing proceeds without a human (P1)

**Goal**: every one of the 7 refusal cases holds (SC-002); a terminal answer overrides a pending approval (G1).

**Independent test**: `pilot/tests/test_approvals.py` plus quickstart.md §2.

- [X] T008 [US2] Write the failing test first, in `pilot/tests/test_approvals.py` (`TestDecide`): with a terminal (patch `has_tty` → True) and a recorded approval, `decide(..., ask_tty=lambda q: "n")` returns `(False, "tty")` **and** `approvals/pending.json` no longer exists. Replace the existing `test_tty_asks_and_ignores_approval_file`, which asserts the opposite
- [X] T009 [US2] Implement G1 in `pilot/specfoundry_pilot/approvals.py` `decide()`: when a terminal is present, delete the run's `approvals/pending.json` (if any) after asking, whatever the answer, and record nothing else (research R5). Update the module docstring
- [X] T010 [P] [US2] Add a test in `pilot/tests/test_approvals.py` that a status-change approval recorded while the run waits for the merge (no `DECLINED` proposal in `reconcile/A.json`) is refused with exit 4 and creates no `approvals/pending.json` (refusal case 3)
- [X] T011 [P] [US2] Add a test in `pilot/tests/test_approvals.py` for the CLI: `cli.main(["approve", "mode", "--run", "<missing>"])` returns 2 (patch `paths.EVIDENCE_ROOT` to a temp dir)

**Checkpoint**: all refusal cases have a unit test; G1 is closed.

---

## Phase 5: User Story 3 — A reviewer can tell how each decision was made (P2)

**Goal**: channels are reported for new runs; old evidence passes unchanged (SC-003, SC-004).

**Independent test**: `pilot/tests/test_committed_evidence.py`.

- [X] T012 [US3] Create `pilot/tests/test_committed_evidence.py` (G4, research R6):
  - find every `specs/*/evidence/*/run.json` whose `state` is `succeeded`;
  - run `audit.audit_dir(<dir>, final=True)` on each, with no secret files;
  - assert every row has `result == "pass"`, and that AU09's and AU10's `detail` contains `channels=[]`;
  - use `subTest` per run;
  - fail if fewer than 3 records are found, so a broken glob can't pass vacuously.
- [X] T013 [P] [US3] Add a test in `pilot/tests/test_approvals.py` that `audit._channels` ignores `approved=false` events and keeps order for two approved retries (`["tty", "harness"]`)

**Checkpoint**: SC-004 is a standing regression test.

---

## Phase 6: User Story 4 — Operators at a terminal see no change (P3)

**Goal**: the guide stays correct (FR-013).

**Independent test**: read `docs/operator-guide.md` §8 against contracts/cli-approve.md.

- [X] T014 [US4] In `docs/operator-guide.md` §8, replace "so if there's none (for example inside some IDE runners or CI), every answer counts as \"no\"" with: without a terminal (an agent's shell, some IDE runners, CI) the run **pauses** at each decision and nothing proceeds; link [Prompts answer "no"](troubleshooting.md#prompts-answer-no) and mention the `pilot` skill for working with an agent. Keep the rest of §8 unchanged
- [X] T015 [US4] Run `python3 pilot/tools/docs_check.py`; fix any problem T014 introduced

**Checkpoint**: the guide matches the behaviour.

---

## Phase 7: Polish and validation

- [X] T016 Run the full test suite and docs check; all pass. Record the test count in this file's notes
- [X] T017 **Owner action**: copy the early run's record from the validation machine into `specs/012-agent-approvals/evidence/20261114T213046Z/` (G6). Then run `pilot/pilot audit --run 20261114T213046Z`, and record the AU09/AU10 lines in `specs/012-agent-approvals/evidence/README.md`
- [X] T018 **Owner action**: refusal cases on a set-up machine per quickstart.md §2 (SC-002); record each result in `specs/012-agent-approvals/evidence/README.md`
- [X] T019 **Owner action**: agent-driven run in **Claude Code** per quickstart.md §3 (SC-001, SC-005). Record whether the harness asked before each `approve`, and the audit's AU09/AU10 lines. Copy the record to `specs/012-agent-approvals/evidence/<run-id>/` — done as a harness check on 2026-11-15 (ask prompt in default mode, deny rule, auto-mode finding), see evidence README; SC-001 is covered by the OpenCode runs `20261114T225256Z` and `20261115T020750Z`.
- [X] T020 **Owner action**: the same in **OpenCode** (SC-005). If either harness doesn't ask, fix `.claude/settings.json` or `opencode.json` (G5), record the working syntax in research.md R3, and repeat
- [X] T021 **Owner action**: one terminal-driven run per quickstart.md §4 (SC-006); AU09/AU10 show `channels=['tty']`
- [X] T022 Privacy check: `grep -rniF '<model host>' specs/012-agent-approvals/` returns nothing, and AU08 passes for every committed 012 run
- [X] T023 Add an Evidence section to `specs/012-agent-approvals/spec.md` (runs, harnesses, channels, refusal results). Then `/speckit-roadmap-debrief`, and with the owner's approval `/speckit-roadmap-write`: 012 spec dir and status

---

## Dependencies & Execution Order

- **Setup (T001–T002)** → **Foundational (T003–T004)** → stories.
- **US2 (T008–T011)** should land before the owner validation (T018–T020), because G1 changes behaviour.
- **US1 (T005–T007)**, **US3 (T012–T013)** and **US4 (T014–T015)** are independent of each other and of US2.
- **Validation:** T016 → T017 (any time) → T018 → T019 → T020 → T021 → T022 → T023.

### Parallel Opportunities

- T002, T003 and T004 together.
- T006, T007 alongside T005 (T007 edits the same skill file as T005, so run it after T005 if done by one person).
- T010, T011 and T013 together, after T008–T009 (same test file: sequence the edits).
- T012 and T014 together (different files).

## Implementation Strategy

### MVP

T001–T011: the contracts, the skill review and G1. That makes the feature safe to validate
with an agent (US1 and US2).

### Then

US3 (G4) and US4 (G2) are small and independent. Then the owner validation (T017–T021), the
evidence and the debrief.

### Owner actions (not automatable)

T017–T021 need the validation machine, a real cluster run, both harnesses and a GitHub merge.
The agent must not answer approvals on the owner's behalf in T019 and T020; the point of those
runs is that the owner does.

## Notes

- Commit after each phase; never commit secrets, hostnames or IPs.
- `[P]` tasks touching the same file must still be applied one after another.
- T016 (2026-11-14): 138 tests, all OK; docs_check 0 problems. `test_committed_evidence`
  audited 002 `20261113T030655Z`, 003 `20261113T180203Z` and 006 `20261114T160532Z`: 11/11
  each, AU09/AU10 `channels=[]`.
- T022 partial (2026-11-14): no model-host matches in `specs/012-agent-approvals/`,
  `.claude/`, `AGENTS.md`, `opencode.json` or `docs/`. AU08 for 012's runs waits for their
  evidence (T017, T019–T021).
