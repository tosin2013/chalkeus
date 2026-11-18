# pilot item run 20261118T212846Z

**State:** succeeded

**Elapsed:** 870 s total, 870 s excluding the human merge wait

**Mode:** `{"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200}`

## Attempts

| Task | Attempt | Induction | Grade | Cost (USD) | Tokens |
|---|---|---|---|---|---|
| 004 | 1 | - | pass | 0.0 | 722467 |

## Draft PRs

- Task 004: [https://github.com/tosin2013/stockroom/pull/21](https://github.com/tosin2013/stockroom/pull/21) (draft=True, author=tosins-specfoundry-pilot[bot])
- Task 004: [https://github.com/tosin2013/stockroom/pull/22](https://github.com/tosin2013/stockroom/pull/22) (draft=True, author=tosins-specfoundry-pilot[bot])
- `checks/results.jsonl`: 16 of 16 pass
- `audit.jsonl`: 10 of 10 pass

## Events

| Seq | Stage | Task | Attempt | Event | Detail |
|---|---|---|---|---|---|
| 1 | run |  |  | run_started | tosin2013/stockroom |
| 2 | gate | 004 |  | gate_allow | seq=1 OK |
| 3 | github |  |  | github_sync | created=0 changed=1 |
| 4 | approval |  |  | approval_channel | step=mode approved=false channel=none |
| 5 | mode |  |  | mode_declined | {"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200} |
| 6 | run |  |  | run_paused(awaiting-mode) |  |
| 7 | github |  |  | decision_posted | step=mode issue=20 comment=6992166470 |
| 8 | approval |  |  | approval_recorded | step=mode channel=github approver=tosin2013 comment=6992249220 |
| 9 | run |  |  | run_resumed | paused(awaiting-mode) |
| 10 | approval |  |  | approval_channel | step=mode approved=true channel=github approver=tosin2013 comment=6992249220 |
| 11 | mode |  |  | mode_confirmed | {"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200} |
| 12 | prepare | 004 |  | toolchain_ok | jdk21-maven Apache Maven 3.9.16 (2bdd9fddda4b155ebf8000e807eb73fd829a51d5) (catalog) |
| 13 | prepare | 004 |  | prefetch_pass |  |
| 14 | baseline | 004 |  | baseline_pass |  |
| 15 | attempt | 004 | 1 | attempt_created | ns=pilot-w-004-1 induction=None |
| 16 | attempt | 004 | 1 | attempt_running | ghcr.io/tosin2013/specfoundry-pilot-worker@sha256:cf0d98fad772bf09ac6a5cc095eddbee7987900d69171eb6877efb3117911ca9 |
| 17 | attempt | 004 | 1 | attempt_collected |  |
| 18 | grade | 004 | 1 | grade_started | pilot-g-004 |
| 19 | grade | 004 | 1 | grade_pass |  |
| 20 | publish | 004 | 1 | change_replayed | 279c67d464ee5704602be566f2034a1f2f99b63e |
| 21 | publish | 004 | 1 | publish_started | pilot/20261118T212846Z/004 |
| 22 | publish | 004 | 1 | pr_opened | https://github.com/tosin2013/stockroom/pull/21 |
| 23 | run |  |  | run_paused(awaiting-merge) |  |
| 24 | run |  |  | run_resumed | paused(awaiting-merge) |
| 25 | reconcile | 004 | 1 | reconcile_proposed | 44cf9ed6dd214c09989d7ef5b1b8993cf9fed2b0 |
| 26 | publish | 004 |  | status_pr_started | pilot/20261118T212846Z/status-004 |
| 27 | publish | 004 |  | status_pr_opened | https://github.com/tosin2013/stockroom/pull/22 |
| 28 | run |  |  | run_paused(awaiting-merge) |  |
| 29 | run |  |  | run_resumed | paused(awaiting-merge) |
| 30 | reconcile | 004 |  | status_pr_checked | pr=22 ok=true reason=None |
| 31 | approval | 004 |  | approval_channel | step=reconcile approved=true channel=github approver=tosin2013 pr=22 |
| 32 | status | 004 |  | task_verified | 1fb140092816ce6bb1490c59579846543f05b66b |
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
