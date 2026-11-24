# Contract: doctor's host area, and `up`'s early refusals

doctor prints a new **host** area before the cluster area. Statuses are doctor's existing
`pass`, `warn` and `fail`; a fail makes doctor exit non-zero, as today.

| Row | Pass | Warn | Fail |
|---|---|---|---|
| `host` | `darwin/arm64 (macOS 27.0.1), validated`; `linux/amd64, validated`; `darwin/amd64 (macOS …), supported by design, not maintainer-validated` | `linux/arm64: untested; the images run, but no maintainer evidence exists` | `unsupported host <os>/<arch>; supported: darwin/arm64, darwin/amd64, linux/amd64 (linux/arm64 untested)` |
| `container runtime` | `Docker Desktop 4.x, 8 CPUs, 15.6 GiB` (macOS); `Docker Engine 28.x, …` (Linux) | macOS runtime that isn't Docker Desktop: `<name> <version>: only Docker Desktop is validated on macOS; the isolation suite and audit still decide` | daemon unreachable: `Docker isn't running: start Docker Desktop` (macOS) / `start the Docker daemon` (Linux) |
| `node platform` (after `up`) | `linux/arm64` | — | `node platform <p> isn't linux/amd64 or linux/arm64` |
| `image platforms` (after `up`, pinned images) | `all 7 images have linux/arm64` | registry unreadable: `couldn't check image platforms: <error>`; local build: `local build: images built for this host` | `<image> has no build for linux/arm64 (pilot/images.lock may predate 031); pull main and run ./chalkeus forge up` |

Memory is not a host-area fail. 030's capacity check owns it (FR-012, R8). On macOS, the
capacity message names Docker Desktop's Settings → Resources.

## `forge up`

Before installing tools, `UpCluster` runs the `host` and `container runtime` checks and refuses
on a fail, using the same message. Warnings print and continue.

## Exit codes and output

Unchanged. The new rows appear in doctor's text report under the area `host`.
