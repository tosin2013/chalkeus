# Research: Go pilot at parity

The owner made the library and policy decisions on 2026-11-15. This file records them with
their rationale, and settles the rest.

## R1 — Go version and module

- **Decision:** module `github.com/tosin2013/specfoundry` at the repository root. `go 1.26`,
  matching spec-status's `go.mod`, so importing it later needs no bump. The `toolchain` line is
  pinned to the Go release CI uses. The guide's install section adds Go, which is not yet on
  the validation machines.
- **Alternatives:**
  - a nested module under `pilot/` (rejected: the operator should share `internal/`);
  - a newer `go` floor (rejected: it would force spec-status to bump too).

## R2 — CLI framework (Cobra) and exit codes

- **Decision:** Cobra, with `SilenceUsage` and `SilenceErrors` set. Every `RunE` returns either
  nil or an `ExitError{Code int, Msg string}`. `main` maps that to `os.Exit(code)` and prints
  `pilot: <msg>` to stderr, exactly as the reference does. Unknown errors map to 2.
  - **Commands:** `github` is a parent command with `sync`, `drift` and `watch`.
  - **Flags:** same names as the reference (`--run`, `--yes`, `--check-only`, `--check`,
    `--root`, `--cluster-only`, `--local-build`, `--from`, `--model`, `--fixture-repo`,
    `--force`), plus the new `watch` flags `--detach` and `--stop` (contracts/watcher.md).
  - **Positional arguments:** the same validation (`approve` steps; `gate` tasks).
- **Help format:** `pilot/tools/docs_check.py` reads the subcommand list and flags from
  `--help`. It gets a Cobra-aware parser (the "Available Commands:" and "Flags:" sections),
  while the Python parser is kept until retirement.
- **Alternatives:** `urfave/cli` and the standard `flag` package (the owner chose Cobra; the
  Kubernetes tooling uses it).

## R3 — Cluster access (client-go)

- **Decision:** client-go, with kubeconfig loading rules (`clientcmd`). The context defaults to
  `kind-specfoundry-m1`, matching today's hard-coded context, and can be overridden by a flag
  or env var later (005). The reference's `kubectl` calls map as follows:

  | Reference (`k8s.py`) | Go |
  |---|---|
  | `apply(yaml)` | decode multi-doc YAML → unstructured → server-side apply through the dynamic client (field manager `specfoundry-pilot`) |
  | `get_json`, `ns_exists`, `job_pod`, `pull_errors`, `wait_pod_ready`, `wait_job` | typed clients + watch or poll with the same timeouts |
  | `logs` | `Pods().GetLogs()` (the framed transport is unchanged) |
  | `exec_` (P06 probes) | `tools/remotecommand` SPDY executor |
  | `can_i --as=system:serviceaccount:…` (P08) | `SubjectAccessReview` with the service account as user, answering yes or no the same way |
  | `create_secret_from_env_file`, `create_secret_from_files`, `create_configmap` | build the objects in Go (values read in-process, R7); same names, keys and labels |
  | `delete_ns` | delete + wait until gone (same timeout) |
- **Alternative:** keep `kubectl` subprocesses (rejected by the owner: a binary dependency and
  text parsing).

## R4 — kind and the container runtime

- **Decision:**
  - **kind:** its Go library (`sigs.k8s.io/kind/pkg/cluster`: `Provider.Create` with the
    same `kind-config.yaml`, `Delete`, `List`).
  - **Docker:** the `docker` CLI via `os/exec` for the registry container, `network connect`,
    image pull/build and teardown checks T02/T03. kind's library already requires the docker
    binary, and the Docker SDK would add a large dependency for a few calls.
  - **Calico:** applied through the k8s package, with the same hash check against `pins.env`.

## R5 — Git

- **Decision:** keep the `git` binary (clone, checkout, fetch from bundle, bundle create, push via
  `GIT_ASKPASS`). The `fixture` package wraps it like `fixture.py`. Pure-Go git libraries don't
  handle bundles well, and the reference's git behaviour is proven.

## R6 — GitHub

- **Decision:** `go-github` for all API calls.
  - **Read token**, resolved in order: `GH_TOKEN`, `GITHUB_TOKEN`, then `gh auth token` if `gh`
    is on PATH. If none is found, commands that need reads fail closed with a hint.
  - **Write credential:** the publisher token from `publisher.env`, or the App installation token.
  - **The two are kept separate,** so the "comments made with the publisher credential never
    count" rule still has a distinct identity to compare against.
  - **Ported in-process** (owner decision): `check_token.py` (probe), `ghops.py` (writes) and
    `app_token.py` (mint).
  - **Write probe semantics unchanged:**
    - a blob POST for contents;
    - a label POST with an invalid colour for issues: 422 means allowed;
    - pull listing;
    - the canary write must fail.
  - **No merge:** a test asserts the set of go-github methods the module calls contains no merge
    or review-approve method.
- **Alternatives:** raw `net/http` (more code to maintain); requiring `gh` (rejected by the owner).

## R7 — Secrets in-process (owner decision)

- **Decision:** credentials are read into memory only when used, as a `secret.Value` type whose
  `String()`, `GoString()` and `MarshalJSON()` all return `"[redacted]"`. They are never logged,
  never written except to their own files (mode 600), and never put in evidence.
  - **Kubernetes Secrets** are built in-process from the env and key files.
  - **The App JWT** is signed with `crypto/rsa` (RS256, iat −60 s, exp +9 min, the same claims as
    the reference).
  - **Tests** assert that captured logs and evidence never contain the test secrets. AU08's scan
    (patterns, key files, env values, PEM headers) is ported unchanged.
