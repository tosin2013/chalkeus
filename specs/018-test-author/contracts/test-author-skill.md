# Contract: the test-author skill

The skill (`.claude/skills/test-author/SKILL.md`) is for a session whose only job is drafting
hidden tests. It MUST say:

1. You are the test author for one item. Start with `pilot/pilot author start --target <t>
   --item <NNN>` and work only in the workspace it prints.
2. Read `spec.md`, `plan.md`, `tasks.md`, `criteria.json` and the project under `base/`.
3. Write tests under `draft/`, laid out like the project, that fail on the current code and pass
   once the spec is implemented. Use only dependencies the project already declares.
4. Write `draft/.acceptance-map.json` covering every criterion you can; list the rest as
   uncovered.
5. Run `pilot/pilot author check …` until it passes. Its output is yours to read.
6. Run `pilot/pilot author submit …` and give the human the PR link. The human reviews and merges.

Never: run `run`, `resume`, `approve`, `github` or `reset-fixture`; push to or open a PR on the
target; edit the target's specs, tasks or roadmap; read `~/.config/specfoundry/`; paste test
contents anywhere but the draft files.
