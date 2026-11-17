# Feature Specification: Any project, any item

**Feature Branch**: `014-any-project`

**Created**: 2026-11-17

**Status**: Draft

**Input**: User description: "Next roadmap item after 016 and 004 were verified (owner, 2026-11-17): 014 Any project, any item (roadmap entry 014). Run one chosen, prepared roadmap item of a real repository under every existing guarantee; per the delivery order, on kind with the fixture and a real repository first, with the OpenShift EAP validation later (after 005)."

## Background

Today the pilot runs only its built-in fixture. Two items are wired in by hand: A is 001 and B is
002, each with a fixed spec folder. The hidden acceptance tests are files in this repository
written for those two items. The worker and grader images carry one toolchain (Python with its
test runner). The only outside host the worker may reach is the model provider. The fixture
proved the guarantees: the fail-closed gate, isolation, independent grading, a draft PR, the
human merge, the status change and the audit. But nobody can point the pilot at their own
project yet.

This feature lets a human run **one prepared item of a real repository**. The guarantees stay
exactly as they are. Under the project's rule that humans plan and agents build (C-12), the
human has already written the item's spec and tasks, verified its dependencies, and written
hidden acceptance tests. The pilot only builds what was prepared, and refuses anything else.

The roadmap's delivery order puts this feature first on a laptop (kind): the fixture stays the
regression test, and one real repository proves the general path. The enterprise validation, a
legacy Java EE upgrade on OpenShift with an in-cluster model, comes later. It needs 005
(OpenShift target) and is outside this feature's first delivery.

Brownfield only: the repository already has code and tests. Starting from an empty repository
is roadmap item 017.

## Clarifications

### Session 2026-11-17

- Q: Where are a project's toolchain, package registries and test command declared? → A: Only
  in operator-side configuration, never in the target repository, so no agent change or merged
  PR can widen the worker's egress. This also works for greenfield (017): the operator sets the
  values before the scaffold runs. *(Refined in the next answer: per-target profiles.)*
- Q: Does the configuration support more than one repository, and what happens in Kubernetes?
  → A: Yes, through per-target profiles. Shared settings stay in `pilot.env`. Each target
  repository gets a profile (`targets/<name>.env` beside it) holding its repository, toolchain,
  test command, registry hosts, companion repository and, in token mode, its publisher
  credential. The operator picks one with `--target <name>`. Without `--target`, today's
  single-repository `pilot.env` works unchanged, and the fixture is one target among others. In
  cluster mode (011), the same fields become a namespaced cluster resource, changeable only by
  people the cluster's access rules allow, never by worker or grader pods.
- Q: Which status does the status-change PR propose after the human merges a real item's PR?
  → A: `verified`, as on the fixture. The evidence is grading (the repository's suite and the
  hidden tests) plus the audit. The human's merge of the status-change PR, or their approval in
  the terminal, is the human review, and dependent items unblock immediately.
- Q: How does the pilot know which files a real item's change may touch? → A: The human
  declares an "Allowed paths" list (globs) in the item's spec when preparing it. An item without
  one is not prepared. The item's own `tasks.md` is always allowed, so the worker can tick its
  boxes.
- Q: How does the grader run an item's hidden acceptance tests? → A: As an overlay. The item's
  folder in the companion repository mirrors the project's layout (for Java, test classes under
  `src/test/java/...`). The grader runs the project's test command on the changed project (the
  visible check), then copies the folder over it and runs the same command again (the hidden
  check). The fixture works the same way.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Run a chosen item of my repository (Priority: P1)

An operator has a real repository prepared with Spec Kit: a roadmap, a spec and tasks for item
NNN, the item's dependencies verified, and hidden acceptance tests written. They configure the
pilot for that repository and ask it to run item NNN. The pilot then does what it does for the
fixture:
- it checks the gate;
- it shows the run mode for approval;
- it runs isolated worker attempts, with retries only on approval;
- it grades each attempt independently, with the repository's own tests and the hidden ones;
- it opens a draft PR;
- after the human merges, it proposes the status change.

The audit passes.

**Why this priority**: this is the feature. Without it, specfoundry only works on its own demo.

**Independent Test**: on the chosen real repository, run one prepared item from start to
finish and check that the audit passes every check that applies to a single-item run.

**Acceptance Scenarios**:

1. **Given** a configured repository whose item NNN is prepared, **When** the operator runs
   item NNN, **Then** the gate allows it, the mode is shown for approval, and only then does a
   worker start.
2. **Given** an attempt whose change fails the hidden acceptance tests, **When** grading ends,
   **Then** the attempt fails, and a further attempt waits for the human's retry approval
   (within the retry limit).
