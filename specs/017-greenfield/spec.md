# Feature Specification: Greenfield bootstrap

**Feature Branch**: `017-greenfield`

**Created**: 2026-11-18

**Status**: Draft

**Input**: Roadmap entry 017 (owner decisions 2026-11-15 and 2026-11-18): start a project from
an empty repository; Java/Maven first, Node.js later; the repository is planned before anything is
built (spec-first), with the controller drafting and the owner deciding (Q13).

## Context

014 runs prepared items on an existing repository. A new project has no code, no tests and no
build history, so the first item is a scaffold and the pilot must grade it with only the hidden
acceptance tests and the dependencies the owner approved while planning. 017 proves that loop on
a new repository and writes it up as a guide others can follow.

This spec was written after the guide (`docs/greenfield.md`) and the first run, by owner
decision ("guide-first"): it records what 017 must deliver and what is still open.

**Reference repository:** `tosin2013/stockroom`, a small Spring Boot 4.1 REST service on Java 21.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Plan a new repository in one approved pull request (Priority: P1)

As an owner, I start from an empty repository and, with my coding agent drafting, open one
planning PR that holds the constitution, the roadmap, the first items' specs and tasks (with
allowed paths), the build file with every dependency the planned items need, and CI. My merge
approves the plan; no build worker has run yet.

**Why this priority**: Everything after it depends on an approved plan, and the offline grader
can only use dependencies approved here (014 R7).

**Independent Test**: The planning PR's tree builds with no source code (the build reports no
tests and succeeds), and `spec status` reads its roadmap with no errors.

**Acceptance Scenarios**:

1. **Given** an empty repository, **When** the planning PR is merged, **Then** the roadmap lists
   the scaffold as the first item with a spec, tasks and allowed paths, and every later item
   depends on it directly or indirectly.
2. **Given** the merged plan, **When** `pilot/pilot doctor --target <t>` runs, **Then** it shows
   the whole roadmap and what each item still needs.
3. **Given** the scaffold's spec, **When** its allowed paths are checked, **Then** none of them
   reaches the build file, CI or planning files.

---

### User Story 2 - Run the scaffold to verified (Priority: P1)

As an owner, I give the scaffold hidden acceptance tests (written by me or drafted by the test
author and merged by me), and the pilot runs it: mode approval, dependency cache, baseline,
attempts, a draft PR I merge, and a status PR whose merge records the item verified.

**Why this priority**: It is the outcome of 017.

**Independent Test**: A run of the scaffold item ends `succeeded`, the item is recorded
`verified` on the target's roadmap, and the audit passes.

**Acceptance Scenarios**:

1. **Given** hidden tests that use classes the scaffold will add (so they don't compile on the
   empty project), **When** the pilot fills the dependency cache, **Then** the cache step passes
   and the grader later runs offline with everything it needs.
2. **Given** the empty project plus the hidden tests, **When** the baseline runs, **Then** the
   hidden tests fail because the code is missing, not because a dependency is missing.
3. **Given** a passing attempt, **When** the owner merges its PR and then the status PR, **Then**
   the item is recorded `verified` and the next item is unblocked.

---

### User Story 3 - A dependent item runs like any existing project (Priority: P2)

As an owner, after the scaffold is verified, I run the next item (a real endpoint that depends on
the scaffold) exactly as in 014, with no greenfield-specific step.

**Why this priority**: It shows the project keeps growing through the roadmap, not only that it
can start.

**Independent Test**: Item 002 of the reference repository runs to `succeeded` and is recorded
`verified`, with no configuration change between the two runs.

**Acceptance Scenarios**:

1. **Given** 001 verified and 002's hidden tests merged, **When** 002 runs, **Then** the readiness
   gate allows it and it is graded against the scaffold the first run built.
2. **Given** 002's change, **When** it is graded, **Then** the existing tests from 001 still pass
   unedited (G7) and the change adds tests (G6).

---

### User Story 4 - A reader can repeat it from the guide (Priority: P2)

As someone new to specfoundry, I follow `docs/greenfield.md` from an empty repository to a
verified first item, using the reference repository as a worked example.

**Why this priority**: The reference and the guide are what make 017 useful beyond one run.

**Independent Test**: Every step a reader takes in the reference run is in the guide, in order,
with the command or GitHub action it needs; the docs check passes.

**Acceptance Scenarios**:

