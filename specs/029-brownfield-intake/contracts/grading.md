# Contract: grading changes (`pilot/images/tools/grade.py`)

Both are additive: a meta without the new fields grades exactly as today, so the golden audit and
every existing test are unchanged.

## `fixed_paths` (attempts and baseline)

- `meta.fixed_paths`: list of paths (exact, not globs). Absent or empty: no effect.
- In an attempt, `G2_allowed_paths` fails when any changed path (added, modified, deleted or
  renamed from/to) is in `fixed_paths`, even if `allowed_paths` matches it. The detail starts
  with `MANIFEST_CHANGED: <path>[, <path>…]`, then the usual `changed=… disallowed=…`.

## `dependency` (baseline only)

- `meta.dependency: true` changes the baseline's pass rule (amended 2026-11-26, owner: "either
  fails"; research R12):
  - `G4_visible_tests` is run and recorded; its detail is prefixed
    `with the approved manifest change: ` and the result is `fail` or `pass` as observed;
  - **visible tests fail:** that is the work, and it proves there is some. `G5_hidden_tests` is
    `skipped` with `not measured: …`, because the hidden tests run inside the same suite and a
    failing suite can't tell them apart;
  - **visible tests pass:** `G5_hidden_tests` runs and must fail, as today;
  - the baseline passes when G1 passes and either the visible tests fail or the hidden tests
    fail (G5 `pass`). Nothing failing means no work: the baseline fails with G5's "carry no
    signal".
- Absent or false: today's rule (G4 must pass, then G5 must pass).

## Tests

`pilot/images/tests/test_grade_dependency.py`:
- a change to a fixed path fails G2 with `MANIFEST_CHANGED`, even under `allowed_paths: ["**"]`;
- a change elsewhere passes G2 as before;
- a renamed or deleted fixed path fails G2 too;
- a dependency baseline with failing visible tests passes, with G5 `skipped` (not measured);
- a dependency baseline with passing visible tests needs failing hidden tests;
- a dependency baseline where nothing fails fails (no signal);
- without `dependency`, failing visible tests still fail the baseline;
- a meta without the fields gives the same results as before (one existing case re-run).
