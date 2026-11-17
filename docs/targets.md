# Your own repositories (targets)

The fixture shows the pilot's guarantees on two built-in items. A **target** is one of your own
repositories. You prepare its items with Spec Kit; the pilot then runs only items that are
prepared, under the same guarantees. This page covers what works today: configuring targets
and checking whether an item is ready. Running an item is being built (roadmap item 014).

## 1. Prepare the repository

For each item you want the pilot to build:

- **The roadmap** (`.specify/memory/roadmap.md`) lists the item. Every dependency is
  `verified`, and the item's `Spec dir` points to its folder.
- **The item's folder** has `spec.md` and `tasks.md`. The spec has no `[NEEDS CLARIFICATION`
  markers left, and no open question in the roadmap names the item.
- **The spec says what the agent may change**, in an `## Allowed paths` section. Each list
  item's first backticked text is one glob:

  ```markdown
  ## Allowed paths

  - `src/main/java/com/example/*`
  - `src/test/java/com/example/*`
  ```

  `*` also matches `/`, so the first line covers everything under that folder. The item's
  `tasks.md` is always allowed, so the agent can tick its tasks. A glob that could reach
  `.github/`, `.specify/`, the roadmap or another item's spec folder is refused.
- **Hidden acceptance tests** live in a separate **private** repository: the *companion*. They
  go in `items/<NNN>/`, laid out like the project (for example `items/002/src/test/java/…`).
  - The pilot reads them on your machine with your own git access, and gives them only to the
    grader. The agent never sees them.
  - They must fail on the current code.
  - They may only use dependencies the project's build already declares.

## 2. Configure the target

Shared settings (model, approvers, publisher mode) stay in `pilot.env`. Each target gets its
own file:

```sh
install -d -m 0700 ~/.config/specfoundry/targets
$EDITOR ~/.config/specfoundry/targets/greeting.env
chmod 600 ~/.config/specfoundry/targets/greeting.env
```

```sh
PILOT_TARGET_REPO=you/greeting-service
PILOT_TARGET_BRANCH=main
PILOT_TOOLCHAIN=jdk17-maven
PILOT_TEST_COMMAND=mvn -B test
PILOT_REGISTRY_HOSTS=repo.maven.apache.org
PILOT_COMPANION_REPO=you/greeting-service-acceptance
PILOT_GITHUB_ISSUES=off
```

- **Your repository declares none of this.** No agent change or merged PR can widen what the
  worker may reach.
- **`PILOT_TOOLCHAIN`** names an entry in `pilot/images/toolchains.json`. If no image exists
  for it yet, the pilot refuses with `TOOLCHAIN_UNAVAILABLE`. Ask for one with the
  [toolchain image request form](https://github.com/tosin2013/specfoundry/issues/new?template=toolchain-image.yml).
- **Optional keys** may only raise the toolchain's defaults: `PILOT_GRADER_MEMORY`,
  `PILOT_GRADER_DEADLINE_S` and `PILOT_CACHE_SIZE`. `PILOT_MAX_INPUT_MB` (default 50) limits
  the repository snapshot.
- **The target repository can't be this one or any target's companion.** The pilot refuses
  with `TARGET_IS_CANARY`.

List what's configured:

```sh
pilot/pilot targets
```

It prints each target's repository, toolchain, registries and companion, or why the target is
invalid. It never prints a secret or your model host. `fixture` is always listed; it is the
repository in `pilot.env`.

`pilot/pilot reset-fixture` only ever acts on the fixture. With `--target` naming any other
target it refuses with `NOT_A_FIXTURE`, because it force-pushes.

## 3. Check that an item is ready

```sh
pilot/pilot gate --target greeting --item 002
```

The check needs no cluster. It clones the target, runs `spec status`, reads the companion
repository, and prints the result as JSON. Exit 0 means ready; exit 10 means refused, with the
first failing rule's code:

| Code | What to fix |
|---|---|
| `ITEM_NOT_ON_ROADMAP` | The item isn't an active item of the target's roadmap |
| `DEP_UNSATISFIED:<id>`, `ALREADY_VERIFIED`, `UNEVALUABLE:…` | The usual gate reasons (see the [troubleshooting guide](troubleshooting.md)) |
| `ITEM_SPEC_MISSING` / `ITEM_TASKS_MISSING` | Add the spec dir to the roadmap entry, or the missing file |
| `ITEM_OPEN_QUESTIONS` | Resolve the spec's clarification markers, or the roadmap open question naming the item |
| `ITEM_NO_ALLOWED_PATHS` / `ITEM_PATHS_TOO_WIDE` | Add or narrow `## Allowed paths` |
| `ITEM_NO_HIDDEN_TESTS` | Add `items/<NNN>/` to the companion repository, or check that your git access can read it |
| `INPUT_TOO_LARGE` | The snapshot exceeds `PILOT_MAX_INPUT_MB` |
| `OPEN_ITEM_PR` | A pilot PR for this item is still open; merge or close it |
