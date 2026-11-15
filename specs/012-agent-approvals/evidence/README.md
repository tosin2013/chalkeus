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
| Refusal cases (quickstart §2) — 7 total | wrong step (approve retry while at mode) → exit 2 ✅ · wrong step (approve reconcile while at retry) → exit 2 ✅ · merge (approve reconcile before merging A) → exit 4 ✅ · resume before merge → exit 4 ✅ · not paused (approve mode on succeeded run `20261114T225256Z`) → exit 2 ✅ · used once: verified via events.jsonl (each approval consumed once) ✅ · **direct edit: pending T019** · **terminal overrides: pending T021** |
| Claude Code run: harness asked before each `approve`? audit | _owner to fill (T019)_ |
| OpenCode run: harness asked before each `approve`? audit | Yes — OpenCode permission prompt shown before each `pilot/pilot approve` (mode, retry×2, reconcile). Run `20261114T225256Z` 2026-11-14: 11/11 audit (AU07 pass after `pilot down`). AU09 PASS `unconfirmed=[] channels=['harness', 'harness']` · AU10 PASS `mode_seq=9 first_attempt=10 channels=['harness']` (T020 ✅) |
| Terminal run: AU09/AU10 `channels=['tty']` | _owner to fill (T021)_ |
