# Notes for coding agents

- **Running forge (`./chalkeus forge`) with a human:** follow `.claude/skills/forge/SKILL.md`.
  The human makes every decision. Never approve a step on your own initiative: run
  `./chalkeus forge approve …` only after the human said yes to that exact step, so that your
  harness asks them before it runs (their answer to that prompt is the approval). Never touch
  a run's `approvals/` folder or evidence, never merge a PR (including forge's status-change
  PR), and never post `/approve` or `/withdraw` comments on GitHub.
- **The harness asks the human before ten forge commands:** `approve`, `reset-fixture`, `down`,
  `publisher-token`, `app-key`, `app-token`, `author`, `step`, `run` and `resume` (027 command
  audit; the rules are in `.claude/settings.json`, and `internal/cli` `TestCommandAudit` fails if
  a command is unclassified or a rule is missing). A new command must be classified in
  `internal/cli/command_audit.go`.
- Never print or commit secrets from `~/.config/chalkeus/` (or the old `~/.config/specfoundry/`), API keys, tokens or the model host.
- Spec Kit workflow skills live in `.claude/skills/speckit-*`.
- Chalkeus is Go (`cmd/chalkeus`, `internal/`); the `./chalkeus` shim builds it. Check changes with
  `go vet ./... && go test ./...` (this includes the docs check, `internal/cli/docs_test.go`).
  Scripts that run inside the images (`pilot/images/**`, `pilot/grading/`) are Python; their
  tests are in `pilot/images/tests/`.
