# 014 evidence: item runs on real repositories

Each folder is one run's evidence record. `run.json` has `"kind": "item"` and names the target
and item. The audit (AU01–AU11, item-run variants in
[contracts/audit-item.md](../contracts/audit-item.md)) checks each one offline.

| Run | Machine | Target | Item | Approvals | Result | Notes |
|---|---|---|---|---|---|---|
| [20261118T030513Z](../../018-test-author/evidence/20261118T030513Z/) | this machine (kind) | `tosin2013/cli-schema-mcp` | 006 | GitHub (mode; merges of #43 and status PR #44) | succeeded; audit 11/11 after `down` | First real target. Python 3.13, PyPI via the registries proxy, offline graders, G1–G7 pass on attempt 1. Hidden tests drafted by the test author (018). Evidence lives under 018 (the active feature then). |
