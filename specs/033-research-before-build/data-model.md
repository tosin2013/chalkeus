# Data model: Research before build (033)

## Signal

| Field | Type | Notes |
|---|---|---|
| `kind` | enum | `ended` (how the attempt ended), `research_task`, `outside_paths`, `manifest_outside_paths`, `egress_refused`, `install_error`, `unknown_reason`, `allowance` |
| `detail` | string | one line, for example `T003: "Locate the suites that leave handles open"` |
| `source` | string | where it came from: `prompt.md:345`, `report.json changed_paths`, `proxy-denials.log` (`target proxy`, 1×), `grade.json G4` |
| `trigger` | enum or empty | `unknown_work`, `environment_need`, or empty (information only) |

Rules:
- `research_task`: an unchecked task in the attempt's task list whose description starts with a
  research phrase (research R3) → `unknown_work`.
- `unknown_reason`: the attempt's `reason` (or report error) is not in the catalogue (R4) →
  `unknown_work`.
- `manifest_outside_paths`: a changed path outside the allowed paths that is a manifest or lock
  file for the toolchain (R6) → `environment_need`. Other paths outside → `outside_paths`, no
  trigger.
- `egress_refused`: refusals other than the probes (R5) → `environment_need` only together with
  an `install_error` or a `manifest_outside_paths` signal; otherwise no trigger.
- `allowance`: the test-run allowance when a baseline time is recorded; no trigger, used for the
  budget choice.

## Diagnosis

| Field | Type | Notes |
|---|---|---|
| `version` | int | `1` |
| `task`, `attempt` | string, int | the attempt diagnosed |
| `ended` | string | the attempt's `reason` (`deadline`, `grade_fail`, …) |
| `usage` | object | `tokens`, `cost_usd`, `seconds` when kept |
| `signals` | list of Signal | in the order of the table above |
| `triggers` | list | distinct triggers from the signals |
| `recommend` | enum | `retry`, `raise_budget`, `back_to_planning` |
| `routes` | list | for `back_to_planning`: `plan_research` (unknown work), `planning_pr` (environment need), `roadmap_item` (scope) |
| `text` | string | the rendered block shown with the retry question (contracts/diagnosis.md) |

Recommendation (FR-003), first match wins:
1. any trigger → `back_to_planning`, routes from the triggers;
2. `ended` is `deadline` or `spend_cap` and the allowance is known and below 5 →
   `raise_budget`;
3. otherwise → `retry`.

Stored in `attempts/<task>-<n>/diagnosis.json`; the attempt record in `run.json` gains
`diagnosis: {recommend, triggers}`; `run.State["decision_note"]` carries `text`.

## Planning decision

An approval record with step `replan` (`approvals/pending.json`, event `approval_recorded`),
then events `approval_channel step=replan`, `run_stopped back-to-planning`; `run.State`:
`state = "stopped(back-to-planning)"`, `stopped = {by, channel, at, task, attempt}`.

State transitions:

```text
paused(awaiting-retry) --approve retry--> in-progress (next attempt)
paused(awaiting-retry) --approve replan--> stopped(back-to-planning)   [final]
```

`stopped(...)` is final everywhere `failed` and `succeeded` are (step mode, watcher, status).

## Baseline timing

| Field | Where | Notes |
|---|---|---|
| `test_seconds` | grader report, G4 detail | seconds for one run of the project test command |
| `worker_deadline_s` | resolved budget | `images.Resolve(...).Worker.DeadlineS` |
| `allowance` | computed | `floor((worker_deadline_s − 120) / test_seconds)`; unknown if no `test_seconds` |
