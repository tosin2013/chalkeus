# Feature Specification: Resource budgets and capacity checks

**Feature Branch**: `030-resource-budgets`

**Created**: 2026-11-24

**Status**: Draft

**Input**: Roadmap entry 030 (added 2026-11-24, roadmap 1.34.0), from onboarding
tosin2013/documcp for 026's live slice run (friction log
`specs/026-slice-runs/evidence/documcp-onboarding.md`, F13). A target's profile may raise the
CPU, memory and time of every pod the target uses, and forge checks before a run that the
cluster can schedule them, so a large or slow project fails early with a message that says what
to change.

Governed by C-02 (gate fails closed), C-03 (isolation and credentials), C-11 (two modes, one
set of guarantees) and C-16 (deterministic grading).

## Context

What happened on documcp (2026-11-24): `forge author check` failed `DRAFT_SUITE_FAILS`. The
project's own tests hit the grader's 600-second per-command limit on the unchanged code. On 2
cores (the grader's CPU) Jest runs its tests in one process, finishes the 1,706 tests in about
65 seconds, then never exits because of leaked handles. The owner unblocked it with
`--forceExit` in the target's test command, but found that nothing lets a target raise CPU or
the command timeout, and the worker runs the suite several times in one attempt.

The budgets today (toolchain catalog and pod templates):

| Pod | CPU | Memory | Time | Profile can raise |
|---|---|---|---|---|
| Dependency prefetch | grader's | grader's | grader's job deadline | memory (through the grader's) |
| Worker (agent attempt) | 2 | 2Gi | 1,200 s per attempt | nothing |
| Grader | 2 | 2Gi | 900 s per job, 600 s per command | memory, job deadline |
| Test author (draft job; `check` uses prefetch and grader) | 2 | 2Gi | its job deadline | nothing |

Every pod requests 500m CPU and 512Mi, far below its limit, so a pod can be scheduled on a node
that can never give it its full budget; it then runs slowly or is killed for memory, and the
failure looks like the project's fault.

Terms:
- **Budget:** a pod role's CPU limit, memory limit and time limits.
- **Pod role:** prefetch, worker, grader or author.
- **Catalog budget:** the toolchain catalog's budget for a role (or the operator-supplied
  toolchain defaults); the floor.
- **Target budget:** the catalog budget with the profile's raises applied.
- **Capacity:** what the cluster can give a pod: the largest node's allocatable CPU and memory
  after Chalkeus's own pods, and any quota or default limits in the namespaces the pod runs in.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Raise a target's budget, and take it back (Priority: P1)

An owner whose project needs more CPU, memory or time adds lines to the target's profile.
Every pod the target uses gets the raised budget on the next run or draft check. Deleting the
lines returns the target to the catalog budget.

**Why this priority**: without it a project like documcp can't be graded at all when its suite
needs more than the fixed budget; everything else in this item builds on it.

**Independent Test**: set a raised worker CPU, grader CPU and command timeout in a target's
profile; start a run; the worker, prefetch and grader pods carry the raised limits and the
grader applies the raised command timeout. Remove the lines; the next run uses the catalog
budget.

**Acceptance Scenarios**:

1. **Given** a profile that raises worker CPU, worker memory and the worker attempt deadline,
   **When** a run starts, **Then** each worker pod carries those limits and that deadline.
2. **Given** a profile that raises grader CPU and the per-command timeout, **When** a run or a
   draft check grades, **Then** the grader and prefetch pods carry that CPU and each test
   command may run for the raised time.
3. **Given** a profile that sets any budget value below the catalog's, **When** the profile is
   read, **Then** it is refused with a message naming the key and the catalog floor.
4. **Given** a profile with a malformed value (not a CPU or memory quantity, or not a positive
   number of seconds), **When** the profile is read, **Then** it is refused with a message naming
   the key.
5. **Given** a run that started with a raised budget, **When** the owner removes the lines
   before resuming it, **Then** the run keeps the budget it started with; the next new run uses
   the catalog budget.
6. **Given** a profile that raises the author's budget, **When** the test author's draft job
   runs, **Then** it carries that budget.

---

### User Story 2 - Know before a run whether the cluster can run it (Priority: P1)

Before a run or a draft check, doctor compares the target budget with the cluster's capacity and
says plainly whether it fits; if it doesn't, it says what to change for that kind of cluster.

**Why this priority**: a raised budget on a laptop is the likeliest way to get pods stuck
`Pending` or killed for memory mid-run, which costs model spend and is hard to diagnose.

**Independent Test**: on a cluster whose largest node has less allocatable CPU than the target's
grader CPU, `forge doctor --target <t>` fails a capacity check with the role, the budget, the
capacity and what to change; with a budget that fits, the check passes and states the headroom.

**Acceptance Scenarios**:

1. **Given** a target budget whose largest role fits the largest node after Chalkeus's own pods,
   **When** doctor runs, **Then** the capacity check passes and shows the largest role's budget
   and the node's remaining capacity.
