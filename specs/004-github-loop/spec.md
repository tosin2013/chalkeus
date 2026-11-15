# Feature Specification: GitHub loop — issues, labels, linked PRs and GitHub approvals

**Feature Branch**: `004-github-loop`

**Created**: 2026-11-15

**Status**: Draft

**Input**: User description: "004 M3 GitHub loop (roadmap entry 004). Tie roadmap entries, issues and PRs together on GitHub: one issue per task, labels mirror spec-status state, PRs carry Closes #N, and reconcile after merge shows no drift. Also in scope: GitHub approvals for the human decisions (mode, retry, approving a status change; merge is already on GitHub), usable in laptop and cluster mode (C-11), reusing 012's pending-step approval record and adding a `github` channel to AU09/AU10; and a GitHub App as a second publisher credential beside the fine-grained token (chosen per install; the App mints a 1-hour installation token per run scoped to the target repo with contents and pull-request write; both pass the same write probe and P09). Out of scope: Bitbucket and other SCMs. Governed by C-01, C-04, C-11."

## Background

Today the pilot's link to GitHub is narrow:
- it opens one draft PR per task;
- the human merges it;
- the operator answers every decision at a terminal, or through an agent harness (012).

Nothing on GitHub shows which roadmap item a PR serves, what state the item is in, or what
the run is waiting for. A reviewer has to read the evidence folder to find out.

012 showed the harness channel's limit: it is only as strong as the harness's permission
rule and permission mode, and an agent in auto mode can run an approval without the human
seeing it. Approving on GitHub ties each decision to the human's own GitHub identity. It is
the stronger channel the roadmap calls for, and the only one available in cluster mode (011),
where there is no terminal or harness.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Follow a run on GitHub (Priority: P1)

An operator starts a run. On the target repository, each roadmap task has an issue, and each
issue carries a label that shows the item's current status. When the pilot publishes a task's
draft PR, the PR says which issue it closes. After the human merges and the status change is
approved, the issue is closed, the label shows the new status, and a reconcile check finds no
difference between the roadmap, the issues and the PRs.

**Why this priority**: this is the core outcome of roadmap 004. Without it, the run's progress
is visible only in the evidence folder.

**Independent Test**: on a set-up machine, run the pilot against the fixture repository with
terminal approvals. Check that the issues, labels and PR links match the roadmap at every
step, and that the post-merge reconcile reports no drift.

**Acceptance Scenarios**:

1. **Given** a target roadmap with tasks and no issues, **When** the operator syncs the roadmap
   to GitHub, **Then** exactly one issue exists per task, each labelled with its item's status.
2. **Given** the sync already ran, **When** it runs again with no roadmap change, **Then** no
   issue or label is created, duplicated or changed.
3. **Given** a task's draft PR is published, **When** a reviewer opens it, **Then** it names the
   issue it closes, and merging it closes that issue.
4. **Given** A's PR is merged and its status change approved, **When** reconcile runs, **Then**
   the roadmap, the issue state and the labels agree, and the drift report is empty.
5. **Given** someone edits a status label by hand on GitHub, **When** the next sync or reconcile
   runs, **Then** it reports the difference and restores the label from the roadmap. The
   roadmap stays the source of truth.

---

### User Story 2 - Approve a decision on GitHub (Priority: P1)

A run pauses at a decision (run mode or retry). The pilot posts what is being decided where
the human will see it on GitHub. The human approves there, under their own GitHub account.
The run continues past exactly that decision, and the audit records that the decision was
approved on GitHub, and by whom.

**Why this priority**: it is the stronger approval channel, it does not depend on a harness's
permission mode, and cluster mode needs it.

**Independent Test**: start a run from a shell with no terminal, approve the mode and the
retry on GitHub only, and check that the run continues and that AU09 and AU10 report the
`github` channel.

**Acceptance Scenarios**:

1. **Given** a run paused at the mode decision, **When** an allowed approver approves it on
   GitHub, **Then** the next resume continues past the mode decision and no further.
2. **Given** a run paused at a retry, **When** an approval for a different step, an older
   attempt or a different run is given on GitHub, **Then** it is not used and the run stays
   paused.
3. **Given** a person who is not an allowed approver approves on GitHub, **When** the run
   resumes, **Then** the approval is not used, and the pilot says why.
4. **Given** an approval was used, **When** the run pauses at a later decision, **Then** the
   earlier approval does not apply.
5. **Given** a terminal is attached, **When** a decision is asked, **Then** the terminal answer
   decides, and any GitHub approval for that decision is ignored (as in 012).
6. **Given** the decisions were approved on GitHub, **When** the run is audited, **Then** AU09
   and AU10 pass and report the `github` channel and the approver's GitHub login.

---

### User Story 3 - Approve a status change on GitHub (Priority: P2)

After the human merges task A's PR, the pilot proposes the roadmap status change for item A
(`planned → verified`). The human approves that change on GitHub. Task B's gate then opens.

**Why this priority**: it completes the loop on GitHub, but the terminal and harness channels
already cover it in laptop mode.

