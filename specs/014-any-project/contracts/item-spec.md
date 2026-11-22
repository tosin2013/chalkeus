# Contract: what a prepared item's spec must contain

014 reads the item's `spec.md` for one new section, plus the existing Spec Kit structure.

## Allowed paths

> **Sliced items (026):** a spec may also give `## Allowed paths (<slice>)` per slice, and the
> companion may hold one folder per slice (`items/NNN/US1/` …). See
> `specs/026-slice-runs/contracts/slices.md`.

```markdown
## Allowed paths

- `src/main/java/com/example/restservice/*`
- `src/test/java/com/example/restservice/*`
```

- **Shape:** a level-2 heading `Allowed paths`, then list items. Each list item's first
  backticked span is one glob.
- **Matching:** the glob uses G2's fnmatch, where `*` also matches `/`. So
  `src/main/java/com/example/*` covers every file below that folder.
- **Always added:** the item's own `<spec dir>/tasks.md`.
- **Refused (`ITEM_PATHS_TOO_WIDE`):** a glob that would match `.github/…`, `.specify/…`,
  `.specify/memory/roadmap.md`, or any other item's spec folder.
- **No section, or no glob:** `ITEM_NO_ALLOWED_PATHS`.

## Tests that may change (optional)

```markdown
## Tests that may change

- `tests/test_tools.py` (the expected error text changes)
```

- **Used only when the target sets `PILOT_TEST_PATHS`.** The grader's G7 re-runs the base's
  version of every test file the change edited or deleted; the files listed here are skipped,
  because the item is meant to change what they expect.
- **Shape:** as `## Allowed paths`. List only what the spec really changes; the human reviewing
  the spec decides.

## Hidden tests (companion repository)

```text
<companion>/items/NNN/          # mirrors the project's layout
└── src/test/java/com/example/restservice/AcceptanceNNNTest.java
```

- **Applying them:** the folder is copied over the changed project, then the project's test
  command runs.
- **Dependencies:** tests may only use dependencies that the base revision's build already
  declares (prefetch resolves the base plus the overlay; the grader is offline).
- **They must fail on the base revision.** If they pass there, the run refuses with
  `BASELINE_HIDDEN_PASSES`.
