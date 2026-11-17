# Research: Any project, any item (014)

Owner decisions this research follows, from the spec's Clarifications (2026-11-17):
- target settings exist only on the operator side, as per-target profiles (cluster resource in
  011);
- the status change proposes `verified`;
- allowed paths are declared in the item's spec;
- hidden tests live in a private companion repository and run as an overlay with the
  project's test command;
- the kind validation uses a small Java/Maven repository.

Code references come from a read of `internal/**` and `pilot/**` on main `59e5a1f`.

## R1 — Run kinds: fixture run and item run

- **Decision:** a run has a `kind`, either `fixture` (today's two-item A→B run) or `item` (one
  prepared item of a target). `run.json` gains `kind`, `target` (profile name and repository)
  and `item`.
  - In an item run, the task label is the item id (for example `003`). Namespaces, PR heads,
    `attempts/<label>-<n>` and `reconcile/<label>.json` follow from the label as today.
  - Records written before 014 have no `kind` and are read as `fixture`.
- **Rationale:** the pipeline hard-codes A→B in about 30 places:
  - `pipeline.go` `Run`, `reconcileA`;
  - `github.go` `reconcileGitHub`, `statusPRMerged`, `ReadyToResume`;
  - `decisions.PendingStep`;
  - `evidence.WriteSummary`.

  A `kind` lets the fixture path stay byte-for-byte as it is (FR-015, SC-005). The item path is
  a straight line: gate → baseline → mode → attempts → publish → merge → status change →
  finish.
- **Alternatives:**
  - Model the fixture as two chained item runs: this changes the fixture's evidence and the
    golden audit.
  - Generalise A/B into a list of N items: that is 015's scheduler, which is out of scope.

## R2 — Target profiles

- **Decision:** `~/.config/specfoundry/targets/<name>.env`, mode 0600, chosen with `--target
  <name>` on `run`, `status`, `audit`, `gate`, `github …` and `reset-fixture`. Keys are in
  [contracts/target-profile.md](contracts/target-profile.md).
  - In token mode, the target's publisher token is in `targets/<name>.publisher.env`.
  - With no `--target`, `pilot.env` alone behaves exactly as today (the implicit target
    `fixture`).
  - `resume` and `approve` read the target from `run.json`, never from a flag, so a paused run
    always continues against its own target.
- **Repository guard:** a target repository must be `owner/name`, must not be this checkout's
  origin (the canary), and must not be another profile's companion repository. The
  `-pilot-fixture` naming rule applies only to the fixture target.
- **011 mapping:** each key has a cluster-resource field of the same meaning; secret file paths
  become Secret references (FR-004b).
- **Alternatives:**
  - Sections inside `pilot.env`: the parser and the 0600 check are per file, and a file per
    target is easier to reason about.
  - A YAML file: the project's configuration is env-style everywhere.

## R3 — Readiness (FR-002, FR-003, FR-007)

The checks run in order, and the first failure refuses with exit 10 (gate refuse) and a reason
code. Each is recorded as a gate row in `gate/<seq>-<label>.json`.
- **(b, c) Dependencies and status:** through `spec status --json` on a fresh clone (the
  existing gate, `gate.EvaluateFresh`). The item must be startable: status `planned`,
  `specced` or `in-progress`, and every dependency verified.
- **(a) Spec and tasks:** the item's `spec_dir` (from the status JSON) must contain `spec.md`
  and `tasks.md` (`ITEM_SPEC_MISSING`, `ITEM_TASKS_MISSING`). `plan.md` is optional for item
  runs; when present it goes into the prompt as today.
- **(d) Open questions:** `spec.md` must not contain `[NEEDS CLARIFICATION`
  (`ITEM_OPEN_QUESTIONS`), and status diagnostics of severity `error` on the item refuse.
  - Roadmap open questions are free text (`Q1 — …`). One that names `NNN`, unless struck
    through or marked resolved, refuses with `ITEM_OPEN_QUESTIONS`.
- **(f) Allowed paths:** an `## Allowed paths` section in `spec.md` with at least one
  backticked glob (`ITEM_NO_ALLOWED_PATHS`). See R11.
- **(e) Hidden tests:** the companion repository has a non-empty `items/NNN/` folder
  (`ITEM_NO_HIDDEN_TESTS`).
