# Contract: per-architecture pins and build args

## `pilot/pins.env` keys

Unchanged keys are not listed (Node 24, Go and Temurin already follow this pattern; Maven and
the bases are architecture-neutral).

| Key | Value (2026-11-24, R3/R4) |
|---|---|
| `OPENCODE_VERSION` | `2.2.7` (unchanged) |
| `OPENCODE_RELEASE` | `anomalyco/opencode@v2.2.7` (unchanged) |
| `OPENCODE_ASSET_AMD64` | `opencode-linux-x64.tar.gz` |
| `OPENCODE_ASSET_ARM64` | `opencode-linux-arm64.tar.gz` |
| `OPENCODE_ASSET_SHA256_AMD64` | `89a6cee227b54bc011179c528a5e63176032462b1a5e21723b616801cddc2ca7` |
| `OPENCODE_ASSET_SHA256_ARM64` | `c1682057f0815cb2abdd6e00305f1146ebaf1c167f0335cdb74ddc80008120e7` |
| `OPENCODE_SHA256_AMD64` | `589253d36810f21217edd51545cd435b328cdc3cf4f451fbe0cce6f88afcf5e1` |
| `OPENCODE_SHA256_ARM64` | `c577783eacf7d1e66661fc7fd904b7646761497b687ff8f8f82ebe4f2f3ea425` |
| `SPEC_VERSION` | `0.4.1` |
| `SPEC_COMMIT` | `64ae0ee3a6a5cfa1e30c5a21547961ae2eea6e5b` |
| `SPEC_RELEASE` | `tosin2013/spec-status@v0.4.1` |
| `SPEC_ASSET_AMD64` | `spec_0.4.1_linux_amd64.tar.gz` |
| `SPEC_ASSET_ARM64` | `spec_0.4.1_linux_arm64.tar.gz` |
| `SPEC_ASSET_SHA256_AMD64` | `2a852613bad97baf0934979c00b5a2238d5bd7ecf1d7ed336004b9b9aeb61d44` |
| `SPEC_ASSET_SHA256_ARM64` | `2aa5b5e303578f4c9a13af2f52f125e2e6646a74959263343952c7343a4bc4ec` |
| `SPEC_SHA256_AMD64` | `5f6c1f28fd4855c59ae6016409529aa1e6704284a3d94fe1960a4a7b5c7c1084` |
| `SPEC_SHA256_ARM64` | `d1a3f145afa4aa539f91cc9db4e8307d09525a4f47cc7c8672a60f8242189941` |
| `PYTHON313_RELEASE` | `astral-sh/python-build-standalone@20261026` (unchanged) |
| `PYTHON313_ASSET_AMD64` | `cpython-3.13.18+20261026-x86_64-unknown-linux-gnu-install_only.tar.gz` |
| `PYTHON313_ASSET_ARM64` | `cpython-3.13.18+20261026-aarch64-unknown-linux-gnu-install_only.tar.gz` |
| `PYTHON313_ASSET_SHA256_AMD64` | `126343c54daff4068efc1ac269518d0cc202d57857794948103dd1454e8881be` |
| `PYTHON313_ASSET_SHA256_ARM64` | `b48451865bde9a39e2f3e71d7fdcbf1f32615f50f24d78dc50d04ede6501cf81` |

Removed (hard switch): `OPENCODE_ASSET`, `OPENCODE_ASSET_SHA256`, `OPENCODE_SHA256`,
`SPEC_ASSET`, `SPEC_ASSET_SHA256`, `SPEC_SHA256`, `PYTHON313_ASSET`, `PYTHON313_ASSET_SHA256`.
Every reader is updated in the same change: `build_args.py`, `internal/images/images.go`,
`internal/evidence/evidence.go`, the workflow's `image-digests.json` step, and the docs that
name them (`docs/harnesses.md`).

## Build args

`build_args.py <image>` and `images.BuildArgs(image)` emit identical maps (a Go test compares
them against the same `pins.env`). For each per-arch field they emit both `_AMD64` and
`_ARM64`. If a key is missing, they exit non-zero with `pins.env: missing <KEY>`.

## Dockerfile selection

Each of `worker`, `tools` and `toolchain-python3.13`:

```dockerfile
ARG TARGETARCH
RUN set -eu; case "${TARGETARCH:-}" in \
      amd64) asset="${X_ASSET_AMD64}"; sum="${X_ASSET_SHA256_AMD64}"; bin="${X_SHA256_AMD64}";; \
      arm64) asset="${X_ASSET_ARM64}"; sum="${X_ASSET_SHA256_ARM64}"; bin="${X_SHA256_ARM64}";; \
      *) echo "unsupported TARGETARCH '${TARGETARCH:-}'"; exit 1;; esac; \
    # download, check against the upstream checksum file AND the pin; extract; check the binary
```

- An empty `TARGETARCH` fails; it does not default to amd64.
- Both checks stay, as today: first the upstream checksum file, then the pin.
- The Node, Go and JDK toolchains' existing `${TARGETARCH:-amd64}` default is tightened to the
  same fail-on-empty rule.
