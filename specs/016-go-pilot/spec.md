# Feature Specification: Go pilot at parity

**Feature Branch**: `016-go-pilot`

**Created**: 2026-11-15

**Status**: Draft

**Input**: User description: "016 Go pilot at parity (roadmap entry 016, constraint C-13). Rewrite the pilot CLI in Go, reproducing the reference Python pilot exactly before any new feature: same commands, exit codes, evidence format and approval channels; the Go audit gives the same results on every committed evidence record (002, 003, 006, 012 — the golden data); a fixture run passes 11/11. Stories: US1 audit+gate parity on golden evidence (P1); US2 fixture run end to end with terminal and harness approvals (P1); US3 GitHub loop (004 behaviour) incl. 004's owner validation on the Go pilot (P2); US4 walk away during a run — the run starts a watcher that survives the terminal closing (issue #25) (P2); US5 retire the Python pilot (P3). Assumptions: end-to-end support stays Linux (kind+Docker); binary also builds for macOS but macOS pilot needs multi-arch images (out of scope); in-cluster image scripts stay Python; spec-status needs a public Go package for library use. Out of scope: new features (014, 015, 011), release packaging (013), porting image scripts."

## Background

The project decided to use one implementation language, Go, for the pilot CLI and the future
operator (roadmap constraint C-13, owner decision 2026-11-15). The pilot that exists today is
in Python. Its behaviour has been proven through four verified roadmap items (002, 003, 006,
012), and their evidence records are committed in the repository. 004 (the GitHub loop) is
implemented in Python, but its owner validation was stopped so that it can be validated on the
Go pilot instead.

This feature replaces the Python pilot with a Go pilot **that behaves the same**. It must not
change what operators type, what they see at each decision, what the run records, or what the
audit concludes. New capabilities wait for later roadmap items (014, 015, 011). The one
addition is walking away during a run (issue #25), which the Go design was chosen to solve.

The language is the subject of this feature (C-13), so naming it is a requirement, not an
implementation detail.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - The audit and the gate give the same answers (Priority: P1)

A security reviewer re-checks the committed evidence of past runs with the new pilot and gets
exactly the conclusions the original pilot gave. An operator evaluating the gate on the fixture
gets the same verdicts.

**Why this priority**: the audit and the gate are the project's two trust anchors (evidence
decides "done"; gates fail closed). If they drift, every other parity claim is meaningless.
They are also pure enough to prove before any cluster work.

**Independent Test**: run the new pilot's audit over every committed pilot evidence record, and
compare each check's result and detail with the reference. Evaluate the gate for both fixture
tasks and compare the verdicts and reasons.

**Acceptance Scenarios**:

1. **Given** the committed evidence records of 002, 003, 006 and 012, **When** the new audit runs
   on each, **Then** every check (AU01–AU11) has the same result and the same detail as the
   reference audit on that record.
2. **Given** the committed failed run of 002, **When** the new audit runs on it, **Then** it fails
   the same checks, for the same reasons, as the reference.
3. **Given** the fixture repository at its reset state, **When** the gate is evaluated for task A
   and task B, **Then** A is allowed and B is refused with the same reasons as the reference.
4. **Given** each gate fail-closed case (missing roadmap, unreadable status, blocked item, and
   the rest of the matrix), **When** the new gate evaluates it, **Then** the verdict is a refusal,
   as in the reference.

---

### User Story 2 - An operator completes a fixture run with the new pilot (Priority: P1)

An operator who followed the operator guide stands up the cluster, runs the pilot, answers the
mode and retry prompts at the terminal, merges task A's PR, resumes, and tears down. Everything
they type and see matches the guide. The run ends with a passing audit. The same run driven by
an agent, with approvals through the harness, also completes.

**Why this priority**: this is the product's core loop. Parity here is what allows the Python
pilot to be retired.

**Independent Test**: on a machine set up per the operator guide, follow the guide's
walkthrough with the new pilot, from stand-up to teardown, then run the audit. Repeat once
through an agent harness.

**Acceptance Scenarios**:

1. **Given** a set-up machine, **When** the operator follows the guide with the new pilot,
   **Then** every command named in the guide exists, accepts the same options, and prints the
   prompts and pauses the guide describes.
