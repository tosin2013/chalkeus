# Research: Test author

## R1 — The test author is a skill run by a separate session

- **Decision:** a skill file, `.claude/skills/test-author/SKILL.md`, followed by a fresh agent
  session (Claude Code, Cursor, OpenCode). The agent reads the item's spec (from the draft
  workspace), writes test files under `<workspace>/draft/`, writes `.acceptance-map.json`, and
  calls `pilot/pilot author check` until it passes, then `pilot/pilot author submit`.
- **Rationale:** roadmap 018 ("a test-author skill … with its own agent session") and the owner's
  levels decision. On a laptop, isolation between sessions is a rule plus the identity, not an
  enforced boundary; strict (011) runs the same skill in a Job.
- **Rules in the skill:** never run `run`, `resume`, `approve`, `github`; never push to, or open
  a PR on, the target; never edit the target's specs, tasks or roadmap; write only under the
  workspace's `draft/`; never paste test contents into chat outside the draft files.
- **Alternatives:** a Job on the laptop (rejected by the owner as too heavy for the minimal
  level); the driving session drafting (not allowed outside dev mode, 019).

## R2 — Helper commands own every side effect

- **Decision:** `pilot/pilot author start|check|submit|status --target <t> --item <NNN>`.
  - `start`: readiness without the hidden-test rule; refuses `COMPANION_NOT_EMPTY`,
    `DRAFT_PR_OPEN`, `AUTHOR_CAN_WRITE_TARGET` (recommended); creates the workspace with
    `spec.md`, `plan.md`, `tasks.md`, the base checkout (read-only reference) and
    `criteria.json`; prints the workspace path, level and identity.
  - `check`: validates the draft layout and mapping, then runs the baseline in the item's grading
    namespace with the draft as the overlay (same snapshot, toolchain, prefetch and grader as a
    run). Pass = suite passes and drafted tests fail. Prints the tests' output to the terminal
    (the author's own tests); evidence keeps the withheld summary.
  - `submit`: only after a passing `check` of the same draft hash; clones the companion with the
    level's identity, writes `<path>/items/<NNN>/`, pushes branch `author/<NNN>-<draft id>`,
    opens the PR with a body listing level, identity, spec hash, check results and the mapping
    table (criteria → test files).
  - `status`: the draft's state and evidence path.
- **Rationale:** the agent never handles credentials, and every refusal has one code (SC-005).

## R3 — Security levels

- **Decision:** profile key `PILOT_SECURITY_LEVEL` (`minimal` default; `recommended`;
  `strict`).
  - minimal: the operator's git access (as the companion reader does today). `start` checks
    whether that identity can push to the target and records `can_write_target: true` with a
    reminder to protect the target's main branch.
  - recommended: `targets/<name>.author.env` holds `PILOT_AUTHOR_TOKEN` (fine-grained, companion
    only). `start` refuses `AUTHOR_CAN_WRITE_TARGET` if the token can push to the target (GitHub
    API `permissions.push`).
  - strict: refused with `LEVEL_NEEDS_CLUSTER_MODE` until 011.
- **Runs:** the level is recorded in `run.json`, shown on the mode-approval line, and reported
  by the audit (AU09 detail), never failing a run.

## R4 — Coverage mapping

- **Decision:** `.acceptance-map.json` at the root of the item's companion folder:
  `{"spec_sha256": …, "criteria": {"US1-AS1": ["tests/test_x.py::test_a"], …}, "uncovered": […]}`.
  Criteria ids come from the spec: `US<n>-AS<m>` (acceptance scenario m of user story n) and
  `FR-<nnn>`. `author start` writes `criteria.json` with the ids and one-line titles.
- **Checks:** `author check` validates that every referenced file exists in the draft and every
  id exists in the spec; `uncovered` must list exactly the ids with no test. Readiness (runs)
  reads the map if present and reports uncovered ids and a stale `spec_sha256`; missing map →
  "no mapping", never a refusal.
- **Overlay:** the map file is excluded from the grader overlay and from `hidden_files`.

## R5 — `HIDDEN_TESTS_IN_TARGET`

- **Decision:** readiness hashes every hidden-test file and every file in the target snapshot;
  any equal content, or the same relative path with test content, refuses the run.
- **Rationale:** the leak path (spec, "Why the rules exist"). P11 checks worker inputs for
  file names and staged extras, but not the snapshot's contents.

## R6 — Author check output

- **Decision:** `grade.py` meta `author_check: true` adds `hidden_output.txt` to the frame
  (never to `grade.json`). The CLI prints it to the author's terminal and discards it; draft
  evidence stores `grade.json` (withheld summary only).

## R7 — Evidence

- **Decision:** `<evidence root>/drafts/<draft id>/draft.json` and `events.jsonl`: target,
  item, spec hash, level, identity login, can_write_target, model, file names and SHA-256s,
  each check's result (withheld detail), mapping summary, PR URL and commit. No file contents.

## R8 — The pilot skill

- **Decision:** the pilot skill gains: never run `pilot/pilot author …` and never read a draft
  workspace or the companion; when an item lacks hidden tests, tell the human to start a
  separate session with the test-author skill.
