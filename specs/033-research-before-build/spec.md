# Feature Specification: Research before build

**Feature Branch**: `033-research-before-build`

**Created**: 2026-11-25

**Status**: Draft

**Input**: Roadmap entry 033 (added 2026-11-25, roadmap 1.37.0), from documcp's US1 run
`20261125T163307Z` (friction F15 and F16 in `specs/026-slice-runs/evidence/documcp-onboarding.md`).
Notice when a slice holds work nobody yet knows how to do, or work that needs the environment
changed, and send it back to the humans' planning with the evidence instead of spending attempts
on it. For greenfield and brownfield targets alike.

Governed by C-01 (scheduling inputs), C-04 (human merge only), C-12 (humans plan, agents build)
and C-16 (deterministic grading). Depends on 026 (slice runs) and 030 (resource budgets).

## Context

What happened on documcp (2026-11-25): US1's attempt 1 used its whole 2,400 s deadline (raised
by 030 from 1,200 s) and 277k tokens. Its log ended inside Jest's force-exit code: it was hunting
leaked handles, task T003 "locate the suites that leave handles open". It changed 8 files, one
of them `package-lock.json`, outside US1's allowed paths. The retry question said only that the
attempt stopped at its deadline. An earlier run (`20261124T152325Z`) had timed out twice at
1,200 s with nothing kept (F15).

Neither a bigger budget (030) nor one task per attempt (032) bounds this. T003 is research: its
answer isn't known when the task is written, so its cost isn't either, and when a project's own
tests are faulty, the faulty tests are the work. The fix is to resolve the unknown in planning
(C-12: humans do the research, an agent may assist them) and then build known changes. The
troubleshooting page still blames "a slow or overloaded model endpoint" for every deadline.

Terms:
- **Trigger:** a reason to send work back to planning. Two kinds:
  - **unknown work:** a slice task that is research rather than a known change, or a failure
    whose cause Chalkeus doesn't recognise (no catalogued failure code matches);
  - **environment need:** the work needs something an attempt can't change: the runtime or
    toolchain, dependencies or the lock file, native install scripts, egress hosts, or the pod
    budget.
