# Tasks: Brownfield intake: tech-debt triage and dependency items

**Input**: `specs/029-brownfield-intake/` (spec.md, plan.md, research.md, data-model.md,
contracts/, quickstart.md)

**Tests**: included. Each story has unit tests; the dependency item runs end to end in the 025
integration harness on kind.

## Phase 1: Setup

- [X] T001 Add `"manifests"` globs per toolchain to `pilot/images/toolchains.json` (research R8) and read them in `internal/images/toolchains.go` (`Toolchain.Manifests`, `IsManifest(path)`), with tests in `internal/images/toolchains_test.go`
  Done as (2026-11-24): the table is in `internal/images/manifests.go` (`Manifests`, `IsManifest`, base-name patterns), not `toolchains.json`, which would rebuild the images (research R8 amendment); tests in `internal/images/manifests_test.go`.

## Phase 2: Foundational (blocks US1 and US2)

- [X] T002 [P] In `internal/ghapp/ghapp.go`, add a read-only installation token request: one repository, a given subset of read permissions, the token returned in memory and never written (`ReadToken`); tests for the request body, a 422 for a permission the App lacks, and that no file is written
  Done as (2026-11-24): `ghapp.ReadToken` in `internal/ghapp/readtoken.go` reads the installation's granted permissions first and asks only for those (all `read`), so no 422; it refuses a token that comes back with any write. Tests in `readtoken_test.go`.
- [X] T003 [P] Add `internal/debt/manifest.go`: resolved-version readers for `package-lock.json` (v2/v3), `requirements*.txt` exact pins (PEP 503 names), `go.mod` requires and `pom.xml` literal or same-file property versions; anything else `unknown` (R5); a diff reader giving each package's from → to between two versions of a file (R14); tests in `internal/debt/manifest_test.go`
  Done as (2026-11-24): `internal/debt/manifest.go` (`Resolved`, `Changes`) and `versions.go` (`Level`, version comparison).
- [X] T004 Add `internal/ghloop/report.go`: `ReportClient(target)` (App mode: T002's read token for `contents` and `pull_requests`, then `checks`, `statuses` and `vulnerability_alerts` one at a time, recording which were granted; otherwise `OperatorClient`), and a `debt.Reader` implementation with GETs only (open PRs, a commit's check runs and combined status, file contents, Dependabot alerts); tests against `internal/fakegh` that fail on any non-GET request (SC-002) and that the publisher credential is never read
  Done as (2026-11-24): the reader is `debt.GitHub` in `internal/debt/github.go` (ghapp imports ghloop, so the client choice is `reportClient` in `internal/cli/debt.go`). GET-only is tested on the recorded documcp responses (`TestDocumcpSnapshot`) rather than `internal/fakegh`, which has no checks, statuses, contents or alerts routes.

## Phase 3: User Story 1 - See the bot-PR pile grouped by root cause (Priority: P1) 🎯 MVP

**Goal**: `./chalkeus forge debt --target <t>` groups the open Dependabot PRs and classifies each one.

**Independent Test**: on the recorded documcp snapshot, every PR failing only the inherited `npm audit` check is in one group, superseded PRs are marked, and no write call is made.

- [X] T005 [P] [US1] Record the documcp test data in `internal/debt/testdata/documcp-20261123/`: the 22 Dependabot PRs open on 2026-11-23 (titles, bodies, head branches and SHAs), their head check runs and statuses, `main`'s head checks and its `package.json` and `package-lock.json` at that commit. No token or private data in the files
  Done as (2026-11-24): `internal/debt/testdata/documcp-20261124/responses.json`, recorded with `testdata/record.py` (read-only `gh api` GETs; bodies and lockfiles trimmed). The same 22 Dependabot PRs as on 2026-11-23; advisories were 403 for the operator's token, so the test covers `unknown`.
