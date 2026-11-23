# Implementation Plan: Rename to Chalkeus

**Branch**: `027-plan` | **Date**: 2026-11-23 | **Spec**: [spec.md](spec.md)

## Summary

027 renames what operators meet:
- the Go module (`github.com/tosin2013/chalkeus`);
- the command (`./chalkeus forge <cmd>`, a hard switch from `pilot/pilot`);
- the settings (`FORGE_*`, `forge.env`, old names still read);
- the config folder (`~/.config/chalkeus`, with a fallback to the old one);
- images (`chalkeus-*`), Kubernetes labels (`chalkeus.io/*`, both accepted by the audit), and
  the cluster and registry (`chalkeus-*`);
- the step-mode label default (`chalkeus:proposed`);
- docs.

It starts with a **command audit**: every command classified free or ask, the repository's
harness rules asking before the 10 ask commands, and a test that keeps it so. The owner renames
the repository the day the module change merges. Decisions: [research.md](research.md); the
name map and the audit: [contracts/](contracts/).

## Technical Context

**Language/Version**: Go (the version in `go.mod`); Python 3.12 in the images (unchanged)

**Primary Dependencies**: cobra (commands), the existing pilot packages; nothing new

**Storage**: the operator's config folder; evidence folders (read, never rewritten)

**Testing**: `go test ./...` (with the docs check), the Python image tests, and the 025
integration scenarios, which run the renamed command end to end

**Target Platform**: Linux and macOS laptops, the kind cluster, GitHub Actions

**Project Type**: a CLI with its cluster manifests and images

**Performance Goals**: none new

**Constraints**:
- old evidence and golden data audit row for row as before (SC-003);
- approvals and the other ask commands always ask (SC-002);
- the tool never moves or copies the config folder (FR-006);
- images stay digest-pinned (C-09).

**Scale/Scope**: about 1,600 old-name mentions in about 540 files. 98 Go files import the module
path; 57 distinct `PILOT_*` names; 443 `pilot/pilot` mentions; 446 image-name mentions; 103
label mentions.

## Constitution Check

| Principle | Effect | Result |
|---|---|---|
| I. Roadmap is the source of truth | 027 is on the roadmap (1.30.0); 013 depends on it | ✅ |
| II. Gates fail closed | gate logic is untouched; renamed settings fall back to the old names, never to "allow" | ✅ |
| III. Evidence defines done | old evidence audits unchanged (both label prefixes, both setting names); nothing is rewritten | ✅ |
| IV. Least privilege | the command audit moves 9 commands from skill text to harness-enforced asks | ✅ improves |
| V. Humans approve, humans merge | `approve` and the other ask commands ask through the harness; the owner renames the repository | ✅ improves |
| VI. Build on, don't rebuild | renames only; no behaviour change | ✅ |
| VII. Repeatable and portable | images still digest-pinned from `pilot/images.lock`; the image workflow publishes the new names | ✅ |

There are no violations. After the design: the same.

## Project Structure

### Documentation (this feature)

```text
specs/027-chalkeus-rename/
├── spec.md
├── plan.md              # this file
├── research.md          # R1-R8 decisions
├── quickstart.md        # how to check each stage, and the owner's rename day
├── contracts/
│   ├── name-map.md      # every old name, its new name, and whether the old one is still read
│   └── command-audit.md # every command: free or ask, and the rule that enforces it
├── checklists/requirements.md
└── tasks.md             # /speckit-tasks
```

### Source code (repository root)

```text
chalkeus                     # new shim (replaces pilot/pilot): builds and runs ./cmd/chalkeus
cmd/chalkeus/                # renamed from cmd/pilot; root command `chalkeus`, subcommand `forge`
internal/cli/                # commands move under `forge`; command-audit table and its test
internal/config/             # CHALKEUS_CONFIG and ~/.config/chalkeus, fallback, FORGE_* with PILOT_* read
internal/paths/              # cluster, context and registry names; FORGE_ROOT (PILOT_ROOT read)
internal/audit/, internal/checks/  # accept specfoundry.io/* and chalkeus.io/* labels
pilot/manifests/             # labels chalkeus.io/* (the folder layout itself stays)
.github/workflows/           # image names, badges, the new command
.claude/settings.json        # ask rules for the 10 ask commands, new form
.claude/skills/forge/        # the pilot skill, renamed and rewritten for the new form
docs/, README.md, AGENTS.md, CONTRIBUTING.md
test/integration/            # builds ./cmd/chalkeus; runs `forge`
```

**Structure decision:** the `pilot/` folder (images, manifests, fixture, grading) keeps its name.
It's internal layout, and moving it would touch every image build and path for no user benefit
(spec Assumptions).

## Stages (each a PR that can merge on its own)

1. **Command audit** (US1a, FR-004), on `pilot/pilot`:
   - a table of every command (free or ask), with a test that fails on an unclassified command,
     and on an ask command without a rule in `.claude/settings.json`;
   - ask rules for `approve`, `reset-fixture`, `down`, `publisher-token`, `app-key`,
     `app-token`, `author`, `step`, `run` and `resume`;
   - the pilot skill's text matched to the rules.
2. **Config folder and settings** (US1, FR-005, FR-006, FR-013):
   - `CHALKEUS_CONFIG` and `~/.config/chalkeus`, with the fallback and the one-time notice;
   - `FORGE_*` and `forge.env`, with `PILOT_*` and `pilot.env` still read (new wins, with a
     notice);
   - unit tests for every combination.

   Behaviour for today's setups is unchanged.
3. **The rename and the hard switch** (US1, US2, FR-001-FR-003, FR-007, FR-011):
   - the module path, `cmd/chalkeus` with `forge`, the `./chalkeus` shim, `pilot/pilot`
     removed;
   - ask rules switched to the new form, the skill renamed to `forge`;
   - docs, CI and the integration harness;
   - the `chalkeus:proposed` default.

   **The owner renames the repository the day this merges** (R6).
4. **Cluster, labels and images** (US2, US3, FR-008-FR-010):
   - `chalkeus-*` cluster and registry, with old-cluster detection in `doctor`, `up` and `down`;
   - `chalkeus.io/*` labels in manifests and checks, with the audit accepting both;
   - the image workflow publishing `chalkeus-*`;
   - the fixture-parity golden regenerated on purpose (R5).
5. **Release notes and close** (FR-012, US4): the operator's and the owner's steps, the old-name
   check (SC-004), the debrief.

Stage 2 can go before or after stage 1. Stages 3 and 4 follow both. Stage 4 can share stage 3's
day, but needs a `down` and `up` on each machine (the cluster name changes).

## Complexity Tracking

None. There is no constitution violation to justify.
