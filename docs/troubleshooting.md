# Troubleshooting

Each entry starts with what you see, then explains the cause, the fix and how to confirm the
fix. Entries are grouped by when they happen. The [operator guide](operator-guide.md) links
here from each step.

To watch a run while it happens, add `-vv` (pod phases and Kubernetes events) or `-vvv` (also
the agent's live activity) to the command, for example `pilot/pilot resume --run <id> -vv`. See
[section 12 of the guide](operator-guide.md#12-when-something-goes-wrong).

When you ask for help, never share your API key, your token, anything under
`~/.config/specfoundry/` or `~/.config/opencode/`, or your provider's hostname. See
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
| `pilot up` fails while pulling images | [Image pull fails](#image-pull-error) |
| `Calico manifest hash mismatch` | [CNI hash mismatch](#cni-hash-mismatch) |
| `tagged reference not allowed` or `not a digest reference` | [Image pinned by tag](#tag-only-image) |
| `open pilot PRs exist on …` | [Open pilot PR left over](#open-pilot-pr) |
| `A's PR is not merged yet` | [Resume before merge](#resume-not-merged) |
| An attempt fails with reason `deadline` or `spend_cap` | [Deadline or spend cap](#deadline-or-spend-cap) |
| `GitHub Actions is not permitted to create or approve pull requests` | [Actions can't open PRs](#actions-cannot-open-prs) |
| `T01`, `T02` or `T03` fails after `pilot/pilot down` | [Leftovers after teardown](#leftovers) |
| `no issue write access to … (needs Issues: read and write)` | [No issue access](#github-issues-access) |
| `github drift: N difference(s)` | [GitHub drift](#github-drift) |
| A `/approve` reply on GitHub didn't resume the run | [GitHub approval rejected](#github-approval-rejected) |
| The run fails with `STATUS_PR_EDITED`, `STATUS_PR_NOT_APPROVER` or `STATUS_PR_MERGED_BY_PUBLISHER` | [Status-change PR refused](#status-pr-refused) |
| `GitHub App: NOT OK`, or `GitHub App token: …` | [GitHub App problems](#github-app) |
| An item run refuses with `ITEM_…`, `INPUT_TOO_LARGE`, `OPEN_ITEM_PR` or `TOOLCHAIN_UNAVAILABLE` | [Item not prepared](#item-not-ready) |
| An item run fails with `BASELINE_SUITE_FAILS` or `BASELINE_HIDDEN_PASSES` | [Baseline fails](#item-baseline) |
| An item run fails with `PREPARE` or `REPLAY_MISMATCH`, or `target … needs its own publisher token` | [Item run setup](#item-setup) |
| An item run fails with `TOOLCHAIN_CONTRACT` or `PREFETCH_FAILED` | [Toolchain or dependencies](#item-toolchain) |

---

## Setup

<a id="no-api-key"></a>
### No API key file: "needs opencode.json and at least one <provider>_api_key file"

- **Symptom:**
  - `pilot/pilot up` or `run` stops with
    `~/.config/specfoundry/opencode needs opencode.json and at least one <provider>_api_key file`;
  - or the attempt's `report.json` shows the harness exited after an HTTP 401 from the model.
- **Cause:** OpenCode resolves `"apiKey": "{file:~/.config/opencode/<provider>_api_key}"`
  inside the worker. If the key file is missing, empty or wrongly named, the provider rejects
  the request.
- **Fix:** if OpenCode works on your machine, run `pilot/pilot import-opencode --force` to copy
  its setup again. Otherwise:
  1. Put the key, on one line, into `~/.config/specfoundry/opencode/<provider>_api_key`.
  2. Make sure `<provider>` matches the name in the `{file:...}` reference.
  3. Set modes: `chmod 700 ~/.config/specfoundry/opencode` and `chmod 600` on each file inside.
- **Confirm:** `pilot/pilot up` passes its config checks. The next attempt's `report.json`
  shows a harness exit of 0 and a non-zero token count.
- **Source:** the 002 live run setup (OpenCode returned 401 until the key file existed).

<a id="token-writes-project"></a>
### Token can write too much: "token can write …: scope it to the fixture repo only"

- **Symptom:** `pilot/pilot publisher-token` prints `publisher token: NOT OK`,
  `canary_push: True`, this problem line, and `not saved: …`. The token was thrown away.
- **Cause:** the token can push to the specfoundry repository you cloned from (the
  "canary"). That usually means a classic token, or a fine-grained token with *All
  repositories* access. The check is a real write test, not GitHub's permission labels, which
  describe your account rather than the token on public repositories.
- **Fix:** delete the token on GitHub. Create a fine-grained token with **Only select
  repositories** set to your fixture repo
  ([guide section 6](operator-guide.md#6-prepare-a-throwaway-fixture-repo-and-a-scoped-token)),
  then run `pilot/pilot publisher-token` again.
- **Confirm:** the output shows `repo_push: True` and `canary_push: False`.
- **Source:** commit 7570552 (the write probe replaced permission labels after the repo went
  public).

<a id="token-expired"></a>
### Token expired or revoked: "token rejected by GitHub (HTTP 401)"

- **Symptom:** `pilot/pilot publisher-token --check-only` (also run by `up`) prints
  `publisher token: NOT OK` and `problem: token rejected by GitHub (HTTP 401)`.
- **Cause:** the fine-grained token has expired, or it was deleted or regenerated on
  GitHub. The last check's `expires:` line shows when it would expire.
- **Fix:** generate a new token with the same settings, then run `pilot/pilot publisher-token`
  and paste it at the hidden prompt. The old `publisher.env` is replaced.
- **Confirm:** `pilot/pilot publisher-token --check-only` prints `publisher token: OK` with a
  future `expires:` date.
- **Source:** `pilot/images/tools/check_token.py`.

<a id="actions-cannot-open-prs"></a>
### Actions can't open PRs: "GitHub Actions is not permitted to create or approve pull requests"

This only affects you if you **fork specfoundry and run its image workflow** yourself.
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
### Image pull fails during `pilot up`

- **Symptom:** `pilot/pilot up` stops after `== images`, with a Docker pull error such as
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
  4. Run `pilot/pilot up` again.
- **Confirm:** the output reaches `== images: pulled by digest from pilot/images.lock` and
  `== up complete`.
- **Source:** 003 validation run `20261113T180203Z` (pull by digest).

<a id="cni-hash-mismatch"></a>
### CNI hash mismatch: "Calico manifest hash mismatch"

- **Symptom:** `pilot/pilot up` stops with
  `Calico manifest hash mismatch: expected <pinned>, got <actual>` and exits 2.
  Nothing has been applied to the cluster.
- **Cause:** the downloaded Calico manifest differs from the one pinned in
  `pilot/pins.env` (`CALICO_SHA256`). Either `pins.env` was edited, or the download was
  altered or truncated. The pilot never applies an unverified network plugin.
- **Fix:**
  1. Restore the pinned values: `git checkout pilot/pins.env`.
  2. Run `pilot/pilot down`, then `pilot/pilot up`.
  3. If it still fails on an untouched checkout, your network is altering the download.
     Don't work around it; open an issue.
- **Confirm:** `pilot/pilot up` completes, and the run's `versions.txt` lists `calico: <version>
  sha256=<pinned value>`.
- **Source:** 003 negative check (a wrong `CALICO_SHA256` exits 2 before applying anything).

<a id="tag-only-image"></a>
### Image pinned by tag: "tagged reference not allowed" or "not a digest reference"

- **Symptom:** `pilot/pilot up` or `run` stops with, for example,
  `WORKER_IMAGE in pilot/images.lock: tagged reference not allowed: '…:latest@sha256:…'`, or
  `not a digest reference`, and exits 2.
- **Cause:** every image must be referenced as `name@sha256:<64 hex>`, with no tag. A tag can
  be moved to different content later; a digest can't.
- **Fix:** restore the lock with `git checkout pilot/images.lock`. If you're mirroring images,
  write the mirror's `name@sha256:…` reference, which has the same digest (see
  [enterprise environments](harnesses.md#enterprise-environments)).
- **Confirm:** `pilot/pilot up` completes. Live check P10 passes during the run (every image
  pinned and non-root).
- **Source:** 003 negative check (a tag-only `WORKER_IMAGE` exits 2 before applying anything).

---

## Run time

<a id="prompts-answer-no"></a>
### Prompts answer "no": no question appears

- **Symptom:**
  - `pilot/pilot run` pauses at the mode prompt (exit 5) without asking, and prints
    `paused: mode not confirmed. no terminal: approve with ...`; or
  - it pauses at the retry (exit 6) without asking; or
  - `resume` prints `waiting for your approval of the status change` (exit 5) without asking.
- **Cause:** prompts are read from your terminal. Without one (an agent's shell tool, an IDE
  task runner, CI, `nohup`, or piped input), nothing is assumed: the run pauses at the
  decision. This is deliberate: nothing proceeds without a human answer.
- **Fix:** either
  - run `pilot/pilot resume --run <run-id>` in a real interactive terminal, for example a plain
    SSH session, and answer there; or
  - when you work with an agent, approve the one pending step with
    `pilot/pilot approve --run <run-id> mode|retry|reconcile`. Your agent harness must ask
    **you** before that command runs; say yes only if you agree. Then run
    `pilot/pilot resume --run <run-id>`. See the `pilot` skill
    (`.claude/skills/pilot/SKILL.md`).
    - **Claude Code:** this repo's `.claude/settings.json` already has the ask rule.
    - **OpenCode:** add the rule once to your own config, `~/.config/opencode/opencode.json`,
      so it covers every project (the repo-root `opencode.json` is gitignored):

      ```json
      "permission": { "bash": {
        "pilot/pilot approve*": "ask",
        "./pilot/pilot approve*": "ask"
      } }
      ```
    - **Any other harness:** no rule is provided. Use a real terminal instead.
    - Keep the harness in its normal permission mode. Claude Code's auto mode, and any bypass
      or auto-accept mode, run `approve` without asking you, so the approval isn't really yours.

  Merging is never approved this way: merge A's PR on GitHub yourself.
- **Confirm:** `pilot/pilot status` shows the run moving on. In the audit, AU09 and AU10 list
  each approval's channel: `tty` (terminal) or `harness` (`pilot approve`).
- **Source:** the 002 live run (prompts read the controlling terminal) and the 006 validation
  (an agent's shell has no terminal; finding F4).

<a id="open-pilot-pr"></a>
### Open pilot PR left over: "open pilot PRs exist on …"

- **Symptom:** `pilot/pilot run` prints
  `open pilot PRs exist on <you>/<you>-pilot-fixture: [...]` and `run pilot reset-fixture first`,
  then exits 2.
- **Cause:** a previous run left draft PRs open. A new run won't start on top of them, so
  results can't mix.
- **Fix:** `pilot/pilot reset-fixture`, then type `y`. This force-resets the fixture repo and
  closes open `pilot/*` PRs.
- **Confirm:** `pilot/pilot run` gets past this check and shows the gate self-checks.
- **Source:** `internal/pipeline/pipeline.go` (pre-run check).

<a id="deadline-or-spend-cap"></a>
### Deadline or spend cap: an attempt fails with reason `deadline` or `spend_cap`

- **Symptom:** an attempt fails before grading. In `summary.md` its grade shows
  `fail (deadline)` or `fail (spend_cap)`. You're asked to confirm a retry, or the run fails
  with `<task>_RETRIES_EXHAUSTED` after the last retry.
- **Cause:**
  - **`deadline`:** the worker ran past its 20-minute limit. This usually means a slow or
    overloaded model endpoint.
  - **`spend_cap`:** the attempt reached `PILOT_SPEND_CAP_USD`, or `PILOT_TOKEN_CAP` when
    the provider reports no cost, and the watchdog stopped it.
- **Fix:**
  - Confirm the retry if the cause looks temporary.
  - For repeated `deadline` failures, try a faster model.
  - For repeated `spend_cap` failures, check the attempt's token count in `summary.md`. A
    reference attempt uses roughly 225,000–480,000 tokens. Raise the cap in
    `~/.config/specfoundry/pilot.env` only if you accept the cost.
  - After `RETRIES_EXHAUSTED`, start over with `pilot/pilot reset-fixture` and then
    `pilot/pilot run`.
- **Confirm:** a later attempt reaches grading. AU06 passes in the audit.
- **Source:** `internal/worker/worker.go` (failure reasons).

<a id="cost-zero"></a>
### Cost shows 0: `Cost (USD)` is `0.0` for every attempt

- **Symptom:** in `summary.md`, every attempt shows `0.0` cost but a large token count.
- **Cause:** your provider, like many OpenAI-compatible gateways, doesn't report a price.
  Nothing is wrong: the spend cap falls back to `PILOT_TOKEN_CAP` (2,000,000 tokens per
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

- **Symptom:** `pilot/pilot resume` prints
  `A's PR is not merged yet. Merge it on GitHub, then run pilot resume again.` and exits 4.
- **Cause:** reconcile only proposes `A verified` after A's PR has actually merged. Nothing
  was changed.
- **Fix:** open the PR link printed by `run` (or check `pilot/pilot status`). Click **Ready for
  review**, then **Merge**, then run `pilot/pilot resume`.
- **Confirm:** `resume` shows the reconcile proposal and asks
  `Apply '001: planned → verified' to the fixture roadmap? [y/N]`.
- **Source:** `internal/pipeline/pipeline.go` (resume check).

---

## GitHub loop

These entries apply only with `PILOT_GITHUB_ISSUES=on` or `PILOT_GITHUB_APPROVALS=on`. See
[the GitHub loop](github-loop.md).

<a id="github-issues-access"></a>
### No issue access: "needs Issues: read and write"

- **Symptom:** `pilot/pilot publisher-token --check-only` (or `app-token --check`) reports
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

- **Symptom:** `pilot/pilot github drift` exits 1, or a run's `github/drift.json` isn't empty.
- **Cause:** something on GitHub disagrees with the roadmap:
  - `missing_issue`: no issue for an item;
  - `label_mismatch`: a status label was changed by hand;
  - `open_but_verified` / `closed_but_not_verified`: an issue was closed or reopened by hand;
  - `pr_without_closes`: a task PR lost its `Closes #N`;
  - `orphan_issue` / `duplicate_issue`: issues for items no longer in the roadmap, or two
    issues for one item.
- **Fix:** run `pilot/pilot github sync`. It recreates missing issues and restores labels and
  bodies. It never closes, reopens or deletes. Fix the rest on GitHub by hand: close an orphan
  or duplicate, or reopen an issue whose item isn't verified.
- **Confirm:** `pilot/pilot github drift` exits 0.
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
  | `not_an_approver` | your login isn't in `PILOT_APPROVERS` |
  | `no_write_access` | your account no longer has write, maintain or admin on the fixture repo |
  | `publisher_identity` | posted with the publisher's account (in token mode, that's your own account through the token) |
  | `app_authored` | posted by a GitHub App or bot |
  | `edited` | the comment was edited after posting |
  | `withdrawn` | you posted `/withdraw <step>` afterwards |
  | `duplicate` | an earlier valid approval was used instead |
- **Fix:** post a new reply that is exactly `/approve <step>`, as yourself, below the latest
  decision post. If `publisher_identity` comes up in token mode, approve from the GitHub web UI
  rather than a tool using the token, or switch to a GitHub App.
- **Confirm:** `pilot/pilot github watch --run <id>` resumes.
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
  - **`STATUS_PR_NOT_APPROVER`:** it was merged by someone not in `PILOT_APPROVERS`;
  - **`STATUS_PR_MERGED_BY_PUBLISHER`:** it was merged with the publisher's account;
  - **`STATUS_NOT_ON_DEFAULT`:** the default branch doesn't show the item `verified`.
- **Fix:** the run can't continue, because the roadmap may now hold an unapproved change.
  Run `pilot/pilot reset-fixture`, then start a new run. Next time, merge the status-change PR
  unedited, from an approver's account.
- **Confirm:** the new run's `reconcile/A-status.json` shows `"ok": true`.
- **Source:** `internal/reconcile/reconcile.go` (`CheckStatusPR`).

<a id="github-app"></a>
### GitHub App problems

- **Symptom:** `pilot/pilot app-token --check` prints `GitHub App: NOT OK`, or a run stops with
  `GitHub App token: …`.
- **Cause and fix:**
  - **`cannot sign with the App key`:** the file isn't the App's private key. Generate a new
    one on the App's page, and store it with `pilot/pilot app-key`.
  - **`GitHub rejected the App JWT`:** `PILOT_APP_ID` is wrong, or the key belongs to another
    App.
  - **`cannot mint an installation token`:** `PILOT_APP_INSTALLATION_ID` is wrong, or the App
    isn't installed on the fixture repo.
  - **`the App lacks …: write`:** add the permission on the App's page, then accept it on the
    installation.
  - **`token covers […]`:** the installation includes more repositories. Limit it to the
    fixture repo.
  - **`both a publisher token and a GitHub App key exist`:** set `PILOT_PUBLISHER_CREDENTIAL`
    to `token` or `app`.
- **Confirm:** `pilot/pilot app-token --check` prints `OK`.
- **Source:** `internal/ghapp/ghapp.go`; see
  [github-app.md](github-app.md).

---

## Your own repositories (targets)

<a id="item-not-ready"></a>
### Item not prepared: the run refuses with exit 10

- **Symptom:** `pilot/pilot run --target <name> --item NNN` prints `readiness NNN: refuse [...]`
  and `item NNN is not prepared`, then exits 10. No worker starts.
- **Cause:** one of the readiness rules failed. Its code is first in the list.
- **Fix:** see the table in [targets](targets.md#3-check-that-an-item-is-ready). Each code names
  what to add or change in the target or its companion repository. `TOOLCHAIN_UNAVAILABLE`
  means the profile names a toolchain with no published image: bring your own image or ask for
  one ([targets](targets.md#your-own-toolchain-image)).
- **Confirm:** `pilot/pilot gate --target <name> --item NNN` prints `"decision": "allow"`.
- **Source:** `internal/readiness/readiness.go`.

<a id="item-baseline"></a>
### Baseline fails: `BASELINE_SUITE_FAILS` or `BASELINE_HIDDEN_PASSES`

- **Symptom:** after you approve the mode, the run prints `baseline for item NNN failed` and
  fails. No worker has run, so no model spend was used.
- **Cause:** on the unchanged code, either:
  - the project's own test command fails (`BASELINE_SUITE_FAILS`), so a change can't be judged
    against it; or
  - the hidden tests already pass (`BASELINE_HIDDEN_PASSES`), so they can't tell a correct change
    from no change.
- **Fix:** for the first, make `PILOT_TEST_COMMAND` pass on the target's branch. For the second,
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
    in `~/.config/specfoundry/targets/<name>.publisher.env` as `PILOT_SCM_TOKEN=…`, mode 0600. Or
    use the GitHub App, installed on the target.
- **Confirm:** the run gets past the step that failed.
- **Source:** `internal/pipeline/item.go`, `internal/fixture/snapshot.go`, `internal/cli/run.go`.

<a id="item-toolchain"></a>
### Toolchain or dependencies: `TOOLCHAIN_CONTRACT` or `PREFETCH_FAILED`

- **Symptom:** after the mode approval, the run fails with `TOOLCHAIN_CONTRACT: …` or
  `PREFETCH_FAILED: …`, before any worker starts.
- **Cause:**
  - `TOOLCHAIN_CONTRACT`: the toolchain image failed a check. The message names it: no
    `/opt/toolchain/toolchain.json`, a rule it breaks, a failing version command, a setuid file,
    or the image can't copy `/opt/toolchain` with `cp`. See
    [the toolchain image contract](../specs/014-any-project/contracts/toolchain-image.md).
  - `PREFETCH_FAILED`: `PILOT_INSTALL_COMMAND` didn't finish on your unchanged code. Usually a
    package host is missing from `PILOT_REGISTRY_HOSTS`, the command is wrong, or a hidden test
    needs a package the project doesn't declare.
- **Fix:** read `<run>/toolchain.json` or `<run>/prefetch.json` (the install's last output) in the
  evidence. Add the missing registry host, correct the command, or add the dependency to the
  project while preparing the item.
- **Confirm:** a new run gets past `prepare` to `baseline`.
- **Source:** `internal/grader/item.go` (`ProbeToolchain`, `Prefetch`).

## Teardown

<a id="leftovers"></a>
### Leftovers after teardown: T01, T02 or T03 fails

- **Symptom:** `pilot/pilot down` reports `FAIL T01 kind cluster deleted (present)`,
  `FAIL T02 … containers=<n>` or `FAIL T03 … containers=<n>`. The audit's AU07 then fails.
- **Cause:** a cluster or container survived deletion, often because Docker was busy or
  restarting.
- **Fix:** run `pilot/pilot down` again. If it still fails, remove the leftovers by hand:

  ```sh
  kind delete cluster --name specfoundry-m1
  docker ps -a --filter name=specfoundry-m1 --format '{{.Names}}'     # remove each with: docker rm -f <name>
  docker rm -f specfoundry-registry
  ```

  Then run `pilot/pilot down` once more, so the teardown checks are recorded.
- **Confirm:** T01–T03 all pass, and `pilot/pilot audit` shows AU07 passing.
- **Source:** `internal/cluster/cluster.go` (teardown checks).
