# M1 pilot run 20261125T162546Z

**State:** succeeded

**Elapsed:** 1705 s total, 1705 s excluding the human merge wait

**Mode:** `{"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200}`

## Attempts

| Task | Attempt | Induction | Grade | Cost (USD) | Tokens |
|---|---|---|---|---|---|
| A | 1 | withheld-acceptance-section | fail (grade_fail) | 0.0 | 53091 |
| A | 2 | - | pass | 0.0 | 17531 |
| B | 1 | - | pass | 0.0 | 32477 |

## Draft PRs

- Task A: [https://github.com/tosin2013/craz3-pilot-fixture/pull/69](https://github.com/tosin2013/craz3-pilot-fixture/pull/69) (draft=True, author=tosins-specfoundry-pilot[bot])
- Task A: [https://github.com/tosin2013/craz3-pilot-fixture/pull/70](https://github.com/tosin2013/craz3-pilot-fixture/pull/70) (draft=True, author=tosins-specfoundry-pilot[bot])
- Task B: [https://github.com/tosin2013/craz3-pilot-fixture/pull/71](https://github.com/tosin2013/craz3-pilot-fixture/pull/71) (draft=True, author=tosins-specfoundry-pilot[bot])
- `checks/results.jsonl`: 32 of 32 pass
- `audit.jsonl`: 11 of 11 pass
- `teardown.jsonl`: 3 of 3 pass

## Events

| Seq | Stage | Task | Attempt | Event | Detail |
|---|---|---|---|---|---|
| 1 | run |  |  | run_started | tosin2013/craz3-pilot-fixture |
| 2 | github |  |  | github_sync | created=0 changed=0 |
| 3 | gate | A |  | gate_allow | seq=1 OK |
| 4 | gate | B |  | gate_refuse | seq=2 DEP_UNSATISFIED:001 |
| 5 | prepare |  |  | budget_resolved | {"worker":{"cpu":"2","memory":"2Gi","deadline_s":1200,"soft_margin_s":120},"raised":[],"source":"default"} |
| 6 | approval |  |  | approval_channel | step=mode approved=false channel=none |
| 7 | mode |  |  | mode_declined | {"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200} |
| 8 | run |  |  | run_paused(awaiting-mode) |  |
| 9 | github |  |  | decision_posted | step=mode issue=68 comment=7167965797 |
| 10 | approval |  |  | approval_recorded | step=mode channel=github approver=tosin2013 comment=7167977363 |
| 11 | run |  |  | run_resumed | paused(awaiting-mode) |
| 12 | approval |  |  | approval_channel | step=mode approved=true channel=github approver=tosin2013 comment=7167977363 |
| 13 | mode |  |  | mode_confirmed | {"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200} |
| 14 | attempt | A | 1 | attempt_created | ns=pilot-w-a-1 induction=withheld-acceptance-section |
| 15 | attempt | A | 1 | attempt_running | ghcr.io/tosin2013/chalkeus-worker@sha256:7f88c9652600739f47aa508fd1e0fb690050843a14ab8fbe3c7b606e29d52ba3 |
| 16 | attempt | A | 1 | attempt_collected |  |
| 17 | grade | A | 1 | grade_started | pilot-g-a-1 |
| 18 | grade | A | 1 | grade_fail | G5_hidden_tests |
| 19 | gate | B |  | gate_refuse | seq=3 DEP_UNSATISFIED:001 |
| 20 | approval | A | 2 | approval_channel | step=retry approved=false channel=none |
| 21 | retry | A | 2 | retry_declined |  |
| 22 | run |  |  | run_paused(awaiting-retry) |  |
| 23 | github | A | 2 | decision_posted | step=retry issue=68 comment=7168103273 |
| 24 | approval | A | 2 | approval_recorded | step=retry channel=github approver=tosin2013 comment=7168106843 |
| 25 | run |  |  | run_resumed | paused(awaiting-retry) |
| 26 | approval | A | 2 | approval_channel | step=retry approved=true channel=github approver=tosin2013 comment=7168106843 |
| 27 | retry | A | 2 | retry_confirmed |  |
| 28 | attempt | A | 2 | attempt_created | ns=pilot-w-a-2 induction=None |
| 29 | attempt | A | 2 | attempt_running | ghcr.io/tosin2013/chalkeus-worker@sha256:7f88c9652600739f47aa508fd1e0fb690050843a14ab8fbe3c7b606e29d52ba3 |
| 30 | attempt | A | 2 | attempt_collected |  |
| 31 | grade | A | 2 | grade_started | pilot-g-a-2 |
| 32 | grade | A | 2 | grade_pass |  |
| 33 | publish | A | 2 | publish_started | pilot/20261125T162546Z/a |
| 34 | publish | A | 2 | pr_opened | https://github.com/tosin2013/craz3-pilot-fixture/pull/69 |
| 35 | gate | B |  | gate_refuse | seq=4 DEP_UNSATISFIED:001 |
| 36 | run |  |  | run_paused(awaiting-merge) |  |
| 37 | run |  |  | run_resumed | paused(awaiting-merge) |
| 38 | reconcile | A | 2 | reconcile_proposed | b5aed0098ef5738add2e2de75074d4477225d243 |
| 39 | publish | A |  | status_pr_started | pilot/20261125T162546Z/status-001 |
| 40 | publish | A |  | status_pr_opened | https://github.com/tosin2013/craz3-pilot-fixture/pull/70 |
| 41 | run |  |  | run_paused(awaiting-merge) |  |
| 42 | run |  |  | run_resumed | paused(awaiting-merge) |
| 43 | reconcile | A |  | status_pr_checked | pr=70 ok=true reason=None |
| 44 | approval | A |  | approval_channel | step=reconcile approved=true channel=github approver=tosin2013 pr=70 |
| 45 | status | A |  | task_verified | 82095be49a698603daae32f85f9c1804db2dd081 |
| 46 | github |  |  | github_sync | created=0 changed=1 |
| 47 | gate | B |  | gate_allow | seq=5 OK |
| 48 | attempt | B | 1 | attempt_created | ns=pilot-w-b-1 induction=None |
| 49 | attempt | B | 1 | attempt_running | ghcr.io/tosin2013/chalkeus-worker@sha256:7f88c9652600739f47aa508fd1e0fb690050843a14ab8fbe3c7b606e29d52ba3 |
| 50 | attempt | B | 1 | attempt_collected |  |
| 51 | grade | B | 1 | grade_started | pilot-g-b-1 |
| 52 | grade | B | 1 | grade_pass |  |
| 53 | publish | B | 1 | publish_started | pilot/20261125T162546Z/b |
| 54 | publish | B | 1 | pr_opened | https://github.com/tosin2013/craz3-pilot-fixture/pull/71 |
| 55 | github |  |  | github_sync | created=0 changed=0 |
| 56 | audit |  |  | audit_done | 10/10 |
| 57 | run |  |  | run_succeeded |  |
| 58 | github |  |  | tracking_closed | issue=68 |
| 59 | teardown |  |  | torn_down | teardown.jsonl |
| 60 | audit |  |  | audit_done | 11/11 |

## Known limits

- Gate runs on the host (no published spec-status CLI image yet, PRD §8 ask 3).
- Grader uses `spec status --json`, not `verify-readiness` (report-only skill mode not shipped, PRD §8 ask 1).
- Reconcile is a deterministic pilot equivalent of the spec-status reconcile skill.
- Spend cap is enforced by the pilot watchdog and deadline unless a provider-side cap is declared (R2).
- Single-node kind only; multi-node policy enforcement not covered.
