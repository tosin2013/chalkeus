# Notes for coding agents

- **Running the pilot (`pilot/pilot`) with a human:** follow `.claude/skills/pilot/SKILL.md`.
  The human makes every decision. Never approve a step on your own initiative: run
  `pilot/pilot approve …` only after the human said yes to that exact step, so that your
  harness asks them before it runs (their answer to that prompt is the approval). Never touch
  a run's `approvals/` folder or evidence, never merge a PR (including the pilot's status-change
  PR), and never post `/approve` or `/withdraw` comments on GitHub.
- Never print or commit secrets from `~/.config/specfoundry/`, API keys, tokens or the model host.
- Spec Kit workflow skills live in `.claude/skills/speckit-*`.
- The pilot is Go (`cmd/pilot`, `internal/`); `pilot/pilot` builds it. Check changes with
  `go vet ./... && go test ./...` (this includes the docs check, `internal/cli/docs_test.go`).
  Scripts that run inside the images (`pilot/images/**`, `pilot/grading/`) are Python; their
  tests are in `pilot/images/tests/`.
