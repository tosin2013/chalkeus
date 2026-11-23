# Feature Specification: Installable release

**Feature Branch**: `013-installable-release`

**Created**: 2026-11-22

**Status**: Draft

**Input**: Roadmap entry 013 (Installable pilot release), owner go-ahead to draft the spec
(2026-11-22). Ship Chalkeus as an installable, versioned release so operators can install it
and run it against any project without cloning this repository, keeping every guarantee and
the full audit. The release is of the Go `chalkeus` binary, with `forge` as the main
subcommand (`chalkeus forge …`), after 027 (repo `tosin2013/chalkeus`). Tab completion ships
on install; the installer offers an opt-in `chk` alias with completion; the README and
install docs carry the smith brand story.

This item depends on 016, 021, 024, 026 and 027. It does not change those items' behaviour;
the installed release includes them.

## Clarifications

### Session 2026-11-22

Owner decision 2026-11-22 (Tosin Akinosho, 6:21 PM ET): all four recommendations below were
accepted.

- Q: Is Windows in the first release? → A: No. Ship Linux and macOS, amd64 and arm64, which
  is what CI already cross-builds (`.github/workflows/go.yml`, `goos: [linux, darwin]`,
  `goarch: [amd64, arm64]`). PowerShell completion is still generated (see User Story 3),
  including for a later Windows build. The first release should only claim platforms the
  project already builds; there is no Windows entry in that matrix, and laptop mode still
  stands up a kind cluster.
- Q: How does the installed binary find the target project and config? → A: Keep today's
  split, under 027 names. Operator config and target profiles live on the operator's
  machine, not in the target repository: `$CHALKEUS_CONFIG` or `~/.config/chalkeus` after
  027 (`internal/config/config.go` `Dir()` today is `$SPECFOUNDRY_CONFIG` or
  `~/.config/specfoundry`; 027 FR-005/FR-006). `--target <name>` reads `targets/<name>.env`
  (`internal/config/targets.go` `LoadTarget`, `TargetsDir`; `docs/targets.md`). The project
  itself is the GitHub repository named in that profile, not the current working directory.
  The fixture remains the default when no target is named. The installed binary MUST NOT
  need this repository's source tree. Today `paths.Root()` walks up from the working
  directory for `pilot/pins.env`, or uses `$PILOT_ROOT`, and fails with "not inside a
  specfoundry checkout" (`internal/paths/paths.go`). The release carries the pins and image
  digests the binary was built with (see the version decision), so operator commands do not
  search for a checkout. A checkout and `$PILOT_ROOT` / its 027 successor stay valid for
  maintainers (for example `up --local-build`). `docs/targets.md` already says target
  settings live on the operator's machine; the gap for 013 is only that checkout-root lookup.
- Q: Does `chalkeus` install harness ask rules, or do users set them (Q8)? → A: Users set
  them. The installer and the install docs print the exact ask (and deny) rules for
  `chalkeus forge …`, and for `chk` when that alias is enabled. The installer MUST NOT
  write or merge files in a user's harness config. Today Claude Code's ask rules are
  committed in this repository's `.claude/settings.json` (they match `pilot/pilot approve`).
  012 FR-010 (amended 2026-11-15) puts OpenCode's rule in each operator's own config,
  documented by the project, not in a committed project file. 027 already says the tool
  never edits harness config files, and that docs name the commands and prefixes to add.
  An installed binary used outside this checkout will not see `.claude/settings.json`. Q8's
  other bullets (agents in a real terminal such as Warp; deny rules for harnesses other than
  Claude Code) stay with Q8 and 008.
- Q: What is the first version, and how does the binary stay matched to its worker image
  digests? → A: The first GitHub Release is `v0.1.0`, stamped into the binary at build time
  the way `cmd/pilot/main.go` already documents (`-ldflags "-X main.version=…"`; the default
  is `dev`). Do not use the roadmap document's 1.30.x version for the binary; there are no
  product tags on this repository today. The same release ships the image digests and input
  pins from that commit (`pilot/images.lock`, written only by the image workflow's PR, and
  `pilot/pins.env`; C-09, `internal/images/images.go`). They are embedded in, or shipped
  beside and verified with, the binary, so an installed copy does not read a checkout.
  `chalkeus forge version` and `doctor` report the release version and that digest set. A
  newer lock on GitHub does not change an already-installed binary; a new release is how
  the operator gets newer images. `up --local-build` remains a checkout-only maintainer
  path (`internal/doctor/images.go`). The binary and the worker images must stay a matched
  pair (C-09).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Install from a GitHub Release and verify the checksum (Priority: P1)

