# M1 pilot run 20261125T182055Z

**State:** paused(awaiting-retry)

**Elapsed:** 2494 s total, 2446 s excluding the human merge wait

**Mode:** `{"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200}`

## Attempts

| Task | Attempt | Induction | Grade | Cost (USD) | Tokens |
|---|---|---|---|---|---|
| A | 1 | withheld-acceptance-section | fail (grade_fail) | 0.052461 | 30907 |
| A | 2 | - | pass | 0.018165 | 13836 |
| B | 1 | - | fail (deadline) | - | - |

## Draft PRs

- Task A: [https://github.com/tosin2013/craz3-pilot-fixture/pull/76](https://github.com/tosin2013/craz3-pilot-fixture/pull/76) (draft=True, author=tosins-specfoundry-pilot[bot])
- `checks/results.jsonl`: 28 of 30 pass

## Events

| Seq | Stage | Task | Attempt | Event | Detail |
|---|---|---|---|---|---|
| 1 | run |  |  | run_started | tosin2013/craz3-pilot-fixture |
| 2 | gate | A |  | gate_allow | seq=1 OK |
| 3 | gate | B |  | gate_refuse | seq=2 DEP_UNSATISFIED:001 |
| 4 | prepare |  |  | budget_resolved | {"worker":{"cpu":"2","memory":"2Gi","deadline_s":1200,"soft_margin_s":120},"raised":[],"source":"default"} |
| 5 | approval |  |  | approval_channel | step=mode approved=false channel=none |
| 6 | mode |  |  | mode_declined | {"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200} |
| 7 | run |  |  | run_paused(awaiting-mode) |  |
| 8 | approval |  |  | approval_recorded | step=mode channel=harness |
| 9 | run |  |  | run_resumed | paused(awaiting-mode) |
| 10 | approval |  |  | approval_channel | step=mode approved=true channel=harness |
| 11 | mode |  |  | mode_confirmed | {"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200} |
| 12 | attempt | A | 1 | attempt_created | ns=pilot-w-a-1 induction=withheld-acceptance-section |
| 13 | attempt | A | 1 | attempt_running | ghcr.io/tosin2013/chalkeus-worker@sha256:7f88c9652600739f47aa508fd1e0fb690050843a14ab8fbe3c7b606e29d52ba3 |
| 14 | attempt | A | 1 | attempt_collected |  |
| 15 | grade | A | 1 | grade_started | pilot-g-a-1 |
| 16 | grade | A | 1 | grade_fail | G5_hidden_tests |
| 17 | gate | B |  | gate_refuse | seq=3 DEP_UNSATISFIED:001 |
| 18 | approval | A | 2 | approval_channel | step=retry approved=false channel=none |
| 19 | retry | A | 2 | retry_declined |  |
| 20 | run |  |  | run_paused(awaiting-retry) |  |
| 21 | run |  |  | run_resumed | paused(awaiting-retry) |
| 22 | approval | A | 2 | approval_channel | step=retry approved=false channel=none |
| 23 | retry | A | 2 | retry_declined |  |
| 24 | run |  |  | run_paused(awaiting-retry) |  |
| 25 | run |  |  | run_resumed | paused(awaiting-retry) |
| 26 | approval | A | 2 | approval_channel | step=retry approved=false channel=none |
| 27 | retry | A | 2 | retry_declined |  |
| 28 | run |  |  | run_paused(awaiting-retry) |  |
| 29 | approval | A | 2 | approval_recorded | step=retry channel=harness |
| 30 | run |  |  | run_resumed | paused(awaiting-retry) |
| 31 | approval | A | 2 | approval_channel | step=retry approved=true channel=harness |
| 32 | retry | A | 2 | retry_confirmed |  |
| 33 | attempt | A | 2 | attempt_created | ns=pilot-w-a-2 induction=None |
| 34 | attempt | A | 2 | attempt_running | ghcr.io/tosin2013/chalkeus-worker@sha256:7f88c9652600739f47aa508fd1e0fb690050843a14ab8fbe3c7b606e29d52ba3 |
| 35 | attempt | A | 2 | attempt_collected |  |
| 36 | grade | A | 2 | grade_started | pilot-g-a-2 |
| 37 | grade | A | 2 | grade_pass |  |
| 38 | publish | A | 2 | publish_started | pilot/20261125T182055Z/a |
| 39 | publish | A | 2 | pr_opened | https://github.com/tosin2013/craz3-pilot-fixture/pull/76 |
| 40 | gate | B |  | gate_refuse | seq=4 DEP_UNSATISFIED:001 |
| 41 | run |  |  | run_paused(awaiting-merge) |  |
| 42 | run |  |  | run_resumed | paused(awaiting-merge) |
| 43 | approval | A |  | approval_channel | step=reconcile approved=false channel=none |
| 44 | reconcile | A | 2 | reconcile_DECLINED |  |
| 45 | run |  |  | run_paused(awaiting-merge) |  |
| 46 | approval | A |  | approval_recorded | step=reconcile channel=harness |
| 47 | run |  |  | run_resumed | paused(awaiting-merge) |
| 48 | approval | A |  | approval_channel | step=reconcile approved=true channel=harness |
| 49 | reconcile | A | 2 | reconcile_applied | 0d5b38acfd6a2c24bb02a1df23ef54d041bf33cb |
| 50 | reconcile | A |  | merge_wait_seconds | 48 |
| 51 | status | A |  | task_verified | 0d5b38acfd6a2c24bb02a1df23ef54d041bf33cb |
| 52 | gate | B |  | gate_allow | seq=5 OK |
| 53 | attempt | B | 1 | attempt_created | ns=pilot-w-b-1 induction=None |
| 54 | attempt | B | 1 | attempt_running | ghcr.io/tosin2013/chalkeus-worker@sha256:7f88c9652600739f47aa508fd1e0fb690050843a14ab8fbe3c7b606e29d52ba3 |
| 55 | attempt | B | 1 | attempt_failed | deadline |
| 56 | approval | B | 2 | approval_channel | step=retry approved=false channel=none |
| 57 | retry | B | 2 | retry_declined |  |
| 58 | run |  |  | run_paused(awaiting-retry) |  |

## Known limits

- Gate runs on the host (no published spec-status CLI image yet, PRD §8 ask 3).
- Grader uses `spec status --json`, not `verify-readiness` (report-only skill mode not shipped, PRD §8 ask 1).
- Reconcile is a deterministic pilot equivalent of the spec-status reconcile skill.
- Spend cap is enforced by the pilot watchdog and deadline unless a provider-side cap is declared (R2).
- Single-node kind only; multi-node policy enforcement not covered.
