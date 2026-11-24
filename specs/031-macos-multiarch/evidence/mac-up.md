# T022–T023: `up` and `down` on the Mac

**Host:** Apple Silicon Mac, macOS 27.0.1, Docker Desktop 4.94.0 (Engine 29.8.2), VM raised to
8.2 GiB, 14 CPUs. `main` at the merge of #196: `pilot/images.lock` holds the multi-platform
index digests.

## 2026-11-24: `up --cluster-only`

`./chalkeus forge up --cluster-only`, exit 0:

```text
== pinned tools
== kind cluster chalkeus-m1
== calico v3.29.3
== cluster only: skipping images, proxies and publisher
```

- The host check passed with no warning (darwin/arm64, Docker Desktop).
- `tools.sh` installed kind v0.27.0 darwin/arm64 and kubectl v1.32.2, each checked against its
  published checksum.
- Node `chalkeus-m1-control-plane`: Ready, Kubernetes v1.32.2, `linux/arm64`, kernel
  `7.0.14-linuxkit`, containerd 2.0.2. `calico-node` ready 1/1.

## 2026-11-24: refusals with the daemon unreachable

`DOCKER_HOST=unix:///nonexistent.sock` makes the daemon unreachable. It takes the same path as a
stopped Docker Desktop, without stopping the cluster above.

- `./chalkeus forge doctor`: `PASS host: darwin/arm64 (macOS 27.0.1), validated`, then
  `FAIL container runtime: Docker isn't running: start Docker Desktop`.
- `./chalkeus forge up --cluster-only`: `chalkeus: Docker isn't running: start Docker Desktop`,
  exit 2. Nothing was installed (no `== pinned tools` line), and the existing cluster was
  untouched.

## 2026-11-24: full `up` (images, proxies, publisher)

**Setup:**
- The model is `deepseek/deepseek-v4-pro`, imported from the owner's OpenCode login with
  `opencode auth export | ./chalkeus forge import-opencode --from-login --auth-file - …`.
- The publisher is the GitHub App `tosins-specfoundry-pilot[bot]`, stored with `app-key`: push
  to the fixture True, push to `tosin2013/chalkeus` False.

**First attempt:** it refused with `missing config file: …/publisher.env`. `config.Load` always
required the token file, even with `FORGE_PUBLISHER_CREDENTIAL=app`. The Linux machine never hit
this because it still had an old token file. The fix (this branch) requires `publisher.env` for
a token and `github-app.pem` (mode 0600) for the App.

**After the fix:** `./chalkeus forge up`, exit 0:

```text
publisher token: OK (tosins-specfoundry-pilot[bot], repo_push True, canary_push False, 1 hour)
== pinned tools
== kind cluster chalkeus-m1
== calico v3.29.3
== images: pulled by digest from pilot/images.lock
== egress proxies
== publisher namespace
== check helpers
== up complete
```

**Running pods:**
- `pilot-control` and `pilot-grader-standin` run `chalkeus-tools@sha256:b5b39a57…`.
- The three egress proxies run `chalkeus-egress@sha256:6ec78c90…`.
- Both are the index digests merged in #196.
- `crictl inspecti` on the node: all 15 images are `"architecture": "arm64"`.

## Pending

- `doctor`'s node and image platform rows. doctor needs `--target`, and it refuses the fixture,
  so these come with T026's target.
- T023 (`down` and the check that nothing remains) comes after the fixture and target runs, so
  their audits can include the teardown (AU07).
