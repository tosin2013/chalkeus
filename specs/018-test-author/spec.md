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

- Q: Which identity should the test author use in laptop mode? → A: A second GitHub App (for
  example "specfoundry-test-author"), installed only on the companion repository, with contents
  and pull-request write. Its tokens are minted per draft and scoped to the companion. It is
  never the publisher's App and never the operator's account. (Owner said "proceed" with the
  recommended options; the owner may override.)
- Q: In laptop mode, does the test author run in the local cluster or as a local agent session?
  → A: In the local cluster, as its own Job in its own namespace with its own egress proxy, like
  a worker but under the test-author role. The same design then carries to cluster mode (011)
  unchanged. (Recommended option; owner may override.)

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

1. **Given** the test author runs, **When** its isolation is checked, **Then** it runs under its
   own identity, separate from the publisher's and the operator's, and that identity can write
   only the companion repository (it cannot write the target repository).
2. **Given** the test author runs, **When** its network reach is checked, **Then** it can reach
   only the model host(s) and the companion repository's host; it cannot reach the worker or
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
- **FR-005**: The test author MUST run under its own identity, whose write access is limited to
  the companion repository; the request is refused if that identity can write the target. In
  laptop mode the identity is a second GitHub App installed only on the companion, separate
  from the publisher's App and the operator's account; its tokens are minted per draft and
  scoped to the companion.
- **FR-006**: The test author MUST run isolated from build workers and the publisher: its own
  network role (model host(s) and the companion's host only), its own execution boundary, and
  no shared storage with workers (C-07, C-08). In both modes it runs in the cluster as its own
  Job in its own namespace (in laptop mode, the local cluster), never as an agent session on
  the operator's machine.
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
- **FR-012**: The same draft request MUST work in laptop mode and, later, cluster mode (011),
  with the same evidence and guarantees (C-11).

### Key Entities

- **Draft request**: target, item, requester, time; its outcome (pull request or a refusal
  reason).
- **Draft**: the files proposed for `<path>/items/<NNN>/`, their hashes, the mapping, the spec
  hash, and the base-check results.
- **Mapping**: for each hidden test, the acceptance criteria it covers; plus uncovered criteria.
- **Test-author identity**: the account or app under which the pull request is opened, with
  write access to the companion only.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: For item 006 of `cli-schema-mcp`, the operator goes from "no hidden tests" to a
  reviewable pull request with one command and no hand-written test code.
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
- Cluster mode (011) is not built yet; this feature builds laptop mode and keeps the design
  usable by the operator later.
- The driving session may run the draft command, but by rule (pilot skill) never reads the
  pull request's files; only the operator reviews them.