3. **Given** an attempt that passes every grading check, **When** grading ends, **Then** a
   draft PR is opened by the publisher identity, linked to the item, and the run waits for the
   human merge.
4. **Given** the human merged the PR, **When** the run resumes, **Then** it proposes the item's
   status change to `verified` for approval, as on the fixture. Only that one line of the
   target repository's roadmap changes.
5. **Given** the run ended, **When** the audit runs, **Then** every check that applies passes.
   The audit does not assume the fixture's two items or its planned first failure.

---

### User Story 2 - The pilot refuses an item that is not prepared (Priority: P1)

An operator asks for an item that isn't ready. The spec or tasks may be missing, a dependency
may not be verified, the spec may still have open questions, no hidden acceptance tests may
exist, or the spec may not say which files the item may change. The pilot refuses before anything starts, says why, and records the refusal.

**Why this priority**: C-12 (humans plan, agents build) and the fail-closed gate are
non-negotiable. A general runner that builds unprepared items breaks the project's core rule.

**Independent Test**: for each readiness rule, prepare an item that breaks only that rule; the
run refuses with that reason and stages no worker.

**Acceptance Scenarios**:

1. **Given** item NNN has no spec or no tasks, **When** the operator runs it, **Then** the
   pilot refuses with a reason naming the missing artifact.
2. **Given** item NNN depends on an item that is not verified, **When** the operator runs it,
   **Then** the pilot refuses, naming the dependency.
3. **Given** item NNN's spec still has unresolved clarification markers, or the roadmap links
   it to an open question, **When** the operator runs it, **Then** the pilot refuses.
4. **Given** item NNN has no hidden acceptance tests, **When** the operator runs it, **Then**
   the pilot refuses. It never grades with the visible tests alone.
5. **Given** the item's status cannot be read (status tool error, malformed roadmap), **When**
   the operator runs it, **Then** the pilot refuses (fail closed).
6. **Given** item NNN's spec has no "Allowed paths" list, **When** the operator runs it,
   **Then** the pilot refuses, naming the missing list.

---

### User Story 3 - The worker gets the toolchain and dependencies my project needs (Priority: P2)

The repository is not written in Python, or it needs packages from its ecosystem's registry.
The operator declares, in their pilot configuration, which supported toolchain the repository
needs, which package registries its build must reach, and the command that runs its tests. The worker and the grader run with that toolchain. The worker can install
dependencies from the declared registries only, and the operator sees those hosts when
approving the run mode.

**Why this priority**: real repositories use other languages and fetch dependencies. Without
this, only Python projects with no dependencies could run. It comes second because the first
real repository can be chosen to fit the existing toolchain while this is built.

**Independent Test**: run an item of a repository that declares a toolchain other than the
fixture's and one package registry. The build installs its dependencies and passes. A request
to any other host is refused and recorded.

**Acceptance Scenarios**:

1. **Given** a configuration that declares a supported toolchain, **When** an attempt runs,
   **Then** the worker and the grader use that toolchain's published, digest-pinned images.
2. **Given** a configuration that declares package registries, **When** the mode is shown for
   approval, **Then** the operator sees the model host and each registry host the worker
   will be allowed to reach.
3. **Given** a worker that tries to reach an undeclared host, **When** the request is made,
   **Then** it is refused, and the refusal appears in the attempt's evidence.
4. **Given** a configuration that declares a toolchain with no published image, **When** the
   operator runs an item, **Then** the pilot refuses. It names the toolchain and points to how
   to request an image (a GitHub issue form on the specfoundry repository).

---

### User Story 4 - The fixture keeps working (Priority: P2)

After this change, the fixture run behaves as before: A's planned first failure and the retry,
B refused until A is verified, and the audit at 11/11. All committed evidence still passes the
audit.

**Why this priority**: the fixture is the fast, deterministic regression test (roadmap note).
Generalising must not weaken it.

**Independent Test**: one fixture run end to end with the audit at 11/11; the golden audit
over every committed record is unchanged.

**Acceptance Scenarios**:

1. **Given** the fixture configuration, **When** the operator runs the pilot as the operator
   guide shows, **Then** the commands, prompts, exit codes and audit result are unchanged.
2. **Given** the committed evidence records, **When** the audit runs, **Then** every result
   matches the recorded expectations.

---

### Edge Cases

- **The item is already `verified` or `implemented`:** the gate refuses, as for the fixture's
  B before A.
- **Item number not on the roadmap, or superseded:** refused, naming the item.
- **Hidden tests pass on the unchanged code:** the item can't tell a correct change from no
  change. The run refuses before staging a worker; the pilot runs the hidden tests against the
  base revision first.
- **The repository's own test suite already fails on the base revision:** refused, with the
  failing check named. The suite is a visible check, so it must pass before the change.
