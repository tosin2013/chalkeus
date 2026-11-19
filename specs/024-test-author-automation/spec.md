# Feature Specification: Test-author automation

**Feature Branch**: `024-test-author-automation`

**Created**: 2026-11-19

**Status**: Draft

**Input**: Roadmap entry 024 (owner decisions 2026-11-19). The test author starts on its own
when an item becomes ready, in laptop mode first, because the laptop release (013) ships with
it. The owner chose three things:
- **the trigger:** the pilot on the laptop;
- **where the author runs:** its own Job on the local cluster, so an unattended agent is
  sandboxed;
- **the harness:** OpenCode, the pinned harness the workers already use.

The operator trigger (011) and a GitHub Actions trigger come later.

## Context

Today (018) the operator drafts an item's hidden tests by opening a separate agent session and
running the test-author skill. That session:
- runs `pilot author start`;
- writes the tests;
- runs `pilot author check`;
- opens the companion pull request with `pilot author submit`.

Every Stockroom item went this way. The operator's only real decision was reviewing and
merging the pull request.

024 removes the manual start. When the operator turns it on for a target, the pilot notices an
item that is ready except for its hidden tests, and drafts them in an isolated Job. It then
checks the draft, and opens the same companion pull request 018 opens. The human still
reviews and merges it, and still approves every run.

Terms (as in 018):
- **Operator:** the human who prepares items and approves runs.
- **Driving session:** whatever runs `pilot/pilot` for the operator. It never drafts, reads or
  runs anything of the test author's.
- **Companion:** the private repository holding hidden tests at
  `<PILOT_COMPANION_PATH>/items/<NNN>/`.
- **Ready for drafting:** every readiness rule (014) passes except `ITEM_NO_HIDDEN_TESTS`, and
  no companion pull request for the item is open.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - An item that becomes ready gets a draft on its own (Priority: P1)

As an operator, after I merge an item's spec and tasks, a companion pull request with drafted
hidden tests appears without my starting anything. I review it and merge it, as I do today.

**Why this priority**: This is the feature. It removes the one manual step left between
planning an item and running it.

**Independent Test**: Turn auto-draft on for a target with a watcher running. Merge a new
item's spec and tasks. A companion pull request for that item opens, checked like a manual
draft (the project's tests pass, the drafted tests fail on today's code), with its coverage
list.

**Acceptance Scenarios**:

1. **Given** auto-draft is on for a target and the watcher is running, **When** an item becomes
   ready for drafting, **Then** within one polling interval plus the drafting time, a companion
   pull request opens for that item. It carries the same check results, coverage list and
   evidence as a manual draft (018).
2. **Given** the draft fails its check (a drafted test passes on today's code, or the project's
   tests break), **When** the check finishes, **Then** no pull request opens, and the reason
   code is recorded. The item is not drafted again until the operator asks, or the spec
   changes.
3. **Given** an open companion pull request for the item, or hidden tests already merged,
   **When** the watcher polls, **Then** it starts nothing for that item.

---

### User Story 2 - The unattended author is sandboxed (Priority: P1)

As an operator, the automatic test author runs isolated: it can reach only the model, holds no
repository credential, and shares nothing with build workers.

**Why this priority**: An agent that starts on its own, with nobody watching, must not be able
to do more than draft tests.

**Independent Test**: During an automatic draft, the isolation checks run against the author's
namespace and pass:
- no SCM credential;
- egress only to the model through its own proxy;
- no route to or from the worker or publisher namespaces;
- no volume shared with workers.

**Acceptance Scenarios**:

1. **Given** a running author Job, **When** the checks run, **Then** its namespace holds only the
   model key secret, its pod has no service-account token, and its network reaches only DNS and
   its own egress proxy, which allows only the model hosts.
2. **Given** the author Job, **When** it finishes, **Then** its draft reaches the pilot only
   through its framed log output, as a worker's change does, and the pilot, not the Job, opens
   the companion pull request with the identity the security level names (018).
3. **Given** a build worker namespace, **When** the checks run, **Then** no author namespace,
   proxy or draft is reachable from it.

---

### User Story 3 - The operator stays in control (Priority: P2)

As an operator, auto-draft is off unless I turn it on for a target. I can see what it did, and
I can stop it.

**Why this priority**: Automation must be opt-in and visible.

