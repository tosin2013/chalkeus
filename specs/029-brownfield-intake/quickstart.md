# Quickstart: validating brownfield intake (029)

Each scenario names the success criterion it proves. Run from the repository root.

## 1. Unit and contract tests (no cluster)

```sh
go vet ./... && go test ./...
python3 -m pytest -q pilot/images/tests/test_grade_dependency.py
```

Expected:
- `internal/debt` classifies the recorded documcp snapshot (`internal/debt/testdata/documcp-20261123/`):
  every PR failing only the inherited `npm audit` check is in one group, and every PR whose
  version `main`'s lockfile already has is `superseded` (SC-001);
- the report's tests against `internal/fakegh` record only GET requests (SC-002);
- the readiness tests cover each `MANIFEST_CHANGE_*` code;
- the grading tests pass: a change to a fixed path fails G2 in every case (SC-004).

## 2. The report on a real target (read-only)

```sh
./chalkeus forge debt --target <t>
./chalkeus forge debt --target <t> --json | jq '.counts'
```

Expected: exit 0; the credential line says `app-read` (App mode) or `operator-read`; nothing
changes on GitHub (check the PRs' timelines). With an App missing Checks: read, the checks
columns say `unknown` and the report says which App setting to grant.

## 3. A fixture dependency item on kind (SC-003)

The integration harness (025) adds a scenario with a fake target:
- a `node24`-style stub project whose test fails after a planned major bump of one dependency;
- a dependency item whose `## Manifest change` names the bump's commit;
- the stub worker fixes the code; a second stub attempt edits `package-lock.json` and must fail
  G2 with `MANIFEST_CHANGED`.

```sh
go test -tags integration ./test/integration/ -run TestDependencyItem
```

Expected: the baseline passes with visible tests failing; the bad attempt fails G2; the good
attempt passes offline; the publish bundle holds the manifest commit (its files equal the
approved commit's blobs) then the fix; the audit passes.

## 4. A live dependency item (SC-005, owner)

On documcp (candidate: zod 3 → 4), with the owner preparing the item:
1. `./chalkeus forge debt --target documcp` lists the zod PR as `needs code changes`.
2. The owner rebases the bot PR if needed, copies its head commit into the item's
   `## Manifest change`, and writes the tasks and hidden tests (for example a snapshot of every
   tool's published input schema).
3. `./chalkeus forge gate --target documcp --item NNN` allows the item.
4. `./chalkeus forge run --target documcp --item NNN`; the owner approves the mode (it shows the
   change), reviews and merges the draft PR, and approves the status change.

Expected: the run succeeds and its audit passes; the merged PR has the manifest commit and the
fix commits; Dependabot closes its own PR.

## 5. Docs (SC-006)

`go test ./internal/cli -run TestDocsCheck` passes, and `docs/targets.md` has the brownfield
section (the report, acting on it, a dependency item) and the F1, F2, F4 and F6 fixes.
