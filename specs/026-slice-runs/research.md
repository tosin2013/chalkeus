# Research: Slice runs

Decisions for 026, with the code they build on. Each says what was chosen, why, and what else
was considered.

## R1 — Where per-phase task counts come from

- **Decision:** read `items[].tasks.phases` from `spec status --json`. This shipped in
  **spec-status 0.4.0** (2026-11-22, closing tosin2013/spec-status#66), in the shape proposed
  there:

  ```json
  {"heading": "Phase 3: User Story 2 — …", "line": 110, "story": 2, "checked": 2, "total": 4}
  ```

  One Go adapter (`internal/slice`) reads it, with three cases:
  - **field absent** (spec-status 0.3.0 or older): any run, doctor or draft naming a slice is
    refused with `SLICE_UNEVALUABLE`;
  - **`null`** (the tasks file wasn't read): refused the same way;
  - **`[]`** (read, but no `## Phase` headings): the item has one slice, the whole item.

  Whole-item runs keep working unchanged in every case. Per 0.4.0's contract:
  - a phase ends at the next `#` or `##` heading;
  - `### Tests for User Story N` stays inside its phase;
  - tasks outside any phase count only toward the item totals.

  Checked on this repository: 014 gives 7 phases, with stories 2, 1, 3, 4 between Setup and
  Foundational and Polish. Its "Added during implementation" tasks fall outside phases.
- **Rationale:** constitution I and C-01 make the roadmap and spec-status JSON the only
  scheduling inputs, and constitution VI says reuse spec-status rather than re-implement it. The
  pilot already reads `tasks.percent` from the same document in three places: the gate, G3 and
  reconcile.
- **Alternatives:**
  - the pilot parses `tasks.md` headings itself: rejected by the owner (2026-11-21), because it
    needs a C-01 amendment and a second parser that could disagree with spec-status;
  - reading `ghloop.Checklist`: it is flat, with no phases;
  - 0.4.0's experimental Go library (`pkg/status`): not adopted for now. The grader image needs
    the CLI anyway, and running one binary keeps the host and the grader on the same code path.
    The library's Go signatures may also change before 1.0. Revisit if the subprocess becomes a
    problem.
- **Pin:** `pilot/pins.env` moves to 0.4.0 in PR #154. The tools image is then rebuilt by the
  image workflow.

## R2 — From phases to slices

- **Decision:** a pure function `slice.FromPhases(phases) []Slice`, following spec FR-003:
  - each phase with `story != null` starts a slice `US<story>`;
  - phases before the first story join the first story's slice;
  - a non-story phase after a story joins the slice before it, so Polish joins the last story;
  - with no story phases, there is one slice, `ALL`, and the item is unsliced.

  Each slice carries its phase headings in order. Slice order is file order.
- **Rationale:** slice ids then match Spec Kit's `[USn]` task labels and spec.md's
  `### User Story N` headings, which the test author already keys on (`author.Criteria`
  `US<n>-AS<m>`). Every rule is decided from JSON, with no text heuristics.
- **Alternatives:** an id from the phase number (`P3`), rejected in clarification; human-written
  ids on headings, rejected in clarification.

## R3 — Telling that an earlier slice is merged

- **Decision:** an earlier slice counts as merged when every phase in it has
  `checked == total` in `spec status` on a fresh clone of the target's default branch, and no
  open pilot PR exists for it.
- **Rationale:** a slice's PR ticks exactly its own tasks (G3s, R5), and only a human merge puts
  that on the default branch. So the default branch's tick state is merged reality, which
  constitution I asks for. It needs no new state store, the same reason 021 gave for its
  manager.
- **Alternatives:**
  - looking up earlier runs' evidence: evidence is per machine, and an owner may have merged a
    slice built on another laptop;
  - GitHub PR search: it says what merged, but not that the merge carried the slice's ticks.

## R4 — One run per slice, and the run's task label

- **Decision:**
  - one slice run is one evidence run, the same `kind=item`;
  - `run.json` gains `slice` (for example `"US2"`), absent for whole-item runs;
  - the task label stays the item id, so attempts, grades and events keep their names
    (`attempts/006-1`);
  - the PR branch is `pilot/<run>/<item>-<slice lower>` (for example `pilot/…/006-us2`);
  - the title is `ItemTitle` plus ` (US2)`.
- **Rationale:**
  - every per-run audit row keeps working with a one-line variant (AU03, R7);
  - 021's `NewestRun` needs only one more field compared;
  - a whole-item run's branch stays `pilot/<run>/<item>`, so its readiness regex still
    matches.
- **Alternatives:**
  - a task label `006-US2`: rejected, because it touches every place that maps a task to an
    item (`ItemOf`, attempts globs, AU01, AU11);
  - one run for all slices: that is the status quo the issue asks to change.

## R5 — Grading a slice

- **Decision:** the grader gets a `slice` object in its meta: the id, the phase headings of
  this slice, and the phase headings of earlier slices. The changes:
  - **G2:** the slice's allowed paths, from `## Allowed paths (US2)` if present, otherwise the
    item's (readiness computes them, as today).
  - **G3:** runs `spec status` on base and on the change and finds phases by heading:
    - this slice's phases must have `checked == total`;
    - every other phase, and the item's tasks outside any phase, must have the same `checked`
      count as on base.

    A whole-item run keeps today's `percent == 100`.
  - **G8_earlier_slices_kept:** new, and only in slice runs with earlier slices. It overlays
    every earlier slice's hidden tests (in order), runs the project's tests, and must pass. Then
    G5 adds this slice's overlay on top and runs again.
  - **G6 and G7:** unchanged code. Their base is the slice's base revision, which already
    contains the earlier slices, so they apply per slice (FR-014).
  - **Baseline:** this slice's G5 must fail on base and G8 must pass there (FR-015).
- **Rationale:**
  - two overlay passes tell "broke an earlier slice" from "this slice is incomplete", and the
    evidence records which one;
  - hidden output stays a summary (`hidden_summary`, C-14);
  - a new check id avoids changing G5's meaning in committed evidence.
- **Alternatives:** one overlay pass with every slice's tests together: on failure it can't say
  which slice failed.

## R6 — Companion layout

- **Decision:**
  - a sliced item keeps hidden tests in `items/NNN/<slice id>/`, each mirroring the project's
    layout;
  - an unsliced item keeps `items/NNN/`;
  - the layout is per-slice when every top-level entry of `items/NNN/` is a directory named
    after one of the item's slice ids;
  - a whole-item run of a sliced item overlays every slice's folder in order (FR-005);
  - `grader.Overlay` gains a list of source folders and keeps its deterministic tar.
- **Rationale:** old companions keep working, and no flag or config is needed.
- **Alternatives:** a manifest file in the companion: more to keep in sync, for no gain.

## R7 — Merge, status and the acceptance check

- **Decision:**
  - **after a non-last slice merges:** reconcile checks that the PR is merged, that this slice's
    phases are 100% on the default branch, and that the tree matches the graded attempt. It
    records `slice_merged` and ends the run as succeeded. It makes no status proposal and no
    roadmap edit (FR-017).
  - **after the last slice merges:** today's full check (`tasks.percent == 100`, tree match),
    then the status proposal.
  - **acceptance check:** if the spec has `## Item acceptance`, the proposal first waits for a
    new decision, `acceptance`. It goes through the existing approvals: `/approve acceptance`
    on GitHub, or `pilot approve acceptance` after the human says yes in the harness. The
    decision is recorded in the run evidence (FR-018). This also applies to unsliced items that
    declare the section.
- **Rationale:** it reuses 004's and 012's decision machinery, so the human gives it in the same
  places as every other decision. It also keeps "the pilot only proposes `verified`" exactly as
  today.
- **Alternatives:** a file in the companion recording the check: not a human decision channel
  the pilot already audits.

## R8 — Audit across slices

- **Decision:**
  - **AU03 variant for a slice run:** exactly 1 task PR, and 0 status PRs unless this is the
    last slice (then at most 1).
  - **New AU13 "slice order"**, in slice runs only:
    - every earlier slice is merged (R3, as recorded in the run's readiness record);
    - the last slice's run records exactly one status proposal for the item across the
      evidence root's slice runs of that item;
    - each earlier slice run found locally passed its own audit.

    Slices built elsewhere show as `merged, evidence elsewhere`, which is not a failure.
  - `pilot audit --target T --item NNN` prints the item view: every local slice run's
    audit, one line each.
- **Rationale:** whole-item runs keep the same rows, so the committed golden evidence audits
  unchanged (FR-022, SC-001).
- **Alternatives:** an item-level evidence folder: it adds a second evidence shape for
  auditors to learn.

## R9 — Step mode and the test author

- **Decision:**
  - **021:** `step.Next` sees an item's **current slice**: the first slice not merged (R3). It
    runs `doctor --baseline --slice S` and then `run --slice S`, and matches `NewestRun` on the
    item and the slice. When a slice run succeeds, the next pass picks the next slice. When the
    last succeeds, the item is verified as today.
  - **024:** `author.Candidates` yields `(item, slice)` for the current slice when its companion
    folder is empty. `Start` writes to `items/NNN/<slice>/`. The criteria are that story's
    `US<n>-AS*`, plus the item's FRs as context.
  - Each slice run still needs its own `/approve mode` (FR-024).
- **Rationale:** with no stored state, the current slice is computed every pass, so the manager
  can stop and restart at any time.
- **Alternatives:** step proposing all slices at once: it breaks one-in-flight and the merge
  between slices.

## R10 — Gate and readiness rules

The gate is unchanged: the item's status stays `planned` or `specced` until the last slice, so
it is never `ALREADY_VERIFIED` (FR-008). Readiness gains new reason codes:

| Code | When |
|---|---|
| `SLICE_UNEVALUABLE` | `--slice` given, but spec-status has no `tasks.phases` |
| `SLICE_UNKNOWN` | the slice id isn't one of the item's slices |
| `SLICE_EARLIER_OPEN` | an earlier slice isn't merged (R3) |
| `SLICE_STARTED` | any task of this slice is already ticked on the default branch |
| `SLICE_MAP_CHANGED` | a local slice run recorded different phase headings for a now-merged slice, or the slice count changed since a slice merged. It uses local evidence only, so a slice built on another machine falls back to `SLICE_STARTED` and `SLICE_EARLIER_OPEN` |
| `ITEM_PARTLY_SLICED` | a whole-item run of a sliced item where some slice is already merged |
| `OPEN_ITEM_PR` (widened) | an open PR for this slice, for an earlier slice, or for the whole item; a whole-item run is also refused by any open slice PR |
| `ITEM_NO_HIDDEN_TESTS` (per slice) | the slice's companion folder is empty |
