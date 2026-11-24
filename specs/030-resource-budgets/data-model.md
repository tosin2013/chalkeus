# Data model: Resource budgets and capacity checks

## Budget (resolved per run, stored in `run.json` → `budget`)

```json
{
  "worker":   {"cpu": "2", "memory": "2Gi", "deadline_s": 2400, "soft_margin_s": 120},
  "grader":   {"cpu": "4", "memory": "4Gi", "deadline_s": 1800, "command_timeout_s": 1200},
  "prefetch": {"cpu": "4", "memory": "4Gi", "deadline_s": 1800},
  "author":   {"cpu": "2", "memory": "2Gi", "deadline_s": 1800},
  "raised":   ["FORGE_WORKER_DEADLINE_S", "FORGE_GRADER_CPU", "FORGE_COMMAND_TIMEOUT_S"],
  "source":   "catalog:node24"
}
```

- `prefetch` always equals `grader`'s CPU, memory and job deadline (FR-004).
- Validation: CPU and memory are Kubernetes quantities; seconds are positive integers; every
  value ≥ the catalog's (FR-002). `soft_margin_s` is fixed (not a profile key).
- Lifecycle: resolved once at run start (or draft check start), written before the mode
  approval, read back on resume (FR-006).

## Catalog entry (toolchain catalog, extended)

`worker` and `author` gain `cpu`, `memory`, `deadline_s` with today's values (2, 2Gi, 1200; the
author's current deadline). `grader` is unchanged.

## Capacity result (`run.json` → `capacity`, `draft.json` → `capacity`, doctor area `capacity`)

```json
{"result": "pass|fail|warn", "role": "grader", "need": {"cpu": "4", "memory": "4Gi"},
 "have": {"node": "chalkeus-control-plane", "cpu": "3.5", "memory": "5.2Gi"},
 "cluster": "local|remote", "quota": null, "change": "<what to change>", "detail": "<warn reason>"}
```

## Attempt (evidence `attempts/<task>-<n>/`)

- `report.json` gains `stopped_by` (`"deadline"` | `"spend_cap"` | null); `spend` now present
  for deadline stops.
- Attempt state `failed`, reason `deadline`, with `spend` and `changed_paths` filled when the
  soft stop collected them.

## Events (new)

| Event | Stage | Detail |
|---|---|---|
| `budget_resolved` | prepare | the budget JSON (compact) |
| `capacity_checked` | prepare | `pass`/`fail`/`warn` and the role |
| `capacity_stop` | attempt | role, pod reason, waited seconds |
