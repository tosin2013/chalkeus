# Feature Specification: Step mode, the laptop manager loop

**Feature Branch**: `021-step-mode`

**Created**: 2026-11-19

**Status**: Draft

**Input**: Roadmap entry 021 (amended 2026-11-19, roadmap 1.24.0). The owner can step away and
leave the pilot running: it carries each proposed item through every automatic step and stops
at each human decision. The owner's decisions:
- **a manager loop:** like an operator's reconcile loop, it delegates to the existing loops;
- **a label proposes an item:** honoured only when an approver added it, found by polling;
- **no standing approval yet:** every run still waits for `/approve mode`;
- **the laptop release (013) ships it.**

## Context

Today an item goes through these steps, and someone types each one:
1. merge the spec and tasks;
2. draft hidden tests (by hand, or with `pilot author watch` since 024) and merge them;
3. run `doctor --baseline`;
4. run `run`, approve the mode, merge the item PR, then merge the status PR.

Steps 3 and 4 always look the same. Only the approvals and merges are human decisions.

Step mode is one **manager** process per target. Each pass it reads where every proposed item
stands, from the roadmap, GitHub and the run evidence, and takes the next automatic step:
- start the test author's drafting (024);
- run `doctor --baseline`;
- start the run.

Then it waits. It never approves, merges or labels anything. Because its state is what GitHub
and the evidence already say, it can be stopped and restarted at any time.

Terms:
- **Owner:** the human who plans items, approves runs and merges PRs.
- **Proposal label:** the label that proposes an item, on the item's GitHub issue (004 mirrors
  roadmap items as issues). Its name follows the product name (roadmap Q7). Until then it is
  the profile setting `PILOT_PROPOSE_LABEL`, default "pilot/go".
- **Approver:** a GitHub login in `PILOT_APPROVERS`, the same people whose `/approve` counts
  (004).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - A proposed item is carried to verified, stopping only for the owner (Priority: P1)

As the owner, I label an item's issue, and the pilot does every automatic step up to each of my
decisions. I answer on GitHub: merge the tests, `/approve mode`, merge the item PR, merge the
status PR. I type no pilot command.

**Why this priority**: This is the feature: leave the pilot running and answer from GitHub.

**Independent Test**: With the manager running for a target, label an item whose hidden tests
are merged. Without typing a pilot command, the item reaches `verified` through the owner's
GitHub answers, and its audit passes.

**Acceptance Scenarios**:

1. **Given** a labelled item whose hidden tests are merged, **When** the manager polls,
   **Then** it runs `doctor --baseline`, posts the result to the item's issue and, if clean,
   starts the run. The run pauses for `/approve mode` as today.
2. **Given** that run, **When** the owner approves the mode and later merges the item PR and
   the status PR, **Then** the run proceeds through its own watcher as today. The manager
   starts nothing else for that item while its run is in progress.
3. **Given** the run succeeded, **When** the manager polls, **Then** it moves to the next
   labelled item, in roadmap order.
4. **Given** a labelled item without hidden tests on a target with auto-draft on (024),
   **When** the manager polls, **Then** it starts one drafting pass for that item. After the
   owner merges the tests PR, it continues with scenario 1.

---

### User Story 2 - Only an approver's label proposes work (Priority: P1)

As the owner, only a label that I (or another approver) added starts anything.

**Why this priority**: A label is the only trigger. Anyone who can label issues mustn't be able
to start work.

**Independent Test**: A non-approver labels an item, and nothing starts. An approver labels it,
and the manager proceeds. The label is removed before the run starts, and nothing starts.

**Acceptance Scenarios**:

1. **Given** the label was last added by a login not in `PILOT_APPROVERS`, **When** the manager
   polls, **Then** the item isn't proposed. The manager prints why, once, and starts nothing.
2. **Given** the label was added by an approver, **When** the manager polls, **Then** the item
   is proposed.
3. **Given** the label is removed before the run starts, **When** the manager polls, **Then**
   the item is no longer proposed. A run that already started isn't stopped: the owner still
   decides at the mode gate.

---

### User Story 3 - Stop, restart, and failures (Priority: P2)

