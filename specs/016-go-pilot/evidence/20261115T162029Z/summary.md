# M1 pilot run 20261115T162029Z

**State:** succeeded

**Elapsed:** 816 s total, 676 s excluding the human merge wait

**Mode:** `{"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200}`

## Attempts

| Task | Attempt | Induction | Grade | Cost (USD) | Tokens |
|---|---|---|---|---|---|
| A | 1 | withheld-acceptance-section | fail (grade_fail) | 0.0 | 248504 |
| A | 2 | - | pass | 0.0 | 161890 |
| B | 1 | - | pass | 0.0 | 501840 |

## Draft PRs

- Task A: [https://github.com/tosin2013/craz3-pilot-fixture/pull/21](https://github.com/tosin2013/craz3-pilot-fixture/pull/21) (draft=True, author=tosin2013)
- Task B: [https://github.com/tosin2013/craz3-pilot-fixture/pull/22](https://github.com/tosin2013/craz3-pilot-fixture/pull/22) (draft=True, author=tosin2013)
- `checks/results.jsonl`: 32 of 32 pass
- `audit.jsonl`: 11 of 11 pass
- `teardown.jsonl`: 3 of 3 pass

## Events

| Seq | Stage | Task | Attempt | Event | Detail |
|---|---|---|---|---|---|
| 1 | run |  |  | run_started | tosin2013/craz3-pilot-fixture |
| 2 | gate | A |  | gate_allow | seq=1 OK |
| 3 | gate | B |  | gate_refuse | seq=2 DEP_UNSATISFIED:001 |
| 4 | approval |  |  | approval_channel | step=mode approved=false channel=none |
| 5 | mode |  |  | mode_declined | {"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200} |
| 6 | run |  |  | run_paused(awaiting-mode) |  |
| 7 | approval |  |  | approval_recorded | step=mode channel=harness |
| 8 | run |  |  | run_resumed | paused(awaiting-mode) |
| 9 | approval |  |  | approval_channel | step=mode approved=true channel=harness |
| 10 | mode |  |  | mode_confirmed | {"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200} |
| 11 | attempt | A | 1 | attempt_created | ns=pilot-w-a-1 induction=withheld-acceptance-section |
| 12 | attempt | A | 1 | attempt_running | ghcr.io/tosin2013/specfoundry-pilot-worker@sha256:7493bf1bc217ec34399f8f0bc058818e162502d192760ce3cfd2e07a09611ad4 |
| 13 | attempt | A | 1 | attempt_collected |  |
| 14 | grade | A | 1 | grade_started | pilot-g-a-1 |
| 15 | grade | A | 1 | grade_fail | G5_hidden_tests |
| 16 | gate | B |  | gate_refuse | seq=3 DEP_UNSATISFIED:001 |
| 17 | approval | A | 2 | approval_channel | step=retry approved=false channel=none |
| 18 | retry | A | 2 | retry_declined |  |
| 19 | run |  |  | run_paused(awaiting-retry) |  |
| 20 | approval | A | 2 | approval_recorded | step=retry channel=harness |
| 21 | run |  |  | run_resumed | paused(awaiting-retry) |
| 22 | approval | A | 2 | approval_channel | step=retry approved=true channel=harness |
| 23 | retry | A | 2 | retry_confirmed |  |
| 24 | attempt | A | 2 | attempt_created | ns=pilot-w-a-2 induction=None |
| 25 | attempt | A | 2 | attempt_running | ghcr.io/tosin2013/specfoundry-pilot-worker@sha256:7493bf1bc217ec34399f8f0bc058818e162502d192760ce3cfd2e07a09611ad4 |
| 26 | attempt | A | 2 | attempt_collected |  |
| 27 | grade | A | 2 | grade_started | pilot-g-a-2 |
| 28 | grade | A | 2 | grade_pass |  |
| 29 | publish | A | 2 | publish_started | pilot/20261115T162029Z/a |
| 30 | publish | A | 2 | pr_opened | https://github.com/tosin2013/craz3-pilot-fixture/pull/21 |
| 31 | gate | B |  | gate_refuse | seq=4 DEP_UNSATISFIED:001 |
| 32 | run |  |  | run_paused(awaiting-merge) |  |
| 33 | run |  |  | run_resumed | paused(awaiting-merge) |
| 34 | approval | A |  | approval_channel | step=reconcile approved=false channel=none |
| 35 | reconcile | A | 2 | reconcile_DECLINED |  |
| 36 | run |  |  | run_paused(awaiting-merge) |  |
| 37 | approval | A |  | approval_recorded | step=reconcile channel=harness |
| 38 | run |  |  | run_resumed | paused(awaiting-merge) |
| 39 | approval | A |  | approval_channel | step=reconcile approved=true channel=harness |
| 40 | reconcile | A | 2 | reconcile_applied | c4ac53493b8fc133a85e0daa0d651b78d3a1d983 |
| 41 | reconcile | A |  | merge_wait_seconds | 140 |
| 42 | status | A |  | task_verified | c4ac53493b8fc133a85e0daa0d651b78d3a1d983 |
| 43 | gate | B |  | gate_allow | seq=5 OK |
| 44 | attempt | B | 1 | attempt_created | ns=pilot-w-b-1 induction=None |
| 45 | attempt | B | 1 | attempt_running | ghcr.io/tosin2013/specfoundry-pilot-worker@sha256:7493bf1bc217ec34399f8f0bc058818e162502d192760ce3cfd2e07a09611ad4 |
| 46 | attempt | B | 1 | attempt_collected |  |
| 47 | grade | B | 1 | grade_started | pilot-g-b-1 |
| 48 | grade | B | 1 | grade_pass |  |
| 49 | publish | B | 1 | publish_started | pilot/20261115T162029Z/b |
| 50 | publish | B | 1 | pr_opened | https://github.com/tosin2013/craz3-pilot-fixture/pull/22 |
| 51 | audit |  |  | audit_done | 10/10 |
| 52 | run |  |  | run_succeeded |  |
| 53 | teardown |  |  | torn_down | teardown.jsonl |
| 54 | audit |  |  | audit_done | 11/11 |

## Known limits

- Gate runs on the host (no published spec-status CLI image yet, PRD §8 ask 3).
- Grader uses `spec status --json`, not `verify-readiness` (report-only skill mode not shipped, PRD §8 ask 1).
- Reconcile is a deterministic pilot equivalent of the spec-status reconcile skill.
- Spend cap is enforced by the pilot watchdog and deadline unless a provider-side cap is declared (R2).
- Single-node kind only; multi-node policy enforcement not covered.