- **The change touches files outside the paths the item's tasks allow:** the attempt fails
  grading, as on the fixture.
- **A registry host is also a general-purpose host (for example a code host):** allowed only
  if declared and shown at the mode approval. The operator decides.
- **The repository name doesn't match the configured target:** the repo guard stops the run.
  The fixture-only naming rule is replaced by the configured targets: a repository with no
  profile is never pushed to.
- **A large repository:** clone and bundle sizes are limited, and the limit is shown in the
  error.

## Requirements *(mandatory)*

### Functional Requirements

**Choosing and checking the item**

- **FR-001**: The operator MUST be able to name the item to run (for example `pilot run --item
  NNN`). The item MUST come from the target repository's roadmap and its status, read through
  the status tool, not from a built-in list.
- **FR-002**: Before any worker starts, the pilot MUST check readiness, and MUST refuse with a
  reason if any rule fails:
  - (a) the item's spec and task list exist;
  - (b) every dependency is recorded `verified`;
  - (c) the item is not yet `implemented` or `verified`;
  - (d) the item's spec has no unresolved clarification markers, and no roadmap open question is
    linked to it;
  - (e) hidden acceptance tests exist for the item;
  - (f) the item's spec declares an "Allowed paths" list.
- **FR-003**: Status that cannot be read MUST refuse (fail closed, C-02), as today.
- **FR-004**: The pilot MUST only push to repositories the operator configured as targets.
  Any other repository stops the run with the existing guard exit code.
- **FR-004a**: The operator MUST be able to configure several target repositories, one profile
  each, and choose one per command (`--target <name>`). Shared settings (model, approvers,
  publisher mode) stay in `pilot.env`. With no `--target`, the single-repository `pilot.env`
  MUST keep working exactly as today. Paused runs MUST resume against the target they started
  with.
- **FR-004b**: A target profile's fields MUST be defined so that cluster mode (011) can hold the
  same fields in a cluster resource with the same meaning. No field may depend on the
  operator's laptop, except the paths of local secret files.

**Building and grading**

- **FR-005**: The worker's prompt MUST be built from the item's own spec and tasks, and from the
  item's "Allowed paths" list. Grading's allowed-paths check MUST use exactly that list plus the
  item's own `tasks.md`.
- **FR-006**: Grading MUST run, in an isolated grader that the worker cannot reach or write to:
  - the existing checks: change applies, allowed paths, tasks checked off;
  - the visible check: the repository's test command on the changed project;
  - the hidden check: the same test command after the item's companion folder is copied over
    the changed project, mirroring its layout. Any failure fails the attempt.
- **FR-007**: Before the first attempt, the pilot MUST run the hidden acceptance tests and the
  repository's suite against the base revision. It MUST refuse if the hidden tests already pass
  (no signal) or the suite already fails.
- **FR-008**: Hidden acceptance tests MUST never enter the worker's namespace, the worker's
  prompt, or the PR. They MUST live in a **private companion repository** per
  project (for example `<project>-acceptance`), with one folder per item. The pilot, on the
  operator's machine, reads the item's folder with the operator's own access and mounts only
  that folder into the grader's namespace, as it mounts the fixture's hidden tests today. So
  the grader still holds no SCM credential (AU04). The publisher never pushes to the companion
  repository, and the worker's egress can't reach it (owner decision 2026-11-17).

**Toolchains and dependencies**

- **FR-009**: The target's profile MUST name one toolchain from
  a list the pilot supports, and the command that runs the repository's own tests. The target
  repository declares nothing; these values are never read from it.
  The worker and the grader MUST then use that toolchain's images: built by the image workflow,
  pinned by digest, hardened like today's images (C-08, C-09).
- **FR-010**: The target's profile MUST name the package registry hosts the build needs.
  The worker's egress MUST allow exactly the model host(s) plus those registries, and the run
  mode shown for approval MUST list them.
- **FR-011**: An undeclared toolchain, or one with no published image, MUST refuse the run. The
  refusal names the toolchain and points to the image request issue form.
- **FR-012**: The specfoundry repository MUST offer a GitHub issue form for requesting a new
  toolchain image (roadmap note, 2026-11-16).

**Evidence and audit**

- **FR-013**: Evidence MUST be recorded per item, under the same record layout. The audit MUST
  check a single-item run without assuming items A and B or the fixture's planned first
  failure:
  - the gate allowed only that item;
  - every attempt was within the caps and the retry limit, and each retry was approved;
  - the mode was approved before staging;
  - the last attempt passed grading;
  - the PR was opened by the publisher identity;
  - no SCM credential reached the worker or the grader;
  - the teardown was verified;
  - no secrets are in the evidence.
- **FR-014**: The fixture's checks that only make sense for the fixture (the B-refused-until-A
  order, the A/1 planned failure) MUST still apply to fixture runs, and MUST NOT apply to
  real-item runs.
