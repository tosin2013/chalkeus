# pilot item run 20261119T154630Z

**State:** succeeded

**Elapsed:** 2133 s total, 2133 s excluding the human merge wait

**Mode:** `{"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200}`

## Attempts

| Task | Attempt | Induction | Grade | Cost (USD) | Tokens |
|---|---|---|---|---|---|
| 008 | 1 | - | pass | 0.0 | 52475 |

## Draft PRs

- Task 008: [https://github.com/tosin2013/stockroom/pull/38](https://github.com/tosin2013/stockroom/pull/38) (draft=True, author=tosins-specfoundry-pilot[bot])
- Task 008: [https://github.com/tosin2013/stockroom/pull/39](https://github.com/tosin2013/stockroom/pull/39) (draft=True, author=tosins-specfoundry-pilot[bot])
- `checks/results.jsonl`: 17 of 17 pass
- `audit.jsonl`: 10 of 10 pass

## Events

| Seq | Stage | Task | Attempt | Event | Detail |
|---|---|---|---|---|---|
| 1 | run |  |  | run_started | tosin2013/stockroom |
| 2 | gate | 008 |  | gate_allow | seq=1 OK |
| 3 | github |  |  | github_sync | created=1 changed=0 |
| 4 | approval |  |  | approval_channel | step=mode approved=false channel=none |
| 5 | mode |  |  | mode_declined | {"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200} |
| 6 | run |  |  | run_paused(awaiting-mode) |  |
| 7 | github |  |  | decision_posted | step=mode issue=37 comment=7014118256 |
| 8 | approval |  |  | approval_recorded | step=mode channel=github approver=tosin2013 comment=7014309837 |
| 9 | run |  |  | run_resumed | paused(awaiting-mode) |
| 10 | approval |  |  | approval_channel | step=mode approved=true channel=github approver=tosin2013 comment=7014309837 |
| 11 | mode |  |  | mode_confirmed | {"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200} |
| 12 | prepare | 008 |  | toolchain_ok | jdk21-maven Apache Maven 3.9.16 (2bdd9fddda4b155ebf8000e807eb73fd829a51d5) (catalog) |
| 13 | prepare | 008 |  | prefetch_pass |  |
| 14 | baseline | 008 |  | baseline_pass |  |
| 15 | attempt | 008 | 1 | attempt_created | ns=pilot-w-008-1 induction=None |
| 16 | attempt | 008 | 1 | attempt_running | ghcr.io/tosin2013/specfoundry-pilot-worker@sha256:cf0d98fad772bf09ac6a5cc095eddbee7987900d69171eb6877efb3117911ca9 |
| 17 | attempt | 008 | 1 | attempt_collected |  |
| 18 | grade | 008 | 1 | grade_started | pilot-g-008 |
| 19 | grade | 008 | 1 | grade_pass |  |
| 20 | publish | 008 | 1 | change_replayed | 2bdcdd4e7a7bae2f5c8bdd1962bc8b55e2075ebc |
| 21 | publish | 008 | 1 | publish_started | pilot/20261119T154630Z/008 |
| 22 | publish | 008 | 1 | pr_opened | https://github.com/tosin2013/stockroom/pull/38 |
| 23 | run |  |  | run_paused(awaiting-merge) |  |
| 24 | run |  |  | run_resumed | paused(awaiting-merge) |
| 25 | reconcile | 008 | 1 | reconcile_proposed | 1653ca1c9b36fb1a50549ae1d7f002ecfef84802 |
| 26 | publish | 008 |  | status_pr_started | pilot/20261119T154630Z/status-008 |
| 27 | publish | 008 |  | status_pr_opened | https://github.com/tosin2013/stockroom/pull/39 |
| 28 | run |  |  | run_paused(awaiting-merge) |  |
| 29 | run |  |  | run_resumed | paused(awaiting-merge) |
| 30 | reconcile | 008 |  | status_pr_checked | pr=39 ok=true reason=None |
| 31 | approval | 008 |  | approval_channel | step=reconcile approved=true channel=github approver=tosin2013 pr=39 |
| 32 | status | 008 |  | task_verified | eb2754856d4bac35d572e320d09d81605d9c81ff |
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
