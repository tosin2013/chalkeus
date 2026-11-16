# M1 pilot run 20261116T041113Z

**State:** failed (AUDIT)

**Elapsed:** 1558 s total, 1558 s excluding the human merge wait

**Mode:** `{"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200}`

## Attempts

| Task | Attempt | Induction | Grade | Cost (USD) | Tokens |
|---|---|---|---|---|---|
| A | 1 | withheld-acceptance-section | fail (grade_fail) | 0.0 | 262984 |
| A | 2 | - | pass | 0.0 | 173789 |
| B | 1 | - | pass | 0.0 | 302468 |

## Draft PRs

- Task A: [https://github.com/tosin2013/craz3-pilot-fixture/pull/26](https://github.com/tosin2013/craz3-pilot-fixture/pull/26) (draft=True, author=tosins-specfoundry-pilot[bot])
- Task A: [https://github.com/tosin2013/craz3-pilot-fixture/pull/27](https://github.com/tosin2013/craz3-pilot-fixture/pull/27) (draft=True, author=tosins-specfoundry-pilot[bot])
- Task B: [https://github.com/tosin2013/craz3-pilot-fixture/pull/28](https://github.com/tosin2013/craz3-pilot-fixture/pull/28) (draft=True, author=tosins-specfoundry-pilot[bot])
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
| 5 | approval |  |  | approval_channel | step=mode approved=false channel=none |
| 6 | mode |  |  | mode_declined | {"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200} |
| 7 | run |  |  | run_paused(awaiting-mode) |  |
| 8 | github |  |  | decision_posted | step=mode issue=25 comment=6907079010 |
| 9 | approval |  |  | approval_recorded | step=mode channel=github approver=tosin2013 comment=6907094514 |
| 10 | approval |  |  | approval_recorded | step=mode channel=github approver=tosin2013 comment=6907094514 |
| 11 | run |  |  | run_resumed | paused(awaiting-mode) |
| 12 | approval |  |  | approval_channel | step=mode approved=true channel=github approver=tosin2013 comment=6907094514 |
| 13 | mode |  |  | mode_confirmed | {"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200} |
| 14 | attempt | A | 1 | attempt_created | ns=pilot-w-a-1 induction=withheld-acceptance-section |
| 15 | attempt | A | 1 | attempt_running | ghcr.io/tosin2013/specfoundry-pilot-worker@sha256:7493bf1bc217ec34399f8f0bc058818e162502d192760ce3cfd2e07a09611ad4 |
| 16 | attempt | A | 1 | attempt_collected |  |
| 17 | grade | A | 1 | grade_started | pilot-g-a-1 |
| 18 | grade | A | 1 | grade_fail | G5_hidden_tests |
| 19 | gate | B |  | gate_refuse | seq=3 DEP_UNSATISFIED:001 |
| 20 | approval | A | 2 | approval_channel | step=retry approved=false channel=none |
| 21 | retry | A | 2 | retry_declined |  |
| 22 | run |  |  | run_paused(awaiting-retry) |  |
| 23 | github | A | 2 | decision_posted | step=retry issue=25 comment=6907119253 |
| 24 | approval | A | 2 | approval_recorded | step=retry channel=github approver=tosin2013 comment=6907124005 |
| 25 | approval | A | 2 | approval_recorded | step=retry channel=github approver=tosin2013 comment=6907124005 |
| 26 | run |  |  | run_resumed | paused(awaiting-retry) |
| 27 | approval | A | 2 | approval_channel | step=retry approved=true channel=github approver=tosin2013 comment=6907124005 |
| 28 | retry | A | 2 | retry_confirmed |  |
| 29 | attempt | A | 2 | attempt_created | ns=pilot-w-a-2 induction=None |
| 30 | attempt | A | 2 | attempt_running | ghcr.io/tosin2013/specfoundry-pilot-worker@sha256:7493bf1bc217ec34399f8f0bc058818e162502d192760ce3cfd2e07a09611ad4 |
| 31 | attempt | A | 2 | attempt_collected |  |
| 32 | grade | A | 2 | grade_started | pilot-g-a-2 |
| 33 | grade | A | 2 | grade_pass |  |
| 34 | publish | A | 2 | publish_started | pilot/20261116T041113Z/a |
| 35 | publish | A | 2 | publish_failed | Error: create secret pilot-publish/scm-token: Forbidden |
| 36 | run |  |  | run_paused(awaiting-publish) |  |
| 37 | run |  |  | run_resumed | paused(awaiting-publish) |
| 38 | publish | A | 2 | publish_started | pilot/20261116T041113Z/a |
| 39 | publish | A | 2 | pr_opened | https://github.com/tosin2013/craz3-pilot-fixture/pull/26 |
| 40 | gate | B |  | gate_refuse | seq=4 DEP_UNSATISFIED:001 |
| 41 | run |  |  | run_paused(awaiting-merge) |  |
| 42 | run |  |  | run_resumed | paused(awaiting-merge) |
| 43 | reconcile | A | 2 | reconcile_proposed | 07d9bb32864c1ffd18d7b5674de5425b5df79c00 |
| 44 | publish | A |  | status_pr_started | pilot/20261116T041113Z/status-001 |
| 45 | publish | A |  | status_pr_opened | https://github.com/tosin2013/craz3-pilot-fixture/pull/27 |
| 46 | run |  |  | run_paused(awaiting-merge) |  |
| 47 | run |  |  | run_resumed | paused(awaiting-merge) |
| 48 | reconcile | A |  | status_pr_checked | pr=27 ok=true reason=None |
| 49 | approval | A |  | approval_channel | step=reconcile approved=true channel=github approver=tosin2013 pr=27 |
| 50 | status | A |  | task_verified | 0c25ed0f26d40d5320823f90bca65f546a556ffb |
| 51 | github |  |  | github_sync | created=0 changed=1 |
| 52 | gate | B |  | gate_allow | seq=5 OK |
| 53 | attempt | B | 1 | attempt_created | ns=pilot-w-b-1 induction=None |
| 54 | attempt | B | 1 | attempt_running | ghcr.io/tosin2013/specfoundry-pilot-worker@sha256:7493bf1bc217ec34399f8f0bc058818e162502d192760ce3cfd2e07a09611ad4 |
| 55 | attempt | B | 1 | attempt_collected |  |
| 56 | grade | B | 1 | grade_started | pilot-g-b-1 |
| 57 | grade | B | 1 | grade_pass |  |
| 58 | publish | B | 1 | publish_started | pilot/20261116T041113Z/b |
| 59 | publish | B | 1 | pr_opened | https://github.com/tosin2013/craz3-pilot-fixture/pull/28 |
| 60 | github |  |  | github_sync | created=0 changed=0 |
| 61 | audit |  |  | audit_done | 9/10 |
| 62 | run |  |  | run_failed | AUDIT |
| 63 | audit |  |  | audit_done | 11/11 |

## Known limits

- Gate runs on the host (no published spec-status CLI image yet, PRD §8 ask 3).
- Grader uses `spec status --json`, not `verify-readiness` (report-only skill mode not shipped, PRD §8 ask 1).
- Reconcile is a deterministic pilot equivalent of the spec-status reconcile skill.
- Spend cap is enforced by the pilot watchdog and deadline unless a provider-side cap is declared (R2).
- Single-node kind only; multi-node policy enforcement not covered.
