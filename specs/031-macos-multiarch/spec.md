# Feature Specification: macOS, multi-arch images and kind on Docker Desktop

**Feature Branch**: `spec-031-macos-multiarch`

**Created**: 2026-11-24

**Status**: Draft

**Input**: Roadmap entry 031 (added 2026-11-23, roadmap 1.35.0), from issue #143
(https://github.com/tosin2013/chalkeus/issues/143). Make forge run on macOS (Apple Silicon and
Intel) with Docker Desktop, with every guarantee and the full audit, so the laptop release
(013), which promises macOS binaries, has a cluster side to drive.

Governed by C-03 (isolation and credentials), C-06 (enforcing CNI), C-07 (egress through
per-role allowlist proxies), C-09 (images built in GitHub, pinned by digest) and C-11 (two
modes, one set of guarantees).

## Context

Today the cluster side runs only on Ubuntu 24.04 amd64. The operator guide says so in three
places (section 2 "Linux only", section 3 "macOS is not supported yet", and section 13's
"not yet scheduled" row). What is amd64-only today:

| Input | Where | arm64 today |
|---|---|---|
| Worker image (OpenCode) | `OPENCODE_ASSET` is the linux-x64 asset, one checksum | no |
| Tools image (spec-status) | `SPEC_ASSET` is the linux_amd64 asset, one checksum | no |
| Egress image | built for one platform | no |
| Python 3.13 toolchain | `PYTHON313_ASSET` is the x86_64 asset, one checksum | no |
| Node 24, Go, JDK 21 + Maven toolchains | amd64 and arm64 checksums already pinned | checksums yes; images single-platform |
| Image lock | one digest per image | needs a digest that covers both platforms |

Already platform-aware: the pinned kind and kubectl install picks the host's OS and
architecture and checks each download against its published checksum. The Go binary already
builds for darwin amd64 and arm64 in CI. Nothing in forge's own code is platform-specific.

Related item: 030 (resource budgets and capacity checks) adds doctor's capacity check for a
local cluster's container runtime. On a Mac, that runtime is Docker Desktop's Linux VM, so 030's
check is where "Docker Desktop has too little memory" is reported. This item makes sure that
check works on macOS; it does not add a second one.

Terms:
- **Platform:** an image's OS and CPU architecture, here `linux/amd64` or `linux/arm64`.
- **Multi-platform image:** one image reference whose digest covers a build per platform; the
  container runtime pulls the build that matches its own platform.
- **Host:** the machine the operator runs `chalkeus forge` on.
- **Upstream input:** a binary or archive downloaded from another project's release and put in
  an image (OpenCode, spec-status, Python, Node, Go, Temurin, Maven).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Every image runs on arm64 and amd64 (Priority: P1)

The maintainer publishes every forge and toolchain image for both platforms, under one pinned
digest each, with every upstream input checked against a pinned checksum for its architecture.
A Linux amd64 operator sees no change; an arm64 cluster pulls arm64 builds.

**Why this priority**: nothing else in this item works without it. Apple Silicon can't run
amd64-only images except under slow emulation, and the isolation proof can't rely on emulation.

**Independent Test**: on an amd64 machine and on an arm64 machine, pull each image by the digest
in the image lock and confirm the pulled build's platform matches the machine. Change one
pinned checksum for one architecture and confirm that architecture's build fails and nothing is
published.

**Acceptance Scenarios**:

1. **Given** the image build runs, **When** it finishes, **Then** every image in the image lock
   (worker, tools, egress and each toolchain) has a build for `linux/amd64` and `linux/arm64`
   under one digest.
2. **Given** an upstream input whose downloaded file doesn't match the pinned checksum for that
   architecture, **When** the build runs, **Then** that build fails, no image is published, and
   there is no fallback to another asset or architecture.
3. **Given** an upstream input that has no pinned checksum for one architecture, **When** the
   build runs, **Then** it fails and names the missing pin.
4. **Given** a Linux amd64 operator on the new images, **When** they run the fixture walkthrough,
   **Then** the result and the audit are the same as before (11/11).
5. **Given** a published image, **When** a reviewer reads the build's evidence, **Then** it lists
   the digest for each platform alongside the image's pinned digest.

---

### User Story 2 - Bring up the local cluster on a Mac (Priority: P1)

An operator on an Apple Silicon Mac with Docker Desktop runs `forge up`. It installs the pinned
kind and kubectl for macOS, creates the cluster in Docker Desktop's Linux VM, applies the
enforcing network plugin only after its manifest's checksum matches, and waits for the nodes,
exactly as on Linux.

**Why this priority**: the cluster is where every guarantee is enforced; without `up` on a Mac
there's no laptop release for Mac operators.

**Independent Test**: on a fresh Apple Silicon Mac with Docker Desktop and the documented
settings, `forge up` succeeds; `forge down` removes everything it created.

**Acceptance Scenarios**:

1. **Given** a Mac with Docker Desktop running and the documented resources, **When** the
   operator runs `forge up`, **Then** the pinned kind and kubectl for the host's OS and
   architecture are installed after their checksums match, the cluster is created, and the
   network plugin is applied only after its manifest's checksum matches.
