# Research: Proven hidden tests (041)

## R1 — Why today's check let a broken test through

- `author check` (`internal/author/check.go`) runs the grader's baseline with the draft as the
  hidden overlay. Pass means the project's tests pass on the base and the hidden tests fail
  (`grade.py` `baseline`, G5 "pass" when the overlay run exits non-zero).
- A test file that doesn't compile exits non-zero, so it "fails on the base" and passes.
- For a dependency item (029), the baseline's visible tests fail with the manifest change, so G5
  is **skipped** in the baseline: the hidden tests' own behaviour on the base is not observed at
  all. documcp 004 was a dependency item.
- The hidden tests run by overlaying the files and running the **whole** test command again
  (`hidden_mode: overlay`); there is no per-test result.

**Decision**: the reference check is a new grader mode, not a change to the baseline: it applies
the reference change on the base and runs the overlay there; passing is required.

## R2 — Per-test granularity without knowing every test framework

Options:
- **A. One run per hidden file**, overlaying only that file. Works with any test command; costs
  one full test-command run per file and repeat (documcp: about 223 s each).
- **B. Parse a JUnit report** (jest-junit, `go test -json`/go-junit-report, pytest `--junitxml`).
  Gives per-case names, but needs a reporter configured per project.
- **C. A focused test command template** in the profile (`FORGE_HIDDEN_TEST_COMMAND`, for example
  `npx jest {files}` or `go test {packages}`), run with only the hidden files.

**Decision**: granularity is the **hidden test file** (A), run with C when the profile sets it
(fast) and with the full test command otherwise. Per-case names (B) are later. `doctor --item`
suggests C when one full run times the number of files and repeats would exceed half the grader
deadline. Most drafts have 1–3 files (documcp 004: 1).

## R3 — Telling "doesn't compile" apart

A framework-neutral signal that a file never ran doesn't exist. The reference run itself is the
signal: a test that cannot compile fails on the reference, and the reason is "did not pass on the
reference change" — distinct from "fails on the base" (FR-011). The owner sees the output (as
today, printed, not stored).

## R4 — Where the reference and the proof live

- The overlay copies everything under `<target>/items/NNN/` into the project, and readiness
  refuses a slice folder mixed with other entries. A reference there would be copied into the
  graded tree and break the layout rule.
- **Decision**: references at `<target>/references/NNN.patch` (or `NNN/<slice>.patch`); proofs at
  `<target>/proofs/NNN.json` (or `NNN/<slice>.json`), both in the private companion, outside the
  overlay. `reference: none` is a proof file with `"reference": "none"`, committed by the owner.
- The proof names the hidden tests' tree hash and the reference's hash; readiness recomputes the
  tree hash, so a changed test without a new proof is `HIDDEN_TESTS_UNVERIFIED`.
- The worker gets neither: the companion is never staged to the worker; P11 is extended to the
  reference's paths and content hash (P11r).

## R5 — Test-author model and its host

- Today the author Job takes `PILOT_MODEL`, the worker's key files and the model egress host.
- **Decision**: `FORGE_AUTHOR_MODEL` (global or target profile; `PILOT_` alias as for other
  keys). Version 1 requires the same provider and host as the worker's model (one key, one model
  egress), and `doctor` refuses a model on another provider. A second provider means a second
  key and egress host: later, if needed.
- Free-model rule: the target's `FORGE_ALLOW_FREE_MODELS` applies to both models.
- `doctor`'s model-key probe (`internal/config/probe.go`) runs for the author's model too.

## R6 — The vote

- Voters are agent Jobs like the author (same isolation, egress only to the model host), given
  the spec, the test plan, one test file at a time and the scale; they return JSON
  `{test, rating, reason, rules}`.
- **Decision**: two voters by default (`FORGE_VOTERS`, at least 2), the test-author model, the
  agreed rating is the higher one when they differ by one, and "disagree" when they differ by two
  or more (shown to the owner, who decides). A test rated 2–3 sends the draft back.
- Votes are per file (R2), so a reason can name a test inside the file in words.

## R7 — The eval set

- **Decision**: eval cases live in the private companion under `evals/<case>/` (`case.json`: the
  target repo and base commit, the item's spec snapshot, the reference patch, the known failure).
  Results are written to the operator's machine (`~/.local/state/chalkeus/evals/`), never to
  committed evidence. The command is `./chalkeus forge author eval`, run by the owner like
  `author watch` (the session that drives runs never runs it: it prints drafted-test outcomes).
- Score: the number of cases whose draft is proven (reference check) and passes the vote.
- Starting cases: documcp 004 (reference: attempt 2's change from run 20261128T012908Z) and
  documcp 001 US2 (reference: to be supplied by the owner; until then the case is `pending`).

## R8 — Costs

- Reference check: 1 base run (existing) + up to `repeats × files` reference runs (default 3).
  documcp 004 with the full test command: about 4 × 223 s ≈ 15 min; with
  `FORGE_HIDDEN_TEST_COMMAND`, seconds.
- Votes: 2 short agent Jobs per hidden file.
- The grader deadline already scales by profile (`FORGE_GRADER_DEADLINE_S`).

## Sources

SWE-bench Verified (OpenAI); SWT-bench (arXiv 2406.12952); "Solving code review at Anthropic"
(2026-10-15); "When to use multi-agent systems" (Anthropic, 2026-01-23); Paradigm 3, "Swarms"
(2026-11-24).
