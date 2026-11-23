# Tasks: Rename to Chalkeus

**Input**: `specs/027-chalkeus-rename/` (spec.md, plan.md, research.md, contracts/, quickstart.md)

**Tests**: included. Each stage has unit tests; the 025 integration scenarios run each command
form end to end.

## Phase 1: Command audit (US1a) — stage 1, on `pilot/pilot`

- [X] T001 [US1a] Add the command-audit table (contracts/command-audit.md) in `internal/cli/command_audit.go`: every command path with its class (ask or free)
- [X] T002 [US1a] Add `internal/cli/command_audit_test.go`: list the commands and fail on a path missing from the table, or on an ask command without a rule in `.claude/settings.json` for both `pilot/pilot <cmd>` and `./pilot/pilot <cmd>`
- [X] T003 [US1a] Add the ask rules for `reset-fixture`, `down`, `publisher-token`, `app-key`, `app-token`, `author`, `step`, `run` and `resume` to `.claude/settings.json` (`approve` has one)
- [X] T004 [US1a] Match `.claude/skills/pilot/SKILL.md` hard rules and `AGENTS.md` to the table, and list the ask commands in `docs/troubleshooting.md#prompts-answer-no` for users of other harnesses

## Phase 2: Config folder and settings (US1) — stage 2

- [X] T005 [US1] In `internal/config`, resolve the config folder (`CHALKEUS_CONFIG`, `~/.config/chalkeus`, then `SPECFOUNDRY_CONFIG`, `~/.config/specfoundry`, default `~/.config/chalkeus`), with a once-per-machine notice whose marker lives outside the config folder (research R3)
- [X] T006 [US1] Read `forge.env`, else `pilot.env`; for every setting and every CLI environment variable, `FORGE_X`, else `PILOT_X`; when both are set, the new one wins and the notice names the ignored one (one lookup function used everywhere)
- [X] T007 [US1] Unit tests in `internal/config/`: every folder combination, both files, both setting names, the notice once; the tool never writes into either folder
- [X] T008 [US1] The audit and the evidence `config.json` accept settings recorded under either prefix; a golden-audit check that nothing changes
  Done as (2026-11-23): by design. Inside the pilot, settings keep their PILOT_ keys (`names.Canonical` maps FORGE_X to PILOT_X), so a run's `config.json` records the same names whichever ones the operator used, and the golden audit is unchanged (`go test ./internal/audit` passes). Credential files (`publisher.env`, `<target>.publisher.env`) keep their keys too, since they become the publisher pod's Secret (research R3).

## Phase 3: The rename and the hard switch (US1, US2) — stage 3; the owner renames the repository the same day

- [X] T009 [US2] Change the module path to `github.com/tosin2013/chalkeus` in `go.mod` and every import
- [X] T010 [US2] Rename `cmd/pilot` to `cmd/chalkeus`: root `chalkeus`, every command under `forge`, `version` at both levels; the command-audit test now walks `forge`
  Done as (2026-11-23): `NewRoot` is `chalkeus` (`version`, `forge`); `NewForge` holds every command. The watcher and step mode re-run the binary as `forge github watch` and `forge run|doctor`. An unknown command under `forge` exits 2.
- [X] T011 [US2] Add the `./chalkeus` shim (builds `./cmd/chalkeus` to `pilot/.build/bin/chalkeus`) and remove `pilot/pilot`
- [X] T012 [US1] Switch `.claude/settings.json` ask rules to `./chalkeus forge <cmd>` and `chalkeus forge <cmd>`; the test checks the new forms; rename `.claude/skills/pilot` to `.claude/skills/forge` and rewrite it for the new form; update `AGENTS.md`
- [X] T013 [US2] Change the step-mode label default to `chalkeus:proposed` in `internal/config/targets.go`; step mode says when a profile relies on the old default and issues carry `pilot/go`
  Done as (2026-11-23): `config.DefaultProposeLabel`; a profile without a label gets `ProposeLabelDefault`, and `step` prints a notice naming open issues that still carry `pilot/go` (`TestOldLabelNotice`).
