# Contract: GitHub loop commands (amends 002 `contracts/cli.md` and 012 `contracts/cli-approve.md`)

## New commands

| Command | Does | Exit |
|---|---|---|
| `pilot github sync` | create missing item issues and labels; update labels and bodies from the roadmap; idempotent | 0 done (prints counts), 2 error (nothing half-written for the failing object; earlier objects stay) |
| `pilot github drift [--run <id>]` | print the drift report; with `--run`, write `github/drift.json` | 0 no drift, 1 drift found, 2 error |
| `pilot github watch --run <id>` | poll the tracking issue every 15 s; resume when a valid approval or the status-PR merge appears | as `resume`; Ctrl-C stops watching and leaves the run paused |
| `pilot app-token --check` | mint an App token, run the write probe, discard the token | 0 ok, 1 not ok |
| `pilot app-key` | store the App private key (mode 600) from stdin; kept only if the probe passes | 0 saved, 1 not saved |

## Changed behaviour (only with `PILOT_GITHUB_APPROVALS=on`)

- On a pause at `mode` or `retry`, the run also posts the decision on the tracking issue (created
  on first use) and prints its URL alongside the 012 hint.
- `resume` with no terminal also reads the tracking issue (research R3). A valid approval is
  recorded with `channel=github` and used once; otherwise the run pauses again (exit 5 or 6)
  and prints why each candidate comment was rejected.
- With a terminal, the terminal decides (012). Any GitHub approval for that decision is
  discarded, and the decision post is marked answered at the terminal.
- After A's merge, reconcile opens a status-change PR and pauses (`awaiting-status-merge`,
  exit 4). Once the human merges it, `resume` checks it and continues (research R6).
- Any GitHub error leaves the run paused (exit 2 for the error, state unchanged).

## Unchanged

- `pilot approve` (harness channel) and terminal prompts work as in 012.
- The pilot never merges. There is no `/approve merge`; `/approve reconcile` is not
  accepted, because the status change is approved by merging its PR.
- The publisher PR body now ends with `Closes #<item issue>`.
