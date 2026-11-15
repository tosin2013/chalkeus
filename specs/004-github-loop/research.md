# Research: GitHub loop

Owner clarifications on 2026-11-15 resolved the spec's open questions. These are the
remaining design decisions.

## R1 — Issue granularity and content

- **Decision:** one issue per roadmap item (owner, 2026-11-15). Its body is regenerated from
  the roadmap and `spec status --json`:
  - the item's description and outcome;
  - its `Spec dir` link;
  - the task checklist from the item's `tasks.md`.

  A hidden marker (`<!-- specfoundry:item=<id> -->`) identifies the issue. Sync updates the
  body only when it differs.
- **Rationale:** status belongs to the item, and a PR delivers one item, so label and
  `Closes #N` map one to one. This settles roadmap Q4.
- **Alternatives:** sub-issues per task; Spec Kit's `taskstoissues` (deprecated, and
  per-checkbox).

## R2 — Who writes to GitHub, and with what credential

- **Decision:** all pilot writes use the **publisher credential**: issues, labels, the tracking
  issue, decision posts, and the status-change PR. They run in a child process with the token
  sourced from `publisher.env` (or the App token file), the same pattern as `check_token.py`,
  or in the publisher Job for PRs. **Reads** (comments, PR state, collaborator permission) use
  the operator's `gh`, as today.
- **Rationale:**
  - pilot posts must be distinguishable from human approvals. With the App they are authored
    by `<app>[bot]`; with the token they carry the publisher marker;
  - the operator's `gh` doesn't exist in cluster mode, while the publisher credential does;
  - the orchestrator never holds the value.
- **Cost:** the token and App need `issues: write` in addition to contents and pull requests
  (spec FR-017 amended). The write probe checks it.
- **Alternatives:** the operator's `gh` for writes, which puts the human's identity on bot
  posts and doesn't work in cluster mode; a separate third credential, which adds setup for
  little gain.

## R3 — Reading an approval

- **Decision:** on `resume` with no terminal, while the run is paused at `mode` or `retry`, the
  pilot:
  1. lists comments on the run's tracking issue created after the decision post;
  2. keeps comments whose first line is exactly `/approve <step>` and whose author is in
     `approvers`;
  3. drops comments made by the publisher identity, made through any GitHub App, or edited
     after creation;
  4. checks the author still has `write`, `maintain` or `admin` on the target repo;
  5. takes the **earliest** valid comment.

  A later `/withdraw <step>` from the same author cancels it. The binding (run, step, task, n)
  comes from the decision post, which carries a hidden marker. A valid approval is written
  through 012's `approvals.record` with `channel=github` and `approver=<login>`, then used by
  `take()` as today.
- **Rationale:** reuses 012's single-use, bound record. The comment id and timestamp go into
  the evidence. Ignoring edited comments prevents rewriting an old comment into an approval.
- **`pilot github watch --run <id>`:** an optional helper that polls every 15 s and resumes when
  an approval appears, so the human doesn't have to come back to the shell.
- **Alternatives:** webhooks, which need a public endpoint and belong in cluster mode (011);
  reactions (👍), which are too easy to give by accident and carry no step name.

## R4 — The identity limit (FR-009a)

- **Decision:** document plainly:
  - **token mode:** the publisher token is the operator's account, and an agent on the
    machine may hold the operator's `gh` login, so it could comment as them. Mitigations:
    - publisher-authored comments never count (marker plus identity check);
    - the skill and `AGENTS.md` forbid agents from posting `/approve`;
    - the Claude Code settings add an ask or deny rule for `gh issue comment` and
      `gh api … comments`;
    - the App is recommended;
  - **App mode:** pilot posts are the bot's. A human comment needs the human's credential,
    which the pilot never holds.

  The `github` channel is still stronger than `harness`: it is tied to a named identity,
  visible to everyone on the repo, and checked against an approver list.
- **Alternatives:** require the App for GitHub approvals. Rejected as a hard rule, because the
  owner keeps the token path simple (roadmap 1.6.0). The audit detail shows the credential
  mode, so a reviewer can weigh it.

## R5 — GitHub App token with the stdlib only

- **Decision:** `ghapp.py` builds the JWT header and payload (iat −60 s, exp +9 min, iss = App
  id) and signs it by piping to `openssl dgst -sha256 -sign <key>` in a child process. The
  key file is mode 600 and is never read into Python. It then:
  - calls `POST /app/installations/{id}/access_tokens` with `repositories: [<repo name>]` and
    permissions `{contents: write, pull_requests: write, issues: write}`;
  - writes the returned token to a mode-600 temporary env file;
  - uses that file in place of `publisher.env` for that run, deleting it at teardown.

  The token is minted per run, and again if expired at publish time. In-cluster, the
  `scm-token` Secret gets the short-lived token only, so P09 is unchanged.
- **Rationale:** no new Python dependency (stdlib-only rule); `openssl` is present on the
  target platform.
- **Alternatives:** the `cryptography` package (a new dependency); `gh` with App auth (not
  built in).

## R6 — Status-change PR

- **Decision:** after A's merge passes today's checks (merged, tasks 100%, tree matches the
  graded attempt), the pilot builds a one-commit bundle with exactly the roadmap status edit
  on top of the merge commit. It passes the bundle to the publisher Job as kind `status`
  (head `pilot/<run>/status-<item>`, draft, body naming the item issue). When it's merged,
  reconcile checks:
  - the PR is merged;
  - its diff against its base equals the proposed one-line edit;
  - the default branch now shows the item `verified`.

  Then it records `task_verified` and B proceeds. Any extra change means `STATUS_PR_EDITED`:
  no approval, the run stays paused. With a terminal or harness approval, today's direct-push
  path remains available (laptop mode); the GitHub path is used when GitHub approvals are on.
- **Rationale:** a human merge is the strongest approval (C-04), and it needs no operator git
  credentials (cluster mode).
- **Alternatives:** an `/approve reconcile` comment plus a direct push, which needs the
  operator's credentials.

## R7 — AU03 and evidence compatibility

- **Decision:** `prs.json` entries gain `kind` (`task` default, or `status`). AU03 now passes
  when:
  - there are exactly 2 `task` PRs (A, B);
  - there are 0 or 1 `status` PRs per verified item;
  - every PR is a draft, authored by the publisher identity.

  Old records have no `kind` and no status PRs, so they pass unchanged. AU09 and AU10 detail
  adds `approvers=[…]` for `github` entries.
- **Alternatives:** a new AU12 for status PRs. Rejected, because AU03 is already the PR check.

## R8 — Drift report

- **Decision:** `pilot github drift` (also run at the end of reconcile) lists:
  - missing item issues;
  - a label that differs from the roadmap;
  - an open issue for a `verified` item, or a closed issue for an unverified one;
  - a task PR without `Closes #N`;
  - an issue for an item no longer in the roadmap.

  `github sync` fixes labels and bodies and creates missing issues. It never deletes or closes
  anything except via the PR merge, and never reopens. It reports the rest. The run's evidence
  stores `github/drift.json`; SC-001 needs it empty at the end.
