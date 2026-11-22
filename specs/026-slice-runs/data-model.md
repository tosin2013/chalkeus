# Data model: Slice runs

All of it is derived or recorded data. Nothing here is a new store: slices are computed from
spec-status JSON every time (research R2, R3).

## Phase (input, from `spec status --json`)

| Field | Type | Notes |
|---|---|---|
| `heading` | string | the level-2 heading text, without `## ` |
| `line` | int | the heading's line in `tasks.md` |
| `story` | int or null | N from "User Story N", else null |
| `checked`, `total` | int | checkbox tasks under the heading |

Lives at `items[].tasks.phases[]`, in file order (spec-status 0.4.0+):
- missing or `null`: slices are unevaluable (R1);
- `[]`: the item has one slice, the whole item.

## Slice (derived, `internal/slice`)

| Field | Type | Notes |
|---|---|---|
| `id` | string | `US<n>`, or `ALL` for an unsliced item |
| `index` | int | 0-based, in file order |
| `phases` | []string | the phase headings in this slice (R2) |
| `checked`, `total` | int | sums over its phases |

Derived states (on the target's default branch):
- **merged:** `checked == total`, and no open pilot PR for the slice (R3)
- **untouched:** `checked == 0`
- **started:** anything else. The run is refused with `SLICE_STARTED`.

**Current slice:** the first slice that isn't merged. Step mode (R9) and the test author use it.

**Validation:**
- ids are unique within an item;
- an item has at least one slice;
- a slice id given on the command line must match exactly. Lower case is accepted and
  normalized, so `us2` means `US2`.

## Slice run (evidence, `run.json` additions)

| Field | Notes |
|---|---|
| `slice` | the slice id; absent in whole-item runs (that absence keeps old evidence valid) |
| `state.readiness.slice` | `{id, index, of, phases, earlier: [ids], allowed_paths_source: "slice" or "item", default_revision}`. `default_revision` is the default-branch commit the earlier slices were read at (AU13). |
| `state.companion.folders` | the ordered companion folders overlaid (earlier slices, then this one) |
| `slices_last` | true when this is the item's last slice. It decides whether a status proposal may follow. |

Events added:
- `slice_merged` (reconcile, for a non-last slice);
- `acceptance_wait` and `acceptance_recorded` (R7).

## Grader meta additions (`meta.json` → `grade.py`)

```json
"slice": {"id": "US2", "phases": ["Phase 4: User Story 2 — …"], "earlier": [["Phase 1: Setup", "Phase 3: User Story 1 — …"]]}
```

Grading inputs:
- `grading/overlay/`: this slice's hidden tests;
- `grading/earlier/<n>/`: each earlier slice's hidden tests, in order (only in slice runs).

## Decision: `acceptance` (R7)

A paused-run decision like `mode`, `retry` and `reconcile`:
- given by `/approve acceptance` from an approver, or by `pilot approve acceptance`;
- needed only when the item's spec has `## Item acceptance`;
- recorded in `approvals/` and the events. Never written by an agent on its own (AGENTS.md).

## Lifecycle of a sliced item

```text
item planned/specced ──► US1 run ─ PR merged ─► slice_merged (no status change)
                         US2 run ─ PR merged ─► [acceptance decision, if declared]
                                              ─► status proposal ─ human merge/approve ─► verified
```

Between slices the roadmap status never changes (FR-017). A verified item never gains a slice
(FR-019). The only way back is a new item, or a human editing the roadmap.
