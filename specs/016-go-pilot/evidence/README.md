# 016 evidence: Go pilot runs

Each folder is one run's evidence record, written by the Go pilot (`run.json` has
`"impl": "go"`). The audit (AU01–AU11) checks each one offline.

| Run | Machine | Approvals | Result | Notes |
|---|---|---|---|---|
| `20261115T162029Z` | development machine | `harness` (the owner ran each `pilot/pilot approve` from the agent session) | 11/11, teardown T01–T03 pass | Pre-check before T037. Followed operator guide §7–§11 with `PILOT_IMPL=go` and `-vv`: gate A allow / B refuse (identical to the Python pilot), mode → A/1 fail (induction, G5) → retry → A/2 pass → draft PR #21 → owner merge → reconcile `001: planned → verified` → B pass → draft PR #22 → `down` → audit 11/11. Exit codes 5, 6, 0, 5, 0 as in the reference. |
| `20261116T021112Z` | validation machine | `harness` (the owner approved each step; the agent ran `pilot/pilot approve` after the owner's yes) | 11/11, teardown T01–T03 pass | T037 validation run. `PILOT_IMPL=go -vv`: gate A allow / B refuse, mode → A/1 fail (induction, G5, 303 479 tokens) → retry → A/2 pass (193 948 tokens) → draft PR #23 → owner merge → reconcile `001: planned → verified` → B pass (314 747 tokens) → draft PR #24 → `down` → audit 11/11. All approvals channel=harness. |
| `20261116T041113Z` | development machine | `github` (the owner's `/approve mode`, `/approve retry` and merges on GitHub; publisher: GitHub App `tosins-specfoundry-pilot[bot]`) | `failed: AUDIT` at run end (AU08 false positive); re-audited 11/11 after the fix, teardown T01–T03 pass | GitHub-loop pre-check before T047/T052, App mode, no terminal. The run's watcher resumed it after each GitHub action: mode → A/1 fail (G5) → retry → A/2 pass → publish failed (`create secret pilot-publish/scm-token: Forbidden`, fixed) → watcher stopped and `resume` by hand → draft PR #26 (`Closes #18`) → owner merge → status-change PR #27 (one line, merged by the owner, `ok: true`) → B pass → draft PR #28 (`Closes #19`) → drift 0 → audit AU08 FAIL. AU08 counted `PILOT_SCM_KIND=app` from `app-token.env` as a secret, so nearly every file matched; no token value or token/key pattern was in the evidence. After the fix, `pilot audit` gives 11/11 with `channels=['github'] approvers=['tosin2013']` on AU09/AU10. `run.json` keeps the original `failed: AUDIT`; `audit.jsonl` is the re-audit. The watcher exited when the run ended. |

Owner decision (2026-11-16): these two harness-approved runs complete T037 and SC-002; no
separate terminal (`tty`) run is required.