- [X] T006 [P] [US1] Add `internal/debt/dependabot.go`: recognise Dependabot PRs and parse single titles (`Bump <pkg> from <a> to <b>[ in /<dir>]`), grouped PRs' body tables and the head branch's ecosystem; other bots and ecosystems without a catalog toolchain are `not triaged` (R3); tests in `internal/debt/dependabot_test.go`
  Done as (2026-11-24): also reads `chore(deps): bump …` titles and `Updates \`pkg\` from A to B` body lines, the forms documcp's PRs use.
- [X] T007 [US1] Add `internal/debt/classify.go`: semver levels (`0.x` minor bumps are major; non-semver `unknown`), the five classes in R4's order with inherited failures compared against the default branch's head, groups keyed by sorted failing checks, `dependency_item` for `needs code changes`; tests in `internal/debt/classify_test.go`, including SC-001 on T005's data
  Done as (2026-11-24): with research R4's shared-check amendment. On documcp: 19 PRs in one group whose failing checks are all shared (`Security Audit`, `deploy` red on main; the auto-merge jobs), 10 superseded, 9 safe, 3 need code changes (esbuild's major; two docs PRs failing `Test Suite (20.x)`).
- [X] T008 [US1] Add advisories (R6): match open Dependabot alerts to PRs by package and first patched version; `unknown` without the permission; tests in `internal/debt/classify_test.go`
  Done as (2026-11-24): in `classify.go` (`fixes`); `TestClassifyRules`.
- [X] T009 [US1] Add `internal/debt/report.go`: gather through `debt.Reader` (at most 100 bot PRs; the rest counted as `not triaged: over the limit`), build the `TriageReport` (data-model.md), and render the text and JSON forms of contracts/cli.md; tests in `internal/debt/report_test.go`
  Done as (2026-11-24): `Build` in `classify.go`, types and rendering in `report.go`; not-triaged PRs listed apart (`not_triaged`), shared checks listed (`shared_checks`).
