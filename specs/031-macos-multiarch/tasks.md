# Tasks: macOS, multi-arch images and kind on Docker Desktop

**Input**: `specs/031-macos-multiarch/` (spec.md, plan.md, research.md, data-model.md,
contracts/, quickstart.md)

**Tests**: included. Unit tests for build args, doctor's host rows and `versions.txt`; the image
workflow proves both platforms on a PR; the isolation suite, fixture run and a target run prove
the Mac end to end.

## Phase 1: Setup

- [X] T001 Replace the single-arch OpenCode, spec-status and Python 3.13 keys in `pilot/pins.env` with the per-arch keys and values of contracts/pins.md: `OPENCODE_ASSET_{AMD64,ARM64}`, `OPENCODE_ASSET_SHA256_{AMD64,ARM64}`, `OPENCODE_SHA256_{AMD64,ARM64}`, `SPEC_ASSET_{AMD64,ARM64}`, `SPEC_ASSET_SHA256_{AMD64,ARM64}`, `SPEC_SHA256_{AMD64,ARM64}`, `PYTHON313_ASSET_{AMD64,ARM64}`, `PYTHON313_ASSET_SHA256_{AMD64,ARM64}`; set `SPEC_VERSION=0.4.1`, `SPEC_COMMIT=64ae0ee3a6a5cfa1e30c5a21547961ae2eea6e5b`, `SPEC_RELEASE=tosin2013/spec-status@v0.4.1`; remove the old keys ("a hard switch as in 027"); update the comment block above the 003 inputs to say every upstream input is pinned per architecture

## Phase 2: Foundational (blocks US1–US4)

- [X] T002 [P] In `pilot/images/build_args.py`, emit both architectures' values for `worker`, `tools` and `toolchain-python3.13` (contracts/pins.md "Build args"), and exit non-zero with `pins.env: missing <KEY>` when any key an image needs is absent; add `pilot/images/tests/test_build_args.py` covering every image's keys, the per-arch values, and the missing-pin message
  Done as (2026-11-24): `pick()` and `per_arch()` in `build_args.py`; every key an image needs goes through `pick`, so a missing one exits with `pins.env: missing <KEY>`. `test_build_args.py` also checks that the two architectures' values differ and that the old keys are gone.
- [X] T003 [P] Mirror T002 in `internal/images/images.go` `BuildArgs` (same keys, same `missing <KEY>` error); add a test in `internal/images/images_test.go` that runs `python3 pilot/images/build_args.py <image>` for each image and compares it with `BuildArgs` on the same `pins.env` (parity, contracts/pins.md)
  Done as (2026-11-24): the Go twin still covers only the three images `--local-build` makes (worker, tools, egress), as before; the existing `TestBuildArgsMatchPythonScript` is the parity test, and `TestBuildArgsMissingPin` was added.
