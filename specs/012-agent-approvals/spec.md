# Feature Specification: Working with an agent — pilot skill and harness approvals

**Feature Branch**: `012-agent-approvals`

**Created**: 2026-11-14

**Status**: Draft

**Input**: User description: "012 Working with an agent: pilot skill and harness approvals (roadmap entry 012). A human and an LLM agent run the pilot together: the agent runs commands and explains each pause; the human makes every decision. Without a terminal every decision pauses instead of being assumed; the human approves the one pending step with `pilot approve` (mode, retry, reconcile; never merge), which the agent harness always asks the human about; approvals are bound to the step and used once; AU09/AU10 record each approval's channel (tty or harness); a project pilot skill and AGENTS.md teach agents the rules. A first cut is already merged (PR #11) and was used in run 20261114T213046Z; this spec formalises the audit-contract change and validates it."

## Background

During the 006 fresh-machine validation (finding F4), an agent ran the pilot from a shell
with no terminal, so every human prompt was answered "no". A suggested `--yes` flag was
rejected: the audit (AU09, AU10) attests that a *human* confirmed the mode and each retry, and
an agent saying yes for itself would make that record false (constitution V, roadmap C-04).
The owner decided instead that a human and an agent should be able to work together, with
the harness collecting the human's yes. A first cut merged in PR #11 and was used in run
`20261114T213046Z`. This spec states what that behaviour must guarantee, changes the audit
contract to record how each decision was made, and validates it.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - A human and an agent complete a run together (Priority: P1)

An operator asks their coding agent to run the pilot. The agent runs the commands, and at
each decision (run mode, retry, status change) it explains in plain words what is being
decided and asks the operator. When the operator agrees, the agent proposes the approval
command; the operator's harness asks the operator before it runs; the operator says yes; the
run continues. Merges are done by the operator on GitHub. The run ends with a passing audit
that shows each decision was approved through the harness.

**Why this priority**: this is the reason for the feature: operators increasingly work
through agents, and today they cannot use the pilot that way at all.

**Independent Test**: on a machine set up per the operator guide, start an agent session in
the repository, ask it to run the pilot using the project skill, approve each decision when
asked, merge the PR on GitHub, and tear down. Check the audit.

**Acceptance Scenarios**:

1. **Given** a set-up machine and an agent session with no terminal, **When** the agent
   starts a run, **Then** the run pauses before any worker is staged, shows the run mode, and
   names the single approval that would let it continue.
2. **Given** a run paused at a decision, **When** the agent proposes the approval, **Then**
   the operator's harness asks the operator before the approval is recorded.
3. **Given** the operator approved, **When** the agent resumes the run, **Then** the run
   continues past exactly that decision and no further decision.
4. **Given** task A's PR is published, **When** the agent resumes before the operator merges,
   **Then** the run reports that the PR is not merged and waits; no approval can replace the
   merge.
5. **Given** all three decisions were approved through the harness and the PR merged, **When**
   the run finishes and is torn down, **Then** the audit passes all 11 checks and reports the
   harness channel for the mode, the retry and the status change.

---

### User Story 2 - Nothing proceeds without a human (Priority: P1)

Whatever the agent does, a decision is only taken when a human made it. Without a terminal
and without an approval, the run waits. An approval only counts for the decision the run is
actually waiting for, after that decision was shown, and only once.

**Why this priority**: this is the guarantee that justified rejecting `--yes`. If it does not
hold, the feature weakens the project's core promise.

**Independent Test**: drive the approval command and the run directly, without an agent,
through each refusal case below, and confirm the run never advances.

**Acceptance Scenarios**:

1. **Given** no terminal and no approval, **When** a run reaches any decision, **Then** it
   pauses and stages nothing further.
2. **Given** a run paused at a retry, **When** an approval for a different step (mode,
   status change) is attempted, **Then** it is refused and the run stays paused.
3. **Given** a run waiting for the merge (status change not yet shown), **When** an approval
   for the status change is attempted, **Then** it is refused with an explanation that the
   merge happens on GitHub.
4. **Given** an approval was used, **When** the run pauses at a later decision, **Then** the
   earlier approval does not apply to it.
5. **Given** an approval recorded for one attempt number or one merge commit, **When** the run
   is waiting for a different one, **Then** the approval is not used.
6. **Given** a run that is not paused, **When** an approval is attempted, **Then** it is
   refused.
7. **Given** the project's harness settings, **When** an agent tries to write or edit a run's
   approval record directly, **Then** the harness blocks it.

---

### User Story 3 - A reviewer can tell how each decision was made (Priority: P2)

A security reviewer reading the evidence can see, for every confirmed decision, whether it
was answered at a terminal or approved through the harness, and the audit states it.

**Why this priority**: the two channels have different strengths; the record must not blur
them.

**Independent Test**: audit one terminal-driven run and one agent-driven run and compare the
channel reported for each decision.

**Acceptance Scenarios**:

1. **Given** a run whose decisions were answered at a terminal, **When** it is audited,
   **Then** the mode and retry checks report the terminal channel.
2. **Given** a run whose decisions were approved through the harness, **When** it is audited,
   **Then** the same checks report the harness channel.
3. **Given** evidence recorded before channels existed (runs from 002, 003 and 006), **When**
   it is audited, **Then** it still passes and reports no channel rather than guessing one.

---

