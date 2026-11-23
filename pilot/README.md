# Chalkeus M1 pilot

The pilot drives one governed delivery cycle for a two-task fixture roadmap (A, then B, which
depends on A), held in a throwaway GitHub repo:

1. **Gate** (`spec status --json`, fail closed)
2. **Isolated worker** (OpenCode in a plain Kubernetes Job)
3. **Independent grader**
4. **Publisher** (the only holder of the SCM token; draft PRs only)
5. **Human merge**
6. **Reconcile and approve**, after which B unblocks

It proves that **no worker starts on a blocked item**. A's first attempt is made to fail on
purpose, B stays blocked through the failure and the retry, and B runs only after A is
verified.

The same cycle runs one prepared item of **your own repository** (014): `--target <name>
--item NNN`, with a toolchain image for the project's language, a dependency cache filled before
any attempt, and hidden tests from a private companion repository. See
[docs/targets.md](../docs/targets.md).

- **Operators:** follow the [operator guide](../docs/operator-guide.md) ([troubleshooting](../docs/troubleshooting.md))
- Spec, plan and contracts: [`specs/002-m1-gated-pilot/`](../specs/002-m1-gated-pilot/)
- Step-by-step validation: [quickstart.md](../specs/002-m1-gated-pilot/quickstart.md)
- Model providers: [docs/providers.md](../docs/providers.md) · other harnesses: [docs/harnesses.md](../docs/harnesses.md)

## Prerequisites

- Docker, about 8 GB free RAM, Go 1.27 (`./chalkeus forge` builds the pilot on first use), git, and
  `gh` logged in as you.
- `spec` 0.2.0 on the host (the gate runs there). The worker, tools and egress images are
  pulled by digest from `pilot/images.lock`; they are built by the
  [`pilot-images`](../.github/workflows/pilot-images.yml) workflow from verified upstream releases.
- An empty GitHub repo named `<anything>-pilot-fixture`, and a fine-grained token for
  **that repo only** (contents: write, pull requests: write).
- Operator configuration under `~/.config/chalkeus/`
  (see [contracts/config.md](../specs/002-m1-gated-pilot/contracts/config.md)):
  - `forge.env`
  - `publisher.env`
  - `opencode/`

## Commands

```sh
./chalkeus forge import-opencode --fixture-repo <owner>/<name>-pilot-fixture  # reuse host OpenCode config; writes forge.env
./chalkeus forge publisher-token # hidden prompt -> verify scope + expiry -> publisher.env (0600) only if OK
./chalkeus forge up              # kind + Calico (hash-checked), images by digest, proxies, publisher namespace
./chalkeus forge up --local-build  # development: build pilot/images/ locally instead of pulling
./chalkeus forge reset-fixture   # reset the throwaway repo to pilot/fixture/
./chalkeus forge run             # gate, confirm mode, A attempts (confirm retries), publish A, pause
#   ... review and merge A's draft PR on GitHub ...
./chalkeus forge resume          # reconcile (approve A -> verified), gate B, B attempts, publish B, audit
./chalkeus forge down            # tear down and verify
./chalkeus forge audit           # re-check the committed evidence (AU01-AU11)
./chalkeus forge gate A|B [--root DIR]   # gate only (exit 0 allow, 10 refuse)
./chalkeus forge status          # current run state
./chalkeus forge targets         # 014: list target profiles (never prints secrets)
./chalkeus forge doctor --target <name> [--item NNN] [--baseline]  # every problem at once, read-only
./chalkeus forge gate --target <name> --item NNN   # an item's readiness (exit 10 = refused, with a code)
./chalkeus forge run --target <name> --item NNN    # an item run on your repository
./chalkeus forge approve --run <id> mode|retry|reconcile  # no terminal: approve the one pending step
#   (your agent harness asks you first; OpenCode needs the rule in your own config,
#    see docs/troubleshooting.md#prompts-answer-no)
./chalkeus forge github sync     # 004: mirror the roadmap as item issues + status labels (idempotent)
./chalkeus forge github drift [--run <id>]   # roadmap vs GitHub (exit 1 on drift)
./chalkeus forge github watch --run <id>     # resume when an /approve reply or a merge appears
./chalkeus forge app-key < key.pem           # store a GitHub App key (kept only if its check passes)
./chalkeus forge app-token --check           # mint an App token and probe its scope
```

GitHub loop settings in `forge.env` (all optional; see `docs/github-loop.md`, `docs/github-app.md`):
`FORGE_GITHUB_ISSUES=on|off`, `FORGE_GITHUB_APPROVALS=on|off`, `FORGE_APPROVERS=login,...`,
`FORGE_PUBLISHER_CREDENTIAL=token|app`, `FORGE_APP_ID`, `FORGE_APP_INSTALLATION_ID`.

Every human decision is a typed `y`. Without a terminal, the run pauses at each decision: approve it with
`./chalkeus forge approve` through your agent harness, or with `/approve <step>` on GitHub when the GitHub loop is on.

## Evidence

Each run writes `evidence/<run-id>/` in the active Spec Kit feature (`FORGE_EVIDENCE_FEATURE`
overrides it). Its layout is in
[contracts/evidence-record.md](../specs/002-m1-gated-pilot/contracts/evidence-record.md); item
runs add `kind: item`, the baseline, the prefetch and the companion's file hashes
([contracts/audit-item.md](../specs/014-any-project/contracts/audit-item.md)). Commit it after
`./chalkeus forge down`.

## Tests

```sh
go test ./...                                                              # the pilot (Go)
python3 -m unittest discover -s pilot/images/tests -t pilot/images/tests   # in-image scripts
```

## Layout

| Path | What |
|---|---|
| `../chalkeus` | The command, at the repository root: builds and runs Chalkeus (`cmd/chalkeus`, `internal/`; the runner is pluggable: plain Jobs, or Kelos in 003) |
| `images/` | `worker` (OpenCode), `tools` (grader + publisher), `egress` (tinyproxy); `common/frame.py` (both images); `tests/` (the in-image scripts' tests and a reference solution) |
| `images/toolchain-*/`, `images/toolchains.json` | 014: the toolchain images Chalkeus builds (`jdk21-maven`, `node24`, `python3.13`, `go`) and their catalog; `toolchain-samples/` are the offline sample projects each one is tested with |
| `manifests/` | Namespace, Job and proxy templates; for item runs, `worker-ns-item`, `grading-ns`, `grader-job-item`, `prefetch-job` and `toolchain-probe-job` |
| `fixture/` | Template of the throwaway repo (roadmap 001 → 002, specs, visible tests) |
| `grading/` | Hidden acceptance tests: mounted only into grader namespaces |
