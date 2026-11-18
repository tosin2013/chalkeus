# Data model: Test author

## Draft

| Field | Notes |
|---|---|
| `draft_id` | UTC timestamp, like run ids |
| `target`, `item`, `spec_sha256` | from readiness |
| `level` | `minimal` \| `recommended` (`strict` refused until 011) |
| `identity` | login of the identity that will open the PR; `can_write_target` (bool) |
| `workspace` | path of the draft workspace |
| `files[]` | `{path, sha256}` relative to the item folder; no contents |
| `checks[]` | each `author check`: time, draft hash, result, reason code, `grade.json` path |
| `mapping` | criteria count, covered count, uncovered ids |
| `pr` | URL, number, branch, commit |
| `state` | `started` → `checked(pass|fail)` → `submitted` \| `refused(<code>)` |

## Acceptance map (`.acceptance-map.json`)

See [contracts/acceptance-map.md](contracts/acceptance-map.md).

## Run additions

- `run.json`: `security_level`; readiness record gains `mapping` (`present`, `uncovered[]`,
  `stale`).
