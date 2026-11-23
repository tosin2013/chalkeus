# 026 evidence: slice runs

| Run | Target | Item | Slice | Result | Notes |
|---|---|---|---|---|---|
| [20261123T042536Z](20261123T042536Z/) | fixture (`tosin2013/craz3-pilot-fixture`) | A, B | none (whole item) | succeeded: A/1 failed G5 (induced), A/2 passed G1–G5; owner merged #59 and status PR #60; B/1 passed, PR #61; audit 11/11 after `down` | T046, quickstart §2: a whole-item run after 026 gives the same audit rows and results as before (SC-001). Model `opencode-go/deepseek-v4-pro`; publisher the GitHub App. Secret scan: 0 model-host hits, 0 token patterns. |

Earlier runs of T046 on the same day, kept on disk and not committed:

- `20261123T023705Z`: refused at the gate (`ALREADY_VERIFIED`); the fixture still had A verified
  from an earlier run. Fixed with `reset-fixture`.
- `20261123T024453Z`: every attempt ended with `harness_error` and 0 tokens. A/1: the LiteLLM proxy
  rejected the key (401). A/2: after switching `pilot.env` to OpenCode Go mid-run, the run still
  used the model saved when it started (`provider.no-route`). Left paused; a model change needs a
  new run.
- `20261123T025903Z`: A/1 failed G5 and A/2 passed, but the publisher was a token from the
  owner's own account, so the owner's merge of status PR #53 was refused
  (`STATUS_PR_MERGED_BY_PUBLISHER`). Fixed by restoring the GitHub App settings in `pilot.env`.
- `20261123T031511Z`: A/1 passed G1–G5 although the acceptance section was withheld, so AU02
  (A/1 fail → A/2 pass → B pass) failed; every other row passed. The model was not misled by the
  induction; not a 026 regression.
