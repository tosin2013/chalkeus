# Feature Specification: Proven hidden tests

**Feature Branch**: `041-proven-hidden-tests`

**Created**: 2026-11-28

**Status**: Draft

**Input**: Roadmap item 041 (`.specify/memory/roadmap.md`, 1.45.1): a test-author model separate
from the worker's, an author eval set, a reference check, a per-test vote, and a test plan in the
spec, so hidden tests are shown to be passable before they grade any attempt.

## Background

The draft check (`author check`, 018) proves two things: the project's own tests still pass on
the unchanged base, and the drafted tests fail there. It doesn't prove the drafted tests **can
pass** on a correct change. A hidden test that cannot even compile also "fails on the base", so
it passes the check, merges, and then fails every attempt.

That happened on documcp:

- **Item 004, attempt 2** (run 20261128T012908Z): the change matched the spec's research helper
  exactly, compiled, and reproduced all 52 tool schemas of the spec's contract. G1–G4 passed and
  G5 failed. The owner then ran the hidden test file by hand against that change: it never
  started, because the test file itself failed type checking (TS18046–TS18048 and TS2345).
- **Item 001, US2**: two attempts showed the same pattern (G1–G4 pass, G5 fail, output withheld).
- Both drafts were written by the same free model the worker used (`opencode-go/beam-free`).

Outside work points the same way:

- SWE-bench validates every task's tests against a known-good patch: fail-to-pass tests must fail
  before it and pass after it.
- SWE-bench Verified's human review of 1,699 samples flagged 61.1% for unit tests that may mark
  valid solutions incorrect.
- Anthropic's code-review practice votes on each finding with separate agents, and turns past
  incidents into an eval set that every reviewer change is scored against.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Choose the test author's model separately (Priority: P1)

An owner runs the worker on a cheap or free model but wants hidden tests drafted by a stronger
one. A hidden test is drafted once and grades every attempt, so a miss costs the most there. The
owner sets a test-author model in the target profile (or the global configuration). Drafts use it,
attempts keep using the worker's model, and the draft's evidence records which model wrote it.

**Why this priority**: it is the smallest change that lowers the chance of a broken draft today,
and every later story is measured with it.

**Independent Test**: set a test-author model different from the worker's model, let
`author watch` (or `author start`) draft an item, and confirm the author Job used the test-author
model, the next attempt used the worker's model, and both are recorded.

**Acceptance Scenarios**:

1. **Given** a profile with a test-author model and a different worker model, **When** a draft
   runs, **Then** the author Job uses the test-author model and the draft's evidence records it.
2. **Given** the same profile, **When** an attempt runs, **Then** the worker uses the worker's
   model, unchanged.
3. **Given** no test-author model is set, **When** a draft runs, **Then** it uses the worker's
   model as today, and `doctor` says so in one line (informational).
4. **Given** a test-author model that is free while the target has not opted in to free models,
   **When** `doctor` or a draft runs, **Then** it is refused with the same rule that applies to the
   worker's model.
5. **Given** a test-author model whose provider rejects the key, **When** `doctor` runs, **Then**
   the model-key check fails for the author's model by name.

---

### User Story 2 - Score the test author against known cases (Priority: P2)

A maintainer changes the test author's prompt, model or checks and wants to know whether drafts
got better or worse. Chalkeus keeps an eval set of past items, each with its spec, base, reference
change and the known failure. It drafts tests for each case and scores them with the reference
check (User Story 3). A change is kept only if the score does not drop.

**Why this priority**: without a score, a new model or prompt is a guess; the documcp cases are
already known failures, so they are the first cases.

**Independent Test**: run the eval with the current settings and with a deliberately weaker
configuration, and confirm both produce a per-case result and a total, and that the weaker one
scores no higher.

**Acceptance Scenarios**:

