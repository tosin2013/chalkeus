# Implementation Plan: Slice runs

**Branch**: `026-slice-runs-plan` | **Date**: 2026-11-22 | **Spec**: [spec.md](spec.md)

## Summary

A slice is one or more phases of an item's approved `tasks.md`. It is computed from spec-status's
per-phase task counts (research R1, R2), never by parsing `tasks.md`.

`pilot run --item NNN --slice US2` is a normal item run with these differences:
- **evidence:** `run.json` carries `slice`;
- **readiness:** new rules, the main one being that earlier slices must be merged on the
  default branch (R3, R10);
- **allowed paths:** per slice when the spec gives them;
- **grading:** G3 checks only this slice's ticks, and a new G8 re-runs the earlier slices'
  hidden tests before G5 (R5);
- **publishing:** one PR per slice (R4);
- **merge:** a non-last slice's merge records `slice_merged` and proposes nothing. The last
  slice's merge leads to the status proposal, after an optional `acceptance` decision (R7).

The audit gains an AU03 variant and AU13 for slice runs only, so committed evidence audits
unchanged (R8). Step mode and the test author follow each item's current slice (R9).

## Technical Context

- **Language:** Go for the pilot (`cmd/pilot`, `internal/`), and Python for the grader script
  (`pilot/images/tools/grade.py`).
- **Dependencies:**
  - spec-status 0.4.0, which adds `tasks.phases` (tosin2013/spec-status#66). It is pinned in
    `pilot/pins.env` by PR #154; the tools image follows through the image workflow (C-09);
  - existing packages: `gate`, `readiness`, `grader`, `publisher`, `reconcile`, `audit`,
    `evidence`, `step`, `author`, `prompt`, `config`.
- **Storage:** no new store. Slices and their state are recomputed from spec-status on a fresh
  clone (R3). Evidence gains fields and events (data-model.md).
- **Testing:**
  - `go test ./...`, with fake spec-status JSON for the slice cases;
  - `python3 -m unittest` for grade.py;
  - a two-slice scenario in 025's integration harness (quickstart §3);
  - one live run (quickstart §5).
- **Target platform:** laptop mode on kind, the same as 014 and 021. Cluster mode (011) reuses the
  same packages.
- **Constraints:**
  - without `--slice`, behaviour, evidence and audit rows are unchanged (FR-005, FR-022);
  - fail closed on any unreadable input (constitution II).
- **Scale:** items with roughly 2–8 stories. One slice per item in flight.

## Constitution Check

| Principle | How 026 holds | ✓ |
|---|---|---|
| I. Roadmap is the source of truth | slices come only from spec-status JSON (R1); merged-ness from the default branch (R3); the pilot writes no intermediate status | ✓ |
| II. Gates fail closed | every new readiness rule refuses on missing or unreadable input; `SLICE_UNEVALUABLE` when spec-status is too old | ✓ |
| III. Evidence defines done | `verified` only after the last slice, with full reconcile; AU13 checks slice order; the acceptance decision is recorded | ✓ |
| IV. Least privilege | no new credential or mount; earlier slices' hidden tests go only to the grader (`grading/earlier/`), never to a worker (P11) | ✓ |
| V. Humans approve and merge | a mode approval per slice, a human merge per slice PR, an optional human acceptance decision; nothing auto-merges | ✓ |
| VI. Build on, don't rebuild | per-phase counts are added to spec-status (#66), not re-implemented; reuses the approvals, overlay and readiness machinery | ✓ |
| VII. Repeatable and portable | no platform-specific code; pins updated with the spec-status release | ✓ |
| Delivery: "one draft PR per item or task" | a slice is a group of the item's tasks, so one PR per slice fits; a non-last slice PR doesn't close the item's issue (FR-016a) | ✓ |

Roadmap constraints: C-01 (R1), C-02 (R10), C-04 (one draft PR per slice), C-12 (humans choose
slices: the pilot only reads them), C-14 (summaries only for G5 and G8). C-15: dev-mode slice runs
are stamped and propose nothing, as today.

No violations. Re-checked after Phase 1 design: still none.

## Project Structure

### Documentation (this feature)

```text
specs/026-slice-runs/
├── spec.md, plan.md, research.md, data-model.md, quickstart.md
├── contracts/  cli.md · slices.md · grading.md · audit.md
└── checklists/requirements.md
```

### Source code (changes)

```text
internal/slice/slice.go         Phase, Slice, FromPhases, State, Current (pure; R2, R3)
internal/readiness/readiness.go parse tasks.phases (internal/gate/gate.go is unchanged)
internal/readiness/readiness.go slice rules (R10); OPEN_ITEM_PR widened
internal/readiness/paths.go     `## Allowed paths (<id>)`
internal/config/targets.go      CompanionSliceDir; per-slice layout detection (R6)
internal/grader/companion.go    Overlay over a list of folders; earlier/ staging
internal/grader/item.go         meta.slice; stage grading/earlier/<n>
internal/pipeline/item.go       RunItem(item, slice); SetItemRun records slice; slices_last
internal/evidence/evidence.go   SetItemRun(target, item, slice); Slice()
internal/prompt/prompt.go       the slice paragraph (contracts/audit.md, Prompt)
internal/publisher/publisher.go branch <item>-<slice>; ItemTitle " (US2)"
internal/reconcile/reconcile.go CheckSlice (slice 100% + tree; no proposal); acceptance wait
internal/pipeline/github.go     slice_merged path; paused:acceptance; /approve acceptance
internal/audit/audit.go         itemAU03 slice variant; itemAU13; item view
internal/step/next.go, gather.go  current slice; NewestRun(item, slice)
internal/author/watch.go, start.go  per-slice candidates and folders; story-scoped criteria
internal/cli/commands.go, run.go, slices.go, approve.go, audit.go, doctor.go, author.go
pilot/images/tools/grade.py     slice G3, G8, slice baseline (contracts/grading.md)
pilot/images/tests/test_grade_slices.py
pilot/pins.env, .github/workflows/go.yml   spec-status release with phases
test/integration/slices_test.go + fixtures/slices{,-acceptance}/   (025 harness)
docs/targets.md, docs/operator-guide.md, .claude/skills/pilot/SKILL.md, docs/troubleshooting.md
```

**Structure decision:** one new pure package (`internal/slice`), and small, explicit branches
on "is this a slice run" in the existing pipeline. There is no second pipeline. That keeps the
default path identical and easy to diff against today.

## Delivery order

1. spec-status 0.4.0: pinned by PR #154 (an owner merge). Then the images are rebuilt, and
   readiness parses `tasks.phases`.
2. `internal/slice` and `pilot slices` (read-only, useful on its own).
3. Readiness rules, and per-slice allowed paths.
4. Companion layout and overlay lists; grader G3s, G8 and baseline.
5. Run and publish: `--slice`, branch, title, prompt paragraph.
6. Reconcile `slice_merged`, and the acceptance decision.
7. Audit: the AU03 variant, AU13 and the item view.
8. Step mode and author per slice.
9. The 025 two-slice scenario, docs, then the live validation.

Nothing is blocked any more. Live runs need the rebuilt tools image (its grader runs
`spec status`).

## Complexity Tracking

None. There are no constitution violations to justify.
