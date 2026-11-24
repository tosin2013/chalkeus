# Feature Specification: Brownfield intake: tech-debt triage and dependency items

**Feature Branch**: `029-brownfield-intake`

**Created**: 2026-11-24

**Status**: Draft

**Input**: Roadmap entry 029 (added 2026-11-23, roadmap 1.33.0), from onboarding
tosin2013/documcp for 026's live slice run (friction log:
`specs/026-slice-runs/evidence/documcp-onboarding.md`). 013, the laptop release, depends on it
(roadmap 1.33.1). The roadmap scopes two parts:
- a **read-only triage report** for a target's open bot dependency PRs (Dependabot first) and
  its baseline problems. It proposes; the owner merges, batches or closes;
- **dependency items:** an owner-approved manifest change becomes an item's plan, so a worker
  can fix the code the change breaks, graded offline from a cache built from that manifest,
  with the same isolation, hidden tests and audit as any item.

Governed by C-01 (scheduling inputs), C-04 (human merge only), C-12 (humans plan, agents
build) and C-14 (hidden tests out of every build worker's reach).

## Context

What happened on documcp (2026-11-23):
- all 22 open Dependabot PRs failed one shared check: the production `npm audit`, red since new
  advisories landed on `main`. So the repository's auto-merge stalled and the pile grew;
- many of the 22 were already superseded by one `npm audit fix` on the default branch;
- a major bump that needs code changes (zod 3 → 4, which changes every MCP tool's published
  input schema) has no path through forge. The grader installs offline from a cache built from
  the **unchanged** manifest, so an attempt can't change dependencies (docs/targets.md
  "Dependencies"), and a dependency change can't be graded.

So an owner with a brownfield repository has two problems forge doesn't help with today:
1. **seeing the pile clearly:** which bot PRs share one root cause, which are already
   superseded, which are safe, and which need code changes;
2. **getting a dependency change that breaks code built and graded** under forge's guarantees.

Terms:
- **Bot PR:** an open pull request opened by a dependency bot (Dependabot in this item).
- **Manifest files:** the files that declare dependencies and pin their resolved versions for
  the target's toolchain (for example `package.json` and `package-lock.json` for `node24`).
- **Approved manifest change:** a change to manifest files only, at one exact commit, that the
  owner approved as an item's plan.
- **Dependency item:** a prepared roadmap item of the target whose plan names an approved
  manifest change.
- **Baseline flag:** a problem in the target's default branch that would make a forge run fail
  or mislead (for example red CI or a flaky test).

## Clarifications

### Session 2026-11-24

- Q: Which GitHub credential does the triage report use to read the target's PRs and checks? →
  A: The target's existing GitHub credential that forge already holds for the GitHub loop
  (token or App), used only for reads. No new secret; the report can read only what forge can
  already read (FR-008).
  **Corrected 2026-11-24 (owner):** the publisher's token is a write credential, so the report
  never uses it. The owner normally adds the repository to the GitHub App: in App mode the
  report mints its own installation token restricted to read permissions, held in memory only;
  without the App it uses forge's existing read client (`GH_TOKEN`, `GITHUB_TOKEN`, then
  `gh auth token`), as the GitHub loop's reads already do.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - See the bot-PR pile grouped by root cause (Priority: P1)

As the owner of a repository with many failing bot PRs, I run one read-only command for a
target and get a report that groups the open bot PRs by the checks they share failing, marks
each PR as superseded, safe or needing code changes, and says why. I decide what to merge, batch
or close; forge changes nothing.

**Why this priority**: it is the first thing a brownfield owner needs, it needs no cluster and
no model, and it explains on its own why a pile like documcp's stalls.

