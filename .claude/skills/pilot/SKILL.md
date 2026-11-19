---
name: "pilot"
description: "Drive the specfoundry pilot (pilot/pilot) together with a human: run, explain each pause, and hand every decision to the human. Use when asked to run, resume, check or tear down the pilot, or when a pilot command paused or failed."
---

# Working the pilot with a human

You run the commands and explain what happens. **The human makes every decision.** You never
decide for them, and you never work around a check.

The full guide is `docs/operator-guide.md`; failures are in `docs/troubleshooting.md`. Read the
matching troubleshooting entry before suggesting any fix.

## Hard rules

1. **Never approve on the human's behalf.** Only run `pilot/pilot approve …` after the human
   said yes to that exact step in this conversation, and let the harness permission prompt
   ask them. If the harness does not ask (a bypass or auto-approve mode), stop and tell the
   human that the approval would not be theirs; ask them to run it themselves. Running
   `approve` after their clear yes is your job, not a breach of this rule: the harness
   prompt is how their yes is recorded. Claude Code's auto mode and any bypass or
   auto-accept mode skip that prompt, so with those modes on, don't run `approve`.
   Before the first approval in a session, ask the human to confirm their harness has the
   ask rule (Claude Code: this repo's `.claude/settings.json`; OpenCode: their own
   `~/.config/opencode/opencode.json`, see `docs/troubleshooting.md#prompts-answer-no`).
   Never create or edit harness config files, and never read per-user ones (they can hold
   provider keys).
2. **Never write, edit, copy or delete anything under a run's `approvals/` folder**, and
   never edit `events.jsonl`, `run.json` or other evidence. An approval must come from
   `pilot approve`.
3. **Never merge a PR.** Merges happen on GitHub, by the human.
   This includes the status-change PR the pilot opens after A's merge (004): merging it is
   how the human approves the roadmap change.
3a. **Never post `/approve` or `/withdraw` comments** on a run's tracking issue, with any
   account or tool (`gh issue comment`, `gh api`, a browser). On GitHub, an approval is only the
   human's if the human writes it. If asked to, refuse, and point them to the issue link the
   pilot printed.
4. **Never print, read aloud or commit secrets:** `~/.config/specfoundry/*`, API keys, the
   GitHub token, or the model host.
5. Don't run `reset-fixture`, `down` or `publisher-token` without the human asking; `reset-fixture`
   force-resets their fixture repo.
6. **Run every command exactly as `pilot/pilot …` from the repository root.** Never use an
   absolute path, `bash pilot/pilot`, `python3 -m …` or another wrapper: the harness rules that
   make `approve` ask the human match that exact prefix.

## Commands

| Command | What it does |
|---|---|
| `pilot/pilot status [--run <id>]` | current run state (JSON); read `state` and `pending` |
| `pilot/pilot up` | stand up the local cluster (several minutes) |
| `pilot/pilot run` | start a run: gate checks, mode, task A attempts, publish A |
| `pilot/pilot targets` | list the human's configured target repositories (no secrets) |
| `pilot/pilot gate --target <name> --item NNN` | check an item of a target is prepared (no cluster); exit 10 = refused, with a code |
| `pilot/pilot run --target <name> --item NNN` | an item run of the human's own repository: readiness, mode, baseline, attempts, draft PR on the target |
| `pilot/pilot resume --run <id>` | continue a paused run |
| `pilot/pilot approve --run <id> mode\|retry\|reconcile` | record the human's yes for the one pending step |
| `pilot/pilot audit [--run <id>]` | evidence audit AU01–AU11 |
| `pilot/pilot down` | tear down and verify |

Always pass `--run <id>` once you know the run id (printed as `== run <id>`). For an item run, `resume`, `approve`
and `audit` take only `--run`; the run remembers its target. Only the human prepares a target's
items and hidden tests ([docs/targets.md](../../../docs/targets.md)); never write or read the
companion repository's hidden tests yourself.