- **Unreadable status:** any failure to read status refuses (fail closed, as today).
- **Baseline (FR-007):** after the mode approval and before the first worker, prefetch (R7)
  runs, then the grader runs once in baseline mode on the base revision (R8). The visible
  suite must pass and the hidden check must fail (`BASELINE_SUITE_FAILS`,
  `BASELINE_HIDDEN_PASSES`). This costs no model spend. It comes after the approval because
  prefetch uses the registry hosts the operator approves there.

## R4 — Getting a real repository into the pods

- **Problem:** inputs travel in ConfigMaps (`worker.go:200`, `grader.go:72-84`), which hold at
  most about 1 MiB. A real repository's full-history bundle is larger.
- **Decision:** for item runs, the base is a **single-commit snapshot bundle**.
  - It is a commit with the base revision's tree (the pilot builds it on the host). The
    worker commits on top of it. Before publishing, the pilot replays the change onto the real
    base revision on the host (`git rebase --onto <base> <snapshot>`). The trees are identical,
    so the graded tree doesn't change and the PR has the right parent. The publisher gets a
    small `base..change` bundle, as today.
  - Snapshot bundles, the change bundle and the hidden overlay reach a
    pod through a **staged upload**: the Job's pod has an init container (the tools image)
    that waits for `/in/.ready` on an `emptyDir`.
  - The pilot streams a tar over the existing exec path (`k8s.Exec`) and then touches
    `.ready`. Small text inputs (`meta.json`, `prompt.md`, settings) stay in a ConfigMap.
- **Limit:** `PILOT_MAX_INPUT_MB`, default 50. A larger snapshot refuses with
  `INPUT_TOO_LARGE`, and the size goes into the message.
- **Fixture:** unchanged. It keeps ConfigMaps, because its bundles are small and its
  evidence must not change.
- **Alternatives:**
  - A PVC: namespaced, so it can't be shared between worker, prefetch and grader namespaces.
  - A registry or OCI artifact: a new dependency.
  - Splitting across several ConfigMaps: still capped by the quota (2–3 ConfigMaps).

## R5 — Toolchains: separate images, copied in by an init container

- **Decision (owner, 2026-11-17):** the `worker` and `tools` images stay as they are, one each,
  and always specfoundry's (FR-011b). A language's tools come in a **toolchain image** that holds
  only `/opt/toolchain`, including a self-describing `toolchain.json`. An init container copies
  it into an `emptyDir` mounted at `/opt/toolchain` in the worker, prefetch and grader
  containers. The format is in [contracts/toolchain-image.md](contracts/toolchain-image.md).
- **Three sources:**
  - **The catalog** (`pilot/images/toolchains.json`): `jdk21-maven`, `node24`, `python3.13`
    and `go`, built by the image workflow from checksum-pinned upstream releases. Those are
    Temurin 21, Apache Maven 3.9.x (needed for `maven.repo.local.tail`), the Node.js 24 LTS
    tarball, `python-build-standalone` 3.13 and the official Go tarball. They are pinned in
    `pilot/images.lock` (C-08, C-09).
  - **Operator-supplied:** `PILOT_TOOLCHAIN_IMAGE=registry/name@sha256:…` in the target
    profile, built from the template in `pilot/images/toolchain-template/`. There is a UBI
    Dockerfile for Red Hat and OpenShift, a Debian one for elsewhere, and a GitHub Actions
    workflow that builds, pushes and attests.
    - It must be digest-pinned and pass the contract probe.
    - It is shown at the mode approval and recorded as `operator-supplied`.
    - A GitHub build attestation is verified (`gh attestation verify`) and recorded when
      present, but not required. That lets enterprise CI (Jenkins, Tekton) build them too.
  - **Requested:** through the issue form, becoming a catalog toolchain.
- **The `python` catalog entry:** what the fixture uses, built into the worker and tools
  images. It has no toolchain image.
- **Catalog content:** the image lock key, the proxy kind (`env` for npm, pip and Go;
  `maven-settings` for Maven) and default resources. The image's own `toolchain.json` supplies
  `PATH` and environment, so ours and operators' work the same way. The image workflow,
  `build_args.py`, `digest-pr` and `images.go` read the image list from the catalog.
- **Smoke test:** each catalog image passes the contract probe under an arbitrary UID. The
  tools image grades a tiny offline sample project per toolchain through `test_command` and an
  overlay.
