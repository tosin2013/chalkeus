# pilot item run 20261118T230306Z

**State:** succeeded

**Elapsed:** 963 s total, 963 s excluding the human merge wait

**Mode:** `{"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200}`

## Attempts

| Task | Attempt | Induction | Grade | Cost (USD) | Tokens |
|---|---|---|---|---|---|
| 006 | 1 | - | pass | 0.0 | 440583 |

## Draft PRs

- Task 006: [https://github.com/tosin2013/stockroom/pull/33](https://github.com/tosin2013/stockroom/pull/33) (draft=True, author=tosins-specfoundry-pilot[bot])
- Task 006: [https://github.com/tosin2013/stockroom/pull/34](https://github.com/tosin2013/stockroom/pull/34) (draft=True, author=tosins-specfoundry-pilot[bot])
- `checks/results.jsonl`: 16 of 16 pass
- `audit.jsonl`: 11 of 11 pass
- `teardown.jsonl`: 3 of 3 pass

## Events

| Seq | Stage | Task | Attempt | Event | Detail |
|---|---|---|---|---|---|
| 1 | run |  |  | run_started | tosin2013/stockroom |
| 2 | gate | 006 |  | gate_allow | seq=1 OK |
| 3 | github |  |  | github_sync | created=0 changed=0 |
| 4 | approval |  |  | approval_channel | step=mode approved=false channel=none |
| 5 | mode |  |  | mode_declined | {"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200} |
| 6 | run |  |  | run_paused(awaiting-mode) |  |
| 7 | github |  |  | decision_posted | step=mode issue=32 comment=6994202089 |
| 8 | approval |  |  | approval_recorded | step=mode channel=github approver=tosin2013 comment=6994212959 |
| 9 | run |  |  | run_resumed | paused(awaiting-mode) |
| 10 | approval |  |  | approval_channel | step=mode approved=true channel=github approver=tosin2013 comment=6994212959 |
| 11 | mode |  |  | mode_confirmed | {"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200} |
| 12 | prepare | 006 |  | toolchain_ok | jdk21-maven Apache Maven 3.9.16 (2bdd9fddda4b155ebf8000e807eb73fd829a51d5) (catalog) |
| 13 | prepare | 006 |  | prefetch_pass |  |
| 14 | baseline | 006 |  | baseline_pass |  |
| 15 | attempt | 006 | 1 | attempt_created | ns=pilot-w-006-1 induction=None |
| 16 | attempt | 006 | 1 | attempt_running | ghcr.io/tosin2013/specfoundry-pilot-worker@sha256:cf0d98fad772bf09ac6a5cc095eddbee7987900d69171eb6877efb3117911ca9 |
| 17 | attempt | 006 | 1 | attempt_collected |  |
| 18 | grade | 006 | 1 | grade_started | pilot-g-006 |
| 19 | grade | 006 | 1 | grade_pass |  |
| 20 | publish | 006 | 1 | change_replayed | 5c5062d2275747f5fd938c95942a45ba6c4cc00d |
| 21 | publish | 006 | 1 | publish_started | pilot/20261118T230306Z/006 |
| 22 | publish | 006 | 1 | pr_opened | https://github.com/tosin2013/stockroom/pull/33 |
| 23 | run |  |  | run_paused(awaiting-merge) |  |
| 24 | run |  |  | run_resumed | paused(awaiting-merge) |
| 25 | reconcile | 006 | 1 | reconcile_proposed | e38abeb0bfeb2f9f9cc653e1a0750f25908c9cef |
| 26 | publish | 006 |  | status_pr_started | pilot/20261118T230306Z/status-006 |
| 27 | publish | 006 |  | status_pr_opened | https://github.com/tosin2013/stockroom/pull/34 |
| 28 | run |  |  | run_paused(awaiting-merge) |  |
| 29 | run |  |  | run_resumed | paused(awaiting-merge) |
| 30 | reconcile | 006 |  | status_pr_checked | pr=34 ok=true reason=None |
| 31 | approval | 006 |  | approval_channel | step=reconcile approved=true channel=github approver=tosin2013 pr=34 |
| 32 | status | 006 |  | task_verified | 588422cdcdf1d144154349573f6fd67426e86a0a |
| 33 | github |  |  | github_sync | created=0 changed=1 |
| 34 | github |  |  | github_sync | created=0 changed=0 |
| 35 | audit |  |  | audit_done | 10/10 |
| 36 | run |  |  | run_succeeded |  |
| 37 | github |  |  | tracking_closed | issue=32 |
| 38 | teardown |  |  | torn_down | teardown.jsonl |
| 39 | audit |  |  | audit_done | 11/11 |
| 40 | audit |  |  | audit_done | 11/11 |

## Known limits

- Gate runs on the host (no published spec-status CLI image yet, PRD §8 ask 3).
- Grader uses `spec status --json`, not `verify-readiness` (report-only skill mode not shipped, PRD §8 ask 1).
- Reconcile is a deterministic pilot equivalent of the spec-status reconcile skill.
- Spend cap is enforced by the pilot watchdog and deadline unless a provider-side cap is declared (R2).
- Single-node kind only; multi-node policy enforcement not covered.
