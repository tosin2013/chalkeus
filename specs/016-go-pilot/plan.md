# Implementation Plan: Go pilot at parity

**Branch**: `016-go-pilot` | **Date**: 2026-11-15 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/016-go-pilot/spec.md`

## Summary

Port the reference Python pilot (`pilot/specfoundry_pilot/`, about 4,000 lines, 208 tests) to
one Go binary, then retire the Python version. Parity is proven by:
- **a golden check:** the Go audit reproduces the reference audit's 86 results on the 8 committed
  evidence records;
- **shared vectors:** language-neutral JSON test cases for the approval rules and the
  status-change PR check;
- **a fixture run** following the operator guide, plus 004's owner validation on the Go pilot.

The one addition is walking away during a run (issue #25): the run starts a detached watcher.

Owner decisions this plan follows:
- **Libraries:** Cobra for the CLI, client-go for the cluster, go-github for GitHub.
- **Secrets:** held in-process, never logged or written.
- **GitHub reads:** a token from `GH_TOKEN`, `GITHUB_TOKEN` or `gh auth token`; gh isn't required.
- **Building:** the `pilot/pilot` shim builds the binary with the Go toolchain.
- **Host helpers** are ported into Go, so the host needs no `python3` afterwards.

## Technical Context

**Language/Version**: Go 1.26 (the `go` directive matches spec-status's `go.mod`, so it can be
imported as a library later), with a pinned `toolchain` line. In-cluster image scripts stay
Python 3.12.

**Primary Dependencies**:
- `github.com/spf13/cobra` (CLI);
- `k8s.io/client-go`, `k8s.io/api`, `k8s.io/apimachinery` (cluster);
- `sigs.k8s.io/kind/pkg/cluster` (stand-up and teardown);
- `github.com/google/go-github` (GitHub API);
- `golang.org/x/term` (terminal prompts);
- `sigs.k8s.io/yaml` (manifests).

Everything else is the standard library. The `git` and `docker` binaries are kept (R5, R4).

**Storage**: files, as today: `~/.config/specfoundry/` (`pilot.env`, `publisher.env`,
`opencode/`, `github-app.pem`) and the run evidence folder `pilot/runs/<id>/`. Formats are
unchanged (contracts of 002, 012 and 004).

**Testing**:
- `go test ./...` with client-go's fake clientset, `net/http/httptest` for GitHub, and fakes for
  `git` and `docker`;
- the golden audit check and the shared vector suite;
- the Python suite stays green until retirement;
- `pilot/tools/docs_check.py` is adapted to Cobra's help format;
- owner validation runs.

**Target Platform**: Linux amd64/arm64 end to end (kind and a container runtime). Builds also
for darwin amd64/arm64, where the pilot reports anything it can't support.

**Project Type**: CLI, a single Go module at the repository root.

**Performance Goals**: human-paced; watcher polling every 15 s, as the reference does.

**Constraints**:
- byte-for-byte identical exit codes and operator-guide wording (FR-002, FR-003);
- evidence identical in fields and meaning;
- fail closed everywhere;
- no secret in logs or evidence;
- no merge code path;
- the Python pilot stays runnable until US5.

**Scale/Scope**:
- 14 commands;
- about 25 Python modules mapped to about 15 Go packages (data-model.md);
- 8 golden records;
- about 20 shared vector cases.

## Constitution Check

| Principle | Check | Result |
|---|---|---|
| I. Roadmap is the source of truth | 016 and C-13 are in roadmap 1.10.0 (PR #26); status changes only via roadmap-write | pass |
| II. Gates fail closed | The gate keeps the reference behaviour and its fail-closed matrix (GM checks); missing or unreadable `spec` output, or a library error, means refuse (FR-006, R8) | pass |
| III. Evidence defines done | Same evidence layout; parity proven by the golden audit check and the shared vectors, not by "tests pass"; 004 validation evidence recorded | pass |
| IV. Least privilege | Cluster isolation unchanged (same manifests, images by digest, P01–P10). In-process secrets are a deliberate change from Python's child-process isolation: values never logged or persisted, a `Secret` type redacts itself in formatting, and AU08 and redaction tests are unchanged (R7) | pass, with the R7 note |
| V. Humans approve, humans merge | Approval rules ported via shared vectors; no merge code path (enforced by a test listing the GitHub calls used) | pass |
| VI. Build on, don't rebuild | Uses spec-status (tool now, library later), kind's library, client-go and go-github instead of hand-rolled clients | pass |
| VII. Repeatable and portable | The shim builds the binary reproducibly (`-trimpath`, version stamping); the same guide walkthrough; a static binary prepares for 013 | pass |

**Post-design re-check:** no new trust in workers or graders; the dependency surface is listed in
R15 and checked by `govulncheck` in CI. Still passes.

## Project Structure

### Documentation (this feature)

```text
specs/016-go-pilot/
├── spec.md, plan.md, research.md, data-model.md, quickstart.md
├── contracts/
│   ├── cli.md            # command tree, flags, exit codes, help format
│   ├── parity.md         # golden audit expectations + shared vector format
│   └── watcher.md        # detached watcher: start, single-instance, stop, status
├── golden/               # audit-expected.json generated once from the Python reference
├── checklists/requirements.md
└── tasks.md              # /speckit-tasks
```

### Source code (repository root)

```text
go.mod, go.sum                       # module github.com/tosin2013/specfoundry
cmd/pilot/main.go                    # Cobra root; exit-code mapping
internal/
├── cli/          # one file per command group (run.go, github.go, setup.go, ...)
├── config/       # pilot.env / publisher.env parsing and validation (same rules)
├── paths/        # repo paths, cluster name, fixture task map
├── evidence/     # Run, events.jsonl, run.json, check rows, summary, versions
├── audit/        # AU01–AU11 + redaction scan
├── gate/         # spec status adapter (exec now, library later) + fail-closed matrix
├── approvals/    # pending-step record, tty/harness/github channels
├── k8s/          # client-go wrapper: apply, jobs, logs, exec, secrets, SAR, pull errors
├── cluster/      # kind up/down (kind library), registry and Calico, teardown checks
├── images/       # pins.env, digests, local build fallback
├── runner/       # Jobs runner (submit, wait, collect, delete)
├── frame/        # framed-log decoding (worker and grader output)
├── worker/, grader/, publisher/   # job specs and result handling
├── reconcile/    # merge checks, status edit, status-change PR check
├── ghloop/       # GitHub client (go-github), sync/drift, tracking issue, approval reading, ghops writes
├── ghapp/        # App JWT (crypto/rsa) + installation token
├── fixture/      # git and fixture helpers (git binary)
├── pipeline/     # run/resume state machine, watch loop
└── watcher/      # detached watcher process (issue #25)
specs/004-github-loop/contracts/vectors/   # shared vectors (approvals.json, status_pr.json)
pilot/pilot                          # shim: builds and execs the Go binary (PILOT_IMPL switch during the port)
.github/workflows/go.yml             # vet, lint, govulncheck, test, build matrix
.github/dependabot.yml               # gomod (k8s modules grouped) + github-actions, weekly; human merge
pilot/specfoundry_pilot/, pilot/tests/   # reference, removed in US5
pilot/images/, pilot/manifests/, pilot/fixture/   # unchanged
```

**Structure Decision**: a single Go module at the root, so the operator (011) can later live in
`operator/` in the same module and import `internal/`.

## Port order (maps to the user stories)

1. **Foundation:** module, Cobra root with exit mapping, config, paths, evidence types,
   `Secret` type, CI, Dependabot.
2. **US1:** generate the golden expectations from Python. Port `audit` (and the redaction scan)
   and `gate` (exec `spec status --json`). The golden test reaches 86/86 and the gate matrix
   matches.
3. **Shared vectors:** extract from the Python tests into JSON; Python tests read them (the
   reference stays green); Go tests read them.
4. **US2:**
   - port `k8s`, `cluster`, `images`, `runner`, `frame`, `worker`, `grader`, `publisher`,
     `reconcile`, `approvals` (tty and harness), `pipeline`, `checks`, and the setup commands;
   - the shim gains `PILOT_IMPL=go`;
   - fixture run 11/11; then the default flips to Go.
5. **US3:** port `ghloop` and `ghapp` (including host helpers); run 004's quickstart on the Go
   pilot.
6. **US4:** the detached watcher; the walk-away validation.
7. **US5:** remove the Python pilot and tests; update docs, skill, README and `docs_check`; remove
   the shim's Python branch.

## Complexity Tracking

| Item | Why needed | Simpler alternative rejected because |
|---|---|---|
| client-go dependency tree | owner decision; native cluster access, needed by the operator anyway | `kubectl` subprocesses keep a hard dependency on a binary and on parsing its text output |
| In-process secrets | owner decision; go-github and Go crypto need the values | child-process isolation in Go adds a second binary or a subcommand for every credential operation, for little real gain on a single-user host |
| Temporary dual implementation | parity has to be proven against a running reference | a big-bang switch would lose the reference before parity is shown |