- [X] T010 [US1] Add `internal/cli/debt.go` (`forge debt --target <t> [--json]`, `requireTarget`, exit 0/2 per contracts/cli.md) and classify `debt` as `free` in `internal/cli/command_audit.go`; tests in `internal/cli/debt_test.go` with a fake reader, plus `TestCommandAudit`
  Done as (2026-11-24): `internal/cli/debt.go`, `debt` free in the command audit; tests in `internal/cli/debt_test.go` (App mode asks only for read and writes nothing to the config folder; token mode never reads the publisher's file). Live on stockroom: `read with app-read`, main's `test` check red.

**Checkpoint**: the report works on its own; no image change.

## Phase 4: User Story 2 - Build and grade a dependency change that breaks code (Priority: P1)

**Goal**: an item with `## Manifest change` runs end to end: the approved change in the snapshot, G2 holding it, the PR carrying it plus the fixes.

**Independent Test**: the integration scenario `TestDependencyItem` on kind (quickstart §3).

- [X] T011 [P] [US2] In `pilot/images/tools/grade.py`, add `fixed_paths` to G2 (`MANIFEST_CHANGED: <path>`) and the `dependency` baseline rule (G4 recorded, not required; G5 runs and must fail), per contracts/grading.md; tests in `pilot/images/tests/test_grade_dependency.py`
  Done as (2026-11-26): `fixed_changed()` lists fixed paths the attempt touched (`--no-renames`, so both sides of a rename count; an unreadable diff fails closed), and G2's detail starts `MANIFEST_CHANGED: …`. The dependency baseline follows the owner's amended rule (contracts/grading.md, research R12 amendment): failing visible tests are the signal and G5 is `skipped` (not measured); passing visible tests need failing hidden tests. 9 tests in `pilot/images/tests/test_grade_dependency.py`.
- [ ] T012 [US2] Owner: merge T011 so the image workflow publishes the tools image, then merge its digest PR for `pilot/images.lock` (T020 needs the new digest)
- [X] T013 [P] [US2] Add `internal/readiness/manifest.go`: parse `## Manifest change` (contracts/item-spec.md) and evaluate `MANIFEST_CHANGE_UNREADABLE`, `_UNRESOLVED` (fetch the commit with depth 2), `_NOT_MANIFEST` (T001's globs and the commit's diff), `_STALE` (each file's blob at the base vs the commit's parent), `_EMPTY` and `_SLICED`; wire into `internal/readiness/readiness.go` after the item rules, recorded in the readiness result; tests in `internal/readiness/manifest_test.go`
  Done as (2026-11-26): `internal/readiness/manifest.go`: `ParseManifestChange` and `CheckManifestChange` (fetch depth 2 from origin; all six codes; on success the file blobs and the bumped packages via `debt.Changes`), wired as rule 4b after the allowed paths; tests in `manifest_test.go` and `TestManifestChangeReadiness` (gate and doctor).
- [X] T014 [P] [US2] In `internal/fixture/snapshot.go`, add a snapshot whose tree replaces the listed files with their blobs at the approved commit, and a replay that first commits that tree on the base (the commit's author and subject; body naming the commit and `Chalkeus-Run:`), checks it equals the snapshot's tree, then replays the worker's commits; tests in `internal/fixture/snapshot_test.go` (tree equality, both commits in the bundle, `REPLAY_MISMATCH` still caught)
  Done as (2026-11-26): `fixture.SnapshotWith` (temporary index: base tree with the files at the commit) and `fixture.ReplayWith` (manifest commit on the base with the approved commit's author and subject, body naming the commit, `Chalkeus-Run:`; tree checked against the snapshot; then the worker's commits); `TestSnapshotAndReplayWithManifest`.
- [X] T015 [US2] Wire dependency items in `internal/pipeline/item.go`: use T014's snapshot and replay when readiness found a manifest change; record `manifest_change` in the run state and write `manifest-change.json` (data-model.md); put `fixed_paths` in the baseline meta and `dependency: true`; tests in `internal/pipeline/item_test.go`
  Done as (2026-11-26): `internal/pipeline/item.go`: `manifest_change` in the run state and `manifest-change.json` (with `replayed_commit` set at publish), `SnapshotWith`/`ReplayWith`, baseline meta `fixed_paths` and `dependency: true`; `TestDependencyItemInputs`.
- [X] T016 [US2] In `internal/worker/worker.go`, put `fixed_paths` in the attempt meta and add one line to the worker's task text: the listed dependency files are approved and fixed, don't edit them, fix the code; tests in `internal/worker/item_test.go`
  Done as (2026-11-26): `worker.Item.FixedPaths` → attempt meta `fixed_paths` and one rule in `prompt.BuildItem`; `TestFixedPathsInMetaAndPrompt`.
- [X] T017 [US2] Show the manifest change in the mode approval (commit, files, each package's from → to via T003's diff reader) and the dependency baseline line (`K visible tests fail: that's the work`) in `internal/pipeline/item.go`; tests for the text
  Done as (2026-11-26): `manifestText` in the mode text (terminal and GitHub): commit, subject, files, each package from → to (level); dependency baseline lines (tests fail: that's the work). The grader doesn't count failing tests, so the line doesn't give a number.
- [X] T018 [US2] Report the manifest-change rules in `forge doctor` and refuse with their codes in `forge gate` (exit 10); tests in `internal/cli/gate_root_test.go` (or a new `internal/cli/manifest_test.go`)
  Done as (2026-11-26): Readiness carries the codes, so `gate` refuses with them (exit 10) and `doctor` lists every one; covered by `TestManifestChangeReadiness`. Troubleshooting entry `#manifest-change`.
- [ ] T019 [US2] Add the integration fixture in `test/integration/fixtures/`: a stub target whose test fails after one planned major bump, a dependency item naming the bump's commit, hidden tests, and two stub attempts (one edits `package-lock.json`, one fixes the code)
- [ ] T020 [US2] Add `test/integration/dependency_test.go` (`TestDependencyItem`): baseline passes with visible tests failing; the bad attempt fails G2 with `MANIFEST_CHANGED`; the good attempt passes offline; the publish bundle holds the manifest commit (blobs equal the approved commit's) then the fix; the audit passes (SC-003, SC-004)

**Checkpoint**: dependency items work end to end on kind.

## Phase 5: User Story 3 - Baseline problems flagged before the first run (Priority: P2)

**Goal**: the report lists baseline flags.

**Independent Test**: on a fixture with a red default-branch check, `--forceExit` in a test script and a package with an install script, the report lists all three with where they were found.

- [X] T021 [US3] Add `internal/debt/flags.go`: `red default branch` (failing head checks), `force exit` (`--forceExit`, Mocha `--exit`, `forceExit: true` in `jest.config.*` or `package.json`), `install scripts` (`hasInstallScript` entries in `package-lock.json`, naming them and linking docs/targets.md's native builds section) with `path:line` (R7); render them in T009's report; tests in `internal/debt/flags_test.go`
  Done as (2026-11-24): `internal/debt/flags.go`, with the report (T009); `flags_test.go` and the documcp snapshot (3 red checks, `--forceExit` in `test:ci`, 11 packages with install scripts).

## Phase 6: User Story 4 - Brownfield onboarding docs (Priority: P3)

**Goal**: a new owner can go from an existing repository to the report and a dependency item using only the docs.

**Independent Test**: the docs check passes; a reader follows the brownfield section.

- [ ] T022 [US4] Add a brownfield section to `docs/targets.md`: run `forge debt`, act on it (merge, batch or close bot PRs yourself), fix baseline flags, prepare a dependency item (`## Manifest change`, rebasing a stale bot PR, hidden tests that guard what must not change); link it from `docs/greenfield.md` and `docs/operator-guide.md`
- [ ] T023 [US4] Documcp friction fixes in `docs/targets.md` and `docs/greenfield.md`: confirm F1 and F6 (fixed by #183); add F2's exact Spec Kit install commands and F4's `adr.dir` check; note F3 and F5
- [ ] T024 [US4] Add a 029 section to `docs/release-notes.md` (the new command, dependency items, the App permissions the report can use: Checks, Commit statuses, Dependabot alerts, all read)

## Phase 7: Polish and close

- [ ] T025 Privacy check: the report's text and JSON, `manifest-change.json` and the test data contain no token, key or model host (scan, counts only); `go vet ./... && go test ./...` and the Python tests pass
- [ ] T026 Live proof (SC-005): the owner prepares documcp's zod 3 → 4 dependency item (manifest change, tasks, hidden tests); a run reaches a merged PR with the audit passing; evidence in `specs/029-brownfield-intake/evidence/`
- [ ] T027 Run `speckit-roadmap-debrief` for 029 and propose the status change to the owner

## Dependencies

- Phase 1 → Phase 2 → US1 and US2 (independent of each other). US3 builds on US1's report
  (T009). US4 follows US1 and US2. Phase 7 last.
- In US2: T011 → T012 → T020; T013 and T014 → T015 → T016, T017, T018 → T019 → T020.
- **MVP:** Phase 1, Phase 2 (T003, T004) and US1: the read-only report, with no image change.

## Parallel opportunities

- T002 and T003; T005 and T006; T011, T013 and T014 (different files).
- US1 (Go only) and US2's T011 (Python, image) can proceed side by side.

## Implementation strategy

1. Ship the report first (Phases 1–3, then Phase 5): useful on any brownfield repository at once,
   and it finds the live candidate for T026.
2. Then the grading change (T011, T012): its image rebuild takes a workflow run and a digest PR.
3. Then dependency items (T013–T020), the docs, and the live proof.
