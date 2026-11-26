# Tasks: Pluggable SCM, phase 1: GitLab and Bitbucket Cloud beside GitHub

**Input**: `specs/037-pluggable-scm/` (spec.md, plan.md, research.md, data-model.md,
contracts/, quickstart.md)

**Tests**: included. A shared contract suite runs against every host implementation (fakegh for
GitHub, httptest fakes for Bitbucket and GitLab); config refusals, doctor rows, AU03, P13 and
`publish.py` have unit tests; the live proof (quickstart.md) is the evidence for each host.

**Order**: Bitbucket Cloud first, then gitlab.com, then self-managed GitLab (owner decision
2026-11-26). Phases follow the spec's priorities: US1, US5, US6, US3 (P1), US2 (P2), US4 and US7
(P3).

**Privacy**: a private Bitbucket target is waiting. Proof runs use throwaway repositories only.
A private target runs only with `FORGE_EVIDENCE_FEATURE` set to an absolute path outside this
repository, and is never named in commits, PRs, docs or evidence here.

## Phase 1: Setup

- [X] T001 Create the package tree `internal/scm/` (doc.go: "Package scm is forge's one interface to Git hosts: change requests, repositories and accounts (037)"), `internal/scm/scmtest/`, `internal/scm/github/`, `internal/scm/bitbucket/`, `internal/scm/gitlab/`, each with a doc.go; `go vet ./...` passes
- [X] T002 Extend the no-merge guard in `internal/ghloop/nomerge_test.go` (or a new `internal/scm/nomerge_test.go`): the go-github method check also scans `internal/scm/github`; a new check fails if any non-test file in `internal/scm/bitbucket` or `internal/scm/gitlab` contains a merge, approve or accept path (`/merge`, `/approve`, `/approvals`, `/unapprove`, `accept`) or calls `CloseChangeRequest` outside `reset-fixture`; `TestPilotNeverImportsTheFake` covers the new fakes (research R12)
  Done as (2026-11-26): the go-github check in `internal/ghloop/nomerge_test.go` already walks the whole module, `internal/scm/github` included. New `internal/scm/nomerge_test.go`: `TestNoMergeOrApprovePathsInHostPackages` (no merge, approve, auto-merge or accept path in any string of `scm/gitlab` and `scm/bitbucket`) and `TestCloseChangeRequestOnlyFromResetFixture` (only `internal/cli/fixture.go` may call it). The fakes are test files, so `TestPilotNeverImportsTheFake` needs no change.

## Phase 2: Foundational (blocks every user story)

- [X] T003 [P] Define the types of data-model.md in `internal/scm/scm.go`: `Kind` (`github`, `gitlab`, `bitbucket`; "empty means `github`"), `Host{Kind, Name, APIBase, GitBase, CABundle}` with `EgressHosts()` (R10), `Account{ID, Name, Bot}` ("Comparison is by ID only"; GitHub IDs lower-cased), `Permission` (`none|read|write|admin`), `Protection`, `RepoInfo{DefaultBranch, CanPush}`, `File{Path, Content}`, `ChangeRequest{ID, URL, State, Draft, HeadRef, HeadSHA, BaseRef, LandedCommit, Author, MergedBy}` with `State` "`open` | `merged` | `closed`", `TokenProbeResult` with the refusal codes `SCM_TOKEN_WIDER_THAN_REPO`, `SCM_TOKEN_WRONG_KIND`, `SCM_TOKEN_MISSING_SCOPE`, `SCM_TOKEN_EXPIRED`, `SCM_TOKEN_WRONG_REPO`, and `*Error{Code, Msg, Status}` with codes `not_found`, `forbidden`, `unauthorized`, `rate_limited`, `conflict`, `invalid`, `unavailable` ("`Msg` never contains a token")
- [X] T004 [P] Define the `Client` interface of contracts/host-interface.md in `internal/scm/client.go` (Host, Whoami, Repository, ResolveAccount, Permission, BranchProtection, OpenChangeRequests, ChangeRequest, ChangedFiles, FileAt, CommitFiles, OpenDraft, CloseChangeRequest, ProbePublisher), with a doc comment per method stating which credential it runs with
- [X] T005 [P] Repository path validation per host in `internal/scm/repo.go` with `internal/scm/repo_test.go`: GitHub "today's `repoRE`"; GitLab "2–20 segments of `[A-Za-z0-9_.-]`, no segment starting with `-` or `.`, no `.git` suffix"; Bitbucket "2 segments, lowercase slug `[a-z0-9_.-]`"; plus `RepoURL(Host, repo)` (`https://<host>/<repo>.git`, GitHub honours `fixture.BaseURL` in test mode) and `ChangeRequestURLPrefix(Host, repo)` (`github.com/<repo>/pull/`, `<host>/<path>/-/merge_requests/`, `bitbucket.org/<ws>/<repo>/pull-requests/`)
  Done as (2026-11-26): also `ValidHostName` (FORGE_SCM_HOST), `LastSegment` (the fixture guard) and `NewHost`; `scm.GitHubAPI`/`GitHubGitBase` are variables for test mode.