**Independent Test**: With auto-draft off, nothing starts. With it on, the watcher prints and
records each decision. Stopping the watcher stops new drafts; a draft in progress finishes or
is cleaned up.

**Acceptance Scenarios**:

1. **Given** a target without `PILOT_AUTHOR_AUTO=on`, **When** an item becomes ready for
   drafting, **Then** nothing starts, and `pilot doctor` still says the item needs hidden tests.
2. **Given** auto-draft on, **When** the watcher decides to draft or skip an item, **Then** it
   prints the decision and the reason, and records it in the draft's evidence.
3. **Given** auto-draft on, **When** the operator stops the watcher during a draft, **Then** no
   pull request opens for that draft, and its namespace is deleted.

---

### Edge Cases

- **Several items ready at once:** they are drafted one at a time, in roadmap order.
- **The spec changes after a failed draft:** the item becomes eligible again, with one new draft
  per spec version.
- **The cluster is down:** the watcher reports it and starts nothing. Readiness polling needs
  no cluster.
- **The strict level:** the Job delivers strict-style isolation on the laptop. The companion
  pull request still uses the identity of the target's level (minimal or recommended). A
  separate App for strict remains 011's.
- **The driving session:** must never start or read an automatic draft. The pilot skill's rules
  for `pilot author` apply to the watcher too.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Auto-draft MUST be off by default and turned on per target with
  `PILOT_AUTHOR_AUTO=on` in the target profile.
- **FR-002**: A pilot watcher on the operator's machine MUST poll each auto-draft target,
  reading the same readiness rules as `pilot gate --target`, and start a draft only for items
  that are ready for drafting, one at a time.
- **FR-003**: The draft MUST run as its own Kubernetes Job, in its own namespace on the local
  cluster:
  - OpenCode at the pinned version, with the test-author skill as its instructions;
  - inputs staged as for a build worker: the spec, plan, tasks, criteria and a read-only
    snapshot of the project;
  - no repository credential.
- **FR-004**: The author namespace MUST have its own egress proxy, allowing only the model hosts.
  It MUST have no route to or from worker, grader or publisher namespaces, and no storage shared
  with them (C-07, C-08).
- **FR-005**: The draft MUST leave the Job only through framed log output. The pilot then runs
  `author check` on it, unchanged from 018. Only a passing check opens a pull request, through
  `author submit` and the identity of the target's level (018).
- **FR-006**: Every automatic draft MUST record evidence like a manual one (018), plus:
  - the trigger (`auto`) and the watcher's decision;
  - the author Job's isolation check results;
  - the Job's token use.
- **FR-007**: A draft MAY run a second author Job when its check fails, and that Job is given the
  failure. After its attempts (`PILOT_AUTHOR_AUTO_ATTEMPTS`, default 2), a failed draft MUST
  record its reason code. It MUST NOT be drafted again automatically until the item's spec
  changes or the operator starts a draft by hand. *(Amended in planning, 2026-11-19: research
  R5.)*
- **FR-008**: The pilot skill and the operator docs MUST say the driving session never starts
  or reads an automatic draft.
- **FR-009**: Manual drafting (018) MUST keep working unchanged.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: On a laptop cluster, a new item of a real target goes from "spec and tasks
  merged" to an open companion pull request with no operator action. That PR passes the same
  check a manual draft passes.
- **SC-002**: In every automatic draft, the author's namespace holds no SCM credential and
  reaches only the model hosts. Its isolation checks pass 100% of the time.
- **SC-003**: No automatic draft ever opens a pull request without a passing check. No item is
  drafted twice for the same spec version.
- **SC-004**: With auto-draft off, 0 drafts start.
- **SC-005**: A run of that item afterwards behaves exactly as with a manual draft: baseline,
  attempts, P11, and audit 11/11 after `down`.

## Assumptions

- The local cluster (kind or minikube) is up while the watcher drafts. Without it, drafting
  waits.
- OpenCode's headless mode can run the test-author skill to completion without a human. The
  model is the operator's worker model (today OpenCode Go).
- The watcher is a long-running pilot process the operator starts, like `pilot github watch`.
  Running it as a system service is out of scope.
- The GitHub Actions trigger and the operator trigger (011) are out of scope; 024 adds them
  later or 011 does.
