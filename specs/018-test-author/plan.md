# Implementation Plan: Test author

**Branch**: `018-plan` | **Date**: 2026-11-17 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/018-test-author/spec.md`

## Summary

Let a **separate agent session** draft an item's hidden acceptance tests and open a pull request
on the companion repository, which a human merges. Build workers never see the tests.

The design has three parts ([research](research.md)):
- **A test-author skill** (R1): `.claude/skills/test-author/SKILL.md`. It is run by a fresh agent
  session, never the session that drives runs. It writes test files into a draft workspace and
  uses only `pilot/pilot author …` commands for everything else.
- **`pilot/pilot author` helper commands** (R2): `start`, `check`, `submit`, `status`. They
  run readiness, prepare the workspace, check the draft against the unchanged base on the
  cluster, check the coverage mapping, and open the companion PR under the level's identity.
  The commands, not the agent, handle credentials.
- **Security levels** (R3): `PILOT_SECURITY_LEVEL=minimal|recommended|strict` in the target
  profile, default minimal. Minimal uses the operator's git access; recommended a
  companion-only token (`targets/<name>.author.env`); strict is refused until cluster mode (011).
  The level is recorded and shown at the start of drafts and runs, and in the audit.

Run-side additions:
- **Coverage mapping** (R4): `.acceptance-map.json` in the item's companion folder; readiness
  reports uncovered criteria; the mapping file is never part of the grader overlay.
- **A new readiness rule** (R5): `HIDDEN_TESTS_IN_TARGET` refuses a run if any hidden-test
  file's content already exists in the target snapshot. This closes the one realistic leak path
  (spec, "Why the rules exist"), which P11 didn't cover.
- **Draft check output** (R6): the author sees its tests' output in its own terminal; evidence
  records only the withheld summary (C-14).

First use: item 006 of `cli-schema-mcp`, at the minimal level.

## Technical Context

**Language/Version**: Go 1.24 (CLI, `cmd/pilot`, `internal/`); Python 3.12 in the tools image
(`grade.py`); Markdown for the skill
**Primary Dependencies**: existing: cobra, client-go, go-github; Spec Kit `spec status`
**Storage**: draft workspaces and draft evidence on the operator's machine
(`<evidence root>/drafts/<draft id>/`); companion via git
**Testing**: `go test ./...`; `pilot/images/tests` (unittest); a kind smoke test of
`author check`; the live first use on item 006
**Target Platform**: laptop mode on kind (Linux, macOS)
**Project Type**: CLI plus in-cluster grading (existing)
**Performance Goals**: `author check` within the item's grader deadline plus prefetch (same as a
run's baseline)
**Constraints**: no test contents in evidence or in the driving session's output (C-14); the
companion PR is the only write; build-worker isolation unchanged
**Scale/Scope**: one draft per item at a time

## Constitution Check

| Principle | How this plan complies |
|---|---|
| I. Roadmap is the source of truth | Drafting needs the same readiness as a run, except hidden tests; the spec hash is recorded |
| II. Gates fail closed | Every author command refuses on doubt (readiness, non-empty companion folder, open draft PR, identity can write the target at recommended) |
| III. Evidence defines done | Draft evidence: level, identity, spec hash, file names and hashes, check results, mapping, PR, commit |
| IV. Least privilege by construction | Build workers unchanged; at recommended the author's identity can't write the target; strict waits for 011; minimal is an owner-approved level, recorded (C-14) |
| V. Humans approve, humans merge | The output is a PR; the pilot never merges |
| VI. Build on, don't rebuild | Reuses readiness, snapshot, baseline grading, prefetch, the companion reader |
| VII. Repeatable and portable | Same commands on any laptop; strict reuses them in 011 |

No violations.

## Project Structure

### Documentation (this feature)

```text
specs/018-test-author/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
└── contracts/
    ├── cli-author.md
    ├── acceptance-map.md
    └── test-author-skill.md
```

### Source Code (repository root)

```text
.claude/skills/test-author/SKILL.md     # new: the skill
internal/cli/author.go                  # new: pilot author start|check|submit|status
internal/author/                        # new: draft workspace, evidence, mapping, submit
internal/config/targets.go              # PILOT_SECURITY_LEVEL; author.env
internal/readiness/                     # HIDDEN_TESTS_IN_TARGET; mapping report
internal/grader/companion.go            # overlay skips .acceptance-map.json
internal/pipeline/item.go               # level at mode approval; reuse for author check
internal/audit/                         # level reported
pilot/images/tools/grade.py             # author_check: return hidden output in the frame only
.claude/skills/pilot/SKILL.md           # the driving session never runs `author`
docs/targets.md                         # levels, why, the test author
```

**Structure Decision**: a new `internal/author` package owns drafts; it reuses `readiness`,
`fixture` (snapshot), `grader` (baseline) and `pipeline` pieces rather than duplicating them.

## Complexity Tracking

None.