2. **Given** each outcome the guide documents (allowed, refused, paused for a retry, paused for
   the merge, declined, error, repo guard), **When** it occurs, **Then** the new pilot exits with
   the same exit code as the reference.
3. **Given** a completed fixture run, **When** the operator tears down and runs the audit,
   **Then** it passes all 11 checks.
4. **Given** the run's evidence folder, **When** it is compared with a reference run's folder,
   **Then** it has the same files, the same record formats and the same event names. A reviewer
   who knows the old layout can read the new one without guidance.
5. **Given** a run with no terminal, **When** a decision is reached, **Then** the run pauses, and a
   human approval through the harness (012) lets it continue exactly as in the reference. The
   audit reports the `tty` and `harness` channels as before.
6. **Given** a credential that fails its check (publisher token), **When** it is stored, **Then**
   it is not saved and the previous one is kept, as in the reference.

---

### User Story 3 - The GitHub loop works on the new pilot and is validated (Priority: P2)

An operator turns on the GitHub loop. They see item issues with status labels and task
checklists, PRs that close their issue, approvals given on GitHub, and a status-change PR whose
merge approves the roadmap change. They can also choose a GitHub App as the publisher. This
is 004's behaviour, validated for the first time, on the new pilot.

**Why this priority**: 004 was implemented but never validated. Its validation moves here, so
this is where 004's outcome gets proven.

**Independent Test**: run 004's quickstart validation (§2–§5) against the new pilot, including
every refusal case, in token mode and in GitHub App mode.

**Acceptance Scenarios**:

1. **Given** the GitHub loop is on, **When** the operator syncs, **Then** there is one issue per
   roadmap item with one status label and its task checklist, and a second sync changes nothing.
2. **Given** a run paused at a decision with GitHub approvals on, **When** an allowed approver
   replies with the approval command, **Then** the run continues past exactly that decision, and
   the audit records the `github` channel and the approver.
3. **Given** each approval-refusal rule defined in 004 (wrong step, posted before the decision,
   not an approver, no write access, publisher identity, app-authored, edited, withdrawn,
   duplicate, GitHub unreachable), **When** that case occurs, **Then** the run does not advance,
   and the reason is recorded exactly as in the reference rules.
4. **Given** task A's PR is merged, **When** the status-change PR is merged unedited by an allowed
   approver, **Then** task B proceeds. **When** it was edited, or merged by the publisher or by a
   non-approver, **Then** the run stops with the corresponding reason.
5. **Given** the GitHub App credential, **When** a run publishes, **Then** the PRs are authored by
   the App, the token is limited to the fixture repository, and the audit passes.

---

### User Story 4 - The operator can walk away during a run (Priority: P2)

With GitHub approvals on, the operator starts a run and closes the terminal. When they approve
on GitHub or merge a PR, the run continues by itself. When the run ends, nothing is left
running.

**Why this priority**: it answers issue #25 from the 004 validation, and the Go design was
chosen partly to solve it.

**Independent Test**: start a run with GitHub approvals on, close the terminal session, approve
and merge on GitHub only, and check that the run completes and that no watcher remains
afterwards.

**Acceptance Scenarios**:

1. **Given** a run with GitHub approvals on, **When** it first pauses, **Then** a watcher for that
   run starts without any extra command, and it keeps running after the terminal closes.
2. **Given** a running watcher, **When** a valid approval or the expected merge appears, **Then**
   the run resumes, using exactly the same rules as a manual resume.
3. **Given** a run that ends (succeeded, failed or torn down), **When** that happens, **Then** its
   watcher exits.
4. **Given** a run, **When** it pauses more than once, **Then** at most one watcher runs for it.
5. **Given** the operator wants the foreground behaviour, **When** they run the watch command
   themselves, **Then** it behaves as today's foreground watch.
6. **Given** the operator checks the run's status, **When** a watcher is running, **Then** the
   status says so.

---

### User Story 5 - The Python pilot is retired without breaking anything (Priority: P3)

Once the new pilot reaches parity, the old one is removed. Operators still use the same command
name, and the guide, troubleshooting, skill and README are all still correct.

**Why this priority**: it completes the pivot and removes the double maintenance. It comes last,
because it must only happen after US1–US4 hold.

