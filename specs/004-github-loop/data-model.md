# Data Model: GitHub loop

## Item issue (target repo)

| Field | Rule |
|---|---|
| title | `<id> — <item title>` |
| marker | `<!-- specfoundry:item=<id> -->` in the body; identifies the issue (never matched by title) |
| body | description, outcome, `Spec dir` link, task checklist (`- [ ]` / `- [x]` copied from `tasks.md`) |
| label | exactly one `status:<status>` label (the 9 roadmap statuses); labels are created on first sync |
| state | open until the item's PR is merged (closed by `Closes #N`) |

## Tracking issue (one per run)

| Field | Rule |
|---|---|
| title | `pilot run <run-id>` |
| marker | `<!-- specfoundry:run=<run-id> -->` |
| decision post | a comment by the publisher with `<!-- specfoundry:decision run=<id> step=<s> task=<t> n=<n> -->`, the plain-words decision (012 `describe`), and the exact command to reply with |
| closed | when the run ends (succeeded, failed or torn down) |

## GitHub approval (read, then recorded)

| Field | Source |
|---|---|
| comment_id, created_at, author | the `/approve <step>` comment |
| run, step, task, n | the decision post it follows |
| checks | author in `approvers`; author ≠ publisher identity; not app-authored; not edited; author has write+ now; no later `/withdraw` |
| recorded as | 012 `approvals/pending.json` with `channel: github`, `approver: <login>`, `comment_id` |

## Status-change PR

| Field | Rule |
|---|---|
| head | `pilot/<run-id>/status-<item>` |
| content | one commit, exactly the roadmap heading edit `[status: planned]` → `[status: verified]` |
| body | names the item issue; says the merge is the approval |
| check on merge | merged; diff equals the proposed edit; the default branch shows `verified` |

## Evidence additions (`pilot/runs/<id>/`)

- `github/issues.json`: item to issue number, and the label set at sync.
- `github/tracking.json`: tracking issue number, decision posts (comment ids).
- `github/approvals.jsonl`: every comment considered, with accept or reject and the reason.
- `github/drift.json`: the final drift report.
- `prs.json`: entries gain `kind: task|status`.
- `events.jsonl`:
  - `approval_channel … channel=github approver=<login>`;
  - `github_sync`, `decision_posted`, `status_pr_opened`, `status_pr_checked`.

## Config additions (`~/.config/specfoundry/`)

| Key | Where | Rule |
|---|---|---|
| `PILOT_GITHUB_ISSUES` | `pilot.env` | `on` / `off` (default `off`); item issues, labels and linked PRs. Added during implementation so the verified 006 walkthrough (whose token lacks Issues access) is unchanged with it off |
| `PILOT_GITHUB_APPROVALS` | `pilot.env` | `on` / `off` (default `off`); `on` implies issues on |
| `PILOT_APPROVERS` | `pilot.env` | comma-separated logins; required when approvals are on; default is the operator's login when set up |
| `PILOT_PUBLISHER_CREDENTIAL` | `pilot.env` | `token` (default) or `app`; both configured and no choice means refuse |
| `PILOT_APP_ID`, `PILOT_APP_INSTALLATION_ID` | `pilot.env` | App mode only |
| `github-app.pem` | file, mode 600 | App private key; never read into Python, never copied to the cluster |

## State changes

`paused(awaiting-mode)` and `paused(awaiting-retry)` are unchanged; with GitHub approvals on, a
decision post is added. `paused(awaiting-merge)` gains a sub-state `awaiting-status-merge`
while the status-change PR is open (`run.json` `pending.status_pr`).
