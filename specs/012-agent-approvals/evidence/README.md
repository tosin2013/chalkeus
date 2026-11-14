# 012 evidence

Each run's record goes in its own folder, copied from `pilot/runs/<run-id>/` on the machine that ran it:

- `20261114T213046Z/`: the early agent-driven run on the first cut (PR #11), with all three decisions approved through the harness.
- `<run-id>/`: the validation run(s) on the finished feature: Claude Code, OpenCode, and one terminal run.

Never paste secrets, hostnames, IPs or personal data here. Check with
`grep -rniF '<model host>' specs/012-agent-approvals/`, which should return nothing.

## Results

| Check | Result |
|---|---|
| `20261114T213046Z` audit AU09 / AU10 lines | _owner to fill (T017)_ |
| Refusal cases (quickstart §2) | _owner to fill (T018)_ |
| Claude Code run: harness asked before each `approve`? audit | _owner to fill (T019)_ |
| OpenCode run: harness asked before each `approve`? audit | _owner to fill (T020)_ |
| Terminal run: AU09/AU10 `channels=['tty']` | _owner to fill (T021)_ |