- [X] T006 Shared HTTP helper in `internal/scm/httpc.go` with `internal/scm/httpc_test.go`: base URL, an auth header func (Bearer or Basic), proxy from the environment, an optional CA bundle (`x509.CertPool` from `Host.CABundle`), a 60 s timeout, `Link`/`next` pagination to the end or a cap of 500 items ("a capped result says so"), status → `*Error` mapping (429 and `RateLimit-Remaining: 0` → `rate_limited`), redirect following for same-host redirects only, and redaction of the token from every error and log line (test: a token in a server's error body never reaches `Error.Msg`)
- [X] T007 Operator credentials and host-side git in `internal/scm/cred.go` with tests: `OperatorCredential(Host)` in R8's order (GitHub `GH_TOKEN`, `GITHUB_TOKEN`, `gh auth token`; GitLab `GITLAB_TOKEN`, `GITLAB_ACCESS_TOKEN`, `glab config get token --host <host>`; Bitbucket `BITBUCKET_TOKEN`), whose error names the variable to set; `GitEnv(Host, token)`: GitHub the existing `gh auth git-credential` helper (unchanged), GitLab and Bitbucket `GIT_ASKPASS=<chalkeus executable> forge askpass` with `CHALKEUS_ASKPASS_USER` (`oauth2` for GitLab, `x-bitbucket-api-token-auth` for Bitbucket) and `CHALKEUS_ASKPASS_TOKEN` in the child environment only, plus `GIT_SSL_CAINFO` when a CA bundle is set
  Done as (2026-11-26): `GitEnv` uses an inline credential helper through git's `GIT_CONFIG_COUNT` variables (git 2.31+) instead of `GIT_ASKPASS`: the helper answers with the host's username and the token from `CHALKEUS_GIT_TOKEN`, so the token never appears on a command line. `TestGitEnvHelperAnswersGit` runs a real `git credential fill` against it. The publisher's Bitbucket git user (`x-token-auth`) and the operator's (`x-bitbucket-api-token-auth`) differ; `GitUser(host, publisher)` returns each.
- [X] T008 Add the hidden `forge askpass` subcommand in `internal/cli/askpass.go` (prints the username for a "Username" prompt and the token for a "Password" prompt, from the two env vars, nothing else; exits 1 when they're unset) and classify it in `internal/cli/command_audit.go` (no harness ask: it holds no decision); `TestCommandAudit` passes
  Done as (2026-11-26): not needed. `GIT_ASKPASS` must be a bare executable path, so a `forge askpass` subcommand couldn't be called through it; T007's inline credential helper replaces it, and no command was added (the command audit is unchanged).
- [X] T009 Config keys and refusals of contracts/config.md in `internal/config/targets.go`, `internal/config/config.go` and `internal/config/targets_test.go`: `Target.SCM`, `Target.SCMHost`, `Target.SCMCABundle` from `PILOT_SCM`, `PILOT_SCM_HOST`, `PILOT_SCM_CA_BUNDLE` (and the same three in `forge.env` for the fixture); repositories validated with `scm` per host (T005) instead of `repoRE` for non-GitHub hosts; refusals with the exact messages of contracts/config.md (unknown kind; host or CA bundle on a non-GitLab host; host not a bare name; CA bundle missing, not mode 0600, not PEM, or outside the config folder; companion on a different host; `FORGE_GITHUB_ISSUES=on`, `FORGE_GITHUB_APPROVALS=on` or `FORGE_PUBLISHER_CREDENTIAL=app` with a non-GitHub host: "<setting> is GitHub-only in this release; the loop on other hosts is roadmap 038"); the repo guard's `-pilot-fixture` suffix applies to the last path segment; a test proves every existing profile in the tests loads unchanged with no `FORGE_SCM`
  Done as (2026-11-26): `internal/config/host.go` (`scmSyntax`, `caBundlePath`) and `host_test.go`. `Target.SCM` stays empty for GitHub, so every existing profile (and `TestLoadTargetGood`'s exact match) loads unchanged. A non-GitHub target doesn't inherit forge.env's `FORGE_GITHUB_APPROVALS` (found while testing: it would have switched the GitHub loop on for a GitLab target). The companion-on-another-host refusal is vacuous: a profile has one `FORGE_SCM`, and the companion is validated in that host's form. `names.Display` now keeps only the publisher credential keys (`PILOT_SCM_TOKEN`, `_KIND`, `_LOGIN`) as `PILOT_`, so the new keys show as `FORGE_SCM*`.
- [X] T010 `Config.FixtureHost()` and `Target.Host()` returning `scm.Host` (GitLab API base `https://<Name>/api/v4`, Bitbucket `https://api.bitbucket.org/2.0`, GitHub `ghloop.API`) in `internal/config/host.go` with a test; `internal/names` displays the new keys as `FORGE_SCM*`
  Done as (2026-11-26): in `internal/config/host.go` with T009; `RepoGuardFor(kind, repo, origin)` for a fixture on any host.
- [X] T011 The shared contract suite in `internal/scm/scmtest/suite.go`: `Run(t, newClient func(t) (scm.Client, Fake))` where `Fake` lets a test create a repository, a branch, an open draft request, a merge (merge commit, squash, fast-forward), a decline, an account with a permission and a protection; cases: open lookup by head prefix (drafts included), state mapping, landed commit for each merge kind, merger id, changed files, `FileAt`, permission, protection, `CommitFiles` + `OpenDraft` produce a draft whose files match, `CloseChangeRequest`, `rate_limited` mapping, and "no method merges or approves"
  Done as (2026-11-26): the owner's merge in the fake is `MergeAsOwner` (the go-github no-merge guard flags any `.Merge` selector). The suite runs once each client exists (T013, T027, T046).
- [X] T012 Evidence fields in `internal/evidence/evidence.go` and its tests: `run.json` `scm: {kind, host}` (written by `SetItemRun` and the fixture run start); `prs.json` entries gain `author_id`; `reconcile/<task>-status.json` gains `merged_by_id`; run-level `publisher_id`; GitHub writes the login in each id field
  Done as (2026-11-26): `Run.SetSCM`/`SCM` (a pre-037 run reads as GitHub) and `SetPublisherID`/`PublisherID` (falls back to `publisher_login`); `run.json` `scm` is written at fixture and item run start. `author_id` and `merged_by_id` are written where the publisher read-back and the status check move behind the interface (T015, T016).

**Checkpoint**: `go vet ./... && go test ./...` pass; no behaviour change yet.

## Phase 3: User Story 1 - GitHub works exactly as before (Priority: P1) 🎯 MVP

**Goal**: every change-request, repository and account operation goes through `scm.Client`, GitHub is its first implementation, and GitHub runs behave as before.

**Independent Test**: quickstart scenario 1: with no `FORGE_SCM`, the full test suite, the isolation suite (plus P13) and the GitHub fixture walkthrough (audit 11/11) pass, and the evidence differs from the last GitHub run only in `scm`, `author_id`, `merged_by_id`, `publisher_id` and P13.

- [X] T013 [US1] Implement `internal/scm/github/client.go` from today's code: `OpenChangeRequests` and `CloseChangeRequest` from `ghloop/client.go` (`OpenPilotPRs`, `ClosePR`), `ChangeRequest` from `ghloop/api.go` `GHReader.PR` (state `MERGED`→`merged`, `CLOSED`→`closed`, `OPEN`→`open`; `LandedCommit` = `merge_commit_sha`), `ChangedFiles` (`pulls/{n}/files`), `FileAt` (contents API), `Permission` (`GetPermissionLevel`, `maintain`→`write`), `BranchProtection`, `Repository` (`permissions.push`), `Whoami`, `ResolveAccount` (the login), `CommitFiles` + `OpenDraft` from `author/submit.go` `openPR` (Git Data API), `ProbePublisher` wrapping `ghloop.Probe`; `internal/scm/github/client_test.go` runs `scmtest.Run` against `internal/fakegh` (add any missing fake endpoints there, e.g. contents)
  Done as (2026-11-26): `internal/scm/github/client.go`. The contract test runs against an httptest fake (`internal/scm/github/fake_test.go`) over a shared in-memory model (`internal/scm/scmtest/model.go`), not fakegh: `TestPilotNeverImportsTheFake` forbids importing fakegh outside `test/`, test files included; the fakegh integration suite still covers GitHub end to end. `OpenDraft` became `OpenRequest(..., draft bool)` (GitHub companion PRs aren't drafts: private repositories on the Free plan have none) and `scm.File` gained `Delete`; contracts/host-interface.md amended. `ProbePublisher` takes the existing probe as an injected function (`scm/github` can't import `ghloop`); GitHub verification keeps calling `ghloop.VerifyFile`. New `internal/scm/hosts` builds the client for a host (`Operator`, `New`, `OpenPilotHeads`); GitLab and Bitbucket return "not built yet" until US5/US2.
- [X] T014 [US1] Make `ghloop.OpenPilotPRs`, `ghloop.ClosePR` and `GHReader.PR` thin wrappers over `scm/github` (signatures unchanged for the loop's callers), so the loop code and its tests don't move
  Done as (2026-11-26): `ghloop.OpenPilotPRs`, `ClosePR` and `PR` had no callers left once T017 moved them to `scm.Client`, so they were removed instead of wrapped; `GHReader.PR` stays for the GitHub loop's status-PR path (GitHub-only).
- [X] T015 [US1] Switch `internal/publisher/publisher.go` (task path :255–268 and status path :408–416) to `scm.Client.ChangeRequest` with the operator credential: fail unless draft, the expected head, and author id equal to the Job result's `author_id` (contracts/publisher.md "Read-back"); record `publisher_id`
  Done as (2026-11-26): `publisher.readBack` (draft, expected head, and the Job's `author_id` when reported, which `publish.py` adds with T030); `Deps.Host`; `prs.json` gains `author_id`; `run.json` `publisher_id`.
- [X] T016 [US1] Switch `internal/reconcile/reconcile.go` `ForFixture` `PRView` to `scm.Client.ChangeRequest`, and change `CheckStatusPR` to take the merger as `scm.Account` and the publisher as an id, comparing approvers by `Account.ID` (GitHub: lower-cased login, so today's cases pass unchanged); update `reconcile_test.go` with id-based cases
  Done as (2026-11-26): `ForFixture(h, …)` reads through `scm.Client.ChangeRequest` and `PRViewOf` (landed commit, R5). `CheckStatusPR` already compares plain string ids case-insensitively, so its signature stays; `reconcile/<task>-status.json` gains `merged_by_id`.
- [X] T017 [US1] Switch the open-PR readiness checks to `scm.Client.OpenChangeRequests`: `internal/pipeline/pipeline.go:509`, `internal/pipeline/item.go:152`, `internal/cli/gate.go:105`, `internal/cli/author.go:175`, `internal/cli/fixture.go` (`reset-fixture` also uses `CloseChangeRequest`)
  Done as (2026-11-26): also `cli/doctor.go`'s readiness. `reset-fixture` uses `RepoGuardFor` and the fixture's host.
- [X] T018 [US1] Switch `internal/cli/doctor.go` (:166–206: repository reads, branch protection, readiness) and `internal/cli/doctor_research.go` (`GetContents` → `FileAt`) to `scm.Client` built from `Target.Host()`
- [X] T019 [US1] Switch `internal/fixture/fixture.go` to `scm.RepoURL` and `scm.GitEnv` for clone, fetch and push (GitHub keeps the `gh` credential helper and `BaseURL` test override); remove the hard-coded `https://github.com/` from `internal/step/gather.go` and `internal/pipeline/github.go:87` by building issue links from `Host` (GitHub-only paths may keep GitHub's form)
  Done as (2026-11-26): `fixture.Git`, `FreshClone` and `URL` take the host (GitHub keeps the `gh` helper and `BaseURL`); every caller passes the target's or fixture's host. Test mode also sets `scm.GitHubAPI`/`GitHubGitBase`. The `github.com` issue links in `step/gather.go` and `pipeline/github.go` are GitHub-only paths and keep GitHub's form.
- [X] T020 [US1] Keep the GitHub-only features GitHub-only (FR-015): refuse `forge debt`, `forge github sync|drift|watch`, `forge step`, `forge app-key` and `forge app-token` for a non-GitHub target or fixture with "<command> is GitHub-only in this release" (exit 2), in `internal/cli/debt.go`, `github.go`, `step.go`; tests in `internal/cli/scm_refusals_test.go`
  Done as (2026-11-26): `gitHubOnly` in `internal/cli/scm_refusals.go`, called by `debt`, `github`, `step` and the App commands. Also: `credsForTarget` no longer auto-picks the GitHub App for a non-GitHub target (found while wiring it).
- [X] T021 [P] [US1] Egress per host: `cluster.SCMHosts(scm.Host)` in `internal/cluster/infra.go` returns `Host.EgressHosts()` (GitHub `github.com`, `api.github.com`; GitLab the instance host; Bitbucket `bitbucket.org`, `api.bitbucket.org`; plus the fake's host in test mode); `ApplyAll` passes the run's host so `AllowlistChanged` re-renders the SCM proxy when it differs; update `cluster_test.go:110`
  Done as (2026-11-26): `SCMHosts(scm.Host)`; `ApplyAll` takes the fixture's host at `up`; new `cluster.EnsureSCMProxy` re-renders the proxy before each publish when the run's host differs.
- [X] T022 [US1] The live check P13 of contracts/checks.md in `internal/checks/checks.go` (and `internal/checks/p13_test.go` for its manifest and decision logic): a short-lived pod in `pilot-publish` under the publisher's NetworkPolicy, holding no credential, requests `https://<run host>/` through the SCM proxy (must get an HTTP response), the other probe host (`gitlab.com` on a GitHub run, `github.com` otherwise; must be refused), and the run host directly without the proxy (must fail); records P13 with the failing request named; `-vv` diagnosis shows the proxy's refusal line; update the package doc ("P01–P13")
  Done as (2026-11-26): `internal/checks/p13.go` (`P13Decision`, `RunPublisherEgressCheck`), manifest `pilot/manifests/publisher-probe.yaml.tmpl`, `k8s.DeletePod`; runs after each task publish. AU04 requires P13 when recorded (contracts/checks.md amended); P13 added to the integration suite's required checks.
- [X] T023 [P] [US1] Secret heuristics: `scmRE` in `internal/checks/checks.go:37` and `scmNameRE` in `internal/config/config.go:40` also match `GITLAB`, `BITBUCKET` and `GLPAT`, with test cases
  Done as (2026-11-26): also the audit's token pattern matches GitLab's `glpat-` tokens.
- [X] T024 [P] [US1] AU03 per host in `internal/audit/audit.go` (`itemAU03`, :690–713) and `internal/audit/slices.go` (:40–69): the URL must contain `scm.ChangeRequestURLPrefix(run's host, repo)`; every PR is a draft and its `author_id` equals the run's `publisher_id`; fixture AU03 gains the author id rule; tests for each host's URL form and a GitHub golden case unchanged (`golden_test.go`)
  Done as (2026-11-26): `prOnHost`/`runHost` in `internal/audit/audit.go`, used by item and slice AU03; tests in `au03_hosts_test.go`; the golden records pass unchanged.
- [ ] T025 [US1] Run quickstart scenario 1 on GitHub (doctor, isolation suite with P13, fixture walkthrough; the owner approves and merges); compare the evidence with the last GitHub fixture run; write `specs/037-pluggable-scm/evidence/github-regression.md` with the run id, the audit and the field-by-field diff

**Checkpoint**: GitHub runs unchanged through the interface; P13 proven on GitHub.

## Phase 4: User Story 5 - Build an item on Bitbucket Cloud (Priority: P1)

**Goal**: a target on Bitbucket Cloud builds end to end: draft PR by the repository token's bot, a human merge, tree match, status change by an approver.

**Independent Test**: quickstart scenario 2 against throwaway Bitbucket repositories: doctor ok, the fixture walkthrough with the audit at 11/11, one target item run with the status change.

- [ ] T026 [P] [US5] An httptest fake of the Bitbucket endpoints in contracts/host-interface.md in `internal/scm/bitbucket/fake_test.go` (repositories, pullrequests list/get/create with `draft`, `q=` filters including the draft quirk of R3, diffstat with a redirect, `src` read and multipart write, `decline`, workspace permissions, branch-restrictions, `/user` returning "Token is invalid or not supported for this endpoint" for access tokens)
- [ ] T027 [US5] Implement `internal/scm/bitbucket/client.go` over `/2.0` with the shared HTTP helper (Bearer auth): state `MERGED`→`merged`, `OPEN`→`open`, `DECLINED`/`SUPERSEDED`→`closed`; `LandedCommit` = `merge_commit.hash`; `MergedBy` = `closed_by` (`{uuid}`); open lookup queries with and without `draft=true` and merges the results (R3); `ChangedFiles` via `diffstat/{dest}..{src}` on explicit hashes; `FileAt` via `src/{rev}/{path}`; `Permission` via `/workspaces/{ws}/permissions/repositories/{repo}?q=user.uuid="…"` (a 403 returns an error naming "the operator credential needs repository admin"); `BranchProtection` reports whether a `restrict_merges` restriction matches the branch; `ResolveAccount` accepts only `{uuid}` or an `account_id`; `CloseChangeRequest` = `decline`; `client_test.go` runs `scmtest.Run` against the fake
- [ ] T028 [US5] `ProbePublisher` for Bitbucket (research R4) in `internal/scm/bitbucket/probe.go` with tests: `GET /2.0/user` must fail with the access-token message (else `SCM_TOKEN_WRONG_KIND`: a user API token), the target repository must read (else `SCM_TOKEN_WRONG_REPO`), listing `/2.0/repositories/{ws}` must show no repository but the target (else `SCM_TOKEN_WIDER_THAN_REPO`); wire it into `forge publisher-token` verification (`internal/cli/setup.go` `verifyPublisher`) for Bitbucket targets
- [ ] T029 [US5] Approvers for Bitbucket at profile load in `internal/config/config.go`: each value must be `{uuid}` or an `account_id`, else refuse ("Bitbucket approvers must be account ids"); `CheckStatusPR` compares by that id
- [ ] T030 [US5] `publish.py` Bitbucket steps per contracts/publisher.md in `pilot/images/tools/publish.py`: read `pr.json`'s `scm` block (absent = GitHub, today's code path unchanged); skip who-am-I; push with user `x-token-auth` via `GIT_ASKPASS`; find an open PR with `q=source.branch.name="<head>" AND state="OPEN"` (and with `draft=true`); open with `draft: true`; result adds `author_id` from the PR's `author.uuid`; errors never print the token
- [ ] T031 [P] [US5] `pilot/images/tests/test_publish_hosts.py`: a local HTTP stub for Bitbucket (and GitHub regression) checking request paths and bodies, the draft flag, reuse of an open PR, the push username, and that the token never appears in output
- [ ] T032 [US5] `internal/publisher/publisher.go` writes the `scm` block into `pr.json` from the run's host; the publisher Job template `pilot/manifests/publisher-job.yaml.tmpl` is unchanged apart from passing the block; the read-back of T015 records the Bitbucket bot's `author_id` as `publisher_id`
- [ ] T033 [US5] Status change and reconcile on non-GitHub hosts in `internal/pipeline/github.go` (`statusPRMerged`, :328–395) and `internal/reconcile/reconcile.go`: for a non-GitHub host, read the head's roadmap with `FileAt(head_sha, roadmap path)` and the changed files with `ChangedFiles` instead of `git fetch origin pull/%d/head` (GitHub keeps its fetch); host-side pushes and clones use `scm.GitEnv`; decisions record channel `terminal`/`harness` (the GitHub loop stays off)
- [ ] T034 [US5] doctor rows for a Bitbucket target (contracts/checks.md) in `internal/cli/doctor.go`: `scm host`, `operator credential` (`BITBUCKET_TOKEN`), `publisher credential` (T028), `approvers` (fails without repository admin, with the exact message of contracts/checks.md), `branch protection` (fails without a `restrict_merges` restriction: "the publisher's token can merge without a restrict_merges restriction"); tests with the fake
- [ ] T035 [US5] Rebuild the tools image for `publish.py` (the `pilot-images` workflow on merge); the owner merges the digest PR; confirm the new tools image contains the Bitbucket steps
- [ ] T036 [US5] Quickstart scenario 2, steps 1–3, on throwaway Bitbucket repositories (owner prerequisites listed there): doctor ok; the `restrict_merges` and workspace-token refusals; the fixture walkthrough with draft PRs (confirm the open-PR lookup finds drafts, R3) and the audit at 11/11; write `specs/037-pluggable-scm/evidence/bitbucket-cloud.md` (run ids, refusal outputs, bot id; no tokens)

**Checkpoint**: the fixture walkthrough passes on Bitbucket Cloud.

## Phase 5: User Story 6 - Hidden tests arrive by change request on the same host (Priority: P1)

**Goal**: the test author's draft becomes a draft PR on the target's host (Bitbucket first), from the test-author identity.

**Independent Test**: with a companion on Bitbucket Cloud, the owner's test-author session submits a checked draft; a draft PR opens from the test-author identity, never the publisher's bot.

- [ ] T037 [US6] Bitbucket `CommitFiles` (`POST /2.0/repositories/{ws}/{repo}/src`, multipart: `branch`, `parents`, `message`, one field per file) and `OpenDraft` in `internal/scm/bitbucket/client.go`, covered by the contract suite
- [ ] T038 [US6] `internal/author/identity.go`: `Identity` holds an `scm.Client` instead of `*github.Client`; `ResolveIdentity` builds it from `Target.Host()` and the test-author token (`PILOT_AUTHOR_TOKEN`) or the operator credential (level minimal), and computes `CanWriteTarget` with `Repository`/`Permission`
- [ ] T039 [US6] `internal/author/submit.go`: `openPR` becomes `CommitFiles` + `OpenDraft` on the identity's client (GitHub path unchanged in behaviour); `OpenAuthorPRs` uses `OpenChangeRequests`; `internal/author/flow_test.go` keeps passing; refuse a submit whose identity id equals the target's recorded publisher id
- [ ] T040 [US6] Auto-draft on other hosts: `internal/author/watch.go` and `start.go` find open drafts and earlier drafts through the identity's client; a test with the Bitbucket fake
- [ ] T041 [US6] With the owner's test-author session (a separate session; this session never runs `forge author` or reads drafts): a draft PR opens on the throwaway Bitbucket companion; then quickstart scenario 2 step 4: one target item run on the throwaway Bitbucket target with the status change, the audit at 11/11; append to `specs/037-pluggable-scm/evidence/bitbucket-cloud.md`

**Checkpoint**: a full item run on Bitbucket Cloud, hidden tests included.

## Phase 6: User Story 3 - The approval and credential guarantees hold on every host (Priority: P1)

**Goal**: the refusal cases are proven on Bitbucket now and on each GitLab host as it lands.

**Independent Test**: each refusal case of US3 is refused and recorded on Bitbucket Cloud; the same tests run for GitLab in Phases 7 and 8.

- [ ] T042 [P] [US3] Table tests in `internal/reconcile/status_hosts_test.go` for `CheckStatusPR` on each host's id form: merged by the publisher's bot id (`STATUS_PR_MERGED_BY_PUBLISHER`), by a non-approver (`STATUS_PR_NOT_APPROVER`), by an approver without write access, and with an extra changed line (`STATUS_PR_EDITED`)
- [ ] T043 [P] [US3] A test in `internal/checks/p13_test.go` per host kind: the allowlist rendered for the publisher contains only that host's endpoints, and the other probe host is chosen as contracts/checks.md says
- [ ] T044 [US3] Quickstart scenario 2 step 5 on Bitbucket Cloud: the three status-change refusals (crafted PRs against the check, or one run each), P13 refusing `github.com`, and a grep of the run's worker and grader evidence for the token (no match); append to `specs/037-pluggable-scm/evidence/bitbucket-cloud.md`

**Checkpoint**: Bitbucket Cloud done (SC-002 to SC-005 for this host). A private Bitbucket target may now be run by the owner, with `FORGE_EVIDENCE_FEATURE` outside this repository.

## Phase 7: User Story 2 - Build an item on gitlab.com (Priority: P2)

**Goal**: a target on gitlab.com builds end to end, with a project access token or a dedicated bot account's token.

**Independent Test**: quickstart scenario 3: doctor ok, the token refusals, the fixture walkthrough and one target item run (squash merge) with the audit at 11/11, the status-change refusals.

- [ ] T045 [P] [US2] An httptest fake of the GitLab endpoints in contracts/host-interface.md in `internal/scm/gitlab/fake_test.go` (projects by URL-encoded path, merge requests with `Draft:` titles and `draft` in responses, states `opened`/`merged`/`closed`/`locked`, `merge_commit_sha` null on fast-forward, `squash_commit_sha`, `merge_user`, `diffs` pagination, `repository/files/:path/raw`, `repository/commits` with actions, members/all, protected_branches, `personal_access_tokens/self` and `/self/associations`, `/user` with `bot` and `project_{id}_bot_…`/`group_{id}_bot_…` usernames, `/users?username=`, `/version`)
- [ ] T046 [US2] Implement `internal/scm/gitlab/client.go` over `/api/v4` (`Authorization: Bearer`): projects addressed by URL-encoded full path (nested subgroups); state `merged`→`merged`, `opened`→`open`, `closed`/`locked`→`closed`; `LandedCommit` = `merge_commit_sha`, else `squash_commit_sha`, else `sha` (R5); `MergedBy` = `merge_user.id`; `OpenDraft` prefixes the title with `Draft: `; `ChangedFiles` via `merge_requests/:iid/diffs`; `Permission` via `members/all/:user_id` (`access_level` ≥ 30 → write, ≥ 40 → admin) combined with `merge_access_levels` of the protected target branch; `ResolveAccount` via `/users?username=` or `/users/:id`; `CommitFiles` via `repository/commits` with `start_branch`; `CloseChangeRequest` via a note, `state_event=close` and branch delete; `client_test.go` runs `scmtest.Run` against the fake
- [ ] T047 [US2] `ProbePublisher` for GitLab (research R4) in `internal/scm/gitlab/probe.go` with tests: `personal_access_tokens/self` active, not expired, scopes include `api` and `write_repository` (else `SCM_TOKEN_MISSING_SCOPE`/`SCM_TOKEN_EXPIRED`); `/user`: a `group_{id}_bot_…` user → `SCM_TOKEN_WIDER_THAN_REPO`; `/self/associations`: exactly one project, the target, and no groups (else `SCM_TOKEN_WIDER_THAN_REPO` or `SCM_TOKEN_WRONG_REPO`); report the kind as `project_token` (bot `project_{id}_bot_…`) or `bot_account_token` (a non-bot account, owner decision 2026-11-26); wire into `verifyPublisher`
- [ ] T048 [US2] Approvers for GitLab at load in `internal/config/config.go` and the run start: numeric ids kept, usernames resolved with `ResolveAccount` and recorded in `run.json` as `approvers: [{given, id}]`; an unresolvable name refuses with its value
- [ ] T049 [US2] `publish.py` GitLab steps per contracts/publisher.md: who-am-I via `GET /user` (bot id); push with user `oauth2`; open lookup `merge_requests?source_branch=<head>&state=opened`; open with title `Draft: <title>`; result `author_id`; add GitLab cases to `pilot/images/tests/test_publish_hosts.py`
- [ ] T050 [US2] GitLab companion: `CommitFiles` + `OpenDraft` covered by the contract suite; a GitLab case in the author tests (T039/T040 paths)
- [ ] T051 [US2] doctor rows for GitLab targets: `operator credential` (`GITLAB_TOKEN`/`glab`), `publisher credential` (T047), `approvers` (ids, Developer or higher and allowed to merge into the protected branch), `branch protection` (warn when unprotected, as GitHub); tests with the fake
- [ ] T052 [US2] Rebuild the tools image for the GitLab `publish.py` steps; the owner merges the digest PR
- [ ] T053 [US2] Quickstart scenario 3 on throwaway gitlab.com projects: doctor ok; the group-token and second-membership refusals; the fixture walkthrough (P13 refusing `github.com`); hidden tests by the owner's test-author session; one target item run with a squash merge and the status change; the status-change refusals; the token grep (`glpat-`); write `specs/037-pluggable-scm/evidence/gitlab-com.md`

**Checkpoint**: gitlab.com done.

## Phase 8: User Story 4 - A self-managed GitLab (Priority: P3)

**Goal**: a target on a self-managed GitLab, by host name, with a corporate CA bundle, builds as on gitlab.com.

**Independent Test**: quickstart scenario 4 on a throwaway GitLab CE container (owner decision 2026-11-26): doctor shows the version and the CA bundle; without the bundle it fails naming the setting; the fixture walkthrough and one target item run (fast-forward merge) with the audit at 11/11.

- [ ] T054 [US4] CA bundle in Go: the shared HTTP helper (T006) trusts `Host.CABundle`; host-side git gets `GIT_SSL_CAINFO` (T007); a certificate error without a bundle returns an error naming `FORGE_SCM_CA_BUNDLE`; tests with an httptest TLS server and a generated CA
- [ ] T055 [US4] CA bundle in the publisher pod: `internal/cluster/infra.go` creates ConfigMap `scm-ca` in `pilot-publish` from the bundle (and deletes it when none); `pilot/manifests/publisher-job.yaml.tmpl` mounts it at `/etc/chalkeus/scm-ca.pem` when present; `pr.json` `scm.ca_file`; `publish.py` uses an `ssl` context with that file and sets `GIT_SSL_CAINFO`; tests in `test_publish_hosts.py` and the cluster render tests
- [ ] T056 [US4] doctor rows `scm host` (self-managed: `GET /version`, fail below 18.0 with "GitLab <v> is older than 18.0") and `scm ca bundle` (a TLS connection with the bundle succeeds) in `internal/cli/doctor.go`, with tests
- [ ] T057 [US4] A setup script for the proof instance, `specs/037-pluggable-scm/evidence/gitlab-ce/up.sh` and `down.sh`: a CA and server certificate for `gitlab.<host-ip>.nip.io` (resolves on the host and inside kind), GitLab CE at a pinned image digest (≥ 18.0) with that certificate and a memory limit of about 4 GB; no secrets written into the repository
- [ ] T058 [US4] Quickstart scenario 4 on the Ubuntu machine: doctor ok; the missing-bundle refusal; the fixture walkthrough and one target item run with a fast-forward merge (P13 refusing `gitlab.com` and `github.com`); tear down; write `specs/037-pluggable-scm/evidence/gitlab-self-managed.md` (GitLab version, certificate fingerprints, run ids)

**Checkpoint**: all three new hosts proven (SC-002, SC-003).

## Phase 9: User Story 7 - Set up and check a new host (Priority: P3)

**Goal**: an operator sets up a target on any host from the docs alone, and doctor checks every piece.

**Independent Test**: quickstart scenario 5: following only `docs/targets.md`, a fresh GitLab and a fresh Bitbucket target pass doctor before the first run; each refusal has a troubleshooting entry.

- [ ] T059 [P] [US7] doctor `commit author` row (warn when the publisher's commit email isn't a valid address, with the message of contracts/checks.md) and `github-only settings` row, for every host including GitHub, in `internal/cli/doctor.go` with tests
- [ ] T060 [P] [US7] `docs/targets.md`: a section per host (Bitbucket Cloud, gitlab.com, self-managed GitLab): the `FORGE_SCM*` keys, the publisher credential kind and scopes (project token or dedicated bot account; repository token), the `restrict_merges` restriction, approver ids, the operator's `GITLAB_TOKEN`/`BITBUCKET_TOKEN`, the CA bundle, what stays GitHub-only, and keeping a private target's evidence outside the repository with `FORGE_EVIDENCE_FEATURE`
- [ ] T061 [P] [US7] `docs/troubleshooting.md`: one entry per new refusal (each config refusal, each `SCM_TOKEN_*` code, the approvers-need-admin row, the missing `restrict_merges` row, the old GitLab version, the CA bundle, P13 failing, a GitHub-only command), with index rows
- [ ] T062 [P] [US7] `docs/operator-guide.md`: supported Git hosts in "Before you start", and the section 13 row for Bitbucket and GitLab
- [ ] T063 [US7] Quickstart scenario 5: follow only `docs/targets.md` for a fresh Bitbucket and a fresh GitLab target; record gaps and fixes in `specs/037-pluggable-scm/evidence/docs-walkthrough.md`

## Phase 10: Polish & Cross-Cutting Concerns

- [ ] T064 [P] `docs/release-notes.md`: the 037 section (hosts, settings, GitHub-only features, P13, the AU03 change)
- [ ] T065 `go vet ./... && go test ./...` (including the docs check) and `python3 -m unittest discover -s pilot/images/tests` pass; `grep -rn 'github.com/' internal --include='*.go'` shows no hard-coded host outside GitHub-only code and tests
- [ ] T066 Privacy check before each PR: `git diff main --stat` and a grep of the diff for the private target's name (supplied by the owner at the time, never written down) find nothing; no evidence from a private target exists under `specs/`
- [ ] T067 `/speckit-roadmap-debrief` for 037, then the roadmap status change through `/speckit-roadmap-write` on the owner's decision

## Dependencies & Execution Order

- **Setup (T001–T002)** → **Foundational (T003–T012)** → user stories.
- **US1 (T013–T025)** first: every later story uses the interface and P13.
- **US5 (T026–T036)** after US1. **US6 (T037–T041)** after US5 (T041 needs the Bitbucket fixture working). **US3 (T042–T044)** after US6; T042 and T043 can start any time after US1.
- **US2 (T045–T053)** after US3 (reuses the Bitbucket-proven publisher, status and author paths). **US4 (T054–T058)** after US2.
- **US7 (T059–T063)**: T059–T062 can start after US5; T063 after US4.
- **Polish (T064–T067)** last; T066 runs before every PR.
- Tools-image rebuilds: T035 (Bitbucket) and T052 (GitLab); T055 changes `publish.py` again, so its rebuild rides with the US4 PR.

## Parallel Opportunities

- Foundational: T003, T004 and T005 in parallel; T006 and T007 after T003.
- US1: T021, T023 and T024 in parallel with T013–T020.
- US5: T026 and T031 in parallel with T027–T030.
- US3: T042 and T043 in parallel.
- US2: T045 in parallel with the start of T046.
- US7: T059–T062 in parallel.

## Implementation Strategy

1. **MVP = US1**: the interface with GitHub behind it and P13, proven unchanged on GitHub (T025). Mergeable on its own.
2. **Bitbucket Cloud (US5 + US6 + US3)**: the first new host, end to end with hidden tests and the refusal cases; after it, the owner can run the private Bitbucket target.
3. **gitlab.com (US2)**, then **self-managed (US4)** on the Ubuntu machine.
4. **Docs (US7)** and polish; then the debrief.

Each phase ends with a PR for the owner to merge; every live step's approvals and merges are the owner's.
