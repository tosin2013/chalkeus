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
| Refusal cases (quickstart §2) — 7 total | wrong step (approve retry while at mode) → exit 2 ✅ · wrong step (approve reconcile while at retry) → exit 2 ✅ · merge (approve reconcile before merging A) → exit 4 ✅ · resume before merge → exit 4 ✅ · not paused (approve mode on succeeded run `20261114T225256Z`) → exit 2 ✅ · used once: verified via events.jsonl (each approval consumed once) ✅ · direct edit: agent refused to write `approvals/pending.json` per AGENTS.md rule 2 (T019) ✅ · terminal overrides: seq 28 approve recorded (harness) → seq 30 resume in terminal answered N (tty, declined) → seq 34 setsid headless resume paused again (none, declined) → seq 38 resume in terminal answered y (tty, confirmed) ✅ (T018 ✅) |
| Terminal run: AU09/AU10 `channels=['tty']` | Run `20261115T023426Z` 2026-11-15: 11/11 audit (AU07 pass). AU09 PASS `unconfirmed=[] channels=['tty', 'tty']` · AU10 PASS `mode_seq=4 first_attempt=5 channels=['tty']`. Terminal-overrides sequence: seq 28 harness approval recorded → seq 30 tty N (declined, paused) → seq 34 setsid headless paused again (none) → seq 38 tty y (confirmed, run continued) (T021 ✅) |
| OpenCode run 2 (T019 — run in OpenCode, not Claude Code; noted): harness asked before each `approve`? audit | Yes — OpenCode permission prompt shown before each `pilot/pilot approve` (mode, retry, reconcile). Direct-edit refused by agent (AGENTS.md rule 2). Run `20261115T020750Z` 2026-11-15: 11/11 audit (AU07 pass). AU09 PASS `unconfirmed=[] channels=['harness']` · AU10 PASS `mode_seq=9 first_attempt=10 channels=['harness']` (T019 ✅ in OpenCode) |
| OpenCode run: harness asked before each `approve`? audit | Yes — OpenCode permission prompt shown before each `pilot/pilot approve` (mode, retry×2, reconcile). Run `20261114T225256Z` 2026-11-14: 11/11 audit (AU07 pass after `pilot down`). AU09 PASS `unconfirmed=[] channels=['harness', 'harness']` · AU10 PASS `mode_seq=9 first_attempt=10 channels=['harness']` (T020 ✅) |
| Terminal run: AU09/AU10 `channels=['tty']` | see row above (T021 ✅) |
