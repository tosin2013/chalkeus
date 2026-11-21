# Feature Specification: Integration tests in CI

**Feature Branch**: `025-ci-integration`

**Created**: 2026-11-21

**Status**: Draft

**Input**: Roadmap entry 025 (owner decisions 2026-11-19 and 2026-11-21). Six bugs in 021 and
024 were found one live step at a time, all in the glue that unit tests with fakes can't see
(seam audit: `specs/021-step-mode/audit-20261119.md`). The owner chose:
- **GitHub:** a fake GitHub API server;
- **the model:** a stub `opencode` in a test worker image;
- **the runner:** a GitHub-hosted one, weekly and on PRs that touch the pilot.

## Context

CI runs today on every PR: unit tests (52% coverage, 15–22% in the glue packages), the Python
tests, the docs check, lint and builds. No job stands up a cluster or runs the pieces together.
The live runs on Stockroom found what CI couldn't:
- a command that didn't load images;
- a check looking for the wrong pod label;
- a process that inherited a terminal;
- a setting that two components read differently;
- a wait shorter than the poll interval;
- a file name two components both chose.

025 adds a job that does what a live run does, but **with nothing real outside the cluster**:
a stub agent instead of a model, and a fake GitHub instead of github.com. It fails when a seam
breaks.

Terms:
- **Stub harness:** a stand-in for `opencode` in a test build of the worker image. It writes
  a canned change, or for the author a canned draft, chosen by the scenario, and prints the
  step events the spend watchdog reads.
- **Fake GitHub:** a small server, kept in this repository, that serves the API calls and git
  traffic the pilot and its publisher make. It holds the scenario's repositories, issues, labels
  and pull requests in memory, and the test acts as the owner through it (comments, labels,
  merges).
- **Test endpoints:** settings that point the pilot and its images at the fake GitHub. They
  exist only for tests.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - A broken seam fails CI before a live run finds it (Priority: P1)

As the maintainer, when a change breaks how the pilot's pieces fit together (a template label, a
command's wiring, a process setting), CI fails on the PR, naming the step and the check.

**Why this priority**: This is the feature. Each of the six live bugs would have failed this
job.

**Independent Test**: Reintroduce one of the six bugs on a branch, for example the author pod
label back to `app=worker`. The job fails at that scenario's step.

**Acceptance Scenarios**:

1. **Given** a PR that changes the pilot's Go code, manifests or images, **When** CI runs,
   **Then** the integration job stands up kind with `pilot up --local-build` (the stub harness
   in the worker image, the fake GitHub reachable) and runs every scenario.
2. **Given** the author pod label is changed back to `app=worker` in the template only,
   **When** the job runs, **Then** the automatic-drafting scenario fails with A01 named.
3. **Given** step mode's subprocesses get a terminal again, **When** the job runs, **Then** the
   step-mode scenario fails because the run didn't pause for `/approve mode`.

---

### User Story 2 - The scenarios cover the paths owners use (Priority: P1)

As the maintainer, the job runs the same paths a live validation runs, end to end, with the
evidence and audit checked.

**Why this priority**: Only the real paths reveal seams.

**Independent Test**: The job's report lists each scenario with its runs, checks and audit
result.

**Acceptance Scenarios**:

1. **Fixture:** the fixture run goes from `run` to B's draft PR, with the stub making A/1 fail
   by induction and A/2 pass. The test approves the retry and merges A's PRs through the fake
   GitHub. The audit passes, except AU07, which needs `down`.
2. **Automatic drafting:** with auto-draft on for a test target, `author watch --once` drafts
   the item's hidden tests in the author Job:
   - A01–A05 pass;
   - `author check` passes;
   - a companion PR opens on the fake GitHub.

   A draft reusing the worker's file path is refused with `DRAFT_PATH_IN_SPEC`.
3. **Step mode:** with the item labelled by an approver through the fake GitHub, `step` (one
   pass at a time):
   - drafts the tests; the test merges them;
   - runs doctor and starts the run, which pauses for `/approve mode` without asking any
     terminal; the test approves on the fake GitHub;
   - waits as the run builds, grades and opens its PR; the test merges the item PR and the
     status PR.

   The run succeeds and is recorded as started by step mode.
4. **Step mode, the debrief's gaps:**
   - a label added by a non-approver starts nothing;
   - a failing doctor produces one comment and no run;
   - a crashed `in-progress` run is handled by a stated rule rather than holding the slot.

---

### User Story 3 - The tests can't leak into real use (Priority: P1)

As the owner, the test endpoints can never be used by accident in a real run, and a run made
with them can never count as evidence.

