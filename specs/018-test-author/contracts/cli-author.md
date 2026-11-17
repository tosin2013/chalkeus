# Contract: `pilot/pilot author`

All subcommands take `--target <name> --item <NNN>`. Exit codes follow the pilot's table
(0 OK, 1 FAIL, 2 ERROR, 10 REFUSE). Output never contains credentials; only `check` prints test
output, and only the author's own draft.

| Command | Does | Refusals (exit 10) |
|---|---|---|
| `start` | readiness except hidden tests; creates the workspace; writes `criteria.json`; prints workspace, level, identity | any readiness code; `COMPANION_NOT_EMPTY` (unless `--replace`); `DRAFT_PR_OPEN`; `AUTHOR_CAN_WRITE_TARGET` (recommended); `LEVEL_NEEDS_CLUSTER_MODE` (strict) |
| `check` | validates layout and mapping; runs the baseline with the draft as overlay | `DRAFT_EMPTY`, `DRAFT_OUTSIDE_FOLDER`, `MAP_INVALID`; result codes (exit 1): `DRAFT_HIDDEN_PASSES`, `DRAFT_SUITE_FAILS` |
| `submit` | opens the companion PR | `DRAFT_NOT_CHECKED` (no passing check for this draft hash); `DRAFT_PR_OPEN` |
| `status` | prints the draft's state and evidence path | — |

Workspace layout:

```text
<workspace>/
├── spec.md  plan.md  tasks.md  criteria.json   # read-only inputs
├── base/                                       # the unchanged project (read-only reference)
└── draft/                                      # the agent writes here only
    ├── <project-relative test files>
    └── .acceptance-map.json
```
