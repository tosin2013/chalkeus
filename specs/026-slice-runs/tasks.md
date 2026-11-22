---
description: "Task list for 026 Slice runs"
---

# Tasks: Slice runs

**Input**: Design documents from `specs/026-slice-runs/`: plan.md, spec.md, research.md (R1–R10),
data-model.md, contracts/ (cli, slices, grading, audit), quickstart.md.

**Tests**: included. The plan's Testing section asks for:
- unit tests for every new rule, against fake spec-status JSON;
- grader tests in `pilot/images/tests/`;
- the committed (golden) evidence auditing unchanged;
- a two-slice scenario in 025's integration harness.

Tests are written with each implementation task, in the same package.

**Organization**: by user story, so 026 can itself be built one slice at a time:
- US1 (whole item unchanged, P1) puts the guards first;
- US3 (a slice stays in bounds, P1) comes before US2 (build and merge slice by slice, P1),
  because a slice run can't pass grading until G2 and G3 know about slices;
- US4 (audit, P2) and US5 (step mode and the test author, P2) build on US2.

**Owner actions** are marked **Owner**. The agent never approves, posts `/approve`, merges, or
labels on the owner's behalf (AGENTS.md).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel (different files, no dependency on an unfinished task).
- **[Story]**: US1–US5 from spec.md.

---

## Phase 1: Setup