**Why this priority**: Redirecting GitHub is a powerful setting. It must stay a test tool.

**Independent Test**: Set a test endpoint without test mode. The pilot refuses. With test mode,
the run's evidence and audit mark it as a test run.

**Acceptance Scenarios**:

1. **Given** a test endpoint is set without `PILOT_TEST_MODE=1`, **When** any pilot command
   starts, **Then** it refuses with a clear message and does nothing.
2. **Given** test mode, **When** a run finishes, **Then** its `config.json` records the test
   endpoints, and the audit adds a failing row stating it's a test run, not evidence.
3. **Given** the published worker and tools images (from `pilot/images.lock`), **When** they
   run, **Then** they contain no stub harness. The stub exists only in local test builds.

---

### User Story 4 - Coverage of the glue only goes up (Priority: P2)

As the maintainer, the glue packages' coverage is visible and can't drop unnoticed.

**Why this priority**: The audit showed the gap. A floor keeps it from coming back.

**Independent Test**: A PR that lowers `internal/cli` coverage below its floor fails CI.

**Acceptance Scenarios**:

1. **Given** each glue package (`cli`, `pipeline`, `worker`, `checks`, `cluster`, `grader`) has
   a floor in a committed file, **When** CI runs, **Then** a package below its floor fails the
   build, naming the package and both numbers.
2. **Given** coverage rises, **When** the maintainer raises the floor, **Then** that's a
   one-line change reviewed like any other.

---

### Edge Cases

- **Runner limits:** a GitHub-hosted runner has a few CPUs and limited memory, so the job keeps
  one scenario's cluster resources at a time and fits in its time limit (target: under 45
  minutes).
- **Forks:** the job needs no secrets, so it runs on PRs from forks too.
- **Flakiness:** a step that waits has a timeout and reports what it waited for. A failed job
  uploads the scenario's evidence and pod logs as artifacts (no secrets exist to leak).
- **Weekly run:** it also catches drift in base images and Kubernetes and kind versions between
  PRs.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: A CI workflow MUST run the integration job weekly and on PRs that change
  `cmd/`, `internal/`, `pilot/manifests/`, `pilot/images/` or the workflow itself, on a
  GitHub-hosted runner, with no repository secrets.
- **FR-002**: The job MUST build the images locally (`pilot up --local-build`). The worker image
  is built with the stub harness, and the published images MUST NOT contain it.
- **FR-003**: The stub harness MUST stand in for `opencode` on the same command line and print
  the step events the watchdog reads. Its output (a change, or a draft) is chosen per scenario,
  including deliberate failures: an induced A/1 failure, a colliding draft path.
- **FR-004**: A fake GitHub server in this repository MUST serve the API calls and git traffic
  the pilot, its publisher and its test author make for the scenarios: repositories, branches,
  contents, issues, comments, labels, issue events, pull requests, merges, and the App and
  token checks the pilot runs. It holds state in memory and exposes test-only calls to act as
  the owner.
- **FR-005**: Test endpoints (the GitHub API and git base addresses for the pilot and its images)
  MUST be refused unless `PILOT_TEST_MODE=1`. With test mode, runs MUST record the endpoints in
  `config.json`, and the audit MUST fail with a row stating it's a test run.
- **FR-006**: The scenarios of User Story 2 MUST run, each asserting:
  - the step outcomes;
  - the isolation check rows (P01–P12, P06r, A01–A05, as each scenario applies);
  - the evidence files;
  - the audit rows (apart from the test-run row).
- **FR-007**: A committed coverage floor for the glue packages MUST fail CI when a package
  drops below it.
- **FR-008**: On failure, the job MUST upload the scenario's evidence, `decisions.jsonl` and
  pod logs as artifacts, and print which step and check failed.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Each of the six live bugs from 021 and 024, reintroduced on a branch, makes the
  job fail at the matching scenario (4 of them by design: images, the pod label, the terminal,
  the issues setting; the timing and file-name bugs where the scenario covers them).
- **SC-002**: The job passes on `main`, weekly and on PRs, in under 45 minutes on a
  GitHub-hosted runner, with no secrets.
- **SC-003**: A test endpoint without test mode is refused in 100% of commands, and every test
  run's audit carries the test-run row.
- **SC-004**: The glue packages' coverage rises from 15–22% and is held by the committed floor.

## Assumptions

- kind runs on a GitHub-hosted Ubuntu runner (Docker is available). The pilot's images build
  there in the job's time limit.
- The fake GitHub covers only what the scenarios call, and grows with them. Real GitHub stays
  covered by the live validation runs, which still close each item.
- The model, prompts and harness behaviour aren't tested here. The stub stands in for them.
