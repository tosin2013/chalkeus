# Implementation Plan: macOS, multi-arch images and kind on Docker Desktop

**Branch**: `plan-031-macos-multiarch` | **Date**: 2026-11-24 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/031-macos-multiarch/spec.md`

## Summary

Most of the cluster side is already platform-neutral: the base images, busybox and the kind
node image are multi-platform indexes, the kind and kubectl install picks the host's OS and
architecture, and forge's Go code has no platform logic (R1, R2). The work is:

1. **Per-architecture pins and Dockerfiles.**
   - `pilot/pins.env` gets an `_AMD64` and an `_ARM64` value for every upstream input that is
     amd64-only today: OpenCode, spec-status (moved to v0.4.1, the first release with
     `linux_arm64` and `darwin_amd64`) and Python 3.13 (R3, R4).
   - The worker, tools and Python toolchain Dockerfiles pick the asset and checksums by
     `TARGETARCH` and fail on any other value, as the Node, Go and JDK toolchains already do.
   - `build_args.py` and its Go twin in `internal/images` pass both architectures' values and
     fail when a pin is missing.
2. **Multi-platform image workflow.**
   - Each image is built natively on `ubuntu-24.04` (amd64) and `ubuntu-24.04-arm` (arm64),
     smoke-tested on its own platform, and pushed by digest.
   - A merge job joins the two into one index per image, and the digest PR writes the index
     digest to `pilot/images.lock`.
   - `image-digests.json` lists the per-platform digests (R5, R6).
3. **Host and runtime checks.**
   - doctor gets a "host" area: host OS and architecture, container runtime and version, the
     cluster node's platform, and each locked image's platforms.
   - It refuses an unsupported host and a stopped Docker Desktop.
   - The memory check is 030's capacity check (R8, R9).
4. **Portable host scripts.** `test/isolation/tools.sh` and `run.sh` use a checksum command
   that exists on both macOS and Linux (R7).
5. **Evidence.** `versions.txt` gains host, runtime and platform lines (R10).
6. **Docs and proof.**
   - The operator guide's macOS path, troubleshooting entries and release notes.
   - Evidence runs on the owner's Apple Silicon Mac: isolation suite, fixture run, target run
     (R11).

## Technical Context

**Language/Version**: Go 1.27 (`cmd/chalkeus`, `internal/`); Python 3.13 stdlib for
`pilot/images/build_args.py`; bash for `test/isolation/*.sh`; Dockerfiles; GitHub Actions YAML

**Primary Dependencies**: Docker Buildx (CI, already used); `docker buildx imagetools` (CI merge
job, and read-only on the host for doctor's platform check); kind v0.27.0 and kubectl v1.32.2
(unchanged pins); spec-status v0.4.1 (was v0.4.0); no new Go module

**Storage**: `pilot/pins.env`, `pilot/images.lock`, `image-digests.json` (CI artifact), run
evidence `versions.txt`; no new store

**Testing**: `go test ./...` (doctor host rows, build-arg parity, versions lines);
`pytest pilot/images/tests/` (`build_args.py` per-arch and missing-pin cases); the image
workflow on a PR (both platforms build and smoke-test; nothing is pushed); the isolation suite
and fixture run on kind on Linux amd64 (unchanged) and on the owner's Mac (new evidence); the
docs check

**Target Platform**: hosts darwin/arm64 (validated), darwin/amd64 (by design), linux/amd64
(validated, unchanged), linux/arm64 (allowed, untested); cluster nodes linux/amd64 and
linux/arm64

**Project Type**: CLI, image build and docs (single Go module plus `pilot/images`)

**Performance Goals**: the image workflow's wall time stays about the same (the two platforms
build in parallel on native runners); doctor's host checks add under 5 seconds (one
`docker info`, one `docker version`, one `kubectl get nodes`, `imagetools inspect --raw` per
locked image)

**Constraints**:
- C-09: images built only in GitHub, inputs pinned by checksum, a mismatch fails the build with
  no fallback, `images.lock` changed only by the digest PR.
- Linux amd64 behaviour and audit unchanged (FR-005, SC-006).
- No isolation check skipped or relaxed for macOS (FR-008, FR-009).
- The spec-status host binary is not checksum-checked by forge today; this item doesn't
  change that.

**Scale/Scope**: 7 images × 2 platforms; 3 inputs gain arm64 pins (OpenCode, spec-status,
Python); 4 host platforms recognised

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | How 031 complies |
|---|---|
| I. The roadmap is the source of truth | Ordinary roadmap item; the gate and scheduling are unchanged |
| II. Gates fail closed | A missing or mismatched per-arch pin fails the build; an unsupported host, a stopped runtime, an image with no build for the node's platform, or an unproven isolation check makes forge refuse. Only "couldn't read the registry" warns (R9), as 030's unreadable capacity does |
| III. Evidence, not CI, defines done | macOS support is proven by isolation, fixture and target runs on the Mac with the audit at 11/11, not by CI building arm64 images; `versions.txt` records host and platforms |
| IV. Least privilege (non-negotiable) | No credential, network policy, egress allowlist or storage change; the CI merge job uses the same `packages: write` token as today's push step |
| V. Humans approve, humans merge | The digest PR, the spec-status bump and every evidence run's PRs are merged by the owner |
| VI. Build on, don't rebuild | Reuses the workflow, the `TARGETARCH` pattern of the Node, Go and JDK toolchains, `tools.sh`'s OS and arch detection, doctor's report and `versions.txt` |
| VII. Repeatable and portable | Every input is pinned per architecture; images are pinned by index digest; the macOS path is documented and proven from a fresh Mac |

**Result:** pass, no violations. **Re-check after Phase 1:** still pass. One trade-off is
recorded in R5: a digest PR now carries index digests, so Linux amd64 users pull new digests
whose amd64 build has the same contents as before; the fixture run on Linux proves behaviour is
unchanged (SC-006).

## Project Structure

### Documentation (this feature)

```text
specs/031-macos-multiarch/
├── plan.md              # this file
├── research.md          # R1–R12
├── data-model.md        # pins, lock entry, image-digests.json, host report, versions lines
├── quickstart.md        # validation scenarios
├── contracts/
│   ├── pins.md          # per-architecture keys in pilot/pins.env and the build args
│   ├── image-workflow.md# build, smoke, merge and digest PR across two platforms
│   └── doctor.md        # the host area: rows, statuses, messages
└── tasks.md             # /speckit-tasks
```

### Source Code (repository root)

```text
pilot/
├── pins.env                         # per-arch keys (contracts/pins.md); SPEC_* to v0.4.1
├── images/
│   ├── build_args.py                # both arches' args; fail on a missing pin
│   ├── worker/Dockerfile            # pick OpenCode asset and checksums by TARGETARCH
│   ├── tools/Dockerfile             # pick spec-status asset and checksums by TARGETARCH
│   ├── toolchain-python3.13/Dockerfile  # pick Python asset by TARGETARCH
│   └── tests/test_build_args.py     # NEW: per-arch and missing-pin cases
.github/workflows/pilot-images.yml   # platform matrix, push by digest, merge job, lock PR
internal/
├── images/images.go                 # local-build args mirror build_args.py (parity test)
├── doctor/
│   ├── host.go                      # NEW: host, runtime, node platform, image platforms rows
│   └── host_test.go
├── cli/doctor.go                    # wire the host area before the cluster area
├── cluster/cluster.go               # `up` refuses an unsupported host or stopped runtime early
└── evidence/evidence.go             # versions.txt host and platform lines
test/isolation/
├── tools.sh                         # portable checksum command
└── run.sh                           # portable checksum command
docs/
├── operator-guide.md                # macOS path; section 2, 3 and 13 updates
├── troubleshooting.md               # Docker Desktop entries
└── release-notes.md                 # 031 section
specs/031-macos-multiarch/evidence/  # Mac runs (isolation, fixture, target)
```

**Structure Decision**: no new package. The host checks are a new file in `internal/doctor`,
called from `cli/doctor.go` and, for the two refusals, from `cluster.UpCluster`. Image work
stays in `pilot/images` and the workflow.

## Delivery order

1. **Pins and Dockerfiles (US1 part 1):** per-arch pins, `build_args.py` and its Go twin with a
   parity test, the three Dockerfiles. On a PR the workflow builds amd64 as before; a local
   `docker buildx build --platform linux/arm64` on the Mac proves arm64 builds.
2. **Multi-platform workflow (US1 part 2):** matrix, push by digest, merge job, digest PR with
   index digests. Merging it publishes; the owner merges the digest PR.
3. **Linux regression (SC-006):** the fixture run on Linux amd64 on the new digests; evidence
   compared with the 026 run.
4. **Host checks and evidence (US4):** doctor host area, `up` refusals, `versions.txt` lines,
   portable scripts. The memory row waits for 030's capacity check (R8).
5. **Mac proof (US2, US3):** `up`, isolation suite, fixture run, a target run on the owner's
   Mac; evidence in `specs/031-macos-multiarch/evidence/`.
6. **Docs (US5)** and release notes; then the debrief.

## Complexity Tracking

No constitution violations to justify.