**Independent Test**: after retirement, run the documentation checks, follow the guide's
walkthrough once more, and confirm that no part of the repository refers to the removed
implementation, except history and evidence.

**Acceptance Scenarios**:

1. **Given** parity was shown (US1–US4), **When** the old pilot is removed, **Then** the
   operator-facing command name and its usage stay the same.
2. **Given** the documentation, the agent skill and the README, **When** they are checked,
   **Then** every command, option and path they name exists.
3. **Given** the committed evidence of past runs, **When** they are audited after retirement,
   **Then** they still pass (or fail) exactly as before.

---

### Edge Cases

- **Records from before newer events existed** (for example, approval channels before 012):
  they are audited exactly as the reference audits them, with no guessed values.
- **A run started with the old pilot, resumed with the new one:** this is not supported. The new
  pilot must say so clearly and refuse, rather than misread the run.
- **A cluster stood up by the old pilot:** the new pilot detects it and either uses it or asks for
  a fresh stand-up. It never half-uses it.
- **A missing dependency** (the gate's status tool, the container runtime, the cluster tool):
  the new pilot fails closed with the same guidance as the reference.
- **The watcher host machine reboots:** the watcher is gone. The run stays paused and can be
  resumed by hand.
- **Two watchers started at once** (for example, two quick pauses): only one survives.
- **The status tool is unavailable as a library:** the gate must still fail closed, never allow.
- **Platform:** on a platform where the pilot can't run end to end (macOS today), the binary still
  starts and reports clearly what is unsupported.

## Requirements *(mandatory)*

### Functional Requirements

**Parity**

- **FR-001**: The new pilot MUST provide every command and option of the reference pilot, under
  the same operator-facing command name:
  `up`, `down`, `run`, `resume`, `approve`, `status`, `audit`, `gate`, `publisher-token`,
  `import-opencode`, `reset-fixture`, `github sync|drift|watch`, `app-key`, `app-token`.
- **FR-002**: Each command MUST return the same exit codes as the reference for the same outcome:
  0 ok, 1 fail, 2 error, 3 repo guard, 4 awaiting merge, 5 declined, 6 awaiting retry, 10 gate
  refusal.
- **FR-003**: The prompts, pauses and messages that the operator guide and troubleshooting quote
  MUST stay the same. Other wording may change only where the docs are updated in the same
  change.
- **FR-004**: Runs MUST write the same evidence layout and record formats as the reference
  (files, fields, event names, check ids), as defined in the 002, 012 and 004 contracts.
- **FR-005**: The audit MUST produce, for every committed pilot evidence record, the same result
  and the same detail for every check as the reference audit.
- **FR-006**: The gate MUST give the same verdicts and reasons as the reference, on the fixture
  and on every fail-closed case, and it MUST fail closed whenever the status information is
  missing or unreadable.
- **FR-007**: Human decisions MUST follow the same rules as the reference:
  - terminal answers;
  - pauses when there is no terminal;
  - harness approvals bound to the step and used once;
  - GitHub approvals with every refusal rule;
  - the status-change PR check;
  - approval channels recorded.
- **FR-008**: Credential handling MUST keep the reference's guarantees:
  - secret values never enter the pilot's logs or evidence;
  - a credential is saved only if its check passes;
  - the GitHub App key never leaves the operator's machine;
  - the App token is limited to the fixture repository and short-lived.
- **FR-009**: The in-cluster isolation MUST be unchanged:
  - the same namespaces, policies and images by digest;
  - only the publisher holds a repository credential;
  - the live checks P01–P10 pass.

**Validation**

- **FR-010**: The project MUST hold a repeatable parity check. It runs the new audit over every
  committed evidence record and compares the results with the reference results (golden data).
- **FR-011**: The approval-refusal rules and the status-change PR check MUST be expressed as shared,
  language-neutral test cases, which the new pilot (and later the operator) must pass.
- **FR-012**: 004's owner validation (its quickstart §2–§5) MUST be run against the new pilot, and
  its results recorded as 004's evidence.

**Walk away (issue #25)**

- **FR-013**: With GitHub approvals on, a run MUST start a watcher for itself at its first pause.
  The watcher MUST keep running after the terminal closes, resume the run under the same rules
  as a manual resume, and exit when the run ends. At most one watcher may run per run.
- **FR-014**: The run's status MUST show whether its watcher is running. The foreground watch
  command MUST remain available.

**Gate integration**

- **FR-015**: The gate MUST read item status through spec-status's own logic as a library, not
  through a reimplementation. Until spec-status offers a public library interface, the gate MAY
  use the status tool's output as today, with identical results.

**Retirement**

- **FR-016**: The reference Python pilot MUST be removed only after FR-001 to FR-014 are shown.
  After removal:
  - the operator-facing command name and usage are unchanged;
  - documentation, the agent skill and the README name only commands, options and paths that
    exist;
  - the in-cluster image scripts are untouched.

**Scope**

- **FR-017**: No new operator-visible capability other than FR-013, FR-014 and FR-018. Items
  014, 015, 011 and 013 stay out of scope.
- **FR-018**: The pilot MUST offer diagnostics on stderr at three levels (`-v` steps and
  timings, `-vv` pod phases, container states, Kubernetes events and model-proxy refusals,
  `-vvv` live agent activity from worker pods), also selectable with `PILOT_VERBOSE=N`.
  Diagnostics MUST NOT change stdout, exit codes or evidence, so they are outside parity, and
  every line MUST be redacted (secret values and the model hosts never appear). Added
  2026-11-15 as a debugging aid for the Go runs (roadmap 1.11.0 note on 016).

### Key Entities

- **Reference pilot**: the existing Python pilot, the definition of correct behaviour until
  retirement.
- **Golden data**: the committed evidence records of 002, 003, 006 and 012 (eight records,
  including one deliberate failure), together with the reference audit's results on each.
- **Parity check**: the repeatable comparison of the new audit's results with the golden data.
- **Shared test cases**: language-neutral inputs and expected outcomes for the approval rules and
  the status-change PR check.
- **Watcher**: a per-run background process that resumes a paused run when a valid approval or
  merge appears.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: On all 8 committed pilot evidence records, 100% of audit results (86 today, because
  the two records without a teardown are audited without the teardown check) match the
  reference in result and detail. That includes the 4 deliberate failures in 002's failed run.
- **SC-002**: A fixture run, from stand-up to teardown, with the new pilot ends with the audit at
  11/11, following the operator guide with 0 deviations from the documented commands, prompts
  and exit codes.
  *Met by the two harness-approved runs recorded in `evidence/README.md`; the owner accepted
  these in place of a terminal (`tty`) run (2026-11-16).*
- **SC-003**: An agent-driven fixture run with harness approvals completes with the audit at
  11/11, with the `harness` channel recorded for every decision.
- **SC-004**: In 004's validation on the new pilot:
  - every refusal case leaves the run unadvanced (0 decisions without a matching approval);
  - token mode and App mode each finish with the audit at 11/11.
- **SC-005**: An operator can close the terminal after starting a run and finish it by acting on
  GitHub only, with 0 commands typed after the start, and 0 watchers left once the run ends.
- **SC-006**: After retirement:
  - the documentation check reports 0 problems;
  - the guide walkthrough still completes;
  - no file outside history and evidence refers to the removed implementation.
- **SC-007**: The pilot's operator-facing behaviour on Linux is unchanged for someone who learned
  it from the current guide: 0 command, option or exit-code changes.

## Assumptions

- **Platform:** end-to-end support stays Linux (kind and a container runtime), as today. The new
  pilot also builds for macOS, but running the pilot on macOS needs multi-architecture images,
  which is out of scope.
- **Image scripts:** the in-cluster scripts (grader, publisher, token check, GitHub writes) stay as
  they are, behind their contracts.
- **spec-status as a library:** this needs a public library interface in spec-status (the same
  owner). That is a cross-repository dependency, planned with this feature. FR-015 allows the
  current tool output until it exists.
- **No mixed runs:** runs started with the reference pilot are not resumed by the new one. A
  fresh fixture reset is expected when switching.
- **Golden data:** 004 has no golden data, because its validation never ran. FR-012 creates its
  evidence.
- **Source of truth:** the roadmap and constitution are unchanged in substance. C-13 is the
  governing constraint.
