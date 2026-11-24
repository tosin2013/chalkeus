# Implementation Plan: Brownfield intake: tech-debt triage and dependency items

**Branch**: `029-brownfield-intake` | **Date**: 2026-11-24 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/029-brownfield-intake/spec.md`

## Summary

Two additions, built on what forge already has:

1. **`./chalkeus forge debt --target <t>`**, a read-only report (no cluster, no model):
   - it reads the target's open Dependabot PRs, their head checks, the default branch's head
     checks and its manifest files through the REST API;
   - it groups the PRs by failing checks, and classifies each one. A failure the default branch
     also has is treated as inherited, not caused by the PR (documcp's `npm audit`);
   - it flags baseline problems.
   - It reads with an App installation token restricted to read permissions, in memory (App
     mode), or with forge's existing read client.
2. **Dependency items**: a `## Manifest change` section in the item's spec names one commit and
   its manifest files.
   - Readiness checks that the change touches only those files and still applies to the base.
   - The snapshot's tree takes the files from that commit, so the worker, baseline and
     prefetch all see the approved manifest without further changes.
   - G2 fails on any change to those files (`fixed_paths`).
   - The baseline accepts failing visible tests (they are the work) but still requires the hidden
     tests to fail.
   - The replay puts a manifest commit on the base before the worker's commits, so the draft PR
     carries the approved change byte for byte plus the fixes.

Plus the catalog's `manifests` globs, the brownfield docs, and the documcp friction fixes.

## Technical Context

**Language/Version**: Go 1.27 (`cmd/chalkeus`, `internal/`); Python 3.13 for the in-image grader
(`pilot/images/tools/grade.py`)

**Primary Dependencies**: `google/go-github/v85` (already used), git (host side, already used);
no new module

**Storage**: run evidence files (`manifest-change.json`, run state); no new store. The report
writes nothing.

**Testing**: `go test ./...` (unit, with `internal/fakegh` for GitHub); `pytest
pilot/images/tests/`; the 025 integration harness on kind (`-tags integration`) for the
dependency item end to end; the docs check

**Target Platform**: the laptop forge (Linux amd64 today), kind cluster for item runs

**Project Type**: CLI and pipeline (single Go module)

**Performance Goals**: the report finishes in under a minute for 100 bot PRs (about 2 + 2×PRs
GETs, within GitHub's rate limits)

**Constraints**: read-only report (FR-007); no change to worker isolation, hidden-test reach,
secrets or merge (FR-016); grading additions are backward-compatible (golden audit unchanged);
`grade.py` changes go through the image workflow and its digest PR (C-09)

**Scale/Scope**: up to 100 open bot PRs per report; four catalog toolchains; one approved change
per dependency item, not sliced (R13)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | How 029 complies |
|---|---|
| I. The roadmap is the source of truth | A dependency item is an ordinary roadmap item with a spec; the report schedules nothing and its suggestions are only text (C-01) |
| II. Gates fail closed | The manifest-change rules refuse before any cluster work; anything the report can't read is `unknown`, never `safe` |
| III. Evidence, not CI, defines done | The report only displays CI results; dependency items are graded by the independent grader with hidden tests, and `manifest-change.json` records what was approved |
| IV. Least privilege (non-negotiable) | The report uses a read-only token (narrowed App token, or the read client); the publisher's write credential is never loaded. Worker and grader isolation are unchanged; the worker can't add dependencies (offline grader, G2 `fixed_paths`) |
| V. Humans approve, humans merge | The owner approves the manifest change (by naming its commit), the mode, and merges; forge never merges, closes or comments on bot PRs (FR-017) |
| VI. Build on, don't rebuild | Reuses the snapshot, replay, prefetch, G2, readiness, `ghloop` and `ghapp`; new code is the report package and small extensions |
| VII. Repeatable and portable | The documcp snapshot is recorded test data; image changes are digest-pinned through the image workflow |

**Result:** pass, no violations. **Re-check after Phase 1:** still pass. The one trade-off
(R12: a dependency item's baseline doesn't re-prove the visible tests pass on the unchanged
code) is covered by the report's `red default branch` flag, and doesn't weaken II–IV.

## Project Structure

### Documentation (this feature)

```text
specs/029-brownfield-intake/
├── plan.md              # this file
├── research.md          # R1–R15
├── data-model.md        # TriageReport, ManifestChange, meta additions, refusal codes
├── quickstart.md        # validation scenarios
├── contracts/
│   ├── cli.md           # forge debt; changes to run, gate, doctor
│   ├── item-spec.md     # ## Manifest change and its readiness rules
│   └── grading.md       # fixed_paths, dependency baseline
└── tasks.md             # /speckit-tasks
```

### Source Code (repository root)

```text
internal/
├── debt/                    # NEW: the triage report
│   ├── dependabot.go        #   parse titles, grouped tables, branches (R3)
│   ├── manifest.go          #   resolved versions per toolchain (R5); also used for R14
│   ├── classify.go          #   groups, classes, levels (R4)
│   ├── flags.go             #   baseline flags (R7)
│   ├── report.go            #   gather via a Reader interface; text and JSON output
│   └── testdata/documcp-20261123/   # recorded API responses (SC-001)
├── ghloop/
│   └── report.go            # NEW: ReportClient (R1), read-only GETs used by debt
├── ghapp/ghapp.go           # read-only token request (permissions subset, in memory)
├── images/toolchains.go     # catalog "manifests" globs (R8)
├── readiness/
│   └── manifest.go          # NEW: ## Manifest change parsing and MANIFEST_CHANGE_* rules (R9, R13)
├── fixture/snapshot.go      # snapshot with replaced files; replay with a manifest commit (R10)
├── pipeline/item.go         # wire: snapshot, fixed_paths, dependency baseline, mode text, evidence (R10–R14)
├── worker/worker.go         # fixed_paths in the attempt meta; prompt line: the files are fixed
└── cli/
    ├── debt.go              # NEW: the command
    └── command_audit.go     # debt: free
pilot/images/
├── toolchains.json          # "manifests" per toolchain
├── tools/grade.py           # fixed_paths in G2; dependency baseline
└── tests/test_grade_dependency.py   # NEW
test/integration/
└── dependency_test.go       # NEW: TestDependencyItem (SC-003)
docs/
├── targets.md               # brownfield section; F1, F2, F4, F6
└── release-notes.md         # 029 section
```

**Structure Decision**: one new package (`internal/debt`) for the report, which needs no cluster
code; everything else extends the packages that own each step today. `internal/debt` takes a
small `Reader` interface (list PRs, checks, statuses, file contents, alerts), implemented by
`ghloop` and faked in tests from the recorded documcp responses.

## Delivery order

1. **Report (US1, US3):** catalog `manifests`, `internal/debt`, `ghloop.ReportClient`, the
   command; recorded-snapshot and fake tests. Shippable on its own: no image change.
2. **Grading (US2 part 1):** `grade.py` `fixed_paths` and dependency baseline, with Python tests.
   The image workflow publishes; its digest PR is merged before step 3's integration test.
3. **Dependency items (US2 part 2):** readiness rules, snapshot and replay, pipeline wiring, mode
   text, evidence; unit tests; the integration scenario on kind.
4. **Docs (US4)** and release notes.
5. **Live proof (SC-005):** the owner prepares documcp's zod 3 → 4 item; debrief.

## Complexity Tracking

No constitution violations to justify.
