# Your own repositories (targets)

The fixture shows the pilot's guarantees on two built-in items. A **target** is one of your own
repositories. You prepare its items with Spec Kit; the pilot then runs only items that are
prepared, under the same guarantees. This page covers configuring targets, checking whether an item
is ready, and running it (roadmap item 014). Today you can use `PILOT_TOOLCHAIN=python3.13`, with
packages installed from PyPI, or the built-in `python` (Python 3.11 with pytest) for a project
whose tests need no packages. The other toolchains follow.

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
  - **One companion can serve all your targets**, in any languages. Give each target its own
    folder and name it with `PILOT_COMPANION_PATH` (section 2):

    ```text
    my-acceptance/                                  (private)
    ├── cli-schema-mcp/items/006/tests/test_bundled_flags.py
    ├── helmdeck/items/002/internal/server/router_test.go
    └── greeting-service/items/002/src/test/java/…/GreetingTest.java
    ```

    The companion is only storage: nothing builds it as a whole. For a run, the pilot copies
    just that item's folder over a clean copy of the project, and the project's own test
    command compiles and runs it. Your editor won't resolve the tests' imports while you browse
    the companion on its own; that's expected.
  - Give access to the companion only to the people who write hidden tests, and never open it
    in the coding agent that runs the pilot: an agent that sees the tests can write code that
    passes them without doing what the spec asks. A separate test-author session may draft them
    for you (see "Drafting hidden tests with the test author" below).
  - The pilot reads them on your machine with your own git access, and gives them only to the
    grader. The agent never sees them.
  - They must fail on the current code.
  - They may only use dependencies the project's build already declares.

## 2. Configure the target

Target settings live on your machine, never in the target repository. They go in a `targets`
folder next to `pilot.env`:

```text
~/.config/specfoundry/              # or $SPECFOUNDRY_CONFIG, if you set it
├── pilot.env                       # shared: model, approvers, publisher mode
├── publisher.env                   # token mode only: the fixture's token
└── targets/                        # you create this folder; the pilot doesn't
    ├── greeting.env                # the profile for `--target greeting`
    └── greeting.publisher.env      # optional: this target's own token (section 4)
```

- **The file name is the target's name.** `--target greeting` reads `targets/greeting.env`.
  A name is lower-case letters, digits and `-`.
- **Create the folder and the file yourself, readable only by you.** For example, on a machine
  where your home is `/home/ubuntu`, the full path is
  `/home/ubuntu/.config/specfoundry/targets/greeting.env`:

```sh
install -d -m 0700 ~/.config/specfoundry/targets
install -m 600 /dev/null ~/.config/specfoundry/targets/greeting.env
$EDITOR ~/.config/specfoundry/targets/greeting.env
```

- **The pilot checks the modes:** the folder must be `0700` and each file `0600`, otherwise it
  stops with `TARGET_MODE`. If you made them with `mkdir` or an editor, run
  `chmod 700 ~/.config/specfoundry/targets` and `chmod 600 ~/.config/specfoundry/targets/*.env`.
- **Check your profiles** with `pilot/pilot targets`: it lists each one, or says why it's invalid.
- **If the file is missing**, the pilot stops with `TARGET_UNKNOWN: no target "greeting":
  create …/targets/greeting.env`, and prints the full path it looked for.

The profile's contents:

```sh
PILOT_TARGET_REPO=you/greeting-service
PILOT_TARGET_BRANCH=main
PILOT_TOOLCHAIN=jdk21-maven
PILOT_TEST_COMMAND=mvn -B test
PILOT_REGISTRY_HOSTS=repo.maven.apache.org
PILOT_COMPANION_REPO=you/my-acceptance
PILOT_COMPANION_PATH=greeting-service       # optional: this target's folder in a shared companion
PILOT_GITHUB_ISSUES=off
```

- **Your repository declares none of this.** No agent change or merged PR can widen what the
  worker may reach.
- **`PILOT_TOOLCHAIN`** names a toolchain specfoundry builds: `jdk21-maven`, `node24`,
  `python3.13` or `go` (`pilot/images/toolchains.json`). Until its image is published, the pilot
  refuses with `TOOLCHAIN_UNAVAILABLE`.
- **Dependencies:** list the package registries in `PILOT_REGISTRY_HOSTS` and set
  `PILOT_INSTALL_COMMAND`. The install runs twice: once for the agent (with the registries
  reachable), and once, before any attempt, into a cache built from your unchanged code plus the
  hidden tests. The grader then runs offline from that cache, so **an attempt can't add
  dependencies**; add new ones to the project yourself when you prepare the item. For Python
  (`python3.13`):

  ```sh
  PILOT_TOOLCHAIN=python3.13
  PILOT_REGISTRY_HOSTS=pypi.org,files.pythonhosted.org
  PILOT_INSTALL_COMMAND=pip install -r requirements.txt      # dependencies only, not -e .
  PILOT_TEST_COMMAND=PYTHONPATH=src python -m pytest -q -p no:cacheprovider tests
  ```

  The packages go into a virtual environment the pilot creates; your tests must find the project's
  own code through `PYTHONPATH` or your pytest configuration.
