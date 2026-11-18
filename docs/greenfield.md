# Starting a new project (greenfield)

This guide builds a project from an **empty repository**, one roadmap item at a time. You plan;
agents build each item, and you review and merge every change. It follows
[Stockroom](https://github.com/tosin2013/stockroom), a small Spring Boot inventory service that
started empty and was built this way (roadmap item 017). Stockroom's planning PR, item PRs and
roadmap are there to read as a reference.

The pilot treats a new project like any other target ([targets.md](targets.md)). The difference is
the order: **everything is planned before anything is built.** The first item is a scaffold, and
each later item grows the project from there.

Before you start, set up specfoundry with the [operator guide](operator-guide.md): `pilot.env`, a
model key, the GitHub App, and `pilot/pilot up`.

## 1. Create an empty repository

Create it on GitHub with **nothing** in it: no README, license or `.gitignore`. Install the
pilot's GitHub App on it: add the repository to the App's installation, as in
[github-app.md](github-app.md). The pilot needs it to open PRs there.

A pull request needs a branch to merge into, so the first commit on `main` is a one-line README.
Everything else arrives through the planning PR.

## 2. Plan the project in one pull request

Plan with your coding agent, which drafts while you decide. The **planning PR** holds everything
an agent needs, and your merge approves it:

| File | What it decides | Stockroom |
|---|---|---|
| Spec Kit (`specify init`), plus the roadmap and spec-status extensions | the planning tools | [`.specify/`](https://github.com/tosin2013/stockroom/tree/main/.specify) |
| `.specify/memory/constitution.md` | how decisions are made, and what every change must meet | six principles: the roadmap decides; you decide in writing; one PR with tests per change; dependencies approved in planning; offline tests; planning files and CI are yours |
| `.specify/memory/roadmap.md` | the items, their order and their status | 001 Scaffold, 002 Products API, then 003–006 planned |
| each first item's `spec.md` and `tasks.md`, in its own folder under `specs/` | the first items in detail, with their **allowed paths** | 001 and 002 specced; later items get specs when they start |
| the build file (`pom.xml`, `package.json`, …) | **every dependency** the planned items need | Spring Boot 4.1.2 on Java 21, from start.spring.io |
| `.github/workflows/` | CI | `mvn -B test` on Java 21 |

- **Declare dependencies now.** The grader runs offline, from a cache filled before any attempt
  from your approved files. An agent can't add a dependency, and an item whose tests need a
  missing one fails. Choose what the whole roadmap will need, and check the empty project
  builds (`mvn -B test` gives "No tests to run. BUILD SUCCESS").
- **The scaffold's spec** names the entry point, the configuration and the first test, and its
  allowed paths cover only source, resources and tests. The build file, CI and planning files
  stay yours.
- **Pick mainstream choices** if the project is a reference. Stockroom's PR description lists
  the dated sources behind its choices.
- **Features that need a live model** (an assistant, generated text) don't fit this loop yet:
  the grader is offline and deterministic ([targets.md](targets.md#ai-driven-apps)).

Merge the planning PR. Then protect `main` (Settings → Branches): require a pull request, and the
CI check if you like, so every change arrives reviewed.

## 3. Add a target profile

Create `~/.config/specfoundry/targets/<name>.env` ([targets.md §2](targets.md)). Stockroom's:

```sh
PILOT_TARGET_REPO=tosin2013/stockroom
PILOT_TARGET_BRANCH=main
PILOT_TOOLCHAIN=jdk21-maven
PILOT_INSTALL_COMMAND=pilot-maven-prefetch
PILOT_TEST_COMMAND=mvn -B test
PILOT_REGISTRY_HOSTS=repo.maven.apache.org
PILOT_COMPANION_REPO=tosin2013/specfoundry-acceptance
PILOT_COMPANION_PATH=stockroom
PILOT_TEST_PATHS=src/test/*
PILOT_CHANGE_TYPE=feat
```

`PILOT_TEST_PATHS` makes every item add or edit a test. `pilot/pilot targets` checks the file.

## 4. Write the first item's hidden tests

The hidden tests decide whether an item is done. They live in your private companion repository
under `<path>/items/001/`, and they must fail on today's code (for a new project, they fail
because the code doesn't exist yet).

- Write them yourself, or have the test author draft them in a **separate** session:
  `/test-author stockroom 001`. It checks them against the unchanged code and opens a PR on the
  companion. Review it and merge.
- Hidden tests may use classes the item will add. They won't compile until the item exists, and
  that's expected: the pilot still fills the cache (`pilot-maven-prefetch`).

## 5. Check, then run

```sh
pilot/pilot doctor --target stockroom --item 001 --baseline
pilot/pilot run --target stockroom --item 001
```

`doctor` reports every problem at once. With `--baseline` it also fills the cache and runs the
hidden tests on today's code, so a run doesn't fail halfway on something you could have fixed
first.

Then the run asks you at each decision:

1. **The run mode** (attempts, retries, spend cap, deadline). Approve it on the tracking issue
   with `/approve mode`, or in the terminal.
2. An agent builds the item, and the grader checks it offline: allowed paths, tasks ticked, your
   tests, the hidden tests, and that tests were added and none were weakened. A passing attempt
   becomes a **draft PR**. Review and merge it.
3. The pilot opens a **status PR** (`docs(roadmap): 001 verified`). Merging it records the item
   verified, which unblocks the items that depend on it.

Stockroom's item 001 took 13 minutes, from approving the mode to recorded verified, and passed on
the first attempt ([run record](../specs/018-test-author/evidence/20261118T161914Z/)).

## 6. The next items

Each next item repeats steps 4 and 5. A `planned` item first gets its spec and tasks: write them,
or draft them with your agent, and merge them. From the scaffold on, the project grows like any
existing one.

**When an item changes what an earlier test expects**, say so in its spec. Stockroom's 001 had a
test that `/api/products` doesn't exist yet; 002 adds that endpoint. The grader puts back the
original version of every existing test a change edits and runs it again (check G7), so that
test would have failed every attempt of 002. Listing the file in 002's spec fixed it
([stockroom#11](https://github.com/tosin2013/stockroom/pull/11)):

```markdown
## Tests that may change

- `src/test/java/dev/stockroom/StockroomApplicationTests.java`: its
  `productsEndpointDoesNotExistYet` test expects `404`; this item adds that endpoint.
```

The test author warns about this when it drafts the item's hidden tests. Planning files are
yours, so make the change in a small PR before the run.

Stockroom's item 002 (the products API) then passed on its first attempt, 5 minutes from
approving the mode to an open PR, with 20 hidden tests
([run record](../specs/018-test-author/evidence/20261118T171710Z/)).

Item 003 (Problem Details errors) needed no planning fix: no earlier test checked an error
body. It passed on its first attempt, 8 minutes from approving the mode to recorded verified
([run record](../specs/018-test-author/evidence/20261118T210446Z/)), and the run closed its
tracking issue when it ended. Item 004 (persistence in embedded H2) followed the same way:
first attempt, 11 minutes ([run record](../specs/018-test-author/evidence/20261118T212846Z/)).
Its spec kept the class the earlier tests use (`ProductStore`) and changed only what backs it,
so every existing test still passed unedited and checked that clients saw no change.

**When you review hidden tests**, check that each one tests behaviour. A test that searches the
item's source for a literal string (such as `problem+json`) can fail a correct item that writes
the same thing another way (`MediaType.APPLICATION_PROBLEM_JSON`); the grader already requires
the item to add tests (check G6).

`pilot/pilot doctor --target <name>` shows the whole roadmap and what each item still needs.
