# Contract: the detached watcher (issue #25)

## Start

- **Automatic:** with `PILOT_GITHUB_APPROVALS=on`, every pause (mode, retry, A's merge, the
  status-change PR) calls `ensureWatcher(run)`.
- **By hand:** `pilot github watch --run <id> --detach` does the same.
- **No second watcher:** `ensureWatcher` starts one only if `github/watch.pid` is absent or
  names a dead process.
- **How it starts:** it runs the same binary as
  `pilot github watch --run <id>`, in a new session (Unix: `setsid`), with stdin from
  `/dev/null`, and stdout and stderr appended to `github/watch.log`.
- **Message:** the run prints `watcher started (pid N); you can close this terminal. Log: <path>`.

## Behaviour

- Every 15 s, it does exactly what a manual check would do:
  - look for a valid approval for the pending step (the 004 rules);
  - look for A's merge, or the status-change PR merge;
  - then resume.
- When the run is no longer paused (succeeded, failed or torn down), it removes the pidfile and
  exits 0.
- **On a GitHub error:** it logs the error and keeps waiting. It never treats an error as an
  approval.

## Stop and status

- `pilot github watch --run <id> --stop` sends SIGTERM and removes the pidfile.
- `pilot status` adds `"watcher": {"pid": N, "running": true, "log": "<path>"}` while one runs.

## Foreground

`pilot github watch --run <id>`, with neither flag, runs the loop in the foreground until the
run ends or Ctrl-C. That's today's behaviour.

## Not covered

Surviving a reboot. The run stays paused, and `pilot resume` or a new `watch` picks it up. In
cluster mode the operator (011) owns this loop.
