# Contract: `./chalkeus forge debt` (new, read-only)

```text
./chalkeus forge debt --target <t> [--json]
```

- **Read-only** (FR-007): every GitHub call is a GET; nothing is written to GitHub, the target,
  the config folder or a run folder. No cluster, no model. Command audit class: `free`.
- **Credential** (FR-008, research R1): App mode mints an in-memory installation token for the
  target repository with read permissions only; otherwise forge's read client. The publisher's
  credential is never loaded. The output names which (`app-read` or `operator-read`), never the
  secret.
- `--target` is required (`requireTarget`).

## Text output

```text
debt report for tosin2013/documcp (target documcp), main @ 1a2b3c4, read with app-read
default branch: 1 failing check: npm audit (production)

22 Dependabot PRs in 2 groups
  fails: npm audit (production) [also failing on main]   20 PRs
    #301  safe                zod-to-json-schema 3.24.5 → 3.24.6 (patch)
    #305  superseded          @types/node 22.10.1 → 22.15.3: main already has 22.15.3
    #311  needs code changes  zod 3.25.76 → 4.1.12 (major) → suggest a dependency item
    …
  fails: npm audit (production), test (node 24)   2 PRs
    #318  needs code changes  typescript 5.6.3 → 5.9.3 (minor): fails test (node 24), main doesn't
    …
not triaged: 1 (#290 github_actions: no catalog toolchain)

baseline flags
  red default branch   npm audit (production)
  install scripts      package-lock.json: 1 package with an install script (tree-sitter-yaml)
                       → see docs/targets.md "Native builds"

counts: safe 11 · superseded 8 · needs code changes 3 · pending 0 · not triaged 1
unreadable: advisories (the App lacks Dependabot alerts: read)
```

- Versions and names above are illustrative.
- A PR line is `#<n>  <class>  <packages> (<level>)[: reason]`. A grouped PR lists its first
  package and `+N more`.
- The report closes with what forge doesn't do: `forge merges, closes and comments on nothing:
  act on the PRs yourself.`

## `--json`

The data model's `TriageReport`, one object on stdout.

## Exit codes

| Exit | When |
|---|---|
| 0 | a report was produced, including one with no bot PRs |
| 2 | config error, unknown target, no read credential (`ErrNoToken`), the App token request refused for the required permissions, or the repository can't be read |

A missing optional permission isn't an error: its columns show `unknown`.

# Contract: changes to existing commands

- **`run --target <t> --item NNN`** for a dependency item: readiness adds the manifest-change
  rules (`item-spec.md`); the mode approval shows the change (research R14); the baseline line
  reads `baseline NNN: pass (hidden tests fail with the approved change; K visible tests fail:
  that's the work)`.
- **`run … --slice <id>`** for a dependency item: refused, `MANIFEST_CHANGE_SLICED`.
- **`doctor --target <t> --item NNN`** reports the manifest-change rules with the others.
- **`gate --target <t> --item NNN`** refuses with the manifest-change codes (exit 10).