- **FR-011:** an unknown toolchain, or a catalog toolchain whose image has no lock digest,
  refuses with `TOOLCHAIN_UNAVAILABLE`. The message names both ways forward: build your own
  from the template, or use the request form.
- **Threat model:** the adversary is the agent, and it can't change the toolchain image, which
  is operator-side like the test command. An operator-supplied toolchain still runs inside the
  grader, so a tampered image could fake a result. That is the operator's supply-chain
  responsibility. The grader's own code (`grade.py`, in the tools image) is always
  specfoundry's.
- **OpenShift later:** UBI-based variants of the catalog toolchains, chosen per cluster profile.
  The worker and tools images don't change.
- **Alternatives:**
  - Combined worker-plus-toolchain images: every OpenCode or toolchain update rebuilds each
    combination.
  - Only specfoundry-built toolchains: "any language" would wait on us.
  - Operator-supplied tools or worker images: they would put the grader's own logic and the
    agent's sandbox in the operator's hands, which is rejected (FR-011b).

## R6 — Egress for registries

> **Amended 2026-11-17 (implementation):** the worker gets **one** per-target proxy,
> `pilot-egress-w-<target>`, allowing the model host(s) plus the registries: pip, npm and Go
> all read the same `HTTPS_PROXY` the agent uses for the model, so a separate registry proxy
> can't serve them. The prefetch pod gets `pilot-egress-r-<target>` (registries only, never the
> model host). Maven still gets `settings.xml` (proxy kind `maven-settings`).

- **Decision:** model traffic keeps the existing `pilot-egress-model` proxy, set at `up`.
  Registry traffic goes through a **per-target registry proxy**, `pilot-egress-r-<target>`.
  - Its allowlist is exactly the profile's `PILOT_REGISTRY_HOSTS`, and the model host is never
    on it.
  - It is created (or re-rendered if the hosts changed) at run start and removed by `down`.
  - Worker pods may reach both proxies, so the worker NetworkPolicy gains a rule for the
    registry proxy's namespace.
  - The prefetch pod (R7) may reach only the registry proxy.
  - OpenCode keeps `HTTPS_PROXY` (model proxy). Maven gets a generated `settings.xml` with a
    `<proxy>` for the registry proxy, passed with `MAVEN_ARGS=-s /in/settings.xml`, plus
    `-Dmaven.repo.local=/home/worker/.m2/repository`.
- **Mode approval:**
  - The terminal prints the mode as today, plus `egress: model host(s) <n> (from pilot.env),
    registries: <hosts>`.
  - `run.json`'s mode records the registry hosts and the model host count, not the model host
    (AU08 still scans for it).
  - The GitHub decision post lists the registry hosts only.
- **Checks:**
  - P06 gains two rows for item runs: a declared registry is reachable through the registry
    proxy, and an undeclared host is refused.
  - The registry proxy's refusals are copied into the attempt's `proxy-denials.log` (SC-004).
- **Alternatives:**
  - Re-render the shared model proxy per run: it mixes model and registry hosts, and 015 runs
    in parallel.
  - One proxy per run: more churn, with no gain over per-target.

## R7 — Dependencies for the grader

> **Amended 2026-11-17 (implementation):** dependencies are installed by the target's
> `PILOT_INSTALL_COMMAND`, after the toolchain's `setup_command` (for Python: a venv at
> `${PILOT_DEPS}/venv`). It runs in the worker's setup init container (into its own home) and in
> the prefetch Job (into the cache). The cache is mounted at the same path (`/deps`) in prefetch
> and graders, so a venv's absolute paths stay valid. For Python the install command installs
> **dependencies only** (for example `pip install -r requirements.txt`), and the test command
> finds the project's code itself (for example `PYTHONPATH=src python -m pytest`): an editable
> install would point at the prefetch's checkout, not the grader's.: per-run PVC, prefetched from the base revision only

- **Decision (owner, 2026-11-17):**
  - **One grading namespace per run.** Item runs use `pilot-g-<label>` instead of
    `pilot-g-<label>-<n>`. It holds one PVC (`deps`, default StorageClass: local-path on kind,
    the cluster default on OpenShift; size from the catalog, 2Gi for each catalog toolchain).
  - **A prefetch Job runs once per run, before the baseline.** It runs in that namespace and
    resolves dependencies on the **human-controlled inputs only**: the base snapshot plus the
    hidden overlay. It runs the project's test command with the local repository on the PVC,
    and its result is ignored except for the cache. It never runs agent-written code.
    - The prefetch pod carries a label that a NetworkPolicy allows to reach DNS and the
      target's registry proxy (R6) only.
  - **Each attempt's grader Job** runs in the same namespace:
    - the PVC is mounted **read-only** as `maven.repo.local.tail`, with `/tmp/m2` as the
      writable local repository;
    - Maven runs offline (`-o`);
    - its pod label has no egress rule, so it falls under the namespace's default deny.
  - The grader namespace never holds an SCM credential or a model key, as today. The worker
    can't reach it (P07, P08 extended to the run-scoped namespace).