As an operator, I download a published release of `chalkeus` from GitHub Releases on
tosin2013/chalkeus, verify its SHA-256 checksum against the published checksum file, and
install the binary onto my `PATH`. I do not clone this repository, and I do not install from
an OS package manager.

**Why this priority**: Without a verified install, nothing else in this item can be used.
Checksums are the only signing 013 offers (Q10; signing beyond checksums is out of scope).

**Independent Test**: On a machine without this repository, follow the install docs for one
supported platform, verify the checksum, and run `chalkeus forge version`. A deliberately
wrong checksum is refused.

**Acceptance Scenarios**:

1. **Given** a published release with a per-platform archive and a SHA-256 checksum file,
   **When** the operator follows the install docs, **Then** the checksum is checked before the
   binary is installed, and `chalkeus forge version` prints the release version.
2. **Given** a checksum that does not match, **When** the operator tries to install, **Then**
   the install is refused and no binary from that archive is left on their `PATH`.
3. **Given** the repository has been renamed to tosin2013/chalkeus, **When** someone follows a
   still-published tosin2013/specfoundry release or install link, **Then** GitHub's redirect
   works, and the install docs use the new name.

---

### User Story 2 - Run against any project without this source tree (Priority: P1)

As an operator with a working config folder and a target profile, I run the installed
`chalkeus forge` against a prepared item of my own project. I am not inside this repository.
The run keeps every existing guarantee (gate, isolation, grading, draft PR, human merge,
status change) and the full audit, including step mode, auto-draft and slice runs, which this
laptop release ships.

**Why this priority**: This is the outcome of 013: an installable release used on any
prepared project, not only from a clone of this repository.

**Independent Test**: On a machine with the installed binary and no clone of this repository,
configure a target as in `docs/targets.md`, run a prepared item (and, where relevant, a
slice, a step-mode pass, and an auto-draft), and confirm the audit.

**Acceptance Scenarios**:

1. **Given** an installed `chalkeus` and a target profile `targets/<name>.env` in the config
   folder, and no clone of this repository, **When** the operator runs
   `chalkeus forge run --target <name> --item NNN` on a prepared item, **Then** the run
   completes under every existing guarantee and the audit is complete.
2. **Given** no `--target` (or `--target fixture`), **When** they run the fixture walkthrough
   with the installed binary, **Then** it behaves as the operator guide describes.
3. **Given** the installed binary, **When** they run `doctor`, **Then** it checks that the
   cluster images match the digest set that binary was released with, and names the release
   version.
4. **Given** the binary cannot find a checkout of this repository, **When** an operator
   command runs, **Then** it still finds config and target profiles, and it still has its
   released image digests; it does not fail with a checkout-root error.

---

### User Story 3 - Tab completion is set up on install (Priority: P1)

As an operator, installing `chalkeus` also sets up tab completion for bash, zsh, fish and
PowerShell, so the long binary name is practical (owner condition, 2026-11-21).

**Why this priority**: Tab completion on install was the owner's condition for choosing the
long binary name.

**Independent Test**: Install on each supported Unix shell (bash, zsh, fish) and generate the
PowerShell script; after a new shell, `chalkeus forge <Tab>` lists subcommands.

**Acceptance Scenarios**:

1. **Given** a successful install, **When** the operator opens a new bash, zsh or fish
   session, **Then** completing `chalkeus` and `chalkeus forge` offers the subcommands.
2. **Given** the same install, **When** they ask for PowerShell completion, **Then** a
   PowerShell completion script is installed or printed, ready to use.
3. **Given** today's CLI, which does not install tab completion
   (`internal/cli/root.go` turns the default completion command off), **When** 013
   ships, **Then** completion is available for the installed binary.

---

### User Story 4 - An opt-in `chk` alias with completion (Priority: P2)

