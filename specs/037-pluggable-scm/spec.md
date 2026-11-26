# Feature Specification: Pluggable SCM, phase 1: GitLab and Bitbucket Cloud beside GitHub

**Feature Branch**: `spec-037-pluggable-scm`

**Created**: 2026-11-26

**Status**: Draft

**Input**: Roadmap entry 037 (added 2026-11-26, roadmap 1.41.0), from issue #144
(https://github.com/tosin2013/chalkeus/issues/144), PRD F11 ("Pluggable SCM. GitHub first;
Bitbucket later"). Let targets, companions and the fixture live on GitLab (gitlab.com and
self-managed) or Bitbucket Cloud as well as GitHub, with the same guarantees, so teams on those
hosts can use the laptop release (013) without mirroring to GitHub. Bitbucket Cloud comes
first, then gitlab.com, then self-managed GitLab, in this item (owner decision 2026-11-26,
reordered from GitLab first because a Bitbucket target is waiting). Phase 2 (038: issues, labels and comment approvals on the other
hosts) is out of scope.

Governed by C-03 (isolation and credentials), C-04 (human merge only), C-07 (egress through
per-role allowlist proxies), C-11 (two modes, one set of guarantees) and C-14 (hidden tests stay
out of every build worker's reach).

## Context

Every interaction with a Git host goes to GitHub today. Where:

| Area | What it does on GitHub | Who holds the credential |
|---|---|---|
| Publisher Job (`publish.py`) | pushes the graded commit to the item branch, opens a draft PR, checks for an open PR for the item | the publisher: a fine-grained token for one repository, or a GitHub App installation token |
| Token check (`check_token.py`) | proves the publisher token reaches only its one repository | the publisher |
| Publisher read-back (Go) | re-reads the PR: draft, head branch | the operator (their own `gh` login or `GH_TOKEN`) |
| Reconcile (Go) | merged state, merge commit, merged tree equals the graded attempt | the operator |
| Status-change PR (Go) | who merged it (an approver, never the publisher), only the one roadmap line changed | the operator |
| Companion PR (test author, 018/024) | pushes a draft of hidden tests, opens a PR on the companion repository | the test-author identity (`author.env`) |
| GitHub loop (004, `ghops.py`) | issues, labels, `Closes #N`, approvals by comment | the publisher and the operator |
| Debt report (029) | reads the target's open Dependabot PRs | the operator |
| doctor | token scope, branch protection, approvers' write access | the operator and the publisher |
| Egress | the publisher's allowlist proxy allows `github.com` and `api.github.com` | — |
| Config | repositories are `owner/name`; `FORGE_APPROVERS` lists GitHub logins | — |

What doesn't depend on the host: the gate, workers, the grader, hidden-test handling inside the
cluster, and the **tree-match check** (it compares files, so it works for merge commits, squash
and fast-forward). The PR title format and the `Specfoundry-Run:` commit trailer carry over.

Terms:
- **Host:** the Git hosting service a repository lives on: GitHub, GitLab (gitlab.com or a
  self-managed instance) or Bitbucket Cloud.
- **Change request:** GitHub's and Bitbucket's pull request, GitLab's merge request. "PR" in
  forge's messages means a change request on any host.
- **Bot identity:** the account the publisher's credential acts as on the host (GitHub App bot,
  GitLab project access token bot user, Bitbucket repository access token bot).
- **Stable account id:** a host's permanent id for an account (GitLab numeric user id,
  Bitbucket account UUID); GitHub logins count as stable, as today.
- **Operator credential:** the human operator's own login for a host, used by forge's reads on
  the operator's machine (today `gh auth token` or `GH_TOKEN`).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - GitHub works exactly as before (Priority: P1)

An operator whose targets and fixture are on GitHub upgrades and changes nothing. Every run,
check and audit behaves as before, because GitHub is now one host behind a common interface
rather than a special case.

**Why this priority**: every existing user is on GitHub; moving the host operations behind one
interface must not change their runs, evidence or guarantees.

**Independent Test**: with no host setting anywhere, run the full test suite, the isolation
suite, the fixture walkthrough and one GitHub target item run; results and audits match the
evidence from before this item.

**Acceptance Scenarios**:

