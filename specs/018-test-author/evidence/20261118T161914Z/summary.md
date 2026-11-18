# pilot item run 20261118T161914Z

**State:** succeeded

**Elapsed:** 788 s total, 788 s excluding the human merge wait

**Mode:** `{"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200}`

## Attempts

| Task | Attempt | Induction | Grade | Cost (USD) | Tokens |
|---|---|---|---|---|---|
| 001 | 1 | - | pass | 0.0 | 256241 |

## Draft PRs

- Task 001: [https://github.com/tosin2013/stockroom/pull/9](https://github.com/tosin2013/stockroom/pull/9) (draft=True, author=tosins-specfoundry-pilot[bot])
- Task 001: [https://github.com/tosin2013/stockroom/pull/10](https://github.com/tosin2013/stockroom/pull/10) (draft=True, author=tosins-specfoundry-pilot[bot])
- `checks/results.jsonl`: 16 of 16 pass
- `audit.jsonl`: 10 of 10 pass

## Events

| Seq | Stage | Task | Attempt | Event | Detail |
|---|---|---|---|---|---|
| 1 | run |  |  | run_started | tosin2013/stockroom |
| 2 | gate | 001 |  | gate_allow | seq=1 OK |
| 3 | github |  |  | github_sync | created=6 changed=0 |
| 4 | approval |  |  | approval_channel | step=mode approved=false channel=none |
| 5 | mode |  |  | mode_declined | {"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200} |
| 6 | run |  |  | run_paused(awaiting-mode) |  |
| 7 | github |  |  | decision_posted | step=mode issue=8 comment=6984437907 |
| 8 | approval |  |  | approval_recorded | step=mode channel=github approver=tosin2013 comment=6984562743 |
| 9 | run |  |  | run_resumed | paused(awaiting-mode) |
| 10 | approval |  |  | approval_channel | step=mode approved=true channel=github approver=tosin2013 comment=6984562743 |
| 11 | mode |  |  | mode_confirmed | {"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200} |
| 12 | prepare | 001 |  | toolchain_ok | jdk21-maven Apache Maven 3.9.16 (2bdd9fddda4b155ebf8000e807eb73fd829a51d5) (catalog) |
| 13 | prepare | 001 |  | prefetch_pass |  |
| 14 | baseline | 001 |  | baseline_pass |  |
| 15 | attempt | 001 | 1 | attempt_created | ns=pilot-w-001-1 induction=None |
| 16 | attempt | 001 | 1 | attempt_running | ghcr.io/tosin2013/specfoundry-pilot-worker@sha256:9d12a8625aca5767d9b4b3396b3980bc0dbce7229f5cf52f4108f617011abd4c |
| 17 | attempt | 001 | 1 | attempt_collected |  |
| 18 | grade | 001 | 1 | grade_started | pilot-g-001 |
| 19 | grade | 001 | 1 | grade_pass |  |
| 20 | publish | 001 | 1 | change_replayed | 92f15b7a9599cdbfce27c4300e6aba9132ef6830 |
| 21 | publish | 001 | 1 | publish_started | pilot/20261118T161914Z/001 |
| 22 | publish | 001 | 1 | pr_opened | https://github.com/tosin2013/stockroom/pull/9 |
| 23 | run |  |  | run_paused(awaiting-merge) |  |
| 24 | run |  |  | run_resumed | paused(awaiting-merge) |
| 25 | reconcile | 001 | 1 | reconcile_proposed | 22bfef4fdaa3a7b1cc74badfc55041dc57645e77 |
| 26 | publish | 001 |  | status_pr_started | pilot/20261118T161914Z/status-001 |
| 27 | publish | 001 |  | status_pr_opened | https://github.com/tosin2013/stockroom/pull/10 |
| 28 | run |  |  | run_paused(awaiting-merge) |  |
| 29 | run |  |  | run_resumed | paused(awaiting-merge) |
| 30 | reconcile | 001 |  | status_pr_checked | pr=10 ok=true reason=None |
| 31 | approval | 001 |  | approval_channel | step=reconcile approved=true channel=github approver=tosin2013 pr=10 |
| 32 | status | 001 |  | task_verified | 528a1b45db13906d5e7a9f80388b74a3f02393e1 |
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
