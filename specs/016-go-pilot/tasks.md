# Tasks: Go pilot at parity

**Input**: `specs/016-go-pilot/`:
- plan.md, spec.md, research.md (R1–R16), data-model.md, quickstart.md;
- contracts/ (cli, parity, watcher).

**Tests**: included. Parity *is* the requirement (FR-005, FR-010, FR-011), and the project's
convention is a unit test for every gate and approval rule.

**Reference**: the Python pilot in `pilot/specfoundry_pilot/` defines correct behaviour until
US5. Port each Go package from its reference module (see the module → package map in
data-model.md), and keep messages, event names and check ids verbatim.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel (different files, no dependency on unfinished tasks).
- Paths are relative to the repository root.

---

## Phase 1: Setup

- [X] T001 Install Go on the dev machine (version per R1) and record it in
  `docs/operator-guide.md` §4 as a new "Go toolchain" install step. Run `docs_check`.
- [X] T002 Create `go.mod` (module `github.com/tosin2013/specfoundry`, `go 1.26`, pinned
  `toolchain`), plus `cmd/pilot/main.go` with a Cobra root command and `pilot version`, with
  the version stamped through `-ldflags "-X main.version=…"` (R1, R2).
- [X] T003 [P] Add `.github/workflows/go.yml`:
  - `go vet`, pinned `golangci-lint`, `govulncheck`, `go test ./...`;
  - builds for linux/darwin × amd64/arm64;
  - the Python suite and `docs_check` until US5 (R15).
- [X] T004 [P] Add `.github/dependabot.yml`:
  - **`gomod`** (root), weekly, with group `kubernetes` (`k8s.io/*`, `sigs.k8s.io/*`) and group
    `go-minor-patch` (other modules, minor and patch updates);
  - **`github-actions`**, weekly;
  - no auto-merge (R15).
- [X] T005 [P] Write `docs/dependencies.md`: every direct Go dependency, why it's needed, and the
  rule that everything else is the standard library (R15).
- [X] T006 Update the shim `pilot/pilot`:
  - build `pilot/.build/bin/pilot` with `go build -trimpath` when it's missing or older than any
    `*.go`, `go.mod` or `go.sum`;
  - `PILOT_IMPL=python|go` chooses the implementation, with Python the default;
  - a clear message when `go` is missing (R14).

---

## Phase 2: Foundational (blocks all stories)

- [X] T007 Implement `ExitError{Code, Msg}` and the exit mapping in `cmd/pilot/main.go`:
  - codes 0 ok, 1 fail, 2 error, 3 repo guard, 4 awaiting merge, 5 declined, 6 awaiting retry,
    10 gate refusal;
  - `pilot: <msg>` on stderr;
  - unknown errors and usage errors give 2.

  Test the mapping in `cmd/pilot/main_test.go` (contracts/cli.md).
- [X] T008 [P] Implement `internal/secret`: a `Value` type whose `String`, `GoString`, `Format` and
  `MarshalJSON` all return `[redacted]`, and `Reveal()` is used only by the HTTP and Kubernetes
  callers. Test it in `internal/secret/secret_test.go` (R7).
- [X] T009 [P] Port `internal/paths` from `paths.py`: repo root, build dir, runs root, cluster name
  `specfoundry-m1`, registry name, fixture `TASKS = {A: "001", B: "002"}`, images JSON, pins.env,
  manifests and the fixture template.
- [X] T010 Port `internal/config` from `config.py`:
  - `DEFAULTS`, `REQUIRED` and `validate_values` rules, including the 004 keys:
    - `PILOT_GITHUB_ISSUES` and `PILOT_GITHUB_APPROVALS`: on or off;
    - approvals need `PILOT_APPROVERS`;
    - `PILOT_PUBLISHER_CREDENTIAL`: token or app, "refuse to guess" when both exist;
    - the App ids must be numeric;
  - mode-600/700 checks, the repo guard (code 3), `model_files` and `check_model_dir`;
  - `public()`, which hashes egress hosts.

  Port the cases from `pilot/tests/test_config.py` into `internal/config/config_test.go`.