- **Any other language:** bring your own toolchain image (next section), or ask for one with the
  [toolchain image request form](https://github.com/tosin2013/specfoundry/issues/new?template=toolchain-image.yml).
- **If the project requires tests with every change**, set `PILOT_TEST_PATHS` to its test files
  (for example `tests/*` or `src/test/*`). The grader then adds two checks:
  - **G6:** the change must add or edit at least one test file. A change without tests fails,
    and the agent is told to add them.
  - **G7:** the change can't pass by weakening a test. The grader puts back the original version
    of every existing test file the change edited or deleted and runs your tests again; they
    must still pass. If an item is meant to change what an existing test expects, list that
    file in the spec under `## Tests that may change`.

  The test paths must also be in the item's `## Allowed paths`, or the agent can't add tests.
  The project's other gates (coverage, linters) can go in `PILOT_TEST_COMMAND` if they pass on
  your current code; the baseline check confirms that before any attempt. Either way, your
  repository's own CI runs on the PR.
- **PR titles follow [Conventional Commits](https://www.conventionalcommits.org/)**, because many
  projects squash-merge and use the PR title as the changelog entry. The item's PR is
  `feat(<item>): <the item's roadmap title>`; set `PILOT_CHANGE_TYPE` (`fix`, `refactor`, `perf`,
  `test`, `docs`, `build`, `ci`, `style` or `chore`) when the item isn't a feature. The
  status-change PR and its commit are `docs(roadmap): <item> verified`. The item's commit
  carries the same title as its PR, because GitHub squash-merges a one-commit PR with the
  commit's message; the worker's message and a `Specfoundry-Run: <run id>` trailer follow. The
  run id is also in each PR's branch name and body.
- **Authorship and sign-off.** The pilot's commits are authored by its publisher identity (your
  GitHub App or token), which is the truth: you didn't write the code. The pilot never adds
  `Signed-off-by` or `Co-authored-by` for you, because a sign-off is your own statement. If your
  project wants a human sign-off on every change (for example a DCO policy), turn on **Settings →
  General → Require contributors to sign off on web-based commits** in the target repository:
  when you squash-merge a pilot PR in the web interface, GitHub adds **your** `Signed-off-by`.
- **Optional keys** may only raise the toolchain's defaults: `PILOT_GRADER_MEMORY`,
  `PILOT_GRADER_DEADLINE_S` and `PILOT_CACHE_SIZE`. `PILOT_MAX_INPUT_MB` (default 50) limits
  the repository snapshot.
- **The target repository can't be this one or any target's companion.** The pilot refuses
  with `TARGET_IS_CANARY`.

### Your own toolchain image

A toolchain image only supplies a language's tools. It holds `/opt/toolchain/` with a
`toolchain.json` describing them. The pilot copies them into the agent's and the grader's
pods; the agent's sandbox and the grader stay specfoundry's own images. The format is in
[the toolchain image contract](../specs/014-any-project/contracts/toolchain-image.md). A
template is coming: a UBI Dockerfile for Red Hat and OpenShift, a Debian one for elsewhere, and
a GitHub Actions workflow that builds and attests the image.

```sh
PILOT_TOOLCHAIN=rust-1.83
PILOT_TOOLCHAIN_IMAGE=quay.io/you/toolchain-rust@sha256:<64 hex>
```

- The image must be pinned by digest; a tag is refused.
- Its name can't be one of specfoundry's toolchains.
- The mode approval shows it as operator-supplied, and the run's evidence records it so.
- It runs your project's tests inside the grader, so keep it trustworthy, as you would any base
  image.

List what's configured:

```sh
pilot/pilot targets
```

It prints each target's repository, toolchain, registries and companion, or why the target is
invalid. It never prints a secret or your model host. `fixture` is always listed; it is the
repository in `pilot.env`.

`pilot/pilot reset-fixture` only ever acts on the fixture. With `--target` naming any other
target it refuses with `NOT_A_FIXTURE`, because it force-pushes.

### Hidden tests: who can see what, and why

A build worker sees only the item's spec and tasks and a snapshot of the target repository, and
it has no route to GitHub. The one realistic way it could learn the hidden tests is if they
**land in the target repository**: an agent copies them there by mistake or because a file
tricked it (a prompt injection), or a bug carries them over. The next run's snapshot would then
contain them. The pilot refuses such a run (`HIDDEN_TESTS_IN_TARGET`), and the **security level**
in the target's profile decides who may write the target and read the companion:

| `PILOT_SECURITY_LEVEL` | For | The test author's identity | What keeps tests out of the target |
|---|---|---|---|
| `minimal` (default) | one developer on a laptop | your own git access | branch protection on the target, your review of every PR, the skill's rules, the `HIDDEN_TESTS_IN_TARGET` check |
| `recommended` | teams | a token that can write **only** the companion, in `targets/<name>.author.env` as `PILOT_AUTHOR_TOKEN` (mode 0600) | GitHub itself: that identity can't write the target |
| `strict` | enterprise, cluster mode | a separate GitHub App, companion only, in its own Job | as recommended, plus enforced network limits (arrives with cluster mode) |

Build-worker isolation is the same at every level. Separate identities are recommended for
teams and expected in enterprise environments, where review alone doesn't scale and who wrote
the tests versus who wrote the code must be clear. Each run records its level, and the audit
reports it. At the minimal level, **protect the target's main branch** so changes arrive only by
reviewed pull request.

### Drafting hidden tests with the test author

Instead of writing an item's hidden tests yourself, you can have a **separate** agent session
draft them. Open a new session (not the one that runs the pilot), load the test-author skill
(`.claude/skills/test-author/`), and give it the target and item. It runs:

```sh
pilot/pilot author start --target greeting --item 002    # checks the item; prints a workspace
pilot/pilot author check --target greeting --item 002    # on the cluster: your tests pass, the drafted tests fail
pilot/pilot author submit --target greeting --item 002   # opens a pull request on the companion
pilot/pilot author status --target greeting --item 002   # where the draft stands
```

- The pull request touches only the item's folder in the companion and includes a coverage
  table: which acceptance criteria of the spec each test covers, and which are uncovered.
- **You review and merge it.** Merging is the approval; the pilot never merges.
- The draft's evidence records names, hashes and results, never the tests' contents.
- The test author never runs `run`, `resume` or `approve`, and never writes the target.

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
| `HIDDEN_TESTS_IN_TARGET` | A hidden test's content is already in the target; remove it (see [hidden tests: who can see what](#hidden-tests-who-can-see-what-and-why)) |

### Check everything at once: `pilot/pilot doctor`

`gate` stops at the first problem. Before you prepare tests or start a run, `doctor` lists
**every** problem in one pass, and changes nothing:

```sh
pilot/pilot doctor --target greeting                       # profile, cluster, GitHub access
pilot/pilot doctor --target greeting --item 002            # plus every readiness rule for the item
pilot/pilot doctor --target greeting --item 002 --baseline # plus, on the cluster, offline:
                                                           # the install and test commands pass today
```

- **Profile:** the target's file, modes, keys and pinned toolchain; the test author's token at
  the recommended level.
- **Cluster:** `pilot/pilot up` completed, its images match `pilot/images.lock`, a model host is
  set.
- **GitHub:** your token reads the target and the companion; the test author's identity for the
  level; a warning if the target's branch isn't protected (or can't be checked).
- **Item:** every readiness rule, with each failure's detail (for example the snapshot size), and
  the hidden tests' coverage.
- **`--baseline`:** the project's tests pass on today's code with no network (the check that
  catches a test suite needing a database or the internet), and the hidden tests, if any, fail.
  It never shows hidden-test output.

Exit 0 means no failures (warnings allowed); exit 1 means at least one failure.

## 4. Run the item

With the cluster up (`pilot/pilot up`):

```sh
pilot/pilot run --target greeting --item 002
```

The run follows the fixture's steps, for one item:

1. **Readiness:** the checks above. A refusal exits 10, and nothing starts.
2. **Mode approval.** The mode is shown with the target, item, toolchain and the hosts the
   worker may reach (`model host(s) N` and your registries). Approve it in the terminal, through
   your agent's harness, or on GitHub with `/approve mode` if GitHub approvals are on for the
   target (`PILOT_GITHUB_APPROVALS=on` in its profile).
3. **Prepare.** The pilot snapshots the target at the checked commit (one commit, no history)
   and reads the item's hidden tests from the companion repository. The hidden tests go only to
   the grader; the evidence keeps their names and hashes in `companion.json`, never their
   contents.
4. **Baseline.** On the unchanged code, your test command must pass and the hidden tests must
   fail. Otherwise the run stops before any model spend.
5. **Attempts.** The agent works on the snapshot. Each attempt is graded on:
   - the allowed paths;
   - the item's tasks being ticked;
   - your test command;
   - the same command with the hidden tests copied in.

   A failed attempt waits for your retry approval, as for the fixture.
6. **Draft PR.** The passing change is replayed onto the real base revision, exactly as graded,
   and opened as a draft PR on the target by the publisher. **You review and merge it.**
7. **Status change.** After your merge, `pilot/pilot resume --run <id>` (or the watcher) proposes
   the item's one-line roadmap change to `verified`. Approve it in the terminal, or merge the
   status-change PR on GitHub.
8. **Audit.** The run ends with the audit. Item runs use the item-run versions of AU01, AU02,
   AU03 and AU11 (only this item, the baseline, one PR, no induction), and AU04 also requires
   that no hidden test reached the worker (P11).

`pilot/pilot resume` and `pilot/pilot approve` take only `--run`: a paused run always continues
against the target it started with. If that target's profile now names another repository, the
pilot refuses to continue.

**Publishing credential:**
- **GitHub App:** install it on the target too; the pilot mints a token for that repository only.
- **Token mode:** give each target its own fine-grained token in
  `~/.config/specfoundry/targets/<name>.publisher.env` (`PILOT_SCM_TOKEN=…`, mode 0600).
