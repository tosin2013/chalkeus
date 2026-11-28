# Contracts: commands, codes and output (041)

## Commands

| Command | Who runs it | Does |
|---|---|---|
| `author check --target T --item N` | test author | today's checks, then the reference check when a reference exists |
| `author submit …` | test author | opens the companion PR with the hidden tests and `proofs/NNN.json` |
| `author watch …` | owner | as today; drafts with `FORGE_AUTHOR_MODEL`, checks, votes, submits |
| `author prove --target T --item N [--slice S]` | owner | proves merged hidden tests against the reference; opens a companion PR with the proof |
| `author eval [--case C]` | owner | drafts every eval case, checks, votes, scores, compares |

`author prove` and `author eval` are classified with `author` in `internal/cli/command_audit.go`
(the harness asks) and are never run by the session that drives runs.

## Draft check codes (new)

| Code | Meaning |
|---|---|
| `DRAFT_FAILS_ON_REFERENCE` | a hidden file did not pass on the reference change (includes files that don't compile or load) |
| `DRAFT_PASSES_ON_BASE` | a hidden file already passes on the base (named) |
| `DRAFT_FLAKY` | a hidden file passed and failed across reference runs |
| `REFERENCE_BROKEN` | the project's own tests fail with the reference applied |
| `REFERENCE_DOESNT_APPLY` | the reference patch doesn't apply to the base |
| `DRAFT_VOTED_DOWN` | a voter-agreed rating of 2–3 (US4) |

## Readiness rule

`HIDDEN_TESTS_UNVERIFIED`: "the hidden tests at <companion>@<commit> have no passing reference
check (missing | failed | stale: tests changed since the proof). Fix: add
`<target>/references/NNN.patch` and run `./chalkeus forge author prove --target T --item N`, or
record `reference: none` in `<target>/proofs/NNN.json` (your decision)."

## Doctor rows (new)

- `author model: <model> (worker: <model>)` — PASS, or INFO "drafts use the worker's model".
- model-key probe for the author's model when it differs.
- `reference: proven n/n files, 3 runs` / `none (owner)` / FAIL unverified.
- `test plan: n of m criteria planned` — WARN per missing criterion; INFO without a plan.
- hint: `FORGE_HIDDEN_TEST_COMMAND` when full runs × files × repeats > half the grader deadline.

## PR body (companion, added lines)

```
Drafted by: <model>
Reference check: fails on the base · passes on the reference (n of n files, 3 runs)
Votes: <file>: 0, 0 · <file>: 1, 0
```
