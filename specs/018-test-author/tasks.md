---
description: "Task list for 018: Test author"
---

# Tasks: Test author

**Input**: Design documents from `specs/018-test-author/` (spec.md, plan.md, research.md,
data-model.md, contracts/, quickstart.md)

**Tests**: Yes. The project's rule is that every change is checked with `go vet ./... && go test
./...` and the image tests (`pilot/images/tests`); this feature adds tests with each task. The
live first use (quickstart) is an owner action.

**Organization**: by user story. US1 (draft) and US2 (tests never reach a worker) are both P1;
US2's identity checks are built into US1's `start`, so they share the foundation. US3 (coverage
mapping) is P2.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel (different files, no dependency on unfinished tasks)
- **[Story]**: US1, US2, US3

## Path Conventions

Go CLI in `cmd/pilot`, `internal/`; tools image script `pilot/images/tools/grade.py`; skills in
`.claude/skills/`; docs in `docs/`.

---

## Phase 1: Setup

- [X] T001 Create package `internal/author/` with a doc comment pointing at
  `specs/018-test-author/` (research R1–R7), and register an empty `author` command group in
  `internal/cli/author.go` (`start`, `check`, `submit`, `status` stubs returning exit 2
  "not implemented") added in `internal/cli/root.go`
  Done as: `internal/author/author.go` (package, codes) and `internal/cli/author.go` (`start`,
  `check`, `submit`, `status`, registered in `root.go`).
- [X] T002 [P] Add the refusal codes of `contracts/cli-author.md` to `internal/cli/exit.go`'s
  table (REFUSE 10 for `COMPANION_NOT_EMPTY`, `DRAFT_PR_OPEN`, `AUTHOR_CAN_WRITE_TARGET`,
  `LEVEL_NEEDS_CLUSTER_MODE`, `DRAFT_EMPTY`, `DRAFT_OUTSIDE_FOLDER`, `MAP_INVALID`,
  `DRAFT_NOT_CHECKED`; FAIL 1 for `DRAFT_HIDDEN_PASSES`, `DRAFT_SUITE_FAILS`) with a test in
  `internal/cli/exit_test.go`
  Done as: codes and exit mapping live in `internal/author/author.go` (`Error.ExitCode`: results
  exit 1, refusals exit 10); the CLI's generic `ExitCode()` handling prints `pilot: CODE: detail`.

---

## Phase 2: Foundational (blocking)

- [X] T003 Add `PILOT_SECURITY_LEVEL` (`minimal` default, `recommended`, `strict`; anything else
  `TARGET_INVALID`) and the optional `targets/<name>.author.env` (`PILOT_AUTHOR_TOKEN`, mode 0600,
  required at `recommended`) to `internal/config/targets.go`, with tests in
  `internal/config/targets_test.go`; `pilot targets` shows `level=…` (`internal/cli/targets.go`)
- [X] T004 Draft evidence in `internal/author/evidence.go`: `Draft` per data-model.md
  (`draft_id`, target, item, spec_sha256, level, identity, can_write_target, workspace, files
  `{path, sha256}`, checks, mapping, pr, state), stored at
  `<evidence root>/drafts/<draft id>/draft.json` plus `events.jsonl`; a test asserts no file
  contents are ever written (`internal/author/evidence_test.go`)
- [X] T005 [P] Exclude `.acceptance-map.json` from the overlay and the hidden-file count in
  `internal/grader/companion.go` (`Overlay`) and `internal/readiness/readiness.go` (count), with
  tests in `internal/grader/companion_test.go` and `internal/readiness/readiness_test.go`
- [X] T006 [P] Criteria ids from a spec in `internal/author/criteria.go`: `US<n>-AS<m>` for each
  numbered acceptance scenario under `### User Story n`, and each `FR-<nnn>` (including suffixes
  such as `FR-006a`), each with a one-line title; tests on `specs/018-test-author/spec.md` and on
  `cli-schema-mcp`'s 006 spec text in `internal/author/criteria_test.go`

**Checkpoint**: levels, evidence and criteria exist; nothing calls them yet.

---

## Phase 3: User Story 1 - Draft hidden tests for a prepared item (P1) 🎯 MVP

**Goal**: `author start → check → submit` turns "no hidden tests" into a companion PR.

**Independent Test**: quickstart steps 2–4 on item 006 open a PR touching only
`cli-schema-mcp/items/006/`.

