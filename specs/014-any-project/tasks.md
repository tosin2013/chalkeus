---
description: "Task list for 014 Any project, any item"
---

# Tasks: Any project, any item

**Input**: Design documents from `specs/014-any-project/`
(plan.md, spec.md, research.md R1–R18, data-model.md, contracts/, quickstart.md)

**Tests**: included. The plan's Testing section requires unit tests for each new rule, the
golden audit staying green, and owner validation runs (quickstart). Tests are written before or
with each implementation task, in the same package.

**Organization**: by user story.
- US2 (readiness, P1) comes before US1 (run an item, P1), because an item run starts with
  readiness. US2 also needs no cluster, so it is the fastest slice to prove.
- US3 (toolchains and egress, P2) builds on US1.
- US4 (the fixture keeps working, P2) is checked throughout and closed at the end.

**Owner actions** are marked **Owner**. The agent never approves, posts `/approve`, merges, or
creates repositories on the owner's behalf.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel (different files, no dependency on an unfinished task).
- **[Story]**: US1–US4 from spec.md.

---

## Phase 1: Setup

- [X] T001 Create the toolchain catalog `pilot/images/toolchains.json`, with entries `python`
  (no image) and `jdk17-maven` (image key `TOOLCHAIN_JDK17_MAVEN_IMAGE`, env, worker, grader
  and cache defaults), exactly as in data-model.md "Toolchain (catalog entry)".
  Done as: then updated to the owner's catalog (2026-11-17): `jdk21-maven`, `node24`,
  `python3.13`, `go`, plus `python` (built in). Environment moved into each image's own
  `toolchain.json`, and operator-supplied images were added (`PILOT_TOOLCHAIN_IMAGE`).
- [X] T002 [P] Add the issue form `.github/ISSUE_TEMPLATE/toolchain-image.yml` (FR-012).
  - Fields: language and version, build tool and version, registry hosts, a link to the
    upstream release with its checksums, and the requester's target platform (kind, minikube,
    plain Kubernetes, OpenShift).
- [X] T003 [P] Create `specs/014-any-project/evidence/README.md`. It has a table: run, machine,
  target, item, approvals, result, notes.

---

## Phase 2: Foundational (blocks every story)

**Purpose:** target profiles, run kinds and audit dispatch. Once this phase is done, the
fixture is still unchanged and the golden audit still passes.

- [X] T004 Add a fixture regression guard in `internal/pipeline/fixture_parity_test.go`. This is
  US4's guard; write it before any other code change. It checks that the rendered fixture
  manifests, mode text, prompt, `meta.json` and summary are byte-identical
  to golden copies captured before 014's first change, in `internal/pipeline/testdata/fixture/`.
