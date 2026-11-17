# Contract: readiness and baseline refusals

- **Gate refusals:** all exit 10 (gate refuse). Each is recorded as a gate row and a
  `gate_refuse` event with the reason code. No worker, prefetch or grader runs.
- **Baseline refusals:** they happen after the mode approval, before any worker. They exit 1
  (FAIL) with the run state `failed`, reason `BASELINE_*`.

| Order | Code | Rule |
|---|---|---|
| 0 | existing gate codes (`STATUS_TOOL_ERROR`, `SCHEMA`, `NOT_STARTABLE`, `DEP_UNSATISFIED:<id>`, …) | `spec status` on a fresh clone. Unreadable means refuse. |
| 1 | `ITEM_NOT_ON_ROADMAP` | `--item NNN` is not on the target's roadmap, or is superseded |
| 2 | `ITEM_SPEC_MISSING` / `ITEM_TASKS_MISSING` | `<spec dir>/spec.md` or `tasks.md` is absent |
| 3 | `ITEM_OPEN_QUESTIONS` | `[NEEDS CLARIFICATION` in `spec.md`; an error diagnostic on the item; or an unresolved roadmap open question that names `NNN` |
| 4 | `ITEM_NO_ALLOWED_PATHS` / `ITEM_PATHS_TOO_WIDE` | see [item-spec.md](item-spec.md) |
| 5 | `ITEM_NO_HIDDEN_TESTS` | `items/NNN/` is missing or empty in the companion repository; or the companion repository can't be read |
| 6 | `INPUT_TOO_LARGE` | the snapshot bundle exceeds `PILOT_MAX_INPUT_MB` |
| 7 | `OPEN_ITEM_PR` | an open `pilot/*/<NNN>` PR exists on the target |
| — | `BASELINE_SUITE_FAILS` | after approval: the project's test command fails on the base revision |
| — | `BASELINE_HIDDEN_PASSES` | after approval: the hidden check passes on the base revision (no signal) |
| — | `PREFETCH_FAILED` | after approval: dependency resolution failed (the registry is unreachable, or the build is broken) |
