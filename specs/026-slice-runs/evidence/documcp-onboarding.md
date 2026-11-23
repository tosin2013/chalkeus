# documcp onboarding: friction log (T047)

Onboarding `tosin2013/documcp` (TypeScript MCP server) as a Chalkeus target, as a new user would:
following the docs as written, for an existing repository (not greenfield). Each entry: what
the docs said, what happened, and the suggested fix. Started 2026-11-23.

## Decisions (owner)

- documcp v2.0 (PRD 2026-11-23) is 026's live item; the PRD's Gate 0 ("after the laptop
  release") is waived for this.
- Node 24 LTS (Node 25 is end of life since 2026-06); uses the existing `node24` toolchain.

## Baseline (2026-11-23, Node 22.23.5, before any change)

- `npm ci`: OK. npm did not run `tree-sitter-yaml@0.5.0`'s install script (`node-gyp rebuild`)
  and listed it for `npm install-scripts approve`.
- `npm run typecheck`: OK. `npm test`: 90 suites, 1706 passed, 14 skipped, 42 s (one worker force-exited
  after the run: leaked handles).

## Friction

| # | Where | Docs say | What happened | Suggested fix |
|---|---|---|---|---|
| F1 | `docs/targets.md` intro | "Today you can use `FORGE_TOOLCHAIN=python3.13` … The other toolchains follow." | The same page's table lists `node24`, `go` and `jdk21-maven` as available | Update the intro to list the catalog toolchains |
| F2 | `docs/greenfield.md` §2, `docs/targets.md` §1 | "Spec Kit (`specify init`), plus the roadmap and spec-status extensions" | No install command is given. `specify` and `uv` weren't installed; Spec Kit is now 1.4.3, where `--ai` became `--integration`; both extensions are "discovery only" in the community catalog and install only with `--from <archive-url>` after a trust prompt | Give the exact commands (below) |
| F3 | (none) | — | documcp's `.gitignore` ignores `.claude/`, so the Spec Kit skills `specify init` writes there aren't committed | Mention that the agent skills are per machine unless the target commits them |

| F4 | roadmap extension config | ADRs default to `docs/adr/` | documcp keeps them in `docs/adrs/`; editing `roadmap-config.yml` had no effect (F5) | Say in the docs to check `adr.dir` |
| F5 | roadmap extension 0.1.1, installed from its source archive | — | The archive carries the extension's own `.specify/`, so `load-config.sh` (which walks up from its own folder) resolves the *extension* as the repo root: it reports `roadmap_exists: true` from the extension's own roadmap and ignores the project's config. The same nested `.specify/` is in Chalkeus's own install | Report upstream (`srobroek/speckit-roadmap`); until then install from a release package without `.specify/`, or set `SPECKIT_ROADMAP_PATH`/`SPECKIT_ROADMAP_ADR_DIR` |
| F6 | `docs/targets.md` §2 (dependencies) | "an attempt can't add dependencies; add new ones to the project yourself when you prepare the item" | True and important for a runtime move: `@types/node` and CI's Node version must go in the owner's planning PR, so a "move to Node N" story can only declare the version and fix code | Add a line on runtime upgrades to the greenfield/targets docs |

Commands that worked (Spec Kit 1.4.3):

```sh
pipx install uv
uvx --from git+https://github.com/github/spec-kit.git@v1.4.3 specify init --here \
  --integration claude --script sh --ignore-agent-tools --force --non-interactive
S="uvx --from git+https://github.com/github/spec-kit.git@v1.4.3 specify"
$S extension add roadmap --from https://github.com/srobroek/speckit-roadmap/archive/refs/tags/v0.1.1.zip
$S extension add spec-status --from https://github.com/tosin2013/spec-status/releases/download/v0.4.0/spec-status-extension_0.4.0.zip
```
