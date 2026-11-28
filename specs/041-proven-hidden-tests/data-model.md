# Data model: Proven hidden tests (041)

All of these live in the private companion or on the operator's machine. Committed evidence keeps
only counts, file names, ratings and hashes (C-14, 035).

## Companion layout (per target)

```
<target>/items/NNN[/<slice>]/...        hidden tests (unchanged, the overlay)
<target>/references/NNN.patch           reference change (or NNN/<slice>.patch)
<target>/proofs/NNN.json                proof of the merged tests (or NNN/<slice>.json)
evals/<case>/case.json                  author eval cases
evals/<case>/reference.patch
evals/<case>/spec.md
```

## Proof (`proofs/NNN.json`)

| Field | Meaning |
|---|---|
| `version` | 1 |
| `item`, `slice` | the item and slice ("" when whole) |
| `reference` | `"patch"` or `"none"` (owner decision) |
| `reference_sha256` | hash of the reference patch (absent for none) |
| `tests_tree_sha256` | hash of the hidden-test files (path + content, sorted) the proof covers |
| `base_revision` | the target commit the check ran on |
| `manifest_commit` | 029 manifest change applied, if any |
| `files` | `[{path, base: "fail"/"pass", reference: ["pass","pass","pass"], verdict}]` |
| `repeats` | reference runs per file |
| `result` | `proven` / `failed` / `none` |
| `model` | the model that drafted the tests |
| `votes` | `[{path, voter, rating, reason, rules}]` (US4) |
| `checked_at`, `forge_version` | when and with which forge |
| `decided_by` | for `none`: the owner's login |

Readiness: `proven` and matching `tests_tree_sha256` → allow; `none` → allow with warning;
anything else, missing or stale → `HIDDEN_TESTS_UNVERIFIED`.

## Draft (`draft.json`, existing, new fields)

- `model`: the test-author model.
- `checks[].reference`: `{result, files, repeats, code}` (no output).
- `votes`: as in the proof.

## Eval case (`evals/<case>/case.json`)

| Field | Meaning |
|---|---|
| `id` | for example `documcp-004` |
| `target_repo`, `base_revision`, `manifest_commit` | what to draft against |
| `item`, `slice` | the item |
| `known_failure` | why it is a case (one line) |
| `status` | `ready` / `pending` (no reference yet) |

## Eval result (`~/.local/state/chalkeus/evals/<timestamp>.json`)

`{model, prompt_sha256, forge_version, cases: [{id, outcome: proven|unproven|error, reason, votes_ok}], score, previous, dropped}`

## Profile keys

| Key | Default | Meaning |
|---|---|---|
| `FORGE_AUTHOR_MODEL` | the worker's model | model for drafts and votes |
| `FORGE_HIDDEN_TEST_COMMAND` | none (full test command) | focused command; `{files}` is replaced |
| `FORGE_REFERENCE_REPEATS` | 3 (min 2) | reference runs per file |
| `FORGE_VOTERS` | 2 (min 2) | voters per hidden file |
