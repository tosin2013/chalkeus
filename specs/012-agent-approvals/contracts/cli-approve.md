# Contract: approvals in the `pilot` CLI (amends 002 `contracts/cli.md`)

## New command

| Command | Does | Exit codes |
|---|---|---|
| `pilot approve --run <id> mode\|retry\|reconcile` | Records the human's yes for the one step the run is paused at (data-model.md). Prints what was approved and the `resume` command to run next. | 0 recorded · 2 not paused, unknown run, or a different step is pending · 4 the run waits for a merge (never approvable here) |

`--run` is required. `approve` stages nothing and changes no roadmap; only `resume` acts on it.

## Changed behaviour of `run` / `resume`

| Situation | Before 012 | After 012 |
|---|---|---|
| mode prompt, terminal, `y` | continue | unchanged |
| mode prompt, terminal, anything else | run failed `MODE_NOT_CONFIRMED`, exit 5 | unchanged |
| mode prompt, **no terminal**, no approval | run failed `MODE_NOT_CONFIRMED`, exit 5 | run `paused(awaiting-mode)`, exit 5, prints the `approve` hint |
| mode prompt, no terminal, matching approval | (n/a) | continue; channel `harness` |
| retry prompt, no terminal, no approval | paused `awaiting-retry`, exit 6 | unchanged, plus the `approve` hint |
| reconcile prompt, no terminal, no approval | paused `awaiting-merge`, exit 5 | unchanged, plus `waiting for your approval…` and the hint |
| any prompt with a terminal | terminal answer decides | unchanged; a pending approval is discarded (R5) |

`resume` also resumes `paused(awaiting-mode)`: it re-shows the mode and decides again.

## Rules (in addition to 002's)

- No command approves a merge.
- An approval applies to exactly one pending step and is used at most once (data-model.md).
- With a terminal, the terminal decides.
- The project ships harness rules that make `pilot approve` ask the human
  (`.claude/settings.json`, `opencode.json`) and deny direct edits to `approvals/`.
