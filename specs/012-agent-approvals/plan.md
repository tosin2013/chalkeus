# Implementation Plan: Working with an agent — pilot skill and harness approvals

**Branch**: `012-agent-approvals` | **Date**: 2026-11-14 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/012-agent-approvals/spec.md`

## Summary

A first cut merged in PR #11:
- `pilot approve`, which approves one pending step;
- runs pause instead of assuming an answer when there's no terminal;
- an approval channel recorded on each decision and reported by AU09 and AU10;
- harness ask rules;
- the `pilot` skill and `AGENTS.md`.

This plan checks that code against the spec, closes the gaps it found, records the CLI and
audit contract changes, and validates the feature with an agent-driven run in both supported
harnesses. Most of the work is verification and documentation; the code changes are small.

## Gap analysis: PR #11 vs the spec

| Requirement | PR #11 | Gap → work |
|---|---|---|
| FR-001 pause with no terminal + hint | done (`approvals.decide`, `pipeline.mode_not_confirmed`, retry, reconcile) | — |
| FR-002 resumable mode pause | done (`paused(awaiting-mode)` in `resume`) | — |
| FR-003 approve mode/retry/reconcile, never merge | done (`approvals.record` refuses `merge`) | — |
| FR-004 only the pending step, after it was shown | done (`pending_step`) | — |
| FR-005 bound and used once | done (`take`) | — |
| FR-006 terminal decides | the terminal answer decides, but a pending approval survives it | **G1:** a terminal answer clears any pending approval, so a later headless `resume` can't use an approval the human overrode with "no" |
| FR-007 channel recorded | done (`approval_channel` events: tty, harness, none) | — |
| FR-008 AU09/AU10 report channels | done (`audit._channels`) | — |
| FR-009 old evidence still passes | not tested | **G4:** a test audits every committed run record (002, 003, 006) and requires a pass with no channel |
| FR-010 harness ask + deny rules | files exist, never exercised | **G5:** owner check in Claude Code and OpenCode (SC-005); record the rule syntax that worked |
| FR-011 skill + AGENTS.md | done | wording reviewed against FR-011's list during T-review |
| FR-012 troubleshooting + README | done | — |
| FR-013 terminal behaviour unchanged; guide correct | behaviour unchanged | **G2:** guide §8 says "every answer counts as no" with no terminal; change it to "the run pauses" and link the troubleshooting entry and the skill |
| Contracts | 002 `contracts/cli.md` and `evidence-record.md` describe the old behaviour | **G3:** 012 contracts (`contracts/cli-approve.md`, `contracts/audit-channels.md`); a one-line "amended by 012" pointer in each 002 contract |
| SC-001 validation run | early run `20261114T213046Z`, evidence not in the repo | **G6:** owner copies that record into `specs/012-agent-approvals/evidence/`; a fresh agent-driven run on the finished feature validates SC-001 |

## Technical Context

**Language/Version**: Python 3.12 (pilot, stdlib only), bash shim `pilot/pilot`

**Primary Dependencies**: none new. Harness configuration files: `.claude/settings.json` (Claude Code), the operator's `~/.config/opencode/opencode.json` (OpenCode; per-user since 2026-11-15)

**Storage**: files in the run's evidence folder: `events.jsonl`, `run.json`, `approvals/pending.json` (deleted when used)

**Testing**: `unittest` (`PYTHONPATH=pilot python3 -m unittest discover -s pilot/tests -t pilot`), `pilot/tools/docs_check.py`; the owner's live validation run

**Target Platform**: Linux operator machine (Ubuntu 24.04), kind cluster per the operator guide

**Project Type**: CLI

**Performance Goals**: n/a (human-paced decisions)

**Constraints**: fail closed; no self-approval path; the terminal flow stays unchanged; no secrets in evidence

**Scale/Scope**: 3 decision kinds, 2 harnesses, 1 audit contract change (AU09/AU10 detail)

## Constitution Check

| Principle | Check | Result |
|---|---|---|
| I. Roadmap is the source of truth | Roadmap 012 exists (1.7.0); status changes only via `/speckit-roadmap-write` | pass |
| II. Gates fail closed | No answer means pause; mismatched or stale approvals are not used; G1 tightens this | pass |
| III. Evidence defines done | Channels recorded in events and reported by the audit; validation run evidence committed | pass |
| IV. Least privilege | No new credentials; workers unchanged; the approval command touches only the run's evidence | pass |
| V. Humans approve, humans merge | Every decision needs a human (terminal or harness prompt); merge never approvable here | pass. The harness channel's limit is documented, not hidden |
| VI. Build on, don't rebuild | Reuses the existing pause/resume states and the harnesses' own permission systems | pass |
| VII. Repeatable and portable | Terminal path unchanged; the guide is corrected (G2); quickstart is runnable | pass |

No violations, so the Complexity Tracking table is empty.

Post-design re-check: the contracts below keep AU09/AU10 pass rules unchanged and only add
detail, so evidence from 002, 003 and 006 stays valid (G4 proves it). Still pass.

## Project Structure

### Documentation (this feature)

```text
specs/012-agent-approvals/
├── spec.md
├── plan.md                    # this file
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── cli-approve.md         # approve command, pause states, exit codes
│   └── audit-channels.md      # channel events and AU09/AU10 detail
├── checklists/requirements.md
├── evidence/                  # 20261114T213046Z (early) + the validation run
└── tasks.md                   # /speckit-tasks
```

### Source Code (repository root)

```text
pilot/specfoundry_pilot/
├── approvals.py               # G1: clear pending approval on a terminal answer
├── pipeline.py                # unchanged unless G1 needs a call site
├── audit.py                   # unchanged (channels already reported)
└── cli.py                     # unchanged
pilot/tests/
├── test_approvals.py          # G1 test
└── test_committed_evidence.py # G4: every committed run record passes the audit
docs/operator-guide.md         # G2: §8 no-terminal sentence
specs/002-m1-gated-pilot/contracts/{cli.md,evidence-record.md}   # G3 pointer lines
.claude/settings.json, opencode.json   # G5: adjusted only if the owner check finds a problem
.claude/skills/pilot/SKILL.md, AGENTS.md   # wording review
```

**Structure Decision**: single project, existing layout. No new modules beyond one test file.

## Complexity Tracking

None.
