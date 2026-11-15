# 004 evidence

Copy each validation run's record here from `pilot/runs/<run-id>/` on the machine that ran it.
Never paste tokens, keys, hostnames, IPs or personal data. The audit's AU08 must pass for
every run.

## Results (owner)

| Check (quickstart) | Result |
|---|---|
| §2 sync: 2 issues, one `status:planned` label each, task checklists; second sync 0/0; hand-edited label restored | |
| §3 token mode: `/approve mode` and `/approve retry` resume the run | |
| §3 refusals: wrong step · second account not listed · edited comment · withdrawn · publisher-token comment | |
| §3 status PR: edited PR refused (`STATUS_PR_EDITED`); unedited merge accepted | |
| §3 audit: 11/11, AU09/AU10 `channels=['github']` with your login; `github drift` empty | |
| §4 App setup time from the docs alone (SC-007, target under 30 min, 0 outside lookups) | |
| §4 App mode: PRs and posts by `<app>[bot]`; probe passes on the fixture, fails on the canary; audit 11/11 | |
| §5 terminal run with GitHub loop off: prompts as in the guide; AU09/AU10 `channels=['tty']` | |
