# M1 pilot run 20261115T023426Z

**State:** succeeded

**Elapsed:** 938 s total, 847 s excluding the human merge wait

**Mode:** `{"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200}`

## Attempts

| Task | Attempt | Induction | Grade | Cost (USD) | Tokens |
|---|---|---|---|---|---|
| A | 1 | withheld-acceptance-section | fail (grade_fail) | 0.0 | 182224 |
| A | 2 | - | fail (grade_fail) | 0.0 | 209259 |
| A | 3 | - | pass | 0.0 | 217701 |
| B | 1 | - | pass | 0.0 | 332582 |

## Draft PRs

- Task A: [https://github.com/tosin2013/craz3-pilot-fixture/pull/16](https://github.com/tosin2013/craz3-pilot-fixture/pull/16) (draft=True, author=tosin2013)
- Task B: [https://github.com/tosin2013/craz3-pilot-fixture/pull/17](https://github.com/tosin2013/craz3-pilot-fixture/pull/17) (draft=True, author=tosin2013)
- `checks/results.jsonl`: 40 of 40 pass
- `audit.jsonl`: 11 of 11 pass
- `teardown.jsonl`: 3 of 3 pass

## Events

| Seq | Stage | Task | Attempt | Event | Detail |
|---|---|---|---|---|---|
| 1 | run |  |  | run_started | tosin2013/craz3-pilot-fixture |
| 2 | gate | A |  | gate_allow | seq=1 OK |
| 3 | gate | B |  | gate_refuse | seq=2 DEP_UNSATISFIED:001 |
| 4 | approval |  |  | approval_channel | step=mode approved=true channel=tty |
| 5 | mode |  |  | mode_confirmed | {"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200} |
| 6 | attempt | A | 1 | attempt_created | ns=pilot-w-a-1 induction=withheld-acceptance-section |
| 7 | attempt | A | 1 | attempt_running | ghcr.io/tosin2013/specfoundry-pilot-worker@sha256:8656c77f8dcf5a29dab2012ccf1cc4f141456775081221de7c70ebcf5748efbc |
| 8 | attempt | A | 1 | attempt_collected |  |
| 9 | grade | A | 1 | grade_started | pilot-g-a-1 |
| 10 | grade | A | 1 | grade_fail | G5_hidden_tests |
| 11 | gate | B |  | gate_refuse | seq=3 DEP_UNSATISFIED:001 |
| 12 | approval | A | 2 | approval_channel | step=retry approved=false channel=tty |
| 13 | retry | A | 2 | retry_declined |  |
| 14 | run |  |  | run_paused(awaiting-retry) |  |
| 15 | approval | A | 2 | approval_recorded | step=retry channel=harness |
| 16 | run |  |  | run_resumed | paused(awaiting-retry) |
| 17 | approval | A | 2 | approval_channel | step=retry approved=true channel=tty |
| 18 | retry | A | 2 | retry_confirmed |  |
| 19 | attempt | A | 2 | attempt_created | ns=pilot-w-a-2 induction=None |
| 20 | attempt | A | 2 | attempt_running | ghcr.io/tosin2013/specfoundry-pilot-worker@sha256:8656c77f8dcf5a29dab2012ccf1cc4f141456775081221de7c70ebcf5748efbc |
| 21 | attempt | A | 2 | attempt_collected |  |
| 22 | grade | A | 2 | grade_started | pilot-g-a-2 |
| 23 | grade | A | 2 | grade_fail | G4_visible_tests |
| 24 | gate | B |  | gate_refuse | seq=4 DEP_UNSATISFIED:001 |
| 25 | approval | A | 3 | approval_channel | step=retry approved=false channel=tty |
| 26 | retry | A | 3 | retry_declined |  |
| 27 | run |  |  | run_paused(awaiting-retry) |  |
| 28 | approval | A | 3 | approval_recorded | step=retry channel=harness |
| 29 | run |  |  | run_resumed | paused(awaiting-retry) |
| 30 | approval | A | 3 | approval_channel | step=retry approved=false channel=tty |
| 31 | retry | A | 3 | retry_declined |  |
| 32 | run |  |  | run_paused(awaiting-retry) |  |
| 33 | run |  |  | run_resumed | paused(awaiting-retry) |
| 34 | approval | A | 3 | approval_channel | step=retry approved=false channel=none |
| 35 | retry | A | 3 | retry_declined |  |
| 36 | run |  |  | run_paused(awaiting-retry) |  |
| 37 | run |  |  | run_resumed | paused(awaiting-retry) |
| 38 | approval | A | 3 | approval_channel | step=retry approved=true channel=tty |
| 39 | retry | A | 3 | retry_confirmed |  |
| 40 | attempt | A | 3 | attempt_created | ns=pilot-w-a-3 induction=None |
| 41 | attempt | A | 3 | attempt_running | ghcr.io/tosin2013/specfoundry-pilot-worker@sha256:8656c77f8dcf5a29dab2012ccf1cc4f141456775081221de7c70ebcf5748efbc |
| 42 | attempt | A | 3 | attempt_collected |  |
| 43 | grade | A | 3 | grade_started | pilot-g-a-3 |
| 44 | grade | A | 3 | grade_pass |  |
| 45 | publish | A | 3 | publish_started | pilot/20261115T023426Z/a |
| 46 | publish | A | 3 | pr_opened | https://github.com/tosin2013/craz3-pilot-fixture/pull/16 |
| 47 | gate | B |  | gate_refuse | seq=5 DEP_UNSATISFIED:001 |
| 48 | run |  |  | run_paused(awaiting-merge) |  |
| 49 | run |  |  | run_resumed | paused(awaiting-merge) |
| 50 | approval | A |  | approval_channel | step=reconcile approved=true channel=tty |
| 51 | reconcile | A | 3 | reconcile_applied | 2449304c0c394bdbd4d38840d7475aed65a20ec1 |
| 52 | reconcile | A |  | merge_wait_seconds | 91 |
| 53 | status | A |  | task_verified | 2449304c0c394bdbd4d38840d7475aed65a20ec1 |
| 54 | gate | B |  | gate_allow | seq=6 OK |
| 55 | attempt | B | 1 | attempt_created | ns=pilot-w-b-1 induction=None |
| 56 | attempt | B | 1 | attempt_running | ghcr.io/tosin2013/specfoundry-pilot-worker@sha256:8656c77f8dcf5a29dab2012ccf1cc4f141456775081221de7c70ebcf5748efbc |
| 57 | attempt | B | 1 | attempt_collected |  |
| 58 | grade | B | 1 | grade_started | pilot-g-b-1 |
| 59 | grade | B | 1 | grade_pass |  |
| 60 | publish | B | 1 | publish_started | pilot/20261115T023426Z/b |
| 61 | publish | B | 1 | pr_opened | https://github.com/tosin2013/craz3-pilot-fixture/pull/17 |
| 62 | audit |  |  | audit_done | 10/10 |
| 63 | run |  |  | run_succeeded |  |
| 64 | teardown |  |  | torn_down | teardown.jsonl |
| 65 | audit |  |  | audit_done | 11/11 |
| 66 | audit |  |  | audit_done | 11/11 |

## Known limits

- Gate runs on the host (no published spec-status CLI image yet, PRD §8 ask 3).
- Grader uses `spec status --json`, not `verify-readiness` (report-only skill mode not shipped, PRD §8 ask 1).
- Reconcile is a deterministic pilot equivalent of the spec-status reconcile skill.
- Spend cap is enforced by the pilot watchdog and deadline unless a provider-side cap is declared (R2).
- Single-node kind only; multi-node policy enforcement not covered.
