# Feature Specification: Slice runs

**Feature Branch**: `026-slice-runs`

**Created**: 2026-11-21

**Status**: Draft

**Input**: Roadmap entry 026 (added 2026-11-21, roadmap 1.27.0), from GitHub issue #142. A run
may build one **slice** of a roadmap item, one phase of its approved `tasks.md`, instead of the
whole item. The roadmap item stays the unit that is verified and that unblocks dependents. The
owner's decisions:
- **one model, not two modes:** by default one slice is the whole item, so today's behaviour is
  unchanged;
- **humans choose the slices:** the human approving `tasks.md` decides where it splits; the
  pilot and agents never re-slice;
- **tick state from spec-status:** per-slice task counts come from spec-status (a release that
  reports per-phase counts), so the scheduling inputs (C-01) don't change;
- **the laptop release (013) ships it.**

## Clarifications

### Session 2026-11-21

- Q: How do phases that aren't user stories (Setup, Foundational, Polish) map to slices? → A:
  They join story slices: earlier phases join the first story's slice, Polish joins the last.
  Slice ids stay `US1`, `US2`, … (FR-003).
- Q: How does `tasks.md` mark slices independent? → A: Not in this version; slices are always
  sequential (FR-007).

## Context

Today one run builds a whole item, grades it, opens one draft PR and, after the merge,
proposes the item `verified` (014). That combines two different things:
- **the unit of done** is the roadmap item: it has the outcome, the dependents and the status;
- **the unit of risk and review** is usually smaller: one user story or phase.

For larger items this hurts:
- the diff is too big to review well before merging;
- one failed attempt throws away the work on every story, and retries cost more;
- high-risk work (a migration, auth, deletes) shares a run and a PR with low-risk work;
- projects that review risky work one story at a time must split each story into its own
  roadmap item, which puts execution detail in the roadmap and changes what "done" means for
  dependents.

Terms:
- **Slice:** one phase of an item's approved `tasks.md`, identified by its **slice id**. An item
  whose tasks aren't split has one slice, the whole item.
- **Open slice:** a slice whose PR hasn't been merged.
- **Owner:** the human who plans items, approves runs and merges PRs.
- **Item-level acceptance check:** an optional check the item's spec declares (for example a
  human live check), recorded by a human before `verified` may be proposed.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - An item without slices runs exactly as today (Priority: P1)

As the owner, I run an item without naming a slice, and nothing changes: one run, one PR, one
`verified` proposal after the merge, and the same audit.

**Why this priority**: Every existing target, fixture and evidence record depends on today's
behaviour. Slices must not break it.

**Independent Test**: Run the fixture and an existing target item without a slice. Results,
evidence and audit match today's, and every committed evidence record still audits the same.

**Acceptance Scenarios**:

1. **Given** a prepared item, **When** the owner runs it without naming a slice, **Then** the
   whole item is built, graded and published in one run, as today.
2. **Given** the committed evidence records (the golden data), **When** they are audited,
   **Then** every result is unchanged.

---

### User Story 2 - Build and merge an item one slice at a time (Priority: P1)

As the owner, I run an item's first slice, review and merge its PR, then run the next slice.
The item is proposed `verified` once, after its last slice merges.

**Why this priority**: This is the feature: smaller PRs and cheaper retries, with "done" still
meaning the whole item.

**Independent Test**: On a two-slice item, run slice 1, merge it, run slice 2, merge it. Exactly
one `verified` proposal appears, after the second merge.

**Acceptance Scenarios**:

1. **Given** an item with two slices and hidden tests for both, **When** the owner runs slice 1,
   **Then** it is gated, graded and published as one draft PR titled with the slice (for example
   `feat(NNN): <title> (US1)`), and no `verified` proposal is made after it merges.
2. **Given** slice 1 is not merged, **When** the owner runs slice 2, **Then** the run is refused
   before anything starts, with a reason naming slice 1.
3. **Given** slice 1 is merged, **When** slice 2 is graded, **Then** the grader also runs slice
   1's hidden tests, and slice 2 fails if they fail.
4. **Given** slice 2 is merged, **When** the pilot checks the item, **Then** it proposes
   `verified` exactly once, through the same channel as today (status-change PR or terminal
   approval).
5. **Given** the item's spec declares an item-level acceptance check, **When** the last slice
   merges, **Then** `verified` is proposed only after a human records that check.

---

### User Story 3 - A slice stays inside its own boundaries (Priority: P1)

As the owner, I can trust that a slice's PR changes only that slice's work.

**Why this priority**: Slices exist to keep risk separated. A slice that edits beyond itself
defeats the point.

**Independent Test**: Make slice attempts that tick another slice's tasks, and that edit a file
outside the slice's allowed paths. Both fail grading.

