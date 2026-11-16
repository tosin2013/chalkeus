# Contract: parity checks (golden audit and shared vectors)

## Golden audit (`specs/016-go-pilot/golden/audit-expected.json`)

```json
{
  "generated_by": "python reference <sha>",
  "records": [
    {"path": "specs/002-m1-gated-pilot/evidence/20261113T030655Z", "final": true,
     "rows": [{"id": "AU01", "result": "pass", "detail": "refusals=… attempts_before=0 allows_before=0"}, "…"]}
  ]
}
```

- **Records:** every directory `specs/*/evidence/<run-id>/` that contains a `run.json`, whatever
  its state. The failed 002 run is included.
- **Audit mode:** final when `teardown.jsonl` exists, as the former `pilot/tests/test_committed_evidence.py` did.
- **Pass rule for the Go audit:** the same row ids in the same order, with an identical `result`
  and an identical `detail` string.
- **Generation:** by `pilot/tools/golden_audit.py` (removed in US5) from the Python reference, run by hand and
  committed. Tests never write it.

## Shared vectors (`specs/004-github-loop/contracts/vectors/`)

### `approvals.json`

```json
{
  "version": 1,
  "cases": [
    {
      "name": "accept-valid",
      "input": {
        "decision": {"comment_id": 100, "step": "retry"},
        "approvers": ["alice"],
        "publisher": "pub-bot",
        "permissions": {"alice": "write"},
        "comments": [{"id": 101, "author": "alice", "body": "/approve retry",
                      "created_at": "t1", "updated_at": "t1", "app": false}]
      },
      "expect": {"accepted": 101, "log": [{"comment_id": 101, "reason": null}]}
    }
  ]
}
```

There is one case per rule from 004 research R3:
- accept;
- `wrong_step`, `before_decision`, `not_an_approver`, `no_write_access`,
  `publisher_identity`, `app_authored`, `edited`, `withdrawn`, `duplicate`;
- a withdrawal by another user doesn't cancel the approval;
- extra words on the first line are rejected as `wrong_step`;
- non-command comments are skipped; author matching ignores case; `maintain` counts as write access.

### `status_pr.json`

Cases for `check_status_pr`:
- exact edit by an approver → ok;
- not merged → `NOT_MERGED`;
- an extra file, an extra line, or the wrong item → `STATUS_PR_EDITED`;
- merged by the publisher → `STATUS_PR_MERGED_BY_PUBLISHER`;
- merged by a non-approver → `STATUS_PR_NOT_APPROVER`.

Inputs: `base_text`, `head_text`, `changed`, `item`, `merged_by`, `state`, `approvers` and
`publisher`.

### Rule

Both the Python reference tests (until retirement) and the Go tests must pass every case. The
operator (011) must pass them too. A new rule means a new case first.