1. **Given** profiles and `forge.env` with no host setting, **When** forge runs anything,
   **Then** the host is GitHub and behaviour is unchanged.
2. **Given** the fixture walkthrough on GitHub, **When** it completes, **Then** the audit is
   11/11 and the evidence matches the previous evidence apart from the recorded host, the
   account ids next to logins, and the new publisher-egress check.
3. **Given** a GitHub target with the GitHub loop on (issues, labels, comment approvals),
   **When** a run goes through its pauses, **Then** the loop works as before.

---

### User Story 2 - Build an item on gitlab.com (Priority: P2)

An operator whose code is on gitlab.com sets the target's host to GitLab, gives the publisher a
project access token for that one project, and runs an item. The worker builds as usual; the
publisher pushes the graded commit and opens a draft merge request as its bot user; the operator
reviews and merges it on GitLab; forge checks the merge (merged, the merge or squash commit, the
merged tree equals the graded attempt) and proposes the status change as a second merge
request, which an approver merges.

**Why this priority**: the second new host; it proves the interface on a host with a different
API shape (merge requests, nested groups, project tokens) after Bitbucket.

**Independent Test**: against a throwaway gitlab.com project, run the fixture walkthrough (audit
11/11) and one target item run end to end, including the status change.

**Acceptance Scenarios**:

1. **Given** a target profile with the host set to GitLab and a project path
   `group/subgroup/project`, **When** the operator runs the gate and an item run, **Then** forge
   uses that project on gitlab.com for every host operation.
2. **Given** a graded attempt, **When** the publisher runs, **Then** it pushes the item branch and
   opens a draft merge request authored by the project token's bot user, with the same title
   format and run trailer as on GitHub.
3. **Given** an open merge request for the item already exists, **When** a new publish would
   open another, **Then** forge refuses as it does on GitHub.
4. **Given** the operator merged the merge request with a merge commit, a squash or a
   fast-forward, **When** reconcile runs, **Then** it records the commit that landed and passes
   only if the merged files equal the graded attempt.
5. **Given** the merge request was closed without merging, **When** reconcile runs, **Then** it
   reports it as not merged.
6. **Given** a status-change merge request merged by a listed approver with only the one roadmap
   line changed, **When** forge checks it, **Then** the status change is accepted.
7. **Given** a completed run on gitlab.com, **When** the audit runs, **Then** it is 11/11 and the
   evidence records the host.

---

### User Story 3 - The approval and credential guarantees hold on every host (Priority: P1)

Whatever the host, only the publisher holds a write credential, it is scoped to one repository,
no host token reaches a worker or the grader, the publisher reaches only its host, and a status
change counts only when a listed human approver merged a change request that changes only the
one roadmap line.

**Why this priority**: these guarantees are the project's reason to exist; a second host that
weakens any of them is worse than no second host.

**Independent Test**: on each new host, try each refusal case (bot merges the status change, a
non-approver merges it, it carries an extra change, a token with wider scope, egress to another
host) and confirm each is refused and recorded.

**Acceptance Scenarios**:

1. **Given** a status-change request merged by the publisher's bot identity, **When** forge checks
   it, **Then** it refuses it, as GitHub's `STATUS_PR_MERGED_BY_PUBLISHER` does.
2. **Given** a status-change request merged by someone not in `FORGE_APPROVERS`, or by an approver
   without write access, **When** forge checks it, **Then** it refuses it and names the reason.
3. **Given** a status-change request that changes anything besides the one roadmap line, **When**
   forge checks it, **Then** it refuses it.
4. **Given** a publisher credential that reaches more than one repository (a GitLab group
   token, a personal token of an account with more than one membership, a Bitbucket project or
   workspace token), **When** doctor or the token check runs, **Then** it refuses the credential
   and says which kind is required.
5. **Given** a run on any host, **When** the egress checks run, **Then** the publisher reaches its
   own host and is refused every other Git host, and workers and the grader reach no Git host.
6. **Given** any run, **When** the audit checks the worker and grader evidence, **Then** no host
   token appears in it (P11 and AU04 unchanged).
