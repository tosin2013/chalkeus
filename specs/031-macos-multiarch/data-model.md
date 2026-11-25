# Data model: macOS, multi-arch images and kind on Docker Desktop

No new store. Four existing files change shape, and doctor gains one area.

## Upstream pin (`pilot/pins.env`)

For each upstream input: a version or release, and per architecture the asset name and its
checksum, plus the extracted binary's checksum where the image checks one. Exact keys are in
[contracts/pins.md](contracts/pins.md).

Validation:
- Every input used by an image has both an `_AMD64` and an `_ARM64` value for each per-arch
  field; a missing one fails `build_args.py` (and its Go twin) naming the key.
- Checksums are 64 lowercase hex characters (SHA-256); Maven keeps its SHA-512.
- Old single-arch keys (`OPENCODE_ASSET`, `OPENCODE_ASSET_SHA256`, `OPENCODE_SHA256`,
  `SPEC_ASSET`, `SPEC_ASSET_SHA256`, `SPEC_SHA256`, `PYTHON313_ASSET`,
  `PYTHON313_ASSET_SHA256`) are removed in the same change, a hard switch as in 027, so nothing
  can read a stale amd64-only value.

## Image lock entry (`pilot/images.lock`)

Unchanged format: `KEY=<registry>/<namespace>/chalkeus-<image>@sha256:<digest>`. The digest is
now the image index's digest, covering `linux/amd64` and `linux/arm64`. It is written only by
the digest PR (C-09).

## Image build record (`image-digests.json`, CI artifact and digest PR body)

```json
{
  "commit": "<sha>",
  "built_at": "<UTC>",
  "registry": "ghcr.io/tosin2013",
  "images": { "worker": "ghcr.io/tosin2013/chalkeus-worker@sha256:<index>" },
  "platforms": {
    "worker": { "linux/amd64": "sha256:<manifest>", "linux/arm64": "sha256:<manifest>" }
  },
  "attestations": { "worker": { "provenance": "mode=max", "sbom": "spdx" } },
  "inputs": {
    "opencode": { "release": "…", "amd64": { "asset": "…", "sha256": "…" }, "arm64": { … } },
    "spec": { "release": "…", "amd64": { … }, "arm64": { … } },
    "python313": { "release": "…", "amd64": { … }, "arm64": { … } },
    "bases": { "debian": "…", "alpine": "…" }
  }
}
```

`images` keeps its current meaning (what the lock gets); `platforms` is new.

## Host report (doctor "host" area)

| Field | Source | Example |
|---|---|---|
| host OS | the binary's GOOS, plus `sw_vers -productVersion` on macOS | `darwin 27.0.1` |
| host architecture | the binary's GOARCH | `arm64` |
| support status | R9 table | `validated` |
| runtime | `docker info` `OperatingSystem`, `docker version` `Server.Platform.Name` and version | `Docker Desktop 4.x` |
| runtime resources | `docker info` `NCPU`, `MemTotal` | `8 CPUs, 15.6 GiB` |
| node platform | `kubectl get nodes -o jsonpath` `.status.nodeInfo.operatingSystem/architecture` | `linux/arm64` |
| image platforms | `docker buildx imagetools inspect --raw <locked ref>` | `worker: linux/amd64, linux/arm64` |

Rows and statuses are in [contracts/doctor.md](contracts/doctor.md).

## Version lines (`versions.txt` in every run's evidence)

New lines, appended after today's `host:` line; existing lines are unchanged:

```text
host_os: darwin 27.0.1
host_arch: arm64
runtime: Docker Desktop 4.x (Docker Engine 28.x), 8 CPUs, 15.6 GiB
node_platform: linux/arm64
image_egress_platform: linux/arm64
image_tools_platform: linux/arm64
image_worker_platform: linux/arm64
image_toolchain-<name>_platform: linux/arm64   # for each toolchain the run used
```

The `spec:` and `opencode:` lines print the pinned version and binary checksum for
`node_platform`'s architecture. A last line, `host_spec: <spec --version> sha256=<checksum>`,
records the spec-status binary the gate actually ran on the host, hashed from the file itself
(added after T013's first run, 20261125T162546Z, whose `spec:` line paired the host's 0.3.0
with the pinned 0.4.1 checksum). On Linux amd64 these equal today's values except for the spec-status bump to
0.4.1 (R4).
