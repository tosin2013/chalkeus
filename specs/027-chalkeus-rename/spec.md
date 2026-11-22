# Feature Specification: Rename to Chalkeus

**Feature Branch**: `027-chalkeus-rename-spec`

**Created**: 2026-11-22

**Status**: Draft

**Input**: Roadmap 1.30.0, entry 027, and the owner's decisions of 2026-11-22. Carry out Q7 and
Q1: the product, its Go module, its command, its docs and its repository become Chalkeus,
without breaking existing setups or their evidence.

## Clarifications

### Session 2026-11-22

- Q: Are the settings' variable names and the config file names renamed in 027? → A: Yes, to
  match the subcommand: `PILOT_*` → `FORGE_*` and `pilot.env` → `forge.env`, with the old names
  still read (with a notice) until a later change removes them.
- Q: Does `pilot/pilot` keep forwarding for a while? → A: No, a hard switch: `pilot/pilot` is
  removed when `./chalkeus forge` arrives. That's safe for approvals, because an old
  `pilot/pilot approve` no longer exists to run; the new ask rules land in the same change.
- Q: Which commands must the harness ask about? → A: A command audit comes first. The repository's
  harness rules ask the human before `approve`, `reset-fixture`, `down`, `publisher-token`,
  `app-key`, `app-token`, `author`, `step`, `run` and `resume`. Today only `approve` is enforced;
  the rest are rules in the pilot skill's text. `author` and `step` are asks, not denials, so the
  separate test-author session can still run `author start/check/submit` with the human's yes.
- Q: How is the work split? → A: Audit first, then rename. The first change classifies every
  command and adds the ask rules, still on `pilot/pilot`. Later changes do the rename and the hard
  switch, after 026 merges (its branch uses `pilot/pilot`).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - An existing operator upgrades and keeps working (Priority: P1)

As an operator with a working setup (a config folder, targets, a model key, a publisher
credential, harness ask rules), I pull the renamed project. Everything keeps working, and I am
told exactly what to move and how. Nothing moves or copies my secrets for me.

**Why this priority**: A rename that breaks the owner's own setup, or that silently stops
`approve` from asking, does harm. Approvals asking the human is a safety property.

**Independent Test**: On a machine with today's setup (`~/.config/specfoundry`, the current ask
rules, the old cluster name), pull the renamed code, run the old and the new command forms, and
run an approval. Nothing fails, and the notices say what to move.

**Acceptance Scenarios**:

1. **Given** only `~/.config/specfoundry` exists, **When** any command runs, **Then** it uses
   that folder, and says once (not on every command) how to move it to `~/.config/chalkeus`,
   keeping its permissions. The tool itself moves and copies nothing.
2. **Given** an operator types the old command (`pilot/pilot <cmd>`) after the rename, **When**
   they run it, **Then** it is gone, and the release notes, `README` and docs show the new form.
   Nothing runs under the old name, including `approve`.
3. **Given** the repository's harness rules after the rename, **When** a session runs any command
   that must ask (`approve`, `reset-fixture`, `down`, the credential commands, `author`, `step`,
   `run`, `resume`) in its new form, **Then** the harness asks the human first.
4. **Given** a cluster stood up before the rename (old cluster and registry names), **When** the
   operator runs `doctor`, `up` or `down`, **Then** the tool recognizes the old cluster and says
   how to remove it. It never runs a second pilot cluster beside it unnoticed.
5. **Given** a target profile that relies on the old proposal-label default, **When** step mode
   runs, **Then** the operator is told that the default changed to `chalkeus:proposed`, and how
   to keep the old label (set it in the profile) or relabel.

---

### User Story 1a - The harness asks before every command that needs a human (Priority: P1)

As the owner, the commands that decide, destroy or hold credentials ask me through the harness,
not just through instructions an agent is trusted to follow. This lands first, on today's command,
and carries over to the new one.

**Why this priority**: Today only `approve` is enforced by a harness rule. `reset-fixture`, `down`,
`publisher-token`, `author` and `step` rely on the pilot skill's text. A rename that rewrites every
rule is the moment to make the list complete and tested.

**Independent Test**: A test lists every command and fails if one isn't classified, or if a
command classified "ask" has no matching harness rule in the repository.

**Acceptance Scenarios**:

1. **Given** the command audit, **When** a session runs `approve`, `reset-fixture`, `down`,
   `publisher-token`, `app-key`, `app-token`, `author`, `step`, `run` or `resume`, **Then** the
   harness asks the human first.
2. **Given** a new command is added later, **When** the tests run, **Then** they fail until it is
   classified as free or ask.
3. **Given** the read-only commands (`status`, `audit`, `doctor`, `gate`, `targets`, `version`,
   `github sync|drift`), **When** a session runs them, **Then** nothing asks.