7. **Given** `FORGE_APPROVERS` with usernames, **When** forge loads it for GitLab or Bitbucket,
   **Then** it resolves them to stable account ids, records the ids, and compares merges by id
   (Bitbucket can't resolve usernames, so a Bitbucket list must name account ids).

---

### User Story 4 - A self-managed GitLab (Priority: P3)

An enterprise operator points a target at their own GitLab instance by host name, supplies the
corporate certificate authority if the instance uses one, and runs items exactly as on
gitlab.com.

**Why this priority**: self-managed GitLab is common in the enterprises the laptop release and
OpenShift direction (005, 011) aim at, but it builds on US2.

**Independent Test**: against one self-managed GitLab instance, run the fixture walkthrough
(audit 11/11) and one target item run.

**Acceptance Scenarios**:

1. **Given** a profile with the host set to GitLab and a host name, **When** forge runs, **Then**
   every host operation and the publisher's egress allowlist use that host name, and gitlab.com
   is not reachable from the publisher.
2. **Given** an instance with a certificate from a corporate authority and that authority's
   bundle supplied in the profile, **When** the publisher and forge's reads connect, **Then** they
   trust it; without the bundle they refuse with a message naming the setting.
3. **Given** an instance whose version lacks an operation forge needs, **When** doctor runs,
   **Then** it fails and names the minimum supported version.

---

### User Story 5 - Build an item on Bitbucket Cloud (Priority: P1)

An operator whose code is on Bitbucket Cloud sets the target's host to Bitbucket, gives the
publisher a repository access token, and runs an item. The publisher pushes the graded commit
and opens a draft pull request as the token's bot identity; the operator merges it on Bitbucket;
forge checks the merge (merged, the merge commit, the merged tree equals the graded attempt) and
proposes the status change as a second pull request, which an approver merges.

**Why this priority**: the first new host (owner decision 2026-11-26: a Bitbucket target is
waiting); it proves the host interface first.

**Independent Test**: against a throwaway Bitbucket Cloud repository, run the fixture walkthrough
(audit 11/11) and one target item run end to end, including the status change.

**Acceptance Scenarios**:

1. **Given** a profile with the host set to Bitbucket and a repository `workspace/repo`, **When**
   the operator runs an item, **Then** the publisher opens the change request as the repository
   token's bot identity, as a draft.
2. **Given** a Bitbucket target branch without a restriction that stops the publisher's token
   from merging, **When** doctor runs, **Then** it fails and says to add one (a repository token
   that can open pull requests can also merge them).
3. **Given** a pull request declined or superseded, **When** reconcile runs, **Then** it counts as
   not merged.
4. **Given** a merged pull request, **When** reconcile and the status-change check run, **Then**
   they read the landed commit and who merged it, and pass or refuse as on GitHub and GitLab.

---

### User Story 6 - Hidden tests arrive by change request on the same host (Priority: P1)

The test author (a separate session, 018/024) drafts an item's hidden tests and opens a draft
change request on the companion repository, which lives on the same host as its target; a human
reviews and merges it.

**Why this priority**: a target item run needs hidden tests in the companion; without this the
operator must push them by hand on the new hosts.

**Independent Test**: with a companion on Bitbucket Cloud (then gitlab.com), submit a checked
draft; a draft change request opens from the test-author identity.

**Acceptance Scenarios**:

1. **Given** a checked draft and a companion on the target's host, **When** the test author
   submits it, **Then** a draft change request opens on the companion from the test-author
   identity, never from the publisher's bot.
2. **Given** auto-draft (024) on a target on a new host, **When** the watcher drafts, **Then** it
   finds open drafts and earlier drafts on that host as it does on GitHub.

---

### User Story 7 - Set up and check a new host (Priority: P3)

An operator follows the guide to set up a target on GitLab or Bitbucket Cloud (credentials,
approver ids, host name, certificate bundle) and runs doctor, which checks each piece before
any run.

**Why this priority**: correct setup prevents most failures, but the runs themselves come first.

**Independent Test**: follow only `docs/targets.md` for a GitLab and a Bitbucket target; doctor
passes; break each setting in turn and doctor names it.

**Acceptance Scenarios**:

1. **Given** a target on a new host, **When** doctor runs, **Then** it checks: the operator
   credential for that host, the publisher credential's kind and scope, the approvers' ids and
   write access, branch protection on the target branch, and that the publisher's commit author
   email is a valid address.
2. **Given** the publisher's commit email is not a valid address (for example `*.invalid`),
   **When** doctor runs, **Then** it warns that some hosting and CI integrations refuse such
   commits and names the setting.
3. **Given** a target on a new host with the GitHub loop, GitHub approvals or the debt report
   requested, **When** the profile loads or the command runs, **Then** forge refuses with a
   message that these are GitHub-only in this release (038 for the loop).

---

### Edge Cases

- A GitLab merge request merged by squash: the landed commit is the squash commit, and the tree
  match decides.
- A fast-forward merge: there is no merge commit; the landed commit is the branch head, and the
  tree match decides.
- An approver renamed their account: ids decide, so the approval still counts; the recorded
  username may differ from the current one.
- A GitLab merge request in a `locked` state, or a Bitbucket pull request `superseded`: not
  merged.
- The project moved or renamed on the host: forge reports it can't find the repository and
  names the profile setting; it doesn't follow redirects silently to a different project.
- A GitLab project in nested subgroups (`a/b/c/project`): accepted; the path is the project's
  full path.
- An operator with no credential for the target's host on their machine: doctor and reads fail
  with a message naming how to provide one; forge never falls back to the publisher's
  credential for reads.
- The fixture and a target on different hosts: each uses its own host; nothing is shared.
- The companion on a different host from its target: refused in this release, with a message.
- A self-managed instance reachable only through a corporate proxy: out of scope beyond the
  existing proxy settings; doctor reports the connection failure.
- gitlab.com's Free plan has no project access tokens: a personal token of a dedicated bot
  account is accepted when that account is a member of the one project only, proven before every
  run (owner decision 2026-11-26).
- An operator credential without repository admin on Bitbucket: forge can't read the approvers'
  effective permissions, so doctor fails that check rather than assume.
- A host's API rate limit: forge reports it and stops, as on GitHub; it doesn't retry in a loop.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: A target profile MUST be able to choose its host (`github`, `gitlab` or
  `bitbucket`); `forge.env` MUST be able to do the same for the fixture. With no setting the
  host MUST be GitHub and behaviour MUST be unchanged.
- **FR-002**: For GitLab, a profile MUST be able to name a self-managed instance by host name;
  without one, gitlab.com is used.
- **FR-003**: Repository names MUST accept `group/subgroup/…/project` for GitLab and
  `workspace/repo` for Bitbucket, and refuse other forms with a message.
- **FR-004**: Every host operation forge uses (push the item branch, open a draft change request,
  find an open request for an item, read its state, the landed commit, who merged it and its
  changed files, check an account's write access, read branch protection) MUST work on GitHub,
  GitLab and Bitbucket Cloud with the same meaning, through one interface.
- **FR-005**: The publisher MUST open item change requests as drafts, using each host's native
  draft state.
- **FR-006**: Reconcile MUST treat only a merged change request as merged (GitLab `closed`
  and `locked`, Bitbucket `declined` and `superseded` are not), record the landed commit (merge,
  squash or fast-forward head), and pass only when the merged files equal the graded attempt.
- **FR-007**: The status-change check MUST accept a merge only when the merger is a listed
  approver with write access, is not the publisher's bot identity, and the request changes only
  the one roadmap line, on every host.
- **FR-008**: `FORGE_APPROVERS` MUST be compared by stable account id on GitLab and Bitbucket.
  On GitLab usernames MUST be resolved to ids and the ids recorded in the evidence; on Bitbucket
  the list MUST name account ids.
- **FR-009**: The publisher MUST be the only holder of a write credential, and that credential
  MUST be scoped to one repository: a GitLab project access token, or a personal token of a
  dedicated bot account that is a member of that one project only; a Bitbucket repository access
  token (GitHub as today). Credentials that reach more than one repository MUST be refused before
  any run. On Bitbucket, where the token can also merge, doctor MUST require a branch restriction
  that stops it.
- **FR-010**: No host credential MUST reach a worker or the grader (C-03; P11 and AU04
  unchanged).
- **FR-011**: The publisher's egress allowlist MUST contain only the chosen host's Git and API
  endpoints; P-checks MUST prove the publisher is refused the other hosts.
- **FR-012**: For a self-managed GitLab, a profile MUST be able to supply a certificate authority
  bundle that the publisher and forge's reads trust; without it, a certificate failure MUST stop
  with a message naming the setting.
- **FR-013**: forge's reads on the operator's machine MUST use the operator's own credential for
  the host, never the publisher's; a missing operator credential MUST stop with a message saying
  how to provide one.
- **FR-014**: The test author's companion change request MUST open on the target's host from the
  test-author identity, as a draft; the companion MUST be on the same host as its target.
- **FR-015**: The GitHub loop (issues, labels, comment approvals), GitHub approvals, the GitHub
  App credential and the debt report MUST stay GitHub-only; asking for them on another host MUST
  be refused with a message.
- **FR-016**: doctor MUST check, for a target on any host: the operator credential, the publisher
  credential's kind and scope, the approvers' ids and write access, branch protection on the
  target branch, the instance's version (self-managed GitLab), and that the publisher's commit
  author email is a valid address.
- **FR-017**: Every run's evidence MUST record the host (and instance host name for
  self-managed GitLab), and each status-change and reconcile record MUST name the merger by
  stable id.
- **FR-018**: Mode, retry and acceptance decisions MUST stay in the terminal or harness on GitLab
  and Bitbucket (phase 2 adds host comments, 038).
- **FR-019**: The operator guide and `docs/targets.md` MUST document each host's setup
  (credentials, approver ids, host name, certificate bundle, the Bitbucket merge restriction), and the
  troubleshooting guide MUST cover each new refusal.

### Key Entities

- **Host setting:** a target's (or the fixture's) host kind and, for self-managed GitLab, its
  host name and certificate bundle.
