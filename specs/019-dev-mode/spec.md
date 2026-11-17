# Feature Specification: Developer mode

**Feature Branch**: `018-test-author` (specced together with 018)

**Created**: 2026-11-17

**Status**: Draft

**Input**: Owner, 2026-11-17: "Ok let have a devmode option that is a flag to the project when
developing we can create a spec for that" (roadmap entry 019, decision C-15).

## Context

specfoundry is in active development. Some of the pilot's steps exist only so that a run's
result can be trusted: hidden tests in a private companion repository, a separate session
drafting them (018), approvals on GitHub. When the goal is to exercise the pilot itself, for
example "does the new grader check behave on a real Python project?", those steps slow work down
without protecting anything that matters. Developer mode lets a maintainer skip them, while
making sure such a run can never be mistaken for a real one.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Exercise the pilot end to end without the real-run ceremony (Priority: P1)

As a specfoundry maintainer, I run an item with `--dev`, give it hidden tests from a local
folder, approve in the terminal, and see the whole pipeline (prepare, baseline, attempts,
grading, publish) work, without a private companion repository or a separate test-author
session.

**Why this priority**: It is the reason for the feature.

**Independent Test**: Run `pilot/pilot run --target <t> --item <NNN> --dev` with hidden tests in
a local folder. The run reaches the grader and, on a passing attempt, opens a draft PR titled
`[dev] …`.

**Acceptance Scenarios**:

1. **Given** `--dev` and a local hidden-test folder, **When** the run starts, **Then** it reads the
   hidden tests from that folder instead of the companion repository.
2. **Given** `--dev`, **When** the run reaches a decision, **Then** the terminal approval is
   enough; GitHub approvals are not required.
3. **Given** `--dev`, **When** the same agent session that drives the run also drafted the
   hidden tests, **Then** the run proceeds (outside dev mode this is not allowed, 018).

---

### User Story 2 - A dev run can never be mistaken for a real one (Priority: P1)

As the owner, I need every dev-mode run to be unmistakable and unable to change project
status, so that only real runs count as evidence.

**Why this priority**: Without it, dev mode would undermine the evidence the whole pilot exists
to produce (constitution III).

**Independent Test**: After a dev run, inspect its start message, evidence, PR title and audit:
each says it is a development run. No status-change PR exists and the roadmap is unchanged.

**Acceptance Scenarios**:

1. **Given** a dev run, **When** it starts, **Then** it prints that it is a development run and
   that its result will not count.
2. **Given** a dev run, **When** it opens a PR, **Then** the title starts with `[dev]` and the
   description says it is a development run.
3. **Given** a dev run whose item PR is merged, **When** the run continues, **Then** it never
   proposes a status change and never opens a status-change PR; the roadmap is unchanged.
4. **Given** a dev run's evidence, **When** it is audited, **Then** the audit reports
   "development run: does not count" and the run cannot be cited as verification.
5. **Given** a run started without `--dev`, **When** it is resumed with `--dev` (or the reverse),
   **Then** the resume is refused: a run's mode is fixed when it starts.

---

### User Story 3 - Build-worker isolation is never relaxed (Priority: P1)

As the owner, I need dev mode to exercise the same worker isolation as a real run, so that
testing in dev mode tests the system we ship.

**Why this priority**: It is the condition for dev mode existing at all (C-15).

**Independent Test**: Compare a dev run's isolation checks with a real run's: the same checks
run and pass (no secrets in workers, model-only egress, no hidden tests in worker inputs).

**Acceptance Scenarios**:

1. **Given** a dev run, **When** isolation checks run, **Then** they are the same set as in a
   real run and must pass; a failure fails the run as it would outside dev mode.
2. **Given** a dev run, **When** hidden tests come from a local folder, **Then** they still reach
   only the grader; P11 still checks that no hidden-test file name or hash is in worker inputs.

---

### Edge Cases

- `--dev` with a target profile that has a companion repository: the local folder, if given,
  wins; without one the companion is used, and the run is still a dev run.
- `--dev` without any hidden tests: refused with the usual `ITEM_NO_HIDDEN_TESTS`.
- The fixture run (`pilot/pilot run` with no target) with `--dev`: allowed; same stamps.
- A dev run's PR is merged by mistake: nothing in the roadmap changes; the merge is recorded.
- Dev mode is never read from a profile or an environment variable: only the command-line
  flag turns it on, so it can't be left on by accident.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: `run` MUST accept `--dev`; nothing else turns developer mode on (no profile key,
  no environment variable).
- **FR-002**: A run's mode (dev or real) MUST be fixed at start and recorded; `resume` MUST
  refuse a different mode.
- **FR-003**: In dev mode, hidden tests MAY come from a local folder (`--hidden <dir>`, laid out
  like the companion's item folder); the evidence records the folder's path and a hash of its
  files, never their contents.
- **FR-004**: In dev mode, terminal approvals MUST be sufficient, and the test-author separation
  of 018 (a separate session) MAY be skipped.
- **FR-005**: Dev mode MUST NOT change build-worker isolation: the same isolation checks run and
  must pass.
- **FR-006**: A dev run MUST be stamped in: the start message, the run's evidence, every PR
  title (`[dev]` prefix) and description, and the audit report.
- **FR-007**: A dev run MUST NOT propose, apply or open a PR for any roadmap status change.
- **FR-008**: The audit MUST report dev runs as "development run: does not count", and no command
  may cite a dev run as verification of an item.

### Key Entities

- **Run mode**: `real` or `dev`, fixed at a run's start and recorded in its evidence.
- **Local hidden-test folder**: a directory laid out like `<path>/items/<NNN>/`, used only in dev
  mode; recorded by path and hash.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A maintainer can run an item end to end in dev mode with no private repository, no
  second agent session and no GitHub approvals.
- **SC-002**: 100% of dev runs are labelled in all four places (start, evidence, PR, audit).
- **SC-003**: 0 roadmap status changes ever result from a dev run.
- **SC-004**: A dev run and a real run of the same item run the same isolation checks, with the
  same results.

## Assumptions

- Dev mode is for specfoundry's own development; operators doing real work use a security level
  (minimal, recommended or strict, roadmap C-14), not dev mode.
- PRs from dev runs may still be opened on the target, as drafts, because exercising the
  publisher is part of testing the pilot; a maintainer can close them.
- The first real run of an item (for example `cli-schema-mcp` 006) is never a dev run.