2. **Given** a local cluster on the owner's machine (the kind of cluster `forge up` creates) that
   can't fit the largest role, **When** doctor runs, **Then** it fails and the message says to
   give the container runtime more CPU or memory, or lower the target's raise, with the numbers.
3. **Given** a remote cluster that can't fit the largest role, **When** doctor runs, **Then** it
   fails and the message says to use a larger node size or node pool, with the numbers.
4. **Given** a namespace quota or default limit that would refuse or cut the role's budget,
   **When** doctor runs, **Then** it fails and names the quota and the namespace to change.
5. **Given** credentials that can't read nodes or quotas, **When** doctor runs, **Then** the
   capacity check warns that it couldn't check, and doesn't fail.
6. **Given** `forge run` or `forge author check`, **When** it starts, **Then** it runs the same
   capacity check first and refuses to start on a fail.

---

### User Story 3 - A pod that can't be scheduled stops the run with the same message (Priority: P2)

If a pod still can't be scheduled for lack of CPU or memory (capacity changed after doctor, or
couldn't be read), the run stops soon with the capacity message instead of waiting out the
pod's deadline.

**Why this priority**: it turns a silent stall into a fast, explained stop; doctor (US2) already
prevents most cases.

**Independent Test**: with capacity unreadable and a budget larger than any node, start a run;
the first pod stays unschedulable, and within the wait limit the run stops with the capacity
message, the role and the budget, and without spending an attempt.

**Acceptance Scenarios**:

1. **Given** a pod unschedulable for lack of CPU or memory for longer than the wait limit,
   **When** forge observes it, **Then** the run stops with the capacity message for that role.
2. **Given** a pod that waits briefly and is then scheduled, **When** the wait is shorter than
   the wait limit, **Then** the run continues.
3. **Given** a run stopped this way, **When** the owner reads its evidence, **Then** the stop is
   recorded as a capacity stop, not as an attempt failure, and counts against no attempt.

---

### User Story 4 - The budget is visible and recorded (Priority: P2)

When the owner approves a run's mode, they see the budget each role will get and which values
the profile raised. The run's evidence, and a draft's, record the budget used.

**Why this priority**: a bigger budget uses more of the owner's machine or cluster; the owner
should approve it knowingly, and an auditor should see it.

**Independent Test**: approve the mode of a run with a raised budget; the approval shows each
role's budget with the raised values marked; the run's evidence records the same budget.

**Acceptance Scenarios**:

1. **Given** a target with raised values, **When** the mode approval is shown, **Then** it lists
   each role's CPU, memory and time limits and marks which came from the profile.
2. **Given** a target with no raises, **When** the mode approval is shown, **Then** it says the
   catalog budget is used.
3. **Given** a finished run or draft check, **When** its evidence is read, **Then** it records
   each role's budget and the capacity check's result.

---

### User Story 5 - Sizing docs (Priority: P3)

An owner reads one docs section on sizing a target: the catalog budgets, the keys, how to tell a
slow suite from a stuck one, and how to read the capacity message.

**Why this priority**: the documcp case was a stuck suite, not a slow one; raising the budget
would not have helped, and the docs should say how to tell.

**Independent Test**: the targets docs have a sizing section with the catalog table, every key,
the capacity message for both kinds of cluster, and the stuck-vs-slow check.

**Acceptance Scenarios**:

1. **Given** the targets docs, **When** an owner looks for how to raise a budget, **Then** they
   find every key, its floor and an example.
2. **Given** a test command that times out, **When** the owner follows the docs, **Then** they
   can tell whether the suite is slow (raise the budget) or never exits (fix the command or the
   leak), with the documcp case as the example.

---

### User Story 6 - An attempt that runs out of time still leaves its work (Priority: P1)

When a worker attempt reaches its deadline, forge still records what the agent changed, its
activity log and its token usage and cost, and the owner sees them when deciding on a retry.

**Why this priority**: documcp's US1 run (`20261124T152325Z`) lost two 20-minute attempts with no
diff, no log and no usage (friction F15): the owner couldn't tell how far the agent got or what
it spent, and the spend cap couldn't count it.

**Independent Test**: run an attempt whose deadline is shorter than the work it is given: the
attempt fails as a deadline, and its evidence has the partial change, the agent's log and its
tokens and cost; the run's spend includes them.

**Acceptance Scenarios**:

1. **Given** an attempt that reaches its deadline, **When** forge records it, **Then** the
   evidence has the agent's change so far (possibly empty), its activity log and its usage,
   marked as stopped by the deadline.
2. **Given** a deadline-stopped attempt, **When** the run's spend is computed, **Then** that
   attempt's tokens and cost count against the caps.
3. **Given** a deadline-stopped attempt, **When** forge asks the owner about a retry, **Then**
   the request states the deadline stop, the time used and the attempt's usage.
4. **Given** a deadline-stopped attempt, **When** grading is considered, **Then** its partial
   change is not graded and is never published.

---

### Edge Cases

- An operator-supplied toolchain image: its defaults are the floor, as for catalog toolchains.
- A raise only for the grader: the prefetch pod follows the grader's budget, as today.
- The worker's prompt states the grader's per-command timeout; with a raised timeout it states
  the raised value.