- **Change request record:** a request's host, id, state (open, merged, not merged), draft
  flag, head branch, landed commit and merger's stable id.
- **Publisher credential:** host kind, credential kind (project, repository or fine-grained
  token, GitHub App), the one repository it reaches, and its bot identity.
- **Approver:** a stable account id per host, with the username it was resolved from.
- **Operator credential:** the human's own login for a host, used only for reads on their machine.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: With no host setting, 100% of existing tests pass and the GitHub fixture evidence
  matches the evidence before this item apart from the recorded host, account ids next to logins
  and the new publisher-egress check.
- **SC-002**: On each of gitlab.com, one self-managed GitLab and Bitbucket Cloud, the fixture
  walkthrough completes with the audit at 11/11.
- **SC-003**: On each of those three, one target item run succeeds end to end: a draft request
  by the bot identity, a human merge, a tree match, and an approved status change.
- **SC-004**: On every host, 100% of the status-change refusal cases (merged by the bot, by a
  non-approver, with an extra change) are refused.
- **SC-005**: On every host, the egress checks prove the publisher reaches only its own host,
  and no host token appears in any worker or grader evidence.
- **SC-006**: An operator new to Chalkeus sets up a GitLab or Bitbucket Cloud target by following
  only the docs, and doctor passes before the first run.