As an operator who wants a short command, I can accept an opt-in `chk` shell alias during
install. It is an alias, not a second binary. Completion works for `chk` as well.

**Why this priority**: Q7 chose `chk` as an opt-in alias so the long name stays the one docs
and agents use.

**Independent Test**: Install once accepting the alias and once declining it. Only the first
creates `chk`, and completion works for it.

**Acceptance Scenarios**:

1. **Given** the installer offers the `chk` alias, **When** the operator accepts, **Then**
   `chk forge …` runs the installed `chalkeus`, and completion works for `chk`.
2. **Given** the operator declines, or is not asked in a non-interactive install that
   defaults to no alias, **Then** only `chalkeus` is installed.
3. **Given** a `chk` command already exists on their `PATH`, **When** they would accept the
   alias, **Then** the installer says so and does not overwrite it.
4. **Given** docs, harness rules and agents, **When** they name the tool, **Then** they use
   `chalkeus`, not `chk`.

---

### User Story 5 - Install docs and the smith story (Priority: P2)

As a new operator, the README and install docs tell me how to install, verify and run
Chalkeus, and they include the smith brand story in these words: the spec is the raw metal,
the worker build is the forging, the grader is the testing and tempering, and the draft PR is
the proven piece a human inspects.

**Why this priority**: The story was an agreed part of 013's docs work (roadmap 1.28.0). It
does not replace the install steps, but it must be present and exact.

**Independent Test**: Read the README and install docs after the change; the sentence above
appears verbatim, and a new operator can install from those pages alone.

**Acceptance Scenarios**:

1. **Given** the README and the install docs, **When** they are read, **Then** they contain
   the smith story verbatim, and they describe checksum-verified install, completion, the
   optional `chk` alias, how config and targets are found, and how to add harness ask rules.
2. **Given** those docs, **When** they name the command, **Then** they use `chalkeus forge …`.

---

### Edge Cases

- **Checksum mismatch or a truncated download:** the install is refused; a partial binary is
  not left on the `PATH`.
- **Non-interactive install:** completion is set up; the `chk` alias is not, unless the
  operator passed an explicit opt-in flag.
- **`chk` already exists:** the alias is not written; the installer explains and continues
  with `chalkeus`.
- **Config folder missing or not readable only by the operator:** commands fail the way they
  do today, with the same class of error, and point at the install docs and `docs/targets.md`.
- **Both the old and new config folders, or both folder settings, exist:** 027's rules apply
  (new name wins, with a notice; the tool does not move secrets).
- **`up --local-build` with an installed binary and no checkout:** it is refused, with an
  explanation that local image builds are a maintainer path from a checkout.
- **A cluster running images from a different release:** `doctor` fails or warns as it does
  today when the cluster does not match the lock (`internal/doctor/images.go`), naming both
  this binary's release and what it found.
- **Unsupported platform (including Windows, which is not in the first release):** the
  install docs say so; no archive is published for it.
- **Older evidence and golden data:** they still audit, as 016 and 027 already require.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: A GitHub Actions workflow MUST publish a versioned GitHub Release of the
  `chalkeus` binary from the tosin2013/chalkeus repository, with a SHA-256 checksum file, in
  the same spirit as the image workflow and the way CI already installs the pinned
  `spec-status` CLI (`.github/workflows/pilot-images.yml`, `.github/workflows/go.yml`).
- **FR-002**: The first release MUST include Linux and macOS, amd64 and arm64, and MUST NOT
  include Windows (owner decision 2026-11-22). PowerShell completion is still generated.
- **FR-003**: The install docs MUST give a checksum-verified install the operator can run
  without cloning this repository. A failed checksum check MUST refuse the install.
- **FR-004**: The installed command MUST be `chalkeus forge …`. Docs, harness-rule examples
  and agent instructions MUST use that form, not a bare `forge` and not `pilot`.
- **FR-005**: An installed binary MUST run against any configured project without this
  repository's source tree, and MUST keep every guarantee and the full audit that the
  in-repository CLI has after 016, 021, 024, 026 and 027.
