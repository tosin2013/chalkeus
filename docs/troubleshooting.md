# Troubleshooting

Each entry starts with what you see, then explains the cause, the fix and how to confirm the
fix. Entries are grouped by when they happen. The [operator guide](operator-guide.md) links
here from each step.

To watch a run while it happens, add `-vv` (pod phases and Kubernetes events) or `-vvv` (also
the agent's live activity) to the command, for example `./chalkeus forge resume --run <id> -vv`. See
[section 12 of the guide](operator-guide.md#12-when-something-goes-wrong).

When you ask for help, never share your API key, your token, anything under
`~/.config/chalkeus/` or `~/.config/opencode/`, or your provider's hostname. See
[section 12 of the guide](operator-guide.md#12-when-something-goes-wrong) for what to share
instead.

## Index by symptom

| You see | Entry |
|---|---|
| `needs opencode.json and at least one <provider>_api_key file`, or the worker fails with HTTP 401 | [No API key file](#no-api-key) |
| `Cost (USD)` is `0.0` for every attempt | [Cost shows 0](#cost-zero) |
| `problem: token can write …: scope it to the fixture repo only` | [Token can write too much](#token-writes-project) |
| `problem: token rejected by GitHub (HTTP 401)` | [Token expired or revoked](#token-expired) |
| A prompt never appears, and the run exits 5 or pauses | [Prompts answer "no"](#prompts-answer-no) |
| `./chalkeus forge up` fails while pulling images | [Image pull fails](#image-pull-error) |
| `Calico manifest hash mismatch` | [CNI hash mismatch](#cni-hash-mismatch) |
| `tagged reference not allowed` or `not a digest reference` | [Image pinned by tag](#tag-only-image) |
| `open pilot PRs exist on …` | [Open pilot PR left over](#open-pilot-pr) |
| `A's PR is not merged yet` | [Resume before merge](#resume-not-merged) |
| An attempt fails with reason `deadline` or `spend_cap` | [Deadline or spend cap](#deadline-or-spend-cap) |
| The retry question recommends `back to planning`, or you want to end a run paused for a retry | [Diagnosis and back to planning](#diagnosis) |
| An attempt's grade shows `fail (<reason>)` | [Attempt failure reasons](#attempt-failure-reasons) |
| On a Mac, new pods don't start (`connection is unauthorized`; `P06` shows `probe not ready`) | [Pods don't start on a Mac](#calico-token-expired) |
| `GitHub Actions is not permitted to create or approve pull requests` | [Actions can't open PRs](#actions-cannot-open-prs) |
| `T01`, `T02` or `T03` fails after `./chalkeus forge down` | [Leftovers after teardown](#leftovers) |
| `the kind cluster specfoundry-m1 … from before the rename` | [Old cluster from before the rename](#old-cluster) |
| `no issue write access to … (needs Issues: read and write)` | [No issue access](#github-issues-access) |
| `github drift: N difference(s)` | [GitHub drift](#github-drift) |
| A `/approve` reply on GitHub didn't resume the run | [GitHub approval rejected](#github-approval-rejected) |
| The run fails with `STATUS_PR_EDITED`, `STATUS_PR_NOT_APPROVER` or `STATUS_PR_MERGED_BY_PUBLISHER` | [Status-change PR refused](#status-pr-refused) |
| `GitHub App: NOT OK`, or `GitHub App token: …` | [GitHub App problems](#github-app) |
| An item run refuses with `ITEM_…`, `INPUT_TOO_LARGE`, `OPEN_ITEM_PR` or `TOOLCHAIN_UNAVAILABLE` | [Item not prepared](#item-not-ready) |
| A slice run refuses with `SLICE_…` or `ITEM_PARTLY_SLICED` | [Slice refused](#slice-refused) |
| A slice baseline fails with `EARLIER_SLICE_BROKEN_ON_BASE` | [Slice baseline](#slice-earlier-broken) |
| An item run fails with `BASELINE_SUITE_FAILS` or `BASELINE_HIDDEN_PASSES` | [Baseline fails](#item-baseline) |
| An item run fails with `PREPARE` or `REPLAY_MISMATCH`, or `target … needs its own publisher token` | [Item run setup](#item-setup) |
| An item run fails with `TOOLCHAIN_CONTRACT` or `PREFETCH_FAILED` | [Toolchain or dependencies](#item-toolchain) |
| `FAIL P06r` during an item attempt | [Registry check fails](#item-registry-check) |
| Every attempt fails at `G1_bundle_applies` with `harness_error`, 0 tokens, and the worker log shows `provider.auth` / 401 | [Model key rejected](#model-key) |
| A run refuses with `HIDDEN_TESTS_IN_TARGET` | [Hidden tests in the target](#hidden-in-target) |
| `./chalkeus forge author` refuses or fails (`COMPANION_NOT_EMPTY`, `DRAFT_PR_OPEN`, `AUTHOR_CAN_WRITE_TARGET`, `LEVEL_NEEDS_CLUSTER_MODE`, `DRAFT_*`, `MAP_INVALID`) | [Test author](#test-author) |

---

## Setup

<a id="no-api-key"></a>
### No API key file: "needs opencode.json and at least one <provider>_api_key file"

- **Symptom:**
  - `./chalkeus forge up` or `run` stops with
    `~/.config/chalkeus/opencode needs opencode.json and at least one <provider>_api_key file`;
  - or the attempt's `report.json` shows the harness exited after an HTTP 401 from the model.
- **Cause:** OpenCode resolves `"apiKey": "{file:~/.config/opencode/<provider>_api_key}"`
  inside the worker. If the key file is missing, empty or wrongly named, the provider rejects
  the request.
- **Fix:** if OpenCode works on your machine, run `./chalkeus forge import-opencode --force` to copy
  its setup again. Otherwise:
  1. Put the key, on one line, into `~/.config/chalkeus/opencode/<provider>_api_key`.
  2. Make sure `<provider>` matches the name in the `{file:...}` reference.
  3. Set modes: `chmod 700 ~/.config/chalkeus/opencode` and `chmod 600` on each file inside.
- **Confirm:** `./chalkeus forge up` passes its config checks. The next attempt's `report.json`
  shows a harness exit of 0 and a non-zero token count.
- **Source:** the 002 live run setup (OpenCode returned 401 until the key file existed).

<a id="token-writes-project"></a>
### Token can write too much: "token can write …: scope it to the fixture repo only"

- **Symptom:** `./chalkeus forge publisher-token` prints `publisher token: NOT OK`,
  `canary_push: True`, this problem line, and `not saved: …`. The token was thrown away.
- **Cause:** the token can push to the Chalkeus repository you cloned from (the
  "canary"). That usually means a classic token, or a fine-grained token with *All
  repositories* access. The check is a real write test, not GitHub's permission labels, which
  describe your account rather than the token on public repositories.
- **Fix:** delete the token on GitHub. Create a fine-grained token with **Only select
  repositories** set to your fixture repo
  ([guide section 6](operator-guide.md#6-prepare-a-throwaway-fixture-repo-and-a-scoped-token)),
  then run `./chalkeus forge publisher-token` again.
- **Confirm:** the output shows `repo_push: True` and `canary_push: False`.
- **Source:** commit 7570552 (the write probe replaced permission labels after the repo went
  public).

<a id="token-expired"></a>
### Token expired or revoked: "token rejected by GitHub (HTTP 401)"

- **Symptom:** `./chalkeus forge publisher-token --check-only` (also run by `up`) prints
  `publisher token: NOT OK` and `problem: token rejected by GitHub (HTTP 401)`.
- **Cause:** the fine-grained token has expired, or it was deleted or regenerated on
  GitHub. The last check's `expires:` line shows when it would expire.
- **Fix:** generate a new token with the same settings, then run `./chalkeus forge publisher-token`
  and paste it at the hidden prompt. The old `publisher.env` is replaced.
- **Confirm:** `./chalkeus forge publisher-token --check-only` prints `publisher token: OK` with a
  future `expires:` date.
- **Source:** `pilot/images/tools/check_token.py`.

<a id="actions-cannot-open-prs"></a>
### Actions can't open PRs: "GitHub Actions is not permitted to create or approve pull requests"

This only affects you if you **fork Chalkeus and run its image workflow** yourself.
Operators who pull the published images never see it.

- **Symptom:** the `pilot-images` workflow builds and pushes the images, but its last step
  fails with this message, so no PR updates `pilot/images.lock`.
- **Cause:** by default, GitHub doesn't let workflows open pull requests.
- **Fix:** in your fork, go to *Settings → Actions → General → Workflow permissions*, tick
  **Allow GitHub Actions to create and approve pull requests**, then re-run the failed job.
- **Confirm:** a PR titled `chore(images): update digests …` appears.
- **Source:** the first `pilot-images` run of 003 ([ci.md](../specs/003-m2-kelos-backend/evidence/ci.md)).

---

## Stand-up

<a id="image-pull-error"></a>
### Image pull fails during `./chalkeus forge up`

- **Symptom:** `./chalkeus forge up` stops after `== images`, with a Docker pull error such as
  `toomanyrequests`, `manifest unknown` or a network timeout.
- **Cause:**
  - the machine can't reach `ghcr.io`, for example because of a proxy or firewall;
  - or a rate limit was hit;
  - or `pilot/images.lock` names a digest that isn't published (only possible on an edited
    or unmerged branch).
- **Fix:**
  1. Check you can reach the registry: `docker pull hello-world`, then try one image from
     `pilot/images.lock` by hand.
  2. Behind a proxy, configure Docker's proxy (see
     [enterprise environments](harnesses.md#enterprise-environments)).
  3. Make sure you're on `main` with no local edits to the lock: `git status`.
  4. Run `./chalkeus forge up` again.
- **Confirm:** the output reaches `== images: pulled by digest from pilot/images.lock` and
  `== up complete`.
- **Source:** 003 validation run `20261113T180203Z` (pull by digest).

<a id="cni-hash-mismatch"></a>
### CNI hash mismatch: "Calico manifest hash mismatch"

- **Symptom:** `./chalkeus forge up` stops with
  `Calico manifest hash mismatch: expected <pinned>, got <actual>` and exits 2.
  Nothing has been applied to the cluster.
- **Cause:** the downloaded Calico manifest differs from the one pinned in
  `pilot/pins.env` (`CALICO_SHA256`). Either `pins.env` was edited, or the download was
  altered or truncated. The pilot never applies an unverified network plugin.
- **Fix:**
  1. Restore the pinned values: `git checkout pilot/pins.env`.
  2. Run `./chalkeus forge down`, then `./chalkeus forge up`.
  3. If it still fails on an untouched checkout, your network is altering the download.
     Don't work around it; open an issue.
- **Confirm:** `./chalkeus forge up` completes, and the run's `versions.txt` lists `calico: <version>
  sha256=<pinned value>`.
- **Source:** 003 negative check (a wrong `CALICO_SHA256` exits 2 before applying anything).

<a id="tag-only-image"></a>
### Image pinned by tag: "tagged reference not allowed" or "not a digest reference"

- **Symptom:** `./chalkeus forge up` or `run` stops with, for example,
  `WORKER_IMAGE in pilot/images.lock: tagged reference not allowed: '…:latest@sha256:…'`, or
  `not a digest reference`, and exits 2.
- **Cause:** every image must be referenced as `name@sha256:<64 hex>`, with no tag. A tag can
  be moved to different content later; a digest can't.
- **Fix:** restore the lock with `git checkout pilot/images.lock`. If you're mirroring images,
  write the mirror's `name@sha256:…` reference, which has the same digest (see
  [enterprise environments](harnesses.md#enterprise-environments)).
- **Confirm:** `./chalkeus forge up` completes. Live check P10 passes during the run (every image
  pinned and non-root).
- **Source:** 003 negative check (a tag-only `WORKER_IMAGE` exits 2 before applying anything).

---

## Run time

<a id="prompts-answer-no"></a>
### Prompts answer "no": no question appears

- **Symptom:**
  - `./chalkeus forge run` pauses at the mode prompt (exit 5) without asking, and prints
    `paused: mode not confirmed. no terminal: approve with ...`; or
  - it pauses at the retry (exit 6) without asking; or
  - `resume` prints `waiting for your approval of the status change` (exit 5) without asking.
- **Cause:** prompts are read from your terminal. Without one (an agent's shell tool, an IDE
  task runner, CI, `nohup`, or piped input), nothing is assumed: the run pauses at the
  decision. This is deliberate: nothing proceeds without a human answer.
- **Fix:** either
  - run `./chalkeus forge resume --run <run-id>` in a real interactive terminal, for example a plain
    SSH session, and answer there; or
  - when you work with an agent, approve the one pending step with
    `./chalkeus forge approve --run <run-id> mode|retry|reconcile`. Your agent harness must ask
    **you** before that command runs; say yes only if you agree. Then run
    `./chalkeus forge resume --run <run-id>`. See the `forge` skill
    (`.claude/skills/forge/SKILL.md`).
    - **Claude Code:** this repo's `.claude/settings.json` already has the ask rules. They cover
      `approve` and the nine other commands that decide, destroy or hold credentials:
      `reset-fixture`, `down`, `publisher-token`, `app-key`, `app-token`, `author`, `step`, `run`
      and `resume`.
    - **OpenCode:** add the rules once to your own config, `~/.config/opencode/opencode.json`,
      so they cover every project (the repo-root `opencode.json` is gitignored). Add one pair for
      each of the ten commands; `approve` is shown:

      ```json
      "permission": { "bash": {
        "./chalkeus forge approve*": "ask",
        "chalkeus forge approve*": "ask"
      } }
      ```
    - **Any other harness:** no rule is provided. Use a real terminal instead.
    - Keep the harness in its normal permission mode. Claude Code's auto mode, and any bypass
      or auto-accept mode, run `approve` without asking you, so the approval isn't really yours.

  Merging is never approved this way: merge A's PR on GitHub yourself.
- **Confirm:** `./chalkeus forge status` shows the run moving on. In the audit, AU09 and AU10 list
  each approval's channel: `tty` (terminal) or `harness` (`./chalkeus forge approve`).
- **Source:** the 002 live run (prompts read the controlling terminal) and the 006 validation
  (an agent's shell has no terminal; finding F4).

<a id="open-pilot-pr"></a>
### Open pilot PR left over: "open pilot PRs exist on …"

- **Symptom:** `./chalkeus forge run` prints
  `open pilot PRs exist on <you>/<you>-pilot-fixture: [...]` and ``run `./chalkeus forge reset-fixture` first``,
  then exits 2.
- **Cause:** a previous run left draft PRs open. A new run won't start on top of them, so
  results can't mix.
- **Fix:** `./chalkeus forge reset-fixture`, then type `y`. This force-resets the fixture repo and
  closes open `pilot/*` PRs.
- **Confirm:** `./chalkeus forge run` gets past this check and shows the gate self-checks.
- **Source:** `internal/pipeline/pipeline.go` (pre-run check).

<a id="deadline-or-spend-cap"></a>
### Deadline or spend cap: an attempt fails with reason `deadline` or `spend_cap`

- **Symptom:** an attempt fails before grading. In `summary.md` its grade shows
  `fail (deadline)` or `fail (spend_cap)`. You're asked to confirm a retry, or the run fails
  with `<task>_RETRIES_EXHAUSTED` after the last retry. The retry question starts with the
  attempt's diagnosis (see [Diagnosis and back to planning](#diagnosis)).
- **Cause:** one of three, and the diagnosis says which the evidence points to:
  - **a slow or overloaded model endpoint:** the attempt ran out of time (the worker deadline,
    `FORGE_WORKER_DEADLINE_S` in the target profile; 1,200 seconds by default) or spend without
    anything else going wrong;
  - **work larger than the budget:** the tasks are known, but the project's tests take so long
    that few full test runs fit in one attempt (the diagnosis and `forge doctor --baseline` show
    the test-run allowance);
  - **unknown work or an environment need:** a task is research ("locate the suites that…"),
    or the work needs the runtime, dependencies, egress or the lock file changed. More time
    doesn't help: the attempt keeps searching, or changes what it isn't allowed to.
  - **`spend_cap`:** the attempt reached `FORGE_SPEND_CAP_USD`, or `FORGE_TOKEN_CAP` when
    the provider reports no cost, and the watchdog stopped it.
- **Fix:** follow the diagnosis's recommendation; the choice is yours.
  - **retry** when the cause looks temporary; for repeated `deadline` failures with nothing
    else in the diagnosis, try a faster model;
  - **raise the budget:** set `FORGE_WORKER_DEADLINE_S` higher in the target profile, then start
    a new run (a run keeps the budget it started with);
  - **back to planning:** end the run with `./chalkeus forge approve --run <id> replan` (or
    `/approve replan` on the tracking issue), resolve the research or the environment change in
    planning, then start a new run.
  - For repeated `spend_cap` failures, check the attempt's token count in `summary.md`. A
    reference attempt uses roughly 225,000–480,000 tokens. Raise the cap in
    `~/.config/chalkeus/forge.env` only if you accept the cost.
  - After `RETRIES_EXHAUSTED`, start over with `./chalkeus forge reset-fixture` and then
    `./chalkeus forge run`.
- **Confirm:** a later attempt reaches grading. AU06 passes in the audit.
- **Source:** `internal/worker/worker.go` (failure reasons), `internal/diagnose` (diagnosis).
- **If the worker never started:** on a Mac, with a cluster more than a day old, see
  [Pods don't start on a Mac](#calico-token-expired).

<a id="calico-token-expired"></a>
### Pods don't start on a Mac: `connection is unauthorized`

- **Symptom:** on a Mac, a cluster that has been up for about a day stops starting pods. An
  attempt fails with reason `deadline` although the model is fine, and `P06` and `P07` fail
  with `model=probe not ready` and an empty `twin=`. The node's kubelet log repeats:

  ```text
  plugin type="calico" failed (add): error getting ClusterInformation: connection is unauthorized: Unauthorized
  ```

  Pods that were already running keep working, so `up`, the proxies and the control pod look
  healthy.
- **Cause:** Calico's network plugin calls the Kubernetes API with its own token, which lasts
  24 hours. `calico-node` writes it to `/etc/cni/net.d/calico-kubeconfig` on the node when it
  starts, and renews it on a timer. That timer runs on the Docker Desktop VM's clock, which
  stops while the Mac sleeps, but the API server checks the token's expiry against real time.
  After enough sleep the token expires before the timer is due, and every new pod fails to
  get a network.
  On Linux, Docker runs on the host with no VM to pause, so the timer keeps time and renews
  the token. 031's Mac run (2026-11-25) found this: the token was written once at `up`, the VM had
  counted about 2.4 hours of uptime in the 24 hours since, and the first failure came 24 hours
  and 1 minute after `up`.
- **Fix:**
  - Restart `calico-node`. It writes a new 24-hour token; nothing else changes:

    ```sh
    kubectl --context kind-chalkeus-m1 -n kube-system rollout restart ds/calico-node
    kubectl --context kind-chalkeus-m1 -n kube-system rollout status ds/calico-node
    ```

  - `./chalkeus forge down` and `./chalkeus forge up` also fix it, more slowly.
  - The attempt that failed this way never ran its worker. Confirm the retry, or, for a run
    whose evidence should show no infrastructure failure, reset the fixture and start a new
    run.
  - To avoid it: on a Mac, restart `calico-node` (or run `up` again) before a run if the
    cluster is more than about 20 hours old, and keep the Mac awake during a run.
- **Confirm:** the node's file `/etc/cni/net.d/calico-kubeconfig` has a fresh time
  (`docker exec chalkeus-m1-control-plane ls -l /etc/cni/net.d/calico-kubeconfig`), and new
  pods reach `Running`.
- **Source:** Calico's `cni-config-monitor` (`token_watch.go` in the `calico-node` log);
  `specs/031-macos-multiarch/evidence/mac-up.md`.

<a id="diagnosis"></a>
### Diagnosis and back to planning: the retry question recommends "back to planning"

- **Symptom:** after a failed attempt, the retry question (and the tracking issue, and
  `summary.md` under `## Diagnosis`) shows `diagnosis of attempt N`, its signals, what they
  point to, and `recommended: back to planning`.
- **Cause:** the attempt's evidence points to work an attempt can't finish:
  - **unknown work:** an unchecked task reads as research (it starts with one of the phrases in
    [targets: research tasks](targets.md#research-tasks)), or the failure reason isn't one of
    the [attempt failure reasons](#attempt-failure-reasons) below;
  - **an environment need:** a dependency manifest or lock file changed outside the allowed
    paths, or an install or download failed while the egress proxy refused hosts;
  - **a repeated timeout:** this attempt and the one before it both stopped at the deadline with
    nothing else in the evidence. Open-ended work isn't always worded as research ("make … pass
    reliably").
  The diagnosis is built from the attempt's kept files, without a model, and the same evidence
  always gives the same diagnosis. Forge never acts on it.
- **Fix:** go back to planning:
  1. end the run: `./chalkeus forge approve --run <id> replan` (your agent harness asks you
     first), or reply `/approve replan` on the tracking issue. The run stops as
     `stopped(back-to-planning)`; its watcher stops and the audit reports AU02 and AU03 as `n/a`;
  2. for unknown work, do the research outside the run (for example run the investigation
     yourself, or with an agent that assists you) and record the answer in the item's
     `research.md` with `/speckit-plan`; then turn it into known tasks with `/speckit-tasks`.
     The diagnosis names a Spec Kit extension for it (`tool for …` lines): the **Research
     Harness** (`/speckit.harness.explore`, then `/speckit.harness.report`, which writes
     `research.md`) for unknown work, and **Bug Triage** (`/speckit.bug.assess` with the
     diagnosis as the report) for a defect in the project. It shows the commands when the target
     has the extension enabled, otherwise how to install it. The Research Harness is a community
     extension that Spec Kit hasn't vetted: review it before you install it. Forge never
     installs or runs either;
  3. for an environment need, make the change in your planning PR (the runtime, dependencies,
     the lock file) or in the target profile (registries);
  4. merge the planning PR and start a new run of the item or slice.
- **Confirm:** `./chalkeus forge doctor --target <t> --item NNN --slice <id> --baseline` shows no
  research-task warning, and the new run's attempt reaches grading.
- **Source:** `internal/diagnose`, `internal/pipeline/diagnose.go`.

<a id="attempt-failure-reasons"></a>
### Attempt failure reasons

- **Symptom:** an attempt's grade in `summary.md` shows `fail (<reason>)`.
- **Cause and fix**, by reason:
  - `deadline`, `spend_cap`: see [Deadline or spend cap](#deadline-or-spend-cap).
  - `grade_fail`: the change was graded and a check failed (G1–G8 in the attempt's
    `grade.json`). The usual case: retry. A first attempt of the fixture's task A fails on
    purpose (AU11).
  - `harness_error`: the agent harness or the worker setup failed (clone, workspace, model
    provider). Look at the attempt's `report.json` `log_tail`; a 401 is a
    [model key problem](#model-key).
  - `no_result`, `bad_frame`, `oversize`: the worker or grader result file was missing,
    unreadable or too large. A crashed pod or a full disk; retry, and look at the pod's events
    with `-vv`.
  - `grader_error`, `grader_timeout`: the grader crashed or ran past its job deadline. Raise
    `FORGE_GRADER_DEADLINE_S` in the target profile if the project's tests are slow.
  - worker report errors `clone_failed`, `checkout_failed`, `workspace_failed`,
    `bundle_failed`, `bad_meta`, `induction_failed`: the worker couldn't set up or package its
    workspace. Retry once; if it repeats, report it with the attempt's `report.json`.
  - `no_frame` (a note on a `deadline`): the worker was stopped before it could write its
    result, so its usage and changes weren't collected.
  - any other reason: the diagnosis reports it as not recognised; go
    [back to planning](#diagnosis) to find the cause before retrying.
- **Source:** `internal/diagnose/catalogue.go` (the list the docs test checks).

<a id="cost-zero"></a>
### Cost shows 0: `Cost (USD)` is `0.0` for every attempt

- **Symptom:** in `summary.md`, every attempt shows `0.0` cost but a large token count.
- **Cause:** your provider, like many OpenAI-compatible gateways, doesn't report a price.
  Nothing is wrong: the spend cap falls back to `FORGE_TOKEN_CAP` (2,000,000 tokens per
  attempt by default).
- **Fix (optional):** to cap in dollars, add per-million-token prices to the model in
  `opencode.json`, for example `"cost": {"input": 3.0, "output": 15.0}`, using your provider's
  real prices.
- **Confirm:** the next run's `summary.md` shows non-zero costs. AU06 still passes.
- **Source:** 002 and 003 runs (cost 0 from the gateway; the token fallback applied).

---

## Merge and resume

<a id="resume-not-merged"></a>
### Resume before merge: "A's PR is not merged yet"

- **Symptom:** `./chalkeus forge resume` prints
  `A's PR is not merged yet. Merge it on GitHub, then run pilot resume again.` and exits 4.
- **Cause:** reconcile only proposes `A verified` after A's PR has actually merged. Nothing
  was changed.
- **Fix:** open the PR link printed by `run` (or check `./chalkeus forge status`). Click **Ready for
  review**, then **Merge**, then run `./chalkeus forge resume`.
- **Confirm:** `resume` shows the reconcile proposal and asks
  `Apply '001: planned → verified' to the fixture roadmap? [y/N]`.
- **Source:** `internal/pipeline/pipeline.go` (resume check).

---

## GitHub loop

These entries apply only with `FORGE_GITHUB_ISSUES=on` or `FORGE_GITHUB_APPROVALS=on`. See
[the GitHub loop](github-loop.md).

<a id="github-issues-access"></a>
### No issue access: "needs Issues: read and write"

- **Symptom:** `./chalkeus forge publisher-token --check-only` (or `app-token --check`) reports
  `no issue write access to <repo>`, or `github sync` fails with HTTP 403.
- **Cause:** the GitHub loop writes issues, labels and comments, and the publisher credential
  lacks *Issues: read and write*.
- **Fix:** edit the fine-grained token on GitHub (*Settings → Developer settings →
  Fine-grained tokens*) and add *Issues: read and write*. For an App, add the permission on the
  App's page, then accept the new permission on the installation.
- **Confirm:** the check prints `OK`.
- **Source:** `pilot/images/tools/check_token.py` (issues probe).

<a id="github-drift"></a>
### GitHub drift: "github drift: N difference(s)"

- **Symptom:** `./chalkeus forge github drift` exits 1, or a run's `github/drift.json` isn't empty.
- **Cause:** something on GitHub disagrees with the roadmap:
  - `missing_issue`: no issue for an item;
  - `label_mismatch`: a status label was changed by hand;
  - `open_but_verified` / `closed_but_not_verified`: an issue was closed or reopened by hand;
  - `pr_without_closes`: a task PR lost its `Closes #N`;
  - `orphan_issue` / `duplicate_issue`: issues for items no longer in the roadmap, or two
    issues for one item.
- **Fix:** run `./chalkeus forge github sync`. It recreates missing issues and restores labels and
  bodies. It never closes, reopens or deletes. Fix the rest on GitHub by hand: close an orphan
  or duplicate, or reopen an issue whose item isn't verified.
- **Confirm:** `./chalkeus forge github drift` exits 0.
- **Source:** `internal/ghloop/sync.go`.

<a id="github-approval-rejected"></a>
### GitHub approval rejected: the run stays paused

- **Symptom:** you replied `/approve <step>` on the tracking issue, but `resume` (or `github
  watch`) pauses again.
- **Cause:** the reply broke a rule. The run's `github/approvals.jsonl` and the
  `approval_rejected` events give the reason:

  | Reason | Meaning |
  |---|---|
  | `wrong_step` | the reply names a different step than the one pending |
  | `before_decision` | posted before the current decision post (for example, under an earlier attempt) |
  | `not_an_approver` | your login isn't in `FORGE_APPROVERS` |
  | `no_write_access` | your account no longer has write, maintain or admin on the fixture repo |
  | `publisher_identity` | posted with the publisher's account (in token mode, that's your own account through the token) |
  | `app_authored` | posted by a GitHub App or bot |
  | `edited` | the comment was edited after posting |
  | `withdrawn` | you posted `/withdraw <step>` afterwards |
  | `duplicate` | an earlier valid approval was used instead |
- **Fix:** post a new reply that is exactly `/approve <step>`, as yourself, below the latest
  decision post. If `publisher_identity` comes up in token mode, approve from the GitHub web UI
  rather than a tool using the token, or switch to a GitHub App.
- **Confirm:** `./chalkeus forge github watch --run <id>` resumes.
- **Source:** `internal/ghloop/approve.go`.

<a id="status-pr-refused"></a>
### Status-change PR refused

- **Symptom:** after you merge the status-change PR, the run fails with one of these:
  - `STATUS_PR_EDITED`;
  - `STATUS_PR_NOT_APPROVER`;
  - `STATUS_PR_MERGED_BY_PUBLISHER`;
  - `STATUS_NOT_ON_DEFAULT`.
- **Cause:** the merge doesn't count as an approval:
  - **`STATUS_PR_EDITED`:** the PR changed more than the one status line;
  - **`STATUS_PR_NOT_APPROVER`:** it was merged by someone not in `FORGE_APPROVERS`;
  - **`STATUS_PR_MERGED_BY_PUBLISHER`:** it was merged with the publisher's account;
  - **`STATUS_NOT_ON_DEFAULT`:** the default branch doesn't show the item `verified`.
- **Fix:** the run can't continue, because the roadmap may now hold an unapproved change.
  Run `./chalkeus forge reset-fixture`, then start a new run. Next time, merge the status-change PR
  unedited, from an approver's account.
- **Confirm:** the new run's `reconcile/A-status.json` shows `"ok": true`.
- **Source:** `internal/reconcile/reconcile.go` (`CheckStatusPR`).

<a id="github-app"></a>
### GitHub App problems

- **Symptom:** `./chalkeus forge app-token --check` prints `GitHub App: NOT OK`, or a run stops with
  `GitHub App token: …`.
- **Cause and fix:**
  - **`cannot sign with the App key`:** the file isn't the App's private key. Generate a new
    one on the App's page, and store it with `./chalkeus forge app-key`.
  - **`GitHub rejected the App JWT`:** `FORGE_APP_ID` is wrong, or the key belongs to another
    App.
  - **`cannot mint an installation token`:** `FORGE_APP_INSTALLATION_ID` is wrong, or the App
    isn't installed on the fixture repo.
  - **`the App lacks …: write`:** add the permission on the App's page, then accept it on the
    installation.
  - **`token covers […]`:** the installation includes more repositories. Limit it to the
    fixture repo.
  - **`both a publisher token and a GitHub App key exist`:** set `FORGE_PUBLISHER_CREDENTIAL`
    to `token` or `app`.
- **Confirm:** `./chalkeus forge app-token --check` prints `OK`.
- **Source:** `internal/ghapp/ghapp.go`; see
  [github-app.md](github-app.md).

---

## Your own repositories (targets)

<a id="item-not-ready"></a>
### Item not prepared: the run refuses with exit 10

- **Symptom:** `./chalkeus forge run --target <name> --item NNN` prints `readiness NNN: refuse [...]`
  and `item NNN is not prepared`, then exits 10. No worker starts.
- **Cause:** one of the readiness rules failed. Its code is first in the list.
- **Fix:** see the table in [targets](targets.md#3-check-that-an-item-is-ready). Each code names
  what to add or change in the target or its companion repository. `TOOLCHAIN_UNAVAILABLE`
  means the profile names a toolchain with no published image: bring your own image or ask for
  one ([targets](targets.md#your-own-toolchain-image)).
- **Confirm:** `./chalkeus forge gate --target <name> --item NNN` prints `"decision": "allow"`.
- **Source:** `internal/readiness/readiness.go`.

<a id="slice-refused"></a>
### Slice refused: `SLICE_…` or `ITEM_PARTLY_SLICED`

- **Symptom:** `./chalkeus forge run --target <name> --item NNN --slice USn` (or a run without
  `--slice`) refuses with exit 10 and one of these codes. Nothing starts.
- **Cause and fix, by code:**
  - `SLICE_UNEVALUABLE`: spec-status reports no per-phase counts for the item. Install
    spec-status 0.4.1 or later (the version in `pilot/pins.env`), and check that the item's
    `tasks.md` is readable.
  - `SLICE_UNKNOWN`: the slice id isn't one of the item's (`./chalkeus forge slices` lists them), or
    two phases name the same user story. Fix `tasks.md`, or run the whole item without `--slice`.
  - `SLICE_EARLIER_OPEN`: an earlier slice isn't merged yet, or its PR is still open. Build and
    merge the slices in order; the detail names the earlier slice.
  - `SLICE_STARTED`: this slice already has ticked tasks on the default branch. Someone merged
    part of it outside the pilot; untick them, or plan the rest as a new item.
  - `SLICE_MAP_CHANGED`: `tasks.md` now groups phases differently from when a merged slice was
    built. The pilot never re-maps slices: restore the phases, or plan a new item.
  - `ITEM_PARTLY_SLICED`: a whole-item run of an item whose slices have begun. Run the remaining
    slices with `--slice`.
- **Confirm:** `./chalkeus forge slices --target <name> --item NNN` shows the slice you want as
  `current`, and `./chalkeus forge doctor --target <name> --item NNN --slice USn` passes.
- **Source:** `internal/readiness/slices.go`, `internal/slice/slice.go`.

<a id="slice-earlier-broken"></a>
### Slice baseline: `EARLIER_SLICE_BROKEN_ON_BASE`

- **Symptom:** a slice run's baseline fails before any attempt, with this reason.
- **Cause:** an earlier slice's hidden tests fail on today's default branch: something merged
  after that slice broke it.
- **Fix:** fix the default branch so the earlier slice's hidden tests pass again, then start the
  slice run again. `<run>/baseline/grade.json` shows G8's summary.
- **Confirm:** `./chalkeus forge doctor --target <name> --item NNN --slice USn --baseline` reports the
  earlier slices' hidden tests passing.
- **Source:** `pilot/images/tools/grade.py` (G8), `internal/grader/item.go`.

<a id="item-baseline"></a>
### Baseline fails: `BASELINE_SUITE_FAILS` or `BASELINE_HIDDEN_PASSES`

- **Symptom:** after you approve the mode, the run prints `baseline for item NNN failed` and
  fails. No worker has run, so no model spend was used.
- **Cause:** on the unchanged code, either:
  - the project's own test command fails (`BASELINE_SUITE_FAILS`), so a change can't be judged
    against it; or
  - the hidden tests already pass (`BASELINE_HIDDEN_PASSES`), so they can't tell a correct change
    from no change.
- **Fix:** for the first, make `FORGE_TEST_COMMAND` pass on the target's branch. For the second,
  make the item's hidden tests in `items/NNN/` test the new behaviour, so they fail today.
  `<run>/baseline/grade.json` shows the output.
- **Confirm:** a new run prints `baseline NNN: pass`.
- **Source:** `pilot/images/tools/grade.py` (baseline mode), `internal/grader/item.go`.

<a id="item-setup"></a>
### Item run setup: `PREPARE`, `REPLAY_MISMATCH`, or a missing target token

- **Symptom:** the run fails with `PREPARE` (it couldn't clone the target or the companion
  repository), or with `REPLAY_MISMATCH` while publishing. Or it stops at the start with
  `target "<name>" needs its own publisher token in …/targets/<name>.publisher.env`.
- **Cause:**
  - `PREPARE`: your git access can't read the target or the companion repository.
  - `REPLAY_MISMATCH`: the graded change couldn't be replayed onto the real base unchanged, so
    nothing is published that wasn't graded.
  - Missing token: in token mode, each target needs its own fine-grained token. The fixture's
    token is scoped to the fixture only.
- **Fix:**
  - `PREPARE`: check `git clone` of both repositories works for you.
  - `REPLAY_MISMATCH`: start a new run, and open an issue if it repeats.
  - Missing token: store a token for that repository only (contents and pull requests: write)
    in `~/.config/chalkeus/targets/<name>.publisher.env` as `PILOT_SCM_TOKEN=…`, mode 0600. Or
    use the GitHub App, installed on the target.
- **Confirm:** the run gets past the step that failed.
- **Source:** `internal/pipeline/item.go`, `internal/fixture/snapshot.go`, `internal/cli/run.go`.

<a id="model-key"></a>
### Model key rejected

- **Symptom:** each attempt ends after a few seconds with `harness_error`, 0 tokens and no
  change, so grading fails at `G1_bundle_applies`. The attempt's `report.json` log shows
  `provider.auth` with status 401 (for example "Key is blocked").
- **Cause:** the model provider refuses the worker's key: blocked (for example after heavy
  use), expired, revoked or wrong. The pilot uses its own copy of the key in
  `~/.config/chalkeus/opencode/`, so fixing the key in your own OpenCode setup doesn't fix the
  pilot's.
- **Fix:** get the key unblocked or a new one, then copy it in again: `./chalkeus forge
  import-opencode` (or `--from-login`), or replace the key file in
  `~/.config/chalkeus/opencode/` (mode 0600).
- **Confirm:** `./chalkeus forge doctor --target <name>` shows `PASS model key`. Each new attempt
  reads the key file again, so a paused run can then retry. That holds only for a new key with
  the same model: a run keeps the `forge.env` it started with, so after changing `FORGE_MODEL`
  or the provider (for example to OpenCode Go), start a new run.
- **Source:** `internal/config/probe.go`, `internal/config/import_login.go`.

<a id="hidden-in-target"></a>
### Hidden tests in the target: `HIDDEN_TESTS_IN_TARGET`

- **Symptom:** `./chalkeus forge gate` or `run` refuses with `HIDDEN_TESTS_IN_TARGET`, naming a
  hidden-test file and where its copy is in the target.
- **Cause:** a file in the target repository has exactly the content of a hidden test. A build
  worker would see it in its snapshot, so the run can't be trusted.
- **Fix:** remove the copy from the target (by pull request), find out how it got there (an
  agent, a script, a copy-paste), and consider the `recommended` security level so the test
  author's identity can't write the target.
- **Confirm:** `./chalkeus forge gate --target <name> --item NNN` allows.
- **Source:** `internal/readiness/readiness.go` (`hiddenInTarget`).

<a id="test-author"></a>
### Test author: `./chalkeus forge author` refuses or fails

- **Symptom:** a test-author command exits 10 (refused) or 1 (the draft's check failed).
- **Cause and fix, by code:**
  - `COMPANION_NOT_EMPTY`: the item already has hidden tests; review them, or start with
    `--replace` to propose new ones (still only a pull request).
  - `DRAFT_PR_OPEN`: a draft pull request is open on the companion; review or close it.
  - `AUTHOR_CAN_WRITE_TARGET`: at the `recommended` level, the token in `targets/<name>.author.env`
    can push to the target; give it access to the companion only.
  - `LEVEL_NEEDS_CLUSTER_MODE`: `strict` needs cluster mode, which isn't built yet; use
    `minimal` or `recommended`.
  - `DRAFT_EMPTY`, `DRAFT_OUTSIDE_FOLDER`: put regular test files under the workspace's `draft/`.
  - `MAP_INVALID`: fix `draft/.acceptance-map.json` as the message says.
  - `DRAFT_HIDDEN_PASSES`: the drafted tests already pass on today's code; they must fail until
    the spec is implemented.
  - `DRAFT_SUITE_FAILS`: with the drafted tests added, the project's own tests fail; the tests may
    break the build or need a dependency the project doesn't declare.
  - `DRAFT_NOT_CHECKED`: run `check` again; the draft changed since its last passing check.
  - `DRAFT_PATH_IN_SPEC`: a drafted file reuses a path the item's spec, plan or tasks name (the
    build worker's own files). Rename it, for example with an `AcceptanceTests` suffix, and
    update the map.
- **Automatic drafting (`author watch`):**
  - `auto-draft is off for target …`: set `FORGE_AUTHOR_AUTO=on` in the target's profile.
  - `the author proxy pilot-egress-a is missing`: run `./chalkeus forge up`.
  - `skip NNN: already drafted automatically for this spec version`: the last automatic draft
    failed its check twice. Change the spec, or draft by hand with the test-author skill.
  - `isolation check A0n failed, no PR`: the author's sandbox wasn't as expected. Read the
    draft's `draft.json` (`jobs[].checks`); don't run auto-draft until it passes.
- **Step mode (`./chalkeus forge step`):**
  - `step mode needs GitHub approvals`: set `FORGE_GITHUB_APPROVALS=on` in the target's profile
    or in `forge.env`.
  - `step mode needs FORGE_APPROVERS`: set it in `forge.env`; only an approver's label counts.
  - `NNN: skip (not proposed: the label was last added by @x, who is not in FORGE_APPROVERS)`:
    an approver must add the label (remove it and add it again).
  - A comment "`./chalkeus forge doctor --baseline` found problems": fix what it lists; step mode tries
    again once the target or the companion changes.
  - A comment "Run … ended without verifying this item": add the label again to retry.
- **Confirm:** `./chalkeus forge author status --target <name> --item NNN` shows `checked(pass)`, then
  `submitted`.
- **Source:** `internal/author/`.

<a id="item-toolchain"></a>
### Toolchain or dependencies: `TOOLCHAIN_CONTRACT` or `PREFETCH_FAILED`

- **Symptom:** after the mode approval, the run fails with `TOOLCHAIN_CONTRACT: …` or
  `PREFETCH_FAILED: …`, before any worker starts.
- **Cause:**
  - `TOOLCHAIN_CONTRACT`: the toolchain image failed a check. The message names it: no
    `/opt/toolchain/toolchain.json`, a rule it breaks, a failing version command, a setuid file,
    or the image can't copy `/opt/toolchain` with `cp`. See
    [the toolchain image contract](../specs/014-any-project/contracts/toolchain-image.md).
  - `PREFETCH_FAILED`: `FORGE_INSTALL_COMMAND` didn't finish on your unchanged code. Usually a
    package host is missing from `FORGE_REGISTRY_HOSTS`, the command is wrong, or a hidden test
    needs a package the project doesn't declare.
- **Fix:** read `<run>/toolchain.json` or `<run>/prefetch.json` (the install's last output) in the
  evidence. Add the missing registry host, correct the command, or add the dependency to the
  project while preparing the item.
- **Confirm:** a new run gets past `prepare` to `baseline`.
- **Source:** `internal/grader/item.go` (`ProbeToolchain`, `Prefetch`).

<a id="item-registry-check"></a>
### Registry check fails: `FAIL P06r`

- **Symptom:** during an attempt on a target with `FORGE_REGISTRY_HOSTS`, the run prints
  `FAIL P06r` with a detail such as `registry=http=000`, `undeclared=http=200` or
  `unexpected=1`.
- **Cause:** the probe pod, which has the worker's network policy, saw something other than
  what the profile declares:
  - `probe_ready=false`: the probe pod didn't start; resume with `-vv` to see its pod events;
  - `registry=http=000`: the first declared registry couldn't be reached through the target's
    proxy. Usually the host is misspelled, or the registry is down;
  - `undeclared=http=` with a status: the proxy let `example.invalid` through, so its allowlist
    isn't applied. Don't run items until this is fixed;
  - `missing=` or `unexpected=` above 0: the install's proxy (`pilot-egress-r-<target>`) allows
    a different host list from the profile. The detail counts hosts and never names them.
- **Fix:** correct `FORGE_REGISTRY_HOSTS` and run again; the pilot re-creates both target
  proxies from the profile when every item run prepares. If the proxy let the undeclared host
  through, run `./chalkeus forge down` and `./chalkeus forge up`.
- **Confirm:** the next attempt prints `PASS P06r`.
- **Source:** `internal/checks/checks.go` (`RegistryProxy`), `internal/cluster/infra.go`
  (`EnsureTargetProxies`).

## Teardown

<a id="old-cluster"></a>
### Old cluster from before the rename

- **Symptom:** `./chalkeus forge up` refuses with `the kind cluster specfoundry-m1 … from before
  the rename to Chalkeus is still here`; `./chalkeus forge doctor` warns `old cluster`; or
  `./chalkeus forge down` prints the same as a note.
- **Cause:** 027 renamed the cluster to `chalkeus-m1` and the registry to `chalkeus-registry`.
  The old ones still run, and they hold the registry port, so only one can be up.
- **Fix:** run the removal command it prints:

  ```sh
  kind delete cluster --name specfoundry-m1 && docker rm -f specfoundry-registry
  ```

  Then `./chalkeus forge up`. Chalkeus never removes the old cluster for you.
- **Confirm:** `kind get clusters` lists only `chalkeus-m1` after `up`.
- **Source:** 027 research R7.

<a id="leftovers"></a>
### Leftovers after teardown: T01, T02 or T03 fails

- **Symptom:** `./chalkeus forge down` reports `FAIL T01 kind cluster deleted (present)`,
  `FAIL T02 … containers=<n>` or `FAIL T03 … containers=<n>`. The audit's AU07 then fails.
- **Cause:** a cluster or container survived deletion, often because Docker was busy or
  restarting.
- **Fix:** run `./chalkeus forge down` again. If it still fails, remove the leftovers by hand:

  ```sh
  kind delete cluster --name chalkeus-m1
  docker ps -a --filter name=chalkeus-m1 --format '{{.Names}}'     # remove each with: docker rm -f <name>
  docker rm -f chalkeus-registry
  ```

  Then run `./chalkeus forge down` once more, so the teardown checks are recorded.
- **Confirm:** T01–T03 all pass, and `./chalkeus forge audit` shows AU07 passing.
- **Source:** `internal/cluster/cluster.go` (teardown checks).
