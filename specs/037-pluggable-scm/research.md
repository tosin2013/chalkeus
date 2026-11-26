# Research: Pluggable SCM, phase 1 (037)

Researched 2026-11-26. Two inputs: a map of where GitHub is coupled in this repository (R1), and
the host APIs from official docs (docs.gitlab.com, developer.atlassian.com,
support.atlassian.com). "Unverified" marks what the docs don't settle; each such point has a
check in the live proof (quickstart.md).

## R1. Where GitHub is coupled today

**Decision**: one new package, `internal/scm`, owns every host operation forge performs on change
requests, repositories and accounts. GitHub becomes its first implementation, built from the
code that exists today. The GitHub loop (issues, labels, comment approvals: `ghloop.Loop`,
`approve.go`, `sync.go`), the GitHub App (`internal/ghapp`) and the debt report (`internal/debt`)
stay GitHub-only and keep their code.

**What the map found** (non-test code, excluding the fake):
- 48 go-github call sites. Only about half go through `ghloop.Reader`/`Writer`; the rest pass a
  raw `*github.Client`: `ghloop.OpenPilotPRs`/`ClosePR`/`ProposedBy`/`PlanningPRs`,
  `author.Identity.Client` and `author.openPR` (the companion PR, opened host-side in Go through
  the Git Data API), `publisher.go` (draft read-back), `reconcile.ForFixture`, `cli/doctor.go`
  (repository, branch protection), `cli/debt.go`.
- Host-side git uses the GitHub CLI: `fixture.go:21` sets
  `credential.helper=!gh auth git-credential`; `ghloop.ReadToken` runs `gh auth token`.
- Fetching the status PR's head uses GitHub's `pull/N/head` refspec (`pipeline/github.go:354`).
- Hard-coded `https://github.com/` URLs: `fixture.BaseURL`, `publish.py:81`, `step/gather.go`,
  `pipeline/github.go:87`; the audit's AU03 matches `github.com/<repo>/pull/` in item and slice
  PR URLs (`audit.go:707`, `slices.go:59`).
- Repositories are two segments everywhere: `repoRE` (`config.go:43`), `SplitRepo`, the fake's
  routes. GitLab subgroups are refused today.
- `originRE` recognises only a GitHub origin (canary for the GitHub token check).
- Secret-name heuristics match only GitHub names: `scmRE` (`checks.go:37`), `scmNameRE`
  (`config.go:40`).
- No P-check tests the publisher's own egress refusal: P06 checks the worker; A03/A04 the author
  namespace.
- Python: only `publish.py` runs in a pod. `check_token.py` ships but nothing runs it (its Go port
  is `ghloop.Probe`); `ghops.py` ships in no image (ported to `ghloop.GHWriter`).
- The no-merge guard (`ghloop/nomerge_test.go`) parses go-github method names.

**Rationale**: constitution VI already names SCMs as "pluggable backends behind adapters". The
loop, the App and the debt report are GitHub features with no phase-1 counterpart (038 for the
loop), so moving them would add risk with no gain.

**Alternatives considered**: extend `ghloop.Reader`/`Writer` with GitLab and Bitbucket
implementations. Rejected: those interfaces mix the loop (issues, labels, comments) with change
requests, and the code that bypasses them would still need its own abstraction.

## R2. Go clients for GitLab and Bitbucket

**Decision**: plain `net/http` with small typed structs for both, in `internal/scm/gitlab` and
`internal/scm/bitbucket`, sharing one HTTP helper (base URL, auth header, proxy from the
environment, optional CA bundle, timeouts, pagination, error mapping). No new Go module.

**Rationale**: forge needs about 12 endpoints per host. The official GitLab client
(`gitlab.com/gitlab-org/api/client-go/v3`, v3.57.0, `go 1.26.0`) is large and brings transitive
dependencies; the main Bitbucket library (`github.com/ktrysmt/go-bitbucket`) returns untyped
`interface{}`. Plain HTTP keeps the no-merge guard simple: a test forbids merge and approve paths
in the two packages (R12).

**Alternatives considered**: client-go v3 for GitLab (official, typed; rejected for size and
guard complexity, revisit if the endpoint count grows in 038); go-bitbucket (untyped).

## R3. Drafts on each host

**Decision**: every host opens a native draft.
- GitHub: `draft: true` (unchanged).
- GitLab: the title starts with `Draft: `; there is no `draft` attribute on create. Responses
  carry `draft: true`. A draft can't be merged until the prefix is removed.
- Bitbucket Cloud: `draft: true` on `POST /2.0/repositories/{ws}/{repo}/pullrequests`. A draft
  isn't eligible for merging until marked ready (announced 2025-04-02).

**Rationale**: resolves roadmap Q14's first half: Bitbucket has a draft state, so the spec's
stand-in (title marker plus branch restrictions) isn't needed. AU03's "every PR is a draft"
holds on all three hosts.

