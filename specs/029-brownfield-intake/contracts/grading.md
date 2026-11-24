# Contract: grading changes (`pilot/images/tools/grade.py`)

Both are additive: a meta without the new fields grades exactly as today, so the golden audit and
every existing test are unchanged.

## `fixed_paths` (attempts and baseline)

- `meta.fixed_paths`: list of paths (exact, not globs). Absent or empty: no effect.
- In an attempt, `G2_allowed_paths` fails when any changed path (added, modified, deleted or
  renamed from/to) is in `fixed_paths`, even if `allowed_paths` matches it. The detail starts
  with `MANIFEST_CHANGED: <path>[, <path>…]`, then the usual `changed=… disallowed=…`.

## `dependency` (baseline only)

- `meta.dependency: true` changes the baseline's pass rule:
  - `G4_visible_tests` is run and recorded; a failure doesn't fail the baseline. Its detail is
    prefixed `with the approved manifest change: ` and the result is `fail` or `pass` as
    observed;
  - `G5_hidden_tests` runs whether G4 passed or failed, and must fail (as today);
  - the baseline passes when G1 passes and G5 passes (hidden tests fail), and G8 when the item
    has earlier slices (never, see R13).
- Absent or false: today's rule (G4 must pass).

## Tests

`pilot/images/tests/test_grade_dependency.py`:
- a change to a fixed path fails G2 with `MANIFEST_CHANGED`, even under `allowed_paths: ["**"]`;
- a change elsewhere passes G2 as before;
- a dependency baseline with failing visible tests and failing hidden tests passes;
- a dependency baseline whose hidden tests pass fails (no signal);
- a meta without the fields gives the same results as before (one existing case re-run).
