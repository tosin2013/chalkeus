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
  - one tracking issue per run, where the pilot posts each decision it waits for;
  - a status-change PR after A's merge; your merge of it approves the roadmap change.

## 2. Turn it on

1. **Give the publisher credential Issues access.** For the fine-grained token, edit it on
   GitHub and add *Issues: read and write*, next to *Contents* and *Pull requests*. Then run
   `pilot/pilot publisher-token --check-only`. For a GitHub App, see
   [github-app.md](github-app.md).
2. **Set the switches** in `~/.config/specfoundry/pilot.env`:

   ```sh
   PILOT_GITHUB_ISSUES=on        # issues, labels and linked PRs
   PILOT_GITHUB_APPROVALS=on     # approvals on GitHub (turns issues on too)
   PILOT_APPROVERS=your-login    # comma-separated GitHub logins allowed to approve
   ```
3. **Sync once** to see the issues: `pilot/pilot github sync`. Running it again changes nothing.

## 3. Approve on GitHub

Start the run from any shell:

```sh
pilot/pilot run
```

At its first pause, the run starts a **watcher** in the background and prints
`watcher started (pid N); you can close this terminal. Log: <path>`. The watcher checks GitHub
every 15 seconds and resumes the run when your approval or the merge appears, under exactly the
same rules as `pilot/pilot resume`. You can close the terminal. There is one watcher per run,
and it exits when the run ends; `pilot/pilot status` shows it while it runs. While it runs,
`pilot/pilot resume` refuses; stop the watcher first to resume by hand.

```sh
pilot/pilot github watch --run <run-id> --stop     # stop the watcher; the run stays paused
pilot/pilot github watch --run <run-id> --detach   # start it again
pilot/pilot github watch --run <run-id>            # or watch in the foreground (Ctrl-C stops)
```

The watcher doesn't survive a reboot. The run stays paused; `pilot/pilot resume` or a new
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
- its author is in `PILOT_APPROVERS` and still has write access to the fixture repo;
- it was not made with the publisher credential, or by a GitHub App.

`/withdraw retry` cancels your approval, as long as the run hasn't resumed yet. Every reply the
pilot looked at is logged in the run's `github/approvals.jsonl`, with the reason it was accepted
or rejected.

With a terminal attached, the terminal decides, and the pilot notes that under the post.

## 4. The status-change PR

After you merge A's PR, the pilot checks it as before: merged, tasks at 100%, and the merged
files match the graded attempt. It then opens a small draft PR that changes exactly one line:
item 001 from `planned` to `verified`.

- **Merging it is the approval.** Run `pilot/pilot resume`, or let `github watch` pick it up.
- The pilot checks that the merged PR is exactly that one-line change, and that an allowed
  approver merged it. If the PR was edited, or merged by the publisher's account or by someone
  not in `PILOT_APPROVERS`, the run stops with the reason.
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
pilot/pilot github drift                 # exit 0 no drift, 1 drift found
pilot/pilot github drift --run <run-id>  # also saves github/drift.json in the run's evidence
```

A completed run saves its drift report, and it should be empty. The fixes are in
[troubleshooting](troubleshooting.md#github-drift).
