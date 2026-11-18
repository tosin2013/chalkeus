# pilot item run 20261118T224610Z

**State:** succeeded

**Elapsed:** 866 s total, 866 s excluding the human merge wait

**Mode:** `{"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200}`

## Attempts

| Task | Attempt | Induction | Grade | Cost (USD) | Tokens |
|---|---|---|---|---|---|
| 007 | 1 | - | pass | 0.0 | 510006 |

## Draft PRs

- Task 007: [https://github.com/tosin2013/stockroom/pull/30](https://github.com/tosin2013/stockroom/pull/30) (draft=True, author=tosins-specfoundry-pilot[bot])
- Task 007: [https://github.com/tosin2013/stockroom/pull/31](https://github.com/tosin2013/stockroom/pull/31) (draft=True, author=tosins-specfoundry-pilot[bot])
- `checks/results.jsonl`: 16 of 16 pass
- `audit.jsonl`: 10 of 10 pass

## Events

| Seq | Stage | Task | Attempt | Event | Detail |
|---|---|---|---|---|---|
| 1 | run |  |  | run_started | tosin2013/stockroom |
| 2 | gate | 007 |  | gate_allow | seq=1 OK |
| 3 | github |  |  | github_sync | created=1 changed=0 |
| 4 | approval |  |  | approval_channel | step=mode approved=false channel=none |
| 5 | mode |  |  | mode_declined | {"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200} |
| 6 | run |  |  | run_paused(awaiting-mode) |  |
| 7 | github |  |  | decision_posted | step=mode issue=29 comment=6993872081 |
| 8 | approval |  |  | approval_recorded | step=mode channel=github approver=tosin2013 comment=6993962693 |
| 9 | run |  |  | run_resumed | paused(awaiting-mode) |
| 10 | approval |  |  | approval_channel | step=mode approved=true channel=github approver=tosin2013 comment=6993962693 |
| 11 | mode |  |  | mode_confirmed | {"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200} |
| 12 | prepare | 007 |  | toolchain_ok | jdk21-maven Apache Maven 3.9.16 (2bdd9fddda4b155ebf8000e807eb73fd829a51d5) (catalog) |
| 13 | prepare | 007 |  | prefetch_pass |  |
| 14 | baseline | 007 |  | baseline_pass |  |
| 15 | attempt | 007 | 1 | attempt_created | ns=pilot-w-007-1 induction=None |
| 16 | attempt | 007 | 1 | attempt_running | ghcr.io/tosin2013/specfoundry-pilot-worker@sha256:cf0d98fad772bf09ac6a5cc095eddbee7987900d69171eb6877efb3117911ca9 |
| 17 | attempt | 007 | 1 | attempt_collected |  |
| 18 | grade | 007 | 1 | grade_started | pilot-g-007 |
| 19 | grade | 007 | 1 | grade_pass |  |
| 20 | publish | 007 | 1 | change_replayed | 3001681a286dcb3706c48ef576816d035f3f06c2 |
| 21 | publish | 007 | 1 | publish_started | pilot/20261118T224610Z/007 |
| 22 | publish | 007 | 1 | pr_opened | https://github.com/tosin2013/stockroom/pull/30 |
| 23 | run |  |  | run_paused(awaiting-merge) |  |
| 24 | run |  |  | run_resumed | paused(awaiting-merge) |
| 25 | reconcile | 007 | 1 | reconcile_proposed | 0d839a2d3b3172461aac1afd5a1354ff467983cb |
| 26 | publish | 007 |  | status_pr_started | pilot/20261118T224610Z/status-007 |
| 27 | publish | 007 |  | status_pr_opened | https://github.com/tosin2013/stockroom/pull/31 |
| 28 | run |  |  | run_paused(awaiting-merge) |  |
| 29 | run |  |  | run_resumed | paused(awaiting-merge) |
| 30 | reconcile | 007 |  | status_pr_checked | pr=31 ok=true reason=None |
| 31 | approval | 007 |  | approval_channel | step=reconcile approved=true channel=github approver=tosin2013 pr=31 |
| 32 | status | 007 |  | task_verified | f683e574b83466e0a84a0323a7220e0689667e71 |
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
