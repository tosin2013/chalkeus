# M1 pilot run 20261125T191440Z

**State:** succeeded

**Elapsed:** 2066 s total, 1930 s excluding the human merge wait

**Mode:** `{"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200}`

## Attempts

| Task | Attempt | Induction | Grade | Cost (USD) | Tokens |
|---|---|---|---|---|---|
| A | 1 | withheld-acceptance-section | fail (grade_fail) | 0.048423 | 24135 |
| A | 2 | - | pass | 0.011721 | 6294 |
| B | 1 | - | pass | 0.016957 | 10066 |

## Draft PRs

- Task A: [https://github.com/tosin2013/craz3-pilot-fixture/pull/77](https://github.com/tosin2013/craz3-pilot-fixture/pull/77) (draft=True, author=tosins-specfoundry-pilot[bot])
- Task B: [https://github.com/tosin2013/craz3-pilot-fixture/pull/78](https://github.com/tosin2013/craz3-pilot-fixture/pull/78) (draft=True, author=tosins-specfoundry-pilot[bot])
- `checks/results.jsonl`: 32 of 32 pass
- `audit.jsonl`: 11 of 11 pass
- `teardown.jsonl`: 3 of 3 pass

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
| 21 | approval | A | 2 | approval_recorded | step=retry channel=harness |
| 22 | run |  |  | run_resumed | paused(awaiting-retry) |
| 23 | approval | A | 2 | approval_channel | step=retry approved=true channel=harness |
| 24 | retry | A | 2 | retry_confirmed |  |
| 25 | attempt | A | 2 | attempt_created | ns=pilot-w-a-2 induction=None |
| 26 | attempt | A | 2 | attempt_running | ghcr.io/tosin2013/chalkeus-worker@sha256:7f88c9652600739f47aa508fd1e0fb690050843a14ab8fbe3c7b606e29d52ba3 |
| 27 | attempt | A | 2 | attempt_collected |  |
| 28 | grade | A | 2 | grade_started | pilot-g-a-2 |
| 29 | grade | A | 2 | grade_pass |  |
| 30 | publish | A | 2 | publish_started | pilot/20261125T191440Z/a |
| 31 | publish | A | 2 | pr_opened | https://github.com/tosin2013/craz3-pilot-fixture/pull/77 |
| 32 | gate | B |  | gate_refuse | seq=4 DEP_UNSATISFIED:001 |
| 33 | run |  |  | run_paused(awaiting-merge) |  |
| 34 | run |  |  | run_resumed | paused(awaiting-merge) |
| 35 | approval | A |  | approval_channel | step=reconcile approved=false channel=none |
| 36 | reconcile | A | 2 | reconcile_DECLINED |  |
| 37 | run |  |  | run_paused(awaiting-merge) |  |
| 38 | approval | A |  | approval_recorded | step=reconcile channel=harness |
| 39 | run |  |  | run_resumed | paused(awaiting-merge) |
| 40 | approval | A |  | approval_channel | step=reconcile approved=true channel=harness |
| 41 | reconcile | A | 2 | reconcile_applied | ef6979a6b4c52d2b0d68805562165338ceb918dd |
| 42 | reconcile | A |  | merge_wait_seconds | 136 |
| 43 | status | A |  | task_verified | ef6979a6b4c52d2b0d68805562165338ceb918dd |
| 44 | gate | B |  | gate_allow | seq=5 OK |
| 45 | attempt | B | 1 | attempt_created | ns=pilot-w-b-1 induction=None |
| 46 | attempt | B | 1 | attempt_running | ghcr.io/tosin2013/chalkeus-worker@sha256:7f88c9652600739f47aa508fd1e0fb690050843a14ab8fbe3c7b606e29d52ba3 |
| 47 | attempt | B | 1 | attempt_collected |  |
| 48 | grade | B | 1 | grade_started | pilot-g-b-1 |
| 49 | grade | B | 1 | grade_pass |  |
| 50 | publish | B | 1 | publish_started | pilot/20261125T191440Z/b |
| 51 | publish | B | 1 | pr_opened | https://github.com/tosin2013/craz3-pilot-fixture/pull/78 |
| 52 | audit |  |  | audit_done | 10/10 |
| 53 | run |  |  | run_succeeded |  |
| 54 | teardown |  |  | torn_down | teardown.jsonl |
| 55 | audit |  |  | audit_done | 11/11 |

## Known limits

- Gate runs on the host (no published spec-status CLI image yet, PRD §8 ask 3).
- Grader uses `spec status --json`, not `verify-readiness` (report-only skill mode not shipped, PRD §8 ask 1).
- Reconcile is a deterministic pilot equivalent of the spec-status reconcile skill.
- Spend cap is enforced by the pilot watchdog and deadline unless a provider-side cap is declared (R2).
- Single-node kind only; multi-node policy enforcement not covered.
