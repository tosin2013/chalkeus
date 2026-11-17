# Implementation Plan: Any project, any item

**Branch**: `014-any-project` | **Date**: 2026-11-17 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/014-any-project/spec.md`

## Summary

Let the pilot run **one prepared item of a real repository** (`pilot run --target <name>
--item NNN`) under every existing guarantee, while the fixture's A→B run stays exactly as it
is.

The design adds an `item` run kind beside the `fixture` kind ([research](research.md) R1):
- **Target profiles** (R2): per-repository settings on the operator's machine. They become a
  cluster resource in 011.
- **Fail-closed readiness checks** (R3): through spec-status, plus spec, tasks, open questions,
  allowed paths and hidden tests. After the mode approval, a **baseline** proves the hidden
  tests fail on the base and the suite passes.
- **Large repositories** (R4): a single-commit snapshot, streamed into pods through a staged
  upload instead of the 1 MiB ConfigMaps.
- **Toolchains** (R5): separate small images, copied into the
  worker, prefetch and grader pods by an init container. The catalog has `jdk21-maven`,
  `node24`, `python3.13` and `go`; an operator may supply their own digest-pinned image
  instead. The worker and tools images are unchanged and always specfoundry's.
- **Registry egress** (R6): a per-target registry proxy, separate from the model proxy and
  shown at the mode approval.
- **Dependencies for the offline grader** (R7): a per-run grading namespace with a PVC. It is
  filled once by a prefetch Job from **human-controlled inputs only** (the base plus the hidden
  overlay) and mounted read-only into each attempt's grader.
- **Hidden tests** (R8, R9): read from a private companion repository on the host and overlaid
  at grade time. The project's own test command runs both checks.
- **Status change** (R12): proposes `verified` from `planned`, `specced` or `in-progress`.
- **Audit** (R13): item-run variants of AU01, AU02, AU03 and AU11, plus new check rows P06r,
  P11, P12 and teardown T04, T05. It still reports 11 checks.

The first validation is on kind with a copy of `spring-guides/gs-rest-service` (R16). The
OpenShift EAP validation follows 005 (R18).

## Technical Context

**Language/Version**: Go (the module's `go` version), as 016. The in-image scripts stay Python
3.11 (Debian). The new toolchain image adds Eclipse Temurin 17 and Apache Maven 3.9.x
(tarballs, checksum-pinned).

**Primary Dependencies**: no new Go modules. It reuses client-go (Jobs, PVCs, exec streaming,
NetworkPolicies), Cobra, and the existing `spec` binary for status. Images: the existing
`worker`, `tools` and `egress`, plus the catalog toolchain images (`toolchain-jdk21-maven`,
`toolchain-node24`, `toolchain-python3.13`, `toolchain-go`).

**Storage**:
- **Files:** `~/.config/specfoundry/targets/*.env`, and the evidence under `pilot/runs/`
  (`specs/014-any-project/evidence/` for validation).
- **In the cluster:** one PVC per item run (default StorageClass: local-path on kind).

**Testing**:
- **Unit tests** (`go test ./...`): profile parsing and validation, readiness rules, the
  allowed-paths parser, the status edit for three source statuses, the item-run audit
  variants (synthetic records), snapshot and replay of a change (git fakes), manifests
  rendering (golden files), and the toolchain catalog.
- **Unchanged:** the golden audit and the shared vectors.
- **In-image:** `grade.py` modes (baseline, overlay, `test_command`) tested in
  `pilot/images/tests/`.
- **Image smoke test:** a Maven case.
- **Owner runs:** the [quickstart](quickstart.md).

**Target Platform**: Linux, kind with Calico (laptop). OpenShift validation is deferred (R18),
but the design stays SCC-compatible.

**Project Type**: CLI plus in-cluster Jobs (existing structure).

**Performance Goals**: human-paced. A small Maven item finishes prefetch plus baseline in under
5 minutes, and each grading in under 10 minutes (catalog deadlines).

**Constraints**:
- The fixture's commands, prompts, exit codes, evidence and 11/11 audit are unchanged
  (FR-015).
- Hidden tests never reach the worker (FR-008).
- No agent-controlled input changes egress or the dependency cache (R6, R7).
- Every image comes from the image workflow, pinned by digest.
- Fail closed everywhere.

**Scale/Scope**:
- 5 catalog toolchains (`python` built in; `jdk21-maven`, `node24`, `python3.13`, `go`), plus
  operator-supplied images;
- 1 to N targets (one run at a time; parallel runs are 015);
- items whose snapshot is up to 50 MB by default;
- about 12 packages touched, 1 new image, 6 contracts.

## Constitution Check

| Principle | Check | Result |
|---|---|---|
| I. Roadmap is the source of truth | Items come only from the target's roadmap and `spec status` (FR-001). The status change is one roadmap line, approved by a human. | pass |
| II. Gates fail closed | Readiness rules extend the gate. Unreadable status, a companion repository that can't be read, a missing toolchain image or a failed baseline all refuse (R3, contracts/readiness.md). | pass |
| III. Evidence defines done | `verified` only after independent grading (the repository's suite plus hidden tests), an approved status change and the audit. The baseline proves the hidden tests carry signal (FR-007). | pass |
| IV. Least privilege | Workers: no SCM credential; model proxy plus a registry proxy only. Graders: no network, no credential, a read-only cache. Prefetch: registries only, no secrets, human inputs only. Hidden tests are fetched on the host with the operator's access; the grader never holds a credential (AU04, P11, P12). | pass |
| V. Humans approve, humans merge | Same decisions (mode, retry, status) and channels; draft PRs only; no merge path. `reset-fixture` can never touch a real target (R17). | pass |
| VI. Build on, don't rebuild | `spec status` for readiness; Spec Kit artifacts as the item's definition; upstream JDK and Maven releases verified by checksum. No new Go dependency. | pass |
| VII. Repeatable and portable | Laptop: kind with its default StorageClass. Enterprise: the design avoids cluster-wide controllers and root, and works under an arbitrary UID. **The OpenShift validation is deferred** until 005 is verified and an in-cluster model is available (R18; roadmap delivery order step 8). | pass, with the R18 note |

**Post-design re-check:**
- **Two new trust edges:**
  - the prefetch pod runs the human's build with registry egress (human inputs only);
  - the per-target registry proxy widens worker egress only to the hosts the operator lists
    and approves.
- **Two new artifacts that can't change grading:** the cache PVC is read-only to graders, and
  the companion repository is read on the host.

Still passes.

## Project Structure

### Documentation (this feature)

```text
specs/014-any-project/
├── spec.md, plan.md, research.md, data-model.md, quickstart.md
├── contracts/
│   ├── target-profile.md   # targets/<name>.env keys, validation, 011 mapping
│   ├── item-spec.md        # "## Allowed paths" and the companion items/NNN layout
│   ├── readiness.md        # refusal codes and order
│   ├── cli.md              # --target/--item, targets, gate, reset-fixture, down
│   ├── grading.md          # meta.json fields, baseline/overlay modes, pod shapes
│   └── audit-item.md       # AU01/02/03/11 item variants, P06r, P11, P12, T04, T05
├── checklists/requirements.md
└── tasks.md                # /speckit-tasks
```

### Source code (repository root)

```text
internal/
├── config/       # + targets.go: profile loading, validation, implicit fixture target
├── paths/        # Tasks/SpecDirs stay for the fixture; + item spec-dir resolution
├── readiness/    # NEW: rules a–f on a fresh clone + status JSON; reason codes
├── gate/         # unchanged adapter; readiness wraps it
├── pipeline/     # + item.go: the item-run state machine; Run/Resume dispatch on kind
├── worker/       # snapshot bundle, staged upload, item preamble inputs, toolchain init, registry proxy env
├── grader/       # run-scoped namespace, prefetch + baseline + attempt Jobs, overlay staging
├── prompt/       # + item preamble (fixture preamble unchanged)
├── publisher/    # replay onto the real base; item wording in the PR body; target repo
├── reconcile/    # status edit from planned|specced|in-progress
├── cluster/      # registry proxy per target; down: T04/T05
├── images/       # toolchain catalog; lock keys from it
├── checks/       # P06r, P11, P12
├── audit/        # kind dispatch + item variants
├── evidence/     # kind/target/item fields; summary.md from run.json tasks
├── ghloop/       # item-id mapping (IssueFor/DriftReport), target repo
├── k8s/          # exec streaming into a pod (tar), PVC create/delete
└── cli/          # --target/--item flags, `targets`, gate --item, reset-fixture refusal
pilot/images/
├── toolchains.json                 # NEW catalog
├── toolchain-{jdk21-maven,node24,python3.13,go}/Dockerfile   # NEW: /opt/toolchain only
├── toolchain-template/             # NEW: Dockerfile.ubi, Dockerfile.debian, toolchain-image.yml
├── tools/grade.py                  # + baseline mode, test_command, overlay
├── tests/                          # + grade.py mode tests, Maven sample
└── smoke.sh                        # + Maven case
pilot/manifests/
├── worker-job.yaml.tmpl, grader-job.yaml.tmpl    # + toolchain init, receiver init, cache mount (item runs)
├── grading-ns.yaml.tmpl            # NEW: run-scoped namespace, PVC, role-labelled NetworkPolicies
├── prefetch-job.yaml.tmpl          # NEW
└── egress.yaml.tmpl                # reused for the registry proxy
pilot/pins.env                      # + TEMURIN_17_*, MAVEN_* pins
.github/workflows/pilot-images.yml  # image list from toolchains.json
.github/ISSUE_TEMPLATE/toolchain-image.yml        # NEW (FR-012)
docs/operator-guide.md, docs/troubleshooting.md, docs/targets.md (NEW)   # operator docs
```

**Structure Decision:** extend the existing packages. Readiness is new logic with its own
reason codes, so it gets its own package. The item-run state machine is a separate file
beside the fixture's in `pipeline/`, so the fixture path isn't edited beyond dispatch.

## Delivery slices (for /speckit-tasks)

1. **Foundations:**
   - target profiles and `pilot targets`;
   - run `kind` and `target` in `run.json`;
   - audit dispatch, with the fixture unchanged and the golden test green.
2. **US2, readiness:** `gate --target --item`, every refusal code. No cluster needed.
3. **US1, the item run on the existing `python` toolchain:** snapshot plus staged upload, the
   item prompt, overlay grading, the run-scoped grading namespace (PVC unused for python),
   the status edit, the audit variants. Validated first on a Python sample target.
4. **US3, toolchains and egress:**
   - the toolchain contract and probe, the catalog images, operator-supplied images, and the template;
   - the registry proxy;
   - prefetch plus the PVC;
   - the baseline;
   - the Maven grading.
5. **Validation:** the owner's `greeting-service` items 001–003 (quickstart §4–§5).
6. **US4, regression:** a fixture run, 11/11.
7. **Docs and close-out:** `docs/targets.md`, operator guide and troubleshooting; the issue form;
   Evidence section; debrief.

## Complexity Tracking

| Item | Why needed | Simpler alternative rejected because |
|---|---|---|
| Two run kinds in one pipeline | The fixture must stay byte-identical (FR-015), and a real item is a single straight path | Rewriting the fixture as item runs changes its evidence and the golden audit |
| A per-run PVC and a prefetch Job | The grader has no network, and must not trust agent-built caches | Grader egress breaks Principle IV; a host relay is slower (kept as a fallback); per-attempt prefetch can be poisoned by agent code |
| Staged upload over exec | ConfigMaps are capped near 1 MiB | A PVC per worker namespace adds a loader step anyway; a registry or OCI artifact adds a dependency |
