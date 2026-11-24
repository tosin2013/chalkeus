# Contract: soft deadline in the worker (US6)

Extends the worker I/O contract (014 `contracts/worker-io.md`).

## Input (`/in/meta.json`)

- `soft_deadline_s` (int): seconds after the harness starts at which the watchdog stops it.
  Set by forge to `worker deadline_s − 120`. Absent (older forge) → no soft deadline.

## Watchdog (`pilot/images/worker/watchdog.py`)

- Tracks elapsed time from its start; when it passes `soft_deadline_s`, sends SIGTERM to the
  harness (as for the spend cap), and, if the harness hasn't exited after 15 s, SIGKILL.
- `/tmp/spend.json` gains `"stopped_by_deadline": true|false` alongside `stopped_by_cap`.

## Entrypoint (`pilot/images/worker/entrypoint.sh`)

- After the harness exits: `ERROR="deadline"` when `stopped_by_deadline`, else the existing
  rules. Then the normal path: copy back, commit, bundle (possibly empty), `finish()`.
- `report.json` gains `"stopped_by": "deadline" | "spend_cap" | null`.

## Forge (`internal/worker/worker.go` `RunAttempt`)

- `report.error == "deadline"` → attempt `failed`, reason `deadline`, with `spend`,
  `changed_paths`, `report.json` and `change.bundle` kept as evidence.
- A hard `DeadlineExceeded` (no frame) is still `failed/deadline`, with `spend: null` and an
  evidence note `no_frame` (the margin was too short; seen only if the harness ignores SIGTERM).
- A deadline-stopped attempt is never graded or published; its spend counts toward the caps.
- The retry decision text states: `stopped at the deadline after <s> s; <tokens> tokens,
  $<cost>; changed: <n> file(s)`.
