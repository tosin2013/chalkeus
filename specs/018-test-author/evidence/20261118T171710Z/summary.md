# pilot item run 20261118T171710Z

**State:** succeeded

**Elapsed:** 4689 s total, 4689 s excluding the human merge wait

**Mode:** `{"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200}`

## Attempts

| Task | Attempt | Induction | Grade | Cost (USD) | Tokens |
|---|---|---|---|---|---|
| 002 | 1 | - | pass | 0.0 | 140734 |

## Draft PRs

- Task 002: [https://github.com/tosin2013/stockroom/pull/13](https://github.com/tosin2013/stockroom/pull/13) (draft=True, author=tosins-specfoundry-pilot[bot])
- Task 002: [https://github.com/tosin2013/stockroom/pull/14](https://github.com/tosin2013/stockroom/pull/14) (draft=True, author=tosins-specfoundry-pilot[bot])
- `checks/results.jsonl`: 16 of 16 pass
- `audit.jsonl`: 10 of 10 pass

## Events

| Seq | Stage | Task | Attempt | Event | Detail |
|---|---|---|---|---|---|
| 1 | run |  |  | run_started | tosin2013/stockroom |
| 2 | gate | 002 |  | gate_allow | seq=1 OK |
| 3 | github |  |  | github_sync | created=0 changed=1 |
| 4 | approval |  |  | approval_channel | step=mode approved=false channel=none |
| 5 | mode |  |  | mode_declined | {"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200} |
| 6 | run |  |  | run_paused(awaiting-mode) |  |
| 7 | github |  |  | decision_posted | step=mode issue=12 comment=6985981379 |
| 8 | approval |  |  | approval_recorded | step=mode channel=github approver=tosin2013 comment=6986052609 |
| 9 | run |  |  | run_resumed | paused(awaiting-mode) |
| 10 | approval |  |  | approval_channel | step=mode approved=true channel=github approver=tosin2013 comment=6986052609 |
| 11 | mode |  |  | mode_confirmed | {"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200} |
| 12 | prepare | 002 |  | toolchain_ok | jdk21-maven Apache Maven 3.9.16 (2bdd9fddda4b155ebf8000e807eb73fd829a51d5) (catalog) |
| 13 | prepare | 002 |  | prefetch_pass |  |
| 14 | baseline | 002 |  | baseline_pass |  |
| 15 | attempt | 002 | 1 | attempt_created | ns=pilot-w-002-1 induction=None |
| 16 | attempt | 002 | 1 | attempt_running | ghcr.io/tosin2013/specfoundry-pilot-worker@sha256:9d12a8625aca5767d9b4b3396b3980bc0dbce7229f5cf52f4108f617011abd4c |
| 17 | attempt | 002 | 1 | attempt_collected |  |
| 18 | grade | 002 | 1 | grade_started | pilot-g-002 |
| 19 | grade | 002 | 1 | grade_pass |  |
| 20 | publish | 002 | 1 | change_replayed | 7f8e61062533849d7fe39ef07a8e841114f95318 |
| 21 | publish | 002 | 1 | publish_started | pilot/20261118T171710Z/002 |
| 22 | publish | 002 | 1 | pr_opened | https://github.com/tosin2013/stockroom/pull/13 |
| 23 | run |  |  | run_paused(awaiting-merge) |  |
| 24 | run |  |  | run_resumed | paused(awaiting-merge) |
| 25 | reconcile | 002 | 1 | reconcile_proposed | 2be4d4ca82f8604ce78edd6aee960207fd6d8ff8 |
| 26 | publish | 002 |  | status_pr_started | pilot/20261118T171710Z/status-002 |
| 27 | publish | 002 |  | status_pr_opened | https://github.com/tosin2013/stockroom/pull/14 |
| 28 | run |  |  | run_paused(awaiting-merge) |  |
| 29 | run |  |  | run_resumed | paused(awaiting-merge) |
| 30 | reconcile | 002 |  | status_pr_checked | pr=14 ok=true reason=None |
| 31 | approval | 002 |  | approval_channel | step=reconcile approved=true channel=github approver=tosin2013 pr=14 |
| 32 | status | 002 |  | task_verified | 12464bf1a8cf5e6778b3d1c35dd633fac84d7d02 |
| 33 | github |  |  | github_sync | created=0 changed=1 |
| 34 | github |  |  | github_sync | created=0 changed=0 |
| 35 | audit |  |  | audit_done | 10/10 |
| 36 | run |  |  | run_succeeded |  |

## Known limits

- Gate runs on the host (no published spec-status CLI image yet, PRD §8 ask 3).
- Grader uses `spec status --json`, not `verify-readiness` (report-only skill mode not shipped, PRD §8 ask 1).
- Reconcile is a deterministic pilot equivalent of the spec-status reconcile skill.
- Spend cap is enforced by the pilot watchdog and deadline unless a provider-side cap is declared (R2).
- Single-node kind only; multi-node policy enforcement not covered.
