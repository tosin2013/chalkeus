# Implementation Plan: Research before build

**Branch**: `033-research-before-build` | **Date**: 2026-11-25 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `specs/033-research-before-build/spec.md`

## Summary

After every failed attempt, forge builds a deterministic diagnosis from the attempt's kept
evidence (how it ended, research-worded tasks, files changed outside the allowed paths, egress
refusals that aren't probes, uncatalogued failure reasons, the test-run allowance) and shows it
with the retry question, in the summary and on the tracking issue, recommending one of retry,
raise the budget, or back to planning. The owner can answer `replan` (terminal or GitHub) to end
the run as `stopped(back-to-planning)`. Doctor warns about research-worded tasks, a low test-run
allowance (the grader now times the test command) and a runtime version the toolchain doesn't
provide. Docs cover the three causes of a deadline and the route back to planning. Decisions in
[research.md](research.md).

## Technical Context

**Language/Version**: Go (module `github.com/tosin2013/chalkeus`, as in `go.mod`); Python 3 for
the grader script inside the tools image (`pilot/images/tools/grade.py`)
**Primary Dependencies**: none new (standard library; existing `internal/*` packages)
**Storage**: run evidence files (`specs/<feature>/evidence/<run>/…`): new
`attempts/<task>-<n>/diagnosis.json`; new fields in `run.json` state
**Testing**: `go vet ./... && go test ./...` (includes the docs checks); `pilot/images/tests/`
for the grader script; kept evidence of documcp run `20261125T163307Z` and the fixture run
`specs/016-go-pilot/evidence/20261117T001332Z` as replay fixtures
**Target Platform**: the forge CLI on Linux and macOS hosts; kind (laptop) and the operator's
cluster mode run the same pipeline code
**Project Type**: CLI (`./chalkeus forge`)
**Performance Goals**: a diagnosis is built in under a second from local files; doctor's new
checks add no cluster round trip beyond the existing baseline
**Constraints**: deterministic, no model calls (FR-006); warnings never refuse a run (FR-015);
`TestFixtureModeAndSummaryUnchanged` keeps the committed fixture summary byte-identical; old
runs (no diagnosis, old grader image) keep working
**Scale/Scope**: one new package (~400 lines with tests), changes in pipeline, evidence,
decisions, ghloop, step, audit, doctor, the grader script and three docs

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Check | Result |
|---|---|---|
| I. Roadmap is the source of truth | Nothing is scheduled from the diagnosis or doctor's warnings; task wording is read for advice only (research R2). | Pass |
| II. Gates fail closed | No gate changes; the new checks are warnings and can't open a gate. | Pass |
| III. Evidence, not CI | The diagnosis and the owner's decision are written to the run's evidence; the audit reports a run ended for planning explicitly. | Pass |
| IV. Least privilege | The diagnosis runs on the host over kept files; no new credential, mount or egress; the worker gains nothing. | Pass |
| V. Humans approve, humans merge | Forge recommends only; `replan` is a human decision through the existing approval channels and allow-list (FR-007, FR-008). | Pass |
| VI. Build on, don't rebuild | Reuses `ghloop.Checklist`, `images.IsManifest`, `images.Resolve`, the decisions and watcher machinery, Spec Kit's `/speckit-plan` research as the route. | Pass |
| VII. Repeatable and portable | Pure Go over files; the grader timing is pinned with the tools image (C-09); same behaviour on laptop and cluster mode. | Pass |

Post-design re-check (after Phase 1): unchanged, all pass. No complexity tracking needed.

## Project Structure

### Documentation (this feature)

```text
specs/033-research-before-build/
├── plan.md              # This file
├── research.md          # Phase 0: R1–R9
├── data-model.md        # Phase 1: signal, diagnosis, planning decision, baseline timing
├── quickstart.md        # Phase 1: replay and end-to-end checks
├── contracts/
│   ├── diagnosis.md     # diagnosis.json and the retry-question / comment text
│   └── cli.md           # approve replan, /approve replan, doctor's new lines, run state
├── checklists/requirements.md
└── tasks.md             # Phase 2 (/speckit-tasks)
```

### Source Code (repository root)

```text
internal/diagnose/            # NEW: Diagnose(attempt evidence) → Diagnosis; research wording;
│                             #      failure-reason catalogue; probe filter; text rendering
├── diagnose.go
├── signals.go
├── wording.go
├── diagnose_test.go          # replays documcp 20261125T163307Z and the fixture's A/1
└── testdata/                 # copies of the kept files the replays need (the documcp run's
                              # evidence isn't committed; no secrets: report, meta, prompt task
                              # list, proxy-denials.log, the attempt record)
internal/pipeline/pipeline.go # runTask: build + record diagnosis; confirmRetry uses it;
│                             # Resume: replan → stopped(back-to-planning)
internal/pipeline/item.go     # PREFETCH/TOOLCHAIN failures name the planning-PR route
internal/evidence/evidence.go # WriteSummary: Diagnosis section (only when present)
internal/decisions/           # step "replan", accepted while paused(awaiting-retry); Hint
internal/ghloop/approve.go    # /approve replan; PostDecision lists it under a retry
internal/watcher/, pipeline/github.go # ReadyToResume on a replan record; stop on final state
internal/step/                # stopped(...) counts as ended
internal/audit/               # AU02/AU03 n/a on stopped(back-to-planning); ItemView state
internal/cli/commands.go, approve.go  # ValidArgs gains replan
internal/cli/doctor.go        # research wording, test-run allowance, runtime version warnings
internal/doctor/              # (warnings use the existing doctor.Warn)
pilot/images/tools/grade.py   # test_seconds in G4 detail and report
pilot/images/tests/           # grader timing test
docs/troubleshooting.md       # deadline entry rewritten; entries for uncatalogued reasons
docs/targets.md               # research and environment changes belong in planning
docs/greenfield.md            # same, for a new project
.claude/skills/forge/SKILL.md # explain the diagnosis and the replan decision
```

**Structure Decision**: one new leaf package, `internal/diagnose`, with no dependency on the
pipeline (it takes paths and plain values), so it is unit-tested on kept evidence; every other
change is in the package that already owns that surface.

## Phasing (one slice per user story)

1. **US1 (P1) diagnosis:** `internal/diagnose`, the pipeline hook, retry question, decision
   comment and summary. Replay tests first.
2. **US2 (P1) back to planning:** `replan` step, GitHub command, the final state, watcher, step
   mode and audit.
3. **US3 (P2) doctor:** research wording and runtime warnings (no image change), then the grader
   timing and the allowance (image rebuild through the image workflow PR).
4. **US4 (P3) docs:** troubleshooting, targets, greenfield, the forge skill; the reason-catalogue
   docs test.

## Complexity Tracking

No violations.
