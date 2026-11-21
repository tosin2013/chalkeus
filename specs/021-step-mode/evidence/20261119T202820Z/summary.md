# pilot item run 20261119T202820Z

**State:** succeeded

**Elapsed:** 5058 s total, 5058 s excluding the human merge wait

**Mode:** `{"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200}`

## Attempts

| Task | Attempt | Induction | Grade | Cost (USD) | Tokens |
|---|---|---|---|---|---|
| 009 | 1 | - | pass | 0.0 | 56397 |

## Draft PRs

- Task 009: [https://github.com/tosin2013/stockroom/pull/44](https://github.com/tosin2013/stockroom/pull/44) (draft=True, author=tosins-specfoundry-pilot[bot])
- Task 009: [https://github.com/tosin2013/stockroom/pull/45](https://github.com/tosin2013/stockroom/pull/45) (draft=True, author=tosins-specfoundry-pilot[bot])
- `checks/results.jsonl`: 17 of 17 pass
- `audit.jsonl`: 10 of 10 pass

## Events

| Seq | Stage | Task | Attempt | Event | Detail |
|---|---|---|---|---|---|
| 1 | run |  |  | run_started | tosin2013/stockroom |
| 2 | run |  |  | run_started_by | step |
| 3 | gate | 009 |  | gate_allow | seq=1 OK |
| 4 | github |  |  | github_sync | created=0 changed=0 |
| 5 | approval |  |  | approval_channel | step=mode approved=true channel=tty |
| 6 | mode |  |  | mode_confirmed | {"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200} |
| 7 | prepare | 009 |  | toolchain_ok | jdk21-maven Apache Maven 3.9.16 (2bdd9fddda4b155ebf8000e807eb73fd829a51d5) (catalog) |
| 8 | prepare | 009 |  | prefetch_pass |  |
| 9 | baseline | 009 |  | baseline_pass |  |
| 10 | attempt | 009 | 1 | attempt_created | ns=pilot-w-009-1 induction=None |
| 11 | attempt | 009 | 1 | attempt_running | ghcr.io/tosin2013/specfoundry-pilot-worker@sha256:2f74234e22ff07650932044abf6588ec1204158cc57ff3675e604c71831b3e1d |
| 12 | attempt | 009 | 1 | attempt_collected |  |
| 13 | grade | 009 | 1 | grade_started | pilot-g-009 |
| 14 | grade | 009 | 1 | grade_pass |  |
| 15 | publish | 009 | 1 | change_replayed | 5cc72479624a9170cad5f47210094c219d099df0 |
| 16 | publish | 009 | 1 | publish_started | pilot/20261119T202820Z/009 |
| 17 | publish | 009 | 1 | pr_opened | https://github.com/tosin2013/stockroom/pull/44 |
| 18 | run |  |  | run_paused(awaiting-merge) |  |
| 19 | run |  |  | run_resumed | paused(awaiting-merge) |
| 20 | reconcile | 009 | 1 | reconcile_proposed | eb48058badd58971e91e552c30d9cf3d5aa47f6f |
| 21 | publish | 009 |  | status_pr_started | pilot/20261119T202820Z/status-009 |
| 22 | publish | 009 |  | status_pr_opened | https://github.com/tosin2013/stockroom/pull/45 |
| 23 | run |  |  | run_paused(awaiting-merge) |  |
| 24 | run |  |  | run_resumed | paused(awaiting-merge) |
| 25 | reconcile | 009 |  | status_pr_checked | pr=45 ok=true reason=None |
| 26 | approval | 009 |  | approval_channel | step=reconcile approved=true channel=github approver=tosin2013 pr=45 |
| 27 | status | 009 |  | task_verified | 00969692462f5bf8efe1634990ea2568f9eb4d25 |
| 28 | github |  |  | github_sync | created=0 changed=1 |
| 29 | github |  |  | github_sync | created=0 changed=0 |
| 30 | audit |  |  | audit_done | 10/10 |
| 31 | run |  |  | run_succeeded |  |

## Known limits

- Gate runs on the host (no published spec-status CLI image yet, PRD §8 ask 3).
- Grader uses `spec status --json`, not `verify-readiness` (report-only skill mode not shipped, PRD §8 ask 1).
- Reconcile is a deterministic pilot equivalent of the spec-status reconcile skill.
- Spend cap is enforced by the pilot watchdog and deadline unless a provider-side cap is declared (R2).
- Single-node kind only; multi-node policy enforcement not covered.
