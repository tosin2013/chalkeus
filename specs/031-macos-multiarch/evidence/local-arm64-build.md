# T009: local arm64 build on the Mac (2026-11-24)

A local check only. Published images still come from the GitHub workflow (C-09); these builds
were never pushed.

**Host:** Apple Silicon Mac, macOS 27.0.1, Docker Desktop 4.94.0 (Engine 29.8.2, VM
`linux/aarch64`, 14 CPUs, 7.7 GiB), Buildx v0.37.2.

**Inputs:** `pilot/pins.env` at this branch (per-architecture pins, spec-status 0.4.1), contexts
prepared as the workflow does (`pilot/images/<image>/` plus `common/frame.py`), arguments from
`pilot/images/build_args.py <image>`.

## Builds

`docker buildx build --platform linux/arm64 --load -t chk-<image>:arm64 <args> <ctx>`

| Image | Platform | Local image ID |
|---|---|---|
| chalkeus-worker | linux/arm64 | `sha256:4ca77cd9db72…` |
| chalkeus-tools | linux/arm64 | `sha256:668d06872875…` |
| chalkeus-egress | linux/arm64 | `sha256:5a49bbade31d…` |
| chalkeus-toolchain-python3.13 | linux/arm64 | `sha256:7193657209da…` |
| chalkeus-toolchain-node24 | linux/arm64 | `sha256:5f27e4364faf…` |
| chalkeus-toolchain-go | linux/arm64 | `sha256:1fd57bea585b…` |
| chalkeus-toolchain-jdk21-maven | linux/arm64 | `sha256:7b7faa75d431…` |

All seven builds passed every checksum: the upstream checksum file, the per-arch pin, and, for
the worker and tools, the extracted binary.

## Smoke test (arbitrary UID 1000123, read-only root)

`WORKER=chk-worker:arm64 TOOLS=chk-tools:arm64 EGRESS=chk-egress:arm64 TOOLCHAINS="…" pilot/images/smoke.sh`:
12 checks passed, `== smoke: all images pass as an arbitrary UID`:
- worker: result frame decoded, OpenCode 2.2.7;
- tools: `grade: pass` on the fixture solution;
- egress: proxy running, and it refused a non-allowlisted host;
- each toolchain reports its version (`Python 3.13.18`, `v24.22.3`, `go1.27.4 linux/arm64`,
  `Apache Maven 3.9.16`), and its offline sample installs online and tests offline with read-only
  dependencies.

The first run hit two host-side portability bugs in `smoke.sh` on macOS: GNU `sed -i`, and an
empty array under `set -u` in bash 3.2. Both are fixed in this change. The images were not
involved.

## Fail-closed checks

| Case | Result |
|---|---|
| tools with a wrong `SPEC_ASSET_SHA256_ARM64` | build fails: `spec_0.4.1_linux_arm64.tar.gz: FAILED` |
| worker with a wrong `OPENCODE_SHA256_ARM64` (extracted binary) | build fails: `opencode: FAILED` |
| tools with the legacy builder (no `TARGETARCH`) | build fails: `unsupported TARGETARCH ''` (before 031 it would have used amd64) |
