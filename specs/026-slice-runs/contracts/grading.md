# Contract: grading a slice

Extends `specs/014-any-project/contracts/grading.md`. A whole-item run is graded exactly as
today. A slice run differs only as below.

## Inputs

- `meta.slice = {id, phases, earlier}`:
  - `phases`: this slice's phase headings;
  - `earlier`: one list of headings per earlier slice.
- `grading/overlay/`: this slice's hidden tests.
- `grading/earlier/<n>/`: earlier slice *n*'s hidden tests, in order. Present only when there
  are earlier slices.

## Checks

| Check | Whole-item run | Slice run |
|---|---|---|
| G1 bundle applies | as today | as today |
| G2 allowed paths | the item's | the slice's or the item's (slices.md) |
| G3 tasks checked | item `percent == 100` | this slice's phases all ticked, and every other phase plus the tasks outside phases have the same `checked` count as on base |
| G4 visible tests | as today | as today |
| G6 tests changed | as today | as today; the base is the slice's base |
| G7 existing tests kept | as today | as today; the base is the slice's base |
| **G8 earlier slices kept** | — (not listed) | overlay `earlier/*` in order, then run the project's tests: they must pass |
| G5 hidden tests | overlay, then tests | added after G8, on top of the earlier overlays: the tests must pass |

Order in a slice run: G1 → G2 → G3 → G6 → G4 → G7 → G8 → G5. A failed check marks the later
ones `skipped`, as today.

For G3, `spec status` runs on both the base checkout and the change. It finds phases by exact
heading, so a heading renamed by the change fails G3 with a detail naming it.

G8 and G5 record only `hidden_summary` (exit code, line count, sha256), never test output
(C-14).

## Baseline (before the first attempt)

| Check | Whole-item run | Slice run |
|---|---|---|
| G4 | must pass | must pass |
| G8 | — | earlier slices' tests must **pass** on the slice's base |
| G5 | must fail | this slice's tests must **fail** on the slice's base, with the earlier overlays in place |

The baseline fails when:
- G8 fails on base, with `EARLIER_SLICE_BROKEN_ON_BASE`: the default branch no longer satisfies
  a merged slice;
- G5 passes on base, with `BASELINE_HIDDEN_PASSES`, as today.

## Compatibility

- `CHECKS` stays G1–G5. G8 is in a new `SLICE_CHECKS` list, added only when `meta.slice` has
  earlier slices.
- Committed grade evidence carries no G8 and grades unchanged.
