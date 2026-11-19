# Contract: `pilot step`

```text
pilot step --target <name> [--once]
```

- **Refuses (exit 2) unless:**
  - `PILOT_GITHUB_ISSUES=on` and `PILOT_GITHUB_APPROVALS=on` for the target;
  - `PILOT_APPROVERS` is set;
  - `pilot up` has completed.
- **Polls** every `PILOT_STEP_INTERVAL_S` (default 120, at least 30). `--once` does one pass:
  - exit 0 when nothing failed;
  - exit 1 when an item's doctor or run failed in that pass.
- **Syncs** the target's item issues at the start of each pass (004), so new items can be
  labelled.
- **Prints one line per decision:** `NNN: <step> (<reason>)`.
- **Default verbosity:** `-vv`.

## Profile keys

| Key | Default | Meaning |
|---|---|---|
| `PILOT_PROPOSE_LABEL` | "pilot/go" | the label that proposes an item (name follows Q7) |
| `PILOT_STEP_INTERVAL_S` | `120` | seconds between passes |

## Decisions (`step.Next`)

| Item state | Action |
|---|---|
| not proposed, blocked, or another item in flight | skip |
| no hidden tests, auto-draft on, no open tests PR | draft (024, this item only) |
| no hidden tests, auto-draft off | comment `needs-tests` (once) |
| tests PR open | wait |
| tests merged, no run, doctor not failed at these commits | doctor; on failure comment `doctor-failed` (once); on success run |
| run paused or running | wait |
| run succeeded | done (go to the next item) |
| run failed | comment `run-failed` (once); leave until relabelled |

## Evidence

`<evidence root>/step/<target>/decisions.jsonl`, one JSON line per action:
- `at`, `item`, `action`, `reason`;
- `run` (the run folder, if one started);
- `comment` (the marker, if one was posted).

It holds no secrets and no test content. A run started by step mode records
`run_started_by step`.
