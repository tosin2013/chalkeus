# Implementation plan: Proven hidden tests (041)

**Spec**: [spec.md](spec.md) · **Research**: [research.md](research.md) ·
**Data model**: [data-model.md](data-model.md) · **Contracts**: [contracts/](contracts/) ·
**Quickstart**: [quickstart.md](quickstart.md)

## Summary

Hidden tests get proven before they grade anything. A separate test-author model drafts them; a
reference change kept in the private companion shows they can pass; independent voters rate each
file against the spec; a test plan in the spec guides drafting; an eval set scores the author.

## Technical context

- **Language**: Go (CLI, `internal/`), Python for the grader (`pilot/images/tools/grade.py`).
- **Touches**: `internal/config` (model keys), `internal/author` (job, check, submit, watch, new
  `prove`, `vote`, `eval`), `internal/grader` and `grade.py` (a `reference` mode),
  `internal/readiness` (a new rule), `internal/checks` (P11r), `internal/doctor` / `internal/cli`
  (doctor rows, commands, command audit), docs.
- **Images**: `grade.py` changes ship through the image workflow (C-09, `images.lock`); no new image.
- **Testing**: `go vet ./... && go test ./...`; grader tests in `pilot/images/tests/`; a cluster
  check against documcp 004 (SC-001).

## Constitution check

- **II Gates fail closed**: `HIDDEN_TESTS_UNVERIFIED` refuses; a missing or unreadable proof is
  unverified. Pass.
- **III Evidence defines done**: proofs and votes are evidence (counts, names, hashes). Pass.
- **IV Least privilege**: the reference and the proofs stay in the companion; P11r proves the
  worker never sees the reference; voters run with the author's isolation. Pass.
- **V Humans approve, humans merge**: `reference: none`, vote disagreements and companion PRs
  stay the owner's. Pass.
- **VI Build on**: reuses the baseline machinery, the author Job, the overlay. Pass.
- **C-14**: no drafted test, output or reference test result reaches committed evidence or the
  session that drives runs; `author eval` and `author prove` print to the owner's terminal only.

## Design by story

### US1 — Test-author model (P1)

- `FORGE_AUTHOR_MODEL` read from the target profile, then the global configuration; empty means
  the worker's model. `internal/cli/author.go` passes it as `JobEnv.Model`.
- Same provider and host as the worker's model in version 1 (research R5); `doctor` refuses
  otherwise, and probes the key with the author's model.
- The draft records `model` in `draft.json`; `PRBody` names it.

### US3 — Reference check (P3, built before US2 because the eval scores with it)

- **Grader**: `grade.py` gets `mode: reference`: apply the item's manifest change (029) and the
  reference patch on the base, check the project's tests pass, then for each hidden file run the
  focused or full test command with only that file overlaid, `repeats` times. Output per file:
  `{file, base: fail|pass, reference: [pass|fail…], verdict}`; hidden output goes to the separate
  frame file for the owner, never into `grade.json` (as `author_check` does today).
- **Base side per file**: the same per-file runs on the base, so "passes on the base" names the
  file. For a dependency item the base includes the manifest change.
- **Go side**: `author check` runs the reference mode after today's baseline when a reference
  exists; new codes `DRAFT_FAILS_ON_REFERENCE`, `DRAFT_PASSES_ON_BASE` (per file),
  `DRAFT_FLAKY`, `REFERENCE_BROKEN`, `REFERENCE_DOESNT_APPLY`. The proof (data model) is written
  into the companion PR at `<target>/proofs/NNN.json` by `author submit`.
- **`author prove --target T --item N`**: runs the reference check on hidden tests already merged
  and opens a companion PR with the proof (for documcp 004 and any item drafted before 041).
- **Readiness**: after the hidden-tests rule, read the proof; recompute the hidden tree hash;
  refuse `HIDDEN_TESTS_UNVERIFIED` when the proof is missing, failed or stale; allow with a
  warning for `reference: none`.
- **P11r**: the attempt's staged inputs, ConfigMaps and prompt contain no reference path or
  content hash (`internal/checks`), and the audit lists it.

### US2 — Author eval set (P2)

- `evals/<case>/case.json` in the companion; `author eval` drafts each case with the current
  settings (one author Job per case, sequentially), runs the reference check and the vote, and
  writes the result to `~/.local/state/chalkeus/evals/<timestamp>.json`; it compares with the
  previous result and prints per-case changes and "score dropped" when it did.
- Results record the author model, the prompt template's hash and the forge version.

### US4 — Per-test vote (P4)

- `internal/author/vote.go`: after a passing reference check, start `FORGE_VOTERS` voter Jobs
  per hidden file (author Job machinery, a voter prompt, read-only inputs: spec, plan, file,
  scale); collect JSON verdicts; combine (research R6).
- A 2–3 sends the draft back (watch redrafts once with the reasons in the prompt; by hand, the
  author sees them); disagreement opens the PR as ready with every vote shown.

### US5 — Test plan (P5)

- A `## Test plan` section in spec.md: one line per criterion id (`US1-AS2: …`). `author.Criteria`
  reads it; the author prompt includes it; the mapping file gets `plan` per test.
- `doctor --item`: warn per criterion without a plan line; info when there is no plan.
- Docs: how to write one (`docs/test-author.md`), with documcp 004 as the example.

## Order and slices

1. US1 (small, used by everything after). 2. US3 grader mode, check, proof, `prove`, readiness,
P11r. 3. US2 eval (needs US3). 4. US4 vote. 5. US5 plan. Each is its own PR.

## Command audit

New commands `author prove`, `author eval` are classified like `author` (the harness asks) and
documented as owner-run; `author eval` and `author prove` are never run by the session that
drives runs (they print drafted-test outcomes). `TestCommandAudit` covers both.

## Risks

- Per-file runs with a slow full test command are expensive: `FORGE_HIDDEN_TEST_COMMAND` and the
  doctor hint address it; repeats never drop below 2.
- A wrong reference blames good tests: `REFERENCE_BROKEN` (project tests fail on the reference)
  catches the common case; the owner fixes the reference.
- Voters on a free model may be noisy: they use the test-author model, and disagreement goes to
  the owner.