2. **Given** a download whose checksum doesn't match, **When** `up` runs, **Then** it stops and
   names the file; nothing unverified is installed.
3. **Given** Docker Desktop is not running, **When** the operator runs `up` or doctor, **Then** it
   refuses with a message that says to start Docker Desktop.
4. **Given** a cluster made by `up` on a Mac, **When** the operator runs `forge down`, **Then**
   everything `up` created is removed, as on Linux.

---

### User Story 3 - Isolation is proven on Docker Desktop, not assumed (Priority: P1)

The isolation suite (the M0 checks from 001) and the fixture run (002) pass on an Apple Silicon
Mac with Docker Desktop: the network plugin enforces network policy, workers reach only their
allowlisted hosts through the egress proxies, and the audit reaches 11/11.

**Why this priority**: Docker Desktop's VM kernel and networking differ from Ubuntu's. A Mac
run that is not proven isolated breaks the project's main promise.

**Independent Test**: run the isolation suite on the Mac; every check passes. Run the fixture
walkthrough; it completes, the owner merges its PR, and after `down` the audit is 11/11.

**Acceptance Scenarios**:

1. **Given** a Mac cluster from `up`, **When** the isolation suite runs, **Then** every check
   passes, with the same checks and expectations as on Linux.
2. **Given** a Mac cluster, **When** the fixture walkthrough runs, **Then** it completes and the
   audit after `down` is 11/11.
3. **Given** a target with at least one toolchain, **When** a target run (014) runs on the Mac,
   **Then** it passes with the same gate, grading and audit as on Linux.
4. **Given** an environment where an isolation check can't be proven (for example, network
   policy isn't enforced), **When** the suite or a run checks it, **Then** forge refuses to run
   and says which check failed; no check is skipped or relaxed for macOS.

---

### User Story 4 - Know which host and platform a run used (Priority: P2)

doctor reports the host's OS and architecture and the container runtime. Every run's evidence
records the host OS, the host architecture and each image's platform, so a reviewer can tell a
Mac run from a Linux run.

**Why this priority**: an auditor comparing runs needs to know where each ran; a support
question starts with "what are you running on".

**Independent Test**: run doctor on the Mac and on Linux; each shows its host OS, architecture
and runtime. Compare a Mac run's evidence with a Linux run's; the host and image platforms
differ and are stated in both.

**Acceptance Scenarios**:

1. **Given** any supported host, **When** doctor runs, **Then** it reports the host OS, the host
   architecture and the container runtime with its version.
2. **Given** Docker Desktop with less memory than forge needs, **When** doctor runs, **Then** it
   fails with a clear message to raise Docker Desktop's memory, through 030's capacity check.
3. **Given** a finished run or draft check, **When** its evidence is read, **Then** it records
   the host OS, host architecture and the platform of each image used.
4. **Given** a host OS or architecture forge doesn't support, **When** doctor or `up` runs,
   **Then** it refuses with a message naming the host and the supported hosts.

---

### User Story 5 - A macOS path in the operator guide (Priority: P2)

An operator with a fresh Mac follows the operator guide's macOS path, from installing tools to a
completed fixture run, without needing anything the guide doesn't mention.

**Why this priority**: SC-005's "works from a fresh Mac" is the release's promise to Mac users;
the guide is how they reach it.

**Independent Test**: on a Mac without forge's prerequisites, follow only the guide; the fixture
run completes with the audit at 11/11.

**Acceptance Scenarios**:

1. **Given** the operator guide, **When** a Mac operator reads "Before you start", **Then** it
   lists the Mac requirements: macOS version, Docker Desktop with its memory and CPU settings,
   and how to install `gh` and `spec` (Homebrew or direct download).
2. **Given** the operator guide's "What will change" table, **When** the item ships, **Then** the
   macOS row moves from "not yet scheduled" to the release, and the "Linux only" notes are
   updated.
