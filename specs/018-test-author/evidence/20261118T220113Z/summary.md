# pilot item run 20261118T220113Z

**State:** succeeded

**Elapsed:** 760 s total, 760 s excluding the human merge wait

**Mode:** `{"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200}`

## Attempts

| Task | Attempt | Induction | Grade | Cost (USD) | Tokens |
|---|---|---|---|---|---|
| 005 | 1 | - | pass | 0.0 | 472606 |

## Draft PRs

- Task 005: [https://github.com/tosin2013/stockroom/pull/25](https://github.com/tosin2013/stockroom/pull/25) (draft=True, author=tosins-specfoundry-pilot[bot])
- Task 005: [https://github.com/tosin2013/stockroom/pull/26](https://github.com/tosin2013/stockroom/pull/26) (draft=True, author=tosins-specfoundry-pilot[bot])
- `checks/results.jsonl`: 16 of 16 pass
- `audit.jsonl`: 10 of 10 pass

## Events

| Seq | Stage | Task | Attempt | Event | Detail |
|---|---|---|---|---|---|
| 1 | run |  |  | run_started | tosin2013/stockroom |
| 2 | gate | 005 |  | gate_allow | seq=1 OK |
| 3 | github |  |  | github_sync | created=0 changed=2 |
| 4 | approval |  |  | approval_channel | step=mode approved=false channel=none |
| 5 | mode |  |  | mode_declined | {"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200} |
| 6 | run |  |  | run_paused(awaiting-mode) |  |
| 7 | github |  |  | decision_posted | step=mode issue=24 comment=6992892258 |
| 8 | approval |  |  | approval_recorded | step=mode channel=github approver=tosin2013 comment=6992973794 |
| 9 | run |  |  | run_resumed | paused(awaiting-mode) |
| 10 | approval |  |  | approval_channel | step=mode approved=true channel=github approver=tosin2013 comment=6992973794 |
| 11 | mode |  |  | mode_confirmed | {"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200} |
| 12 | prepare | 005 |  | toolchain_ok | jdk21-maven Apache Maven 3.9.16 (2bdd9fddda4b155ebf8000e807eb73fd829a51d5) (catalog) |
| 13 | prepare | 005 |  | prefetch_pass |  |
| 14 | baseline | 005 |  | baseline_pass |  |
| 15 | attempt | 005 | 1 | attempt_created | ns=pilot-w-005-1 induction=None |
| 16 | attempt | 005 | 1 | attempt_running | ghcr.io/tosin2013/specfoundry-pilot-worker@sha256:cf0d98fad772bf09ac6a5cc095eddbee7987900d69171eb6877efb3117911ca9 |
| 17 | attempt | 005 | 1 | attempt_collected |  |
| 18 | grade | 005 | 1 | grade_started | pilot-g-005 |
| 19 | grade | 005 | 1 | grade_pass |  |
| 20 | publish | 005 | 1 | change_replayed | 48ece5b307988115c3fa8a986556341a974f79c1 |
| 21 | publish | 005 | 1 | publish_started | pilot/20261118T220113Z/005 |
| 22 | publish | 005 | 1 | pr_opened | https://github.com/tosin2013/stockroom/pull/25 |
| 23 | run |  |  | run_paused(awaiting-merge) |  |
| 24 | run |  |  | run_resumed | paused(awaiting-merge) |
| 25 | reconcile | 005 | 1 | reconcile_proposed | 917998fad8ce84df408d064a2bce328989a881ef |
| 26 | publish | 005 |  | status_pr_started | pilot/20261118T220113Z/status-005 |
| 27 | publish | 005 |  | status_pr_opened | https://github.com/tosin2013/stockroom/pull/26 |
| 28 | run |  |  | run_paused(awaiting-merge) |  |
| 29 | run |  |  | run_resumed | paused(awaiting-merge) |
| 30 | reconcile | 005 |  | status_pr_checked | pr=26 ok=true reason=None |
| 31 | approval | 005 |  | approval_channel | step=reconcile approved=true channel=github approver=tosin2013 pr=26 |
| 32 | status | 005 |  | task_verified | 1ffbee82c74d5f61a476076463ff7a80ae44b376 |
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