**Independent Test**: merge A's PR, approve the status change on GitHub only, and check that
the roadmap shows the new status, the issue and label follow, and B's gate allows.

**Acceptance Scenarios**:

1. **Given** A's PR is merged, **When** the pilot proposes the status change, **Then** the
   proposal is visible on GitHub and shows the exact roadmap change.
2. **Given** the status-change PR is open, **When** an allowed approver merges it, **Then** the
   pilot checks that the merged edit is exactly the proposed one, and B's gate allows.
3. **Given** the status-change PR was edited before the merge (any change beyond the proposed
   edit), **When** it is merged, **Then** the pilot does not treat it as an approval, and
   reports the difference.

The status change is approved by merging a status-change PR (FR-012).

---

### User Story 4 - Choose the publisher credential per install (Priority: P2)

An operator chooses, per install, between the fine-grained token (the default) and a GitHub
App. With the App, each run gets a short-lived token for the target repository only, and its
PRs show the App as their author. Both credentials pass the same checks.

**Why this priority**: teams and cluster mode should not depend on one person's token. The
token stays the simplest path for a single operator.

**Independent Test**: set up the App on the fixture repository, run the pilot end to end, and
check that the PRs are authored by the App, that the write probe passes on the fixture and
fails on a canary repository, and that P09 and the audit pass.

**Acceptance Scenarios**:

1. **Given** no App is configured, **When** the operator runs the pilot, **Then** it uses the
   fine-grained token exactly as today.
2. **Given** an App is configured for the target repository, **When** a run publishes, **Then**
   the PR's author is the App, and the token used expires within one hour.
3. **Given** the App could also write to another repository, **When** a run requests its
   token, **Then** the token is limited to the target repository, and the write probe shows
   the canary is not writable.
4. **Given** either credential, **When** the checks run, **Then** only the publisher holds the
   credential (P09), and the audit's credential checks pass.
5. **Given** the App's private key, **When** a run executes in laptop mode, **Then** the key
   never enters the cluster or the evidence; only the short-lived token does.

---

### Edge Cases

- **Out of step with the roadmap:** an issue exists for a task that is no longer in the
  roadmap, or was closed by hand. Sync reports it and never deletes it.
- **Rate limits or GitHub outages:** sync stops with a clear error and changes nothing
  partially. A run never treats "GitHub unreachable" as "approved" (fails closed).
- **Approver changes their mind:** the approval is withdrawn before the run resumes. A
  withdrawn approval is not used.
- **Duplicates:** two approvals arrive for the same decision. Only one is used; the other is
  reported.
- **Approvals on PRs:** the approval is given on a PR or issue that belongs to a different run.
  It is not used.
- **Expired App token mid-run:** a fresh token is minted for the publish step. An expired token
  is never retried silently as a success.
- **Token and App both configured:** the operator's choice decides. If no choice is recorded,
  the pilot refuses to guess and asks for one.
- **Fork PRs or outside contributors:** comments from accounts without the approver right are
  ignored for approval purposes.

## Requirements *(mandatory)*

### Functional Requirements

**Issues, labels and links**

- **FR-001**: The operator MUST be able to sync the target roadmap to GitHub, creating exactly
  one issue per task, idempotently (a re-run with no change creates or changes nothing).
- **FR-002**: Each task issue MUST carry one status label that mirrors the item's recorded
  status. Sync and reconcile MUST restore a label that differs from the roadmap, and report
  that they did so.
- **FR-003**: Every draft PR the publisher opens MUST name the task issue it closes, so that
  merging the PR closes the issue.
- **FR-004**: After a merge and an approved status change, reconcile MUST check the roadmap,
  issue state, labels and PR links against each other and report any drift. A complete run
  ends with an empty drift report.
- **FR-005**: Scheduling inputs stay the roadmap and the status tool's output (C-01). Issues and
  labels are a projection of the roadmap and MUST NOT be read as a source of scheduling
  decisions.

**GitHub approvals**

- **FR-006**: For each decision the run waits on (mode, retry, status change), the pilot MUST
  show on GitHub what is being decided and how to approve it, before any approval is
  accepted.
- **FR-007**: An approval on GitHub MUST be bound to its run, step, task, attempt number and,
  for a status change, the merge commit. It MUST be used at most once, and a mismatched or
  withdrawn approval MUST NOT be used.
- **FR-008**: Only allowed approvers MAY approve. Allowed approvers are the GitHub logins
  listed in the operator's configuration (default: the operator alone). When an approval is
  read, each approver MUST still have write access to the target repository; otherwise the
  approval is not used.
- **FR-009**: A human approves by replying with an approval command (naming the step) on the
  run's tracking issue. The pilot opens one tracking issue per run and posts each pending
  decision there. A comment made with the publisher credential MUST never count as an
  approval.
- **FR-009a**: The documentation and the `pilot` skill MUST state the limit of this channel. An
  approval binds to a GitHub identity, so anyone able to act as that identity can approve:
  - in token mode the publisher token is the operator's own account;
  - an agent on the operator's machine may hold the operator's GitHub login.

  Agents MUST NOT post approval commands, and the GitHub App credential is recommended when
  GitHub approvals are used.