- [X] T011 Port `internal/evidence` from `evidence.py`:
  - `Run` (`run.json` load and save, `event()` with seq and timestamp format
    `%Y-%m-%dT%H:%M:%SZ`), `check_row`, `read_jsonl`, `write_json`, `append_jsonl`;
  - `new_run`, `load_run`, `active_run`, `write_config_json`, `write_versions`, `write_summary`;
  - `run.json` gains `"impl": "go"` (R16);
  - sorted keys wherever the reference sorts them (R9).

  Test it in `internal/evidence/evidence_test.go`.
- [X] T012 [P] Add the Cobra command skeletons with the reference's flags and positional validation
  (stub bodies return `ExitError{2,"not implemented"}`), in
  `internal/cli/{setup,run,approve,gate,audit,status,github,app}.go`. Commands: `up`
  (`--cluster-only`, `--local-build`), `down`, `publisher-token` (`--check-only`),
  `import-opencode` (`--from`, `--model`, `--fixture-repo`, `--force`), `reset-fixture` (`--yes`),
  `run`, `resume` (`--run`), `approve {mode|retry|reconcile} --run`, `gate {A|B} [--root]`,
  `audit [--run]`, `status [--run]`, `github sync|drift [--run]|watch --run [--detach|--stop]`,
  `app-key`, `app-token --check`.
- [X] T013 Teach `pilot/tools/docs_check.py` Cobra's help format ("Available Commands:" and
  "Flags:"), keeping the Python parser while `PILOT_IMPL` allows both. Run it against both
  implementations, and extend `pilot/tests/test_docs_check.py`.

**Checkpoint:** `go build ./...` and `go test ./...` pass; `pilot/pilot version` works with
`PILOT_IMPL=go`.

---

## Phase 3: User Story 1 — the audit and the gate give the same answers (P1) 🎯 MVP

**Goal:** the Go audit matches the reference's 86 results on the 8 committed records, and the gate
verdicts match.

**Independent test:** `go test ./internal/audit/... ./internal/gate/...` passes the golden check,
plus quickstart §2.

- [X] T014 [US1] Write `pilot/tools/golden_audit.py`: it runs the Python reference audit over
  every `specs/*/evidence/<run-id>/` that has a `run.json`, using the same mode as
  `test_committed_evidence.py` (final iff `teardown.jsonl` exists). It writes
  `specs/016-go-pilot/golden/audit-expected.json` in the contracts/parity.md format
  (`generated_by`, and per record: `path`, `final`, and `rows` of `{id, result, detail}`).
  Run it and commit the output (8 records, 86 rows).
- [X] T015 [P] [US1] Write the golden test `internal/audit/golden_test.go`: load
  `audit-expected.json`, audit each record with the Go audit, and require the same ids in the
  same order, with identical `result` and `detail`.
- [X] T016 [US1] Port `internal/audit` from `audit.py`:
  - AU01–AU11 exactly, including the detail strings (Python list reprs such as `['harness']`
    reproduced verbatim);
  - `_channels` and `_approvers`;
  - the AU03 rule with `kind`;
  - the redaction scan: regex patterns including PEM private-key headers, key files, and env
    values;
  - `run()`, which writes `audit.jsonl`.

  Then wire `pilot audit [--run]` in `internal/cli/audit.go`. T015 must pass at 86/86.
- [X] T017 [P] [US1] Port the reference audit unit cases (`test_audit.py`, `test_audit_github.py`,
  `test_redaction.py`) into `internal/audit/audit_test.go`, using a Go evidence builder that
  mirrors `pilot/tests/evidence_builder.py`.
- [X] T018 [US1] Port `internal/gate` from `gate.py`, behind a `StatusSource` interface with the
  `ExecStatus` implementation (`spec status --json --root <dir>`). Schema 1.0 is required:
  an unknown schema, a missing binary or a parse error all refuse (R8). Keep
  `evaluate`/`evaluate_fresh` semantics, reasons and `out_dir` records. Wire `pilot gate {A|B}
  [--root]` (exit 0 allow, 10 refuse).
- [X] T019 [P] [US1] Port the gate fail-closed matrix (`checks.gate_matrix`, GM rows) and the
  `test_gate.py` cases into `internal/gate/gate_test.go`, plus the fixture-verdict check.