**Unverified**: a community report says Bitbucket's PR list leaves drafts out unless filtered
with `draft=true`. The open-PR lookup queries with and without the filter and the proof checks it.

## R4. Publisher credentials and the one-repository guarantee

**Decision**:
- **GitLab:** a project access token (scopes `api` and `write_repository`, role Developer) is the
  preferred kind. Owner decision 2026-11-26: a personal access token of a **dedicated bot
  account** is also accepted, because gitlab.com's Free plan has no project access tokens
  (Premium and Ultimate only; self-managed allows them on any license). Either way the token
  must reach exactly one project, proven by `GET /api/v4/personal_access_tokens/self` (active,
  scopes, expiry, `user_id`) and `GET /personal_access_tokens/self/associations` (17.4+: exactly
  one project, the target, and no groups). Group access tokens (bot username `group_{id}_bot_…`)
  and personal tokens of accounts with more memberships are refused.
- **Bitbucket Cloud:** a repository access token (scopes `repository:write` and
  `pullrequest:write`; git user `x-token-auth`; API `Authorization: Bearer`). Such tokens can't
  read their own identity (`GET /2.0/user` fails with "Token is invalid or not supported for this
  endpoint") or any permission API. The check is: `GET /2.0/user` fails that way (so it isn't an
  Atlassian user API token); the target repository reads; listing the workspace's repositories
  shows no repository besides the target (a project or workspace token shows more; heuristic,
  confirmed in the proof). The bot's account id is read back from the `author` of the first PR
  it opens and recorded.
- **Bitbucket merge capability:** `pullrequest:write` also allows merging. doctor requires a
  `restrict_merges` branch restriction on the target branch (access tokens can't be exempted from
  it), read with the operator's credential.

**Rationale**: the guarantee is "the publisher's credential reaches one repository", proven before
every run (FR-009). On GitLab the associations API proves it for both token kinds. On Bitbucket
the token kind proves it by construction, and the probes refuse the wider kinds.

**Alternatives considered**: requiring Premium on gitlab.com (owner chose the bot-account path);
asking for a second "canary" repository per Bitbucket workspace as GitHub's check does (more
setup; kept as a fallback if the listing probe proves unreliable).

## R5. Merged state, landed commit and merger

**Decision**: one normalised record per change request: state `open`, `merged` or `closed`; the
landed commit; the merger as an account (stable id plus display name).
- GitHub: `merged`, `merged_at`, `merge_commit_sha`, `merged_by.login` (unchanged; the id stays
  the login so existing evidence and approver lists keep working).
- GitLab: `state` `merged` is merged; `opened` is open; `closed` and `locked` are closed. Landed
  commit: `merge_commit_sha`, else `squash_commit_sha`, else `sha` (fast-forward merges leave
  `merge_commit_sha` null; GitLab itself falls back to the diff head). Merger: `merge_user.id`
  (`merged_by` is deprecated since 14.7).
- Bitbucket: `MERGED` is merged; `OPEN` is open; `DECLINED` and `SUPERSEDED` are closed. Landed
  commit: `merge_commit.hash`. Merger: `closed_by.uuid`.

**Rationale**: the tree-match check already decides whether the landed files equal the graded
attempt, so the landed commit only needs to be recorded faithfully.

## R6. Changed files and the status-change head

**Decision**: `ChangedFiles` per host (GitHub `pulls/{n}/files`; GitLab
`merge_requests/:iid/diffs`, paginated; Bitbucket `diffstat/{dest}..{source}` on explicit hashes,
following redirects). The status-change check reads the roadmap at the request's head commit with
`FileAt(rev, path)` through the API (GitLab `repository/files/:path/raw?ref=`; Bitbucket
`src/{commit}/{path}`) instead of GitHub's `pull/N/head` fetch. GitHub keeps its git fetch, so
GitHub evidence is unchanged.

## R7. Approver identity and write access

**Decision**: approvers are compared by stable id.
- GitHub: the login, case-insensitive (unchanged).
- GitLab: numeric user id. A username in `FORGE_APPROVERS` is resolved with `GET /users?username=`
  and recorded as `name → id`. Write access: `GET /projects/:id/members/all/:user_id`,
  `access_level` ≥ 30 (Developer), and allowed to merge into the protected target branch
  (`merge_access_levels`).
- Bitbucket: account UUID (`{…}`) or `account_id`. Usernames can't be resolved through the API, so
  `FORGE_APPROVERS` must list ids for Bitbucket. Write access:
  `GET /2.0/workspaces/{ws}/permissions/repositories/{repo}?q=user.uuid="…"` (effective
  permission `write` or `admin`), which needs **repository admin** on the operator's credential;
  without it doctor fails that row (fail closed).

## R8. The operator's own credential

