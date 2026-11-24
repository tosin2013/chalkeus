# Research: Brownfield intake (029)

Each decision: what we chose, why, and what we rejected.

## R1. The report's credential (FR-008)

- **Decision:** a new `ghloop.ReportClient(target)`:
  - **App mode** (`FORGE_PUBLISHER_CREDENTIAL=app` for the target, with `FORGE_APP_ID`,
    `FORGE_APP_INSTALLATION_ID` and the stored App key, as `ghapp` uses today): mint an installation token for the one target repository with **read**
    permissions only (`contents`, `pull_requests`, `checks`, `statuses`; `vulnerability_alerts`
    asked separately, see below). The token stays in memory: nothing is written to
    `app-token*.env`, unlike `ghapp.Mint`.
  - **otherwise:** `ghloop.OperatorClient()` (`GH_TOKEN`, `GITHUB_TOKEN`, then `gh auth token`),
    the read client the GitHub loop, `gate` and `doctor` already use.
- **Why:** owner decision 2026-11-24. The App is the owner's normal path, and GitHub lets an
  installation token be narrowed to a subset of the App's permissions, so the report never holds
  a write permission. The publisher's credential stays the only one that writes (C-03).
- **Optional permissions:** GitHub refuses a token request that names a permission the App
  wasn't granted (HTTP 422). So the report asks for the required set (`contents`,
  `pull_requests`) first, then adds `checks`, `statuses` and `vulnerability_alerts` one request
  at a time, and records which it got. A missing one shows as `unknown` in its column, and the
  report names the App setting to grant (Checks: read, Commit statuses: read, Dependabot alerts:
  read). It never fails open to `safe` (spec edge cases).
- **Rejected:** the publisher token or a full-permission App token (a write credential for a
  read-only job); a new read-only token (another secret to store and rotate).

## R2. Reading GitHub: API only, no clone

- **Decision:** the report reads through the REST API with the R1 client: open PRs, each PR's
  head check runs and combined status, the default branch's head checks, and the default
  branch's manifest files through the contents API. No clone and no git credential.
- **Why:** FR-007 (no cluster, no running the target's code); private repositories work with the
  same credential; one code path is tested against `internal/fakegh`.
- **Limits:** at most 100 open bot PRs per report (one page); more are counted and named as
  `not triaged: over the limit`. Every API call is a GET; a test fails if the fake sees any other
  method (SC-002).

## R3. Recognising Dependabot PRs

- **Decision:** a PR is a Dependabot PR when its author is `dependabot[bot]`. Its packages come
  from, in order:
  1. the title, `Bump <pkg> from <a> to <b>` (optionally `in /<dir>`);
  2. for grouped PRs (`Bump the <group> group …`), the body's update table
     (`| Package | From | To |`);
  3. the head branch, `dependabot/<ecosystem>/<dir>/<pkg>-<version>`, for the ecosystem.
- Other bot authors (`renovate[bot]`, others of type Bot) and Dependabot ecosystems with no
  catalog toolchain (`github_actions`, `docker`, …) are `not triaged`, with the reason.
- **Rejected:** Dependabot's GraphQL "dependencies" fields (not available to every token) and
  parsing the PR's diff (heavy; the title and table are stable Dependabot output).

## R4. Classification rules (FR-003)

Each PR gets exactly one class, the first that matches:

1. **`not triaged`** — R3's cases, or a package it can't parse.
2. **`superseded`** — the default branch's manifest already resolves every package the PR bumps
   to the PR's version or newer (R5), or another open Dependabot PR bumps the same packages to a
   newer version.
3. **`pending`** — some head check hasn't finished, and none has failed.
4. **`needs code changes`** — the bump is a major level for any package, **or** the PR fails a
   check that the default branch's head doesn't fail.
5. **`safe`** — every remaining failing check also fails on the default branch's head (inherited,
   not caused by the PR), and every bump is patch or minor.

