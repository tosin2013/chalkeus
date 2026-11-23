# Research: Rename to Chalkeus

## R1 — Command structure

- **Decision:** the binary built from `cmd/chalkeus` has root command `chalkeus`, and every
  pilot command moves under one subcommand, `forge` (`chalkeus forge run`, `chalkeus forge
  approve`, …). `chalkeus version` also works at the root. In the repository, the shim
  `./chalkeus` (replacing `pilot/pilot`) builds `./cmd/chalkeus` on first use and whenever a Go
  source changes, as `pilot/pilot` does today. It writes the binary to `pilot/.build/bin/chalkeus`.
- **Rationale:** it matches 013's installed form (`chalkeus forge …`), so docs and harness rules
  are the same in the repository and after install, apart from the leading `./`.
- **Alternatives:** a `forge` binary (rejected by Q7: the product name is the binary); keeping
  `pilot` as a hidden alias (rejected by the owner: hard switch).

## R2 — Command audit

- **Decision:** a table in `internal/cli` lists every command path (`approve`, `author start`,
  `github watch`, …) with its class:
  - **ask:** `approve`, `reset-fixture`, `down`, `publisher-token`, `app-key`, `app-token`,
    `author` (every subcommand), `step`, `run`, `resume`;
  - **free:** every other command.

  A test walks the cobra command tree and fails when:
  - a command path isn't in the table;
  - an ask command has no rule in `.claude/settings.json` matching both `pilot/pilot <cmd>` and
    `./pilot/pilot <cmd>` (stage 1), or `./chalkeus forge <cmd>` and `chalkeus forge <cmd>`
    (from stage 3).

  The pilot skill's "hard rules" list the same ask commands.
- **Rationale:** today only `approve` is enforced. The rest are skill text an agent is trusted to
  follow (constitution IV and V). The test keeps the table complete as commands are added: 026
  added `slices` (free) and the `acceptance` step of `approve` (ask, through `approve`).
- **Alternatives:** deny rules for `author` and `step` (rejected: the separate test-author session
  runs `author start/check/submit`, and these rules apply to every Claude session in the
  repository).

## R3 — Config folder and settings

- **Decision:** one lookup in `internal/config`:
  - **Folder:** `CHALKEUS_CONFIG`, else `~/.config/chalkeus` if it exists. Otherwise
    `SPECFOUNDRY_CONFIG`, else `~/.config/specfoundry` if it exists, and say once (a marker file
    under the build folder, not in the config folder) how to move it, with
    `mv ~/.config/specfoundry ~/.config/chalkeus`, keeping modes. With neither, the default is
    `~/.config/chalkeus`.
  - **Settings file:** `forge.env`, else `pilot.env`.
  - **Each setting:** `FORGE_X`, else `PILOT_X`. When both are set, the new one wins, and the
    notice names the ignored old one. The same rule applies to the environment variables the CLI
    reads (`FORGE_ROOT`/`PILOT_ROOT`, `FORGE_TEST_MODE`/`PILOT_TEST_MODE`, …).
- **Rationale:** FR-005, FR-006 and FR-013. One lookup means one set of tests and no forgotten
  setting.
- **Not renamed:** the variables the pilot sets inside its own pods (for example the publisher's
  token variable and the worker's model variable). They're an internal contract between the
  pilot and its images, never typed by operators, and renaming them would change every image for
  no user benefit. Evidence keeps recording settings under the names the run used. The audit
  reads both.

## R4 — Kubernetes labels

- **Decision:** the manifests set `chalkeus.io/*` in place of `specfoundry.io/*`. The checks
  select on the new labels. The audit's rules accept either prefix in recorded evidence.
- **Rationale:** FR-009; the owner accepted that the project doesn't own `chalkeus.io`. Live
  clusters are created fresh by `up` after stage 4, so no running object needs both.
- **Alternatives:** dual labels on every object (rejected: doubles manifest noise for a
  disposable laptop cluster).

## R5 — The fixture-parity golden

- **Decision:** `TestFixtureManifestsUnchanged` pins the fixture's rendered manifests (014
  FR-015). Stage 4 regenerates its golden **on purpose**, in its own commit, and the diff is only
  the label prefix and the image names. The PR says so.
- **Rationale:** the parity rule exists to catch accidental changes. A deliberate rename is
  recorded, not hidden.

## R6 — The order around the repository rename

- **Decision:**
  - stage 3 changes the module path;
  - the owner renames tosin2013/specfoundry to tosin2013/chalkeus in GitHub settings the same
    day, right after the merge;
  - then every machine runs `git remote set-url origin https://github.com/tosin2013/chalkeus.git`
    (optional, since GitHub redirects);
  - the Go module proxy is left to pick up the new path on first install.
- **Rationale:** inside the repository, builds use the local module, so nothing breaks between
  the merge and the rename. Only `go install github.com/tosin2013/chalkeus/...` needs the
  rename, and nothing uses that before 013.
- **Risk:** GitHub's redirect is replaced if a new repository is ever created under the old name.
  The release notes say not to.

## R7 — Old cluster detection

- **Decision:** `doctor`, `up` and `down` look for a kind cluster or registry container with the
  old names:
  - `doctor` warns, with the command to remove it;
  - `up` refuses to create the new cluster beside a running old one, unless the operator has
    removed it (the port 5001 registry would collide anyway);
  - `down` offers the old cluster's removal command.
- **Rationale:** FR-010; one pilot cluster per machine (the registry port is fixed).

## R8 — The old-name check (SC-004)

- **Decision:** a docs-check rule lists any "specfoundry", "pilot/pilot", `PILOT_` or
  `specfoundry.io` mention in user-facing files (`README.md`, `docs/`, `CONTRIBUTING.md`,
  `AGENTS.md`, `.claude/skills/`), outside an allowed list: migration notes, history sections, and
  the name map itself. Past items' `specs/` and evidence are history and are not checked.
- **Rationale:** a measurable SC-004, and a guard against new mentions.