- [X] T005 Write tests for target profiles in `internal/config/targets_test.go`. Cover:
  - name rule `^[a-z0-9][a-z0-9-]{0,30}$`, with `fixture` reserved;
  - `targets/` must be 0700 and each `<name>.env` 0600 (`TARGET_MODE`);
  - required keys `PILOT_TARGET_REPO` (owner/name), `PILOT_TOOLCHAIN`, `PILOT_TEST_COMMAND`
    (non-empty) and `PILOT_COMPANION_REPO` (owner/name);
  - `PILOT_REGISTRY_HOSTS` entries match `^[A-Za-z0-9.-]+$`;
  - `PILOT_TARGET_BRANCH` defaults to `main`;
  - `PILOT_GITHUB_ISSUES` defaults to `off`;
  - overrides may only raise the catalog defaults;
  - `TARGET_UNKNOWN`, `TARGET_INVALID` (names the key) and `TARGET_IS_CANARY` (this checkout's
    origin, or another target's companion repository);
  - the implicit `fixture` target built from `pilot.env` alone is identical to today's
    `Config`.
- [X] T006 Implement `internal/config/targets.go` per contracts/target-profile.md:
  - `LoadTarget(dir, name)`, `ListTargets(dir)`, `Target` struct;
  - `Config.Target()` with the fixture default;
  - the publisher env path is `targets/<name>.publisher.env` in token mode;
  - toolchain validation against `pilot/images/toolchains.json` and `pilot/images.lock`
    (`TOOLCHAIN_UNAVAILABLE`, whose message links `.github/ISSUE_TEMPLATE/toolchain-image.yml`).
- [X] T007 [P] Add the `targets` command in `internal/cli/targets.go`, wired in
  `internal/cli/commands.go`. It lists name, repository, branch, toolchain, registries and
  companion, and never prints a secret or the model host.
- [X] T008 Add the global `--target <name>` flag in `internal/cli/root.go`, and thread the
  selected `Target` into `run`, `gate`, `github …`, `app-token --check` and `reset-fixture`.
  - `resume`, `approve`, `status` and `audit` read the target from `run.json`, never from the
    flag (FR-004a).
  - With no flag, nothing changes.
  Done as: `--target` is a per-command flag (not global), on `gate` and `reset-fixture` now;
  `run`, `github …` and `app-token --check` gain it with the item run (US1: T027, T031). `resume`,
  `approve`, `status` and `audit` never take it: they read the target from `run.json`.
- [X] T009 Make `reset-fixture` refuse any target but `fixture`: exit 2, `NOT_A_FIXTURE`
  (R17), in `internal/cli/fixture.go`, with a test in `internal/cli/fixture_test.go`.
- [X] T010 Record the run kind in `internal/evidence/evidence.go`:
  - `run.json` gains `kind` (`fixture` | `item`; absent reads as `fixture`), `target` (`{name,
    repo, branch, toolchain}`) and `item`;
  - `WriteSummary` lists the tasks found in `run.json` instead of the fixed `A`,`B`;
  - the item-run summary title is "pilot item run".

  Tests in `internal/evidence/evidence_test.go`: a pre-014 record reads as `fixture`, and the
  fixture summary is byte-identical.
- [X] T011 Add `kind` dispatch in `internal/audit/audit.go`: `kind=item` selects variant
  functions (stubbed to fail until US1); fixture and missing `kind` use today's checks.
  `internal/audit/golden_test.go` must stay green unchanged.
- [X] T012 [P] Make `internal/images/images.go` read the image list and lock keys from
  `pilot/images/toolchains.json` plus the fixed `worker`, `tools`, `egress`.
  - Add `ToolchainImage(name)`, which returns the digest, or `TOOLCHAIN_UNAVAILABLE` when none
    is locked.
  - Tests in `internal/images/images_test.go`.

**Checkpoint:** `go test ./...` is green, the golden audit is unchanged, and `pilot/pilot
targets` lists `fixture`.

---

## Phase 3: User Story 2 — refuse an item that isn't prepared (P1)

**Goal:** `pilot gate --target <name> --item NNN` refuses every unprepared item with a reason
code, without a cluster.

**Independent test:** for each rule in contracts/readiness.md, an item that breaks only that
rule refuses with its code. A prepared item gives `allow`.

- [X] T013 [P] [US2] Write tests for the allowed-paths parser in
  `internal/readiness/paths_test.go`, per contracts/item-spec.md:
  - an `## Allowed paths` level-2 heading, then list items whose first backticked span is a
    glob;
  - the item's `<spec dir>/tasks.md` is always added;
  - `ITEM_NO_ALLOWED_PATHS` when the section or its globs are missing;
  - `ITEM_PATHS_TOO_WIDE` for a glob that matches `.github/…`, `.specify/…`,
    `.specify/memory/roadmap.md` or another item's spec folder.

  Matching uses fnmatch, where `*` matches `/`.
- [X] T014 [P] [US2] Implement `internal/readiness/paths.go` (`AllowedPaths(specText,
  specDir, otherSpecDirs) ([]string, error)`).
- [X] T015 [US2] Write tests for the readiness rules in `internal/readiness/readiness_test.go`,
  using a temp target repository fixture and synthetic `spec status` JSON. Each code in
  contracts/readiness.md order 0–7 is produced by breaking only its rule:
  - existing gate codes;
  - `ITEM_NOT_ON_ROADMAP`;
  - `ITEM_SPEC_MISSING`, `ITEM_TASKS_MISSING`;
  - `ITEM_OPEN_QUESTIONS`: a `[NEEDS CLARIFICATION` marker, an error diagnostic, or an
    unstruck roadmap open question naming NNN;
  - the paths codes;
  - `ITEM_NO_HIDDEN_TESTS`, including when the companion repository can't be read;
  - `INPUT_TOO_LARGE`;
  - `OPEN_ITEM_PR`.

  Also test that the first failing rule wins, and that unreadable status refuses.
- [X] T016 [US2] Implement `internal/readiness/readiness.go`. `Check(target, item) (Result,
  []Row)`:
  - fresh clone at the target branch, and `gate.EvaluateFresh` (startable: `planned`,
    `specced`, `in-progress`; dependencies verified);
  - spec-dir resolution from the status JSON;
  - spec and tasks presence; the open-question checks; `AllowedPaths`;
  - a shallow clone of the companion repository on the host, with the operator's git helper
    (as `fixture.FreshClone`): `items/NNN/` must be non-empty;
  - the snapshot size against `PILOT_MAX_INPUT_MB` (default 50);
  - open `pilot/*/<NNN>` PRs.

  Returns rows for `gate/<seq>-<NNN>.json`.
- [X] T017 [US2] Add `pilot gate --target <name> --item NNN` in `internal/cli/gate.go`:
  readiness only, exit 10 with the reason code on refuse, exit 0 on allow. `gate {A|B}` is
  unchanged. CLI tests in `internal/cli/gate_test.go`.

**Checkpoint:** every refusal code is reachable from the CLI on a local test repository. No
cluster is used.

---

## Phase 4: User Story 1 — run a chosen item (P1), on the `python` toolchain

**Goal:** `pilot run --target <name> --item NNN` runs one prepared item end to end:
mode approval → baseline → attempts → draft PR → human merge → status change `verified` →
audit. It runs on the existing toolchain first (plan slice 3).

**Independent test:** a small Python sample target, with a companion repository and one item,
completes and the audit passes 11/11 (T036, Owner).

**Inputs into pods (R4)**

- [ ] T018 [US1] Add `ExecStream(ns, pod, container, tar io.Reader, dir)` and
  `CreatePVC/DeletePVC` in `internal/k8s/k8s.go`, with fake-clientset tests in
  `internal/k8s/k8s_test.go`. The exec stream is a tar unpacked by the receiver; secrets are
  never logged.
- [X] T019 [US1] Write snapshot and replay tests in `internal/fixture/snapshot_test.go`, using
  real `git` in a temp dir:
  - `Snapshot(clone, base)` builds a one-commit bundle whose tree equals `base`'s tree;
  - `Replay(clone, base, snapshot, changeBundle)` rebases the change onto `base`, and the
    replayed tree is identical to the graded tree;
  - a conflict-free replay is required, and a mismatch fails with `REPLAY_MISMATCH`.
- [X] T020 [US1] Implement `internal/fixture/snapshot.go` (`Snapshot`, `Replay`; uses the
  existing git helpers).
- [ ] T021 [US1] Add the input receiver for item runs to `pilot/manifests/worker-job.yaml.tmpl`
  and `pilot/manifests/grader-job.yaml.tmpl`, behind a template switch so fixture rendering is
  byte-identical:
  - an init container (tools image) waits for `/in/.ready` on an emptyDir;
  - `meta.json`, `prompt.md` and `settings.xml` stay in a ConfigMap.

  Golden render tests in `internal/worker/worker_test.go` and `internal/grader/grader_test.go`
  cover both kinds.

**Prompt, worker, grading**

- [X] T022 [P] [US1] Add `prompt.BuildItem(spec, plan, tasks, tasksPath, allowedPaths,
  toolchain, testCommand)` in `internal/prompt/prompt.go`, per R10.
  - It has a generic preamble, no "Python standard library" line, and no induction.
  - `Build` (the fixture) stays unchanged.
  - Tests in `internal/prompt/prompt_test.go`, including the fixture's byte-identical output.
- [ ] T023 [US1] Add `worker.MakeItemInputs` in `internal/worker/worker.go`:
  - the snapshot bundle via `ExecStream` instead of the inputs ConfigMap;
  - `meta.json` gains `test_command`, `hidden_mode:"overlay"`, `spec_sha256`, `toolchain`,
    `mode:"attempt"`, and `allowed_paths` from readiness;
  - label = item id (namespaces `pilot-w-<nnn>-<n>`);
  - `MakeInputs` (fixture) unchanged.
- [X] T024 [P] [US1] Extend `pilot/images/tools/grade.py` per contracts/grading.md:
  - optional `meta.mode` (`attempt` | `baseline`), `test_command` (run via `sh -c` from the
    project root), `hidden_mode:"overlay"` (copy `/grading/overlay/` over the project, then
    rerun `test_command`), and `command_timeout_s` (default 180);
  - baseline: G1 applies the base only, G2 and G3 are skipped, G4 must pass, G5 must fail;
  - with none of the new fields, behaviour and output are unchanged.

  Tests in `pilot/images/tests/test_grade_modes.py`, using a tiny Python sample project:
  baseline pass, baseline with hidden passing (fail), overlay fail, overlay pass, and a fixture
  meta with unchanged output.
- [ ] T025 [US1] Add a run-scoped grading namespace in `internal/grader/grader.go` and the new
  `pilot/manifests/grading-ns.yaml.tmpl`:
  - `pilot-g-<nnn>` per run, with quota, LimitRange and a default-deny NetworkPolicy;
  - `role=grader` pods have no egress rule;
  - baseline and attempt grader Jobs run inside it;
  - the overlay tar is staged via `ExecStream` (from the host companion clone);
  - the namespace is deleted at run end.

  The fixture keeps `pilot-g-<t>-<n>`. Tests cover rendering and the namespace lifecycle with
  the fake clientset.
- [X] T026 [US1] Write `companion.json` evidence in `internal/grader/companion.go`: repository,
  commit, file count and file list with SHA-256s, never contents (R9). Test that no file
  content reaches the evidence dir.

**Pipeline**

- [ ] T027 [US1] Implement the item-run state machine in `internal/pipeline/item.go`, per the
  data-model.md state diagram:
  1. readiness, where a refusal exits 10;
  2. the mode approval, which shows target, item and toolchain, and egress `model host(s) <n>`
     plus registries (contracts/cli.md text). The GitHub post leaves out the model host count;
  3. the baseline: a pass continues, otherwise `failed: BASELINE_SUITE_FAILS` or
     `BASELINE_HIDDEN_PASSES`, exit 1;
  4. attempts, with retries approved via the existing `confirmRetry`;
  5. on a pass, `Replay` onto the real base, then publish;
  6. `awaiting-merge` → status change → `verified` → finish.

  Fixture `Run` and `Resume` dispatch on `kind` and are otherwise untouched. Resume reads the
  target from `run.json`.
- [ ] T028 [US1] Generalize the A-specific GitHub and resume paths in
  `internal/pipeline/github.go` (`statusPending`, `reconcileGitHub`, `statusPRMerged`,
  `ReadyToResume`) and in `internal/decisions/decisions.go` (`PendingStep`, `Describe`,
  `reconcile/<label>.json`).
  - They take the run's task label instead of `"A"`.
  - For item runs, nothing comes after `verified` except finish. The fixture still continues to
    B.
  - Tests in `internal/pipeline/github_test.go` and `internal/decisions/decisions_test.go` for
    both kinds.
- [X] T029 [US1] Generalize the status edit in `internal/reconcile/reconcile.go` (R12):
  - `StatusEdit` accepts `planned|specced|in-progress` → `verified` for exactly one line;
  - `rec["proposed"]` names the old status;
  - the commit message and `publisher.StatusBody` name the item and the old status;
  - the fixture output is unchanged.

  Tests in `internal/reconcile/reconcile_test.go` for the three source statuses plus the
  existing vectors.
- [X] T030 [P] [US1] Item wording in `internal/publisher/publisher.go`: the PR body names the
  target and item, and the evidence path comes from the run, not the hard-coded 002 path. The
  PR head is `pilot/<run>/<nnn>`. The fixture body is unchanged (test).
- [X] T031 [P] [US1] Map GitHub issues by item id in `internal/ghloop/sync.go` (`IssueFor`,
  `DriftReport`) instead of `paths.Tasks`, keeping the fixture mapping. Tests in
  `internal/ghloop/sync_test.go`.

**Checks and audit**

- [ ] T032 [US1] Add the P11 check in `internal/checks/checks.go`: no overlay file name or
  SHA-256 from `companion.json` appears in the worker namespace's ConfigMaps, staged-input
  list or `prompt.md`. Tests in `internal/checks/checks_test.go`.
- [X] T033 [US1] Implement the item-run audit variants in `internal/audit/audit.go`, per
  contracts/audit-item.md:
  - AU01: only NNN allowed, readiness rows pass, `other_attempts=0`;
  - AU02: the baseline passed before the first attempt, then `fail*` and exactly one `pass`,
    which is last;
  - AU03: exactly 1 task PR and at most 1 status PR, drafts by the publisher identity, on the
    target repository;
  - AU04: also requires P11 (and P12 in US3);
  - AU11: no attempt induced, and each prompt's `spec_sha256` equals the SHA-256 of `spec.md`
    at the base.

  Tests with synthetic item records in `internal/audit/audit_item_test.go` (pass and each
  failure).
- [ ] T034 [US1] Add `-v` timings and `-vv` pod and event lines for the readiness, baseline and
  grading-namespace steps, in `internal/pipeline/item.go` and `internal/runner` (FR-018 rules:
  stderr only, redacted).
- [ ] T035 [US1] Write the operator docs:
  - new `docs/targets.md`: preparing a target (Spec Kit, `## Allowed paths`, the companion
    `items/NNN/` layout, hidden tests that fail on the base and use only declared
    dependencies), the profile keys, `pilot targets`, `gate --item`, `run --target --item`,
    and the refusal codes;
  - links from `docs/operator-guide.md`;
  - troubleshooting entries for each new code in `docs/troubleshooting.md`.

  `go test ./internal/cli -run TestDocsCheck` must pass.
- [ ] T036 [US1] **Owner action:** a Python sample target, run end to end on kind.
  1. Create a small public Python repository with a test suite.
  2. Create its private companion repository with `items/001/` (failing tests on the base).
  3. Add `targets/pysample.env`.
  4. Run `pilot/pilot run --target pysample --item 001` → approve → merge → status PR → `down`
     → audit 11/11.
  5. Record it in `specs/014-any-project/evidence/README.md`.

**Checkpoint:** US1 and US2 work on the existing toolchain; the fixture is unchanged.

---

## Phase 5: User Story 3 — toolchain and registries (P2)

**Goal:** any language works. A Java/Maven target builds with a catalog toolchain and Maven
Central only; another language builds with an operator-supplied toolchain image. The grader
works offline from a cache filled from human inputs (R5, R6, R7).

**Independent test:** `greeting-service` item 001 runs end to end. An undeclared host is
refused and recorded. The audit passes 11/11, including P06r and P12 (T053, Owner).

**Toolchain image (R5)**

- [ ] T037 [US3] Add pins to `pilot/pins.env` for the catalog toolchains (owner decision
  2026-11-17), each with linux amd64 and arm64 checksums from the upstream release:
  - Eclipse Temurin 21 (SHA-256);
  - Apache Maven 3.9.x (SHA-512 from the Apache `.sha512`);
  - the Node.js 24 LTS tarball (SHA-256 from `SHASUMS256.txt`);
  - `python-build-standalone` 3.13 (SHA-256);
  - Go (SHA-256 from go.dev).
- [ ] T038 [US3] Create one image per catalog toolchain: `pilot/images/toolchain-jdk21-maven/`,
  `toolchain-node24/`, `toolchain-python3.13/` and `toolchain-go/`. Each one:
  - has a fetch stage that verifies every tarball against the pins;
  - has a final stage that holds only `/opt/toolchain/` with its `toolchain.json`
    (contracts/toolchain-image.md: `name`, `version_command`, `path`, `env`, `proxy`), files group
    0 and `g=u`, no setuid, and a static copy helper;
  - gets a branch in `pilot/images/build_args.py`.
- [ ] T039 [US3] Make `.github/workflows/pilot-images.yml` data-driven from
  `pilot/images/toolchains.json` plus `worker`, `tools`, `egress`. That covers the context
  loop, the push loop, `image-digests.json` and the `digest-pr` key list, including the four
  `TOOLCHAIN_*_IMAGE` keys in `pilot/images.lock`.
- [ ] T040 [US3] Extend `pilot/images/smoke.sh`.
  - Every catalog toolchain image passes the contract checks under an arbitrary UID.
  - The tools image grades one tiny offline sample per toolchain, under
    `pilot/images/tests/samples/{maven,npm,pytest,go}/`, with its dependencies pre-seeded,
    through `test_command` and an overlay.
- [ ] T041 [US3] Write the toolchain contract probe in `internal/toolchain/probe.go`, and the new
  `pilot/manifests/toolchain-probe-job.yaml.tmpl`.
  - After the mode approval, a no-network Job in the run's grading namespace copies the
    toolchain under an arbitrary UID with group 0 and checks: `toolchain.json` parses and
    follows the rules (paths under `/opt/toolchain`, no proxy or secret-like env names, `proxy`
    is `env` or `maven-settings`), the files are readable, `version_command` exits 0 within
    60 s, and nothing is setuid or setgid.
  - A failure refuses with `TOOLCHAIN_CONTRACT`, naming the check.
  - Evidence goes to `toolchain.json`: source (`catalog` | `operator-supplied`), image digest,
    name, version line, attestation.
  - The `toolchain.json` parsing and rule checks are unit-tested in
    `internal/toolchain/probe_test.go`.
- [ ] T042 [US3] Handle operator-supplied toolchain images (FR-011a, FR-011b).
  - `PILOT_TOOLCHAIN_IMAGE` in the profile (already parsed and checked for a digest) is used
    for the toolchain init container only. The worker and tools images are never replaceable.
  - The mode approval adds `toolchain <label> (operator-supplied <image>)`.
  - When `gh` is available, run `gh attestation verify oci://<image> --owner <registry owner>`
    and record the result:
    - **verified:** the run continues;
    - **absent:** recorded, and the run continues;
    - **present but failing:** the run refuses with `TOOLCHAIN_ATTESTATION_FAILED`.
  - Tests in `internal/toolchain/attest_test.go`, with a fake `gh`.
- [ ] T043 [P] [US3] Add the own-image template in `pilot/images/toolchain-template/`:
  - `Dockerfile.ubi`, from a Red Hat UBI language image (for example `ubi9/nodejs-22`,
    `ubi9/openjdk-21`), copying the runtime into `/opt/toolchain`;
  - `Dockerfile.debian`, from any base;
  - `toolchain.json.example`;
  - `toolchain-image.yml`, a GitHub Actions workflow that builds, pushes, attests
    (`actions/attest-build-provenance`) and prints the digest;
  - `README.md` with the steps.

  `pilot/images/smoke.sh` builds the Debian example once and runs the contract probe on it.

**Toolchain wiring**

- [ ] T044 [US3] Add the toolchain init container to item-run worker, prefetch and grader pods,
  in `pilot/manifests/worker-job.yaml.tmpl` and `pilot/manifests/grader-job.yaml.tmpl`. It
  copies `/opt/toolchain` into an emptyDir, and the catalog sets `JAVA_HOME`, `MAVEN_HOME` and
  the PATH prefix. Item-run worker volumes come from the catalog (`/home/worker` 1Gi, `/tmp`
  512Mi, `MAVEN_OPTS=-Xmx768m`). Golden render tests.

**Registry egress (R6)**

- [ ] T045 [US3] Add the per-target registry proxy in `internal/cluster/infra.go`:
  - `EnsureRegistryProxy(target)` renders `pilot/manifests/egress.yaml.tmpl` as
    `pilot-egress-r-<target>`, with an allowlist of exactly `PILOT_REGISTRY_HOSTS` (never the
    model host). It is re-rendered when the hosts change;
  - the worker NetworkPolicy in `pilot/manifests/worker-ns.yaml.tmpl` gains a rule for that
    proxy (item runs only);
  - `down` deletes all `pilot-egress-r-*` and writes teardown row T04.

  Tests in `internal/cluster/cluster_test.go`.
- [ ] T046 [US3] Generate `settings.xml` with a `<proxy>` for `pilot-egress-r-<target>` and
  pass `MAVEN_ARGS=-s /in/settings.xml`, in `internal/worker/worker.go` and
  `internal/grader/grader.go`, for `proxy: maven-settings` toolchains. OpenCode keeps
  `HTTPS_PROXY` (the model proxy). Tests.
- [ ] T047 [US3] Add the P06r check in `internal/checks/checks.go`: a declared registry is
  reachable through the registry proxy from a probe pod in the worker policy, and
  `example.invalid` is refused. Copy the registry proxy's refusals into the attempt's
  `proxy-denials.log` (extending `worker.ProxyDenials`). Tests.

**Dependency cache (R7)**

- [ ] T048 [US3] Add the cache PVC and prefetch:
  - in `pilot/manifests/grading-ns.yaml.tmpl`, a PVC `deps` (size from the catalog or
    `PILOT_CACHE_SIZE`, default StorageClass), and a NetworkPolicy letting `role=prefetch`
    pods reach only DNS and `pilot-egress-r-<target>`;
  - a new `pilot/manifests/prefetch-job.yaml.tmpl`;
  - `grader.Prefetch` in `internal/grader/grader.go`: once per run, after the mode approval,
    on the base snapshot plus the overlay (**never agent code**), running `test_command` with
    `-Dmaven.repo.local=/deps`. Its exit code is recorded, not judged. A failure to resolve is
    `PREFETCH_FAILED`;
  - `prefetch.json` evidence (result, duration, bytes).
- [ ] T049 [US3] Mount the cache in grader Jobs: `/deps` read-only, Maven with `-o -s
  /in/settings.xml -Dmaven.repo.local=/tmp/m2 -Dmaven.repo.local.tail=/deps`. Grader resources
  and deadline come from the catalog (2 CPU, 2Gi, 900 s; 600 s per command) or the profile.
  The namespace and PVC are deleted at run end; `down` deletes leftovers and writes teardown
  row T05.
- [ ] T050 [US3] Add the P12 check in `internal/checks/checks.go`: grader pods in
  `pilot-g-<nnn>` have no egress rule, and the prefetch pod's rule allows only DNS and the
  registry proxy. Extend P07 and P08 to the run-scoped namespace. Make AU04 require P12 and
  AU07 require T04 and T05 for item runs, in `internal/audit/audit.go`. Tests.
- [ ] T051 [US3] Update `docs/targets.md` and `docs/troubleshooting.md`:
  - toolchains and the request form;
  - registry hosts at the mode approval;
  - an attempt can't add dependencies (the human adds them while preparing the item);
  - entries for `TOOLCHAIN_UNAVAILABLE`, `PREFETCH_FAILED` and `BASELINE_*`.

**Validation (Owner)**

- [ ] T052 [US3] **Owner action:** prepare the validation repository (quickstart §1).
  - Copy `spring-guides/gs-rest-service` `complete/` (Apache-2.0) into
    `<owner>/greeting-service`. Remove Gradle; keep the license and notice.
  - Spec Kit roadmap with items 001 (validate `name`), 002 (greeting templates) and 003
    (greetings by language, depends on 002), each with `## Allowed paths`.
  - Private `<owner>/greeting-service-acceptance` with `items/00N/` hidden JUnit tests that
    fail on the base.
  - App installed on the target if App mode; `targets/greeting.env`.
- [ ] T053 [US3] **Owner action:** quickstart §3–§4.
  1. Readiness refusals on a scratch branch: each code once.
  2. Run `greeting-service` item 001 end to end: mode approval shows `repo.maven.apache.org` →
     prefetch → baseline pass → attempts → draft PR → merge → status PR → `down` → audit 11/11.
  3. Check by hand that no hidden-test name is in prompts or the PR.

  Copy the records to `specs/014-any-project/evidence/`.
- [ ] T054 [US3] **Owner action:** quickstart §5. Run item 002, then 003 (unblocked by 002).
  Then run the fixture with no configuration edit between runs (SC-007). Record them.

- [ ] T055 [US3] **Owner action (SC-008):** a language specfoundry has no image for (for example
  Rust or .NET).
  1. Build a toolchain image from `pilot/images/toolchain-template/` (UBI or another base) in
     your own repository.
  2. Set `PILOT_TOOLCHAIN_IMAGE` to its digest.
  3. Run one prepared item end to end: the mode approval shows the image as operator-supplied,
     and the audit is 11/11.
  4. Record it in `specs/014-any-project/evidence/README.md`.

**Checkpoint:** a Java/Maven project, and a project in a language specfoundry has no image for,
run under every guarantee on kind.

---

## Phase 6: User Story 4 — the fixture keeps working (P2)

**Goal:** the fixture run and the committed evidence are unchanged (FR-015, SC-005).

**Independent test:** one fixture run, 11/11; the golden audit is unchanged.

- [ ] T056 [US4] **Owner action:** a fixture run on the final branch, following the operator
  guide: `up` → `run` → … → `down` → audit 11/11. Record it in
  `specs/014-any-project/evidence/README.md`.

---

## Phase 7: Polish and close-out

- [ ] T057 [P] Do a privacy check: AU08 passes for every 014 record, and no model host, token,
  key or hidden-test content appears in `specs/014-any-project/`.
- [ ] T058 [P] Update `README.md` ("What works today"), `pilot/README.md` (layout: toolchains,
  manifests) and `.claude/skills/pilot/SKILL.md` (`--target`/`--item`; refusal codes go to
  troubleshooting). Run the docs check.
- [ ] T059 Add an Evidence section to `specs/014-any-project/spec.md`. Run
  `/speckit-roadmap-debrief` for 014. Then, with the owner's approval, run
  `/speckit-roadmap-write`:
  - 014 status and spec dir;
  - a note on 011 (its cluster resource takes the target profile fields, FR-004b);
  - 017 unblocked.

---

## Dependencies & execution order

- **Setup (T001–T003)** → **Foundational (T004–T012)** blocks every story.
- **US2 (T013–T017)** needs Foundational and no cluster. **US1 (T018–T036)** needs US2's
  readiness (T016).
- **US3 (T037–T055)** needs US1's pipeline, grading namespace and overlay grading.
  - T037–T043 (the catalog images, the probe's unit tests and the own-image template) can start
    in parallel with US1, because they touch different files.
  - T052 (the owner preparing the repository) can start any time.
- **US4:** its guard (T004) is the first Foundational task, written before any code change;
  the fixture run (T056) is last.
- **Polish (T057–T059)** comes last.

```text
Setup → Foundational → US2 → US1 → US3 → US4(T056) → Polish
                       ↘ T037–T043 (images) ────┘
```

## Parallel opportunities

- **Foundational:** T007 and T012 alongside T005–T006.
- **US2:** T013 and T014 alongside T015.
- **US1:** T022, T024, T030 and T031 are independent files; T018 and T019 can be written
  together.
- **US3:** the image track (T037–T043) runs alongside the wiring track (T044–T050) until T053.
- **Polish:** T057 and T058.

## Implementation strategy

1. **MVP = Foundational + US2.** Readiness refusals for real repositories, with no cluster.
   This is useful on its own to check that a project is prepared.
2. **Then US1 on the Python toolchain**, proven by T036, before any Java work. That isolates the
   general path from toolchain issues.
3. **Then US3 for Java/Maven**, validated on `greeting-service` (T053–T054).
4. **US4's guard (T004)** is in place from early on, and T056 confirms the fixture at the end.
5. **One PR per phase or slice; the owner merges.**
