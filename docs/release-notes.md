# Release notes

## Brownfield intake: the debt report (029, part 1)

New: `./chalkeus forge debt --target <name> [--json]`, a read-only report for an existing
repository ([targets.md](targets.md#an-existing-repository-the-debt-report)):
- its open Dependabot PRs, grouped by the checks they fail, with the checks shared by many PRs (or
  failing on the default branch) set apart, and each PR marked `superseded`, `safe`,
  `needs code changes`, `pending` or `not triaged`;
- baseline flags: a red default branch, test scripts that force an exit, packages with install
  scripts.

It writes nothing anywhere and runs nothing of the target's. With a GitHub App it mints a token
restricted to read permissions, in memory; otherwise it uses your read token. To read private
repositories' checks and the advisories each PR fixes, grant the App **Checks**, **Commit
statuses** and **Dependabot alerts** (all read). Nothing else to do.

## Chalkeus: forge's own output (027, stage 5)

Forge's messages, help and GitHub text now use the new names too:

| Before | Now |
|---|---|
| messages that name a setting, for example `not in PILOT_APPROVERS` | `FORGE_*`, for example `not in FORGE_APPROVERS` (the `PILOT_SCM_*` publisher keys keep their names) |
| `… in pilot.env` | `… in forge.env` |
| PR and issue text "Opened by the specfoundry … publisher", "Generated from the roadmap by specfoundry" | Chalkeus |
| hidden issue markers `<!-- specfoundry:item=…` and `<!-- specfoundry:decision …` | `<!-- chalkeus:… -->` |
| commit trailer `Specfoundry-Run: <run id>` | `Chalkeus-Run: <run id>` |

Nothing to do. Forge still reads the old markers: an issue synced before is matched to its item,
and its next sync rewrites the body with the new marker; a run paused before the update finds the
decision it already posted instead of posting it again. Commits made before keep their old
trailer. Scripts that matched the old message text need the new names.

## Chalkeus: the cluster, labels and images (027, stage 4)

| Before | Now |
|---|---|
| kind cluster `specfoundry-m1` (context `kind-specfoundry-m1`) | `chalkeus-m1` (`kind-chalkeus-m1`) |
| local registry container `specfoundry-registry` | `chalkeus-registry` (still on port 5001) |
| Kubernetes labels `specfoundry.io/role`, `specfoundry.io/run`, `specfoundry.io/draft` | `chalkeus.io/…` |
| images `ghcr.io/tosin2013/specfoundry-pilot-*` | `ghcr.io/tosin2013/chalkeus-*` |

Runs and evidence from before are unchanged: the audit never read the labels, and old images
stay published.

### Each operator's steps

1. **Finish or stop any run in progress first.** A paused run lives in the old cluster.
2. **Take the old cluster down** with the command from before the update, or by hand:

   ```sh
   kind delete cluster --name specfoundry-m1 && docker rm -f specfoundry-registry
   ```

   If you skip this, `./chalkeus forge up` refuses and prints that command, and
   `./chalkeus forge doctor` warns about it. Chalkeus never removes it for you.
3. **`git pull`, then `./chalkeus forge up`.** It creates `chalkeus-m1`.

### The images

The image workflow publishes `chalkeus-*` when this change reaches `main`, then opens its usual
digest PR for `pilot/images.lock`. Until that PR is merged, `up` pulls the old
`specfoundry-pilot-*` digests from the lock, which stay published. The owner checks once that
the new packages are public (GitHub → Packages → each `chalkeus-*` → Package settings), since
`up` pulls them without credentials.


## Chalkeus: the rename (027, stage 3)

specfoundry is now **Chalkeus**, and its pilot is **forge**. The in-repository command is
`./chalkeus forge <cmd>`. `pilot/pilot` is removed: it's a hard switch, with no alias. Every
command, flag and exit code is the same; only the name in front changes.

| Before | Now |
|---|---|
| `pilot/pilot run`, `pilot/pilot approve …` | `./chalkeus forge run`, `./chalkeus forge approve …` |
| `pilot/pilot version` | `./chalkeus version` (or `./chalkeus forge version`) |
| `~/.config/specfoundry/` | `~/.config/chalkeus/` (the old folder is still read) |
| `pilot.env` | `forge.env` (the old name is still read) |
| `PILOT_*` settings, for example `PILOT_MODEL` | `FORGE_*`, for example `FORGE_MODEL` (the old names are still read) |
| `SPECFOUNDRY_CONFIG` | `CHALKEUS_CONFIG` (the old name is still read) |
| step mode's default label `pilot/go` | `chalkeus:proposed` |
| the `pilot` agent skill | the `forge` skill (`.claude/skills/forge/`) |
| github.com/tosin2013/specfoundry | github.com/tosin2013/chalkeus (GitHub redirects the old address) |

The kind cluster, the local registry, the image names and the Kubernetes labels moved in
stage 4 ([below](#chalkeus-the-cluster-labels-and-images-027-stage-4)).

Every old name that still works prints a one-time notice with its new name. When you set both
an old and a new name, the new one wins and the notice says which one is ignored. The full map is
[contracts/name-map.md](../specs/027-chalkeus-rename/contracts/name-map.md).

### The owner's steps, on the day

1. Merge the stage 3 pull request.
2. The same day, rename the repository: GitHub → Settings → General → Repository name →
   `chalkeus`. GitHub redirects the old address for clones, pulls and links.
3. Never create a new repository named `specfoundry` under the same account: it would replace
   the redirect.
4. Check that the go, integration and image workflows pass on the renamed repository.

### Each operator's steps

1. **Update your checkout.** `git pull`. Optionally point it at the new address (the old one
   redirects):

   ```sh
   git remote set-url origin https://github.com/tosin2013/chalkeus.git
   ```

2. **Use the new command.** `./chalkeus forge doctor` builds it on first use. Scripts and notes
   that call `pilot/pilot` must change; it no longer exists.
3. **Update your own harness rules.** This repository's `.claude/settings.json` already asks before
   the ten commands that decide, destroy or hold credentials, in the new form. If you added rules
   to your own harness config (for example OpenCode's `~/.config/opencode/opencode.json`), replace
   each `pilot/pilot <cmd>` pair with `./chalkeus forge <cmd>` and `chalkeus forge <cmd>`
   ([prompts answer "no"](troubleshooting.md#prompts-answer-no) lists the ten).
4. **Restart your coding agent's session** in this repository, so it loads the `forge` skill and
   the new rules.
5. **Move the config folder (optional, recommended).** Until you do, Chalkeus reads the old one
   and says so once. Keep its permissions with `mv`:

   ```sh
   mv ~/.config/specfoundry ~/.config/chalkeus
   ```

   Chalkeus never moves, copies or writes into either folder itself.
6. **Rename your settings (optional).** In `forge.env` (or `pilot.env`) and in each
   `targets/<name>.env`, `FORGE_X` replaces `PILOT_X` with the same suffix. Keep the publisher
   credential files (`publisher.env`, `targets/<name>.publisher.env`) as they are: their keys
   don't change.
7. **Step mode's label.** If a target profile sets no `FORGE_PROPOSE_LABEL` (or
   `PILOT_PROPOSE_LABEL`), step mode now looks for the label `chalkeus:proposed`. Either relabel
   the issues you proposed with `pilot/go`, or keep the old label in the profile:

   ```sh
   FORGE_PROPOSE_LABEL=pilot/go
   ```

   `./chalkeus forge step` says when open issues still carry `pilot/go` while the profile relies
   on the default.

Runs, evidence and audits from before the rename are unchanged: an old run resumes and audits as
before with the new command.
