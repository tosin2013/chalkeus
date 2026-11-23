# Tasks: Rename to Chalkeus

**Input**: `specs/027-chalkeus-rename/` (spec.md, plan.md, research.md, contracts/, quickstart.md)

**Tests**: included. Each stage has unit tests; the 025 integration scenarios run each command
form end to end.

## Phase 1: Command audit (US1a) — stage 1, on `pilot/pilot`

- [ ] T001 [US1a] Add the command-audit table (contracts/command-audit.md) in `internal/cli/audit_table.go`: every command path with its class (ask or free)
- [ ] T002 [US1a] Add `internal/cli/command_audit_test.go`: walk the cobra tree and fail on a path missing from the table, or on an ask command without a rule in `.claude/settings.json` for both `pilot/pilot <cmd>` and `./pilot/pilot <cmd>`
- [ ] T003 [US1a] Add the ask rules for `reset-fixture`, `down`, `publisher-token`, `app-key`, `app-token`, `author`, `step`, `run` and `resume` to `.claude/settings.json` (`approve` has one)
- [ ] T004 [US1a] Match `.claude/skills/pilot/SKILL.md` hard rules and `AGENTS.md` to the table, and list the ask commands in `docs/troubleshooting.md#prompts-answer-no` for users of other harnesses

## Phase 2: Config folder and settings (US1) — stage 2

- [ ] T005 [US1] In `internal/config`, resolve the config folder (`CHALKEUS_CONFIG`, `~/.config/chalkeus`, then `SPECFOUNDRY_CONFIG`, `~/.config/specfoundry`, default `~/.config/chalkeus`), with a once-per-machine notice whose marker lives outside the config folder (research R3)
- [ ] T006 [US1] Read `forge.env`, else `pilot.env`; for every setting and every CLI environment variable, `FORGE_X`, else `PILOT_X`; when both are set, the new one wins and the notice names the ignored one (one lookup function used everywhere)
- [ ] T007 [US1] Unit tests in `internal/config/`: every folder combination, both files, both setting names, the notice once; the tool never writes into either folder
- [ ] T008 [US1] The audit and the evidence `config.json` accept settings recorded under either prefix; a golden-audit check that nothing changes

## Phase 3: The rename and the hard switch (US1, US2) — stage 3; the owner renames the repository the same day

- [ ] T009 [US2] Change the module path to `github.com/tosin2013/chalkeus` in `go.mod` and every import
- [ ] T010 [US2] Rename `cmd/pilot` to `cmd/chalkeus`: root `chalkeus`, every command under `forge`, `version` at both levels; the command-audit test now walks `forge`
- [ ] T011 [US2] Add the `./chalkeus` shim (builds `./cmd/chalkeus` to `pilot/.build/bin/chalkeus`) and remove `pilot/pilot`
- [ ] T012 [US1] Switch `.claude/settings.json` ask rules to `./chalkeus forge <cmd>` and `chalkeus forge <cmd>`; the test checks the new forms; rename `.claude/skills/pilot` to `.claude/skills/forge` and rewrite it for the new form; update `AGENTS.md`
- [ ] T013 [US2] Change the step-mode label default to `chalkeus:proposed` in `internal/config/targets.go`; step mode says when a profile relies on the old default and issues carry `pilot/go`
- [ ] T014 [US2] Update every doc, `README.md`, `CONTRIBUTING.md`, CI badges, workflow steps (`go.yml` builds `./cmd/chalkeus`), the integration harness (builds `./cmd/chalkeus`, runs `forge`) and the stub's references
- [ ] T015 [US1] Release notes in `docs/` for stage 3: the owner's rename day and each operator's steps (the new command, their own harness rules, the config move)

## Phase 4: Cluster, labels and images (US2, US3) — stage 4

- [ ] T016 [US2] Rename the kind cluster, context and registry to `chalkeus-*` in `internal/paths` and `pilot/kind-config.yaml`; `doctor` warns about an old cluster, `up` refuses beside one, and `down` prints its removal command (research R7); tests
- [ ] T017 [US3] Set `chalkeus.io/*` labels in `pilot/manifests/` and select on them in `internal/checks`; the audit accepts both prefixes in evidence; the golden audit passes unchanged
- [ ] T018 [US3] Regenerate the fixture-parity golden on purpose, in its own commit, with a diff of only the label prefix and image names (research R5)
- [ ] T019 [US2] The image workflow publishes `ghcr.io/tosin2013/chalkeus-*`; `images.lock` keys and digests follow through the bot's PR; old images stay published

## Phase 5: Close (US4)

- [ ] T020 Add the old-name check to the docs check (research R8), with its allowed list
- [ ] T021 [US4] On the renamed repository: the go, integration and image workflows pass (SC-005); record the rename day in `specs/027-chalkeus-rename/evidence/README.md`
- [ ] T022 The owner's own setup upgraded in place (SC-001): a fixture run and an approval after the steps in the release notes; privacy check; the 027 debrief

## Dependencies

- Phase 1 and Phase 2 are independent. Phase 3 needs both. Phase 4 follows Phase 3 (it can share
  its day). Phase 5 follows.
- **MVP:** Phase 1 (enforced asks), which is valuable on its own even if the rename waited.