- **Why base only:** agent-written code or build plugins running with network access and a
  writable cache could plant a fake test library and make hidden tests "pass". When prefetch
  sees only human inputs, the cache is trustworthy, and agent code runs only in the worker and
  in the offline grader.
- **Consequence:** an attempt can't add dependencies; an offline build that needs a new one
  fails, and so does the attempt. An item that needs a new dependency has it added to the
  build file by the human while preparing the item. The quickstart's preparation guide says
  so, and says hidden tests may use only dependencies the base build declares.
- **Defaults** (catalog; a profile may raise them):

  | Toolchain | Grader | Prefetch | Worker | Cache PVC |
  |---|---|---|---|---|
  | each catalog toolchain (and operator-supplied) | 2 CPU, 2Gi, deadline 900 s, 600 s per command | same as grader | `/home/worker` 1Gi, `/tmp` 512Mi, `MAVEN_OPTS=-Xmx768m` | 2Gi |
  | `python` | unchanged | none | unchanged | none |
- **Teardown:** the grading namespace and its PVC are deleted when the run ends, and `down`
  deletes any that remain. Their deletion is a teardown row.
- **Alternatives:**
  - Relay a cache tar through the host: no StorageClass needed, but slower; it remains a
    fallback.
  - Prefetch per attempt on the changed code: allows new dependencies, but needs
    cache-poisoning safeguards.
  - Give the grader registry egress: breaks "the grader has no network".

## R8 — Grading changes

- **Decision:** `grade.py` reads optional `meta.test_command`, `meta.hidden_mode` and
  `meta.mode` (`attempt` or `baseline`).
  - **Fixture:** neither field is set, so G4 and G5 are byte-identical to today.
  - **Item run:**
    - G1–G3 unchanged;
    - G4 runs `test_command` on the changed project;
    - G5 copies `/grading/overlay/` (the companion `items/NNN/` tree) over the project and
      runs `test_command` again.
  - **Grader namespace:** per run for item runs (R7); per attempt for the fixture, unchanged.
  - **Baseline mode:** no change bundle. G1 applies the base only and G2/G3 are skipped. It
    reports `g4_ok` and `g5_failed_as_expected`.
- **Rationale:** this matches the clarified overlay answer, and the pipeline decides what each
  result means.
- **Note:** an attempt that edits a hidden test's path is caught by G2 unless the allowed
  paths include it. The overlay always wins, because it is copied last.

## R9 — Companion repository handling

- **Decision:** on the host, the pilot shallow-clones `PILOT_COMPANION_REPO` with the
  operator's own git credentials (the same helper as `FreshClone`), takes `items/NNN/`, and
  tars it for the grader's staged upload.
- **Evidence** records:
  - the companion commit SHA;
  - the overlay's file list and a SHA-256 of the tar;

  but never file contents.
- **Kept out of the worker:** no companion path or content in the worker namespace, prompt,
  inputs ConfigMap or PR.
  - A new check, P11, asserts no overlay file name appears in the worker's ConfigMaps or
    staged input list.
  - AU08's scan also covers the overlay's SHA-256 list.
- **Alternatives:** have the grader fetch the companion repository itself. It would need an
  SCM credential, which breaks AU04 (no grader holds one).

## R10 — Prompt for item runs

