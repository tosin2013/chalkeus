# Research: Integration tests in CI

## R1 — Test endpoints and test mode

- **Decision:** three settings, read from the environment:
  - `PILOT_TEST_MODE=1`;
  - `PILOT_TEST_GITHUB_API` (an http(s) URL);
  - `PILOT_TEST_GIT_BASE` (an http(s) URL ending in `/`).

  `config.Load` refuses either address without test mode (exit 2). With test mode:
  - `ghloop.API` and `fixture.BaseURL` take the addresses. Every raw call already goes through
    them (`ghloop/probe.go`, `ghapp`).
  - The publisher and test-author Jobs get `GITHUB_API` and `GIT_BASE` environment variables,
    which `publish.py`, `ghops.py` and `check_token.py` read instead of their hard-coded
    `api.github.com` and `github.com`.
  - The SCM proxy's allowlist becomes the fake server's host.
  - `config.json` records `test_mode` and both addresses.
  - The audit adds **AU12** "not a test run", failing, only when `config.json` says test mode.
    The golden data is unchanged, because older records have no `test_mode`.
- **Rationale:** the smallest change that reaches every GitHub call. It can't be set by accident
  (two variables, refused alone), and it marks its runs.

## R2 — The stub harness

- **Decision:** `pilot/images/worker/stub-opencode`, a small script. `images.BuildAll` copies it
  over `/usr/local/bin/opencode` only for a test build (`pilot up --local-build` with
  `PILOT_TEST_MODE=1`), and the release workflow never does.
  - It reads the scenario from `PILOT_STUB_SCENARIO`, which the pilot passes to worker and
    author Jobs only in test mode.
  - It writes the scenario's canned files from `pilot/images/worker/stub-scenarios/<name>/`,
    baked into the test image, into the workspace.
  - It prints one `step_finish` event (a token count) for the watchdog, and exits 0, or as the
    scenario says.
- **Rationale:** the real entrypoint, frame, watchdog, grader and author paths all run. Only the
  model's choices are canned. Published images can't contain it, because their build never
  copies it.

## R3 — The fake GitHub

- **Decision:** `cmd/fakegithub`, a Go server in this repository. It's built and run only by the
  test harness, never released.
  - **REST:** the subset the pilot, publisher and test author call:
    - `/user`; `/repos/{o}/{r}` with permissions; collaborator permission; branch protection;
    - pulls (list, get, create, edit, files, merge);
    - issues (list, get, create, edit, comments, events);
    - labels;
    - git refs, commits and trees, contents, blobs;
    - the App installation token.

    Everything is held in memory, per repository.
  - **Git:** smart HTTP through `git http-backend` (CGI) over bare repositories in a temporary
    folder. Pushes update the refs the REST side reads.
  - **Owner actions** under `/_test/`: comment as a login, add or remove a label as a login,
    merge a PR as a login, and read the state.
  - **Address:** it listens on the runner. kind's pods reach it through the Docker network's
    gateway (the SCM proxy's egress already allows non-cluster addresses).
- **Rationale:** a Go server can reuse go-github's types for its JSON, and `git http-backend`
  gives real git. It covers exactly what the scenarios call and grows with them.

## R4 — The harness

- **Decision:** Go tests in `test/integration/` behind the `integration` build tag. Each
  scenario:
  - seeds the fake: the target and companion repositories from `test/integration/fixtures/`,
    a test target profile in a temporary config folder, and approvers;
  - runs the pilot binary as a subprocess with test mode;
  - acts as the owner through `/_test/`;
  - asserts outcomes, the `checks/results.jsonl` rows, the evidence files and the audit rows.

  One `pilot up --local-build` serves all scenarios, and `down` runs at the end.
- **Rationale:** the same binary and commands an operator uses. Go tests give readable failures
  and artifacts.

## R5 — CI

- **Decision:** a workflow `integration.yml` on `ubuntu-24.04`:
  - triggers: weekly (`schedule`), and on PRs touching `cmd/`, `internal/`, `pilot/manifests/`,
    `pilot/images/` or itself;
  - steps: build, the kind tools via `pilot up`, `go test -tags integration ./test/integration/`
    with a 40-minute timeout;
  - on failure: evidence and pod logs uploaded as artifacts;
  - no secrets.
- **Coverage floor:** `.github/coverage-floor.txt` (package, minimum %), checked by the existing
  `go` workflow after its coverage step. A package below its floor fails, naming both numbers.

## R6 — Stages

Each stage is mergeable on its own:
1. test endpoints and AU12;
2. the stub harness;
3. the fake GitHub's REST and git parts;
4. the harness with the fixture scenario;
5. the automatic-drafting and step-mode scenarios;
6. the debrief gaps (including a stale-run rule for crashed `in-progress` runs);
7. the workflow and the coverage floor.
