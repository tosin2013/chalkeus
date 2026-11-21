# pilot item run 20261119T223729Z

**State:** succeeded

**Elapsed:** 144424 s total, 144424 s excluding the human merge wait

**Mode:** `{"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200}`

## Attempts

| Task | Attempt | Induction | Grade | Cost (USD) | Tokens |
|---|---|---|---|---|---|
| 010 | 1 | - | pass | 0.0 | 79180 |

## Draft PRs

- Task 010: [https://github.com/tosin2013/stockroom/pull/49](https://github.com/tosin2013/stockroom/pull/49) (draft=True, author=tosins-specfoundry-pilot[bot])
- Task 010: [https://github.com/tosin2013/stockroom/pull/50](https://github.com/tosin2013/stockroom/pull/50) (draft=True, author=tosins-specfoundry-pilot[bot])
- `checks/results.jsonl`: 17 of 17 pass
- `audit.jsonl`: 11 of 11 pass
- `teardown.jsonl`: 3 of 3 pass

## Events

| Seq | Stage | Task | Attempt | Event | Detail |
|---|---|---|---|---|---|
| 1 | run |  |  | run_started | tosin2013/stockroom |
| 2 | run |  |  | run_started_by | step |
| 3 | gate | 010 |  | gate_allow | seq=1 OK |
| 4 | github |  |  | github_sync | created=0 changed=0 |
| 5 | approval |  |  | approval_channel | step=mode approved=false channel=none |
| 6 | mode |  |  | mode_declined | {"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200} |
| 7 | run |  |  | run_paused(awaiting-mode) |  |
| 8 | github |  |  | decision_posted | step=mode issue=48 comment=7024170217 |
| 9 | approval |  |  | approval_recorded | step=mode channel=github approver=tosin2013 comment=7024210525 |
| 10 | run |  |  | run_resumed | paused(awaiting-mode) |
| 11 | approval |  |  | approval_channel | step=mode approved=true channel=github approver=tosin2013 comment=7024210525 |
| 12 | mode |  |  | mode_confirmed | {"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200} |
| 13 | prepare | 010 |  | toolchain_ok | jdk21-maven Apache Maven 3.9.16 (2bdd9fddda4b155ebf8000e807eb73fd829a51d5) (catalog) |
| 14 | prepare | 010 |  | prefetch_pass |  |
| 15 | baseline | 010 |  | baseline_pass |  |
| 16 | attempt | 010 | 1 | attempt_created | ns=pilot-w-010-1 induction=None |
| 17 | attempt | 010 | 1 | attempt_running | ghcr.io/tosin2013/specfoundry-pilot-worker@sha256:2f74234e22ff07650932044abf6588ec1204158cc57ff3675e604c71831b3e1d |
| 18 | attempt | 010 | 1 | attempt_collected |  |
| 19 | grade | 010 | 1 | grade_started | pilot-g-010 |
| 20 | grade | 010 | 1 | grade_pass |  |
| 21 | publish | 010 | 1 | change_replayed | 8260070a77acff888cf410f58efe3b027c64b638 |
| 22 | publish | 010 | 1 | publish_started | pilot/20261119T223729Z/010 |
| 23 | publish | 010 | 1 | pr_opened | https://github.com/tosin2013/stockroom/pull/49 |
| 24 | run |  |  | run_paused(awaiting-merge) |  |
| 25 | run |  |  | run_resumed | paused(awaiting-merge) |
| 26 | reconcile | 010 | 1 | reconcile_proposed | 3ca60bf4a08d894297611d96f0e006ddef1536ee |
| 27 | publish | 010 |  | status_pr_started | pilot/20261119T223729Z/status-010 |
| 28 | publish | 010 |  | status_pr_opened | https://github.com/tosin2013/stockroom/pull/50 |
| 29 | run |  |  | run_paused(awaiting-merge) |  |
| 30 | run |  |  | run_resumed | paused(awaiting-merge) |
| 31 | reconcile | 010 |  | status_pr_checked | pr=50 ok=true reason=None |
| 32 | approval | 010 |  | approval_channel | step=reconcile approved=true channel=github approver=tosin2013 pr=50 |
| 33 | status | 010 |  | task_verified | 56a57136cb5cccb3a137ff296d1f116ad4ee8676 |
| 34 | github |  |  | github_sync | created=0 changed=1 |
| 35 | github |  |  | github_sync | created=0 changed=0 |
| 36 | audit |  |  | audit_done | 10/10 |
| 37 | run |  |  | run_succeeded |  |
| 38 | github |  |  | tracking_closed | issue=48 |
| 39 | teardown |  |  | torn_down | teardown.jsonl |
| 40 | audit |  |  | audit_done | 11/11 |

## Known limits

- Gate runs on the host (no published spec-status CLI image yet, PRD §8 ask 3).
- Grader uses `spec status --json`, not `verify-readiness` (report-only skill mode not shipped, PRD §8 ask 1).
- Reconcile is a deterministic pilot equivalent of the spec-status reconcile skill.
- Spend cap is enforced by the pilot watchdog and deadline unless a provider-side cap is declared (R2).
- Single-node kind only; multi-node policy enforcement not covered.
