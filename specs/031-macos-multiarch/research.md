# Research: macOS, multi-arch images and kind on Docker Desktop

Findings gathered 2026-11-23 and 2026-11-24 on the maintainer's Apple Silicon Mac (macOS 27,
Docker Desktop, VM `linux/arm64`) and from the upstream releases.

## R1 — Which pinned images are already multi-platform

**Decision**: keep `DEBIAN_BASE`, `ALPINE_BASE`, `REGISTRY_IMAGE`, the isolation suite's busybox
and kind's default node image as pinned; no re-pin.

**Rationale**: `docker buildx imagetools inspect` on each pinned digest returns an OCI image
index that includes `linux/amd64` and `linux/arm64/v8` (Debian also 386, arm/v7, ppc64le;
Alpine and registry more). A digest of an index resolves to the right build on either platform,
so an arm64 build stage gets an arm64 base without any change. kind v0.27.0's default
`kindest/node:v1.32.2@sha256:f226…` is also an index.

**Alternatives considered**: per-platform base pins (more keys, no benefit while the index is
pinned).

## R2 — Host-side code that assumes Linux

**Decision**: change only the checksum command in `test/isolation/tools.sh` and `run.sh` (R7).

**Rationale**:
- Forge's Go code execs no Linux-only tool.
- The watcher's `/proc/<pid>/cmdline` read already falls back to a signal check when `/proc`
  is absent (macOS).
- `tools.sh` already maps `uname -s`/`uname -m` to kind's and kubectl's `darwin`/`arm64` asset
  names.
- The local registry is published on `127.0.0.1:5001`, which Docker Desktop forwards.
- `firstLine("uname", "-srm")` in `versions.txt` works on macOS.

**Alternatives considered**: none needed.

## R3 — Upstream arm64 inputs and their checksums

**Decision**: pin per architecture, verified 2026-11-24 against each release's own checksum file
and by hashing the downloaded files:

| Input | amd64 (unchanged where noted) | arm64 |
|---|---|---|
| OpenCode v2.2.7 asset | `opencode-linux-x64.tar.gz` `89a6cee2…` (unchanged) | `opencode-linux-arm64.tar.gz` `c1682057f0815cb2abdd6e00305f1146ebaf1c167f0335cdb74ddc80008120e7` |
| OpenCode binary | `589253d3…` (unchanged) | `c577783eacf7d1e66661fc7fd904b7646761497b687ff8f8f82ebe4f2f3ea425` |
| Python 3.13.18+20261026 asset | `…x86_64-unknown-linux-gnu-install_only.tar.gz` `126343c5…` (unchanged) | `…aarch64-unknown-linux-gnu-install_only.tar.gz` `b48451865bde9a39e2f3e71d7fdcbf1f32615f50f24d78dc50d04ede6501cf81` |
| spec-status v0.4.1 asset | `spec_0.4.1_linux_amd64.tar.gz` `2a852613bad97baf0934979c00b5a2238d5bd7ecf1d7ed336004b9b9aeb61d44` | `spec_0.4.1_linux_arm64.tar.gz` `2aa5b5e303578f4c9a13af2f52f125e2e6646a74959263343952c7343a4bc4ec` |
| spec-status binary | `5f6c1f28fd4855c59ae6016409529aa1e6704284a3d94fe1960a4a7b5c7c1084` | `d1a3f145afa4aa539f91cc9db4e8307d09525a4f47cc7c8672a60f8242189941` |

The amd64 OpenCode and Python values recomputed here match today's pins, which confirms the
method. Node 24, Go and Temurin already have both architectures pinned; Maven is
architecture-neutral (one SHA-512).

**Rationale**: C-09 needs a pin for every binary in every image; the worker and tools images
check two values each (the archive and the extracted binary), so both need per-arch pins.