3. **Given** the troubleshooting guide, **When** a Mac operator hits a Docker Desktop problem
   (not running, too little memory, a network plugin that doesn't enforce), **Then** there's an
   entry for it.

---

### Edge Cases

- An Intel Mac: it uses the amd64 images (the same as Linux) and the darwin amd64 binary; it is
  supported by design but has no maintainer evidence (see Assumptions).
- An arm64 Linux host: the arm64 images run there too, but this item proves only macOS arm64 and
  Linux amd64; doctor doesn't refuse arm64 Linux, and the docs mark it untested.
- Another Docker-compatible runtime on the Mac (for example OrbStack or Colima): out of scope;
  doctor names the runtime and warns that only Docker Desktop is validated, and the isolation
  suite and audit still decide whether a run is trustworthy.
- Rosetta or emulation enabled in Docker Desktop: images are pulled for the VM's native
  platform; an image pulled for the other platform is a fail in the evidence check, not a
  silent slowdown.
- The image lock still holding single-platform digests (an older checkout): doctor reports the
  image's platforms and fails on a Mac when an image has no arm64 build.
- A toolchain image an operator supplies (022): it must have a build for the cluster's
  platform; doctor says so if it doesn't.
- A test target whose own dependencies have no arm64 build (for example a native package):
  the grader fails like any build failure; the docs say how to recognise it.
- A checksum tool missing or behaving differently on macOS: `up` refuses rather than installing
  an unchecked file.
- Docker Desktop's VM restarted between `up` and a run: doctor notices the cluster is gone and
  says to run `up` again, as on Linux.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Every image in the image lock (worker, tools, egress and every toolchain) MUST be
  published for `linux/amd64` and `linux/arm64` under one digest.
- **FR-002**: Every upstream input MUST have a pinned checksum per architecture. A build MUST
  fail when a download doesn't match its pin or a pin is missing, with no fallback (C-09).
- **FR-003**: The image lock MUST record each image's multi-platform digest; the build evidence
  MUST record the per-platform digests under it.
- **FR-004**: Images MUST be built in GitHub (C-09), each platform's build MUST run the same
  smoke checks, and a failure on either platform MUST publish neither.
- **FR-005**: For Linux amd64 operators, the images MUST behave as before: the fixture run's
  result and its audit (11/11) MUST be unchanged.
- **FR-006**: `forge up` MUST work on macOS (arm64 and amd64) with Docker Desktop: it installs
  the pinned kind and kubectl for the host after their checksums match, creates the cluster, and
  applies the network plugin only after its manifest's checksum matches, as on Linux.
- **FR-007**: `forge down` MUST remove on macOS everything `up` created, as on Linux.
- **FR-008**: The isolation suite (001) MUST pass on macOS arm64 with Docker Desktop with the
  same checks and expectations as on Linux; no check may be skipped, weakened or marked
  not-applicable for macOS.
- **FR-009**: When an isolation guarantee can't be proven on the host, forge MUST refuse to run
  and name the failed check (C-02, C-03, C-06, C-07).
- **FR-010**: doctor MUST report the host OS, host architecture and container runtime with its
  version, and refuse an unsupported host with a message naming the supported hosts.
- **FR-011**: doctor MUST refuse with a message to start Docker Desktop when it isn't running.
- **FR-012**: The local cluster's memory and CPU check MUST be 030's capacity check, and on macOS
  its message MUST name Docker Desktop's resource settings.
- **FR-013**: doctor MUST fail when an image needed by the run has no build for the cluster's
  platform, and name the image.
- **FR-014**: Every run's and draft check's evidence MUST record the host OS, host architecture,
  and the platform of each image used.
- **FR-015**: The operator guide MUST have a macOS path (requirements, Docker Desktop settings,
  tool installs) and update its "Linux only" notes and the section 13 macOS row; the
  troubleshooting guide MUST cover Docker Desktop not running, too little memory and a network
  plugin that doesn't enforce.

### Key Entities

- **Image lock entry:** an image name and its multi-platform digest, with per-platform digests
  in the build evidence.
- **Upstream pin:** an upstream input's version and one checksum per architecture.
- **Host report:** host OS, host architecture, container runtime and its version (doctor and
  evidence).
- **Image platform record:** for each image a run used, the platform that was pulled.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: On an Apple Silicon Mac with Docker Desktop, `forge up` succeeds, the isolation
  suite passes every check, and the fixture run completes with the audit at 11/11.
- **SC-002**: A target run with at least one toolchain passes on the same Mac with the same
  audit as on Linux.
- **SC-003**: 100% of images in the image lock pull by their pinned digest on both amd64 and
  arm64, and each pulls the build for its machine's platform.
- **SC-004**: 100% of upstream inputs in every image are checked against a pinned checksum for
  their architecture at build time; one wrong pin fails the build.
- **SC-005**: An operator with a fresh Mac reaches a completed fixture run following only the
  operator guide.
- **SC-006**: The Linux amd64 fixture evidence after this item matches the evidence before it,
  apart from the new host and platform fields.
- **SC-007**: A reviewer can tell from any run's evidence alone which host OS, architecture and
  image platforms it used.

## Assumptions

- The maintainer's machine (Apple Silicon, Docker Desktop, linux/arm64 VM) is where the macOS
  evidence is produced; an Intel Mac is supported by design (amd64 images, darwin amd64 binary)
  without maintainer evidence, and the docs say so.
- Docker Desktop is the only validated container runtime on macOS; Podman is out of scope
  (roadmap 031), and other Docker-compatible runtimes are warned about, not refused.
- GitHub provides native arm64 runners, so arm64 builds don't need emulation; if they had to,
  emulated builds are acceptable for building but not for the isolation proof.
- forge's memory need on a Mac is the same as on Linux (8 GB for Docker Desktop's VM), judged by
  030's capacity check.
- Windows (013 "where the pilot supports it"), Podman and other Docker Desktop alternatives, and
  any relaxation of isolation on macOS are out of scope (roadmap 031).
- The binary's darwin builds already exist in CI; this item doesn't change how the binary is
  released (that's 013).