**Decision**: forge's reads on the operator's machine use the operator's login for that host,
never the publisher's (FR-013).
- GitHub: `GH_TOKEN`, `GITHUB_TOKEN`, then `gh auth token` (unchanged).
- GitLab: `GITLAB_TOKEN`, then `GITLAB_ACCESS_TOKEN`, then `glab config get token --host <host>`.
  `glab` has no `auth token` command, and `glab auth login` now stores tokens in the OS keyring,
  so the env var is the documented path and the `glab` call is best effort.
- Bitbucket: `BITBUCKET_TOKEN`, an Atlassian API token with scopes, sent as Bearer. App passwords
  are gone (no new ones since 2025-09-09; removal announced for 2026-07-28). There is no official
  Bitbucket CLI to read a token from.

Host-side git (clone, fetch, push for reconcile and the fixture) uses a credential helper that
answers from the same operator credential: `gh` for GitHub (unchanged); for GitLab and Bitbucket
the chalkeus binary itself as `GIT_ASKPASS`, reading the token from its environment, so the token
never appears on a command line. Git usernames: GitLab `oauth2`; Bitbucket API tokens
`x-bitbucket-api-token-auth`.

## R9. Self-managed GitLab

**Decision**: `FORGE_SCM_HOST` names the instance; API base `https://<host>/api/v4`, git base
`https://<host>/`. `FORGE_SCM_CA_BUNDLE` names a PEM file in the config folder (mode 0600). It is
trusted by forge's Go HTTP client and host-side git (`http.sslCAInfo`), and mounted into the
publisher pod (ConfigMap `scm-ca`) for `publish.py` (an `ssl` context with that file) and its git
(`GIT_SSL_CAINFO`). Minimum version **18.0**: everything forge uses exists by 17.4
(`personal_access_tokens/self/associations`), and GitLab supports only its last three monthly
releases (19.4–19.6 today); 18.0 is the spec's "current and previous major". doctor reads
`GET /api/v4/version` and refuses older instances.

**Proof instance** (owner decision 2026-11-26, roadmap Q14): a throwaway GitLab Community Edition
container at a pinned version on the Ubuntu box, served with a certificate from a self-signed CA,
so the CA-bundle path is exercised. It needs about 4 GB of memory while it runs and is removed
after the proof.

## R10. Egress per host

**Decision**: `cluster.SCMHosts` takes the run's host: GitHub `github.com`, `api.github.com`;
GitLab the instance host (`gitlab.com` by default); Bitbucket `bitbucket.org`,
`api.bitbucket.org`. The publisher's proxy is re-rendered when the allowlist changes (the
existing `AllowlistChanged` path), so a run's publisher reaches only its own host. A new live
check, **P13**, proves it: from the publisher namespace, the run's host is reachable through the
SCM proxy and one other Git host is refused (`gitlab.com` for a GitHub run, `github.com`
otherwise). P13 also runs for GitHub runs: it is the only change to GitHub evidence besides the
recorded host.

## R11. The companion change request (test author)

**Decision**: `scm.Client` gets `CommitFiles(repo, base, branch, files, message)` and
`OpenDraft(repo, head, base, title, body)`; the test author's `Submit` uses them with the
test-author identity (`targets/<name>.author.env`, `PILOT_AUTHOR_TOKEN`, a token for that host).
- GitHub: the Git Data API, as today.
- GitLab: `POST /projects/:id/repository/commits` with `branch`, `start_branch` and `actions`
  (one commit, any number of files), then a `Draft:` MR.
- Bitbucket: `POST /2.0/repositories/{ws}/{repo}/src` (multipart: `branch`, `parents`, `message`,
  one field per file), then a draft PR.

The companion must be on the same host as its target (spec assumption); a different host is
refused at profile load.

## R12. Guards and heuristics

**Decision**:
- The no-merge guard covers the new packages: a test fails if `internal/scm/gitlab` or
  `internal/scm/bitbucket` contain a merge, approve or accept path (GitLab `/merge`,
  `/approve`, `/approvals`; Bitbucket `/merge`, `/approve`), and the go-github method check keeps
  covering `internal/scm/github`.
- `scmRE` (P04, P09) and `scmNameRE` (model folder) also match `GITLAB`, `BITBUCKET`, `GLPAT`.
- AU03 checks the change-request URL against the host's form: `github.com/<repo>/pull/`,
  `<host>/<path>/-/merge_requests/`, `bitbucket.org/<ws>/<repo>/pull-requests/`.

## R13. Tests without live hosts

**Decision**: each new implementation has an `httptest` fake of the endpoints it calls
(`internal/scm/gitlab/fake_test.go`, `internal/scm/bitbucket/fake_test.go`) and runs a shared
contract suite (`internal/scm/scmtest`) that the GitHub implementation also passes against
`fakegh`. The integration suite (`test/integration`, fakegh) stays GitHub-only. The live proof on
gitlab.com, the GitLab CE container and Bitbucket Cloud (SC-002, SC-003) is the evidence for the
new hosts. `publish.py` gets per-host unit tests with a local HTTP stub, as
`test_publisher_idempotency.py` does today.
