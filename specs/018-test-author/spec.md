# Feature Specification: Test author

**Feature Branch**: `018-test-author`

**Created**: 2026-11-17

**Status**: Draft

**Input**: User description: "018 Test author (roadmap entry 018, owner decision 2026-11-17). A
separate test-author agent drafts an item's hidden acceptance tests; humans still plan and
approve; build workers never write or see those tests. For a prepared item, the test author
opens a pull request on the private companion repository; a human reviews and merges it. …
First use: item 006 of tosin2013/cli-schema-mcp, companion tosin2013/specfoundry-acceptance,
path cli-schema-mcp."

## Context

Hidden acceptance tests decide whether a build worker's change does what an item's spec asks.
Until now a human had to write them by hand (014). Roadmap 1.16 (C-12, C-14) allows a separate
**test author** to draft them, as long as a human reviews and merges the draft and no build
worker, worker session or publisher can ever see the tests. This feature adds that test author.

Terms used below:
- **Operator**: the human who prepares items and approves runs.
- **Driving session**: whatever runs `pilot/pilot` for the operator (a terminal, or a coding
  agent following the pilot skill). It launches build runs.
- **Test author**: the agent that drafts hidden tests. It is never the driving session's own
  agent, and never a build worker.
- **Companion**: the private repository holding hidden tests, at
  `<PILOT_COMPANION_PATH>/items/<NNN>/` (014, C-14).

## Clarifications

### Session 2026-11-17

- Q: Which identity should the test author use? → A: It follows the security level (roadmap
  C-14): **minimal** (laptop default) the operator's own git access; **recommended** (teams) a
  token or App that can write only the companion; **strict** (enterprise, cluster default) a
  separate GitHub App installed only on the companion. Separate identities are recommended for
  teams and expected in enterprise environments. (Owner, 2026-11-17: "recommended and minimal
  settings"; a solo developer shouldn't need several Apps. Replaces an earlier answer that
  required a second App everywhere.)
- Q: In laptop mode, does the test author run in the local cluster or as a local agent session?
  → A: As a **separate agent session** following a test-author **skill** (never the session that
  drives runs); in cluster mode, as its own Job running an agent with the same skill (roadmap
  018 scope). Replaces an earlier answer that put a Job in the local cluster.
- Q: Should there be a developer mode? → A: Yes, as its own spec (019, roadmap C-15): an
  explicit `--dev` flag that may let the same session draft tests, but stamps the run as a
  development run that never counts and never changes a roadmap status.

## Why the rules exist

A build worker sees only the item's spec and tasks and a snapshot of the **target repository**,
and it has no route to GitHub. The one realistic way it could learn hidden tests is if they land
in the target repository: the test author copies them there by mistake or through a prompt
injection in a file it reads, or a publisher bug carries them over. The next run's snapshot
would then contain them. So what matters is **who can write the target** and **who can read the
companion**; the security levels (roadmap C-14) set exactly that:

| Level | Test author's identity | What prevents tests reaching the target |
|---|---|---|
| minimal (laptop default) | the operator's own access | branch protection on the target, human review of every PR, the skill's rules, P11 |
| recommended (teams) | a token or App that can write only the companion | GitHub itself: the identity can't write the target |
| strict (enterprise, cluster default) | a separate App, companion only, in its own Job | as recommended, plus enforced egress (C-07) |

Build-worker isolation is identical at every level.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Draft hidden tests for a prepared item (Priority: P1)

As an operator with an item whose spec and tasks are ready but which has no hidden tests, I ask
for a draft and receive a pull request on the companion repository containing tests that fail
on today's code, so I only have to review and merge them.

**Why this priority**: This is the whole point of the feature. Without it, every item needs
hand-written tests before a run.

**Independent Test**: For item 006 of `cli-schema-mcp` (spec merged, companion folder empty),
request a draft. A pull request appears on `specfoundry-acceptance` adding files only under
`cli-schema-mcp/items/006/`; its description says the drafted tests fail on the unchanged base
and the project's own tests still pass.

**Acceptance Scenarios**:

1. **Given** an item that passes every readiness rule except `ITEM_NO_HIDDEN_TESTS`, **When**
   the operator requests a draft, **Then** a pull request is opened on the companion repository
   that adds files only under the item's folder, and nothing is written anywhere else.
