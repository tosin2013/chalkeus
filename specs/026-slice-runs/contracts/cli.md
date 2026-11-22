# Contract: CLI changes for slice runs

Every flag is optional. Without `--slice`, every command behaves exactly as today (FR-005,
SC-001). Slice ids are given as listed by `pilot slices`, case-insensitive (`us2` = `US2`).

## `pilot run --target <t> --item NNN [--slice <id>]`

- **With `--slice`:** readiness adds the slice rules (see slices.md). On allow, it builds,
  grades and publishes one slice. The PR has branch `pilot/<run>/<item>-<id lower>` and title
  `<type>(NNN): <roadmap title> (<id>)`.
- **Exit codes are unchanged:**
  - a slice refusal exits like any readiness refusal, with the reason code in the output;
  - a non-last slice that merged ends with the run `succeeded` and no status proposal. It prints
    `slice US1 merged; next: --slice US2`.
- `--slice ALL` is refused (`SLICE_UNKNOWN`). The whole item is run by leaving out `--slice`.

## `pilot slices --target <t> --item NNN` (new, read-only)

Prints the item's slices from spec-status on a fresh clone of the default branch:

```text
item 006 — 3 slices (from spec status 0.4.0)
  US1  merged     5/5   Phase 1: Setup · Phase 2: Foundational · Phase 3: User Story 1 — …
  US2  untouched  0/4   Phase 4: User Story 2 — …
  US3  untouched  0/6   Phase 5: User Story 3 — … · Phase 6: Polish
current: US2
```

- `--json` gives the same data as `[{id, index, state, checked, total, phases}]`.
- Exit codes: 0 when evaluable; 2 when `tasks.phases` is missing (`SLICE_UNEVALUABLE`).

## `pilot doctor --target <t> --item NNN [--slice <id>] [--baseline]`

Reports the slice rules as part of its one-pass report. `--baseline` runs the slice baseline:
this slice's hidden tests fail and earlier slices' pass (grading.md). `--baseline --slice` uses
the same staging as a slice run's baseline (`grading/earlier/<n>/` plus this slice's overlay).

## `pilot approve acceptance` (new decision kind)

Records the item-level acceptance check for a paused run that waits for it. It is valid only
while the run's state is `paused:acceptance`.
- On GitHub, the equivalent is `/approve acceptance` from an approver.
- The pilot skill's rule holds: an agent runs it only after the human said yes to that exact
  step.

## `pilot author start|watch … [--slice <id>]`

- `start --slice` drafts into `items/NNN/<id>/`.
- `watch` picks each candidate item's current slice by itself (research R9).

## `pilot audit --target <t> --item NNN` (new form)

Prints one line per local run of the item (whole-item and slice runs), with its slice, state and
audit result, then the newest run's full audit. `pilot audit [--run <id>]` is unchanged.

## `pilot step`

No new flags. When a proposed item has slices, step runs `doctor --baseline --slice <current>`
and then `run --slice <current>`.