- [X] T001 **Owner** Merge PR #154 (spec-status 0.4.0 pin: `pilot/pins.env`, operator guide, docs check). Then confirm on `main` that `.github/workflows/go.yml` installs 0.4.0, and that `spec status --json --root .` gives item 014 seven `tasks.phases` (research R1).
- [ ] T002 **Owner** Merge the image-digest PR the `pilot-images` workflow opens for the 0.4.0 pin, so the tools image's grader has `tasks.phases` (C-09). Live runs (T047, T048) need it.
- [X] T003 [P] Create `internal/slice/testdata/` with spec-status JSON fixtures: `phases-014.json` (014's 7 phases, copied from real 0.4.0 output), `phases-none.json` (`"phases": []`), `phases-null.json` (`"phases": null`), `phases-absent.json` (0.3.0 shape, no field), and `phases-dup-story.json` (two phases with `story: 2`).

---

## Phase 2: Foundational (blocks every story)

- [X] T004 Create `internal/slice/slice.go` with `Phase{Heading string; Line int; Story *int; Checked, Total int}` and `Slice{ID string; Index int; Phases []string; Checked, Total int}`, and `FromPhases([]Phase) ([]Slice, error)`, following contracts/slices.md "Deriving slices":
  - a story phase starts `US<n>`;
  - leading non-story phases join the first story's slice;
  - any other non-story phase joins the slice before it;
  - with no story phases, one slice `ALL`;
  - a duplicate story number is an error naming both headings.
- [X] T005 Add `Parse(statusItem map[string]any) (phases []Phase, ok bool)` to `internal/slice/slice.go`: `ok=false` when `tasks.phases` is absent or `null`, and `[]` gives one `ALL` slice (research R1). Add `State(s Slice) string` returning `merged`/`untouched`/`started` from the counts (data-model.md; the open-PR half of "merged" is applied by readiness), `Current([]Slice) (Slice, bool)` for the first non-merged slice, and `Find(slices, id string)`, which takes `us2` as `US2` and refuses `ALL` (data-model.md "Validation").
- [X] T006 [P] Write `internal/slice/slice_test.go`, a table test over the T003 fixtures. Assert:
  - the 014 mapping equals the table in contracts/slices.md (US2: Setup, Foundational, Story 2; US1; US3; US4: Story 4, Polish), with order US2, US1, US3, US4;
  - the error case for a duplicate story;
  - `null` and absent both give `ok=false`.
- [X] T007 Extend `internal/evidence/evidence.go`: `(*Run).SetSlice(slice string)` writes `slice` into `run.json` only when non-empty, and `(*Run).Slice() string` reads it (`SetItemRun` keeps its signature, so its two callers are unchanged). Add a test that a whole-item run's `run.json` has no `slice` key (FR-022).
- [X] T008 [P] Add `CompanionSliceDir(t Target, item, slice string) string` to `internal/config/targets.go` (`[<PILOT_COMPANION_PATH>/]items/NNN/<slice>`), plus `SliceLayout(entries []string, sliceIDs []string) (perSlice bool, stray []string)`, which implements contracts/slices.md "Companion layout": per-slice only when every top-level entry is a folder named after a slice id. Add tests in `internal/config/targets_test.go`.
- [X] T009 Change `internal/grader/companion.go` `Overlay` to take an ordered list of source folders and produce one deterministic, sorted tar, still excluding `.acceptance-map.json`. With one folder, the bytes must match today's output (test with a fixed folder in `internal/grader/companion_test.go`).

**Checkpoint:** `go test ./internal/slice/ ./internal/evidence/ ./internal/config/ ./internal/grader/` passes, and nothing user-visible has changed.

---

## Phase 3: User Story 1 - An item without slices runs exactly as today (Priority: P1)

**Goal:** guards that prove the default path is untouched while the rest is built.

**Independent Test:** the committed evidence audits with the same results, and the fixture run passes its full audit, with no `--slice` (SC-001).

- [X] T010 [US1] Reuse `TestGolden` (`internal/audit/golden_test.go`, `specs/016-go-pilot/golden/audit-expected.json`) as FR-022's guard. Check that it covers at least one evidence record each from 014, 021 and 024. Add any that are missing to `audit-expected.json` with their current rows, **before T007** (before any 026 code lands). No new snapshot file.
- [X] T011 [P] [US1] Add a whole-item case to `pilot/images/tests/test_grade_slices.py`: a meta with no `slice` key produces exactly the checks G1–G5 (plus G6, G7 when test paths are set), with no G8, and G3's detail is `tasks percent=…` as today.
- [X] T012 [US1] In `internal/readiness/readiness_test.go`, add a case: an item whose spec-status output has no `tasks.phases` field (0.3.0 shape), run without a slice, is allowed exactly as before. The result has no slice rules (`SLICE_*` absent).

**Checkpoint:** US1's guards are green. Each later phase must keep them green.

---

## Phase 4: User Story 3 - A slice stays inside its own boundaries (Priority: P1)

**Goal:** per-slice allowed paths, and a G3 that accepts only this slice's ticks.

**Independent Test:** slice attempts that tick another slice's task, or edit outside the slice's allowed paths, fail grading (SC-004).

- [X] T013 [P] [US3] In `internal/readiness/paths.go`, read `## Allowed paths (<id>)` (same list shape as `## Allowed paths`). Add `SliceAllowedPaths(spec, specDir, item, sliceID string) (globs []string, source string, err error)`: the slice's list if present, else the item's, with `source` `slice` or `item`. Every existing rule still applies: protected paths give `ITEM_PATHS_TOO_WIDE`, `<specDir>/tasks.md` is always added, and another item's spec is refused. Add tests in `internal/readiness/paths_test.go`.
- [X] T014 [US3] Add `slice_counts(status_json, item_id)` to `pilot/images/tools/grade.py`. It returns `{heading: (checked, total)}` plus the item's out-of-phase checked count (item `checked` minus the sum over phases), or `None` when `tasks.phases` is absent or null.
- [X] T015 [US3] Make G3 slice-aware in `pilot/images/tools/grade.py` when `meta.slice` is set, per contracts/grading.md:
  - run `spec status --json` on the base too (a `git worktree` at `base`);
  - pass only when every heading in `meta.slice.phases` has `checked == total`, and every other heading, plus the out-of-phase count, equals the base;
  - the detail names each offending heading;
  - a heading missing on either side fails with `heading changed: …`.
- [X] T016 [P] [US3] Add slice G2 and G3 cases to `pilot/images/tests/test_grade_slices.py`:
  - ticking all of US1's tasks passes;
  - also ticking a US2 task fails G3;
  - leaving a US1 task unticked fails G3;
  - a renamed phase heading fails G3;
  - a file outside `meta.allowed_paths` fails G2.

  Use small git repos built in a temp folder, and a stub `spec` on `PATH` that prints prepared JSON.

**Checkpoint:** a stub slice change is graded correctly through G1–G3.

---

## Phase 5: User Story 2 - Build and merge an item one slice at a time (Priority: P1) 🎯 MVP

**Goal:** `pilot run --slice`, readiness, grading with earlier slices, one PR per slice, merge without a status change until the last slice, and the optional acceptance decision.

**Independent Test:** on a two-slice item, US2 is refused before US1 merges; after both merges, exactly one `verified` proposal exists (SC-002, SC-003, SC-005).

Readiness and CLI:
- [X] T017 [US2] Parse `tasks.phases` into readiness's status item in `internal/readiness/readiness.go` with `slice.Parse` (no gate decision change; `internal/gate/gate.go` stays as is). Add `Request.Slice string`. When it is set, evaluate the contracts/slices.md rules in this order, each failing closed with its reason code:
  - `SLICE_UNEVALUABLE`
  - `SLICE_UNKNOWN`
  - `SLICE_EARLIER_OPEN` (earlier slices not `merged` on the fresh clone)
  - `SLICE_STARTED`

  Also check `SLICE_MAP_CHANGED`: a local slice run of this item recorded different phase headings for a slice that is now merged, or the slice count changed since a slice merged. `SLICE_EARLIER_OPEN`'s detail names the earlier slice ids. Record `readiness.slice = {id, index, of, phases, earlier, allowed_paths_source, default_revision}`, using T013 for the allowed paths.
- [X] T018 [US2] In `internal/readiness/readiness.go`:
  - add `ITEM_PARTLY_SLICED`: no slice named, 2+ slices, and any slice merged or started;
  - widen `OPEN_ITEM_PR` to match heads `^pilot/[^/]+/<item>(-us[0-9]+)?$`, refusing on an open PR for this slice, an earlier slice or the whole item; a whole-item run is refused by any slice PR;
  - make `ITEM_NO_HIDDEN_TESTS` check the slice's companion folder, or every slice folder for a whole-item run of a sliced item, using T008. A mixed layout fails with the stray entries in the detail.
- [X] T019 [P] [US2] Add `internal/readiness/slices_test.go`. Use the T003 fixtures and fake PR heads. It covers:
  - one case per reason code in contracts/slices.md, including `SLICE_MAP_CHANGED`;
  - `SLICE_EARLIER_OPEN`'s detail naming the earlier slice;
  - allow cases for US1 of a fresh item and for US2 after US1 is merged;
  - `--slice` on a verified item gives `ALREADY_VERIFIED`;
  - an item with a merged slice still gates as startable (FR-008, FR-019).
- [X] T020 [US2] Add `--slice` to `runCmd` and `doctor` in `internal/cli/commands.go`. Pass it through `internal/cli/run.go` to `p.RunItem(item, slice)`, and on resume read it from `run.json` (`targetFor`). Refuse `--slice` without `--item`.
- [X] T021 [P] [US2] Add `internal/cli/slices.go`, the `pilot slices --target <t> --item NNN [--json]` command from contracts/cli.md, as a read-only fresh clone plus spec status. Exit 2 on `SLICE_UNEVALUABLE`. Add a test for the text and JSON output against a fixture.

Run, grade, publish:
- [X] T022 [US2] In `internal/pipeline/item.go`, `RunItem(item, slice)` calls `SetItemRun(target, item, slice)`, passes `Slice` to readiness, and sets `run.State["slices_last"]` (this slice is the item's last). `prepareItem` overlays this slice's companion folder (or all slice folders in order for a whole-item run of a sliced item), and records `companion.folders` (data-model.md).
- [X] T023 [US2] In `internal/grader/item.go`, when the run has a slice, write `meta.slice = {id, phases, earlier}` and stage each earlier slice's companion folder as `grading/earlier/<n>/`, using T009 per folder. Never stage anything from `earlier/` into worker inputs (P11). Add a test that the worker input tar has no companion path.
- [X] T023a [US2] In `internal/cli/doctor.go` and the doctor baseline path, when `--slice` is given: run readiness with `Slice`, stage `grading/earlier/<n>/` and this slice's overlay with T023's helper, and report G8 (must pass) and G5 (must fail) on today's default branch. Without `--slice`, nothing changes. Test it in `internal/cli/doctor_test.go` with fakes.
- [X] T024 [US2] Add G8 and the slice baseline to `pilot/images/tools/grade.py`, per contracts/grading.md:
  - add `SLICE_CHECKS = ["G8_earlier_slices_kept"]`, used only when `meta.slice.earlier` is non-empty;
  - in `visible_and_hidden`, overlay `grading/earlier/*` in order, run the tests (G8, `hidden_summary` only), then overlay `grading/overlay` and run again (G5);
  - in `baseline`, G8 must pass and G5 must fail. G8 failing on base gives the failure `EARLIER_SLICE_BROKEN_ON_BASE`.
- [X] T025 [P] [US2] Add G8 cases to `pilot/images/tests/test_grade_slices.py`:
  - an earlier slice's test broken by the change fails G8, and G5 is skipped;
  - all passing passes;
  - the baseline with G8 passing and G5 failing passes;
  - the baseline with G8 failing reports `EARLIER_SLICE_BROKEN_ON_BASE`;
  - no hidden output appears in any check detail.
- [X] T026 [P] [US2] In `internal/prompt/prompt.go` `BuildItem`, when a slice is set, add the paragraph from contracts/audit.md "Prompt" after the task list: the slice id, that only its tasks are to be done and ticked, and its phase headings. Leave the spec and `tasks.md` text unaltered. Add a test that the spec's hash in the prompt equals `meta.spec_sha256` (AU11).
- [X] T027 [P] [US2] In `internal/publisher/publisher.go`, for a slice run, make the head `pilot/<run>/<item>-<lower(slice)>` and `ItemTitle` append ` (<slice>)` (contracts/cli.md). For a slice run that isn't `slices_last`, the body says `Part of #N` in place of `Closes #N` (the body builders near `publisher.go:101` and `:305`, fed by `closes` in `internal/pipeline/github.go`) (FR-016a). The status PR naming is unchanged. Add tests in `internal/publisher/publisher_test.go` and `body_test.go` for both bodies.

Merge and status:
- [X] T028 [US2] Add `CheckSlice(task, item, slice string, pr int, changed []string)` to `internal/reconcile/reconcile.go`. It requires the PR merged, every phase of the slice at `checked == total` on the default branch (spec status on a fresh clone), and the tree matching the graded attempt. It returns a `slice_merged` record and no roadmap edit. Codes are as in `Check` (0, 1, 4).
- [X] T029 [US2] In `internal/pipeline/github.go` and `internal/pipeline/pipeline.go` (`reconcileGitHub`, `reconcileA`): when the run has a slice and not `slices_last`, call `CheckSlice`, log the `slice_merged` event, end the run `succeeded`, and print `slice <id> merged; next: --slice <next>`. When `slices_last`, or for a whole-item run, keep today's `Check`, then the status proposal.
- [X] T030 [US2] Add the acceptance decision (research R7):
  - `internal/readiness` reads an optional `## Item acceptance` section from spec.md into `readiness.acceptance` (the list items);
  - in `internal/pipeline/github.go` and `pipeline.go`, before the status proposal, when it is non-empty: pause `paused:acceptance`, log `acceptance_wait`, and post the decision comment listing the checks;
  - resume on `/approve acceptance` from an approver, or on `pilot approve acceptance`, logging `acceptance_recorded`.
- [X] T031 [US2] Add `acceptance` to `approveCmd`'s `ValidArgs` in `internal/cli/commands.go`, and to the GitHub approval parser (`internal/ghloop`). It is valid only in state `paused:acceptance`. Update `.claude/skills/pilot/SKILL.md` with the new pause and the rule that the agent runs `pilot approve acceptance` only after the human says yes to that exact step.
- [X] T032 [P] [US2] Add pipeline tests in `internal/pipeline/slices_test.go` with fakes:
  - US1 merge gives `slice_merged`, run `succeeded`, no status PR, no roadmap edit;
  - US2 merge (last) gives one status proposal;
  - with `## Item acceptance`, it pauses until `acceptance` is approved;
  - a dev-mode slice run never proposes (C-15) — *not applicable yet: developer mode (019) isn't implemented in the pilot; the case is added with 019*;
  - a US1 PR body contains `Part of #N` and no `Closes`, and the last slice's contains `Closes #N` (FR-016a).

**Checkpoint:** a two-slice item can be run end to end against the fakes, with the US1 guards still green.

---

## Phase 6: User Story 4 - One audit for the item across its slices (Priority: P2)

**Goal:** slice runs audit correctly, and one view shows every run of the item.

**Independent Test:** after a two-slice item is verified, the item view lists both runs and the last run's AU13 passes (SC-006).

- [X] T033 [US4] In `internal/audit/audit.go`, add an `itemAU03` slice variant (contracts/audit.md): exactly 1 draft task PR whose head ends `/<item>-<lower(slice)>`, and status PRs 0 unless `slices_last`, then at most 1. It is selected in `itemVariants` only when `run.json` has `slice`.
- [X] T034 [US4] Add `itemAU13` to `internal/audit/audit.go` (contracts/audit.md "AU13"), only for slice runs. It checks:
  - the readiness record lists earlier slices as merged, with the revision;
  - no status proposal unless `slices_last`;
  - for the last slice, at most one proposal across local slice runs of the same item and target.

  Earlier runs missing locally are reported as `merged on <rev>, evidence elsewhere`, which is not a failure.
- [X] T035 [P] [US4] Add `internal/audit/slices_test.go` with synthetic run folders:
  - a US1 run passes AU03 and AU13;
  - a US1 run with a status PR fails AU03;
  - a last-slice run with two proposals across runs fails AU13;
  - a missing earlier run is reported but passes.

  T010's golden test must stay green.
- [X] T036 [US4] Add the item view, `pilot audit --target <t> --item NNN`, in `internal/cli/audit.go`. It shows one line per local run of the item (slice, state, audit pass/fail), then the newest run's full audit, and writes nothing. Add a test.

---

## Phase 7: User Story 5 - Step mode and the test author work per slice (Priority: P2)

**Goal:** 021 and 024 follow each item's current slice.

**Independent Test:** a labelled two-slice item with auto-draft on is carried slice by slice, with one mode approval and one merge per slice (SC-008).

- [X] T037 [US5] In `internal/step/gather.go`, add `CurrentSlice` to the item state (from `slice.Current` over a fresh clone's spec status). `NewestRun(root, extra, target, item, slice, watching)` also matches `run.json` `slice` (empty means a whole-item run).
- [X] T038 [US5] In `internal/step/next.go` and `internal/step/manager.go`:
  - when the item has 2+ slices, `doctorThenRun` runs `pilot doctor --target T --item N --slice S --baseline`, then `pilot run --target T --item N --slice S`;
  - a succeeded non-last slice run makes the next pass pick the next slice;
  - one slice in flight per item, and one item in flight per target, as today.
- [X] T039 [P] [US5] Extend `internal/step/manager_test.go` and `next_test.go`:
  - a two-slice item gives the doctor and run commands with `--slice US1`, then, after a fake merge, `--slice US2`;
  - a whole-item item gives commands without `--slice`;
  - each slice run waits for its own `/approve mode` (FR-024).
- [X] T040 [US5] In `internal/author/watch.go` and `start.go`:
  - `Candidates` yields `(item, slice)` for the current slice when its companion folder (`CompanionSliceDir`) is empty;
  - `Start` writes to that folder;
  - `skip` and `previousAuto` key on item plus slice;
  - add `--slice` to `pilot author start` (`internal/cli/author.go`).
- [X] T041 [US5] In `internal/author/criteria.go` and `mapping.go`, for a slice draft, limit the criteria to the slice's stories (`US<n>-AS*`; a slice id `USn` maps to story n) and keep the item's FRs as context. Add a test that a US2 draft lists only `US2-AS*` criteria.

---

## Phase 8: Polish and close-out

- [ ] T042 [P] Add the 025 two-slice scenario: `test/integration/fixtures/slices/` (a target with a two-story `tasks.md` and `## Allowed paths (US1)`), `test/integration/fixtures/slices-acceptance/items/001/US1|US2/`, stub-model rules in `pilot/images/stub/scenarios/slices/`, and `test/integration/slices_test.go` asserting quickstart §3 steps 1–6.
- [ ] T043 [P] Update `docs/targets.md` (sliced items, `## Allowed paths (<id>)`, companion layout, `## Item acceptance`), `docs/operator-guide.md` (`pilot slices`, `--slice`), and `docs/troubleshooting.md` (each new reason code and its fix). `go test ./internal/cli/` (the docs check) must pass.
- [ ] T044 [P] Update `specs/014-any-project/contracts/item-spec.md` and `grading.md` with a one-line pointer each to 026's contracts for sliced items.
- [ ] T045 Run `go vet ./... && go test ./...` and `python3 -m unittest discover -s pilot/images/tests`. Fix any failures.
- [ ] T046 Run the fixture with no `--slice` (quickstart §2): audit 11/11 after `down`. Record the run id in `specs/026-slice-runs/evidence/README.md`.
- [ ] T047 **Owner** Choose a live target item with 2+ stories of different risk. Prepare its tasks, `## Allowed paths (<id>)` where wanted, and its per-slice hidden tests (drafted by the test author or by hand, reviewed and merged by the owner).
- [ ] T048 Run the live item slice by slice on kind (quickstart §5), with the owner approving each mode and merging each PR. Put the evidence in `specs/026-slice-runs/evidence/`; the last run audits fully after `down`.
- [ ] T049 [P] Run `speckit-roadmap-debrief` for 026, and propose the status change to the owner. The pilot never records `verified` by itself.

---

## Dependencies & execution order

- **Setup (T001–T003)** first. T001 (the owner merging #154) is needed before the real spec-status 0.4.0 output is available; T003's fixtures may be copied from a local 0.4.0. T002 is needed only before the live runs (T046–T048).
- **T010 before T007:** extend the golden set before any evidence code changes.
- **Foundational (T004–T009)** blocks every story.
- **US1 (T010–T012)** comes next and stays green throughout.
- **US3 (T013–T016)** comes before US2, whose grading needs slice G2 and G3.
- **US2 (T017–T032)**, then **US4 (T033–T036)** and **US5 (T037–T041)**. US4 and US5 are
  independent of each other.
- **Polish (T042–T049)** last. T048 needs T002, T047 and every story.

Within US2: readiness (T017–T019) → CLI (T020–T021) → run and grade (T022–T025, with T023a after T023) → publish (T026–T027) → merge and status (T028–T032).

## Parallel opportunities

- **Setup:** T003 alongside T001.
- **Foundational:** T006 and T008 alongside T004–T005; T009 is independent.
- **US1:** T011 is independent of T010 and T012.
- **US3:** T013 (Go) alongside T014–T015 (Python); T016 once T015 lands.
- **US2:**
  - T019 once T017–T018 land;
  - T021 alongside T020;
  - T025 once T024 lands;
  - T026 and T027 alongside T028–T030.
- **US4 and US5:** can be worked in parallel once US2 is done.
- **Polish:** T042, T043, T044 and T049 can proceed together.

## Implementation strategy

1. **MVP = Phases 1–5:** a two-slice item can be built, graded, published and merged slice by
   slice, with one status proposal at the end. Only US1's guards protect the default path in this
   increment. Ship it behind no flag: without `--slice` nothing changes.
2. Add US4 (audit) before any slice run counts as evidence for `verified`.
3. Add US5 (step mode and author), so the laptop release (013) can carry sliced items without
   commands.
4. Close with the 025 scenario, docs and the live validation.

This file is itself split into user-story phases, so it can be built with
`pilot run --slice` once the MVP lands.
