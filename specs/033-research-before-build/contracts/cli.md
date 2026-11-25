# Contract: CLI, GitHub and doctor surfaces (033)

## `forge approve … replan`

```text
./chalkeus forge approve --run <id> replan
```

- Accepted only while the run is `paused(awaiting-retry)`; otherwise refused with the step the
  run is waiting for (as today).
- Harness: covered by the existing `Bash(./chalkeus forge approve:*)` ask rule; no new command
  in `internal/cli/command_audit.go`.
- Output: `recorded: end run <id> for planning (back to planning). Now run: ./chalkeus forge
  resume --run <id>` (or the watcher ends it).
- `resume` then prints `stopped: back to planning (by <who>, <channel>); the slice can be run
  again after the planning change is merged.` and exits 0.

## GitHub

- `/approve replan` on the run's tracking issue, by an allowed approver with write permission,
  while the run waits for a retry. `/withdraw replan` before the watcher acts withdraws it.
- The retry decision comment lists both: `reply /approve retry to retry, or /approve replan to
  end the run and go back to planning`.

## Run state

- New final state `stopped(back-to-planning)`; `forge status` prints it; step mode counts it as
  ended; the watcher exits; the grading namespace is removed.

## Audit

On a `stopped(back-to-planning)` run, AU02 and AU03 print `n/a  run ended by the owner for
planning`; every other check runs as before.

## Doctor (`forge doctor --target <t> --item NNN [--slice USn] [--baseline]`)

New lines in the `item` section (warnings never change the exit code):

```text
  WARN research tasks: T003 (tasks.md:26) "Locate the suites …"; resolve them in the plan's research first (docs/targets.md#research-tasks)
  WARN runtime: the slice names Node 22; toolchain node24 provides Node 24 (change it in your planning PR)
```

New lines in the `baseline` section (with `--baseline`):

```text
  PASS test-run time: one run of the test command took 65 s
  WARN test-run allowance: 34 full test runs fit in one attempt (deadline 2400 s)   # only below 5
  INFO test-run time: not recorded by this grader image                            # old image
```
