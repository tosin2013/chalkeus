# The GitHub loop: issues, labels and approvals on GitHub

The GitHub loop is optional, and off by default. With it on, the pilot mirrors the fixture
roadmap on GitHub, and you can make the run's decisions there instead of at a terminal. The
roadmap stays the source of truth: GitHub shows a copy, and the pilot puts that copy back
whenever it drifts.

**Contents:**
1. [What you get](#1-what-you-get)
2. [Turn it on](#2-turn-it-on)
3. [Approve on GitHub](#3-approve-on-github)
4. [The status-change PR](#4-the-status-change-pr)
5. [Who an approval proves](#5-who-an-approval-proves)
6. [Check for drift](#6-check-for-drift)

## 1. What you get

- **One issue per roadmap item.** Its body shows the item's description, outcome and task
  checklist, and it carries exactly one `status:<status>` label. The pilot creates and updates
  these issues; it never deletes, closes or reopens them.
- **Linked PRs.** Each task PR ends with `Closes #<issue>`, so merging it closes the item's
  issue.
- **A drift report.** It lists anything on GitHub that disagrees with the roadmap.
- **With approvals on, also:**
  - one tracking issue per run, where the pilot posts each decision it waits for. When the run
    ends (succeeded or failed), the pilot posts the outcome there and closes the issue; a paused
    run keeps it open;
  - a status-change PR after A's merge; your merge of it approves the roadmap change.

## 2. Turn it on

1. **Give the publisher credential Issues access.** For the fine-grained token, edit it on
   GitHub and add *Issues: read and write*, next to *Contents* and *Pull requests*. Then run
   `./chalkeus forge publisher-token --check-only`. For a GitHub App, see
   [github-app.md](github-app.md).
2. **Set the switches** in `~/.config/chalkeus/forge.env`:

   ```sh
   FORGE_GITHUB_ISSUES=on        # issues, labels and linked PRs
   FORGE_GITHUB_APPROVALS=on     # approvals on GitHub (turns issues on too)
   FORGE_APPROVERS=your-login    # comma-separated GitHub logins allowed to approve
   ```
3. **Sync once** to see the issues: `./chalkeus forge github sync`. Running it again changes nothing.

> **The publisher can't approve.** Comments and merges made with the publisher's identity never
> count, so an agent holding the credential can't approve its own work. If your fine-grained
> token is from your own account, your `/approve` is refused as `publisher_identity` and your
> merge of the status-change PR as `STATUS_PR_MERGED_BY_PUBLISHER`. For approvals on GitHub,
> publish with a GitHub App ([github-app.md](github-app.md)), or with a token from a second
> account that isn't in `FORGE_APPROVERS`.

## 3. Approve on GitHub

Start the run from any shell:

```sh
./chalkeus forge run
```

At its first pause, the run starts a **watcher** in the background and prints
`watcher started (pid N); you can close this terminal. Log: <path>`. The watcher checks GitHub
every 15 seconds and resumes the run when your approval or the merge appears, under exactly the
same rules as `./chalkeus forge resume`. You can close the terminal. There is one watcher per run,
and it exits when the run ends; `./chalkeus forge status` shows it while it runs. While it runs,
`./chalkeus forge resume` refuses; stop the watcher first to resume by hand.

```sh
./chalkeus forge github watch --run <run-id> --stop     # stop the watcher; the run stays paused
./chalkeus forge github watch --run <run-id> --detach   # start it again
./chalkeus forge github watch --run <run-id>            # or watch in the foreground (Ctrl-C stops)
```

The watcher doesn't survive a reboot. The run stays paused; `./chalkeus forge resume` or a new
`github watch` picks it up.

At each decision, the run prints a link to its tracking issue. The decision post there says
what is being decided, and gives the exact reply:

```text
/approve mode        # start staging workers in the mode shown
/approve retry       # retry the failed task (the post names the task and attempt)
```

A reply counts only if all of these hold:
- its first line is exactly that command;
- it comes after the decision post, and was never edited;
- its author is in `FORGE_APPROVERS` and still has write access to the fixture repo;
- it was not made with the publisher credential, or by a GitHub App.

`/withdraw retry` cancels your approval, as long as the run hasn't resumed yet. Every reply the
pilot looked at is logged in the run's `github/approvals.jsonl`, with the reason it was accepted
or rejected.

The bot answers each `/approve` once, on the tracking issue, so everyone watching the issue is
notified:
- `` ✅ `/approve retry` from @you accepted (comment N); the run is resuming. ``
- `` ❌ `/approve retry` from @someone not accepted (`not_an_approver`): @someone is not in
  FORGE_APPROVERS. The run stays paused. ``

With a terminal attached, the terminal decides, and the pilot notes that under the post.

## 4. The status-change PR

After you merge A's PR, the pilot checks it as before: merged, tasks at 100%, and the merged
files match the graded attempt. It then opens a small draft PR that changes exactly one line:
item 001 from `planned` to `verified`.

- **Merging it is the approval.** Run `./chalkeus forge resume`, or let `github watch` pick it up.
- The pilot checks that the merged PR is exactly that one-line change, and that an allowed
  approver merged it. If the PR was edited, or merged by the publisher's account or by someone
  not in `FORGE_APPROVERS`, the run stops with the reason.
- The pilot never merges anything.

## 5. Who an approval proves

An approval on GitHub proves that **an account** approved. Anyone who can act as that account
can approve:

- **Token mode:** the fine-grained token belongs to your account. An agent on your machine may
  also hold your `gh` login, and could comment as you. The pilot ignores comments made with the
  publisher token, and the project's agent rules forbid posting `/approve`. A GitHub App as the
  publisher keeps the bot and you apart; see [github-app.md](github-app.md).
- **App mode:** the pilot's posts and PRs are the bot's. Your reply needs your own credentials,
  which the pilot never holds.

The audit shows each decision's channel (`tty`, `harness` or `github`) and, for GitHub, who
approved. A reviewer can weigh them.

## 6. Check for drift

```sh
./chalkeus forge github drift                 # exit 0 no drift, 1 drift found
./chalkeus forge github drift --run <run-id>  # also saves github/drift.json in the run's evidence
```

A completed run saves its drift report, and it should be empty. The fixes are in
[troubleshooting](troubleshooting.md#github-drift).
