# Contract: budget keys in a target profile

All optional. Each value must be ≥ the toolchain catalog's (or the operator-supplied toolchain's
defaults); a lower or malformed value refuses the profile with
`TARGET_INVALID: <KEY> <value> is below the catalog's <floor> for <toolchain>` or
`TARGET_INVALID: <KEY> must be <a CPU quantity | a memory quantity | whole seconds>`.
`PILOT_*` spellings are accepted with the deprecation warning, as for every other key (027).

| Key | Role | Unit | node24 floor |
|---|---|---|---|
| `FORGE_WORKER_CPU` | worker | CPU quantity (`2`, `3500m`) | 2 |
| `FORGE_WORKER_MEMORY` | worker | memory quantity (`4Gi`) | 2Gi |
| `FORGE_WORKER_DEADLINE_S` | worker attempt | seconds | 1200 |
| `FORGE_GRADER_CPU` | grader and prefetch | CPU quantity | 2 |
| `FORGE_GRADER_MEMORY` (exists) | grader and prefetch | memory quantity | 2Gi |
| `FORGE_GRADER_DEADLINE_S` (exists) | grader and prefetch job | seconds | 900 |
| `FORGE_COMMAND_TIMEOUT_S` | each test command in the grader | seconds | 600 |
| `FORGE_AUTHOR_CPU` | test author's draft job | CPU quantity | 2 |
| `FORGE_AUTHOR_MEMORY` | test author's draft job | memory quantity | 2Gi |
| `FORGE_AUTHOR_DEADLINE_S` | test author's draft job | seconds | catalog value |

Rules:
- `FORGE_COMMAND_TIMEOUT_S` must be ≤ `FORGE_GRADER_DEADLINE_S` (resolved values); otherwise
  `TARGET_INVALID: FORGE_COMMAND_TIMEOUT_S exceeds the grader deadline`.
- `FORGE_WORKER_DEADLINE_S` must be > the soft-deadline margin (120 s) plus 60 s.
- Removing a key returns that value to the floor for the next run or draft; a resumed run keeps
  the budget in its `run.json`.
- `./chalkeus forge targets` prints the raised keys for each target.