- **FR-010**: GitHub approvals MUST reuse 012's pending-step approval record and rules (one
  pending step, bound, used once; the terminal decides when present), and add a `github`
  channel.
- **FR-011**: The audit's retry check (AU09) and mode check (AU10) MUST report the `github`
  channel and the approver's GitHub login for each decision approved on GitHub. Their pass
  rules stay unchanged, and evidence from earlier runs still passes.
- **FR-012**: On GitHub, a status change is approved by merging a status-change PR. After A's
  PR is merged, the publisher opens a draft PR containing exactly the roadmap status edit; the
  human's merge of that PR is the approval (C-04). The pilot MUST then check that the merged
  edit is exactly the proposed one before unblocking B. With this channel, no direct push of
  the roadmap by the operator is needed.
- **FR-013**: A merge MUST NOT be approvable through the pilot. Merges stay a human action on
  GitHub (C-04).
- **FR-014**: GitHub approvals MUST work in laptop mode now. The record and channel MUST NOT
  depend on a local terminal or harness, so cluster mode (011) can reuse them unchanged.
- **FR-015**: If GitHub cannot be reached or an approval cannot be verified, the run MUST stay
  paused (fail closed).

**Publisher credential**

- **FR-016**: The operator MUST be able to choose, per install, between a fine-grained token (the
  default) and a GitHub App as the publisher credential.
- **FR-017**: With the App, each run MUST use an installation token that expires within one
  hour and is limited to the target repository, with only contents and pull-request write.
- **FR-018**: Both credentials MUST pass the same write probe (fixture writable, canary not)
  before a run publishes, and the same checks that only the publisher holds a credential (P09).
- **FR-019**: In laptop mode the App's private key MUST stay on the host. Only the short-lived
  token may enter the cluster, and neither may appear in the evidence (AU08).
- **FR-020**: Setup documentation MUST cover creating and installing the App, choosing the
  credential, and the checks that confirm it. A token or key that fails its check MUST NOT be
  saved (as in 006 F3).

**Out of scope**

- **FR-021**: Other source-control hosts (Bitbucket and others) are out of scope.

### Key Entities

- **Task issue**: the GitHub issue for one roadmap task. It links to its item and carries one
  status label.
- **Status label**: a label naming a roadmap status, mirrored from the roadmap.
- **Linked PR**: a publisher draft PR that names the task issue it closes.
- **Drift report**: the differences between the roadmap, the issues, the labels and the PRs.
- **Tracking issue**: one issue per run where the pilot posts each pending decision and
  approvers reply with approval commands.
- **GitHub approval**: an approval command from an allowed approver on the tracking issue (mode,
  retry), or a human merge of the status-change PR (status change), bound to that decision
  and used once.
- **Status-change PR**: a draft PR containing exactly one roadmap status edit, opened after A's
  merge.
- **Approver**: a GitHub login listed in the operator's configuration that still has write
  access to the target repository (FR-008).
- **Publisher credential**: either the fine-grained token or the GitHub App, chosen per
  install.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A full run on the fixture repository ends with:
  - one issue per task;
  - labels matching the roadmap;
  - every published PR linked to its issue;
  - an empty drift report after reconcile.
- **SC-002**: Re-running the sync with no roadmap change creates or changes 0 issues and 0 labels.
- **SC-003**: A run with no terminal and no harness completes with every decision approved on
  GitHub, and the audit passes all 11 checks, reporting the `github` channel and the approver
  for the mode, each retry and the status change.
- **SC-004**: In every refusal case (wrong step, wrong run, stale attempt or merge commit,
  non-approver, withdrawn approval, a comment made with the publisher credential, an edited
  status-change PR, GitHub unreachable), the run does not advance: 0
  decisions are taken without a matching approval by an allowed approver.
- **SC-005**: A run with the GitHub App publishes PRs authored by the App, using a token that
  expires within one hour and cannot write to the canary repository. The audit passes all 11
  checks.
- **SC-006**: Evidence from earlier runs (002, 003, 006, 012) still passes the audit.
- **SC-007**: An operator can set up the GitHub App from the documentation alone, in under 30
  minutes, with 0 outside lookups.

## Assumptions

- The target repository is the throwaway fixture repository used since 002. Its roadmap and
  tasks follow the same structure.
- Issue creation is the pilot's own deterministic step. The deprecated Spec Kit
  `taskstoissues` skill is the model for what an issue holds, not a dependency.
- The roadmap stays the source of truth; GitHub shows a copy (C-01, constitution I).
- 012's approval record and the `tty` and `harness` channels stay as they are; `github` is
  added beside them.
- The fine-grained token stays the default publisher credential (roadmap 1.6.0 decision); the
  App is optional and recommended for teams and cluster mode.
- Owner decisions 2026-11-15:
  - approvals are command comments on a per-run tracking issue;
  - approvers are an explicit list in the operator's config;
  - a status change is approved by merging a status-change PR.
- Cluster mode (011) is not built here. This feature only keeps the approval record and
  channel free of laptop-only assumptions.
