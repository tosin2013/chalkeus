# Research: Working with an agent — pilot skill and harness approvals

No NEEDS CLARIFICATION items remained after the spec. The decisions below record why the
first cut (PR #11) is shaped as it is, and what this plan changes.

## R1 — How a human's yes reaches a headless run

- **Decision:** a separate command, `pilot approve --run <id> <step>`, which the agent harness
  always asks the human about before it runs. The human's answer in the harness permission
  prompt is the approval.
- **Rationale:** the harness, not the agent, collects the yes, so the agent can't make the
  decision itself without defeating a control the human configured. It works in both
  supported harnesses with no new infrastructure, and it leaves the terminal path untouched.
- **Alternatives considered:**
  - **`--yes` flag on `run`/`resume`:** rejected (006 F4). The agent could pass it itself, and
    AU09/AU10 would attest to a human decision that never happened.
  - **The agent relays "the user said yes" in chat:** rejected for the same reason: the pilot
    can't tell a relayed yes from a self-made one.
  - **A one-time code shown only to the human:** there's no channel the agent can't read on
    the same machine (no terminal; stdout and files are visible to it).
  - **GitHub approvals:** strongest (the human's GitHub identity), but they need the 004
    GitHub loop. Deferred to 004, which reuses this approval record and adds a `github`
    channel.

## R2 — What an approval is bound to

- **Decision:**
  - one pending approval per run (`approvals/pending.json`), accepted only for the step the
    run is paused at, after that step was shown;
  - bound to task and attempt number (retry), or to the merge commit of the shown proposal
    (status change);
  - deleted when read, whether it matched or not.
- **Rationale:** an approval must mean "I saw this decision and said yes to it". Binding
  prevents approving ahead of time, reusing an approval for a later decision, or approving a
  proposal the human never saw.
- **Alternatives considered:**
  - a list of approvals, which allows pre-approving several steps (rejected);
  - approvals without binding, where a retry approval could carry over to the next attempt
    (rejected).

## R3 — Harness rule syntax and its limits

- **Decision:**
  - **Claude Code**, in `.claude/settings.json`: `permissions.ask` with
    `Bash(pilot/pilot approve:*)`, `Bash(./pilot/pilot approve:*)` and
    `Bash(python3 -m specfoundry_pilot approve:*)`. `permissions.deny` blocks file edits under
    `**/approvals/**`.
  - **OpenCode**, in `opencode.json`: `permission.bash` patterns
    `pilot/pilot approve*`, `./pilot/pilot approve*` and
    `python3 -m specfoundry_pilot approve*`, each set to `ask`.
- **Rationale:** both harnesses support per-command "ask" rules that the project can commit.
  That keeps the control in the repository, so every operator gets it by cloning.
- **Limits (documented, not hidden):**
  - prefix rules don't catch every way to invoke the program, for example an absolute path;
  - a bypass or auto-approve mode skips prompts;
  - an agent with shell access could write a file by other means.

  The skill forbids all three, and the audit labels these approvals `harness`, never `tty`.
  The owner check (G5) confirms the rules prompt in practice, and corrects their syntax if
  either harness reads them differently.
- **Alternatives considered:**
  - signing approvals with a local key: the agent can read the key on the same machine, so it
    adds nothing;
  - refusing to run without a terminal: that's the status quo the owner asked to change.

## R4 — Recording the channel without breaking old evidence

- **Decision:** a separate `approval_channel` event per decision
  (`step=<s> approved=<true|false> channel=<tty|harness|none>`). AU09/AU10 keep their pass
  rules and add `channels=[…]` to their detail.
- **Rationale:** the existing `mode_confirmed` and `retry_confirmed` events are unchanged, so
  every committed record from 002, 003 and 006 still passes. Records from before 012 show no
  channel instead of a guessed one.
- **Alternatives considered:**
  - changing the `mode_confirmed` detail format: that breaks parsers of older records;
  - failing AU09/AU10 when no channel is recorded: that would fail valid historical evidence.

## R5 — A terminal answer overrides a pending approval (G1)

- **Decision:** when a decision is answered at a terminal, any pending approval for that run is
  deleted, whatever the answer.
- **Rationale:** if the human approved through the harness and then said "no" at a terminal,
  the later "no" is the current decision. Without this, a later headless `resume` could use
  the earlier yes.
- **Alternative considered:** keeping the approval for the case where the terminal answer is
  yes. That's harmless but adds a rule for no benefit; one pending decision, one answer.

## R6 — Proving old evidence still passes (G4)

- **Decision:** a unit test runs the read-only `audit_dir` on every committed run record whose
  `run.json` state is `succeeded` (002 `20261113T030655Z`, 003 `20261113T180203Z`, 006
  `20261114T160532Z`, plus any added later). It requires every row to pass and AU09/AU10 to
  report no channels.
- **Rationale:** SC-004 then becomes a standing regression check rather than a one-off.
  Failed runs (002 `20261113T030451Z`) are excluded, because they're evidence of failure by
  design.
- **Alternative considered:** a manual `pilot audit --run` for each record. It's not
  repeatable, and it misses records added later.
