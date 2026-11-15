# Contract: GitHub evidence and audit (amends 012 `contracts/audit-channels.md`)

## Events

- `approval_channel`: the `channel` value may now be `github`. A `github` entry adds
  `approver=<login>` and `comment=<id>`.
- For the status change, the approval is the merge:
  `step=reconcile approved=true channel=github approver=<merger login> pr=<n>`.
- New events: `github_sync`, `decision_posted`, `approval_rejected` (with its reason),
  `status_pr_opened`, `status_pr_checked`.

## Audit

| Check | Pass rule | Detail |
|---|---|---|
| AU03 | **changed:** exactly 2 `task` PRs (A, B), plus 0 or 1 `status` PR per verified item. All are drafts, authored by the publisher identity. A missing `kind` means `task` | lists PRs with `kind` |
| AU09 | unchanged | `channels=[…]`, plus `approvers=[…]` for `github` entries |
| AU10 | unchanged | same addition |
| AU08 | unchanged; also scans `github/` | |

## Compatibility

- Records from 002, 003, 006 and 012 have no `kind` and no `github` entries, so they pass
  unchanged (`pilot/tests/test_committed_evidence.py`).
- `github` is a third channel beside `tty` and `harness`. The detail shows the approver, so a
  reviewer can tell who decided.