1. **Given** the guide, **When** a reader reaches a decision (merge the plan, approve the mode,
   merge the item PR, merge the status PR), **Then** the guide says what they are approving.
2. **Given** the reference repository, **When** a reader opens its planning PR, **Then** it
   explains the choices and cites dated sources.

---

### User Story 5 - A Node.js greenfield follows (Priority: P3)

As an owner, I repeat the loop on a new Node.js 24 repository with npm.

**Why this priority**: Owner decision 2026-11-18: Java first, Node later.

**Independent Test**: A Node.js scaffold item runs to `verified` on the `node24` toolchain.

**Acceptance Scenarios**:

1. **Given** a new Node.js repository planned the same way, **When** its scaffold runs, **Then**
   the dependency cache is filled from the approved `package-lock.json` and the grader runs
   offline.

### Edge Cases

- **Hidden tests that compile on the empty project** (they look classes up at run time): the
  cache step runs them, they fail, and the cache is filled from that run.
- **A project with no tests at all yet**: the project's own test command must still succeed on
  the planning tree (Maven reports no tests), so the baseline can tell "no tests" from "broken".
- **A later item changes what an earlier item's test expects** (Stockroom 002 adds the endpoint
  001 tested as missing): the later item's spec lists that file under `## Tests that may
  change`, or G7 fails every attempt.
- **A dependency the plan forgot**: the grader is offline, so an attempt that needs it fails; the
  fix is a planning change merged by the owner, never an agent edit to the build file.
- **The pilot's evidence folder** is shared by item runs (today `specs/018-test-author/evidence/`);
  017's runs are recorded there and linked from the guide.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: A greenfield target MUST be configured with the same profile keys as any 014 target;
  no greenfield-only setting.
- **FR-002**: The dependency cache MUST be filled only from the owner-approved build file and the
  hidden tests, never from agent code (014 R7), including when the hidden tests don't compile on
  the empty project.
- **FR-003**: For Maven, the toolchain MUST provide a prefetch command that downloads the
  test-runner pieces the grader needs even when the hidden tests don't compile on the base.
- **FR-004**: The scaffold item MUST be graded like any item (G1–G7 when the profile sets
  `PILOT_TEST_PATHS`).
- **FR-005**: The planning files, the build file and CI MUST stay outside every item's allowed
  paths (constitution C-04 of the reference repository; 014's allowed-path rules).
- **FR-006**: `docs/greenfield.md` MUST walk a reader from an empty repository to a verified first
  item, and point to the reference repository and the run record.
- **FR-007**: The reference repository MUST have at least the scaffold and one dependent item
  recorded `verified` through the pilot.
- **FR-008**: A Node.js 24 greenfield MUST follow the Java one, on the `node24` toolchain.
- **FR-009**: The greenfield validation SHOULD pass on every cluster profile (kind, minikube,
  plain Kubernetes, OpenShift) as those profiles become available (roadmap note 2026-11-16);
  kind is the first.

### Key Entities

- **Planning PR**: the one pull request that holds the constitution, roadmap, first specs, build
  file and CI; its merge approves the plan.
- **Scaffold item**: the roadmap's first item; it adds the entry point, configuration and the
  project's first test.
- **Reference repository**: the public repository built this way, kept as a worked example.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: The reference repository's scaffold goes from approved mode to recorded `verified`
  with no step outside the guide. *(Met 2026-11-18: run 20261118T161914Z, 13 minutes, first
  attempt.)*
- **SC-002**: The scaffold's baseline fails on the hidden tests for a missing-code reason, and the
  cache step passes. *(Met in the same run.)*
- **SC-003**: A dependent item (002) runs to `verified` with no configuration change after 001.
  *(Met 2026-11-18: run 20261118T171710Z, first attempt; one planning fix first, stockroom#11,
  declaring the 001 test that 002 changes.)* Item 003 followed with no planning fix (run
  20261118T210446Z, first attempt, 8 minutes from mode approval to verified).
- **SC-004**: A Node.js scaffold runs to `verified`.
- **SC-005**: Each recorded run's audit passes 11/11 after `down`.

## Assumptions

- The owner creates repositories; the controller drafts plans and specs, and the owner merges
  them (Q13).
- Hidden tests come from the owner or the test author (018) and are merged by the owner.
- Features that need a live model are out of scope (roadmap C-16).
- Generating constitutions, roadmaps or specs as a product, and project templates, are out of
  scope (roadmap 017 scope out).