---

### User Story 2 - A new operator sees only Chalkeus (Priority: P1)

As a new operator following the docs, I install and run Chalkeus. I meet one name throughout:
the command is `./chalkeus forge <cmd>`, the config folder is `~/.config/chalkeus`, the images
are `chalkeus-*`, and step mode proposes items with the `chalkeus:proposed` label.

**Why this priority**: The name was decided (Q7) so that the laptop release (013) ships under
it. Mixed names in docs and output confuse every new user.

**Independent Test**: Follow the operator guide from a clean machine to a fixture run. Every
command, path, image, label and message uses Chalkeus, except migration notes and history.

**Acceptance Scenarios**:

1. **Given** a clean machine, **When** the operator follows the docs, **Then** every command
   shown is `./chalkeus forge <cmd>` and every path is under `~/.config/chalkeus`.
2. **Given** `pilot up` in its new form, **When** it stands up the cluster, **Then** the cluster,
   registry, images and Kubernetes labels use the Chalkeus names.
3. **Given** step mode with no label set in the profile, **When** it runs, **Then** it looks for
   `chalkeus:proposed`.

---

### User Story 3 - Older evidence still audits (Priority: P2)

As the owner, every run recorded before the rename still audits as it did, and the evidence
links still resolve.

**Why this priority**: The project's claims rest on evidence. A rename that makes old evidence
fail its audit, or makes its image digests unresolvable, weakens every earlier `verified`.

**Independent Test**: Run the audit on committed evidence from before the rename (including the
golden data) and compare row by row with the results before the change.

**Acceptance Scenarios**:

1. **Given** evidence recorded with the old Kubernetes labels and image names, **When** it is
   audited after the rename, **Then** every row has the same result as before.
2. **Given** the old images, **When** an older record's digests are looked up, **Then** they
   still resolve: old images stay published.

---

### User Story 4 - The repository rename lands cleanly (Priority: P2)

As the owner, I rename the repository from tosin2013/specfoundry to tosin2013/chalkeus in
GitHub's settings on the day the code change merges. Installing from the new module path works,
old links redirect, and CI, badges, the image bot and the GitHub App keep working.

**Why this priority**: The module path and the repository name must agree, or installing from
source fails. The order is the risk, so it's planned.

**Independent Test**: After the merge and the rename, install from the new module path, open an
old link, and run the CI workflows and the image workflow once.

**Acceptance Scenarios**:

1. **Given** the code change is merged and the repository renamed, **When** someone installs
   from the new module path, **Then** it works.
2. **Given** an old repository or issue link, **When** someone opens it, **Then** GitHub
   redirects it.
3. **Given** the renamed repository, **When** the go, integration and image workflows run,
   **Then** they pass, and the image bot's PR pins `chalkeus-*` images.
4. **Given** the code change is merged but the repository is not yet renamed, **When** the owner
   checks the release notes, **Then** they say to rename it the same day, and why.

---

### Edge Cases

- **Both config folders exist:** the new one is used, and the operator is told the old one is
  still there and can be removed once they've checked it.
- **The new and old config-folder settings are both set:** the new one wins, with a notice.
  Only the old one set: it is honoured, with a notice saying what to rename.
- **A setting given under both names** (for example `FORGE_MODEL` and `PILOT_MODEL`, or both
  `forge.env` and `pilot.env` in the folder): the new name wins, and the notice names the ignored
  old one, so a stale value is never used silently.
- **An item run paused across the upgrade:** runs started before the rename are not migrated.
  The operator is told to finish them with the previous version, or reset the fixture, before
  upgrading (release notes, and a clear refusal if such a run is resumed).
- **Issues already labelled with the old proposal label** stay unproposed under the new default,
  and step mode says so, naming both labels.
- **Ask rules edited by the user:** the repository's shared rules are updated. A user's own rules
  (for example in another harness) are theirs: the docs say exactly which commands and prefixes
  to add, and the tool never edits harness config files.
- **Work in flight on the old command** (026's branch, the validation machine's agent): the
  rename lands after 026 merges, and its release notes name every change those users make.
- **Forks and clones** keep working through GitHub's redirect. Their remotes can be updated at
  leisure, and the docs say how.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The Go module MUST be `github.com/tosin2013/chalkeus`, and every import MUST use it.
- **FR-002**: The in-repository command MUST be `./chalkeus forge <cmd>`, offering every
  command `pilot/pilot <cmd>` offers today, with the same exit codes.
- **FR-003**: `pilot/pilot` MUST be removed in the same change that adds `./chalkeus forge` (a
  hard switch). Docs, skills, harness rules, CI and the integration tests MUST use the new form
  in that change.
