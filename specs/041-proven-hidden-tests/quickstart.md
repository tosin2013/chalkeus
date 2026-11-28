# Quickstart: proving documcp 004's hidden tests (041)

The first use, and SC-001.

1. **Author model** (US1): in `~/.config/chalkeus/targets/documcp.env` set
   `FORGE_AUTHOR_MODEL=<a stronger model on the same provider>`; `./chalkeus forge doctor --target
   documcp` shows `author model: … (worker: opencode-go/beam-free)` and probes its key.
2. **Reference** (owner): commit attempt 2's change from run 20261128T012908Z as
   `documcp/references/004.patch` in `tosin2013/specfoundry-acceptance`.
3. **Prove the merged tests**: `./chalkeus forge author prove --target documcp --item 004`.
   Expected today: `DRAFT_FAILS_ON_REFERENCE` (the merged test file doesn't compile); the owner
   sees the output in their terminal.
4. **Redraft or fix** the tests; `author check` now runs the reference check; `author submit`
   opens the PR with `proofs/004.json`. The PR body shows the model, "passes on the reference
   (1 of 1 files, 3 runs)" and the votes.
5. **Readiness**: `./chalkeus forge doctor --target documcp --item 004` shows the reference row
   as proven; without the proof it refuses with `HIDDEN_TESTS_UNVERIFIED`.
6. **Eval** (US2): `./chalkeus forge author eval` drafts documcp-004 (and 001 US2 once it has a
   reference) and prints the score.