2. **Given** a draft whose tests pass on the unchanged base (no signal) or break the project's
   own suite, **When** the test author finishes, **Then** no pull request is opened and the
   operator is told why (`DRAFT_HIDDEN_PASSES` or `DRAFT_SUITE_FAILS`).
3. **Given** the item's companion folder already has files, **When** a draft is requested,
   **Then** it is refused (`COMPANION_NOT_EMPTY`) unless the operator explicitly asks to replace
   them, and a replacement is still only a pull request.
4. **Given** an item that is not otherwise ready (for example no spec, open questions, no
   allowed paths), **When** a draft is requested, **Then** it is refused with the same readiness
   reason a run would give.

---

### User Story 2 - The tests never reach a build worker (Priority: P1)

As an operator, I need assurance that drafting tests with an agent does not weaken the
guarantee that build workers never see them.

**Why this priority**: It is the condition under which the owner allowed agent-drafted tests
(C-14). Without it, the feature must not ship.

**Independent Test**: Draft tests for an item, merge them, then run the item. The run's checks
show that no hidden-test file name or content hash reached any worker input (P11 passes), and
the test author's own evidence contains no test contents.

**Acceptance Scenarios**:

1. **Given** the recommended or strict level, **When** the test author's identity is checked
   before the model is called, **Then** it can write the companion and cannot write the target;
   otherwise the draft is refused (`AUTHOR_CAN_WRITE_TARGET`). At the minimal level the check
   reports that the identity can write the target and records it, and the operator is reminded
   to protect the target's main branch.
2. **Given** the strict level, **When** the test author's network reach is checked, **Then** it
   can reach only the model host(s) and the companion's host; it cannot reach the worker or
   publisher proxies, and no worker can reach it.
3. **Given** the test author is running or has run, **When** the evidence and the driving
   session's output are inspected, **Then** neither contains test contents: only file names,
   counts, hashes, the identity and the companion commit.
4. **Given** the test author, **When** it acts, **Then** it never starts, steers or reviews a
   build run, never touches a worker branch, and never edits the target's specs, tasks or
   roadmap.

---

### User Story 3 - Every acceptance criterion is covered (Priority: P2)

As an operator reviewing a draft, I want to see which acceptance criteria each hidden test
covers and which criteria have no test, so I can judge the draft quickly and spot gaps.

**Why this priority**: It makes review fast and catches silent gaps, but drafts are useful
without it.

**Independent Test**: For item 006, the draft's pull request lists each acceptance scenario and
functional requirement of the spec with the tests that cover it; any criterion with no test is
listed as uncovered.

**Acceptance Scenarios**:

1. **Given** a draft, **When** it is opened as a pull request, **Then** it includes a mapping
   from each hidden test to the spec's acceptance criteria it covers.
2. **Given** a criterion no test covers, **When** the mapping is checked, **Then** the criterion
   is flagged as uncovered in the pull request and, after merge, in the run's readiness record
   and at the mode approval.
3. **Given** merged hidden tests with no mapping (for example written by hand), **When** an item
   is checked, **Then** the run is not refused for that; the mapping check reports "no mapping".

---

### Edge Cases

- The model produces no files, files outside the item's folder, or files that aren't tests: no
  pull request; the reason is reported.
- The drafted tests need a dependency the project doesn't declare: the draft is rejected
  (`DRAFT_SUITE_FAILS` or `DRAFT_HIDDEN_PASSES` explains it), because the grader is offline.
- The spec changes after a draft was opened: the pull request records the spec's hash, and the
  operator can request a new draft.
- The model is unavailable or exceeds the spend cap: the draft stops; nothing is written.
- A second draft request for the same item while one is open: refused (`DRAFT_PR_OPEN`).
- The companion is the same repository as the target, or the identity can write the target:
  refused before the model is called.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The operator MUST be able to request a draft of an item's hidden tests for a
  configured target; the request runs the same readiness rules as a run, except the hidden-test
  rule.
- **FR-002**: The test author MUST receive the item's spec, plan and tasks and the project's
  base snapshot, and nothing from any build run.
- **FR-003**: Before opening a pull request, the drafted tests MUST be checked against the
  unchanged base: the project's own tests pass and the drafted tests fail. Otherwise no pull
  request is opened.
