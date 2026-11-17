# Contract: toolchain image

A toolchain image gives the worker, prefetch and grader pods a language's tools. The same
contract covers the images specfoundry builds and the images operators build themselves (owner
decision 2026-11-17). The worker image and the grader/publisher **tools** image are always
specfoundry's; a toolchain image only supplies files.

## Layout

```text
/opt/toolchain/
├── toolchain.json      # required: describes the toolchain (below)
├── bin/                # optional: put on PATH first
└── …                   # anything else the toolchain needs (a JDK, Node.js, a Python venv, Go)
```

- **How it reaches the pods:** an init container copies `/opt/toolchain/` into an `emptyDir`
  mounted at `/opt/toolchain` in the worker, prefetch and grader containers. It runs `cp -a
  /opt/toolchain/. /dest/`, so the image must contain `cp`, or a static copy helper at
  `/opt/toolchain/.copy`.
- **Ownership:** files must be readable by group 0 (`chgrp -R 0 /opt/toolchain && chmod -R g=u
  /opt/toolchain`). The pods run with an arbitrary UID and group 0, as on OpenShift's
  restricted SCC.
- **Linking:** binaries must run from `/opt/toolchain` in the worker's and grader's base image
  (Debian 12 today; UBI on OpenShift). Statically linked or relocatable distributions work best:
  official Temurin, Node.js and Go tarballs, and `python-build-standalone`.

## `toolchain.json`

```json
{
  "name": "node24",
  "version_command": ["node", "--version"],
  "path": ["/opt/toolchain/node/bin"],
  "env": {"NPM_CONFIG_CACHE": "/home/worker/.npm"},
  "proxy": "env"
}
```

| Field | Rule |
|---|---|
| `name` | Recorded in evidence; it is free text for an operator-supplied image |
| `version_command` | Runs in the probe and must exit 0; its first output line is recorded |
| `path` | Absolute paths under `/opt/toolchain`, put first on `PATH` |
| `env` | Extra environment. It may not set `HTTPS_PROXY`, `HTTP_PROXY`, `NO_PROXY` or any `*TOKEN*`/`*KEY*`/`*SECRET*` name (the pilot sets the proxies) |
| `proxy` | `env`: the package manager honours `HTTPS_PROXY` (npm, pip, Go, cargo, …). `maven-settings`: the pilot passes a generated `settings.xml` (Maven, Gradle's Maven repositories). |

## Where a target's toolchain comes from

| Source | Profile | Trust |
|---|---|---|
| **specfoundry catalog** (`pilot/images/toolchains.json`): `jdk21-maven`, `node24`, `python3.13`, `go` | `PILOT_TOOLCHAIN=<name>` | Built by this repository's image workflow from checksum-verified upstream releases, pinned in `pilot/images.lock` (C-09) |
| **Operator-supplied** | `PILOT_TOOLCHAIN_IMAGE=registry/name@sha256:<64 hex>` (and optionally `PILOT_TOOLCHAIN=<label>`) | Chosen and pinned by the operator, outside anything the agent can change. Shown at the mode approval and recorded as `operator-supplied`. A GitHub build attestation is verified and recorded when present, but not required. |
| **Requested** | the [toolchain image request form](https://github.com/tosin2013/specfoundry/issues/new?template=toolchain-image.yml) | Becomes a catalog toolchain |

An operator-supplied toolchain still runs inside the grader: it executes the project's tests,
so a tampered image could fake a result. Keeping that image trustworthy is the operator's
supply-chain responsibility, as for any base image they choose. The agent can't influence it.

## Probe (every run, after the mode approval, before prefetch)

A short Job in the run's grading namespace, with no network, runs the toolchain's init copy,
then checks four things:

1. `toolchain.json` parses and follows the rules above.
2. The files are readable under an arbitrary UID with group 0.
3. `version_command` exits 0 within 60 s.
4. Nothing under `/opt/toolchain` is setuid or setgid.

A failure refuses the run with `TOOLCHAIN_CONTRACT` and names the failed check. The result goes
in `toolchain.json` evidence: source, image digest, name, version line, attestation (verified,
absent or failed).

## Building one: templates

`pilot/images/toolchain-template/` holds the template:
- `Dockerfile.debian`: for kind, minikube and plain Kubernetes;
- `Dockerfile.ubi`: for OpenShift, from a Red Hat UBI language image, for example `ubi9/nodejs-22`
  or `ubi9/openjdk-21`;
- `toolchain-image.yml`: a GitHub Actions workflow that builds, pushes and attests the image.

Copy them into your own repository, set the base and the tools, and push. Then put the printed
digest in `PILOT_TOOLCHAIN_IMAGE`.