As the owner, I can stop the manager and restart it later, and it picks up where things stand.
A failure stops only that item, and says what to fix.

**Why this priority**: A long-running process will be interrupted, and a failing item mustn't
spin.

**Independent Test**:
- Stop the manager while a run waits for a merge; start it again; it starts nothing new for
  that item and continues once the run ends.
- Make `doctor` fail for an item; it posts the findings once and waits.

**Acceptance Scenarios**:

1. **Given** a run in progress for an item, **When** the manager restarts, **Then** it finds the
   run from the evidence and waits for it. It starts no second run.
2. **Given** `doctor --baseline` fails for an item, **When** the manager polls, **Then** it posts
   the findings to the item's issue once and doesn't retry. It retries once the target's branch
   or the companion changes.
3. **Given** a run that failed (for example, its retries were declined), **When** the manager
   polls, **Then** it posts that, leaves the item alone until the owner relabels it, and moves
   to the next labelled item.

---

### Edge Cases

- **Several labelled items:** one is in flight at a time per target, in roadmap order. A
  blocked item (unsatisfied dependency) is skipped until its dependency is verified.
- **Another run on the same target**, started by hand: the manager waits until it ends.
- **The cluster is down:** the manager reports it and starts nothing. Reading GitHub and the
  roadmap needs no cluster.
- **The driving session:** a coding agent driving runs never starts the manager. It may start
  drafting (024), so the owner starts it in their own terminal, like `author watch`.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: `pilot step --target <name>` MUST run the manager loop in the foreground, polling
  every `PILOT_STEP_INTERVAL_S` seconds (default 120). `--once` MUST do one pass and exit. It is
  off unless started: no profile setting starts it.
- **FR-002**: An item MUST be proposed only when its issue carries `PILOT_PROPOSE_LABEL` and the
  most recent time that label was added, by the issue's timeline, was by an approver.
- **FR-003**: Each pass MUST consider proposed items that are active and startable, with
  satisfied dependencies (`spec status`), in roadmap order. At most one item per target is in
  flight.
- **FR-004**: For the first such item, the manager MUST take exactly one next step:
  - **needs tests, auto-draft on:** start a drafting pass (024);
  - **needs tests, auto-draft off:** post once that hidden tests are needed;
  - **tests PR open:** wait;
  - **tests merged, no run yet:** run `doctor --baseline`. On failure, post the findings once.
    On success, start the run (as `pilot run --target --item`), which pauses for
    `/approve mode` and starts its own watcher;
  - **run in progress:** wait;
  - **run succeeded:** go to the next item;
  - **run failed:** post once and leave the item.
- **FR-005**: The manager MUST NOT approve, merge, label, close or edit anything. It posts only
  comments on the item's issue, with the pilot's identity, and starts only what FR-004 lists.
- **FR-006**: The manager MUST derive its state from the roadmap, GitHub and the run and draft
  evidence. It MUST record each decision (item, step, reason) in
  `<evidence root>/step/<target>/decisions.jsonl`, with no secrets and no test content.
- **FR-007**: Runs started by the manager MUST be recorded as started by step mode (an event in
  the run's evidence). Every guarantee and the audit MUST be unchanged.
- **FR-008**: The pilot skill and the operator docs MUST say the owner starts the manager in
  their own terminal. The driving session never does.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: On Stockroom, a labelled item with merged hidden tests goes to `verified` with no
  pilot command typed after starting the manager. Only the owner's GitHub answers are needed,
  and the run's audit is 11/11 after `down`.
- **SC-002**: A label added by a non-approver starts nothing, in 100% of passes.
- **SC-003**: Restarting the manager at any point never starts a second run, or a second draft,
  for the same item.
- **SC-004**: A failing `doctor` produces exactly one comment and no run, until the target or
  the companion changes.

## Assumptions

- GitHub issues are on for the target (`PILOT_GITHUB_ISSUES=on`), since items are labelled on
  their issues. The manager refuses to start without them.
- The run loop's GitHub approvals are on (`PILOT_GITHUB_APPROVALS=on`), so the owner answers
  every gate from GitHub.
- The standing mode approval, the operator's version (011) and signed webhooks (010) are out
  of scope.
