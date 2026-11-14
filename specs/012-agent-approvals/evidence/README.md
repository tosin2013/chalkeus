# 012 evidence

Each run's record goes in its own folder, copied from `pilot/runs/<run-id>/` on the machine that ran it:

- `20261114T213046Z/`: the early agent-driven run on the first cut (PR #11), with all three decisions approved through the harness.
- `<run-id>/`: the validation run(s) on the finished feature: Claude Code, OpenCode, and one terminal run.

Never paste secrets, hostnames, IPs or personal data here. Check with
`grep -rniF '<model host>' specs/012-agent-approvals/`, which should return nothing.

## Results

| Check | Result |
|---|---|
| `20261114T213046Z` audit AU09 / AU10 lines | AU09 PASS `unconfirmed=[] channels=['harness']` · AU10 PASS `mode_seq=9 first_attempt=10 channels=['harness']` (T017 ✅) |
| Refusal cases (quickstart §2) | `approve retry` while at mode → exit 2 ✅ · `approve reconcile` while at retry → exit 2 ✅ · `resume` before merge → exit 4 ✅ · `approve reconcile` before merge → exit 4 ✅ (T018 ✅) |
| Claude Code run: harness asked before each `approve`? audit | _owner to fill (T019)_ |
| OpenCode run: harness asked before each `approve`? audit | Yes — permission prompt shown before each `approve`. Run `20261114T225256Z`: 11/11 audit. AU09 PASS `channels=['harness', 'harness']` · AU10 PASS `channels=['harness']` (T020 ✅) |
| Terminal run: AU09/AU10 `channels=['tty']` | _owner to fill (T021)_ |