- **FR-015**: The fixture run MUST keep its commands, prompts, exit codes and 11/11 audit, and
  the audit of committed evidence MUST be unchanged.

**Approvals and GitHub**

- **FR-016**: Every human decision the fixture run has (mode, retry, status change after the
  merge) MUST exist for a real item too. It MUST be usable from the terminal, the agent harness
  and GitHub, as today. The GitHub loop (item issues, labels, PR linking, drift) MUST work
  against the real repository's roadmap.
- **FR-017**: After the human merges the item's PR, the status change MUST propose `verified`
  for that item only, changing one roadmap line, and MUST be approved by a human (terminal,
  harness, or merge of the status-change PR) before it is recorded.

### Key Entities

- **Target repository:** a real repository the operator allows the pilot to push to, with its
  roadmap, specs, tasks and own test suite.
- **Target profile:** the operator's per-repository settings: repository, toolchain, test
  command, registry hosts, companion repository and (token mode) publisher credential. A file
  beside `pilot.env` on a laptop; a cluster resource in cluster mode (011).
- **Prepared item:** a roadmap item that passes every readiness rule (FR-002), including an
  "Allowed paths" list in its spec; the only thing the pilot will build.
- **Hidden acceptance tests:** a human-written test set per item, kept in the project's private
  companion repository in a folder that mirrors the project's layout. The grader runs them; the worker never sees them.
- **Companion repository:** a private repository holding a project's hidden acceptance tests,
  one folder per item. The operator can read it; the pilot passes one item's folder to the
  grader. Neither the worker nor the publisher can reach it.
- **Toolchain:** a supported language environment with published worker and grader images,
  named in the operator's configuration.
- **Registry allowlist:** the package registry hosts the operator declares for a repository, added to the
  worker's egress for that run and shown at the mode approval.
- **Single-item run:** one run of one prepared item, with its evidence and audit.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: One prepared item of the chosen real repository runs from start to finish on a
  laptop: draft PR opened, merged by the human, status change approved. The audit passes 100% of
  the checks that apply.
- **SC-002**: For each readiness rule, an item that breaks only that rule is refused before any
  worker starts: 6 of 6 rules (FR-002 a–f), plus unreadable status.
- **SC-003**: 0 hidden acceptance test files or their contents appear in any worker namespace,
  worker prompt, PR, or worker-visible evidence across the validation runs.
- **SC-004**: A worker request to a host outside the model and declared registries is refused
  and recorded: 100% of attempted undeclared requests in the validation runs.
- **SC-005**: The fixture run still ends 11/11 with 0 changes to commands, prompts or exit
  codes, and the audit of all committed evidence is unchanged.
- **SC-006**: An operator who has prepared a repository can start the item's run from the docs
  alone, in under 15 minutes of setup beyond the fixture's, with 0 outside lookups.
- **SC-007**: With two targets configured (the fixture and the real repository), the operator
  runs each by name with 0 configuration edits between runs, and each run's evidence records
  which target it used.

## Assumptions

- **First delivery is on kind (laptop).** The OpenShift EAP validation (roadmap 014 notes) needs
  005 and an in-cluster model. It is planned as a later validation of this feature, not part of
  its first delivery.
- **Brownfield only:** the repository has code and a test suite that passes on the base
  revision. Greenfield is 017.
- **One item per run, run by a human:** parallel runs are 015, choosing items automatically is
  out of scope, and writing specs or plans is out of scope (C-12).
- **The human prepares the repository with Spec Kit:** constitution, roadmap, spec, tasks, and
  the hidden acceptance tests. The status tool reads its roadmap as it does the fixture's.
- **Allowed paths are written by the human** in each item's spec (a list of globs), as part of
  preparing the item.
- **Toolchains:** the supported list starts with the fixture's (Python) and JDK/Maven. More are added through the image request form.
- **The validation repository (owner decision 2026-11-17):** a small public **Java/Maven**
  service with JUnit tests, chosen at planning time and copied into an owner-controlled
  repository, prepared with a few dependent items. Its toolchain (JDK and Maven) and its
  registry (Maven Central) are the first beyond the fixture's. They are also what the later EAP
  validation on OpenShift needs.
- **Publishing and GitHub:** both publisher modes (token and GitHub App) and the GitHub loop work
  for the real repository as they do for the fixture. The App must be installed on each target
  repository; in token mode, each target's profile names its own fine-grained token.
- **The run's limits stay as today:** spend cap, token cap, deadline and retry limit; the mode
  approval may show different values per repository.
- **Fixture refusal on reset:** `reset-fixture` stays fixture-only. A real repository is never
  force-reset by the pilot.