- [X] T014 [US2] Update every doc, `README.md`, `CONTRIBUTING.md`, CI badges, workflow steps (`go.yml` builds `./cmd/chalkeus`), the integration harness (builds `./cmd/chalkeus`, runs `forge`) and the stub's references
  Done as (2026-11-23): commands, the config folder, `forge.env`, `FORGE_*` (publisher credential keys `PILOT_SCM_*` unchanged), the product name and GitHub links in README, CONTRIBUTING, AGENTS, `pilot/README.md`, `docs/`, both skills and the issue template; `go.yml` (shim, build, coverage floor module path); the integration harness; the docs and wording checks. Left for stage 4: cluster, registry, image and label names. Left for stage 5 (T020): `PILOT_*` names in error messages, the `pilot:` status-PR and commit titles, and history (`.specify/memory/roadmap.md`, past specs).
- [X] T015 [US1] Release notes in `docs/` for stage 3: the owner's rename day and each operator's steps (the new command, their own harness rules, the config move)

## Phase 4: Cluster, labels and images (US2, US3) — stage 4

- [X] T016 [US2] Rename the kind cluster, context and registry to `chalkeus-*` in `internal/paths` and `pilot/kind-config.yaml`; `doctor` warns about an old cluster, `up` refuses beside one, and `down` prints its removal command (research R7); tests
- [X] T017 [US3] Set `chalkeus.io/*` labels in `pilot/manifests/` and select on them in `internal/checks`; the audit accepts both prefixes in evidence; the golden audit passes unchanged
  Done as (2026-11-23): manifests and A05 use `chalkeus.io/*`. The audit never read labels, so "accepts both prefixes" needs no code; the golden audit passes unchanged.
- [X] T018 [US3] Regenerate the fixture-parity golden on purpose, in its own commit, with a diff of only the label prefix and image names (research R5)
- [X] T019 [US2] The image workflow publishes `ghcr.io/tosin2013/chalkeus-*`; `images.lock` keys and digests follow through the bot's PR; old images stay published
  Done as (2026-11-23): #172's merge ran pilot-images on `main`, which published the seven `chalkeus-*` images; the bot's digest PR #173 updated `pilot/images.lock`. Each locked digest pulls without credentials (anonymous GHCR token, HTTP 200), and `./chalkeus forge up` created `chalkeus-m1` from them (doctor: images match the lock).

## Phase 5: Close (US4)

- [X] T020 Add the old-name check to the docs check (research R8), with its allowed list
  Done as (2026-11-23): D5 in `internal/cli/docs_test.go` (`checkOldNames`, `TestOldNameCheck`): any `specfoundry` (any case), `pilot/pilot` or `PILOT_` in the default docs files fails `TestDocsCheck`, outside an allowed list where each entry gives its reason (the release notes; `PILOT_SCM_*`; the companion repository's name; the old config folder; the README's rename note; the old-cluster entry; quoted output of an old run; the `Specfoundry-Run:` trailer forge still writes). Command output is not covered by this check: its old names (error messages naming `PILOT_*`, the `Specfoundry-Run:` trailer, PR and issue bodies, the `<!-- specfoundry:… -->` markers, user agents) are listed for the owner to decide. Also: `--target` on the five commands that require it no longer claims a default in its help.
- [ ] T021 [US4] On the renamed repository: the go, integration and image workflows pass (SC-005); record the rename day in `specs/027-chalkeus-rename/evidence/README.md`
- [ ] T022 The owner's own setup upgraded in place (SC-001): a fixture run and an approval after the steps in the release notes; privacy check; the 027 debrief

## Dependencies

- Phase 1 and Phase 2 are independent. Phase 3 needs both. Phase 4 follows Phase 3 (it can share
  its day). Phase 5 follows.
- **MVP:** Phase 1 (enforced asks), which is valuable on its own even if the rename waited.
