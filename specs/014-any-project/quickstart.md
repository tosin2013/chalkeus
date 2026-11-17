# Quickstart: validate 014 on kind

The owner runs this; the agent never approves, posts `/approve`, or merges. Formats are in
[contracts/](contracts/).

## 0. Checks before any run

```sh
go vet ./... && go test ./...      # unit tests, golden audit, docs check, item-run audit vectors
```

The fixture must still pass end to end (SC-005). Do this once after the feature lands:

```sh
pilot/pilot up && pilot/pilot run      # as in the operator guide; finish A and B; then down + audit = 11/11
```

## 1. Prepare the validation repository (owner)

1. Copy `spring-guides/gs-rest-service`'s `complete/` folder (Apache-2.0) as one squashed
   commit into a new public repository you own, for example `<you>/greeting-service`.
   - Delete the Gradle files.
   - Keep the license and a notice of the source.
   - Check that `mvn -B test` passes on JDK 17.
2. Run Spec Kit in it: constitution, roadmap with three items, then spec and tasks for each:
   - `001` validate `name`;
   - `002` greeting templates;
   - `003` greetings by language, depending on `002`.

   Each `spec.md` gets an `## Allowed paths` section ([contracts/item-spec.md](contracts/item-spec.md)).
3. Create a private companion repository, `<you>/greeting-service-acceptance`.
   - Add `items/001/src/test/java/...` with JUnit tests that **fail** on the current code.
     They may only use dependencies `pom.xml` already declares.
   - Do the same for `002` and `003`.
4. If you use the GitHub App, install it on `greeting-service` too.

## 2. Configure the target

```sh
install -d -m 0700 ~/.config/specfoundry/targets
$EDITOR ~/.config/specfoundry/targets/greeting.env     # keys: contracts/target-profile.md
chmod 600 ~/.config/specfoundry/targets/greeting.env
pilot/pilot targets                                     # lists fixture and greeting, no secrets
pilot/pilot app-token --check --target greeting         # App mode: fixture-like probe on the target
```

## 3. Readiness refusals (SC-002)

```sh
pilot/pilot gate --target greeting --item 003     # expect refuse: DEP_UNSATISFIED:002
```

Then, on a scratch branch of the target (`PILOT_TARGET_BRANCH`), break one rule at a time and
check that the expected code appears ([contracts/readiness.md](contracts/readiness.md)):
- remove `tasks.md`;
- add a `[NEEDS CLARIFICATION` marker;
- remove `## Allowed paths`;
- rename `items/001`;
- make a hidden test pass on the base.

## 4. Run item 001 end to end (SC-001, SC-003, SC-004)

```sh
pilot/pilot up
pilot/pilot run --target greeting --item 001
```

1. **Mode approval.** It shows the target, item, toolchain and registries. Approve it in the
   terminal, the harness, or on GitHub (`/approve mode`).
2. **Prefetch and baseline.** These run next. Expect `baseline: pass` (suite passes, hidden
   tests fail on the base).
3. **Attempts.** On a failed attempt, approve the retry if you want one.
4. **Your merge.** A draft PR opens on `greeting-service` with `Closes #N` if issues are on.
   Merge it.
5. **Status change.** The PR proposes `001: planned → verified`. Merge it, or approve it in the
   terminal.
6. **Finish:**

   ```sh
   pilot/pilot down
   pilot/pilot audit --run <id>     # 11/11, including P06r, P11, P12, T04, T05
   ```

Then confirm the isolation results by hand:
- No hidden test file name appears in the worker's evidence or the PR (P11 and a `grep` of
  `attempts/*/prompt.md`).
- The proxy denials file lists any undeclared host the worker tried.

## 5. A dependent item and a second target (SC-007)

1. Run `002`, then `003`, which is now unblocked.
2. Without editing any configuration, run the fixture:

   ```sh
   pilot/pilot run
   ```

   Each run's `run.json` names its target.

## 6. Record

Copy the run records to `specs/014-any-project/evidence/`, and fill in `evidence/README.md`
(run id, target, item, audit, notes).