- **Amended 2026-11-24 (T007, on documcp's real checks): shared checks.** A failing check is
  **shared**, and doesn't count against any one PR, when it also fails on the default branch's
  head, or when it doesn't run there and fails on more than half of the PRs whose checks were
  read (at least 3). documcp's auto-merge jobs only run on PRs and fail on all 22 because the
  required `Security Audit` is red; counting them as each PR's own failure marked every patch
  bump `needs code changes`. A PR-only check that fails on a few PRs (`Test Suite (20.x)` on two)
  stays the PR's own. Rules 4 and 5 read "own" and "shared" in place of "the default branch
  doesn't fail" and "also fails on the default branch". A check name that appears more than once
  (several workflows with one job name) is failing when any run of it fails.
- **Why rule 4/5 compare with the default branch:** documcp's case. All 22 PRs failed the
  production `npm audit`, which was red on `main` itself, so the failure said nothing about each
  bump. Treating an inherited failure as the PR's would mark every PR `needs code changes` and
  hide the real ones. The report states the inherited checks in the group heading and as a
  baseline flag (R7), so the owner sees the shared root cause first.
- **Grouping (FR-002):** the key is the sorted set of the PR's failing check names (inherited and
  its own). PRs with no failing check share the group `no failing checks`.
- **Semver level:** from the from/to versions (`major` when the first non-zero component
  changes, so `0.3 → 0.4` is major); versions that aren't semver give `unknown`, which counts as
  major for rule 4.

## R5. "Superseded" from the default branch's manifest (FR-005)

Per catalog toolchain, a small reader of the resolved version on the default branch:

| Toolchain | File read | Version of `<pkg>` |
|---|---|---|
| `node24` | `package-lock.json` (v2/v3) | `packages["node_modules/<pkg>"].version` |
| `python3.13` | `requirements*.txt` | an exact `<pkg>==<v>` pin (names normalised per PEP 503) |
| `go` | `go.mod` | the `require` line's version |
| `jdk21-maven` | `pom.xml` | a literal `<version>` of the dependency; a `${property}` resolved from `<properties>` in the same file |

- Anything the reader can't resolve (an unpinned requirement, a parent POM, a version range) is
  `unknown`, which never makes a PR `superseded`.
- The files are listed in the catalog (R8), and in the PR's directory (`in /<dir>`) when it has
  one.

## R6. Advisories (FR-004)

- **Decision:** with `vulnerability_alerts` read (R1), list the repository's open Dependabot
  alerts and match them by package and fixed version: a PR "fixes" an alert when its to-version
  is the alert's first patched version or newer. Without the permission, the column is
  `unknown`.
- **Rejected:** parsing the advisory links from the PR body (Dependabot only adds them to some
  PRs, and not to grouped ones consistently).

## R7. Baseline flags (FR-006)

Read statically, from the API, without running anything:

| Flag | How it's found |
|---|---|
| `red default branch` | a failing check run or status on the default branch's head, naming each |
| `force exit` | `--forceExit` (Jest) or `--exit` (Mocha) in `package.json` scripts; `forceExit: true` in `jest.config.*` or the `jest` key of `package.json` |
| `install scripts` | `package-lock.json` entries with `"hasInstallScript": true`, counted and named. Through forge's proxy a native build that downloads headers fails (documcp friction F11); the flag links the `--ignore-scripts` advice in docs/targets.md |

- **Deferred:** flaky tests and tests that write into the working tree need the tests to run.
  forge's baseline already runs them before any attempt; a report that runs them would break
  FR-007. Recorded as a note in the docs instead.

## R8. Manifest files in the catalog

- **Decision:** each entry of `pilot/images/toolchains.json` gets `"manifests"`, a list of globs:
  - `node24`: `package.json`, `package-lock.json`, `**/package.json`, `**/package-lock.json`;
  - `python3.13`: `requirements*.txt`, `**/requirements*.txt`, `pyproject.toml`;
  - `go`: `go.mod`, `go.sum`, `**/go.mod`, `**/go.sum`;
  - `jdk21-maven`: `pom.xml`, `**/pom.xml`.
- `internal/images` reads it; the report (R5) and the dependency-item gate (R9) use it.
- **Amended 2026-11-24 (T001):** the table lives in Go (`internal/images/manifests.go`), matched
  against a path's base name, not in `toolchains.json`: that file triggers the image workflow, and
  this metadata changes no image.
- **Note (superseded by the amendment above):** `toolchains.json` is under `pilot/images/`, which triggers the image workflow, so
  this change rebuilds the images anyway; R11's `grade.py` change needs that rebuild too (one
  digest PR).

## R9. Declaring an approved manifest change

- **Decision:** a `## Manifest change` section in the item's `spec.md` (beside
  `## Allowed paths`), contract in `contracts/item-spec.md`:

  ```markdown
  ## Manifest change

  - Commit: `<40-hex commit>`
  - Files: `package.json`, `package-lock.json`
  ```

- **The change is the commit's own diff against its first parent.** A Dependabot PR is one
  commit on top of the default branch as it was, so its tip is the change. An owner's own bump is
  committed as one commit (on any branch or PR of the target).