When an item has no hidden tests (`ITEM_NO_HIDDEN_TESTS`), the human writes them, or opens a
**separate** agent session with the test-author skill (`.claude/skills/test-author/`) to draft
them. You are the session that drives runs, so you never act as the test author: never run
any `pilot/pilot author` command, never read a draft workspace or draft evidence, and never open
the companion's pull requests.

To diagnose a stuck or failing step, add `-vv` to the command (pod phases, Kubernetes events,
model-proxy refusals, the first failing grader check; stderr only, redacted). Use `-vvv` (the
agent's live activity) only when the human asks for it.

## Exit codes and what to do

| Exit | Meaning | Your move |
|---|---|---|
| 0 | OK. After `run`: A's draft PR is open | Give the PR link; ask the human to review and merge it on GitHub, then tell you |
| 1 | FAIL | Quote the reason; find it in `docs/troubleshooting.md` |
| 2 | ERROR (setup, config) | Quote the message; find it in `docs/troubleshooting.md` |
| 3 | GUARD (repo guard) | The fixture repo name is wrong; explain, don't edit config yourself |
| 4 | AWAIT_MERGE | A's PR isn't merged yet; ask the human to merge it on GitHub |
| 5 | DECLINED, or paused for mode / reconcile approval | See "Decisions" |
| 6 | AWAIT_RETRY: paused for retry approval | See "Decisions" |
| 10 | REFUSE (gate refused) | Expected for B before A is verified. For an item run, the first code names the problem: find it in `docs/troubleshooting.md` (`#item-not-ready`) and explain it; never edit the target or its profile yourself |

## Decisions

With GitHub approvals on (`PILOT_GITHUB_APPROVALS=on`), each pause also prints a tracking-issue
link. The human can reply there with `/approve <step>` instead. Tell them that option exists,
then wait. The run starts a background watcher at its first pause (`watcher started (pid N)`),
which resumes on its own when the approval or merge appears; `pilot/pilot status` shows it.
`resume` refuses while it watches; don't stop it unless the human asks. For the status change, the pilot opens a small draft PR; the human's merge is the
approval.

Without a terminal, the pilot pauses at each decision and prints
`no terminal: approve with pilot/pilot approve --run <id> <step>`. For each one:

1. **Explain it in plain words**, using what the pilot printed:
   - `mode`: the run mode (one attempt at a time, retry limit, spend cap, deadline). Saying
     yes lets it start staging workers.
   - `retry`: which task failed, the attempt number, and why (from `pilot/pilot status` or the
     attempt's `grade.json`). The first attempt of A fails on purpose (AU11 induction).
   - `reconcile`: the roadmap change it proposes (for example `001: planned → verified`) and
     that A's PR was merged with tasks at 100%. Saying yes unblocks task B.
2. **Ask the human:** "Do you approve <step>?" Wait for their answer. Silence or anything but
   a clear yes means no.
3. On yes: run `pilot/pilot approve --run <id> <step>`. The harness asks them again; that
   click is the approval. Then run `pilot/pilot resume --run <id>`.
4. On no: leave it. For a retry, the run stays paused; for the mode, nothing was started.

`approve` only accepts the step the run is waiting for, after it was shown, and each approval
is used once. If it refuses, read its message to the human; don't retry with another step.

## A normal run, start to finish

1. `pilot/pilot status`. If a run is paused, ask whether to continue it or start fresh.
2. `pilot/pilot run`. It pauses at `mode` → Decisions.
3. `resume`. Attempt A/1 runs (minutes) and fails as designed; it pauses at `retry` → Decisions.
4. `resume`. A/2 should pass; the PR is published and the run pauses for the merge.
5. The human merges on GitHub. Then `resume`: it shows the reconcile proposal and pauses at
   `reconcile` → Decisions.
6. `resume`. Task B runs and is published; the audit runs. Report `run <id> succeeded` and
   the evidence path.
7. Only if the human asks: `pilot/pilot down`, then `pilot/pilot audit --run <id>` (AU07
   needs the teardown).

The audit lists each approval's channel in AU09 and AU10: `tty` or `harness`. Report them
as they are.
