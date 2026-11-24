# Data model: Brownfield intake (029)

## TriageReport (`forge debt --json`)

| Field | Type | Notes |
|---|---|---|
| `version` | int | 1 |
| `target` | string | the profile name |
| `repo` | string | `owner/name` |
| `default_branch` | string | |
| `default_commit` | string | the head the report read (40 hex) |
| `generated_at` | string | RFC 3339, UTC |
| `credential` | string | `app-read` or `operator-read` (R1); never a secret |
| `permissions` | object | each optional permission → `granted` or `missing` (R1) |
| `default_checks_failing` | string[] | check names failing on the default branch's head |
| `shared_checks` | object | check name → why it's shared (research R4 amendment) |
| `groups` | Group[] | ordered by size, largest first |
| `not_triaged` | PREntry[] | class `not triaged`, kept out of the groups |
| `baseline_flags` | BaselineFlag[] | |
| `unreadable` | string[] | what the report couldn't read, for example `advisories: missing vulnerability_alerts` |
| `counts` | object | class → number of PRs |

## Group

| Field | Type | Notes |
|---|---|---|
| `failing_checks` | string[] | the group's key, sorted; empty for `no failing checks` |
| `shared` | string[] | the subset that is shared (fails on the default branch, or on most PRs) |
| `prs` | PREntry[] | ordered by PR number |

## PREntry

| Field | Type | Notes |
|---|---|---|
| `number` | int | |
| `url` | string | |
| `author` | string | for example `dependabot[bot]` |
| `ecosystem` | string | from the branch (R3); `""` when unknown |
| `directory` | string | from `in /<dir>`; `/` by default |
| `packages` | Package[] | one, or many for a grouped PR |
| `level` | string | the highest of the packages: `patch`, `minor`, `major` or `unknown` |
| `checks` | object | `failing`: string[], `pending`: string[], `passing`: int |
| `advisories` | string[] or `"unknown"` | GHSA ids it fixes (R6) |
| `class` | string | `superseded`, `safe`, `needs code changes`, `pending` or `not triaged` |
| `reason` | string | one sentence, for example `default branch already has zod 3.25.76` |
| `dependency_item` | bool | true for `needs code changes`: the report suggests it as a dependency item |

### Package

| Field | Type |
|---|---|
| `name` | string |
| `from` | string |
| `to` | string |
| `default_branch_version` | string, or `"unknown"` (R5) |
| `level` | string |

## BaselineFlag

| Field | Type | Notes |
|---|---|---|
| `kind` | string | `red default branch`, `force exit` or `install scripts` (R7) |
| `where` | string | a check name, or `path:line` |
| `detail` | string | |

## ManifestChange (item spec → run state → `manifest-change.json`)

| Field | Type | Notes |
|---|---|---|
| `commit` | string | 40 hex, from `## Manifest change` |
| `parent` | string | the commit's first parent |
| `files` | object[] | `path`, `before` (blob at parent), `after` (blob at commit) |
| `packages` | Package[] | `from`/`to` read from the diff (R14); `default_branch_version` omitted |
| `base_revision` | string | the run's base; each listed file's blob there equals `before` (R9) |
| `snapshot_tree` | string | the tree the worker, baseline and prefetch started from |
| `replayed_commit` | string | the manifest commit on the base in the publish bundle (R10), set at publish |

## Grading meta additions

| Field | Where | Notes |
|---|---|---|
| `fixed_paths` | attempt and baseline `meta.json` | the listed files; G2 fails on any change to one (R11) |
| `dependency` | baseline `meta.json` | true: G4 recorded, not required; G5 must still fail (R12) |

## Readiness refusal codes (new)

`MANIFEST_CHANGE_UNREADABLE`, `MANIFEST_CHANGE_UNRESOLVED`, `MANIFEST_CHANGE_NOT_MANIFEST`, `MANIFEST_CHANGE_STALE`,
`MANIFEST_CHANGE_EMPTY`, `MANIFEST_CHANGE_SLICED` (R9, R13; `contracts/item-spec.md`).