**Independent Test**: on a recorded snapshot of a repository's open bot PRs and check results
(documcp's 22 PRs of 2026-11-23, or a made-up fixture), the report puts every PR that fails only
the shared `npm audit` check in one group, marks the PRs whose bumped version the default
branch's lockfile already has as superseded, and makes no write call to GitHub.

**Acceptance Scenarios**:

1. **Given** a target with open Dependabot PRs whose failing checks are the same, **When** the
   owner runs the report, **Then** those PRs appear in one group headed by the shared failing
   check, with a count.
2. **Given** a bot PR that bumps a package to a version the default branch's lockfile already
   resolves to (or newer), **When** the report runs, **Then** that PR is marked `superseded`,
   naming the default branch's version.
3. **Given** a bot PR whose checks all pass and whose bump is a patch or minor level, **When**
   the report runs, **Then** it is marked `safe`.
4. **Given** a bot PR with a major-level bump, or with a failing check that other PRs don't share,
   **When** the report runs, **Then** it is marked `needs code changes`, and the report suggests
   it as a dependency item (User Story 2).
5. **Given** any run of the report, **Then** forge makes no write to GitHub: no comment, label,
   merge, close, approval or re-run.

---

### User Story 2 - Build and grade a dependency change that breaks code (Priority: P1)

As the owner, I approve a manifest change (for example a bot PR's branch, or a bump I prepared)
as the plan of a prepared item. A forge run applies it, fills the dependency cache from the
approved manifest, lets the worker fix the code the change breaks, and grades the result offline
against my hidden tests. The draft PR contains the approved manifest change and the code fixes,
and I merge it as usual.

**Why this priority**: it is the part nothing else covers. Without it, a major bump like
zod 3 → 4 has no governed path.

**Independent Test**: on a small fixture target whose test fails after a planned major bump of
one dependency, a dependency item with that bump as its approved manifest change runs end to
end on kind: the attempt fixes the code, the grader passes offline from the cache built from the
approved manifest, the draft PR contains the manifest change byte for byte plus the fixes, and
the audit passes.

**Acceptance Scenarios**:

1. **Given** a dependency item whose plan names an approved manifest change at one commit,
   **When** the run starts, **Then** the mode approval shows the manifest change (files, the
   packages and versions it changes, and the commit) before any worker starts.
2. **Given** that item, **When** the dependency cache is filled, **Then** it is filled from the
   base snapshot with the approved manifest change applied, plus the hidden tests, through the
   same registry-only proxy as today.
3. **Given** the worker's change bundle, **When** it is graded, **Then** the manifest files in the
   result equal the approved manifest change exactly; any other change to them fails the
   allowed-paths check (G2) with a code that names the file.
4. **Given** the approved change breaks visible tests at the base, **When** the baseline is
   recorded, **Then** it is recorded with the approved change applied, and those failures are
   the work: the grader requires the visible and hidden tests to pass afterwards.
5. **Given** a dependency item, **Then** worker isolation, secrets handling, the hidden tests'
   reach and the human merge are exactly as in any item run, and the same isolation checks and
   audit rows pass.

---

### User Story 3 - Baseline problems flagged before the first run (Priority: P2)

As the owner, the same report tells me about problems in my default branch that would make a
forge run fail or mislead, so I can fix them before I prepare an item.

**Why this priority**: each flag saves a failed or misleading run, but the owner can work
without them.

**Independent Test**: on a fixture repository with a red default-branch check, a test that
writes into the working tree, and a test command that forces exit, the report lists each as a
baseline flag with where it was found.

**Acceptance Scenarios**:

1. **Given** the default branch's latest checks include a failing one, **When** the report
   runs, **Then** it flags `red default branch` naming the check.
2. **Given** a test command with a force-exit style workaround (for example `--forceExit`), or
   install scripts the package manager held back for approval, **When** the report runs,
   **Then** each is flagged with the file and line where it was found.
3. **Given** no problem is found, **Then** the report says so; it never claims the baseline is
   healthy beyond the checks it ran.

---

### User Story 4 - Brownfield onboarding docs (Priority: P3)

As a new owner with an existing repository, the docs tell me what to clean before my first run
and how: running the report, acting on it, preparing a dependency item, and the onboarding
fixes the documcp friction log found.

**Why this priority**: needed for the release, but it follows the behaviour above.

**Independent Test**: the docs check passes, and a reader can go from an existing repository to
the report and a prepared dependency item using only the docs.

**Acceptance Scenarios**:

1. **Given** the targets and greenfield docs, **When** this item merges, **Then** they cover the
   brownfield path and fix friction entries F1, F2, F4 and F6 of the documcp log (F3 and F5 as
   notes).

---

### Edge Cases

- A bot PR from a bot other than Dependabot, or from an ecosystem with no catalog toolchain:
  listed under `not triaged` with the reason, never guessed.
- A bot PR whose head commit has no check results yet: marked `pending`, not `safe`.
- A bot PR that changes files other than manifest files: it can't be an approved manifest
  change as is; the report says so.
- Two bot PRs that bump the same package to different versions: the older is marked
  `superseded` by the newer.
- The approved manifest change's commit is not reachable from the target, or touches a
  non-manifest file: the gate refuses the item with a code, before any cluster work.
- The approved manifest change can't be installed (a version that doesn't exist, a registry not
  in `FORGE_REGISTRY_HOSTS`): the prefetch fails, the run stops with the existing prefetch
  failure, and no worker starts.
- The worker tries to add or change a dependency beyond the approved change: the grader's
  offline install fails or G2 fails; never a silent pass.
- The report's credential can't read check results or advisories: those columns say
  `unknown`; the report never fails open to `safe`.

## Requirements *(mandatory)*

### Functional Requirements

**Triage report (US1, US3)**

- **FR-001**: Forge MUST provide a read-only command that produces a triage report for one
  target's open bot PRs, as text for the owner and as JSON.
- **FR-002**: The report MUST group bot PRs by the set of checks failing on each PR's head
  commit, and name each group by its shared failing checks.
- **FR-003**: The report MUST classify each bot PR as exactly one of `superseded`, `safe`,
  `needs code changes`, `pending` or `not triaged`, with the reason (rules in the edge cases and
  acceptance scenarios).
- **FR-004**: The report MUST show for each PR the package, the from and to versions, the
  semver level of the bump, and the advisories it fixes when the credential can read them
  (`unknown` otherwise).
- **FR-005**: `superseded` MUST be decided from the default branch's own manifest files (the
  resolved version there is the bumped version or newer), not from the PR's title alone.