- **FR-006**: Operator config MUST be read from `$CHALKEUS_CONFIG` or `~/.config/chalkeus`
  (027), with `--target <name>` reading `targets/<name>.env`, as today in
  `internal/config/config.go` and `internal/config/targets.go` (owner decision 2026-11-22).
  The target project is the repository named in that profile, not the current working
  directory. The binary MUST NOT require a checkout to find those profiles or the image
  digests it was released with.
- **FR-007**: Install MUST set up tab completion for bash, zsh, fish and PowerShell, for
  `chalkeus` and `chalkeus forge`.
- **FR-008**: The installer MUST offer an opt-in `chk` shell alias, not a second binary, with
  completion wired to it. The default is off. It MUST NOT overwrite an existing `chk`. Docs
  and agents always say `chalkeus`.
- **FR-009**: The README and install docs MUST include this smith story verbatim: the spec is
  the raw metal, the worker build is the forging, the grader is the testing and tempering,
  and the draft PR is the proven piece a human inspects.
- **FR-010**: The installer and install docs MUST print the exact harness ask (and deny)
  rules for `chalkeus forge …`, and for `chk` when the alias is enabled. The installer MUST
  NOT write or merge the operator's harness config files. Users set the rules (owner
  decision 2026-11-22; 012 FR-010, 027).
- **FR-011**: The first GitHub Release MUST be `v0.1.0`, stamped into the binary at build
  time (`cmd/pilot/main.go`; owner decision 2026-11-22). The same release MUST carry the
  `pilot/images.lock` digests and the needed `pilot/pins.env` inputs from that commit, so
  the binary and the worker images stay a matched pair (C-09). `chalkeus forge version` and
  `doctor` MUST report the release version and that digest set. A newer lock on GitHub MUST
  NOT change an already-installed binary.
- **FR-012**: OS package managers (Homebrew, apt, winget and the like) are out of scope.
- **FR-013**: Signing beyond SHA-256 checksums is out of scope (Q10; revisit there).

### Key Entities

- **Release**: a GitHub Release of `chalkeus`, tagged with the version, with per-platform
  archives and a SHA-256 checksum file.
- **Installed binary**: `chalkeus` on the operator's `PATH`, used as `chalkeus forge …`,
  carrying the release version and the image digest set it was built for.
- **Opt-in alias**: `chk`, a shell alias to that binary, never a second published binary.
- **Config folder and target profile**: as 027 and `docs/targets.md`; the profile names the
  target project.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: An operator on a machine without this repository installs from the published
  release, the checksum matches, and `chalkeus forge version` prints that release's version,
  in one documented command sequence.
- **SC-002**: The same operator, with a configured target and no clone of this repository,
  runs one prepared item to a draft PR and a passing audit, with the same guarantees as a
  checkout-based run.
- **SC-003**: After install, tab completion works for `chalkeus forge` in bash, zsh and fish
  on a supported platform, and a PowerShell completion script is produced.
- **SC-004**: 100% of a checksum-mismatch attempt is refused (no matching binary left on the
  `PATH`).
- **SC-005**: The verbatim smith story appears in both the README and the install docs, and
  every install command shown there uses `chalkeus forge`.

## Assumptions

- 027 has been carried out: the repository is tosin2013/chalkeus, the in-repo command is
  `./chalkeus forge`, and the config folder, settings names, images and labels use the
  Chalkeus names, with the migration 027 already specifies.
- 016, 021, 024 and 026 are already in the binary this release ships. This spec does not
  reopen their behaviour.
- Laptop mode still needs a local cluster (kind today). The release does not remove that
  requirement.
- The internal `pilot/` folder layout can stay until a later item moves it; 027 already
  assumes that. Only the published command and names the operator meets have to be Chalkeus.
- GitHub keeps redirecting tosin2013/specfoundry URLs after the rename.
- The chalkeus.dev domain stays a separate, undecided money call and is not needed for this
  release.

## Out of scope

- OS package managers.
- Signing, notarization, attestations or provenance beyond published SHA-256 checksums (Q10).
- The chalkeus.dev domain.
- Helm, OperatorHub, Artifact Hub, and a certified catalog (Q10, after 011).
- Changing how laptop mode uses kind. Windows is not in the first release (owner decision
  2026-11-22).
- Teaching harnesses other than Claude Code and OpenCode (008, Q8).
