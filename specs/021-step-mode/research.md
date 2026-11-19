# Research: Step mode, the laptop manager loop

## R1 — A reconcile loop with a pure decision

- **Decision:**
  - A new Go package, `step` (added by this feature), has a pure function `Next(ItemState) Action`, the state
    machine of FR-004.
  - A gatherer builds `ItemState` from the roadmap, GitHub and the evidence.
  - A loop runs gather, `Next`, act, then sleeps.
- **Rationale:** the decisions can be tested without a cluster or GitHub. The operator (011)
  can reuse `Next` unchanged, which is the "like an operator" the owner asked for.

## R2 — The proposal label

- **Decision:** read the issue's events (`GET /repos/{o}/{r}/issues/{n}/events`). The item is
  proposed when the label is on the issue now and the **last** `labeled` event for that label
  was by an approver (`PILOT_APPROVERS`).
- **Rationale:** labels carry no author, but the events API records who added each one. A
  label re-added by a non-approver after an approver's doesn't count.
- **Label name:** `PILOT_PROPOSE_LABEL` in the target profile, default "pilot/go" until Q7.

## R3 — Delegation, not reimplementation

- **Drafting:** `author.Pass` from 024, restricted to the item (a new `WatchEnv.Only`).
- **Doctor and run:** the pilot binary itself, as a subprocess:
  - `doctor --target --item --baseline`;
  - `run --target --item` with `PILOT_STARTED_BY=step`. The run records that, pauses for
    `/approve mode`, and starts its own detached watcher, exactly as when typed.

  Output goes to `step/<target>/<item>/<n>-doctor.log` or `<n>-run.log`.
- **Rationale:** the run path stays the one that is audited. The manager adds no second way to
  start work.

## R4 — State without a store (SC-003)

An item's state comes from four sources:
- **readiness** (the author's rules): ready, hidden files present or not;
- **open companion PRs** with the head prefix `author/<NNN>-`;
- **runs:** the newest item run of this target and item in the evidence root (from
  `run.json`). Its state is paused, succeeded or failed;
- **`step/<target>/decisions.jsonl`**: only to post each comment once, keyed by the item and
  what it is about (for example `doctor-failed@<target commit>:<companion commit>`). It also
  lists the run folders the manager started, so a changed active feature doesn't hide them.

## R4a — Item issues

Each pass first runs 004's issue sync for the target, the same sync a run does at its start. A
newly planned item then has an issue to label before any run exists.

## R5 — Comments

The manager comments on the item's issue (004's `ByItem` mapping), with the publisher identity
the run loop uses (`GHWriter`). Each comment:
- starts with an HTML marker `<!-- pilot-step:<kind> -->`;
- has one short line plus the findings (doctor's FAIL and WARN lines only, which never carry
  secrets);
- is posted once (R4).

## R6 — Verbosity and who starts it

- `pilot step` defaults to `-vv`, as `author watch` does since #114.
- The pilot skill gets one rule: the human starts `pilot step` in their own terminal (it may
  draft tests). The driving session never does.
