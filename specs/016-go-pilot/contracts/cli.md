# Contract: the Go `pilot` command line (supersedes the reference CLI at parity)

The operator-facing contract is the reference's, from `specs/002-m1-gated-pilot/contracts/cli.md`
as amended by 012 and 004. This file states what must stay identical, and the only additions.

## Command tree

```text
pilot up [--cluster-only] [--local-build]
pilot down
pilot publisher-token [--check-only]
pilot import-opencode [--from DIR] [--model P/M] [--fixture-repo O/R] [--force]
pilot reset-fixture [--yes]
pilot run
pilot resume [--run ID]
pilot approve {mode|retry|reconcile} --run ID
pilot gate {A|B} [--root DIR]
pilot audit [--run ID]
pilot status [--run ID]
pilot github sync
pilot github drift [--run ID]
pilot github watch --run ID [--detach | --stop]   # --detach/--stop are new (issue #25)
pilot app-key            # key on stdin
pilot app-token --check
pilot version            # new; prints the stamped version
```

## Exit codes (identical)

| Code | Meaning |
|---|---|
| 0 | ok |
| 1 | fail (audit fail, drift found, check not OK) |
| 2 | error (config, GitHub error, no run, …) |
| 3 | repo guard (fixture repo is this repository) |
| 4 | awaiting merge |
| 5 | declined / paused for mode or reconcile |
| 6 | awaiting retry |
| 10 | gate refusal |

Errors print `pilot: <message>` to stderr. Usage errors (unknown command or flag) exit 2.

## Help output

Cobra's format (`Usage:`, `Available Commands:`, `Flags:`). `pilot/tools/docs_check.py` reads it
to check that the docs name only existing commands and flags.

## Wording

Every prompt, pause message and hint quoted in `docs/operator-guide.md`,
`docs/troubleshooting.md`, `docs/github-loop.md`, `docs/github-app.md` and
`.claude/skills/pilot/SKILL.md` is reproduced verbatim. A test extracts the quoted strings and
checks that they appear in the Go sources' message table.
