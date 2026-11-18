# 018 evidence: the test author's first use

| Run | Target | Item | Hidden tests | Result | Notes |
|---|---|---|---|---|---|
| [20261118T030513Z](20261118T030513Z/) | `tosin2013/cli-schema-mcp` | 006 | drafted by a separate test-author session (14 tests, one file, every criterion covered), merged by the owner (`specfoundry-acceptance` #1) | succeeded: attempt 1 passed G1–G7; owner merged #43 and status PR #44; audit 11/11 after `down` | Level `minimal`. Secret scan: 0 model-host hits, 0 token patterns; the hidden test's file name appears only in `companion.json` (names and hashes by design). |

Earlier runs of the same item, kept on disk and not committed: `20261118T015314Z` (the provider
had blocked the model key; three attempts, 0 tokens) and `20261118T024058Z` (the grader parsed a
cut-off `spec status` output, fixed in PR #70).
