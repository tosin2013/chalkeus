---
name: "test-author"
description: "Draft the hidden acceptance tests for one prepared roadmap item of a specfoundry target, and open a pull request on the private companion repository for a human to review and merge. Use only in a session dedicated to test authoring, never in the session that runs the pilot."
---

# Test author

You draft the **hidden acceptance tests** for one item of a target repository. They decide
whether a build worker's change does what the item's spec asks. Build workers never see them,
and the pilot only uses them after a human reviews and merges your pull request
([specs/018-test-author](../../../specs/018-test-author/spec.md), roadmap C-12, C-14).

## Before you start

- This session must be **only** the test author. If this session has run `pilot/pilot run`,
  `resume`, `approve` or `github`, stop and ask the human to open a fresh session.
- Ask the human for the target name and item number (for example `cli-schema-mcp`, `006`).

## Steps

1. Run `pilot/pilot author start --target <name> --item <NNN>` from the specfoundry root. It
   checks that the item is ready, then prints a **workspace** path. Work only there. If it
   refuses, tell the human the code and its message; don't work around it.
2. Read, in the workspace: `spec.md`, `plan.md`, `tasks.md`, `criteria.json`, and the project as
   it is today under `base/` (read-only).
3. Write tests under `draft/`, laid out like the project (for example `draft/tests/test_x.py`
   for a project whose tests live in `tests/`). The tests must:
   - **fail on today's code** and pass once the spec is implemented;
   - test behaviour the spec states, through the project's public interfaces, not its internals;
   - use only the project's test framework and dependencies it already declares (the grader is
     offline);
   - not depend on anything outside the project (network, clock, other files of yours).
4. Write `draft/.acceptance-map.json`:
   ```json
   {"version": 1, "spec_sha256": "<from criteria.json>",
    "criteria": {"US1-AS1": ["tests/test_x.py::test_name"]},
    "uncovered": ["<ids from criteria.json with no test>"]}
   ```
   Cover every criterion you can; list the rest under `uncovered`, honestly.
5. Run `pilot/pilot author check --target <name> --item <NNN>`. It runs your tests on the
   unchanged project on the cluster (a few minutes) and shows you their output. Fix and re-run
   until it prints `DRAFT_OK`:
   - `DRAFT_HIDDEN_PASSES`: your tests already pass today; they must fail until the change exists.
   - `DRAFT_SUITE_FAILS`: with your tests added, the project's own tests fail; don't break them.
   - `MAP_INVALID`: fix the map as the message says.
6. Run `pilot/pilot author submit --target <name> --item <NNN>` and give the human the pull
   request link. **The human reviews and merges it.** You don't.

## Automatic drafting

With `PILOT_AUTHOR_AUTO=on`, `pilot/pilot author watch` runs these same rules in a sandboxed Job
on the cluster, with no human in the session (specs/024-test-author-automation). Its prompt is
built from steps 3 and 4 above; a test keeps the two in step, so change both together.

## Never

- Run `pilot/pilot run`, `resume`, `approve`, `github`, `reset-fixture`, `up` or `down`.
- Push to, or open a pull request on, the **target** repository, or edit its specs, tasks or
  roadmap. Your only output is the companion pull request that `submit` opens.
- Write outside the workspace's `draft/` folder.
- Paste test contents anywhere except the draft files (not in chat summaries for other sessions,
  issues or the target).
- Read `~/.config/specfoundry/` or print tokens.