1. **Given** the eval set with documcp 004 and 001 US2, **When** the maintainer runs the eval,
   **Then** each case is drafted, checked, and reported as proven, unproven (with the reason:
   doesn't compile or load, passes on the base, fails on the reference, flaky) or error, with a
   total score.
2. **Given** a previous eval result, **When** the eval runs again after a change, **Then** the
   report shows the difference per case and says whether the score dropped.
3. **Given** a case's hidden tests, **When** the eval reports, **Then** it shows test names and
   outcomes to the maintainer only; nothing of the drafted tests is written to committed evidence
   or shown to the session that drives runs.

---

### User Story 3 - Prove drafted tests against a reference change (Priority: P3)

A test author (or `author watch`) finishes a draft. Before the companion PR opens as ready,
Chalkeus runs the drafted tests twice more on the cluster: on the base plus the item's reference
change, where every drafted test must pass, and once per test, where each must fail on the base
and pass on the reference. The reference run is repeated to catch flaky tests. The PR body shows
the result, and readiness blocks a run on hidden tests that were never proven.

**Why this priority**: it catches exactly the documcp failure (tests that cannot pass) before any
attempt is spent, and it is what the eval (User Story 2) scores with.

**Independent Test**: for a prepared item with a reference change, submit (a) a draft with a type
error in the test file, (b) a draft with one test that already passes on the base, (c) a correct
draft; confirm (a) and (b) are refused with the failing test named and (c) is proven.

**Acceptance Scenarios**:

1. **Given** an item with a reference change in the companion, **When** a draft is checked,
   **Then** the check records: the project's tests pass on the base; every drafted test fails on
   the base; every drafted test passes on the base plus the reference, in each repeated run.
2. **Given** a drafted test file that does not compile or load, **When** it is checked, **Then**
   the check fails with "the drafted tests did not run on the reference change" (not "fail on the
   base"), and the PR does not open as ready.
3. **Given** one drafted test that already passes on the base, **When** it is checked, **Then**
   the check fails and names that test.
4. **Given** a drafted test that passes on the reference in one run and fails in another,
   **When** it is checked, **Then** the check fails and calls the test flaky.
5. **Given** a proven draft, **When** its companion PR opens, **Then** the PR body shows "fails
   on the base · passes on the reference (n of n tests, k runs)".
6. **Given** hidden tests merged without a passing reference check for their commit, **When**
   `doctor --item` or a run checks readiness, **Then** it refuses with `HIDDEN_TESTS_UNVERIFIED`
   and says how to fix it (add a reference and re-check, or record `reference: none`).
7. **Given** an item whose owner recorded `reference: none`, **When** readiness runs, **Then** it
   allows the run with a warning that the hidden tests are unproven.
8. **Given** any item, **When** an attempt runs, **Then** the worker has no access to the
   reference change (an isolation check, like the hidden-test check P11, proves it).

---

### User Story 4 - Independent votes on each test (Priority: P4)

A reference proves a test can pass; it doesn't prove the test asks only for what the spec asks.
After the reference check, separate agents with a fresh context (the spec, the test, the
validity scale; not the drafting history) rate each drafted test on SWE-bench Verified's 0–3
validity scale and check the spec's own rules (for example "no message wording"). Tests rated
2 or 3 go back to the author; the votes appear in the PR body for the owner.

**Why this priority**: it catches tests that are too narrow or check things the spec doesn't ask
for, which the reference check can't; it needs User Story 3's proven tests first.

**Independent Test**: submit a proven draft with one test that asserts an exact error message the
spec says not to assert; confirm the vote rates it 2 or 3, names the rule, and sends it back.

**Acceptance Scenarios**:

1. **Given** a proven draft, **When** the vote runs, **Then** each test gets a rating from 0 to 3
   from each voter, with a one-line reason, and the result is the agreed rating.
2. **Given** a test rated 2 or 3, **When** the vote finishes, **Then** the draft goes back to the
   author with the rating and reason, and the PR does not open as ready.
3. **Given** a test that breaks a rule the spec states for its tests, **When** it is voted on,
   **Then** it is rated at least 2 with that rule named.
4. **Given** voters that disagree, **When** the vote finishes, **Then** the PR body shows every
   vote and the owner decides.

---

### User Story 5 - A test plan in the spec (Priority: P5)

The planner writes, for each acceptance criterion, what its hidden test asserts and how: how the
software is started, where it stores data, and how results are compared (for example by issue
code and path, never by message wording). The test author drafts from that plan, the worker reads
the same plan, and the owner reviews the plan instead of test code.

**Why this priority**: it removes guesswork from drafting and makes the vote's rules explicit; it
changes how specs are written, so it comes after the checks that measure its effect.

**Independent Test**: prepare an item whose spec has a test plan with one criterion missing a
plan line; confirm `doctor --item` and the author flag that criterion, and that a draft follows
the plan lines that exist.

**Acceptance Scenarios**:

1. **Given** a spec with a test plan, **When** the author drafts, **Then** the author's input
   includes the plan and the mapping file links each test to its criterion and plan line.
2. **Given** a criterion without a plan line, **When** `doctor --item` runs, **Then** it warns
   and names the criterion.
3. **Given** a spec without any test plan, **When** `doctor --item` runs, **Then** it reports
   that drafting has no plan to follow (informational, not blocking).

### Edge Cases

- **The reference change is wrong or stale.** If the project's own tests fail on base plus
  reference, the check stops with "the reference change breaks the project's tests" and blames
  the reference, not the draft.
- **The reference doesn't apply** to the item's base (the base moved): the check stops and asks
  for a refreshed reference.
- **A sliced item.** Each slice has its own reference, applied on top of the earlier slices'
  merged changes; earlier slices' tests are re-run as a regression check.
- **A dependency item (029).** The reference applies on top of the approved manifest change, the
  same way the worker's change does.
- **The drafted tests take a long time.** The repeated reference runs count against the grader's
  deadline; the number of repeats is reduced, never below 2, and the record says how many ran.
- **No reference and no `reference: none`.** The draft can be checked and submitted as today, but
  readiness blocks the run with `HIDDEN_TESTS_UNVERIFIED`.
- **Hidden-test output.** Only the owner sees drafted-test output (as today, printed, not stored).
  The PR body and evidence carry counts, test names, ratings and hashes, never test code or
  failure lines (C-14, 035).

## Requirements *(mandatory)*

### Functional Requirements

**Test-author model (US1)**

- **FR-001**: The target profile and global configuration MUST accept a test-author model setting
  separate from the worker's model; drafts MUST use it when set, and the worker's model otherwise.
- **FR-002**: The test-author model MUST follow the same rules as the worker's model: the free-
  model opt-in, the model-key check in `doctor`, the spend and token caps, and egress only to the
  configured model host.
- **FR-003**: Each draft's evidence MUST record the model that wrote it; the companion PR body MUST
  name it.

**Author eval set (US2)**

- **FR-004**: Chalkeus MUST provide an eval set of cases, each holding a spec, a base, a reference
  change and the failure that made it a case; the first cases MUST be documcp 004 and 001 US2.
- **FR-005**: An eval run MUST draft each case with the current author settings, check it with the
  reference check, and report per case (proven, or unproven with its reason, or error) and a total.
- **FR-006**: An eval run MUST compare with a stored previous result and say whether the score
  dropped; the result MUST record the author's model, prompt version and check version.
- **FR-007**: Eval cases and results MUST live where the session that drives runs cannot read the
  drafted tests (the private companion or the maintainer's machine), and nothing of the drafted
  tests MUST reach committed evidence.

**Reference check (US3)**

- **FR-008**: An item (or slice) MAY have a reference change stored in the private companion next
  to its hidden tests; the owner MAY instead record `reference: none` for the item.
- **FR-009**: The draft check MUST, in addition to today's checks, run the drafted tests on the
  base plus the reference change and require every drafted test to pass, and require each drafted
  test to fail on the base and pass on the reference on its own.
- **FR-010**: The reference run MUST be repeated at least twice (default 3); a test whose outcome
  differs between runs MUST fail the check as flaky.
- **FR-011**: A drafted test file that does not compile or load MUST fail the check with its own
  reason, distinct from "the drafted tests fail on the base".
- **FR-012**: When the project's own tests fail on base plus reference, or the reference does not
  apply, the check MUST stop and blame the reference.
- **FR-013**: The companion PR body MUST show the reference-check result (tests passing on the
  reference, tests failing on the base, runs) or state `reference: none`.
- **FR-014**: Readiness MUST refuse a run with `HIDDEN_TESTS_UNVERIFIED` unless the merged hidden
  tests' commit has a passing reference check or the owner recorded `reference: none` (then a
  warning).
- **FR-015**: The reference change MUST never reach the worker; an isolation check MUST prove it
  for every attempt, and the audit MUST include it.

**Per-test vote (US4)**

- **FR-016**: After a passing reference check, each drafted test MUST be rated 0–3 by at least two
  independent voters given only the spec, the test plan (if any), the test and the scale.
- **FR-017**: A test rated 2 or 3 by the agreed result MUST send the draft back to the author with
  the rating and reason; disagreement MUST be shown to the owner, who decides.
- **FR-018**: Voters MUST check the rules the spec states for its tests (for example "no message
  wording") and name a broken rule in the reason.
- **FR-019**: Votes MUST appear in the PR body (rating and one-line reason per test, no test code)
  and in the draft's evidence.

**Test plan (US5)**

- **FR-020**: A spec MAY contain a test plan that maps each acceptance criterion to what its
  hidden test asserts and how (start-up, storage, comparison style).
- **FR-021**: The author's input MUST include the test plan when present, and the mapping file
  MUST link each test to its criterion and plan line.
- **FR-022**: `doctor --item` MUST warn about criteria without a plan line, and report
  (informational) when a spec has no test plan.

**Across stories**

- **FR-023**: Nothing in this feature MUST give the session that drives runs access to hidden
  tests, drafted tests, their output or the reference change's test results beyond counts, names,
  ratings and hashes (C-14).
- **FR-024**: Every new decision (accepting `reference: none`, overriding a disagreeing vote) MUST
  stay the owner's (C-12); Chalkeus recommends.

### Key Entities

- **Test-author model**: the model setting drafts use; recorded per draft.
- **Reference change**: a known-good change for an item or slice, kept in the private companion;
  or the owner's `reference: none`.
- **Reference-check record**: per draft commit: base result, reference results per run, per-test
  outcomes, flaky tests, the reason for a failure.
- **Vote**: per test and voter: rating 0–3, reason, rules named.
- **Test plan**: per acceptance criterion: what is asserted and how.
- **Eval case / eval result**: a past item with its reference and known failure; a scored run of
  the author over all cases, with model, prompt and check versions.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: documcp 004's merged draft (the one that does not compile), checked again with its
  reference (004 attempt 2's change), is refused before merge with the "did not run on the
  reference" reason.
- **SC-002**: After this feature, no attempt fails G5 because a hidden test cannot pass on a
  correct change: zero cases in the next 10 hidden-test drafts across targets, counted from run
  evidence.
- **SC-003**: The owner can see, in the companion PR alone, which model wrote the tests, whether
  they were proven against a reference, and each test's rating, without opening test code.
- **SC-004**: An eval run over the starting cases finishes within one working session (under an
  hour on the operator's cluster) and gives the same score twice in a row for unchanged settings.
- **SC-005**: Switching the test author to a stronger model raises the eval score or keeps it
  equal; the result is recorded with both models named.
- **SC-006**: The worker never has the reference change: the isolation check passes in 100% of
  attempts and the audit reports it.

## Assumptions

- References are written by the planner, the owner, or taken from a branch; Chalkeus does not
  write them. For documcp 004, attempt 2's change (run 20261128T012908Z) is the first reference,
  since it matched the spec and its contract.
- Voters are agent Jobs like the author's (isolated, egress only to the model host), using the
  test-author model unless configured otherwise; the voting scale is SWE-bench Verified's.
- Repeating the reference run three times fits within the grader's deadline for current targets;
  the deadline can be raised in the profile (`FORGE_GRADER_DEADLINE_S`).
- Hidden-test failure output shown to the owner is 040's work; this feature only uses counts and
  names.
- Mutation testing and patch-coverage scores are out of scope (roadmap 041 scope out).
- Depends on 014 (hidden tests, isolation) and 024 (test-author automation); governed by C-12 and
  C-14.
