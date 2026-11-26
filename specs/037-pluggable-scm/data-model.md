# Data model: Pluggable SCM, phase 1 (037)

All types live in `internal/scm` unless noted. Field names are Go; JSON names (evidence) are in
backticks.

## Host

The Git host a repository lives on, from a target profile or `forge.env`.

| Field | Type | Rules |
|---|---|---|
| Kind | `github` \| `gitlab` \| `bitbucket` | from `FORGE_SCM`; empty means `github` |
| Name | string | GitHub `github.com`; GitLab `FORGE_SCM_HOST` or `gitlab.com`; Bitbucket `bitbucket.org`. `FORGE_SCM_HOST` is refused for GitHub and Bitbucket |
| APIBase | URL | GitHub `https://api.github.com` (test mode may override); GitLab `https://<Name>/api/v4`; Bitbucket `https://api.bitbucket.org/2.0` |
| GitBase | URL | `https://<Name>/` |
| CABundle | path | `FORGE_SCM_CA_BUNDLE`, GitLab only; a PEM file in the config folder, mode 0600 |
| EgressHosts | []string | R10 |

Evidence: `run.json` gains `scm: {kind, host}`.

## Repo path

| Kind | Form | Validation |
|---|---|---|
| github | `owner/name` | today's `repoRE` |
| gitlab | `group[/subgroup…]/project` | 2–20 segments of `[A-Za-z0-9_.-]`, no segment starting with `-` or `.`, no `.git` suffix |
| bitbucket | `workspace/repo_slug` | 2 segments, lowercase slug `[a-z0-9_.-]` |

The fixture's repo-guard suffix (`-pilot-fixture`) applies to the last segment.

## Account

| Field | Type | Rules |
|---|---|---|
| ID | string | stable: GitHub login (lower-cased for comparison), GitLab numeric user id, Bitbucket `{uuid}` |
| Name | string | display or username at read time; informational only |
| Bot | bool | GitLab `bot: true`; GitHub App or `Bot` type; Bitbucket: the recorded publisher id |

Comparison is by ID only.

## Approver list

`FORGE_APPROVERS` (comma-separated). Per host:
- GitHub: logins, leading `@` dropped (unchanged).
- GitLab: numeric ids or usernames. Usernames are resolved at load (`GET /users?username=`) and the
  run records `approvers: [{given, id}]` in `run.json`.
- Bitbucket: `{uuid}` or `account_id` only; a value that is neither is refused at load.

## ChangeRequest

| Field | JSON | Notes |
|---|---|---|
| ID | `number` | GitHub PR number, GitLab MR iid, Bitbucket PR id |
| URL | `url` | web URL (AU03 checks its form) |
| State | `state` | `open` \| `merged` \| `closed` (R5 mapping); GitHub's `MERGED`/`OPEN`/`CLOSED` spelling is kept in existing evidence fields |
| Draft | `draft` | native on all hosts |
| HeadRef, HeadSHA | `head`, `head_sha` | |
| BaseRef | `base` | |
| LandedCommit | `merge_commit` | R5: merge, squash or fast-forward head |
| Author | `author`, `author_id` | |
| MergedBy | `merged_by`, `merged_by_id` | zero until merged |

State transitions: `open → merged` (a human merge) or `open → closed` (declined, superseded,
closed, locked). forge never performs either transition on an item or status request; it closes
only fixture requests in `reset-fixture`, and only when the human asked.

## PublisherCredential

| Field | Notes |
|---|---|
| Host | the target's Host |
| Kind | GitHub `token` \| `app`; GitLab `project_token` \| `bot_account_token`; Bitbucket `repository_token` |
| Repo | the one repository it reaches (proven, R4) |
| BotID | Account ID of the identity it acts as; Bitbucket: recorded from the first PR's author |
| Expires | when the host reports it |

Stored as today in `targets/<name>.publisher.env` (`PILOT_SCM_TOKEN`; `PILOT_SCM_KIND`,
`PILOT_SCM_LOGIN` for the App), copied into Secret `pilot-publish/scm-token`.

## OperatorCredential

The human's own token for a host, resolved per R8 each time it is needed; never written to disk by
forge, never put in a Secret, never passed to a pod.

## TokenProbeResult

Per host, the result of the publisher-credential check (`ghloop.ProbeResult` generalised): kind
detected, repository reached, scopes, expiry, and one line per probe with pass or fail. Refusal
codes: `SCM_TOKEN_WIDER_THAN_REPO`, `SCM_TOKEN_WRONG_KIND`, `SCM_TOKEN_MISSING_SCOPE`,
`SCM_TOKEN_EXPIRED`, `SCM_TOKEN_WRONG_REPO`.

## Evidence changes

| File | Change |
|---|---|
| `run.json` | `scm: {kind, host}`; `approvers` resolution for GitLab |
| `prs.json` | each entry gains `author_id` |
| `reconcile/<task>.json` | unchanged fields; `merge_commit` is the landed commit (R5) |
| `reconcile/<task>-status.json` | gains `merged_by_id` |
| `checks.json` | gains P13 |

GitHub runs gain only `scm`, `author_id` (= the login), `merged_by_id` (= the login) and P13.
