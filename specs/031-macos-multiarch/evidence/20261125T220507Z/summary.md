# pilot item run 20261125T220507Z

**State:** succeeded

**Elapsed:** 663 s total, 571 s excluding the human merge wait

**Mode:** `{"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200}`

## Attempts

| Task | Attempt | Induction | Grade | Cost (USD) | Tokens |
|---|---|---|---|---|---|
| 011 | 1 | - | pass | 0.087891 | 67742 |

## Draft PRs

- Task 011: [https://github.com/tosin2013/stockroom/pull/55](https://github.com/tosin2013/stockroom/pull/55) (draft=True, author=tosins-specfoundry-pilot[bot])
- `checks/results.jsonl`: 17 of 17 pass
- `audit.jsonl`: 11 of 11 pass
- `teardown.jsonl`: 3 of 3 pass

## Events

| Seq | Stage | Task | Attempt | Event | Detail |
|---|---|---|---|---|---|
| 1 | run |  |  | run_started | tosin2013/stockroom |
| 2 | gate | 011 |  | gate_allow | seq=1 OK |
| 3 | prepare |  |  | budget_resolved | {"worker":{"cpu":"2","memory":"2Gi","deadline_s":1200,"soft_margin_s":120},"raised":[],"source":"catalog:jdk21-maven"} |
| 4 | approval |  |  | approval_channel | step=mode approved=false channel=none |
| 5 | mode |  |  | mode_declined | {"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200} |
| 6 | run |  |  | run_paused(awaiting-mode) |  |
| 7 | approval |  |  | approval_recorded | step=mode channel=harness |
| 8 | run |  |  | run_resumed | paused(awaiting-mode) |
| 9 | approval |  |  | approval_channel | step=mode approved=true channel=harness |
| 10 | mode |  |  | mode_confirmed | {"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200} |
| 11 | prepare | 011 |  | toolchain_ok | jdk21-maven Apache Maven 3.9.16 (2bdd9fddda4b155ebf8000e807eb73fd829a51d5) (catalog) |
| 12 | prepare | 011 |  | prefetch_pass |  |
| 13 | baseline | 011 |  | baseline_pass |  |
| 14 | attempt | 011 | 1 | attempt_created | ns=pilot-w-011-1 induction=None |
| 15 | attempt | 011 | 1 | attempt_running | ghcr.io/tosin2013/chalkeus-worker@sha256:fca83917f97bbb66296947b9d40e86e61be4852b061a03368559e182ccfcecd6 |
| 16 | attempt | 011 | 1 | attempt_collected |  |
| 17 | grade | 011 | 1 | grade_started | pilot-g-011 |
| 18 | grade | 011 | 1 | grade_pass |  |
| 19 | publish | 011 | 1 | change_replayed | feb9fbc18f9cba11ef56446adf21729e6f05f4ec |
| 20 | publish | 011 | 1 | publish_started | pilot/20261125T220507Z/011 |
| 21 | publish | 011 | 1 | pr_opened | https://github.com/tosin2013/stockroom/pull/55 |
| 22 | run |  |  | run_paused(awaiting-merge) |  |
| 23 | run |  |  | run_resumed | paused(awaiting-merge) |
| 24 | approval | 011 |  | approval_channel | step=reconcile approved=false channel=none |
| 25 | reconcile | 011 | 1 | reconcile_DECLINED |  |
| 26 | run |  |  | run_paused(awaiting-merge) |  |
| 27 | approval | 011 |  | approval_recorded | step=reconcile channel=harness |
| 28 | run |  |  | run_resumed | paused(awaiting-merge) |
| 29 | approval | 011 |  | approval_channel | step=reconcile approved=true channel=harness |
| 30 | reconcile | 011 | 1 | reconcile_applied | ecf37bb70ef29e9c7a8579e9f118fc0832643244 |
| 31 | reconcile | 011 |  | merge_wait_seconds | 92 |
| 32 | status | 011 |  | task_verified | ecf37bb70ef29e9c7a8579e9f118fc0832643244 |
| 33 | audit |  |  | audit_done | 10/10 |
| 34 | run |  |  | run_succeeded |  |
| 35 | teardown |  |  | torn_down | teardown.jsonl |
| 36 | audit |  |  | audit_done | 11/11 |

## Known limits

- Gate runs on the host (no published spec-status CLI image yet, PRD §8 ask 3).
- Grader uses `spec status --json`, not `verify-readiness` (report-only skill mode not shipped, PRD §8 ask 1).
- Reconcile is a deterministic pilot equivalent of the spec-status reconcile skill.
- Spend cap is enforced by the pilot watchdog and deadline unless a provider-side cap is declared (R2).
- Single-node kind only; multi-node policy enforcement not covered.
