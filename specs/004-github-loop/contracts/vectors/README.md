# Shared vectors

Language-neutral test cases for the GitHub approval rules (research R3) and the status-change PR
check (research R6). Every implementation must pass every case: the Python reference
(`pilot/tests/test_vectors.py`), the Go pilot (`internal/ghloop/vectors_test.go`), and later the
operator (roadmap 011). A new rule needs a new case first.

## `approvals.json`

Each case has:
- **`input.decision`:** `comment_id` of the decision post, and `step`;
- **`input.step`:** the pending step;
- **`input.approvers`:** the allowed logins;
- **`input.publisher`:** the publisher identity;
- **`input.permissions`:** login → `admin|maintain|write|read|none`; missing means `none`;
- **`input.comments`:** each with `id`, `body`, `author`, `created_at`, `updated_at` and `app`.

`expect.accepted` is the id of the accepted comment, or `null`. `expect.log` lists every
`/approve…` candidate in id order, with its `reason` (`null` means accepted).

## `status_pr.json`

Each case has `input` (`state`, `merged_by`, `base_text`, `head_text`, `changed`, `item`,
`approvers`, `publisher`) and `expect` (`ok`, `reason`).

The expectations were produced by the Python reference functions (`ghapprove.evaluate`,
`reconcile.check_status_pr`) and are checked by its tests.
