# Contract: two-platform image workflow

Amends specs/003-m2-kelos-backend/contracts/image-workflow.md. Triggers, the no-rebuild-loop
rules (F1), registry parameterisation and the digest PR's single-file rule are unchanged.

## Jobs

| Job | Runs on | Does | On pull requests |
|---|---|---|---|
| `build` (matrix `platform: [amd64, arm64]`) | `ubuntu-24.04` / `ubuntu-24.04-arm` | Builds every image with `--load` for its own platform; runs `smoke.sh`; on `main`, pushes each image by digest (`push-by-digest=true,name-canonical=true`, no tag) with provenance `mode=max` and an SBOM; uploads `digests-<platform>.json` (`{image: sha256:…}`) | build and smoke only |
| `merge` (needs both `build` legs) | `ubuntu-24.04` | For each image, `docker buildx imagetools create -t <ref>:sha-<short> <ref>@<amd64> <ref>@<arm64>`; reads the index digest with `imagetools inspect`; checks that the index lists exactly `linux/amd64` and `linux/arm64`; writes `image-digests.json` (data-model) | skipped |
| `digest-pr` (needs `merge`) | `ubuntu-24.04` | Unchanged, except `images` now holds index digests | skipped |

## Guarantees

- **Both or neither:** if either build leg fails, `merge` doesn't run, so no tag is created and
  the lock is not proposed.
- **Native smoke tests:** each platform's images are smoke-tested on that platform as an
  arbitrary UID, using the existing `smoke.sh` unchanged.
- **Pins checked inside the build:** a pin mismatch fails that leg's `docker buildx build`, so
  nothing reaches the push step (C-09, FR-002).
- **Platform check:** `merge` fails if an index lists any platform other than the two expected,
  or is missing either.
- **Permissions:** `build` keeps `contents: read`, `packages: write`; `merge` gets the same;
  `digest-pr` is unchanged. No other secret is available.