- **SC-007**: A reviewer can tell from any run's evidence alone which host it used and who
  merged each change request, by stable id.

## Assumptions

- The owner provides, for the live proof, throwaway projects or repositories and the matching
  tokens on gitlab.com and Bitbucket Cloud. The self-managed proof uses a throwaway GitLab
  Community Edition container at a pinned version, with a self-signed CA (owner decision
  2026-11-26, roadmap Q14). The test-author identity on each host is a separate account or token
  from the publisher's.
- The companion repository is on the same host as its target; the fixture's host is set in
  `forge.env` independently of any target.
- Supported self-managed GitLab versions are the current and previous major releases (18.0 and
  later); older instances are refused by doctor with the minimum version.
- Bitbucket Cloud's API supports draft pull requests, and drafts can't be merged (research R3).
- The operator authenticates to GitLab and Bitbucket with that host's usual personal token or
  CLI login, read from the environment or the host's CLI, as `GH_TOKEN` and `gh` are today.
- Bitbucket Data Center/Server, Azure DevOps, webhooks, and the phase-2 loop features (038) are
  out of scope. The debt report (029) reads Dependabot, which is GitHub-only.
- Settings use the `FORGE_*` names (read as `PILOT_*` keys since 027); the issue's `PILOT_SCM`,
  `PILOT_SCM_HOST` and `PILOT_APPROVERS` become `FORGE_SCM`, `FORGE_SCM_HOST` and
  `FORGE_APPROVERS`.
