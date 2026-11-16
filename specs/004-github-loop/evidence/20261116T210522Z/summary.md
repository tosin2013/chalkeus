# M1 pilot run 20261116T210522Z

**State:** failed (STATUS_PR_MERGED_BY_PUBLISHER)

**Elapsed:** 6064 s total, 6064 s excluding the human merge wait

**Mode:** `{"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200}`

## Attempts

| Task | Attempt | Induction | Grade | Cost (USD) | Tokens |
|---|---|---|---|---|---|
| A | 1 | withheld-acceptance-section | fail (grade_fail) | 0.0 | 188681 |
| A | 2 | - | pass | 0.0 | 163672 |

## Draft PRs

- Task A: [https://github.com/tosin2013/craz3-pilot-fixture/pull/34](https://github.com/tosin2013/craz3-pilot-fixture/pull/34) (draft=True, author=tosin2013)
- Task A: [https://github.com/tosin2013/craz3-pilot-fixture/pull/35](https://github.com/tosin2013/craz3-pilot-fixture/pull/35) (draft=True, author=tosin2013)
- `checks/results.jsonl`: 22 of 22 pass
- `audit.jsonl`: 8 of 11 pass

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
| 8 | github |  |  | decision_posted | step=mode issue=33 comment=6929314004 |
| 9 | approval |  |  | approval_rejected | step=mode comment=6929339557 author=tosin2013 reason=publisher_identity |
| 10 | approval |  |  | approval_rejected | step=mode comment=6929478417 author=tosin2013 reason=publisher_identity |
| 11 | approval |  |  | approval_recorded | step=mode channel=harness |
| 12 | run |  |  | run_resumed | paused(awaiting-mode) |
| 13 | approval |  |  | approval_channel | step=mode approved=true channel=harness |
| 14 | mode |  |  | mode_confirmed | {"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200} |
| 15 | attempt | A | 1 | attempt_created | ns=pilot-w-a-1 induction=withheld-acceptance-section |
| 16 | attempt | A | 1 | attempt_running | ghcr.io/tosin2013/specfoundry-pilot-worker@sha256:7493bf1bc217ec34399f8f0bc058818e162502d192760ce3cfd2e07a09611ad4 |
| 17 | attempt | A | 1 | attempt_collected |  |
| 18 | grade | A | 1 | grade_started | pilot-g-a-1 |
| 19 | grade | A | 1 | grade_fail | G5_hidden_tests |
| 20 | gate | B |  | gate_refuse | seq=3 DEP_UNSATISFIED:001 |
| 21 | approval | A | 2 | approval_channel | step=retry approved=false channel=none |
| 22 | retry | A | 2 | retry_declined |  |
| 23 | run |  |  | run_paused(awaiting-retry) |  |
| 24 | github | A | 2 | decision_posted | step=retry issue=33 comment=6931215580 |
| 25 | approval | A | 2 | approval_recorded | step=retry channel=harness |
| 26 | run |  |  | run_resumed | paused(awaiting-retry) |
| 27 | approval | A | 2 | approval_channel | step=retry approved=true channel=harness |
| 28 | retry | A | 2 | retry_confirmed |  |
| 29 | attempt | A | 2 | attempt_created | ns=pilot-w-a-2 induction=None |
| 30 | attempt | A | 2 | attempt_running | ghcr.io/tosin2013/specfoundry-pilot-worker@sha256:7493bf1bc217ec34399f8f0bc058818e162502d192760ce3cfd2e07a09611ad4 |
| 31 | attempt | A | 2 | attempt_collected |  |
| 32 | grade | A | 2 | grade_started | pilot-g-a-2 |
| 33 | grade | A | 2 | grade_pass |  |
| 34 | publish | A | 2 | publish_started | pilot/20261116T210522Z/a |
| 35 | publish | A | 2 | pr_opened | https://github.com/tosin2013/craz3-pilot-fixture/pull/34 |
| 36 | gate | B |  | gate_refuse | seq=4 DEP_UNSATISFIED:001 |
| 37 | run |  |  | run_paused(awaiting-merge) |  |
| 38 | run |  |  | run_resumed | paused(awaiting-merge) |
| 39 | reconcile | A | 2 | reconcile_proposed | f7d4187f64edc286cc2a7508e5dedbf87bd1d4a7 |
| 40 | publish | A |  | status_pr_started | pilot/20261116T210522Z/status-001 |
| 41 | publish | A |  | status_pr_opened | https://github.com/tosin2013/craz3-pilot-fixture/pull/35 |
| 42 | run |  |  | run_paused(awaiting-merge) |  |
| 43 | run |  |  | run_resumed | paused(awaiting-merge) |
| 44 | reconcile | A |  | status_pr_checked | pr=35 ok=false reason=STATUS_PR_MERGED_BY_PUBLISHER |
| 45 | approval | A |  | approval_channel | step=reconcile approved=false channel=github approver=tosin2013 pr=35 reason=STATUS_PR_MERGED_BY_PUBLISHER |
| 46 | run |  |  | run_failed | STATUS_PR_MERGED_BY_PUBLISHER |
| 47 | audit |  |  | audit_done | 8/11 |

## Known limits

- Gate runs on the host (no published spec-status CLI image yet, PRD §8 ask 3).
- Grader uses `spec status --json`, not `verify-readiness` (report-only skill mode not shipped, PRD §8 ask 1).
- Reconcile is a deterministic pilot equivalent of the spec-status reconcile skill.
- Spend cap is enforced by the pilot watchdog and deadline unless a provider-side cap is declared (R2).
- Single-node kind only; multi-node policy enforcement not covered.