- **Gate rules** (readiness, before any cluster work; codes in `contracts/item-spec.md`):
  - `MANIFEST_CHANGE_UNRESOLVED`: the commit can't be fetched from the target (GitHub serves
    any commit reachable from a branch or a PR ref);
  - `MANIFEST_CHANGE_NOT_MANIFEST`: a file listed isn't a manifest file of the target's
    toolchain (R8), or the commit's diff touches a file that isn't listed;
  - `MANIFEST_CHANGE_STALE`: a listed file at the run's base differs from the commit's parent:
    the default branch moved those files since. The owner rebases the bot PR or prepares a new
    change; forge never does (scope out);
  - `MANIFEST_CHANGE_EMPTY`: the listed files are the same at the commit and its parent.
- **Rejected:** naming a PR number (it moves as the bot rebases; a commit is exact and is what
  the owner reviewed); a separate plan file (the item's spec already carries its grading inputs).

## R10. Applying the change: the snapshot, the replay

- **Snapshot:** today `fixture.Snapshot` makes a parentless commit of the base tree. For a
  dependency item, the snapshot's tree is the base tree with the listed files replaced by their
  blobs at the approved commit (`git read-tree` + `update-index` + `write-tree`). The worker, the
  baseline and the prefetch all start from the snapshot bundle, so all three see the same
  manifest (FR-012, FR-013) with no further change.
- **Replay:** `fixture.ReplayAs` checks that the snapshot's tree is the base's and rebuilds the
  worker's commits on the base. For a dependency item it first makes one commit on the base with
  the snapshot's tree (author and subject from the approved commit; body naming the commit and
  `Chalkeus-Run:`), checks its tree equals the snapshot's, and replays the worker's commits on
  top. The publish bundle (`base..HEAD`) then holds both commits, so the draft PR contains the
  approved change byte for byte and the fixes (FR-015). The publisher is unchanged.
- **Why two commits:** the owner reviews the code changes apart from the change they approved.

## R11. G2 holds the manifest files (FR-014)

- **Decision:** the grading meta gets `fixed_paths` (the listed files). In
  `pilot/images/tools/grade.py`, any changed path in `fixed_paths` fails `G2_allowed_paths`
  with `MANIFEST_CHANGED: <file>`, whatever `allowed_paths` allows. Since the worker starts from
  the snapshot, "unchanged from the snapshot" is exactly "equal to the approved change".
- The tools image is rebuilt by the image workflow (C-09), and its digest PR updates
  `pilot/images.lock`; the integration harness runs on the new digest.
- **Rejected:** a host-side check before replay only (the grader is the authority for grading,
  and its result is what the audit reads).

## R12. The baseline for a dependency item (spec US2 scenario 4)

- **Decision:** the baseline runs on the snapshot, as today, with `meta.dependency = true`:
  `G4_visible_tests` is recorded but a failure doesn't fail the baseline (those failures are the
  work); `G5_hidden_tests` runs even when G4 failed and must still fail (the hidden tests carry a
  signal). The run says how many visible tests fail after the change, so the owner sees the size
  of the work at the start.
- **Trade-off:** today's baseline also proves the project's tests pass on the unchanged code. A
  dependency item doesn't re-prove it; the report's `red default branch` flag covers it before
  the item is prepared.
- **Rejected:** a second baseline on the unchanged base (another grader pod and bundle per run,
  for a fact the default branch's CI and the report already show).

## R13. Dependency items and slices

- **Decision:** not combined in 029. A dependency item is graded as one slice; the gate refuses
  `run --slice` for it with `MANIFEST_CHANGE_SLICED`. After a first slice merged, the change
  would already be in the base (stale by R9), so slicing needs its own design.

## R14. Mode approval and evidence (FR-011)

- The mode text adds: the commit (short), the files, and each package's from → to version,
  read from the diff with R5's readers.
- The run records `manifest_change` in `run.json`'s state and writes `manifest-change.json`
  (commit, parent, files, each file's blob before and after, packages). The audit is unchanged:
  its rows already cover the grading, the replay and the publish (FR-016).

## R15. Where the command lives

- `./chalkeus forge debt --target <t> [--json]`: read-only, so the command audit classes it
  `free` (no harness ask rule); `internal/cli/command_audit.go` gets the row.
- Text output for the owner; `--json` is the data model's `TriageReport`. Exit 0 with a report
  (even an empty one), 2 for a config or credential error.