- [X] T007 [US1] `author start` in `internal/author/start.go` + `internal/cli/author.go`: run
  readiness with the hidden-test rule skipped (add a `SkipHidden` option to `readiness.Check`,
  tested in `internal/readiness/readiness_test.go`); refuse `COMPANION_NOT_EMPTY` when the item's
  companion folder has files (unless `--replace`) and `DRAFT_PR_OPEN` when the companion has an
  open PR from branch `author/<NNN>-*`; create the workspace (`spec.md`, `plan.md`, `tasks.md`,
  `criteria.json`, read-only `base/` checkout of the readiness commit, empty `draft/`); write
  draft evidence; print workspace, level and identity
  Done as: readiness is injected (`Env.Ready`, `readiness.Deps.SkipHidden`), so `start` is
  tested without spec status; `base/` files are read-only, its folders writable (cleanup).
- [X] T008 [US1] Draft validation in `internal/author/validate.go`: `DRAFT_EMPTY` (no files in
  `draft/` other than the map), `DRAFT_OUTSIDE_FOLDER` (symlinks, absolute or `..` paths); a
  stable draft hash (sorted paths + SHA-256s); tests in `internal/author/validate_test.go`
- [X] T009 [US1] `grade.py` `author_check` in `pilot/images/tools/grade.py`: with meta
  `author_check: true`, a baseline grading also emits the hidden tests' output as a second frame
  file `hidden_output.txt`; `grade.json` keeps the withheld summary; tests in
  `pilot/images/tests/test_grade_modes.py` (output in the frame file, not in grade.json; absent
  without the flag)
- [X] T010 [US1] `author check` in `internal/author/check.go`: validate the draft (T008), build
  the overlay from `draft/` (excluding the map), and reuse the item pipeline's preparation and
  baseline (snapshot, grading namespace, toolchain probe, target proxies, prefetch with the draft
  as overlay, `GradeItem` in baseline mode with `author_check`) via exported helpers in
  `internal/pipeline/item.go`; map the result to `DRAFT_OK`, `DRAFT_HIDDEN_PASSES` or
  `DRAFT_SUITE_FAILS`; print `hidden_output.txt` to the terminal and never store it; record the
  check (draft hash, result, `grade.json`) in draft evidence; tear the grading namespace down
  Done as: `pipeline.AuthorBaseline` reuses `prepareItem` and the baseline grader; the draft
  overlay is pre-written so the companion isn't read. The kind run of this path is T024.
- [X] T011 [US1] `author submit` in `internal/author/submit.go`: refuse `DRAFT_NOT_CHECKED`
  unless the last check passed for the current draft hash; clone the companion with the level's
  identity (minimal: the operator's git access; recommended: `PILOT_AUTHOR_TOKEN`), copy
  `draft/` to `<path>/items/<NNN>/`, commit `test(<NNN>): hidden acceptance tests for <title>`,
  push branch `author/<NNN>-<draft id>`, open the PR (body: target, item, spec hash, level,
  identity, check result, mapping table; no file contents); record PR URL and commit; a test
  with a local bare repository as the companion in `internal/author/submit_test.go`
  Done as: the PR is created through the GitHub Git Data API (blobs, tree, commit, branch, PR),
  so no token sits in a git config or command line; tested against a fake API.
- [X] T012 [US1] `author status` in `internal/cli/author.go`: the draft's state and evidence
  path, from draft evidence
- [X] T013 [US1] The test-author skill `.claude/skills/test-author/SKILL.md` per
  `contracts/test-author-skill.md`, with the same frontmatter shape as `.claude/skills/pilot/`

**Checkpoint**: a separate session can draft and submit tests for a prepared item.

---

## Phase 4: User Story 2 - The tests never reach a build worker (P1)

**Goal**: identity rules per level, the new leak check, and no test contents outside the draft.

**Independent Test**: at recommended with a token that can push to the target, `start` refuses;
a hidden test copied into the target makes `gate` refuse `HIDDEN_TESTS_IN_TARGET`; draft
evidence contains no test text.

- [X] T014 [US2] Identity check in `internal/author/identity.go`: resolve the level's identity
  (login) and whether it can push to the target (GitHub API `permissions.push`); refuse
  `AUTHOR_CAN_WRITE_TARGET` at recommended; at minimal record `can_write_target` and print the
  "protect the target's main branch" reminder; refuse `LEVEL_NEEDS_CLUSTER_MODE` at strict;
  called first by `start` and again by `submit`; tests with a fake API in
  `internal/author/identity_test.go`
- [X] T015 [US2] `HIDDEN_TESTS_IN_TARGET` readiness rule in `internal/readiness/readiness.go`:
  refuse when any hidden-test file's SHA-256 equals any file's in the target checkout (map file
  excluded); tests in `internal/readiness/readiness_test.go`; add the code to
  `specs/014-any-project/contracts/readiness.md` and `docs/troubleshooting.md`
- [X] T016 [P] [US2] Level on runs: record `security_level` in `run.json`, add `level=<…>` to the
  mode-approval line (`modeExtra` in `internal/pipeline/item.go`), and report it in the audit's
  AU09 detail (`internal/audit/audit.go`), never failing a run; tests in the existing pipeline
  and audit tests (update golden wording only for item runs)
  Done as: the level and hidden-test coverage are on the mode-approval line (terminal and
  GitHub), `security_level` in `run.json`, and in the audit's AU04 detail (the hidden-tests row;
  AU09 is about retries).
