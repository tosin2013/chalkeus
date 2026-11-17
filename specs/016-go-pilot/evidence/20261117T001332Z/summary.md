# M1 pilot run 20261117T001332Z

**State:** succeeded

**Elapsed:** 850 s total, 850 s excluding the human merge wait

**Mode:** `{"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200}`

## Attempts

| Task | Attempt | Induction | Grade | Cost (USD) | Tokens |
|---|---|---|---|---|---|
| A | 1 | withheld-acceptance-section | fail (grade_fail) | 0.0 | 175477 |
| A | 2 | - | pass | 0.0 | 181680 |
| B | 1 | - | pass | 0.0 | 475951 |

## Draft PRs

- Task A: [https://github.com/tosin2013/craz3-pilot-fixture/pull/43](https://github.com/tosin2013/craz3-pilot-fixture/pull/43) (draft=True, author=tosins-specfoundry-pilot[bot])
- Task A: [https://github.com/tosin2013/craz3-pilot-fixture/pull/44](https://github.com/tosin2013/craz3-pilot-fixture/pull/44) (draft=True, author=tosins-specfoundry-pilot[bot])
- Task B: [https://github.com/tosin2013/craz3-pilot-fixture/pull/45](https://github.com/tosin2013/craz3-pilot-fixture/pull/45) (draft=True, author=tosins-specfoundry-pilot[bot])
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
| 8 | github |  |  | decision_posted | step=mode issue=42 comment=6933452830 |
| 9 | approval |  |  | approval_recorded | step=mode channel=github approver=tosin2013 comment=6933476116 |
| 10 | run |  |  | run_resumed | paused(awaiting-mode) |
| 11 | approval |  |  | approval_channel | step=mode approved=true channel=github approver=tosin2013 comment=6933476116 |
| 12 | mode |  |  | mode_confirmed | {"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200} |
| 13 | attempt | A | 1 | attempt_created | ns=pilot-w-a-1 induction=withheld-acceptance-section |
| 14 | attempt | A | 1 | attempt_running | ghcr.io/tosin2013/specfoundry-pilot-worker@sha256:7493bf1bc217ec34399f8f0bc058818e162502d192760ce3cfd2e07a09611ad4 |
| 15 | attempt | A | 1 | attempt_collected |  |
| 16 | grade | A | 1 | grade_started | pilot-g-a-1 |
| 17 | grade | A | 1 | grade_fail | G5_hidden_tests |
| 18 | gate | B |  | gate_refuse | seq=3 DEP_UNSATISFIED:001 |
| 19 | approval | A | 2 | approval_channel | step=retry approved=false channel=none |
| 20 | retry | A | 2 | retry_declined |  |
| 21 | run |  |  | run_paused(awaiting-retry) |  |
| 22 | github | A | 2 | decision_posted | step=retry issue=42 comment=6933509719 |
| 23 | approval | A | 2 | approval_recorded | step=retry channel=github approver=tosin2013 comment=6933554907 |
| 24 | run |  |  | run_resumed | paused(awaiting-retry) |
| 25 | approval | A | 2 | approval_channel | step=retry approved=true channel=github approver=tosin2013 comment=6933554907 |
| 26 | retry | A | 2 | retry_confirmed |  |
| 27 | attempt | A | 2 | attempt_created | ns=pilot-w-a-2 induction=None |
| 28 | attempt | A | 2 | attempt_running | ghcr.io/tosin2013/specfoundry-pilot-worker@sha256:7493bf1bc217ec34399f8f0bc058818e162502d192760ce3cfd2e07a09611ad4 |
| 29 | attempt | A | 2 | attempt_collected |  |
| 30 | grade | A | 2 | grade_started | pilot-g-a-2 |
| 31 | grade | A | 2 | grade_pass |  |
| 32 | publish | A | 2 | publish_started | pilot/20261117T001332Z/a |
| 33 | publish | A | 2 | pr_opened | https://github.com/tosin2013/craz3-pilot-fixture/pull/43 |
| 34 | gate | B |  | gate_refuse | seq=4 DEP_UNSATISFIED:001 |
| 35 | run |  |  | run_paused(awaiting-merge) |  |
| 36 | run |  |  | run_resumed | paused(awaiting-merge) |
| 37 | reconcile | A | 2 | reconcile_proposed | 8b62661c1386b60b3652283c706e252960881c15 |
| 38 | publish | A |  | status_pr_started | pilot/20261117T001332Z/status-001 |
| 39 | publish | A |  | status_pr_opened | https://github.com/tosin2013/craz3-pilot-fixture/pull/44 |
| 40 | run |  |  | run_paused(awaiting-merge) |  |
| 41 | run |  |  | run_resumed | paused(awaiting-merge) |
| 42 | reconcile | A |  | status_pr_checked | pr=44 ok=true reason=None |
| 43 | approval | A |  | approval_channel | step=reconcile approved=true channel=github approver=tosin2013 pr=44 |
| 44 | status | A |  | task_verified | 8422f86e54f8e11b4fcf2d23d83e66e40620ee62 |
| 45 | github |  |  | github_sync | created=0 changed=1 |
| 46 | gate | B |  | gate_allow | seq=5 OK |
| 47 | attempt | B | 1 | attempt_created | ns=pilot-w-b-1 induction=None |
| 48 | attempt | B | 1 | attempt_running | ghcr.io/tosin2013/specfoundry-pilot-worker@sha256:7493bf1bc217ec34399f8f0bc058818e162502d192760ce3cfd2e07a09611ad4 |
| 49 | attempt | B | 1 | attempt_collected |  |
| 50 | grade | B | 1 | grade_started | pilot-g-b-1 |
| 51 | grade | B | 1 | grade_pass |  |
| 52 | publish | B | 1 | publish_started | pilot/20261117T001332Z/b |
| 53 | publish | B | 1 | pr_opened | https://github.com/tosin2013/craz3-pilot-fixture/pull/45 |
| 54 | github |  |  | github_sync | created=0 changed=0 |
| 55 | audit |  |  | audit_done | 10/10 |
| 56 | run |  |  | run_succeeded |  |
| 57 | teardown |  |  | torn_down | teardown.jsonl |
| 58 | audit |  |  | audit_done | 11/11 |

## Known limits

- Gate runs on the host (no published spec-status CLI image yet, PRD §8 ask 3).
- Grader uses `spec status --json`, not `verify-readiness` (report-only skill mode not shipped, PRD §8 ask 1).
- Reconcile is a deterministic pilot equivalent of the spec-status reconcile skill.
- Spend cap is enforced by the pilot watchdog and deadline unless a provider-side cap is declared (R2).
- Single-node kind only; multi-node policy enforcement not covered.
