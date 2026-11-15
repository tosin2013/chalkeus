# Implementation Plan: GitHub loop — issues, labels, linked PRs and GitHub approvals

**Branch**: `004-github-loop` | **Date**: 2026-11-15 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/004-github-loop/spec.md`

## Summary

The pilot gets a GitHub projection of the target roadmap:
- one issue per roadmap item, with a status label and the item's task checklist;
- every published PR closes its item issue;
- a drift report compares the roadmap, issues, labels and PRs.

Decisions can be approved on GitHub:
- mode and retry by an `/approve <step>` comment on a per-run tracking issue;
- the status change by merging a status-change PR that the publisher opens.

GitHub approvals reuse 012's pending-step rules and add a `github` channel. A GitHub App
becomes an optional second publisher credential: a per-run 1-hour token, limited to the
target repository.

All GitHub writes keep today's pattern: the credential is sourced into a child process or the
publisher Job only, never into the orchestrator. Merges stay human.

## Technical Context

**Language/Version**: Python 3.12 (stdlib only), bash shim `pilot/pilot`; the publisher image's
Python tools

**Primary Dependencies**: none new in Python. `openssl` (already on Ubuntu) signs the GitHub App
JWT (R5). `gh` stays the operator's read-only view, as today.

**Storage**: run evidence folder (`events.jsonl`, `run.json`, `prs.json`, new `github/` folder,
`approvals/pending.json`); `~/.config/specfoundry/` for the approver list, credential choice
and App key (mode 600)

**Testing**: `unittest` with fake GitHub responses (no network); `pilot/tools/docs_check.py`;
owner validation runs on the fixture repository

**Target Platform**: Linux operator machine, kind cluster (laptop mode). Design stays valid for
cluster mode (011).

**Project Type**: CLI

**Performance Goals**: human-paced. Approval polling at most every 15 s while waiting, within
GitHub's rate limits.

**Constraints**:
- fail closed on any GitHub error;
- no credential value in orchestrator memory, logs or evidence;
- the roadmap is the source of truth;
- no merge code path;
- the terminal flow is unchanged.

**Scale/Scope**: 1 target repo; 2 items in the fixture; 3 decision kinds; 2 credential types

## Constitution Check

| Principle | Check | Result |
|---|---|---|
| I. Roadmap is the source of truth | Issues and labels are written *from* the roadmap; drift is reported and labels are restored, never read back as status (FR-002, FR-005) | pass |
| II. Gates fail closed | GitHub unreachable, an unverifiable approver, or a mismatched or edited approval means the run stays paused (FR-015, SC-004) | pass |
| III. Evidence defines done | Issues, the tracking issue, approvals with approver login, and status-change PRs are recorded in `github/` and events; the audit reports the `github` channel | pass |
| IV. Least privilege | App token: 1 hour, target repo only, contents, PR and issues write. The key stays on the host. Token values only in child processes or the publisher Job; P09 unchanged | pass, with a note: the token gains `issues: write` (justified in R2) |
| V. Humans approve, humans merge | Approvals come only from listed approvers; the publisher's own comments never count; the status change is approved by a human **merge**; no merge path | pass. The identity limit in token mode is documented (FR-009a, R4) |
| VI. Build on, don't rebuild | Reuses 012's approval record, the publisher Job and image, `check_token`'s child-process pattern, and spec-status for item and task data | pass |
| VII. Repeatable and portable | Fixture repo, idempotent sync, documented App setup (SC-007) | pass |

No violations. Post-design re-check: AU03 changes from "exactly 2 PRs" to "exactly 2 task PRs,
plus at most one status-change PR per verified item, all drafts by the publisher". Old
records have no status-change PR, so they still pass (SC-006). Still pass.

## Design overview

```text
roadmap (source) ──sync──► item issues + status labels + task checklist   (publisher credential, child process)
                                   ▲ Closes #N
run ── pause(mode|retry) ──► tracking issue comment "decision pending"     (publisher credential)
        ▲                          │ approver: "/approve retry"
        └── resume: read comments (operator gh read) ─ verify approver list + write access
                                   ─ bind to run/step/n ─ record approvals/pending.json (channel=github)
A merged ──► publisher opens status-change PR (roadmap edit only) ── human merges ──► pilot checks diff == proposed
                                                                                    ─► task_verified, B's gate allows
reconcile/drift ──► roadmap vs issues/labels/PRs ── report; restore labels
```

## Project Structure

### Documentation (this feature)

```text
specs/004-github-loop/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── cli-github.md          # new commands, approval flow, exit codes
│   ├── github-objects.md      # issue, label, tracking-issue and status-change PR formats
│   └── audit-github.md        # github channel, AU03 change, new evidence files
├── checklists/requirements.md
└── tasks.md                   # /speckit-tasks
```

### Source Code (repository root)

```text
pilot/specfoundry_pilot/
├── ghsync.py          # NEW: roadmap → issues/labels/checklist; drift report (pure functions + I/O seam)
├── ghapprove.py       # NEW: tracking issue, decision posts, reading /approve comments, approver checks
├── ghapp.py           # NEW: App JWT (openssl child) → installation token in a child-only env file
├── approvals.py       # take(): accept channel=github records; describe() reused for posts
├── pipeline.py        # post decisions to GitHub on pause; resume polls GitHub approvals
├── reconcile.py       # status-change PR path (publisher) beside today's push path
├── publisher.py       # PR body "Closes #N"; kind=task|status in prs.json
├── audit.py           # AU03 rule; AU09/AU10 approver logins for github channel
├── config.py          # approvers, credential choice, App settings
└── cli.py             # `github sync|drift|watch`, credential setup commands
pilot/images/tools/
├── ghops.py           # NEW: GitHub writes run with the token in env only (issues, labels, comments)
└── check_token.py     # probe adds issues write
pilot/tests/            # test_ghsync, test_ghapprove, test_ghapp, test_audit_github
docs/                   # operator guide § GitHub loop; App setup; troubleshooting entries
.claude/skills/pilot/SKILL.md, AGENTS.md   # never post /approve
```

**Structure Decision**: single project, existing layout. Three new modules keep the GitHub logic
testable apart from the pipeline.

## Phasing (for tasks)

1. **P1a, US1 sync and links:** `ghsync`, `ghops` (issues, labels), PR `Closes #N`, drift report.
2. **P1b, US2 GitHub approvals:** tracking issue, decision posts, comment reading, approver
   checks, the `github` channel, AU09/AU10 logins.
3. **P2, US3 status-change PR:** publisher opens it; merge check; AU03 rule.
4. **P2, US4 GitHub App:** `ghapp` token minting, probe, docs.
5. **Validation:** owner runs on the fixture in both credential modes (quickstart).

## Complexity Tracking

| Item | Why needed | Simpler alternative rejected because |
|---|---|---|
| Token gains `issues: write` | Item issues, labels and the tracking issue are issue objects | Using the operator's own `gh` for writes would put the human's identity on pilot posts, and doesn't exist in cluster mode |