- [X] T020 [P] [US1] Open an issue in tosin2013/spec-status proposing the public package
  `pkg/status` with `Load(root, opts) (Report, error)`, returning the schema 1.0 structs and a
  version constant (R8). Link it from `specs/016-go-pilot/research.md` R8.

**Checkpoint:** golden 86/86. Gate A allows and B refuses with reasons identical to
`PILOT_IMPL=python`.

---

## Phase 4: Shared vectors (blocks US3; serves FR-011)

- [X] T021 Write the shared vectors, in the contracts/parity.md format:
  - `specs/004-github-loop/contracts/vectors/approvals.json`: one case per R3 rule (accept,
    `wrong_step`, `before_decision`, `not_an_approver`, `no_write_access`,
    `publisher_identity`, `app_authored`, `edited`, `withdrawn`, `duplicate`, a withdrawal by
    another user, extra words);
  - `specs/004-github-loop/contracts/vectors/status_pr.json`: exact edit, `NOT_MERGED`,
    `STATUS_PR_EDITED` (extra file, extra line, wrong item), `STATUS_PR_MERGED_BY_PUBLISHER`,
    `STATUS_PR_NOT_APPROVER`.
- [X] T022 Refactor `pilot/tests/test_ghapprove.py` and `pilot/tests/test_status_pr.py` to load the
  vector files, and add `pilot/tests/test_vectors.py` running `ghapprove.evaluate` and
  `reconcile.check_status_pr` on every case. The Python suite stays green.

---

## Phase 5: User Story 2 — a fixture run with the new pilot (P1)

**Goal:** the full guide walkthrough on the Go pilot: 11/11, the same exit codes and wording, and
the harness channel working.

**Independent test:** quickstart §3, at a terminal and through an agent.

- [X] T023 [P] [US2] Port `internal/k8s` (R3) using client-go with the kubeconfig and context
  `kind-specfoundry-m1`:
  - `Apply` (server-side apply of multi-document YAML; field manager `specfoundry-pilot`),
    `Render` (the same `${VAR}` template substitution as `k8s.render`);
  - `WaitJob`, `JobPod`, `Logs`, `DeleteNS` (wait until gone), `NSExists`;
  - `CreateSecretFromEnvFile`, `CreateSecretFromFiles`, `CreateConfigMap` (same names, keys,
    labels);
  - `CanI` (SubjectAccessReview for `system:serviceaccount:<ns>:<sa>`; answers yes or no),
    `Exec` (remotecommand);
  - `PullErrors`, `WaitPodReady` (same timeouts).

  Test it with the fake clientset in `internal/k8s/k8s_test.go`.
- [X] T024 [P] [US2] Port `internal/frame` from `frame.py`, with the reference's
  `test_frame.py` cases in `internal/frame/frame_test.go`.
- [X] T025 [P] [US2] Port `internal/fixture` from `fixture.py`: the git binary wrapper, `gh`
  replaced by go-github reads (R6), `fresh_clone`, `reset` (same prompt "Type y to continue: "),
  `open_pilot_prs`, `url`. Wire `pilot reset-fixture [--yes]`.
- [X] T026 [P] [US2] Port `internal/images` from `images.py`: pins.env, digest references (tags
  rejected with the same messages), `--local-build`, `load()`. Port the cases from `test_pins.py`
  and `test_image_source.py`.
- [X] T027 [US2] Port `internal/cluster` from `cluster.py` and `infra.py`:
  - kind's library: create with `pilot/kind-config.yaml`, delete, list;
  - the registry container and `docker network connect` via the docker CLI;
  - the Calico hash check;
  - egress proxies, the publisher namespace and the `scm-token` Secret (credential per
    `config.credential`), check helpers, `fail_on_pull`;
  - teardown checks T01–T03 to `teardown.jsonl`;
  - the `up.ok` marker.

  Wire `pilot up [--cluster-only] [--local-build]` and `pilot down`. Port
  `test_calico_hash.py` and `test_pull_errors.py`.
- [X] T028 [US2] Port `internal/runner` (the `base.py` interface and the `jobs.py` Jobs runner:
  submit, wait, collect, status, delete) on top of `internal/k8s`.
- [X] T029 [US2] Port the `internal/worker` (with `prompt.py` induction and the spend watchdog) and
  `internal/grader` packages from `worker.py`, `prompt.py` and `grader.py`: the same manifests,
  deadline, caps and grade.json handling. Port `test_prompt.py`, `test_watchdog.py` and
  `test_grading_reference.py`.
