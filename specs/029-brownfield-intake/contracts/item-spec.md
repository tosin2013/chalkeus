# Contract: the `## Manifest change` section of an item's spec

An item becomes a **dependency item** when its `spec.md` has this section (research R9). It sits
beside `## Allowed paths`; every other section keeps its meaning.

```markdown
## Manifest change

- Commit: `3f9c2d0e8b7a6c5d4e3f2a1b0c9d8e7f6a5b4c3d`
- Files: `package.json`, `package-lock.json`
```

- **Commit:** exactly one, 40 hex characters, in backticks. The change is this commit's diff
  against its first parent.
- **Files:** one or more backticked paths, relative to the repository root, no globs.
- Other lines in the section are ignored (the owner may add a note, for example which bot PR it
  came from).

## Readiness rules

Evaluated after the existing item rules, on the target's default branch at the run's base. Each
failure refuses the item with its code; `doctor` reports all of them.

| Code | Refused when |
|---|---|
| `MANIFEST_CHANGE_UNREADABLE` | the section exists but has no single 40-hex commit, or no file |
| `MANIFEST_CHANGE_UNRESOLVED` | the commit can't be fetched from the target repository |
| `MANIFEST_CHANGE_NOT_MANIFEST` | a listed file doesn't match the toolchain's `manifests` globs (catalog), or the commit's diff touches a file that isn't listed |
| `MANIFEST_CHANGE_STALE` | a listed file at the base differs from the commit's parent |
| `MANIFEST_CHANGE_EMPTY` | the listed files are identical at the commit and its parent |
| `MANIFEST_CHANGE_SLICED` | the run names `--slice` |

Each message names the file or commit and says what the owner can do, for example
`MANIFEST_CHANGE_STALE: package-lock.json changed on main since 3f9c2d0's parent: rebase the bot
PR (or commit the change again on the current main) and update the commit in spec.md`.

## What the run does with it

- The snapshot's tree is the base tree with the listed files taken from the commit (research
  R10). The worker, the baseline and the prefetch start from it.
- The grading meta's `fixed_paths` lists the files; G2 fails on any change to them
  (`grading.md`).
- The publish bundle holds a manifest commit on the base (the approved files, the commit's
  author and subject, a body naming the commit and `Chalkeus-Run: <run>`), then the worker's
  replayed commits.

The tasks, the hidden tests and the `## Allowed paths` list are written as for any item. The
allowed paths need not list the manifest files: the worker may not change them.