- **Decision:** the fixture keeps today's Preamble exactly (FR-015). An item run uses a
  generic preamble:
  - change only the allowed paths (the item's list, rendered from the spec);
  - use the project's toolchain (named);
  - run the project's test command;
  - tick tasks in `<spec_dir>/tasks.md`.

  The spec, plan (if present) and tasks follow as today. Item runs have no induction.

## R11 — Allowed paths

- **Decision:** `spec.md` has an `## Allowed paths` heading followed by list items, each a
  backticked glob, for example `` - `src/main/java/**` ``.
  - Matching is G2's existing fnmatch, where `*` also matches `/`, so `src/main/java/*` covers
    subfolders. The contract says so explicitly.
  - The item's `<spec_dir>/tasks.md` is always added.
  - A glob that matches `.github/**`, `.specify/**` or the roadmap file refuses
    (`ITEM_PATHS_TOO_WIDE`), so an item can't change the roadmap or CI.

## R12 — Status change

- **Decision:** `reconcile.StatusEdit` accepts `planned`, `specced` or `in-progress` →
  `verified` for the one item, still exactly one line. The PR body names the old status.
- **Fixture:** unchanged (`planned`).
- `CheckStatusPR` is unchanged (exactly one changed file, the edit equals `StatusEdit(base)`,
  merged by an approver, not by the publisher).

## R13 — Audit for item runs

- **Decision:** same 11 ids, so a run still reports "11/11". The checks below dispatch on
  `kind`; the rest are unchanged. Fixture and pre-014 records keep today's checks, which keeps
  the golden data unchanged.

  | Check | Item-run variant |
  |---|---|
  | AU01 | Only the target item was gated `allow`. No other item was attempted, and the readiness rows are recorded. |
  | AU02 | Grade results are 0 or more `grade_fail` followed by exactly one `grade_pass` as the last attempt, and a passing `baseline` came before the first attempt. |
  | AU03 | Exactly 1 task PR plus at most 1 status PR, all drafts by the publisher identity. |
  | AU11 | No attempt was induced, and every prompt contains its spec unaltered (the spec's SHA-256 is recorded in `meta.json`). |
- New P-rows (P06 registry, P11 overlay) are in `checks/results.jsonl`. AU04 also requires P11
  for item runs.

## R14 — GitHub loop on a target

- **Decision:** the profile carries `PILOT_GITHUB_ISSUES` and `PILOT_GITHUB_APPROVALS`.
  - Both default to `pilot.env`'s values. Issues default to `off` for a new target, because
    sync mirrors the whole roadmap into the project's issues and the owner should opt in.
  - `IssueFor` and `DriftReport` map by item id instead of `paths.Tasks`.
  - The GitHub App must be installed on the target repository. `app-token --check --target
    <name>` probes it, with this repository as the canary.

## R15 — Evidence location for item runs

- **Decision:** item runs write to `pilot/runs/<run-id>/` (git-ignored) unless
  `PILOT_EVIDENCE_FEATURE` or an active feature points elsewhere, as today.
- **Validation runs:** copied into `specs/014-any-project/evidence/`.
- `summary.md` lists the run's tasks from `run.json`, not a fixed A/B list.

## R16 — Validation repository (kind)

- **Recommendation:** `spring-guides/gs-rest-service`, `complete/` folder.
  - Apache-2.0; Spring Boot 4 (JDK 17+, validated on the `jdk21-maven` toolchain); JUnit 5 with two tests; Maven Central only.
  - Copied as one squashed commit into an owner-controlled public repository, with a private
    companion `<name>-acceptance`. The Gradle files are removed.
  - Three prepared items, each with hidden tests under `src/test/java/...`:
    - 001: validate `name`; return 400 with an error body;
    - 002: an in-memory greeting-templates resource;
    - 003: greetings by language using templates (depends on 002).
- **Fallback:** Gilded Rose (Java) if Spring's memory needs block the first run. It is MIT,
  JUnit only, and has a much smaller dependency set.
- **Not chosen:**
  - `spring-guides/tut-rest`: has no tests, so it isn't brownfield.
  - `spring-petclinic`: too large.
  - The Javalin samples: Kotlin, Selenium and JUnit 4.

## R17 — Fixture-only commands

- `reset-fixture` refuses any target other than the fixture: it force-pushes, so it must never
  touch a real project.
- The run's open-PR precheck: for item runs, an open `pilot/*/<item>` PR on the target
  refuses (`OPEN_ITEM_PR`). For the fixture it is unchanged.

## R18 — Constitution VII (OpenShift)

- **Decision:** this feature's validation is on kind. The OpenShift EAP validation needs 005
  and an in-cluster model, so it is deferred to the roadmap's step 8.
- **The design stays OpenShift-ready:**
  - images run with an arbitrary UID and group 0;
  - no new cluster-wide controllers;
  - the proxy model is unchanged;
  - staged uploads use exec, which the restricted SCC allows.