- **FR-004**: A command audit MUST classify every command as **free** or **ask**. The repository's
  harness rules MUST ask the human before every **ask** command: `approve`, `reset-fixture`,
  `down`, `publisher-token`, `app-key`, `app-token`, `author`, `step`, `run` and `resume`. A test
  MUST fail if a command is unclassified, or if an **ask** command has no matching rule. The audit
  lands first, on `pilot/pilot`, and the rename carries it over to `./chalkeus forge`. The pilot
  skill's text MUST match the rules.
- **FR-005**: The default config folder MUST be `~/.config/chalkeus`, overridable with
  `CHALKEUS_CONFIG`.
- **FR-006**: When only `~/.config/specfoundry` exists (or only `SPECFOUNDRY_CONFIG` is set), the
  tool MUST use it and print, once per operator, how to move it with its permissions intact. The
  tool MUST NOT move, copy or read beyond today's reads in the old folder.
- **FR-007**: The step-mode proposal label default MUST be `chalkeus:proposed`. A profile that
  sets the label keeps its own.
- **FR-008**: Images MUST be published as `ghcr.io/tosin2013/chalkeus-*` from the change on. Old
  images MUST stay published.
- **FR-009**: The Kubernetes labels the pilot sets and reads MUST use `chalkeus.io/*`. The audit
  MUST accept `specfoundry.io/*` and `chalkeus.io/*` in evidence, so older records and golden
  data audit unchanged.
- **FR-010**: The kind cluster and local registry MUST use Chalkeus names. `doctor`, `up` and
  `down` MUST recognize a cluster with the old names and say how to remove it.
- **FR-011**: User-facing docs, CI badges, release and install text, and GitHub App references
  MUST use Chalkeus. Migration notes and history (past items' specs, evidence, debriefs) keep
  the old name.
- **FR-012**: The release notes for the change MUST give the owner's steps in order: merge, then
  rename the repository the same day. They MUST also give each existing operator's steps: move
  the config folder, take down the old cluster, add the new prefix to their own harness rules,
  and keep or replace the old proposal label.
- **FR-013**: The forge settings MUST be named `FORGE_*` (today's `PILOT_*`, for example
  `FORGE_MODEL`, `FORGE_APPROVERS`, `FORGE_TEST_MODE`), and the main settings file `forge.env`
  (today's `pilot.env`). The old variable and file names MUST still be read, each with a one-time
  notice naming its new name; when both are set, the new name wins and the notice says so.
  Evidence keeps recording the settings it used; the audit accepts records with either names.
  The folder setting itself is `CHALKEUS_CONFIG` (FR-005), because it is the product's, not
  forge's.

### Key Entities

- **Command form**: the old `pilot/pilot <cmd>` (removed by the rename) and the new
  `./chalkeus forge <cmd>`.
- **Command audit**: every command with its class (free or ask) and the harness rule that
  enforces an ask.
- **Config folder**: new default `~/.config/chalkeus`, old `~/.config/specfoundry`. A fallback
  and a one-time notice; no automatic move.
- **Name map**: every old name, its new name, and whether the old one is still accepted: module,
  command, config folder, settings (`PILOT_*` → `FORGE_*`, `pilot.env` → `forge.env`), images,
  labels, cluster and registry, label default, repository.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: An existing setup upgraded in place keeps working: the owner's own setup runs a
  fixture run and an approval after pulling the change, with no step beyond those in the release
  notes, and no secret moved or copied by the tool.
- **SC-002**: 100% of commands are classified, and every **ask** command (10 today) is asked
  through the harness, before and after the rename; the audit test enforces it.
- **SC-003**: 100% of committed evidence from before the change audits with the same result on
  every row as before.
- **SC-004**: User-facing docs and command output name the product Chalkeus everywhere, apart
  from migration notes and history. A check lists any old name outside an allowed list.
- **SC-005**: The day the code change merges, the repository is renamed, and the go, integration
  and image workflows pass on the renamed repository.

## Assumptions

- The internal folder layout (`pilot/` with images, manifests and fixture) stays as it is. Only
  the command and names users meet change. A later item can move folders.
- Old images aren't deleted. GitHub keeps redirecting old repository URLs after a rename.
- The project doesn't own `chalkeus.io`, which the owner accepts for label prefixes (decided
  2026-11-22). If a domain is decided later, the labels may change again under the same
  both-prefixes rule.
- The installed `chalkeus` binary, tab completion and the `chk` alias are 013's. 027 only adds
  the in-repository `./chalkeus` command.
- The work is several changes: the command audit first, on today's command; then the rename and
  the hard switch, after 026 merges.
- Runs in flight across the upgrade aren't migrated (see Edge Cases).