**Alternatives considered**: trusting only the upstream checksum file (rejected: C-09 pins the
value in this repository, so a compromised release can't change it silently).

## R4 — spec-status v0.4.1

**Decision**: move `SPEC_*` from v0.4.0 to v0.4.1 (commit
`64ae0ee3a6a5cfa1e30c5a21547961ae2eea6e5b`).

**Rationale**: v0.4.0 shipped only `linux_amd64` and `darwin_arm64`; its release workflow built
exactly those two by design (spec-status spec 004). tosin2013/spec-status#82 asked for
`linux_arm64` and `darwin_amd64`. v0.4.1 (2026-11-24) has all four archives in its checksums
file, and its notes record a smoke test of each on its own platform, darwin/amd64 included. It
is a patch release: no CLI or JSON-schema change (schema 1.0), so the gate is unaffected. The
bump also applies to Linux amd64, so the Linux regression run (SC-006) covers it.

**Alternatives considered**: building spec-status from source at `SPEC_COMMIT` inside the tools
image (rejected: a second trust path and a Go toolchain in the build, for a problem the
upstream release now solves).

## R5 — How the lock pins a multi-platform image

**Decision**: `pilot/images.lock` keeps one line per image, `KEY=<ref>@sha256:<index digest>`.
The per-platform digests go in `image-digests.json` (the build's evidence, shown in the digest
PR's body).

**Rationale**:
- Kubernetes and Docker resolve an index digest to the node's platform, so pods keep pulling
  by one pinned digest.
- doctor's `DigestMatches` compares the text after `@`, so it works unchanged.
- The lock's format and its single writer (the digest PR) stay the same.

**Trade-off**: every image gets a new digest even where the amd64 contents are identical; the
Linux fixture run on the new digests is the proof that nothing changed (SC-006).

**Alternatives considered**: one lock line per platform (rejected: the pod spec would need to
pick a digest by node platform, which the runtime already does with an index).

## R6 — Building two platforms in CI

**Decision**: a `platform` matrix on the build job, using native runners `ubuntu-24.04` (amd64)
and `ubuntu-24.04-arm` (arm64):
- each leg builds every image with `--load`, runs `smoke.sh`, and on `main` pushes by digest
  only (`push-by-digest=true`, no tag), with provenance and an SBOM;
- a `merge` job then runs `docker buildx imagetools create` per image to make the index tagged
  `sha-<short>`, and writes `image-digests.json` with the index digest and both platform
  digests;
- the digest PR job is unchanged apart from reading the index digest.

**Rationale**:
- Native runners keep build time flat and avoid QEMU.
- Each platform is smoke-tested on real hardware of that platform (FR-004).
- If either leg fails, the merge job doesn't run, so nothing gets tagged or reaches the lock.
- Pushing by digest with no tag leaves orphan manifests only, never a half-published tag.

**Alternatives considered**:
- One job with `--platform linux/amd64,linux/arm64` under QEMU: slower, and the arm64 smoke
  test would run emulated.
- Building arm64 on the Mac: rejected by C-09; images are built only in GitHub.

## R7 — Checksum command on macOS

**Decision**: in `tools.sh` and `run.sh`, use `sha256sum` when present, otherwise
`shasum -a 256`; refuse if neither exists.

**Rationale**: the maintainer's macOS 27 has `/sbin/sha256sum`, but older macOS releases ship
only `shasum`. Both accept `-c -` with `<hash>  <file>` lines. If neither tool exists, the
script refuses rather than installing an unchecked file.

**Alternatives considered**: doing the check in Go inside `UpCluster` (larger change; the script
also runs on its own for the isolation suite).

## R8 — The memory check and 030

**Decision**: the "too little memory for Docker Desktop" fail is 030's capacity check (FR-012).
031's host rows show Docker Desktop's CPUs and memory (from `docker info`) as information, so
the operator sees the numbers even before 030 lands.

**Rationale**:
- Docker Desktop's VM is the local cluster's capacity; 030 already judges a local cluster
  against the container runtime's resources.
- A second check would give two answers to one question.
- If 031's Mac proof runs before 030 is implemented, the documented 8 GB setting carries it.
  The tasks mark the memory-fail task as depending on 030.

**Alternatives considered**: a fixed 8 GB check in 031 (rejected: it duplicates 030 and ignores
a raised budget).

## R9 — Recognising the host and runtime

**Decision**:

| Host (`runtime.GOOS/GOARCH` of the binary) | doctor |
|---|---|
| darwin/arm64 | pass (validated) |
| linux/amd64 | pass (validated) |
| darwin/amd64 | pass with "supported by design; not maintainer-validated" |
| linux/arm64 | warn "untested" |
| anything else | fail `unsupported host` |

- **Runtime:** `docker info` `OperatingSystem` and `docker version` `Server.Platform.Name`;
  "Docker Desktop" passes on macOS, other runtimes warn "only Docker Desktop is validated on
  macOS".
- **Stopped runtime:** `docker info` fails. doctor fails with "start Docker Desktop" on macOS
  ("start the Docker daemon" on Linux), and `up` refuses before installing anything.
- **Image platforms:** for each locked image, `docker buildx imagetools inspect --raw` lists
  the index's platforms.
  - An image without the node's platform fails and names the image.
  - A registry that can't be read warns.
  - A local build (`--local-build`) builds for the host's platform and skips this row with a
    note, as the images row already does.

**Rationale**: the Go binary's own GOOS/GOARCH is the host platform without exec; Docker's two
fields distinguish Docker Desktop from OrbStack, Colima and Engine.

**Alternatives considered**: refusing non-Docker-Desktop runtimes (rejected in the spec's
Assumptions: the isolation suite and the audit decide trust, not the runtime's brand).

## R10 — Evidence fields

**Decision**: `versions.txt` keeps every current line and adds `host_os`, `host_arch`,
`runtime` (name, version, VM CPUs and memory), `node_platform`, and `image_<k>_platform` per
image.

**Rationale**:
- `versions.txt` already records the host (`uname -srm`), kind and the node image, and nothing
  in the audit parses it.
- Adding lines keeps Linux evidence comparable: only new lines differ (SC-006, SC-007).
- The `opencode` and `spec` lines show the pinned binary checksum for the node's platform.

**Alternatives considered**: a new `host.json` (rejected: one more file for the same reader).

## R11 — Proving isolation on Docker Desktop

**Decision**: run the unmodified M0 isolation suite (`test/isolation/run.sh`), then the fixture
walkthrough and one target run, on the maintainer's Mac. Evidence goes in
`specs/031-macos-multiarch/evidence/`, and the audit runs after `down`.

**Rationale**: Docker Desktop's VM kernel (LinuxKit) and its networking differ from Ubuntu's.
Calico's enforcement and the egress proxies have to be shown working there, not assumed. The
suite's expectations are not edited for macOS (FR-008). If a check fails, the fix goes into
`up` or the docs, never into the check.

**Risk**: Calico on Docker Desktop may need a kernel feature LinuxKit lacks. Calico's default
iptables data plane is expected to work there, but that is unproven until the suite runs. If
enforcement can't be proven, FR-009 applies: forge refuses on that host, and the docs say so.

## R12 — Intel Mac and arm64 Linux

**Decision**: neither gets a maintainer evidence run in this item.
- **Intel Mac:** supported by design. It uses the amd64 images (identical to Linux's) and a
  darwin/amd64 binary, which Chalkeus CI already builds; spec-status v0.4.1 provides
  `darwin_amd64` (smoke-tested upstream).
- **arm64 Linux:** allowed with a warning.
- The docs list both with their validation status, matching 005's community-validation model.

**Rationale**: the spec's Assumptions; the maintainer has no Intel Mac or arm64 Linux host.
