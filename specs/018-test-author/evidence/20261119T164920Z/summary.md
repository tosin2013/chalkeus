# M1 pilot run 20261119T164920Z

**State:** succeeded

**Elapsed:** 1324 s total, 1324 s excluding the human merge wait

**Mode:** `{"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200}`

## Attempts

| Task | Attempt | Induction | Grade | Cost (USD) | Tokens |
|---|---|---|---|---|---|
| A | 1 | withheld-acceptance-section | fail (grade_fail) | 0.0 | 53302 |
| A | 2 | - | pass | 0.0 | 20638 |
| B | 1 | - | pass | 0.0 | 29587 |

## Draft PRs

- Task A: [https://github.com/tosin2013/craz3-pilot-fixture/pull/47](https://github.com/tosin2013/craz3-pilot-fixture/pull/47) (draft=True, author=tosins-specfoundry-pilot[bot])
- Task A: [https://github.com/tosin2013/craz3-pilot-fixture/pull/48](https://github.com/tosin2013/craz3-pilot-fixture/pull/48) (draft=True, author=tosins-specfoundry-pilot[bot])
- Task B: [https://github.com/tosin2013/craz3-pilot-fixture/pull/49](https://github.com/tosin2013/craz3-pilot-fixture/pull/49) (draft=True, author=tosins-specfoundry-pilot[bot])
- `checks/results.jsonl`: 32 of 32 pass
- `audit.jsonl`: 11 of 11 pass
- `teardown.jsonl`: 3 of 3 pass

## Events

| Seq | Stage | Task | Attempt | Event | Detail |
|---|---|---|---|---|---|
| 1 | run |  |  | run_started | tosin2013/craz3-pilot-fixture |
| 2 | github |  |  | github_sync | created=0 changed=1 |
| 3 | gate | A |  | gate_allow | seq=1 OK |
| 4 | gate | B |  | gate_refuse | seq=2 DEP_UNSATISFIED:001 |
| 5 | approval |  |  | approval_channel | step=mode approved=false channel=none |
| 6 | mode |  |  | mode_declined | {"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200} |
| 7 | run |  |  | run_paused(awaiting-mode) |  |
| 8 | github |  |  | decision_posted | step=mode issue=46 comment=7015786665 |
| 9 | approval |  |  | approval_recorded | step=mode channel=github approver=tosin2013 comment=7015832877 |
| 10 | run |  |  | run_resumed | paused(awaiting-mode) |
| 11 | approval |  |  | approval_channel | step=mode approved=true channel=github approver=tosin2013 comment=7015832877 |
| 12 | mode |  |  | mode_confirmed | {"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200} |
| 13 | attempt | A | 1 | attempt_created | ns=pilot-w-a-1 induction=withheld-acceptance-section |
| 14 | attempt | A | 1 | attempt_running | ghcr.io/tosin2013/specfoundry-pilot-worker@sha256:cf0d98fad772bf09ac6a5cc095eddbee7987900d69171eb6877efb3117911ca9 |
| 15 | attempt | A | 1 | attempt_collected |  |
| 16 | grade | A | 1 | grade_started | pilot-g-a-1 |
| 17 | grade | A | 1 | grade_fail | G5_hidden_tests |
| 18 | gate | B |  | gate_refuse | seq=3 DEP_UNSATISFIED:001 |
| 19 | approval | A | 2 | approval_channel | step=retry approved=false channel=none |
| 20 | retry | A | 2 | retry_declined |  |
| 21 | run |  |  | run_paused(awaiting-retry) |  |
| 22 | github | A | 2 | decision_posted | step=retry issue=46 comment=7015947168 |
| 23 | approval | A | 2 | approval_recorded | step=retry channel=github approver=tosin2013 comment=7015975000 |
| 24 | run |  |  | run_resumed | paused(awaiting-retry) |
| 25 | approval | A | 2 | approval_channel | step=retry approved=true channel=github approver=tosin2013 comment=7015975000 |
| 26 | retry | A | 2 | retry_confirmed |  |
| 27 | attempt | A | 2 | attempt_created | ns=pilot-w-a-2 induction=None |
| 28 | attempt | A | 2 | attempt_running | ghcr.io/tosin2013/specfoundry-pilot-worker@sha256:cf0d98fad772bf09ac6a5cc095eddbee7987900d69171eb6877efb3117911ca9 |
| 29 | attempt | A | 2 | attempt_collected |  |
| 30 | grade | A | 2 | grade_started | pilot-g-a-2 |
| 31 | grade | A | 2 | grade_pass |  |
| 32 | publish | A | 2 | publish_started | pilot/20261119T164920Z/a |
| 33 | publish | A | 2 | pr_opened | https://github.com/tosin2013/craz3-pilot-fixture/pull/47 |
| 34 | gate | B |  | gate_refuse | seq=4 DEP_UNSATISFIED:001 |
| 35 | run |  |  | run_paused(awaiting-merge) |  |
| 36 | run |  |  | run_resumed | paused(awaiting-merge) |
| 37 | reconcile | A | 2 | reconcile_proposed | 0d9780809640c3f8ba45e60006bd2b536d0f5ff4 |
| 38 | publish | A |  | status_pr_started | pilot/20261119T164920Z/status-001 |
| 39 | publish | A |  | status_pr_opened | https://github.com/tosin2013/craz3-pilot-fixture/pull/48 |
| 40 | run |  |  | run_paused(awaiting-merge) |  |
| 41 | run |  |  | run_resumed | paused(awaiting-merge) |
| 42 | reconcile | A |  | status_pr_checked | pr=48 ok=true reason=None |
| 43 | approval | A |  | approval_channel | step=reconcile approved=true channel=github approver=tosin2013 pr=48 |
| 44 | status | A |  | task_verified | 19318675144e4343aab3b9609b44eded7f52b51f |
| 45 | github |  |  | github_sync | created=0 changed=1 |
| 46 | gate | B |  | gate_allow | seq=5 OK |
| 47 | attempt | B | 1 | attempt_created | ns=pilot-w-b-1 induction=None |
| 48 | attempt | B | 1 | attempt_running | ghcr.io/tosin2013/specfoundry-pilot-worker@sha256:cf0d98fad772bf09ac6a5cc095eddbee7987900d69171eb6877efb3117911ca9 |
| 49 | attempt | B | 1 | attempt_collected |  |
| 50 | grade | B | 1 | grade_started | pilot-g-b-1 |
| 51 | grade | B | 1 | grade_pass |  |
| 52 | publish | B | 1 | publish_started | pilot/20261119T164920Z/b |
| 53 | publish | B | 1 | pr_opened | https://github.com/tosin2013/craz3-pilot-fixture/pull/49 |
| 54 | github |  |  | github_sync | created=0 changed=0 |
| 55 | audit |  |  | audit_done | 10/10 |
| 56 | run |  |  | run_succeeded |  |
| 57 | github |  |  | tracking_closed | issue=46 |
| 58 | teardown |  |  | torn_down | teardown.jsonl |
| 59 | audit |  |  | audit_done | 11/11 |

## Known limits

- Gate runs on the host (no published spec-status CLI image yet, PRD §8 ask 3).
- Grader uses `spec status --json`, not `verify-readiness` (report-only skill mode not shipped, PRD §8 ask 1).
- Reconcile is a deterministic pilot equivalent of the spec-status reconcile skill.
- Spend cap is enforced by the pilot watchdog and deadline unless a provider-side cap is declared (R2).
- Single-node kind only; multi-node policy enforcement not covered.
