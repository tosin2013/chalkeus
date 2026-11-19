# Implementation Plan: Step mode, the laptop manager loop

**Branch**: `021-step-mode` | **Date**: 2026-11-19 | **Spec**: [spec.md](spec.md)

## Summary

`pilot step --target <name>` is a reconcile loop. Each pass it gathers every proposed item's
state, and for the first item in flight it takes the one next step from a pure decision function
(`step.Next`, research R1). It then delegates:
- drafting goes to 024's `author.Pass`;
- `doctor --baseline` and `run` go to the pilot binary itself (R3).

It comments once per finding on the item's issue (R5), and keeps no state beyond GitHub, the
evidence and a decisions log (R4).

## Technical Context

- **Language:** Go (`cmd/pilot`, `internal/`).
- **Dependencies:** existing pilot packages: `ghloop` (issues, events, comments), `author`
  (024), `readiness`, `evidence`, `config`. The pilot binary is the subprocess.
- **Storage:** `<evidence root>/step/<target>/decisions.jsonl`, plus doctor and run logs.
- **Testing:** `go test ./...` with fakes for GitHub, the subprocess and the gatherer. One live
  run on Stockroom (quickstart).
- **Constraints:**
  - it never approves, merges, labels or edits (FR-005);
  - runs are started only through the normal `run` path;
  - one item in flight per target.

## Constitution Check

- **C-01, C-02:** items come only from the roadmap and `spec status`; the gate and readiness
  still decide ✓
- **C-04:** every merge and approval stays the owner's ✓
- **C-11:** laptop mode; the operator reuses `step.Next` (011) ✓
- **C-12:** humans plan, label and approve ✓

No violations.

## Project Structure

```text
internal/step/next.go        ItemState, Action, Next (pure, FR-004)
internal/step/gather.go      the state from readiness, PRs, runs and the decisions log (R4)
internal/step/manager.go     the loop: gather, Next, act; comments; decisions log
internal/ghloop/labels.go    ProposedBy: the last `labeled` event's actor (R2)
internal/author/watch.go     WatchEnv.Only (drafting one item)
internal/pipeline/item.go    PILOT_STARTED_BY recorded as a run event (FR-007)
internal/config/targets.go   PILOT_PROPOSE_LABEL, PILOT_STEP_INTERVAL_S
internal/cli/step.go         `pilot step --target <name> [--once]`
docs/targets.md, .claude/skills/pilot/SKILL.md, docs/troubleshooting.md
```