- [X] T030 [US2] Port `internal/publisher` from `publisher.py`:
  - the task PR job (`_submit`) and the PR body, including `Closes #N` when issues are on;
  - `prs.json` with `kind` and `closes`;
  - idempotency; the status PR path is ported in US3.

  Port `test_publisher_idempotency.py`.
- [X] T031 [US2] Port `internal/checks` from `checks.py`: P01–P10 with the same descriptions,
  requirements and detail strings, and the gate matrix rows.
- [X] T032 [US2] Port `internal/approvals` from `approvals.py`: `has_tty` (`x/term`),
  `pending_step`, `describe`, `record` (channel harness or github), `take`/`take_record`,
  `decide` (a terminal answer discards a pending approval), `note`, `hint`, and `ask_tty` from
  `/dev/tty` (R13). Port `test_approvals.py` and `test_ask_tty.py`. Wire `pilot approve`.
- [X] T033 [US2] Port `internal/reconcile` from `reconcile.py`: `status_edit`, the `Reconciler`
  `check` and `run` methods, `for_fixture` (graded clone from the bundle), and the push path. Port
  `test_reconcile.py`.
- [X] T034 [US2] Port `internal/pipeline` from `pipeline.py`:
  - `mode_of`, `confirm_mode`, `confirm_retry`, `mode_not_confirmed`, `gate_task`, `_pause`,
    `_fail`, `run_task`, `publish_task`, `finish`, `run`, `resume`, with every printed message
    verbatim;
  - resume refuses runs without `impl: "go"` (R16).

  Wire `pilot run`, `pilot resume [--run]` and `pilot status [--run]`.
- [X] T035 [US2] Port the setup commands: `publisher-token [--check-only]` (hidden prompt through
  `x/term`, staging file, keep only on pass; 006 F3) using the Go write probe from T040, and
  `import-opencode` from `import_opencode.py` (port `test_import_opencode.py`). Until T040 lands,
  the probe calls a stub that refuses.
- [X] T036 [US2] Add the wording test `internal/cli/wording_test.go`: extract the backtick- and
  code-quoted operator messages from `docs/operator-guide.md`, `docs/troubleshooting.md` and
  `.claude/skills/pilot/SKILL.md`, and assert that each appears in the Go sources
  (contracts/cli.md).
  Done as: literal chunks of quoted spans that the Python reference contains (frozen in
  `internal/cli/testdata/wording.txt` for after US5); GitHub App messages are listed as pending
  until US3.
- [X] T037 [US2] **Owner action:** quickstart §3 with `PILOT_IMPL=go`, the terminal run following
  the guide (11/11, the same prompts and exit codes). Then the agent run with harness approvals
  (`channels=['harness']`). Copy the records to `specs/016-go-pilot/evidence/<run-id>/` and
  record the results in `specs/016-go-pilot/evidence/README.md`.
  Done as: two fixture runs, both with harness approvals (`20261115T162029Z` on the development
  machine, `20261116T021112Z` on the validation machine), each 11/11 with the documented
  commands and exit codes. Owner decision (2026-11-16): no separate terminal run; the `tty`
  prompt code is unchanged from US2 and covered by its tests.
- [X] T038 [US2] After T037 passes, flip the default in `pilot/pilot` to `PILOT_IMPL=go`.

**Checkpoint:** the Go pilot runs the fixture end to end. Python is still available through
`PILOT_IMPL=python`.

---

## Phase 6: User Story 3 — the GitHub loop works on the new pilot and is validated (P2)

**Goal:** 004's behaviour in Go (sync, drift, GitHub approvals, the status-change PR, the App), and
004 validated.

**Independent test:** 004 quickstart §2–§5 on the Go pilot.

- [X] T039 [P] [US3] Implement the GitHub client setup in `internal/ghloop/client.go` (go-github):
  - read token resolution `GH_TOKEN` → `GITHUB_TOKEN` → `gh auth token`, with a fail-closed
    hint;
  - the write client from `publisher.env` or the App token (`secret.Value`).
  Done as: `ReadToken`/`OperatorClient` (reads) and `NewWriter` (publisher credential from
  `publisher.env` or the App token file) in `internal/ghloop`; the CLI resolves the credential
  lazily (`credsFor`), so an App token is minted or refreshed when used.