**Acceptance Scenarios**:

1. **Given** a slice run, **When** the change ticks a task of another slice, **Then** grading
   fails.
2. **Given** the spec gives allowed paths for the slice, **When** the change edits a file
   outside them, **Then** grading fails. **Given** it gives none for the slice, **Then** the
   item's allowed paths apply.
3. **Given** a slice run, **When** it is graded, **Then** it passes only if every task of this
   slice is ticked.

---

### User Story 4 - One audit for the item across its slices (Priority: P2)

As the owner or a security reviewer, I read one item's audit and see every slice run under it.

**Why this priority**: Evidence is how `verified` is justified. It must cover the item, not one
slice.

**Independent Test**: After a two-slice item is verified, its audit lists both slice runs and
passes.

**Acceptance Scenarios**:

1. **Given** a slice run, **When** its evidence is written, **Then** it records both the item
   and the slice.
2. **Given** an item verified through two slices, **When** it is audited, **Then** the audit
   shows both runs under one item, with one task PR per slice and one status proposal for the
   item.

---

### User Story 5 - Step mode and the test author work per slice (Priority: P2)

As the owner using step mode (021) with auto-draft (024), a proposed item is carried slice by
slice, and the test author drafts each slice's hidden tests.

**Why this priority**: The laptop release ships step mode and auto-draft. Slices must work
with them or the release would have two incompatible ways to run an item.

**Independent Test**: Label a two-slice item with auto-draft on. The test author drafts slice 1,
then the run for slice 1 waits for its own mode approval; after its merge the same happens for
slice 2; then the status proposal follows.

**Acceptance Scenarios**:

1. **Given** a ready slice with no hidden tests and auto-draft on, **When** the test author
   polls, **Then** it drafts that slice's hidden tests in the slice's own companion folder.
2. **Given** a proposed item with two slices, **When** step mode runs it, **Then** slices run in
   order, each waiting for its own `/approve mode`, and the next slice starts only after the
   previous slice's PR is merged.

---

### Edge Cases

- **Re-slicing after a slice merged:** if the approved `tasks.md` phases change after a slice of
  the item merged, the run is refused until a human resolves it; the pilot never re-maps
  slices.
- **A slice id that doesn't exist** in the approved `tasks.md`: refused before anything starts.
- **Running without a slice on an item that already has a merged slice:** refused; the owner
  must run the remaining slices.
- **An open PR for the same slice:** refused, as today for the item. Another slice's open PR
  blocks only slices that must follow it.
- **A slice with no hidden tests:** refused, as today for an item.
- **Hidden tests of the new slice already pass on its base:** refused, as today
  (`BASELINE_HIDDEN_PASSES`).
- **spec-status doesn't report per-phase counts** (an older release): any run naming a slice is
  refused as unevaluable (fails closed); runs without a slice still work.
- **Gaps found after the item is verified:** they never add a slice to a verified item. They
  need a new roadmap item, or an explicit reopen that a human approves.
- **Developer mode (C-15):** slice runs in dev mode are stamped and never propose a status, as
  today.

## Requirements *(mandatory)*

### Functional Requirements

**Slices**

- **FR-001**: An item MUST have one or more slices. An item whose approved `tasks.md` isn't
  split MUST have exactly one slice, the whole item.
- **FR-002**: Slices MUST come from the phase headings of the item's approved `tasks.md`. The
  pilot and agents MUST NOT create, merge or re-order slices.
- **FR-003**: Slice ids MUST be stable and readable: a phase headed "User Story N" has the id
  `USN`. Phases that aren't user stories join a story slice: phases before the first story
  (Setup, Foundational) join the first story's slice, and phases after the last story (Polish)
  join the last story's slice. A non-story phase between two stories joins the story before it.
  An item with no "User Story" phases has one slice, the whole item.
- **FR-004**: Per-slice task counts MUST come from spec-status's JSON report. The pilot MUST NOT
  read `tasks.md` phases itself to decide whether a slice may start or is complete (C-01).

**Running and gating**

- **FR-005**: The owner MUST be able to name a slice when running an item. Running without one
  MUST behave exactly as today for an item with one slice.
- **FR-006**: The gate for a slice MUST refuse (failing closed) unless all of these hold:
  - the item's dependencies are verified;
  - every earlier slice of the item is merged;
  - every task of this slice is unticked;
  - the slice exists in the approved `tasks.md`.
- **FR-007**: Slices MUST run in `tasks.md` order. Marking slices independent, so they can run
  without waiting for earlier ones, is not in this version (see Q12 and 015).
- **FR-008**: An item with an open slice MUST NOT be treated as already verified or already
  implemented by the gate.