- **FR-004**: The output MUST be a single pull request on the companion repository that adds or
  changes files only under `<PILOT_COMPANION_PATH>/items/<NNN>/`. A human merges it; the pilot
  never merges (C-04).
- **FR-005**: The test author's identity MUST follow the security level (roadmap C-14):
  minimal (the laptop default) uses the operator's own access; recommended uses a token or App
  that can write only the companion; strict uses a separate App installed only on the
  companion. At recommended and strict, a draft MUST be refused if the identity can write the
  target. The level and identity MUST be recorded in the evidence.
- **FR-006**: The test author MUST be a skill run by an agent session separate from build
  workers and from the session that drives runs: on a laptop a separate agent session; in
  cluster mode its own Job and namespace. It shares no storage with workers (C-08). At the
  strict level its egress is limited to the model host(s) and the companion's host (C-07).
- **FR-006a**: `pilot/pilot` MUST provide the helper steps the skill uses, so that the session
  never handles raw credentials or test contents in its output: check a draft against the
  unchanged base (offline, with the run's dependency cache), check the mapping, and open the
  companion pull request under the level's identity.
- **FR-007**: Evidence for a draft MUST record: the target, item, spec hash, identity, model,
  companion repository, folder, the pull request, the commit it adds, file names and hashes,
  the base check's results and the mapping; it MUST NOT record test contents. Output shown to
  the driving session follows the same rule.
- **FR-008**: The draft MUST include a mapping from each hidden test to the spec's acceptance
  criteria (acceptance scenarios and functional requirements) and list criteria with no test.
- **FR-009**: When a run reads the companion, the mapping (if present) MUST be checked and any
  uncovered criteria reported in the readiness record and at the mode approval; a missing
  mapping is reported, not refused.
- **FR-010**: The meanings of `ITEM_NO_HIDDEN_TESTS`, `BASELINE_HIDDEN_PASSES`, G5 and P11 MUST
  not change; merged drafts are read exactly like hand-written tests.
- **FR-011**: The test author MUST NOT start, steer or review build runs, touch worker branches,
  or edit the target's specs, tasks or roadmap.
- **FR-012**: The same skill and helper steps MUST work in laptop mode and, later, cluster mode
  (011), with the same evidence (C-11).
- **FR-013**: The level MUST be shown at the start of each draft and each run, and reported by
  the audit; a lower level never fails a run.

### Key Entities

- **Draft request**: target, item, requester, time; its outcome (pull request or a refusal
  reason).
- **Draft**: the files proposed for `<path>/items/<NNN>/`, their hashes, the mapping, the spec
  hash, and the base-check results.
- **Mapping**: for each hidden test, the acceptance criteria it covers; plus uncovered criteria.
- **Test-author identity**: the account, token or app under which the pull request is opened;
  companion-only at the recommended and strict levels.
- **Security level**: minimal, recommended or strict (roadmap C-14), chosen in the target's
  profile; recorded in every draft's and run's evidence.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: For item 006 of `cli-schema-mcp`, the operator goes from "no hidden tests" to a
  reviewable pull request without writing test code, and at the minimal level without creating
  any new App or token.
- **SC-002**: 100% of opened drafts fail on the unchanged base and keep the project's own tests
  passing.
- **SC-003**: In every run that uses drafted tests, no test file name or content hash appears in
  any worker input, prompt, pull request body or evidence file outside the companion record
  (P11 passes; an evidence scan finds no test content).
- **SC-004**: Every opened draft lists, for 100% of the spec's acceptance criteria, either the
  tests that cover it or "uncovered".
- **SC-005**: A refused draft always names one reason code the operator can act on.

## Assumptions

- The test author uses the same model configuration as build workers (`pilot.env`), with its
  own spend cap per draft.
- The test author needs the project's dependencies only to check its draft; that check reuses
  the run's offline grading (cache and grader) rather than giving the test author registry
  access.
- Drafting is requested explicitly by the operator; it is never triggered automatically.
- Cluster mode (011) is not built yet; this feature builds the skill, the helper steps and
  laptop mode at all three levels (strict's enforced egress arrives with 011's Job).
- Developer mode (019) is a separate feature; there, the same session may draft tests.
- The session that drives runs never acts as the test author and never reads the companion's
  files (pilot skill); only the operator reviews them.
