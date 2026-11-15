# 016 evidence: Go pilot runs

Each folder is one run's evidence record, written by the Go pilot (`run.json` has
`"impl": "go"`). The audit (AU01–AU11) checks each one offline.

| Run | Machine | Approvals | Result | Notes |
|---|---|---|---|---|
| `20261115T162029Z` | development machine | `harness` (the owner ran each `pilot/pilot approve` from the agent session) | 11/11, teardown T01–T03 pass | Pre-check before T037. Followed operator guide §7–§11 with `PILOT_IMPL=go` and `-vv`: gate A allow / B refuse (identical to the Python pilot), mode → A/1 fail (induction, G5) → retry → A/2 pass → draft PR #21 → owner merge → reconcile `001: planned → verified` → B pass → draft PR #22 → `down` → audit 11/11. Exit codes 5, 6, 0, 5, 0 as in the reference. |

Still to do (T037): the terminal run on the validation machine (expect `channels=['tty']`) and
the agent run there with harness approvals.