- [X] T004 Update every other reader of the removed keys: the `image-digests.json` step in `.github/workflows/pilot-images.yml` (`inputs` per arch, data-model.md), `internal/evidence/evidence.go` (`spec:` and `opencode:` lines print the checksum for the node's architecture, R10; temporarily amd64 until T020 supplies the node platform), and `docs/harnesses.md`; `grep -rn 'OPENCODE_ASSET\b\|OPENCODE_SHA256\b\|SPEC_ASSET\b\|SPEC_SHA256\b\|PYTHON313_ASSET\b' --include='*' .` finds nothing outside `specs/`
  Done as (2026-11-24): also `.github/workflows/go.yml` and `integration.yml`, which install `spec` on amd64 runners and now check `SPEC_SHA256_AMD64` (missed by the plan). The docs check then required the spec 0.4.0 → 0.4.1 references in `docs/operator-guide.md`, `docs/troubleshooting.md` and the `docs_test.go` fixture.

**Checkpoint**: `go test ./...` and `pytest pilot/images/tests/` pass; nothing yet builds arm64.

## Phase 3: User Story 1 - Every image runs on arm64 and amd64 (Priority: P1) 🎯 MVP

**Goal**: every image in the lock is one index digest with `linux/amd64` and `linux/arm64` builds, each input checked against its per-arch pin.

**Independent Test**: quickstart §2 and §3: arm64 builds locally on the Mac and fails on a wrong pin; after merge, every lock entry lists both platforms.

- [X] T005 [P] [US1] In `pilot/images/worker/Dockerfile`, add `ARG TARGETARCH` and select `OPENCODE_ASSET_*`, `OPENCODE_ASSET_SHA256_*` and `OPENCODE_SHA256_*` with the `case` of contracts/pins.md ("An empty `TARGETARCH` fails; it does not default to amd64"); keep both checks (upstream `SHA256SUMS`, then the pin) and the extracted-binary check
- [X] T006 [P] [US1] Same for `pilot/images/tools/Dockerfile` (`SPEC_ASSET_*`, `SPEC_ASSET_SHA256_*`, `SPEC_SHA256_*`; upstream `spec_0.4.1_checksums.txt` then the pin)
- [X] T007 [P] [US1] Same for `pilot/images/toolchain-python3.13/Dockerfile` (`PYTHON313_ASSET_*`, `PYTHON313_ASSET_SHA256_*`; upstream `SHA256SUMS` then the pin)
- [X] T008 [P] [US1] In `pilot/images/toolchain-node24/Dockerfile`, `toolchain-go/Dockerfile` and `toolchain-jdk21-maven/Dockerfile`, change `${TARGETARCH:-amd64}` to fail on an empty or unknown `TARGETARCH` (contracts/pins.md)
- [X] T009 [US1] Build every image for `linux/arm64` on the Mac (quickstart §2, all seven images, `--platform linux/arm64 --load`), run `pilot/images/smoke.sh` against the arm64 builds, and repeat one build with a deliberately wrong `_ARM64` checksum to confirm it fails; record the commands and results in `specs/031-macos-multiarch/evidence/local-arm64-build.md`. A local check only: published images still come from GitHub (C-09)
  Done as (2026-11-24): `evidence/local-arm64-build.md`. All seven images build for linux/arm64 and pass `smoke.sh` (12 checks); a wrong arm64 archive or binary checksum and a missing `TARGETARCH` each fail the build. `smoke.sh` needed two macOS host fixes first (GNU `sed -i`; an empty array under `set -u` in bash 3.2).
- [X] T010 [US1] Rework `.github/workflows/pilot-images.yml` per contracts/image-workflow.md:
  Done as (2026-11-24): `build` (matrix `platform`, asserts each loaded image is `linux/<arch>`), `merge` and `digest-pr` (needs `merge`); actionlint reports only the five notes already on `main`. The PR run proves both build legs; `merge` runs only on `main`, so it is proven at T012.
  - a `platform` matrix on `build`, on `ubuntu-24.04` and `ubuntu-24.04-arm`;
  - each leg builds with `--load` and smoke-tests; on `main` it pushes by digest (`push-by-digest=true,name-canonical=true`, provenance `mode=max`, SBOM) and uploads `digests-<platform>.json`;
  - a new `merge` job (`needs: build`, skipped on PRs) runs `docker buildx imagetools create -t <ref>:sha-<short>` per image, fails unless each index lists exactly `linux/amd64` and `linux/arm64`, and writes `image-digests.json` with `images` (index digests) and `platforms` (data-model.md);
  - `digest-pr` needs `merge`; the F1 no-rebuild-loop rules and the lock-only diff check are unchanged.
- [X] T011 [US1] Update `specs/003-m2-kelos-backend/contracts/image-workflow.md` with a short "Amended by 031" note pointing to `specs/031-macos-multiarch/contracts/image-workflow.md`
- [X] T012 [US1] Owner: merge T001–T011 so the workflow publishes both platforms, then merge the digest PR that updates `pilot/images.lock` with index digests. Then run quickstart §3's `imagetools inspect` loop and record its output in `specs/031-macos-multiarch/evidence/image-platforms.md`
  Done as (2026-11-25): the owner merged #193 and the digest PRs (#196, #200); the `imagetools inspect` loop lists `linux/amd64` and `linux/arm64` for all seven images in `pilot/images.lock` ([evidence/image-platforms.md](evidence/image-platforms.md)).

**Checkpoint**: published images are multi-platform; nothing on the host has changed yet.

## Phase 4: Linux regression (FR-005, SC-006)

- [X] T013 [US1] On the Linux amd64 machine, after T012: `./chalkeus forge up`, the fixture walkthrough, `down`, audit (quickstart §4). Compare with run 20261123T042536Z (026): same check results and audit 11/11; `versions.txt` differs only in the new image digests, the spec 0.4.1 line and (after T020) the new lines. Evidence in `specs/031-macos-multiarch/evidence/<run>/`
  Done as (2026-11-25): run [20261125T170145Z](evidence/20261125T170145Z/), approvals on GitHub (`/approve mode`, `/approve retry`, and the owner's merges of #73 and status PR #74). Checks 32/32, audit 11/11 and teardown 3/3, with the same ids and results as 026's run; A/1 failed on G5_hidden_tests (induction), A/2 and B passed. `versions.txt` differs only in the run id, the `chalkeus-*` index digests, `spec 0.4.1` and the new host lines.
  - A first run, [20261125T162546Z](evidence/20261125T162546Z/), had the same results, but this machine's own spec-status was 0.3.0. The gate runs the host's copy, and `up` doesn't install it, so its `spec:` line paired 0.3.0 with the pinned 0.4.1 checksum. After installing the pinned 0.4.1 (sha256 checked), the run was repeated. Follow-up in the same change: `versions.txt` now records the pinned version on `spec:` and the binary the host actually ran on a new `host_spec:` line, and doctor's host area gains a `spec-status` row that warns when the host's copy isn't the pinned release (contracts/doctor.md, data-model.md).

## Phase 5: User Story 4 - Know which host and platform a run used (Priority: P2)

Built before US2 and US3 because `up`'s early refusals and the evidence lines are part of the Mac proof.

**Goal**: doctor's host area, `up`'s early refusals, and the host and platform lines in every run's evidence.

**Independent Test**: unit tests for each row and status of contracts/doctor.md; doctor on the Mac shows `darwin/arm64 … validated` and `Docker Desktop …`; with Docker Desktop quit, doctor and `up` refuse with "start Docker Desktop".

- [X] T014 [P] [US4] Add `internal/doctor/host.go` with pure functions for the `host` row (the R9 table: darwin/arm64 and linux/amd64 pass "validated"; darwin/amd64 pass "supported by design, not maintainer-validated"; linux/arm64 warn "untested"; anything else fail with the supported list) and the `container runtime` row (Docker Desktop passes on macOS; another runtime warns "only Docker Desktop is validated on macOS; the isolation suite and audit still decide"; an unreachable daemon fails with "start Docker Desktop" on macOS, "start the Docker daemon" on Linux), taking GOOS, GOARCH, the macOS version and parsed `docker info`/`docker version` fields as inputs; tests in `internal/doctor/host_test.go` for every row and status of contracts/doctor.md
- [X] T015 [P] [US4] Add to `internal/doctor/host.go` the `node platform` row (pass for `linux/amd64` or `linux/arm64`, otherwise fail) and the `image platforms` row: for each pinned lock image, the platforms from `docker buildx imagetools inspect --raw` JSON. Fail naming the image when the node's platform is missing ("pilot/images.lock may predate 031"); warn when the registry can't be read; warn "local build: images built for this host" for `--local-build`. Tests use recorded index JSON in `internal/doctor/testdata/`
- [X] T016 [US4] Wire the host area into `internal/cli/doctor.go` before the cluster area: read GOOS and GOARCH from `runtime`, `sw_vers -productVersion` on macOS, `docker info --format '{{json .}}'` and `docker version --format '{{json .}}'`, and after `up` the node's `.status.nodeInfo.operatingSystem/architecture`; add a test with fakes for the commands
  Done as (2026-11-24): the types and probes are in a new leaf package `internal/hostinfo` (evidence needs them too, and `doctor` already depends on `evidence`); `doctor` keeps the rows. `node platform` and `image platforms` are in the **cluster** area after `images` (one header per area; contracts/doctor.md amended). Today's published images are already indexes (provenance makes them so) listing only linux/amd64, so a pre-031 lock on an arm64 node fails with "no build for linux/arm64". Tested at the seam (`TestHostRowsAndLockImages`, `TestProbes` with faked commands): `runDoctor` has no test harness. Live on the Mac: `PASS host: darwin/arm64 (macOS 27.0.1), validated`, `PASS container runtime: Docker Desktop 4.94.0 …, 14 CPUs, 7.7 GiB`.
- [X] T017 [US4] In `internal/cluster/cluster.go` `UpCluster`, run the `host` and `container runtime` checks before "pinned tools" and refuse on a fail with the same message, printing warnings and continuing (contracts/doctor.md "`forge up`"); test that `up` installs nothing on a fail
  Done as (2026-11-24): `cluster.HostCheck`, called first in `UpCluster`; `TestUpClusterRefusesBeforeInstalling` shows a stopped runtime refuses before "pinned tools".
- [X] T018 [P] [US4] In `test/isolation/tools.sh` and `test/isolation/run.sh`, use `sha256sum` when present, otherwise `shasum -a 256`, and refuse with a message if neither exists (R7)
  Done as (2026-11-24): `tools.sh` defines `sha256()`; run on the Mac it installed and verified kind v0.27.0 darwin/arm64 and kubectl v1.32.2.
- [ ] T019 [US4] Depends on 030's capacity check: confirm its local-cluster message names Docker Desktop's Settings → Resources on macOS (FR-012). If 030 isn't implemented yet, leave this task open, and note in the operator guide (T027) that Docker Desktop needs 8 GB; no second memory check is added (R8)
- [X] T020 [US4] In `internal/evidence/evidence.go` `WriteVersions`, append after `host:` the lines `host_os`, `host_arch`, `runtime` (name, version, CPUs, memory), `node_platform` and `image_<k>_platform` for each image (data-model.md "Version lines"; existing lines unchanged); make the `spec:` and `opencode:` lines print the pinned checksum for `node_platform`'s architecture; tests in `internal/evidence/` for the new lines on a darwin/arm64 host with a linux/arm64 node and on linux/amd64
  Done as (2026-11-24): `evidence.HostLines`; `image_<k>_platform` is the node's platform (the runtime pulls the index's build for the node; a local build is made for the host, which is the node). Tests in `internal/evidence/versions_test.go`.
- [X] T021 [US4] Confirm the audit still passes with the new `versions.txt` lines (`go test ./internal/audit/...`) and that nothing parses `versions.txt` by position
  Done as (2026-11-24): only `WriteVersions` and the isolation suite's own copy touch `versions.txt`; nothing reads it back. `go test ./...` passes.

**Checkpoint**: doctor and the evidence describe the host on both Linux and the Mac.

## Phase 6: User Story 2 - Bring up the local cluster on a Mac (Priority: P1)

**Goal**: `forge up` and `down` work on an Apple Silicon Mac with Docker Desktop.

**Independent Test**: quickstart §5 and §6's `up`: `up` succeeds after doctor passes; `down` removes everything.

- [ ] T022 [US2] On the Mac (Docker Desktop with 8 GB memory and 4+ CPUs), after T012 and T017: run `./chalkeus forge doctor`, `./chalkeus forge up` and `./chalkeus forge doctor` again. Then quit Docker Desktop and confirm doctor and `up` refuse with "start Docker Desktop" and install nothing (quickstart §5). If `up` fails, fix the cause in `internal/cluster/` or `test/isolation/tools.sh`, never by relaxing a check. Evidence in `specs/031-macos-multiarch/evidence/mac-up.md`
- [X] T023 [US2] On the Mac, run `./chalkeus forge down` and confirm every container, network and registry that `up` created is gone (`docker ps -a`, `docker network ls`, `kind get clusters`); add the result to `specs/031-macos-multiarch/evidence/mac-up.md`
  Done as (2026-11-25): after the fixture run; T01–T03 pass and AU07 passes. Only kind's own empty `kind` network remains, which kind keeps by design ([evidence/mac-up.md](evidence/mac-up.md)). T026 will need a new `up`.

## Phase 7: User Story 3 - Isolation is proven on Docker Desktop, not assumed (Priority: P1)

**Goal**: the unmodified isolation suite, the fixture run and a target run pass on the Mac with the audit at 11/11.

**Independent Test**: quickstart §6 and §7.

- [X] T024 [US3] On the Mac, run `bash test/isolation/run.sh` against a fresh `up` with no edits to the suite's checks or expectations (FR-008). Record the evidence under `specs/031-macos-multiarch/evidence/<run>/`. If a check fails, find the cause; if enforcement can't be proven on Docker Desktop, implement FR-009 (forge refuses on that host, naming the check) and stop here for an owner decision
  Done as (2026-11-24): run `20261124T183824Z-isolation`, 37 of 37 checks passed on Docker Desktop 4.94.0 (node `linux/arm64`, LinuxKit kernel), including Calico's default deny, the allowlisted egress, and teardown. Evidence: `evidence/20261124T183824Z-isolation/`.
- [X] T025 [US3] On the Mac, run the fixture walkthrough (operator guide section 6); the owner approves the modes and merges the PRs; then `down` and `./chalkeus forge audit` give 11/11 (SC-001). `versions.txt` shows `host_os: darwin …`, `host_arch: arm64`, `node_platform: linux/arm64`. Evidence under `specs/031-macos-multiarch/evidence/<run>/`
  Done as (2026-11-25): run [20261125T191440Z](evidence/20261125T191440Z/). Checks 32/32 (identical to T013), audit 11/11 after `down`, and host lines `darwin 27.0.1` / `arm64` / `linux/arm64`. An earlier run, [20261125T182055Z](evidence/20261125T182055Z/), failed at B/1 because Calico's CNI token had expired (T034, troubleshooting `#calico-token-expired`).
- [ ] T026 [US3] Owner: choose one prepared item of a target with a catalog toolchain. On the Mac, run `./chalkeus forge run --target <t> --item <NNN>`; the owner approves and merges; the audit after `down` passes (SC-002). Evidence under `specs/031-macos-multiarch/evidence/<run>/`

**Checkpoint**: the Mac has the same proof as Linux.

## Phase 8: User Story 5 - A macOS path in the operator guide (Priority: P2)

**Goal**: an operator with a fresh Mac reaches a completed fixture run using only the guide.

**Independent Test**: the docs check passes; quickstart §8.

- [ ] T027 [US5] In `docs/operator-guide.md`:
  - section 2: replace "Linux only" with the supported hosts and their validation status (R9, R12);
  - section 3: replace "macOS is not supported yet" with the Mac requirements (macOS version, Docker Desktop with 8 GB memory and 4+ CPUs under Settings → Resources, `gh` and `spec` 0.4.1 via Homebrew or direct download with checksum);
  - section 13: move the macOS row to "shipped in 031", and add rows for Intel Mac (by design) and arm64 Linux (untested).
- [ ] T028 [P] [US5] In `docs/troubleshooting.md`, add entries for "Docker isn't running" (macOS), Docker Desktop memory or CPUs too low (pointing to 030's capacity message), an image with no build for the node's platform, and a network plugin that doesn't enforce policy on the host
- [ ] T029 [P] [US5] Add a 031 section to `docs/release-notes.md`: multi-platform images, per-arch pins (old keys removed), spec-status 0.4.1, doctor's host area, macOS support status
- [ ] T030 [US5] Fresh-Mac walkthrough (SC-005): on a Mac user account without forge's prerequisites, follow only the operator guide's macOS path to a completed fixture run; record each step and any gap in `specs/031-macos-multiarch/evidence/fresh-mac.md`, and fix the guide for every gap found

## Phase 9: Polish and close

- [ ] T031 Privacy check: the new evidence files and `versions.txt` contain no token, key or model host (scan, counts only); `go vet ./... && go test ./...` and `pytest pilot/images/tests/` pass
- [ ] T032 Add `specs/031-macos-multiarch/evidence/README.md` listing every run (host, platform, result, audit) as in 026's evidence README
- [ ] T034 [US2] Found on the Mac (evidence/mac-up.md, 2026-11-25): Calico's 24-hour CNI token isn't renewed when Docker Desktop's VM sleeps, and new pods then fail with `connection is unauthorized`. Before staging each attempt (and in doctor's cluster area), read the token's `exp` from the node's `/etc/cni/net.d/calico-kubeconfig`. If it expires within the attempt's deadline plus a margin, restart `calico-node` and wait for a fresh token, or refuse with a message naming troubleshooting `#calico-token-expired`. Never print the token itself. Unit-test the expiry decision.
- [ ] T033 Run `speckit-roadmap-debrief` for 031 and propose the status change to the owner. Forge never records `verified` by itself

## Dependencies

- T001 → T002, T003 → T004 → US1.
- In US1: T005–T008 (parallel) → T009 → T010 → T011 → T012 (owner merges; images published).
- T012 → T013 (Linux regression) and → US2 and US3 (the Mac pulls the published arm64 images).
- US4: T014, T015 and T018 in parallel → T016 → T017; T020 → T021. T019 waits on 030.
- US2 (T022, T023) needs T012 and T017. US3 (T024–T026) needs US2. T025 and T026 also need
  T020, so the evidence carries the host lines.
- US5's docs (T027–T029) can start after US4; T030 needs US3. Phase 9 last.

## Parallel opportunities

- T002 and T003; T005, T006, T007 and T008 (different Dockerfiles).
- T014, T015 and T018 (different files); T028 and T029.
- US4's code (T014–T021) can proceed while T010–T012 run in CI and wait for the owner's merges.

## Implementation strategy

1. **MVP (US1):** pins, Dockerfiles and the two-platform workflow (T001–T012). That alone makes
   the images runnable on arm64, and the Linux regression run (T013) proves nothing changed for
   current users.
2. **Host visibility (US4):** doctor's host area, `up`'s refusals and the evidence lines, so the
   Mac proof records where it ran.
3. **Mac proof (US2, US3):** `up`, the isolation suite, the fixture run and a target run on the
   owner's Mac.
4. **Docs (US5),** the fresh-Mac walkthrough, then the debrief.
