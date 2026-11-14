# Data Model: Working with an agent

All data lives in the run's evidence folder (`pilot/runs/<run-id>/` or a spec's `evidence/`).

## Decision (derived from run state)

| Field | Values | Source |
|---|---|---|
| step | `mode`, `retry`, `reconcile`; `merge` is reported but never approvable | `run.json` `state` + `pending`, and `reconcile/A.json` |
| task | `null` (mode), `A`/`B` (retry), `A` (reconcile) | `run.json` `pending.task` |
| n | attempt number (retry), else `null` | `run.json` `pending.n` |
| merge_commit | merge commit of the shown proposal (reconcile) | `reconcile/A.json` |

Pending step by run state:

| `run.json` state | Pending step |
|---|---|
| `paused(awaiting-mode)` | mode |
| `paused(awaiting-retry)` | retry (`pending.task`, `pending.n`) |
| `paused(awaiting-merge)` and `reconcile/A.json` has `reason: DECLINED`, `proposed`, `merge_commit` | reconcile |
| `paused(awaiting-merge)` otherwise | merge (human on GitHub; not approvable) |
| anything else | none |

## Approval (`approvals/pending.json`)

| Field | Type | Rule |
|---|---|---|
| step | string | equals the pending step when recorded |
| task, n | string/int or null | copied from the pending step |
| merge_commit, proposed, item_id | string | reconcile only; copied from `reconcile/A.json` |
| channel | `harness` | set by `pilot approve` |
| at | UTC timestamp | |

Lifecycle: **recorded** (by `pilot approve`, replacing any earlier one), then **used** (it
matches the step being decided; deleted) or **discarded** (mismatch read, or a terminal answer
for that run; deleted). It's never in the record after use.

## Channel events (`events.jsonl`)

| event | stage | detail |
|---|---|---|
| `approval_recorded` | `approval` | `step=<s> channel=harness` |
| `approval_channel` | `approval` | `step=<s> approved=<true\|false> channel=<tty\|harness\|none>` |

Existing events (`mode_confirmed`, `mode_declined`, `retry_confirmed`, `retry_declined`,
`reconcile_*`) are unchanged.

## Run state additions

- `paused(awaiting-mode)`: new. Nothing is staged.
- `proposed_mode`: the mode shown, saved before the decision, so `approve` can describe it.
