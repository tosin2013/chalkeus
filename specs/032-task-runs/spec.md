# Feature Specification: Task runs within a slice

**Feature Branch**: `032-task-runs`

**Created**: 2026-11-24

**Status**: Draft

**Input**: Owner proposal (2026-11-24) after documcp's US1 slice run `20261124T152325Z`: both
attempts hit the worker's 1,200-second deadline and left nothing behind (friction F15 in
`specs/026-slice-runs/evidence/documcp-onboarding.md`). "We do not have to do the US all at once;
it could be split by task, but it needs to be automated, like `forge step`. The argument is also
token usage." A slice is built one task (or a small group of tasks) per attempt, on one slice
branch, automatically, and graded in full once at the slice's end; the owner still reviews one
PR per slice.

Governed by C-01 (scheduling inputs), C-04 (human merge only), C-12 (humans plan, agents build),
C-14 (hidden tests out of every build worker's reach) and C-16 (deterministic grading). Builds on
026 (slice runs) and 021 (step mode).

## Context

Today one attempt builds a whole slice. A large slice (documcp US1: declare Node 24, find and
close leaked handles, fix two flaky tests, stop a test writing into the repository, update the
docs) doesn't fit one attempt's time, and an attempt that times out loses all its work. Each
retry starts from nothing and spends the slice's whole context again.

A slice's tasks are already small, human-approved units in `tasks.md`. Running them one at a
time keeps each attempt short and its context small, and keeps finished tasks when a later one
fails.

What doesn't change: the slice is still what is graded with hidden tests, published as one draft
PR and merged by a human; the item is still what is verified.

Terms:
- **Task step:** one attempt that builds the next unticked task (or the next group of tasks) of
  the slice, on the slice branch.
- **Slice branch:** the branch a task-run slice is built on; each passed task step adds its
  commit there.
- **Step check:** the grade of a task step (FR-006).
- **Slice check:** the full grade of the slice after its last task step (026's grading).
- **Handoff note:** a short note a worker leaves for the next task step: what it found and what
  it left undone.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Build a slice task by task, review one PR (Priority: P1)

The owner turns on task runs for a target and runs a slice. After one mode approval, forge builds
the slice's tasks in `tasks.md` order, one task step at a time, each on the same slice branch.
When the last task passes, forge runs the full slice check and opens one draft PR for the slice.

**Why this priority**: it is the whole feature; it lets a slice too big for one attempt be built
at all.

**Independent Test**: on the fixture with task runs on, run a two-task slice: two task steps run,
each adds one commit and ticks its task; the slice check passes; one draft PR opens whose commits
are the two task steps.

**Acceptance Scenarios**:

1. **Given** a target with task runs on and a ready slice with N unticked tasks, **When** the
   owner runs the slice and approves the mode once, **Then** forge runs task steps in `tasks.md`
   order without asking again while each step check passes.
2. **Given** a task step whose step check passes, **When** the next step starts, **Then** it
   starts from the slice branch with every earlier task's commit.
3. **Given** the slice's last task step passes, **When** forge continues, **Then** it runs the
   slice check and, on pass, opens one draft PR for the slice (026's title and rules).
4. **Given** a target with task runs off (the default), **When** a slice runs, **Then** it runs as
   one attempt, exactly as today.

---

### User Story 2 - A failed task costs only that task (Priority: P1)

When a task step fails or runs out of time, forge keeps every passed task, asks the owner whether
to retry that task, and retries it alone.

**Why this priority**: it is what makes the split worth it: two documcp timeouts lost all their
work; with task steps, only the failing task is redone.

**Independent Test**: on the fixture, make the second of three task steps fail once: the first
task's commit stays; forge pauses for a retry of task 2 only; on approval it retries task 2, then
runs task 3.

**Acceptance Scenarios**:

1. **Given** a task step that fails its step check or its deadline, **When** forge stops it,
   **Then** the slice branch keeps every earlier passed task and the run pauses for the owner's
   retry decision for that task.
2. **Given** the owner approves the retry, **When** it runs, **Then** only that task is attempted
   again, and the retry counts against that task's retry limit.
3. **Given** the owner declines, or the task's retries are used up, **When** the run stops,
   **Then** the evidence lists the passed tasks and the failed one, and the slice branch is kept
   for a later run.
4. **Given** a later run of the same slice, **When** it starts, **Then** it continues from the
   first task not yet passed on the kept slice branch, after the gate and baseline checks pass.

---

### User Story 3 - Hidden tests at the slice's end (Priority: P1)

Task steps aren't graded with the slice's hidden tests (those fail until the whole slice is
done). After the last task, the slice check grades everything as 026 does. If it fails, forge
asks the owner before one fix-up step that sees the failure summary and the whole slice.

**Why this priority**: it keeps 026's guarantee (a slice PR passes its hidden tests) while
letting intermediate steps be graded on what they can pass.

**Independent Test**: on the fixture, a slice whose tasks pass their step checks but whose slice
check fails: forge pauses; on approval, one fix-up step runs; the slice check runs again.

**Acceptance Scenarios**:

1. **Given** a task step, **When** it is graded, **Then** the hidden tests of this slice are not
   run, and nothing about them reaches the worker (C-14).
2. **Given** the last task passes, **When** the slice check runs, **Then** it runs this slice's
   hidden tests, every earlier slice's hidden tests and the project's tests, with 026's G2, G3,
   G6 and G7 rules over the whole slice change.
3. **Given** a failed slice check, **When** the owner approves a fix-up, **Then** one fix-up step
   runs with the whole slice's tasks and the same failure feedback 026 gives a retry, and the
   slice check runs again.

---

### User Story 4 - See the plan, the progress and the spend per task (Priority: P2)

The mode approval shows the tasks that will run and the budget; the run reports each task step's
result, time, tokens and cost; the evidence keeps them per task.

**Why this priority**: the owner approves the run knowingly and can see where time and tokens
went, which is the token argument for this feature.

**Independent Test**: approve a task-run slice: the approval lists the task ids in order; after
the run, the summary has one row per task step with grade, seconds, tokens and cost.

**Acceptance Scenarios**:

1. **Given** a task-run slice, **When** the mode approval is shown, **Then** it lists the task
   ids and their order, tasks per step, the retry limit per task and the spend and token caps
   for the slice as a whole.
2. **Given** a running slice, **When** each task step ends, **Then** forge reports which task
   passed or failed and how long it took, on the terminal and in the run's GitHub issue.
3. **Given** a finished or stopped run, **When** its evidence is read, **Then** each task step
   has its grade, time, tokens and cost, and the slice totals are stated.
4. **Given** the slice's spend or token cap is reached between task steps, **When** forge checks
   it, **Then** the run stops before the next step, with the passed tasks kept.

---

### User Story 5 - Tasks that don't change files, and handoff between steps (Priority: P2)

Some tasks investigate or verify rather than change files (documcp T003 "locate the suites that
leave handles open", T007 "run the suite three times"). Such a task passes by ticking it and
leaving a handoff note; the next task step gets the notes of earlier steps.

**Why this priority**: without it, an investigation task either fails for "no change" or its
findings are lost before the task that needs them.

**Independent Test**: a slice whose first task only investigates: its step passes with only the
tick and a handoff note; the second step's prompt includes that note.

**Acceptance Scenarios**:

1. **Given** a task step that changes no file besides ticking its task, **When** it is graded,
   **Then** it passes if the step check otherwise passes and it left a handoff note.
2. **Given** earlier task steps left handoff notes, **When** the next step starts, **Then** its
   prompt includes them, in order, within a size limit.
3. **Given** a handoff note, **When** the slice's PR opens, **Then** the notes are in the run's
   evidence, not in the target repository.

---

### Edge Cases

- A task marked `[P]` (parallel): task steps still run one at a time, in order; parallel task
  steps are open question Q12 (015).
- A task already ticked on the slice branch when the run starts: it is skipped.
- A task step that ticks another task, or a task of another slice: the step check fails (026
  FR-011 applied per step).
- A task step that changes files outside the slice's allowed paths: the step check fails (G2).
- The base branch moves while the slice is in progress: the slice check runs on the slice branch
  as published; a conflict is the owner's to resolve on the PR, as today.
- A slice with one task: one task step, then the slice check; the same as a slice run plus one
  check.
- Step mode (021): with task runs on, step mode runs each slice task by task and pauses only at
  the decisions above (mode per slice, retries, fix-ups, PR merge).
- Grouping: with more than one task per step, a group never crosses the slice's end; a group's
  step ticks exactly the group's tasks.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: A target's profile MUST be able to turn task runs on; off by default. With them
  off, slice runs MUST behave exactly as in 026.
- **FR-002**: With task runs on, a slice MUST be built by task steps, each building the next
  unticked task, or the next group of up to K unticked tasks, of the slice in `tasks.md` order.
  K MUST default to 1 and MUST be part of the approved mode.
- **FR-003**: The tasks and their order MUST come from the slice's approved `tasks.md` (C-01);
  forge MUST NOT split, merge, reorder or add tasks.
- **FR-004**: Every task step MUST start from the slice branch with every earlier passed step's
  commit; each passed step MUST add its own commit.
- **FR-005**: One mode approval MUST cover all the slice's task steps; the spend and token caps
  MUST apply to the slice as a whole.
- **FR-006**: A step check MUST pass only if: the project's test command passes; the change
  stays within the slice's allowed paths; exactly the step's tasks are newly ticked; and the
  change is either a file change or a tick with a handoff note. It MUST NOT run any hidden
  tests.
- **FR-007**: A failed or timed-out task step MUST keep every earlier passed step, and MUST
  pause for the owner's retry decision for that task; a retry MUST attempt only that task (or
  group) and count against its own retry limit.
- **FR-008**: A stopped slice MUST keep its slice branch, and a later run of the same slice MUST
  continue from the first task not passed on it, after the gate and baseline pass.
- **FR-009**: After the last task step, the slice check MUST run 026's full grading over the
  whole slice change; only a passing slice check opens the slice's draft PR.
- **FR-010**: A failed slice check MUST pause for the owner's decision on one fix-up step; the
  fix-up step MUST see the whole slice's tasks and the slice check's failure feedback, never the
  hidden tests' contents (C-14).
- **FR-011**: The mode approval MUST list the task ids in order, tasks per step, the retry
  limit per task and the slice caps.
- **FR-012**: Each task step's grade, duration, tokens and cost MUST be recorded in the run's
  evidence and reported on the terminal and in the run's GitHub issue; the summary MUST give
  slice totals.
- **FR-013**: When a slice cap is reached between task steps, the run MUST stop before the next
  step, keeping passed steps.
- **FR-014**: Each task step MUST be able to leave a handoff note; the notes of earlier steps
  MUST be given to later steps, in order, within a size limit; notes MUST be kept in the run's
  evidence and MUST NOT be committed to the target.
- **FR-015**: The item's audit MUST cover task steps: every step's commit is in the slice's PR,
  every ticked task was ticked by its own step, and the slice check passed before the PR opened.
- **FR-016**: Step mode (021) MUST run task-run slices task by task, pausing only at the mode
  approval, retries, fix-ups and PR merges.
- **FR-017**: The targets docs MUST explain when to turn task runs on, how tasks should be
  written for it (one change per task; investigation tasks leave a note), and what each step
  check and the slice check grade.

### Key Entities

- **Task step:** the task ids it builds, its attempt number, its commit (if any), its step
  check result, duration, tokens, cost and handoff note.
- **Slice branch:** the branch a task-run slice is built on; its commits are the passed steps.
- **Slice check:** 026's grade over the slice change, run once after the last step (and after a
  fix-up).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: documcp's US1 slice, unchanged, completes with task runs on within the catalog
  worker deadline per task step, and opens one draft PR that passes its hidden tests.
- **SC-002**: A failed task step loses only its own work: 100% of earlier passed steps are on the
  slice branch afterwards.
- **SC-003**: A retry after a failed task step uses fewer tokens than retrying the whole slice
  would; the evidence shows tokens per step so the owner can compare.
- **SC-004**: The owner gives exactly one mode approval per slice when every step passes, and
  reviews exactly one PR per slice.
- **SC-005**: Every task step's grade, time, tokens and cost is in the evidence; the audit
  passes for a task-run slice and for a slice run without task runs.

## Assumptions

- Tasks in an approved `tasks.md` are small enough for one attempt; a task that isn't is the
  owner's to split when preparing the item (C-12).
- One task per step is the default; grouping (K > 1) is the owner's choice in the mode, for
  slices with many tiny tasks.
- The slice branch lives on the target under forge's publisher identity, as a slice's branch
  does today; the draft PR opens only after the slice check passes.
- Collecting a timed-out worker's diff and usage is 030's; with task steps a timeout loses at
  most one step.
- Running task steps in parallel is out of scope (Q12, 015).