- **FR-009**: Readiness MUST refuse a slice when that slice already has an open PR, and when an
  earlier slice that this one must follow has an open PR.

**Allowed paths**

- **FR-010**: When the spec gives allowed paths for a slice (for example
  `## Allowed paths (US2)`), the slice run MUST use them; otherwise it MUST use the item's
  allowed paths. Every existing allowed-paths rule (protected paths, other items' spec folders)
  MUST still apply.

**Grading**

- **FR-011**: Grading a slice MUST fail if the change ticks any task outside this slice, or
  leaves any task of this slice unticked.
- **FR-012**: Grading a slice MUST run this slice's hidden tests, every earlier slice's hidden
  tests as a regression check, and the project's test command. Any failure MUST fail the slice.
- **FR-013**: Hidden tests MUST be kept per slice in the companion (`items/NNN/<slice>/`). An
  item with one slice MUST keep using `items/NNN/` as today.
- **FR-014**: The checks that the change adds tests (G6) and keeps existing tests passing (G7)
  MUST apply to each slice, measured against the base that slice starts from (the merged state
  after the earlier slices).
- **FR-015**: The baseline check MUST apply per slice: this slice's hidden tests must fail on its
  base, and earlier slices' hidden tests must pass there.

**Publishing and status**

- **FR-016**: Each slice run MUST open its own draft PR, merged only by a human (C-04), titled
  with the item and the slice (for example `feat(NNN): <title> (US2)`).
- **FR-017**: The pilot MUST propose `verified` only after the item's last slice merges, and
  exactly once per item. It MUST NOT write any intermediate roadmap status.
- **FR-018**: When the item's spec declares an item-level acceptance check, the pilot MUST NOT
  propose `verified` until a human has recorded that check in the run evidence.
- **FR-019**: A verified item MUST NOT gain new slices. A gap found after verification MUST
  need a new roadmap item or an explicit reopen that a human approves.

**Evidence and audit**

- **FR-020**: Every slice run's evidence MUST record the item and the slice.
- **FR-021**: The item's audit MUST cover all its slice runs: only this item's slices were
  attempted, each slice had one final passing attempt and one task PR, and the item had at most
  one status proposal.
- **FR-022**: Committed evidence records from before this feature MUST audit with unchanged
  results.

**Interactions**

- **FR-023**: Automatic drafting (024) MUST draft hidden tests per slice, and MUST trigger for a
  slice that is ready but has no hidden tests.
- **FR-024**: Step mode (021) MUST run an item's slices in order, wait for each slice's PR merge
  before the next, and require a mode approval for each slice run.
- **FR-025**: Humans MUST still prepare and approve the slicing; the pilot MUST never pick up a
  slice by itself (C-12).

### Key Entities

- **Roadmap item:** the unit of done. It has the outcome, dependencies, dependents and the
  status. Unchanged.
- **Slice:** one phase of the item's approved `tasks.md`. It has an id, its tasks, optional
  allowed paths, hidden tests in its companion folder, at most one open PR, and a merged or open
  state.
- **Slice run:** one pilot run of one slice. Its evidence names the item and the slice.
- **Item-level acceptance check:** optional, declared in the item's spec, recorded by a human.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Every committed evidence record audits with the same results as before, and the
  fixture run passes its full audit, with no slice named.
- **SC-002**: On a two-slice item, slice 2 is refused in 100% of attempts made before slice 1
  merges.
- **SC-003**: On a two-slice item, exactly one `verified` proposal is made, and only after slice
  2 merges.
- **SC-004**: A slice attempt that ticks another slice's tasks, or edits outside its allowed
  paths, fails grading in 100% of cases.
- **SC-005**: Slice 2's grading runs slice 1's hidden tests, shown in its grading evidence
  (summary only, never test contents).
- **SC-006**: The item's audit lists both slice runs under one item and passes.
- **SC-007**: Each slice's PR contains only that slice's work, so a reviewer reviews a smaller
  diff than the whole item's.
- **SC-008**: A two-slice item goes from label to `verified` in step mode with only GitHub
  answers: one mode approval and one merge per slice, then the status merge.

## Assumptions

- spec-status will report per-phase task counts (each phase's heading, line and checked/total)
  in `status --json`. Until it does, slice runs are refused and whole-item runs work. spec-status
  0.3.0, pinned 2026-11-21, reports only item-level counts.
- Spec Kit's `tasks.md` already groups tasks into phases, most of them one user story each.
- One slice is built at a time per item. Running slices or `[P]` tasks of one item in parallel is
  out of scope (Q12, 015).
- 015 (released waves) treats an item's slices as sequential steps inside that item's unit.
- The isolation, credential and hidden-test rules (C-03, C-07, C-08, C-14) apply unchanged to
  every slice run.