- **Change from the reference, recorded:** the Python pilot kept values out of its own process
  through child processes. This is accepted, because the operator (011) holds credentials
  in-process by necessity, and the threat model (one operator's host) is unchanged.

## R8 — spec-status

- **Decision, now:** run `spec status --json --root <clone>` and parse schema 1.0, as the
  reference does. A failure, an unknown schema or a missing binary all refuse.
- **Decision, later (FR-015):** ask spec-status for a public package, for example
  `github.com/tosin2013/spec-status/pkg/status` with `Load(root, opts) (Report, error)`. It
  would return the schema 1.0 structs that `--json` emits today, and a version constant. The
  gate's adapter interface (`StatusSource`) has two implementations (exec, library), and a test
  asserts they give identical reports on the fixture.
- **Cross-repo task:** tosin2013/spec-status#65 carries this API proposal (opened 2026-11-15). 016
  doesn't block on it.

## R9 — Evidence format

- **Decision:** Go structs with JSON tags that mirror the reference's fields. Files the reference
  writes with sorted keys are written from maps or alphabetically ordered structs; timestamps use
  the same `%Y-%m-%dT%H:%M:%SZ` UTC format; event names, check ids and detail strings are copied
  exactly. Parity means the same fields and meaning; whitespace may differ. The audit is the judge
  (R10).

## R10 — Golden audit check

- **Decision:** generate `specs/016-go-pilot/golden/audit-expected.json` **once from the Python
  reference**: for each committed pilot record, the rows (id, result, detail), audited in the same
  mode as `test_committed_evidence.py` (final only when `teardown.jsonl` exists). The Go test audits
  every record and compares row by row (86 rows today). Detail strings are compared exactly;
  anything that legitimately differs, such as a Python list repr like `['harness']`, is
  reproduced in Go, because operators and reviewers read these details. The file is regenerated
  only by an explicit tool run, never by the test.

## R11 — Shared vectors

- **Decision:** move the approval-rule and status-PR cases into
  `specs/004-github-loop/contracts/vectors/approvals.json` and `status_pr.json`
  (contracts/parity.md). Python tests (`test_ghapprove.py`, `test_status_pr.py`) are refactored to
  load them, so the reference proves the vectors. Go tests load the same files. The operator will
  too.

## R12 — Watcher (issue #25)

- **Decision:** the polling loop runs in a goroutine with a context. Detached mode starts the
  binary itself (`os.Executable()`) with `github watch --run <id>`:
  - **Unix:** `SysProcAttr{Setsid: true}`, stdin from `/dev/null`, stdout and stderr to
    `pilot/runs/<id>/github/watch.log`.
  - **Windows:** a `DETACHED_PROCESS` build-tagged file, compiled but not supported end to end.
  - **Single instance:** a pidfile `pilot/runs/<id>/github/watch.pid`, with a liveness check.
  - **Started by** the run's pause points when GitHub approvals are on.
  - **Stops** itself when the run leaves the paused state; `--stop` sends a signal.
  - `pilot status` shows the watcher.

  Details are in contracts/watcher.md.

## R13 — Terminal prompts

- **Decision:** open `/dev/tty` (Unix); no terminal, or any error, means "no" (fail closed), as
  `reconcile.ask_tty` does. `golang.org/x/term.IsTerminal` replaces `approvals.has_tty`. The same
  question strings are used.

## R14 — Shim and switchover

- **Decision:** `pilot/pilot` builds `pilot/.build/bin/pilot` with `go build -trimpath -ldflags
  "-X main.version=<git describe>"` when the binary is missing or older than any `*.go`, `go.mod`
  or `go.sum` file, then execs it.
  - **During US2:** `PILOT_IMPL=python|go` chooses the implementation, with Python the default.
  - **After the US2 fixture run passes:** Go becomes the default.
  - **US5:** removes the Python branch.
  - **No Go toolchain:** a clear message points at the guide's install step.

## R15 — Dependencies, CI and supply chain

- **Direct dependencies:** the libraries listed in plan.md, nothing else; the list is kept in
  `docs/dependencies.md` (new) for reviewers.
- **CI** (`.github/workflows/go.yml`):
  - `go vet`, `golangci-lint` (pinned), `govulncheck`, `go test ./...`;
  - builds for linux/darwin × amd64/arm64;
  - the Python suite until US5;
  - `docs_check`.
- **Supply chain:** `go.sum` is committed; module checksums are verified by the Go toolchain
  (sum database).
- **Dependabot (owner request, 2026-11-15),** enabled in 016 through `.github/dependabot.yml`:
  - **`gomod`** (root), weekly. The `k8s.io/*` and `sigs.k8s.io/*` modules form one group,
    because client-go, api and apimachinery must move in lockstep. Other Go modules are grouped
    by minor and patch updates. Major versions come as separate PRs.
  - **`github-actions`**, weekly. This keeps workflow action versions current, including the
    existing `docs-check.yml` and `pilot-images.yml`.
  - **Dependabot security updates** are switched on in the repository settings (an owner step,
    listed in the quickstart).
  - Every Dependabot PR runs the Go CI (vet, lint, `govulncheck`, tests, the golden audit and
    the vectors). **A human merges it; nothing auto-merges (C-04).**
  - **Not covered:** the image base images. They are passed as build arguments from
    `pilot/pins.env` and pinned by digest, which Dependabot can't update. They stay with the
    existing pilot-images workflow.

## R16 — Mixed runs and the old cluster

- **Decision:** `run.json` gains `"impl": "go"`, which is additive; the audit ignores it. `resume`
  on a run without it refuses with "this run was started by the Python pilot; reset the fixture and
  start a new run". `up` detects an existing cluster by name and reuses it when its pins match;
  otherwise it asks for `pilot down`, as the reference does today.
