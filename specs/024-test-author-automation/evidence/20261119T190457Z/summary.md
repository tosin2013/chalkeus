# pilot item run 20261119T190457Z

**State:** paused(awaiting-merge)

**Elapsed:** 396 s total, 396 s excluding the human merge wait

**Mode:** `{"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200}`

## Attempts

| Task | Attempt | Induction | Grade | Cost (USD) | Tokens |
|---|---|---|---|---|---|
| 009 | 1 | - | pass | 0.0 | 62090 |

## Draft PRs

- Task 009: [https://github.com/tosin2013/stockroom/pull/43](https://github.com/tosin2013/stockroom/pull/43) (draft=True, author=tosins-specfoundry-pilot[bot])
- `checks/results.jsonl`: 16 of 17 pass

## Events

| Seq | Stage | Task | Attempt | Event | Detail |
|---|---|---|---|---|---|
| 1 | run |  |  | run_started | tosin2013/stockroom |
| 2 | gate | 009 |  | gate_allow | seq=1 OK |
| 3 | github |  |  | github_sync | created=1 changed=0 |
| 4 | approval |  |  | approval_channel | step=mode approved=false channel=none |
| 5 | mode |  |  | mode_declined | {"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200} |
| 6 | run |  |  | run_paused(awaiting-mode) |  |
| 7 | github |  |  | decision_posted | step=mode issue=42 comment=7019215711 |
| 8 | approval |  |  | approval_recorded | step=mode channel=github approver=tosin2013 comment=7019232692 |
| 9 | run |  |  | run_resumed | paused(awaiting-mode) |
| 10 | approval |  |  | approval_channel | step=mode approved=true channel=github approver=tosin2013 comment=7019232692 |
| 11 | mode |  |  | mode_confirmed | {"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200} |
| 12 | prepare | 009 |  | toolchain_ok | jdk21-maven Apache Maven 3.9.16 (2bdd9fddda4b155ebf8000e807eb73fd829a51d5) (catalog) |
| 13 | prepare | 009 |  | prefetch_pass |  |
| 14 | baseline | 009 |  | baseline_pass |  |
| 15 | attempt | 009 | 1 | attempt_created | ns=pilot-w-009-1 induction=None |
| 16 | attempt | 009 | 1 | attempt_running | ghcr.io/tosin2013/specfoundry-pilot-worker@sha256:2f74234e22ff07650932044abf6588ec1204158cc57ff3675e604c71831b3e1d |
| 17 | attempt | 009 | 1 | attempt_collected |  |
| 18 | grade | 009 | 1 | grade_started | pilot-g-009 |
| 19 | grade | 009 | 1 | grade_pass |  |
| 20 | publish | 009 | 1 | change_replayed | 82bfcb9dbf4c4c6fcf1b4900a032f450a7819bd4 |
| 21 | publish | 009 | 1 | publish_started | pilot/20261119T190457Z/009 |
| 22 | publish | 009 | 1 | pr_opened | https://github.com/tosin2013/stockroom/pull/43 |
| 23 | run |  |  | run_paused(awaiting-merge) |  |

## Known limits

- Gate runs on the host (no published spec-status CLI image yet, PRD §8 ask 3).
- Grader uses `spec status --json`, not `verify-readiness` (report-only skill mode not shipped, PRD §8 ask 1).
- Reconcile is a deterministic pilot equivalent of the spec-status reconcile skill.
- Spend cap is enforced by the pilot watchdog and deadline unless a provider-side cap is declared (R2).
- Single-node kind only; multi-node policy enforcement not covered.