- [X] T040 [US3] Port the write probe from `pilot/images/tools/check_token.py` into
  `internal/ghloop/probe.go`:
  - blob POST for contents;
  - pulls list;
  - label POST with the invalid colour, where 422 means issues write;
  - canary write must fail;
  - App kind skips `/user`.

  Port the `test_check_token.py` cases.
- [X] T041 [US3] Port `internal/ghloop` sync and drift from `ghsync.py`: `desired_state`,
  `checklist`, `issue_body` (marker `<!-- specfoundry:item=<id> -->`), `plan_ops`, `by_item`,
  `drift_report` (the R8 kinds), `sync`, `drift`, `issue_for`. The writes, ported from
  `ghops.py`: ensure labels, create, edit, comment, close; no merge. Port `test_ghsync.py` and
  `test_ghops.py`. Wire `pilot github sync|drift`.
  Done as: `internal/ghloop/sync.go` (pure functions and `Loop.Sync`/`Drift`) and `api.go`
  (`Reader`/`Writer` over go-github; ghops error wording kept). Issue bodies are byte-identical to
  the reference's, so switching implementation does not rewrite issues.
- [X] T042 [US3] Port the approvals part of `internal/ghloop` from `ghapprove.py`:
  `ensure_tracking`, `post_decision` (marker
  `<!-- specfoundry:decision run=<id> step=<s> task=<t> n=<n> -->`), `mark_answered`,
  `evaluate`, `find_approval` (`github/approvals.jsonl`, `approval_rejected` events), `collect`.
  Then wire them into `internal/pipeline` pauses and `resume`.
  Done as: `internal/ghloop/approve.go`; wired into `internal/pipeline/github.go` (post on
  pause, note a terminal answer, collect on resume without a terminal).
- [X] T043 [P] [US3] Write `internal/ghloop/vectors_test.go`: run `evaluate` on every case in
  `approvals.json`, and `reconcile.CheckStatusPR` on every case in `status_pr.json`
  (contracts/parity.md).
  Done as: `internal/ghloop/vectors_test.go` (15 approval cases, all status-PR cases).
- [X] T044 [US3] Port the status-change PR: `reconcile.CheckStatusPR` and the status bundle,
  `publisher.PublishStatus`, and the pipeline's `reconcile_github` and `status_pr_merged`
  (`pending.status_pr`, `reconcile/A-status.json`, a merger approval note). Port
  `test_status_pr.py`.
  Done as: `publisher.PublishStatus`, `pipeline.reconcileGitHub` and `statusPRMerged`. Change
  from the reference: a failure to fetch or read the merged status PR pauses the run with a
  message instead of raising.
- [X] T045 [US3] Port `internal/ghapp` from `ghapp.py` and `tools/app_token.py`:
  - the JWT through `crypto/rsa` RS256 (iat −60 s, exp +9 min, iss = App id);
  - the installation token request (`repositories: [<name>]`, permissions contents, pull
    requests and issues: write), with permission and repository checks;
  - the token cache with a 5-minute margin, `bot_login`, `discard`;
  - `store_key` (stdin to a staging file, kept only if the check passes) and `check`.

  Port `test_ghapp.py`, with a throwaway key generated in the test. Wire `pilot app-key` and
  `pilot app-token --check`. Refresh the publisher secret before each publish in App mode.
  Done as: `internal/ghapp` (key parsed and JWT signed in-process, R7). Change from the
  reference: in App mode `pilot up` probes the minted App token; the reference probed
  `publisher.env`, which App mode does not have.
- [X] T046 [US3] Add `internal/ghloop/nomerge_test.go`, which asserts that the module's source calls no
  go-github merge or review-approve method (R6).
  Done as: an AST scan of every non-test Go file for merge and review-approve calls and
  hand-built merge or review URLs (mutation-checked).
- [ ] T047 [US3] **Owner action:** run `specs/004-github-loop/quickstart.md` §2–§5 on the Go
  pilot, in token mode and App mode, with every refusal case (a second GitHub account is
  needed). Record it in `specs/004-github-loop/evidence/README.md` and copy the run records.
  This replaces 004's superseded T035–T037.