- [X] T017 [P] [US2] Pilot skill rule in `.claude/skills/pilot/SKILL.md`: the session that drives
  runs never runs `pilot/pilot author …`, never reads a draft workspace or the companion, and,
  when an item has no hidden tests, tells the human to open a separate session with the
  test-author skill (or write them)
- [X] T018 [US2] No-contents test in `internal/author/check_test.go` / `submit_test.go`: a marker
  string in a drafted test never appears in draft evidence, in `grade.json`, or in the PR body

**Checkpoint**: P1 complete; drafting is safe at minimal and recommended.

---

## Phase 5: User Story 3 - Every acceptance criterion is covered (P2)

**Goal**: the coverage map is written, validated, shown in the PR and reported by runs.

**Independent Test**: a draft whose map lists every criterion of item 006 passes `check`; a
missing id fails `MAP_INVALID`; after merge, `gate` reports `uncovered=[]`.

- [X] T019 [US3] Map validation in `internal/author/mapping.go` per
  `contracts/acceptance-map.md` (known ids, referenced files exist in the draft, `uncovered`
  equals the ids not covered, `spec_sha256` matches); `MAP_INVALID` with the first problem;
  called by `check`; tests in `internal/author/mapping_test.go`
- [X] T020 [US3] Mapping table in the companion PR body (criterion id, title, test references;
  uncovered ids listed) in `internal/author/submit.go`
- [X] T021 [US3] Run-side report in `internal/readiness/readiness.go`: read the map if present;
  record `mapping: {present, uncovered, stale}` in the readiness record; "no mapping" when
  absent (never a refusal); show uncovered ids at the mode approval; tests in
  `internal/readiness/readiness_test.go`

---

## Phase 6: Polish & cross-cutting

- [X] T022 [P] `docs/targets.md`: section "Hidden tests: who can see what, and why" (the leak
  path, the three levels, what each enforces, minimal-level advice) and section "Drafting hidden
  tests with the test author" (the skill, the commands, the PR); `docs/troubleshooting.md`
  entries for each new code
- [X] T023 [P] `specs/014-any-project/contracts/grading.md`: `author_check` meta field and the
  `hidden_output.txt` frame file
- [X] T024 Run `go vet ./... && go test ./...` and the image tests; kind smoke test of
  `author check` with a sample Python project and a drafted test that fails on the base
  Done as (2026-11-18): `go vet ./...`, `go test ./...` and the image tests pass in CI (#62).
  Kind smoke test with the pinned tools image from #63, on a throwaway Python sample (not a real
  item; the session that drives runs never runs `pilot/pilot author`): toolchain probe, prefetch,
  baseline with `author_check` passed; the author's `hidden_output.txt` (853 bytes) carried the
  hidden test's marker; `grade.json` did not.
- [X] T025 **Owner action:** first use (quickstart): a separate session drafts item 006's tests
  for `cli-schema-mcp`; the owner reviews and merges the companion PR; `pilot/pilot gate
  --target cli-schema-mcp --item 006` allows
  Done as (2026-11-18): a separate session with the test-author skill drafted 14 hidden tests
  for `cli-schema-mcp` item 006 (one file, every criterion covered; `DRAFT_OK`); the owner merged
  `specfoundry-acceptance` PR #1. Run `20261118T030513Z` then passed attempt 1 on all seven grader checks; the
  owner merged `cli-schema-mcp` PR #43 and the status PR #44; audit 10/10 (AU07 needs `down`).
  Evidence: `specs/018-test-author/evidence/20261118T030513Z/`. Earlier runs `20261118T015314Z` (provider
  blocked the model key) and `20261118T024058Z` (the grader's 3000-character cut, fixed in #70)
  stay on disk, not committed.

---

## Dependencies & Execution Order

- Phase 1 → Phase 2 → US1 → US2 → US3 → Polish. US2's identity check (T014) is called by US1's
  `start` and `submit`; build T014 right after T007 if working in order.
- T009 (grade.py) needs a new tools image before T010 can run on kind: merge, then the digest PR.
- T025 needs everything merged, the new tools image pinned, and `pilot/pilot up`.

## Parallel Opportunities

- T002, T005, T006 (different files) after T001.
- T016, T017 alongside T014–T015.
- T022, T023 at any time after T019.

## Implementation Strategy

MVP = Phases 1–4 (US1 + US2): enough for item 006 at the minimal level. US3 adds the mapping;
the skill already asks for the map, so `check` only starts enforcing it in T019.
