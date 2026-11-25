# 031 evidence: macOS, multi-platform images and Docker Desktop

## Runs

| Run | Host | Target | Item | Result | Notes |
|---|---|---|---|---|---|
| [20261124T183824Z-isolation](20261124T183824Z-isolation/) | darwin/arm64, Docker Desktop | (isolation suite) | — | 37/37 checks pass | T024: the M0 isolation suite, unmodified, on the Mac. Calico enforces NetworkPolicy on Docker Desktop's VM. |
| [20261125T162546Z](20261125T162546Z/) | linux/amd64, Docker Engine | fixture | A, B | succeeded; checks 32/32, audit 11/11 | First T013 run; not counted. The host's own spec-status was 0.3.0, so `versions.txt` paired 0.3.0 with the pinned 0.4.1 checksum. Led to the `host_spec:` line and doctor's `spec-status` row (#201). |
| [20261125T170145Z](20261125T170145Z/) | linux/amd64, Docker Engine | fixture | A, B | succeeded: A/1 failed G5 (induced), A/2 and B passed; checks 32/32, audit 11/11, teardown 3/3 | **T013**, the Linux regression: same check ids and results as 026's run 20261123T042536Z. Approvals on GitHub. |
| [20261125T182055Z](20261125T182055Z/) | darwin/arm64, Docker Desktop | fixture | A, B | paused at B's retry; checks 28/30 | First Mac fixture run; not counted. A passed and was reconciled; B/1's worker never started because Calico's 24-hour CNI token had expired (the VM's clock stopped while the Mac slept). P06/P07 failed with `probe not ready`. Led to troubleshooting `#calico-token-expired` (#206) and T034's check (#207). |
| [20261125T191440Z](20261125T191440Z/) | darwin/arm64, Docker Desktop | fixture | A, B | succeeded: A/1 failed G5 (induced), A/2 and B passed; checks 32/32, audit 11/11 | **T025**: check ids and results identical to T013. Then **T023**: `down` passed T01–T03. |
| [20261125T220507Z](20261125T220507Z/) | darwin/arm64, Docker Desktop | stockroom (`tosin2013/stockroom`) | 011 | succeeded: attempt 1 passed G1–G5; checks 17/17, audit 11/11 | **T026**: a new item (Low-stock products), `jdk21-maven` for arm64. Hidden tests from a separate test-author session, which caught a spec/plan conflict before the build (stockroom#54). T034's token check ran live. |

All runs used the model `deepseek/deepseek-v4-pro` and the GitHub App publisher
`tosins-specfoundry-pilot[bot]`. On the Mac, the owner approved with `./chalkeus forge approve`
(channel `harness`).

## Notes

| File | What it records |
|---|---|
| [local-arm64-build.md](local-arm64-build.md) | T009: every image built locally for linux/arm64, smoke-tested, and three fail-closed checks (wrong pins, legacy builder) |
| [image-platforms.md](image-platforms.md) | T012: every image in `pilot/images.lock` is an index with linux/amd64 and linux/arm64 |
| [mac-up.md](mac-up.md) | T022–T026 on the Mac: `up`, refusals with Docker stopped, doctor's rows, the Calico token finding, the fixture and target runs, `down` |
| [fresh-mac.md](fresh-mac.md) | T030's replacement: the guide's macOS install steps run as written in a throwaway home (the full walkthrough moved to roadmap 034) |
| [privacy.md](privacy.md) | T031: secret scan of this folder (none found), test results, two findings (draft evidence committed by mistake; prefetch log tails) |