**Checkpoint:** 004 is validated on the Go pilot.

---

## Phase 7: User Story 4 — walk away during a run (P2)

**Goal:** the run starts a detached watcher; the operator closes the terminal and finishes on
GitHub.

**Independent test:** quickstart §5.

- [X] T048 [US4] Implement `internal/watcher` per contracts/watcher.md:
  - `Ensure(run)`: the pidfile `pilot/runs/<id>/github/watch.pid` with a liveness check; start
    `os.Executable()` `github watch --run <id>`, with Unix `SysProcAttr{Setsid:true}`, stdin
    from `/dev/null`, and output appended to `github/watch.log`;
  - a Windows build-tagged `DETACHED_PROCESS` variant;
  - `Stop(run)` (SIGTERM, remove the pidfile);
  - `Status(run)`.
  Done as: `internal/watcher` (`proc_unix.go` with `Setsid`, `proc_windows.go` compiled only).
  Liveness also checks `/proc/<pid>/cmdline` where available, so a reused pid doesn't count.
- [X] T049 [US4] Port the watch loop (`ready_to_resume` and `watch`, with no-progress backoff) into
  `internal/pipeline/watch.go`, as a goroutine with a context. Wire `pilot github watch --run`
  (foreground), `--detach` and `--stop`. On exit, remove the pidfile when the run is no longer
  paused.
  Done as: the loop stayed in `internal/pipeline/github.go` (`ReadyToResume`, `Watch`; ported in
  US3); `--detach`/`--stop` in `internal/cli/github.go`. Added (contracts/watcher.md "One
  resumer"): `resume` refuses while another process is the run's live watcher.
- [X] T050 [US4] Call `watcher.Ensure` from every pause when `PILOT_GITHUB_APPROVALS=on`: mode,
  retry, A's merge, and the status PR. Print "watcher started (pid N); you can close this
  terminal. Log: <path>". Add `watcher` to `pilot status` output.
  Done as: `ensureWatcher` at the decision post (mode, retry), A's merge pause and the
  status-change PR pauses; `pilot status` adds `watcher` while one runs (never saved in run.json).
- [X] T051 [P] [US4] Write `internal/watcher/watcher_test.go`:
  - `Ensure` is idempotent;
  - a stale pidfile is replaced;
  - the loop exits when the run succeeds or fails;
  - no watcher when approvals are off;
  - the spawn arguments use a new session.
  Done as: `internal/watcher/watcher_test.go` (idempotent, stale pidfile, new session, a real
  detached process and pid reuse) and `internal/pipeline/github_test.go` (approvals off, loop exit
  removes its pidfile, resume refused while a watcher runs).
- [ ] T052 [US4] **Owner action:** quickstart §5. Start a run, close the terminal, approve and merge
  on GitHub only, then check the run completes and no watcher remains. Record it in
  `specs/016-go-pilot/evidence/README.md`. Comment on issue #25 with the result.

---

## Phase 8: User Story 5 — retire the Python pilot (P3)

**Goal:** Go only. The operator-facing command and docs are unchanged.

**Independent test:** quickstart §6.

- [ ] T053 [US5] Remove `pilot/specfoundry_pilot/`, `pilot/tests/`, the Python-only host tools
  (`pilot/tools/app_token.py`, `pilot/tools/golden_audit.py` once its output is committed), and
  the `PILOT_IMPL` Python branch in `pilot/pilot`. Keep `pilot/images/**` scripts, because
  in-image Python stays.
  **Before removing:** move `pilot/specfoundry_pilot/frame.py` to `pilot/images/common/frame.py`, and
  update `.github/workflows/pilot-images.yml` (its trigger paths and the `cp` into each build context)
  and the Go `--local-build` context copy. Both the worker and tools images COPY `frame.py`.
  `pilot/images/build_args.py` stays, because the image workflow runs it.
- [ ] T054 [US5] Update `pilot/tools/docs_check.py` to drop the Python help parser, or port it
  into `go test` as `internal/cli/docs_test.go` and remove the Python script. Update
  `.github/workflows/go.yml` and `.github/workflows/docs-check.yml` to match.
- [ ] T055 [P] [US5] Update the docs to describe the Go pilot: `pilot/README.md` (layout, tests:
  `go test ./...`), `README.md` ("How it's built"), `CONTRIBUTING.md` (toolchain),
  `.claude/skills/pilot/SKILL.md` (paths), and `AGENTS.md`. Then run the docs check.
- [ ] T056 [US5] Confirm that nothing outside history and evidence refers to the removed code
  (`grep -rn "specfoundry_pilot\|python3 -m specfoundry_pilot" --exclude-dir=evidence`).
  Update the harness ask rules in `.claude/settings.json` (drop the `python3 -m
  specfoundry_pilot approve` pattern).
- [ ] T057 [US5] **Owner action:** one more guide walkthrough with the audit at 11/11 after
  retirement. Record it in `specs/016-go-pilot/evidence/README.md`.

- [X] T060 [US2] Verbose levels (FR-018, owner request 2026-11-15): global `-v` count flag and
  `PILOT_VERBOSE` in `internal/cli/root.go`; `internal/verbose` (levels, redaction of secret
  values and model hosts, OpenCode event formatting); the runner's pod watcher
  (`internal/runner/watch.go`: pod phases, events, model-proxy refusals at `-vv`; `tail -F
  /tmp/harness.jsonl` by exec in worker pods at `-vvv`); step and grade timings in the
  pipeline; host commands at `-v`. Docs: operator guide §12, troubleshooting, pilot skill.

---

## Phase 9: Polish and closeout

- [ ] T058 [P] Run a privacy check: AU08 passes for every 016 and 004 run record, and no token,
  key or model host appears in `specs/016-go-pilot/` or `specs/004-github-loop/`.
- [ ] T059 Add Evidence sections to `specs/016-go-pilot/spec.md` and
  `specs/004-github-loop/spec.md`. Run `/speckit-roadmap-debrief` for 016 and for 004. Then,
  with the owner's approval, `/speckit-roadmap-write`: the 016 and 004 statuses and spec dirs,
  and 013 unblocked.

---

## Dependencies & Execution Order

- **Setup (T001–T006)** → **Foundational (T007–T013)** → everything else.
- **US1 (T014–T020)** is the MVP and needs only the foundation. T014 (golden generation) must run
  while the Python reference exists.
- **Shared vectors (T021–T022)** can run in parallel with US1. They block T043 (US3).
- **US2 (T023–T038)** needs the foundation, and uses `audit` and `gate` from US1 at run end.
  T035's probe depends on T040; use a refusing stub until then, or do T040 early.
- **US3 (T039–T047)** needs US2's pipeline and publisher, plus the vectors.
- **US4 (T048–T052)** needs US3 (GitHub approvals detect readiness).
- **US5 (T053–T057)** needs US1–US4 to be proven, and T038 (Go is the default).
- **Polish (T058–T059)** comes last.

### Parallel opportunities

- T003, T004 and T005 within Setup; T008, T009 and T012 within Foundational.
- In US1, T015, T017, T019 and T020 can be written alongside T016/T018.
- In US2, T023–T026 are independent packages (k8s, frame, fixture, images). T027–T034 follow in
  order.
- In US3, T039 and T043 can be done alongside T040–T042.
- The docs tasks (T055) can be done alongside code tasks in their phase.

## Implementation Strategy

### MVP

Setup, Foundational and US1: a Go binary whose audit and gate match the reference on all golden
data. It's useful immediately as an independent audit of the committed evidence.

### Then

1. The vectors.
2. US2: the full fixture run; flip the default to Go.
3. US3: the GitHub loop and 004's validation.
4. US4: walk away.
5. US5: retire Python.
6. Closeout.

### Owner actions (not automatable)

- **T001:** Go on the machines.
- **T037:** the fixture runs.
- **T047:** 004's validation (a second GitHub account, the App).
- **T052:** walking away.
- **T057:** the final walkthrough.
- **T004:** the repository settings for Dependabot security updates (quickstart §0b).

The agent never approves, posts `/approve`, or merges.

## Notes

- Port messages, event names, check ids and detail strings verbatim. The wording test (T036)
  and the golden test (T015) are the guards.
- Never log or persist a `secret.Value`. Evidence must pass AU08.
- Commit after each task group. Work on branch `016-go-pilot`, open PRs per phase, and never
  push to `main`.
