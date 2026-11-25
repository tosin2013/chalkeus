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

## 2026-11-25: Calico's network token expired after 24 hours

The first Mac fixture run, [20261125T182055Z](20261125T182055Z/), passed A/1 (the induced
failure), A/2 and the reconcile. Then B/1's worker never started. It hit the 1200 s deadline,
and P06 and P07 failed with `model=probe not ready` and an empty `twin=`. The cluster dated from
the `up` above.

**What the node showed:**
- The kubelet log repeated `plugin type="calico" failed (add): error getting
  ClusterInformation: connection is unauthorized: Unauthorized`.
  - 416 lines, from 18:42:21 to 19:02:20 UTC.
  - Every new pod failed at sandbox creation: B/1's worker and the P06/P07 probe twin.
  - Pods that were already running were unaffected.
- `/etc/cni/net.d/calico-kubeconfig` was last written at 2026-11-24 18:41:01 UTC, when `up`
  ran. The first failure came 24 hours and 1 minute later.
- The `calico-node` log has a single `token_watch.go … Wrote updated CNI kubeconfig file` line,
  from its start, and none since.
- The VM's `/proc/uptime` read 8,789 s (about 2.4 hours) after more than 24 hours of real time.
  `pmset -g log` shows the Mac sleeping and waking repeatedly.

**Cause:**
- The CNI token lasts 24 hours. The token written after the restart decodes to
  `iat 2026-11-25T19:09:18Z`, `exp 2026-11-26T19:09:18Z`.
- `calico-node` renews it on a timer that runs on the VM's clock, which stops while Docker
  Desktop's VM is paused. The API server checks expiry against real time.
- On Linux, Docker runs on the host, so this doesn't happen. That is why T013 never saw it.

**Fix applied:**
- `kubectl -n kube-system rollout restart ds/calico-node` wrote a new token at 19:09:18 UTC.
- A throwaway pod then reached `Running` with a Calico address.

**What we do with it:**
- Run 20261125T182055Z stays as the record of this failure; it doesn't count for T025.
- T025 is repeated as a new run after `reset-fixture`.
- Docs: troubleshooting entry `#calico-token-expired`, and the macOS note in operator guide
  section 3.
- Code fix: T034.

## 2026-11-25: fixture run and `down` (T025, T023)

The repeated fixture run, [20261125T191440Z](20261125T191440Z/), ran after the Calico restart
and a `reset-fixture`. Approvals came from the owner with `./chalkeus forge approve` (channel
`harness`).

- A/1 failed on G5_hidden_tests (the induction). A/2 and B/1 passed. Draft PRs #77 (task A,
  merged by the owner) and #78 (task B).
- `checks/results.jsonl`: 32 of 32. The check ids and results are identical to T013's Linux run
  20261125T170145Z.
- `versions.txt` host lines:
  - `host_os: darwin 27.0.1`, `host_arch: arm64`;
  - `runtime: Docker Desktop 4.100.1 (247138) (Engine 29.9.3), 14 CPUs, 8.2 GiB`;
  - `node_platform: linux/arm64`, and `linux/arm64` for the egress, tools and worker images;
  - `host_spec: spec 0.4.1` (the darwin/arm64 binary).
- Cost: about $0.08 and 40,500 tokens across the three attempts (`deepseek/deepseek-v4-pro`).

`./chalkeus forge down`, exit 0. Then `./chalkeus forge audit --run 20261125T191440Z`: 11 of 11
pass, including AU07 (teardown).

```text
== teardown
  PASS T01 kind cluster deleted (absent)
  PASS T02 no kind node containers remain (containers=0)
  PASS T03 local registry container removed (containers=0)
```

What remains on the host after `down`:
- `kind get clusters`: `No kind clusters found.`
- `docker ps -a`: no kind or registry containers.
- `docker network ls`: only kind's shared `kind` network, created by the first `kind create
  cluster` on 2026-11-24 18:38 UTC, with no containers attached. kind keeps this network on
  purpose for later clusters and never deletes it. `down` doesn't remove it on any host, and
  T01–T03 don't check networks.

## Pending

- `doctor`'s node and image platform rows. doctor needs `--target`, and it refuses the fixture,
  so these come with T026's target.
