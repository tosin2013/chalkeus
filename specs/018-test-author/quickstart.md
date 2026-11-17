# Quickstart: Test author (first use, `cli-schema-mcp` item 006, minimal level)

## Prerequisites

- Cluster up (`pilot/pilot up`); profile `targets/cli-schema-mcp.env` with
  `PILOT_COMPANION_REPO=tosin2013/specfoundry-acceptance` and `PILOT_COMPANION_PATH=cli-schema-mcp`.
- `PILOT_SECURITY_LEVEL` unset (minimal).
- The target's `main` branch protected (recommended at minimal).

## Steps

1. Open a **new** agent session (not the one that runs the pilot) and load the test-author
   skill.
2. In it: `pilot/pilot author start --target cli-schema-mcp --item 006`.
   Expected: the workspace path, `level=minimal`, the identity, and `can_write_target=true` with
   the branch-protection reminder.
3. The agent drafts tests and the map, and runs `pilot/pilot author check …`.
   Expected eventually: `DRAFT_OK: project tests pass; drafted tests fail on the base`.
4. `pilot/pilot author submit …`. Expected: a PR on `specfoundry-acceptance` touching only
   `cli-schema-mcp/items/006/`.
5. The human reviews and merges it.
6. In the pilot session: `pilot/pilot gate --target cli-schema-mcp --item 006`.
   Expected: `allow`, `hidden_files ≥ 1`, mapping `uncovered=[]` (or listed).

## Checks

- Draft evidence (`<evidence root>/drafts/<id>/`) holds no test contents (grep for a string from
  the tests finds nothing).
- A run's mode-approval line shows `level=minimal`.
- Put a copy of a hidden test into the target → `gate` refuses `HIDDEN_TESTS_IN_TARGET`.