- A cluster with several node sizes: capacity is judged on the largest node that the role's pods
  may be scheduled on.
- Chalkeus's own pods (egress proxies, publisher) already running take part of a node; capacity
  is what remains.
- Capacity readable but a quota unreadable: the node part is checked; the quota part warns.
- A budget that fits only one role at a time: roles run one after another (prefetch, then worker,
  then grader), so the check judges the largest role, not the sum.
- The fixture target: it has no profile keys for budgets; it uses the catalog budget and gets the
  same capacity check.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: A target's profile MUST accept raises for: worker CPU, worker memory, worker
  attempt deadline, grader CPU, per-command timeout, author CPU, author memory and author job
  deadline, alongside today's grader memory and grader job deadline.
- **FR-002**: Every budget value MUST be at least the catalog budget for its role; a lower or
  malformed value MUST be refused when the profile is read, naming the key and the floor.
- **FR-003**: A value the profile doesn't set MUST be the catalog budget; deleting a line MUST
  return that value to the catalog budget for the next run or draft.
- **FR-004**: The prefetch pod MUST get the grader's CPU and memory, as today.
- **FR-005**: The grader MUST apply the per-command timeout from the target budget, and the
  worker's prompt MUST state it.
- **FR-006**: A run MUST keep the budget it started with when resumed, even if the profile
  changed.
- **FR-007**: doctor MUST check the target budget against the cluster's capacity: the largest
  role's CPU and memory limits against the largest eligible node's allocatable resources after
  Chalkeus's own pods, and against any quota or default limit in the namespaces the role runs in.
- **FR-008**: A failed capacity check MUST name the role, its budget, the capacity found, and
  what to change: for a local cluster on the owner's machine, the container runtime's resources
  or a lower raise; for a remote cluster, the node size or node pool; for a quota, the quota and
  its namespace.
- **FR-009**: When nodes or quotas can't be read, the capacity check MUST warn and not fail.
- **FR-010**: `forge run` and `forge author check` MUST run the capacity check before starting
  and refuse to start when it fails (C-02).
- **FR-011**: A pod unschedulable for lack of CPU or memory for longer than a wait limit MUST
  stop the run with the FR-008 message; the stop MUST be recorded as a capacity stop and MUST
  NOT count as an attempt.
- **FR-012**: The mode approval MUST show each role's budget and mark the values the profile
  raised.
- **FR-013**: A run's and a draft check's evidence MUST record each role's budget and the
  capacity check's result.
- **FR-014**: Raising a budget MUST NOT change isolation: network policy, egress allowlists,
  credentials and storage stay as they are for each role (C-03).
- **FR-015**: The targets docs MUST have a sizing section: the catalog budgets, every key and
  its floor, the capacity messages, and how to tell a slow test command from one that never
  exits.
- **FR-016**: An attempt stopped by its deadline MUST still yield its change so far, the agent's
  activity log and its token usage and cost in the attempt's evidence, marked as a deadline
  stop; the worker MUST stop the agent early enough before the pod's deadline to collect them.
- **FR-017**: A deadline-stopped attempt's usage MUST count against the run's spend and token
  caps, and the retry request MUST state the stop, the time used and the usage; its partial
  change MUST NOT be graded or published.

### Key Entities

- **Budget:** per role, CPU limit, memory limit and time limits (attempt deadline for the
  worker; job deadline and per-command timeout for the grader; job deadline for the author).
- **Capacity check result:** pass, fail or warn; the role judged; its budget; the capacity
  found; the kind of cluster; what to change.
- **Capacity stop:** a run stop caused by an unschedulable pod, with the role and the message.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: With a raised budget, 100% of the target's pods in a run and in a draft check carry
  the raised limits; with the lines removed, 100% carry the catalog budget.
- **SC-002**: On a cluster too small for the target budget, the owner learns it from doctor, in
  one message that names the change to make, before any model spend.
- **SC-003**: A pod that can't be scheduled stops the run within the wait limit (at most 2
  minutes), with no attempt spent.
- **SC-004**: The documcp target, with its suite unchanged, can be configured from the docs alone
  to get a budget it can pass, or the docs lead the owner to the stuck-suite fix.
- **SC-005**: Every run's evidence states the budget used, and the audit can compare it with the
  mode approval.
- **SC-006**: 100% of deadline-stopped attempts have their change, log and usage in the
  evidence, and the run's spend total includes them.

## Assumptions

- Kubernetes schedules a pod on its requests; requests stay small and the check judges the
  limits, because a pod scheduled where its limit can't be met runs slowly or is killed for
  memory.
- Roles run one after another for a run (prefetch, worker attempts, grader), so the largest
  role decides capacity; Chalkeus's own pods are counted as already running.
- The wait limit for an unschedulable pod defaults to 2 minutes; it isn't a profile key.
- Lowering below the catalog, autoscaling or provisioning nodes, per-run override flags and the
  operator's cluster mode (011) are out of scope (roadmap 030).
- Whether a cluster is local is known from the cluster `forge up` created; any other cluster is
  treated as remote.
