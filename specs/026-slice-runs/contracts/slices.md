# Contract: slices, readiness and the companion layout

## Input: spec-status per-phase counts

The pilot reads `items[].tasks.phases[]` from `spec status --json` (spec-status 0.4.0 and
later; shape in data-model.md). It never reads `tasks.md` headings itself (C-01).
- **Field missing (older spec-status) or `null` (tasks file not read):** any command naming a
  slice refuses with `SLICE_UNEVALUABLE`, and whole-item runs are unaffected.
- **Field present with no story phases:** the item has one slice, the whole item.

## Deriving slices (pure; `slice.FromPhases`)

1. Walk the phases in file order.
2. A phase with `story = N` starts slice `USN`.
3. Phases before the first story phase join the first story's slice.
4. Any other non-story phase joins the slice before it.
5. If there are no story phases, the item has one slice, `ALL`.

Example (014's tasks.md layout):

| Phase | Slice |
|---|---|
| Phase 1: Setup | US2 |
| Phase 2: Foundational | US2 |
| Phase 3: User Story 2 — refuse… | US2 |
| Phase 4: User Story 1 — run a chosen item | US1 |
| Phase 5: User Story 3 — toolchain | US3 |
| Phase 6: User Story 4 — fixture | US4 |
| Phase 7: Polish and close-out | US4 |

Slices run in file order, so the order here is US2, US1, US3, US4. The ids follow the stories;
the order follows the file (FR-007).

A duplicate story number (two phases with `story = 2`) is refused with `SLICE_UNKNOWN` and a
detail naming both headings. The human fixes `tasks.md`.

## Allowed paths per slice

```markdown
## Allowed paths (US2)

- `src/main/java/com/example/stock/*`
```

- **Shape:** the level-2 heading `Allowed paths (<id>)`, then the same list form as
  `## Allowed paths` (014 item-spec contract).
- **Which list applies:** the slice's section if present, otherwise the item's `## Allowed paths`.
- **Every existing rule applies:** protected paths are refused with `ITEM_PATHS_TOO_WIDE`; the
  item's `tasks.md` is always added; another item's spec folder is refused.
- **Evidence:** readiness records which source applied (`allowed_paths_source`).

## Readiness rules added (run, doctor, author)

Evaluated after the existing rules, on the same fresh clone:

| Code | Refuses when |
|---|---|
| `SLICE_UNEVALUABLE` | a slice is named, but `tasks.phases` is missing |
| `SLICE_UNKNOWN` | the id isn't one of the item's slices, or is `ALL`, or story numbers repeat |
| `SLICE_EARLIER_OPEN` | an earlier slice isn't merged: its phases aren't all ticked on the default branch, or it has an open pilot PR. The detail names the earlier slice ids |
| `SLICE_MAP_CHANGED` | a local slice run of this item recorded different phase headings for a slice that is now merged, or the slice count changed since a slice merged. Only local evidence is used; without it, `SLICE_STARTED` and `SLICE_EARLIER_OPEN` catch what they can |
| `SLICE_STARTED` | this slice has a ticked task on the default branch |
| `ITEM_PARTLY_SLICED` | no slice is named, the item has 2+ slices, and some slice is merged or started |
| `OPEN_ITEM_PR` | an open PR whose head is `pilot/<run>/<item>` or `pilot/<run>/<item>-us<n>`, for this slice, an earlier one, or the whole item |
| `ITEM_NO_HIDDEN_TESTS` | the companion folder for this slice (or the item) has no test files |

Fails closed: an error reading any input is a refusal with a reason, never an allow.

## Companion layout

```text
<companion>/[<PILOT_COMPANION_PATH>/]items/NNN/        # unsliced item (today)
<companion>/[<PILOT_COMPANION_PATH>/]items/NNN/US1/    # sliced item: one folder per slice,
<companion>/[<PILOT_COMPANION_PATH>/]items/NNN/US2/    # each mirroring the project's layout
```

- **Per-slice layout:** every top-level entry of `items/NNN/` is a folder named after one of
  the item's slice ids.
- **Mixed layout** (slice folders next to other files): refused with `ITEM_NO_HIDDEN_TESTS` and
  a detail naming the stray entries.
- **Slice run:** overlays this slice's folder. Earlier slices' folders go to the grader's
  `earlier/` (grading.md).
- **Whole-item run of a sliced item:** overlays every slice's folder in order, as one overlay.
- **Hidden tests must fail on the base revision:** the rule is unchanged, applied to the slice's
  own folder on the slice's base (`BASELINE_HIDDEN_PASSES`).
