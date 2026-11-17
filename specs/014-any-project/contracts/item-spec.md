# Contract: what a prepared item's spec must contain

014 reads the item's `spec.md` for one new section, plus the existing Spec Kit structure.

## Allowed paths

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