- **Signal:** one fact from a run's evidence or the item's files that points to a trigger, with
  where it came from (for example "proxy refused `github.com` 2 times", "`package-lock.json`
  changed, outside the allowed paths", "task T003 says `locate`").
- **Diagnosis:** the signals for one failed attempt, the triggers they point to, and one
  recommended choice.
- **Choices:** **retry** (the cause looks temporary), **raise the budget** (the work is known but
  larger than the budget), **back to planning** (unknown work or an environment need).
- **Test-run allowance:** how many full runs of the project's test command fit in one attempt:
  the worker deadline divided by the measured time of one baseline test run.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - A failed attempt says why, and what to do next (Priority: P1)

An attempt fails. Before the owner answers the retry question, forge shows a short diagnosis from
the attempt's evidence: how it ended, the signals it found, the trigger they point to, and the
three choices with one recommended. The same diagnosis is in the run's summary and on the
tracking issue. The owner picks retry, raises the budget for a new run, or goes back to planning.

**Why this priority**: today a deadline failure says only "stopped at the deadline", so the
owner retries blind; on documcp that cost two runs and three 20–40 minute attempts. The
diagnosis turns a failed attempt into planning input.

**Independent Test**: replay the kept evidence of documcp run `20261125T163307Z` attempt 1: the
diagnosis names the research task T003, `package-lock.json` changed outside the allowed paths,
and the proxy denials, and recommends back to planning.

**Acceptance Scenarios**:

1. **Given** an attempt that stopped at its deadline in a slice with a research task, **When**
   the retry question is shown, **Then** it names the task and its research wording and
   recommends back to planning.
2. **Given** an attempt whose kept change touches a dependency manifest or lock file outside the
   allowed paths, or whose egress proxy refused hosts, **When** the retry question is shown,
   **Then** it names each file or host and the environment need, and recommends back to
   planning with the owner's planning PR as the route.
3. **Given** an attempt that stopped at its deadline with no unknown-work or environment signal,
   **When** the retry question is shown, **Then** it recommends raising the budget if the
   slice's test-run allowance is low, and otherwise retry.
4. **Given** an attempt that failed with a reason or error code Chalkeus has no catalogue entry
   for, **When** the retry question is shown, **Then** it says the cause isn't recognised, shows
   the error, and recommends back to planning (investigate before retrying).
5. **Given** an attempt that failed grading in the ordinary way (hidden tests or checks failed,
   no other signal), **When** the retry question is shown, **Then** it recommends retry, as
   today.
6. **Given** any diagnosis, **When** the owner reads it, **Then** each signal says where it came
   from (file, log or task line), and no choice is taken for them.

---

### User Story 2 - The owner ends a run to go back to planning (Priority: P1)

The owner reads the diagnosis and decides to go back to planning. They record that decision on
the run, from the terminal (through the harness) or on the tracking issue. The run ends in a
final state that says so, its watcher stops, and the evidence keeps the diagnosis and the
decision. After the planning change is merged, the owner starts a new run of the same slice.

**Why this priority**: today a run paused for a retry can only be retried or left paused
forever, with its watcher still running; "back to planning" must be a recorded decision, not
silence.

**Independent Test**: pause a fixture run at a retry, record "back to planning"; the run's state
is final, the watcher has stopped, the evidence records who decided and through which channel,
and a new run of the same item or slice starts normally.

**Acceptance Scenarios**:

1. **Given** a run paused for a retry, **When** the owner records "back to planning", **Then**
   the run ends in a final state naming that decision, and no further attempt starts.
2. **Given** that decision on the tracking issue by someone allowed to approve, **When** the
   watcher sees it, **Then** it ends the run the same way and stops watching.
3. **Given** that decision from someone not allowed to approve, **When** it is seen, **Then** it
   is ignored, as other approvals are.
4. **Given** a run ended this way, **When** the owner starts a new run of the same item or slice,
   **Then** it isn't refused because of the ended run.
5. **Given** a run ended this way, **When** the audit runs, **Then** it reports the run as ended
   by the owner for planning, not as a failure of forge.

---

### User Story 3 - Doctor warns before a run (Priority: P2)

Before a run, `forge doctor` for an item or slice warns about work that is likely to need
research or an environment change, and about a deadline that allows too few test runs, so the
owner can fix the plan before spending an attempt.

**Why this priority**: catching it before a run is cheaper than after, but the warnings are
heuristics, so they come after the evidence-based diagnosis.

**Independent Test**: run doctor (with the baseline) on documcp item 001 slice US1 as it stood on
2026-11-25: it warns that T003 reads as research, and reports the measured baseline test time
and the resulting test-run allowance.

**Acceptance Scenarios**:

1. **Given** a slice with a task whose wording is research ("locate", "find", "investigate",
   "identify the cause", "figure out", "determine why"), **When** doctor checks the slice,
   **Then** it warns, naming the task and line, and points to resolving it in the plan's
   research before the tasks.
2. **Given** a baseline check, **When** it runs the project's test command, **Then** doctor
   reports how long one run took and the test-run allowance under the target's worker deadline,
   and warns when the allowance is below the threshold.
3. **Given** a slice whose tasks or spec name a runtime version, and a target whose toolchain
   provides a different one, **When** doctor checks the slice, **Then** it warns with both
   versions and points to the owner's planning PR.
4. **Given** any of these warnings, **When** doctor finishes, **Then** they are warnings only:
   the run is never refused because of them.

---

### User Story 4 - The docs explain the triggers and the route back (Priority: P3)

An owner, new project or existing, reads what each trigger means, what the diagnosis shows, and
how to go back to planning: research in the plan, regenerate the tasks, or a planning PR for the
environment, then a new run.

**Why this priority**: the route back runs through Spec Kit and the owner's repository, not
forge; without docs the recommendation is a dead end.

**Independent Test**: from the troubleshooting entry for a deadline, a reader can find the three
choices and the route for each, with a greenfield and a brownfield example.

**Acceptance Scenarios**:

1. **Given** the troubleshooting entry for a deadline, **When** read, **Then** it names the
   configurable worker deadline, the three causes (slow model, work larger than the budget,
   unknown work or an environment need) and the choice for each.
2. **Given** the docs for preparing an item, **When** read, **Then** they say research-style
   tasks belong in the plan's research, and environment changes in the owner's planning PR, for
   both greenfield and brownfield targets.

---

### Edge Cases

- **Several triggers at once** (research task and an environment need): the diagnosis lists
  every signal; the recommendation is back to planning, naming both routes.
- **No baseline time recorded** (doctor never ran with the baseline): the test-run allowance is
  reported as unknown, never guessed; the diagnosis doesn't use it.
- **Research wording in an ordinary task** ("find and replace", "locate the config file and set
  X"): doctor's warning may be a false positive; it's a warning only, and the owner can ignore
  it. The run diagnosis uses research wording only together with a failure.
- **Proxy denials for hosts the attempt didn't need** (a tool probing the network): reported as
  signals, but alone they don't make the recommendation back to planning unless the attempt
  failed in a way tied to the network (install or download errors).
- **Lock file changed inside the allowed paths** (the owner allowed it): not an environment
  signal.
- **The fixture**: its induced first failure (AU11) is diagnosed like any other attempt; the
  diagnosis must not change the fixture's expected decisions or audit results.
- **A run that fails without a retry question** (retries exhausted, spend cap of the run): the
  diagnosis is still written to the summary and evidence.
- **Old runs**: a run started before this feature resumes without a diagnosis and isn't broken
  by its absence.

## Requirements *(mandatory)*

### Functional Requirements

**Diagnosis (US1)**

- **FR-001**: After every failed attempt, forge MUST build a diagnosis from the attempt's kept
  evidence and the slice's tasks, and record it in the run's evidence.
- **FR-002**: The diagnosis MUST check at least these signals, each with its source:
  1. how the attempt ended (deadline, spend cap, grading failure, error) and its usage;
  2. unchecked tasks in the slice whose wording is research;
  3. files changed outside the allowed paths, marking dependency manifests and lock files;
  4. hosts the egress proxy refused, with counts;
  5. prefetch or install failures;
  6. a failure reason or error code with no catalogue entry;
  7. the slice's test-run allowance when a baseline time is recorded.
- **FR-003**: The diagnosis MUST map signals to triggers and recommend exactly one choice:
  back to planning for any unknown-work or environment-need trigger; raise the budget for a
  deadline or spend-cap failure with a test-run allowance below the threshold and no trigger;
  retry otherwise.
- **FR-004**: The recommendation for back to planning MUST name the route: the plan's research
  then the task list (unknown work), the owner's planning PR (environment need), or a new or
  amended roadmap item (when the signals show the slice's scope is wrong).
- **FR-005**: The diagnosis MUST be shown with the retry question, in the run's summary, and in
  the decision comment on the tracking issue when GitHub approvals are on.
- **FR-006**: The diagnosis MUST be deterministic: the same evidence gives the same diagnosis.
  It MUST NOT call a model.
- **FR-007**: Forge MUST NOT act on its own recommendation: no automatic retry, budget change,
  run end, or edit to specs, tasks, the roadmap or the target.

**Back to planning (US2)**

- **FR-008**: The owner MUST be able to answer a run paused for a retry with "back to planning",
  through the same channels and the same approver rules as other decisions (harness-gated
  terminal command, tracking-issue comment).
- **FR-009**: That decision MUST end the run in a final state that names it, stop its watcher,
  and record the decision, who made it and the channel in the evidence.
- **FR-010**: A run ended this way MUST NOT block a new run of the same item or slice.
- **FR-011**: The audit MUST treat a run ended this way as an owner decision, distinct from a
  failure or an exhausted retry limit.

**Doctor (US3)**

- **FR-012**: Doctor MUST warn, for the checked slice (or the item without slices), about each
  unchecked task whose wording is research, naming the task and its line.
- **FR-013**: A baseline check MUST record how long one run of the test command took, and
  doctor MUST report the test-run allowance under the target's worker deadline and warn when it
  is below the threshold.
- **FR-014**: Doctor MUST warn when the slice's tasks or spec name a runtime version the
  target's toolchain doesn't provide.
- **FR-015**: Every check in FR-012 to FR-014 MUST be a warning: none refuses a run.
- **FR-016**: The research wording list and the allowance threshold MUST be documented, and the
  same list MUST be used by doctor and the diagnosis.

**Docs (US4)**

- **FR-017**: The troubleshooting entry for a deadline MUST name the configurable worker
  deadline and the three causes, and link the diagnosis; the "slow model" explanation stays as
  one cause, not the only one.
- **FR-018**: The docs for preparing an item MUST say where research and environment changes
  belong, with a greenfield and a brownfield example, and how to go back to planning from a
  diagnosis.

### Key Entities

- **Signal:** kind, detail, source (file, log or task line).
- **Diagnosis:** attempt, how it ended, signals, triggers, recommended choice, route.
- **Planning decision:** run, decision ("back to planning"), who, channel, time.
- **Baseline timing:** target, item or slice, test-command duration, worker deadline, test-run
  allowance.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: On the kept evidence of documcp run `20261125T163307Z` attempt 1, the diagnosis
  recommends back to planning and names T003, `package-lock.json` and the refused hosts.
- **SC-002**: On the fixture's induced first failure, the diagnosis recommends retry, and the
  fixture's audit results are unchanged.
- **SC-003**: An owner can decide what to do after a failed attempt from the retry question
  alone, without opening the evidence folder, in every scenario of US1.
- **SC-004**: A run ended for planning leaves no running watcher and doesn't block the next run
  of its item or slice.
- **SC-005**: Doctor on documcp item 001 slice US1 as of 2026-11-25 shows the research warning
  for T003 and the test-run allowance; no doctor result changes from pass to fail because of
  this feature.

## Assumptions

- **Allowance threshold:** warn when one attempt affords fewer than 5 full test runs (one to
  check the baseline, a few to iterate, one to confirm). Documented and adjustable later.
- **Research wording:** a fixed, documented list of verbs and phrases matched at the start of a
  task's description (after its ID and markers), case-insensitive. Not a model judgment
  (FR-006).
- **"Catalogued" causes** are the failure reasons and codes documented in
  `docs/troubleshooting.md`; the plan decides how the diagnosis reads that catalogue and keeps
  it in step with the docs.
- **Test-run counts inside an attempt** are reported only when the attempt's evidence records
  them; the diagnosis doesn't estimate them.
- **No research by forge:** forge recommends; the owner (or an agent assisting them, C-12) does
  the research and the planning change.
- **Ending a paused run** for any other reason stays out of scope; "back to planning" is the
  only new decision.
