# Contract: the command audit

Every command is **ask** (the repository's harness rules ask the human before it runs) or
**free**. A test fails on a command missing from this table, or on an ask command without a
matching rule. New commands must be added here.

| Command | Class | Why |
|---|---|---|
| `approve` (every step, including 026's `acceptance`) | ask | records the human's decision |
| `reset-fixture` | ask | force-pushes the fixture repository |
| `down` | ask | deletes the cluster |
| `publisher-token` | ask | writes the publisher credential |
| `app-key` | ask | stores the GitHub App private key |
| `app-token` | ask | mints a GitHub App token |
| `author` (every subcommand) | ask | the test author's session only; never the driving session |
| `step` | ask | the owner's manager loop; never the driving session |
| `run` | ask | starts a run (owner decision 2026-11-22) |
| `resume` | ask | continues a run (owner decision 2026-11-22) |
| `up` | free | stands up the local cluster; no decision |
| `status`, `audit`, `doctor`, `gate`, `targets`, `slices`, `version` | free | read-only |
| `github sync`, `github drift` | free | mirror and compare; no decision |
| `github watch` | free | resumes only after a human's approval or merge on GitHub |
| `import-opencode` | free | copies the operator's own model config into the config folder |

Rule forms (Claude Code, `.claude/settings.json` `permissions.ask`):
- stage 1: `Bash(pilot/pilot <cmd>:*)` and `Bash(./pilot/pilot <cmd>:*)`;
- from stage 3: `Bash(./chalkeus forge <cmd>:*)` and `Bash(chalkeus forge <cmd>:*)`.

Other harnesses: the docs list the same commands, and users add them to their own config. The
tool never writes harness config.
