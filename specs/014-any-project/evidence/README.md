# 014 evidence: item runs on real repositories

Each folder is one run's evidence record. `run.json` has `"kind": "item"` and names the target
and item. The audit (AU01–AU11, item-run variants in
[contracts/audit-item.md](../contracts/audit-item.md)) checks each one offline.

| Run | Machine | Target | Item | Approvals | Result | Notes |
|---|---|---|---|---|---|---|
| [20261118T030513Z](../../018-test-author/evidence/20261118T030513Z/) | this machine (kind) | `tosin2013/cli-schema-mcp` | 006 | GitHub (mode; merges of #43 and status PR #44) | succeeded; audit 11/11 after `down` | First real target. Python 3.13, PyPI via the registries proxy, offline graders, G1–G7 pass on attempt 1. Hidden tests drafted by the test author (018). Evidence lives under 018 (the active feature then). |
| [20261118T161914Z](../../018-test-author/evidence/20261118T161914Z/) | this machine (kind) | `tosin2013/stockroom` | 001 | GitHub (mode; merges of PR #9, status #10) | succeeded; audit 10/10 (AU07 needs the teardown, which `down` records only in the latest run) | Scaffold; first Java target (`jdk21-maven`), Maven Central via the target's proxies. |
| [20261118T171710Z](../../018-test-author/evidence/20261118T171710Z/) | this machine (kind) | `tosin2013/stockroom` | 002 | GitHub (mode; merges of PR #13, status #14) | succeeded; audit 10/10 (AU07 needs the teardown, which `down` records only in the latest run) | Products API; 20 hidden tests; `## Tests that may change` relaxed G7 for one earlier test. |
| [20261118T210446Z](../../018-test-author/evidence/20261118T210446Z/) | this machine (kind) | `tosin2013/stockroom` | 003 | GitHub (mode; merges of PR #17, status #18) | succeeded; audit 10/10 (AU07 needs the teardown, which `down` records only in the latest run) | Problem Details errors; the run closed its tracking issue (#88). |
| [20261118T212846Z](../../018-test-author/evidence/20261118T212846Z/) | this machine (kind) | `tosin2013/stockroom` | 004 | GitHub (mode; merges of PR #21, status #22) | succeeded; audit 10/10 (AU07 needs the teardown, which `down` records only in the latest run) | Persistence in embedded H2; earlier tests passed unedited. |
| [20261118T220113Z](../../018-test-author/evidence/20261118T220113Z/) | this machine (kind) | `tosin2013/stockroom` | 005 | GitHub (mode; merges of PR #25, status #26) | succeeded; audit 10/10 (AU07 needs the teardown, which `down` records only in the latest run) | Reservations; review found a stale-read flaw the tests missed, planned as 007. |
| [20261118T224610Z](../../018-test-author/evidence/20261118T224610Z/) | this machine (kind) | `tosin2013/stockroom` | 007 | GitHub (mode; merges of PR #30, status #31) | succeeded; audit 10/10 (AU07 needs the teardown, which `down` records only in the latest run) | The 005 fix; hidden tests fire 50 concurrent reservations. |
| [20261118T230306Z](../../018-test-author/evidence/20261118T230306Z/) | this machine (kind) | `tosin2013/stockroom` | 006 | GitHub (mode; merges of PR #33, status #34) | succeeded; audit 11/11 after `down` | Paging and filtering; the run `down` followed, so AU07 passes. |
| [20261119T154630Z](../../018-test-author/evidence/20261119T154630Z/) | this machine (kind) | `tosin2013/stockroom` | 008 | GitHub (mode; merges of PR #38, status #39) | succeeded; audit 10/10 (AU07 needs a later `down`) | The name filter's literal wildcards, found in the review of 006. First run on OpenCode Go (52k tokens). First live P06r row (pass); the agent's request to `github.com` was refused by the target's proxy and recorded (SC-004). |
| [20261119T164920Z](../../018-test-author/evidence/20261119T164920Z/) | this machine (kind) | the fixture (`craz3-pilot-fixture`) | A, B | GitHub (mode, retry; merges of #47, status #48, #49) | succeeded; audit 11/11 after `down` | T056 and SC-005: the fixture unchanged on the final code, first on OpenCode Go (A/1 53k, A/2 21k, B/1 30k tokens). Ran right after Stockroom 008 with no configuration edit (SC-007). |

The Stockroom records stay under 018 (the active feature when they ran), like the cli-schema-mcp
record; this table links them rather than copying them. Every run passed on its first attempt.
