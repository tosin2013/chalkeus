# Contract: auditing slice runs

Extends `specs/014-any-project/contracts/audit-item.md`. Whole-item runs keep exactly their
rows, so committed evidence audits unchanged (FR-022). Slice runs are item runs whose
`run.json` has `slice`.

## Row changes in a slice run

| Row | Change |
|---|---|
| AU01 | unchanged (the task label is still the item); the readiness record must also hold the slice rules, all `pass` |
| AU02 | unchanged: baseline pass, failed attempts, one final pass. The baseline includes G8 when there are earlier slices |
| AU03 | exactly 1 draft task PR whose head ends `/<item>-<slice lower>`, and status PRs: 0 unless `slices_last`, then at most 1 |
| AU11 | unchanged: every prompt carries the full spec unaltered. The slice id appears in the prompt as `Build only slice US2: …` (see Prompt) |
| **AU13 slice order** | new, slice runs only (below) |

## AU13 — slice order

Passes when all of these hold:
1. the readiness record lists every earlier slice as merged (R3), with the default branch
   revision it read;
2. no event in this run is a status proposal, unless `slices_last`;
3. when `slices_last`, the run proposed `verified` at most once, and no other local slice run of
   the same item and target proposed one.

Earlier slice runs that aren't in this evidence root (built on another machine) are reported as
`US1: merged on <rev>, evidence elsewhere`. That is not a failure, because the default branch is
the authority (R3).

Detail example: `item=006 slice=US2 of 3 earlier=[US1 merged@ab12cd3] status_proposals=0`.

## Item view

`pilot audit --target <t> --item NNN` lists every local run of the item with `slice`, `state`
and the audit result, then prints the newest run's full audit. It writes nothing.

## Prompt

The worker prompt for a slice run adds one paragraph after the task list. It names the slice,
says that only that slice's tasks are to be done and ticked, and lists that slice's phase
headings. The spec and the full `tasks.md` are still given unaltered, so AU11's hash check is
unchanged.
