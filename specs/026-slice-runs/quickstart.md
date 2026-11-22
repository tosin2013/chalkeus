# Quickstart: validating slice runs

How to show 026 works, from unit tests to a live two-slice item. The rules are in
[contracts/](contracts/), and the state shapes in [data-model.md](data-model.md).

## Prerequisites

- spec-status 0.4.0 or later (it adds `tasks.phases`), pinned in `pilot/pins.env`, with the
  tools image rebuilt from that pin. Check it with
  `spec status --json --root . | jq '.items[] | select(.id=="014") | .tasks.phases'`. It must
  print 7 phases.
- The pilot set up as in `docs/operator-guide.md`, and a target profile with a companion
  repository.

## 1. Unit and contract tests (no cluster)

```sh
go vet ./... && go test ./...                      # includes internal/slice, readiness, audit, step
python3 -m unittest discover -s pilot/images/tests # includes the slice grading tests
```

What they must show:
- `slice.FromPhases` gives the table in contracts/slices.md for 014's layout;
- every readiness code in contracts/slices.md refuses on its case;
- `grade.py` slice G3 passes only on the slice's ticks, and G8 runs before G5;
- the committed evidence audits unchanged (SC-001).

## 2. Whole item unchanged (SC-001)

```sh
pilot/pilot up && pilot/pilot run      # the fixture, no --slice
pilot/pilot audit                      # same rows and results as before 026
```

## 3. Two-slice item in CI (025 harness, stub model)

Add a test target with a two-story `tasks.md`, and a companion with `items/001/US1/` and
`items/001/US2/`. The scenario asserts, in order:

1. `pilot slices` lists US1 (untouched) and US2 (untouched), current US1.
2. `run --slice US2` is refused with `SLICE_EARLIER_OPEN` (SC-002).
3. `run --slice US1` passes, and its PR title ends `(US1)`. After the fake merge: `slice_merged`
   and no status PR (SC-003, first half).
4. `run --slice US2`: the grade evidence shows G8 ran and passed with the summary only (SC-005).
   After the merge, exactly one status PR (SC-003).
5. Stub attempts that tick a US2 task during US1, or edit outside `## Allowed paths (US1)`,
   fail G3 or G2 (SC-004).
6. `pilot audit --item 001` lists both runs; the US2 run's AU13 passes (SC-006).

## 4. Step mode and auto-draft (SC-008)

With `PILOT_AUTHOR_AUTO=on`, label the two-slice item, then confirm:
- step drafts `items/001/US1/`, waits for the tests PR merge, runs doctor and US1, and waits for
  `/approve mode`;
- after the US1 merge it does the same for US2;
- after the US2 merge it opens the status PR;
- the only human actions are GitHub answers: two tests merges, two mode approvals, two PR
  merges and the status merge.

## 5. Live validation (closes the item)

One real target item with at least two stories of different risk (for example a Stockroom item
with a schema change and a UI story), run slice by slice on kind with a real model. Its evidence
goes in `specs/026-slice-runs/evidence/`, and the last run audits fully after `down`.

## Acceptance check

On an item whose spec has `## Item acceptance`:
- after the last slice merges, the run pauses at `paused:acceptance`;
- `/approve acceptance` resumes it, and only then does the status proposal appear;
- the decision is in `approvals/` and the events.