- **FR-006**: The report MUST list baseline flags: a failing check on the default branch's
  latest commit, a force-exit style workaround in the test command or test configuration, and
  install scripts held back for approval; each with where it was found.
- **FR-007**: The report MUST NOT write to GitHub or to the target in any way, MUST NOT run the
  target's code or tests, and MUST work without a cluster or a model.
- **FR-008**: The report MUST read with a read-only credential and make read calls only: in
  App mode, an installation token for the target repository restricted to read permissions,
  held in memory only; otherwise forge's existing read client. It MUST NOT load the publisher's
  write credential, MUST NOT print any credential, and adds no new secret.

**Dependency items (US2)**

- **FR-009**: An item's plan MUST be able to name an approved manifest change by an exact commit
  and the manifest files it changes; that is what makes the item a dependency item.
- **FR-010**: The gate MUST refuse a dependency item, with a code, when the commit can't be
  resolved, when its change touches a file that is not a manifest file of the target's
  toolchain, or when the manifest files named don't match the change.
- **FR-011**: The mode approval MUST show the approved manifest change (commit, files, packages
  and versions) before any worker starts, and the run's evidence MUST record it.
- **FR-012**: The run MUST apply the approved manifest change to the base snapshot before the
  baseline, the worker and the prefetch, so that all three see the same manifest.
- **FR-013**: The dependency cache MUST be filled from the base snapshot with the approved
  change applied, plus the hidden tests, through the target's registry-only proxy, as for any
  item.
- **FR-014**: G2 MUST allow the named manifest files only with exactly the approved content;
  any other difference in them fails G2 with a code naming the file.
- **FR-015**: The draft PR MUST contain the approved manifest change and the worker's code
  changes, so that merging it leaves the default branch consistent.
- **FR-016**: Worker isolation, the hidden tests' reach (C-14), secrets handling, the human
  merge (C-04) and every isolation check and audit row MUST be unchanged for dependency items.
- **FR-017**: Forge MUST NOT merge, close, approve, label or comment on bot PRs, and MUST NOT
  create manifest changes itself (C-04, C-12; roadmap scope out).

**Docs (US4)**

- **FR-018**: The docs MUST describe the brownfield path (the report, acting on it, preparing a
  dependency item) and fix friction entries F1, F2, F4 and F6 of the documcp log.

### Key Entities

- **Triage report:** for one target at one time: the default branch commit it read, the
  groups, each bot PR's classification and reason, the baseline flags, and what it couldn't
  read.
- **Bot PR entry:** number, bot, package, from and to versions, semver level, failing checks,
  advisories (or `unknown`), classification, reason.
- **Approved manifest change:** commit, manifest files, the packages and versions it changes;
  recorded in the item's plan and in the run's evidence.
- **Baseline flag:** kind, where it was found, and the detail.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: On documcp's 22 Dependabot PRs of 2026-11-23 (as a recorded snapshot), the report
  puts every PR that fails only the shared `npm audit` check in one group, and marks every PR
  already superseded by the default branch's lockfile as `superseded`.
- **SC-002**: Over every report run in the tests, forge makes zero write calls to GitHub.
- **SC-003**: A fixture dependency item (a major bump that breaks a visible test) runs end to
  end on kind: graded offline, the draft PR contains the approved manifest change byte for byte
  plus the fixes, and the audit passes as for any item.
- **SC-004**: A worker change to a manifest file beyond the approved change fails G2 in 100% of
  the test cases.
- **SC-005**: A live dependency item on a real target (documcp's zod 3 → 4 is the candidate)
  reaches a merged PR with its hidden tests passing, and the owner records it.
- **SC-006**: A new owner can go from an existing repository to the report and a prepared
  dependency item using only the docs.

## Assumptions

- Dependabot only in this item; other bots appear as `not triaged`. Ecosystems are those with a
  catalog toolchain (`node24`, `python3.13`, `go`, `jdk21-maven`); manifest files are defined
  per toolchain in the catalog.
- "Safe" uses the PR's existing check results and its semver level only; the report never runs
  the PR's code. An owner who wants more runs the PR through CI or a dependency item.
- An approved manifest change combines as many bumps as the owner wants (batching is the
  owner's choice); one dependency item names one approved change.
- When the dependency item's PR merges, the bot closes its own superseded PRs as it does today;
  forge doesn't touch them.
- Hidden tests for a dependency item guard what must not change (for example a snapshot of a
  tool's public input schemas), written as for any item (018, 024).
- The report is a laptop-mode command first; cluster mode (011) can call it later.