### User Story 4 - Operators at a terminal see no change (Priority: P3)

An operator who runs the pilot in a real terminal gets the same prompts and outcomes as
before.

**Why this priority**: the guide and its validated walkthrough must stay correct.

**Independent Test**: run the pilot in a terminal and compare prompts, pauses and exit codes
with the operator guide.

**Acceptance Scenarios**:

1. **Given** a terminal, **When** the operator answers each prompt, **Then** the prompts,
   pauses and exit codes match `docs/operator-guide.md`.
2. **Given** a terminal and a leftover approval record, **When** a prompt appears, **Then**
   the operator's answer decides, not the record.
3. **Given** a terminal, **When** the operator answers "no" to the mode, **Then** the run
   ends as before (not paused).

---

### Edge Cases

- The harness runs in a mode that skips permission prompts (bypass or auto-approve): the
  skill tells the agent to stop and ask the operator to run the approval themselves; the
  documentation states that a harness approval is only as strong as the harness setting.
- The operator approves, then the run is resumed from a terminal: the terminal answer decides
  and the leftover approval is not used.
- Two approvals in a row for the same step: the second replaces the first; only one is ever
  pending.
- The status change was shown for one merge commit and the PR history changed before resume:
  the approval does not apply.
- The run fails or is reset after an approval: the approval belongs to that run only.
- An agent harness other than the two supported ones: no ask rule exists; the documentation
  says so and points to the terminal or, later, GitHub approvals (roadmap 004).
- The agent loses track of the run id: `status` shows it, and every approval names the run.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Without a terminal, every human decision (run mode, retry, status change) MUST
  pause the run instead of being assumed, and MUST say which approval would let it continue.
- **FR-002**: A paused run waiting for the mode confirmation MUST be resumable; nothing is
  staged before the mode is approved.
- **FR-003**: The system MUST provide an approval command for the mode, retry and status-change
  decisions. It MUST NOT offer a way to approve a merge; merges stay on GitHub.
- **FR-004**: An approval MUST be accepted only for the decision the run is currently waiting
  for, and only after that decision was shown (mode printed; the failed attempt graded; the
  status-change proposal printed for a merged PR).
- **FR-005**: An approval MUST be bound to its decision (task, attempt number, and for a status
  change the merge commit) and used at most once. A mismatched approval MUST NOT be used.
- **FR-006**: When a terminal is available, the terminal answer MUST decide and any approval
  record MUST be ignored.
- **FR-007**: Each decision MUST be recorded in the run's evidence with its channel: terminal,
  harness, or none (no answer).
- **FR-008**: The audit's retry check (AU09) and mode check (AU10) MUST report the channel of
  each confirmed decision. Their pass/fail rules are otherwise unchanged.
- **FR-009**: Evidence recorded before channels existed MUST still pass the audit, reporting no
  channel.
- **FR-010**: The project MUST ship harness settings for Claude Code and OpenCode that make the
  approval command always ask the human, and settings that deny an agent direct edits to
  approval records.
- **FR-011**: The project MUST ship a `pilot` skill and an `AGENTS.md` pointer that teach an
  agent: the commands and exit codes; to explain each decision and ask the human; to propose
  the approval only after a clear yes; never to approve for the human, edit approvals or
  evidence, merge, or print secrets; and to stop if the harness would not ask the human.
- **FR-012**: The troubleshooting entry for missing prompts and the pilot README MUST describe
  both ways forward (a terminal, or an approval through the harness) and state the harness
  channel's limits.
- **FR-013**: Terminal behaviour, prompts and exit codes documented in the operator guide MUST
  stay unchanged.

### Key Entities

- **Decision**: a point where the run needs a human: mode, retry (per task and attempt),
  status change (per task and merge commit). Merge is a human action on GitHub, not a decision
  the pilot records.
- **Approval**: a human's yes to one pending decision, recorded through the approval command;
  bound to that decision; used once.
- **Channel**: how a decision was answered: terminal, harness, or none.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: An operator and an agent complete a full run together on a set-up machine, with
  all 3 decisions approved through the harness, the merge done on GitHub, and an audit of
  11/11 after teardown.
- **SC-002**: In every one of the 7 refusal cases of User Story 2, the run does not advance (0
  decisions taken without a matching human approval).
- **SC-003**: The audit reports a channel for 100% of confirmed decisions in runs recorded
  after this feature.
- **SC-004**: All previously committed run evidence (002, 003, 006) still passes the audit.
- **SC-005**: The operator sees the harness ask before every approval in both supported
  harnesses (Claude Code and OpenCode), confirmed by the owner.
- **SC-006**: A terminal-driven run behaves as the operator guide describes, with no change
  to prompts or exit codes.

## Assumptions

- The first cut in PR #11 is the starting point; this spec may change it where the
  requirements demand.
- Supported harnesses are Claude Code and OpenCode. Others get no ask rule in this item.
- The harness channel trusts the harness permission rule and the agent not to forge records
  by other means; it is documented as weaker than the terminal and than GitHub approvals,
  which roadmap 004 adds as a third channel reusing the same approval record.
- Run `20261114T213046Z` (agent-driven, all three decisions through the harness) is early
  evidence; its record is copied into this spec's evidence folder, and SC-001 is validated by a
  fresh run on the finished feature.
- Validation uses the same throwaway fixture repo and machine setup as the operator guide.
