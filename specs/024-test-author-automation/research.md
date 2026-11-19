# Research: Test-author automation

Decisions for [plan.md](plan.md). Each one reuses what 014 and 018 already built, where it can.

## R1 — The watcher

- **Decision:** `pilot author watch --target <name>` is a long-running, foreground pilot
  process. Every `PILOT_AUTHOR_INTERVAL_S` seconds (default 120) it:
  1. reads the target's roadmap with `spec status` on a fresh clone;
  2. for each active item that isn't verified and whose dependencies are satisfied, in roadmap
     order, runs the author's readiness (`readiness.Check` with `SkipHidden`, as
     `author start` does);
  3. drafts the first eligible item (R2), then polls again.

  It refuses to start unless the target sets `PILOT_AUTHOR_AUTO=on`.
- **Rationale:** it's the same shape as `pilot github watch`, so operators already know it. One
  draft at a time keeps cluster load and token use predictable.
- **Alternatives:** a GitHub Actions workflow (later; it needs the model key as a repository
  secret); running inside `github watch` (rejected: a run's watcher must stay about that run).

## R2 — Eligibility

- **Decision:** an item is eligible when all of these hold:
  - the author's readiness says `allow`;
  - `hidden_files` is 0;
  - no open companion PR has the head prefix `author/<NNN>-`;
  - no earlier automatic draft exists for the same item and spec SHA-256, whatever its
    result.
- **Rationale:** this enforces SC-003, so an item is never drafted twice for the same spec
  version. It also stops a failing item from looping (FR-007). A spec change makes the item
  eligible again.

## R3 — The author Job reuses the worker image

- **Decision:** the Job runs the **worker image** (OpenCode 2.2.7, the spend watchdog, framed
  output), with `meta.json` `role: "author"`. In that role the entrypoint:
  - unpacks the staged author workspace into `/work/task`: `spec.md`, `plan.md`, `tasks.md`,
    `criteria.json`, a read-only `base/`, and an empty `draft/`;
  - runs OpenCode with the author prompt (R4);
  - emits `report.json` plus `draft.tar` (only `draft/`) as the result frame.

  The pilot unpacks `draft.tar` into the draft's host workspace. `author check` and
  `author submit` then run unchanged.
- **Rationale:** no new image to build, pin or harden (C-09). The framed-log channel is already
  proven to carry results without shared storage (C-08).
- **Alternatives:** a new author image (more to maintain, nothing gained); running OpenCode on
  the host (rejected by the owner: an unattended agent must be sandboxed).

## R4 — The prompt

- **Decision:** `internal/prompt` gets an author template built from the test-author skill's
  rules. The tests must:
  - fail today and pass once the spec is implemented;
  - test behaviour through public interfaces;
  - use only declared dependencies;
  - come with the acceptance map's format and honest `uncovered` ids.

  The steps that call `pilot author …` are left out: the Job has no pilot. The skill remains the
  source; a test checks that the template keeps the skill's rules.
- **Rationale:** the same rules for manual and automatic drafts.

## R5 — One retry with feedback, inside a draft

- **Decision:** if the check fails with `DRAFT_HIDDEN_PASSES` or `DRAFT_SUITE_FAILS`, the
  watcher runs **one** more author Job on the same draft. That Job's prompt adds:
  - the failing code;
  - the drafted tests' own output on the base (the author's own tests, never stored, as in
    018).

  At most `PILOT_AUTHOR_AUTO_ATTEMPTS` Jobs (default 2) run per draft. After the last one, the
  draft is recorded `checked(fail)` and the item waits for a spec change or a manual draft.
- **Rationale:** manual sessions iterated with `author check`. One bounded retry gives the
  unattended author the same feedback without looping.
- **Spec impact:** FR-007 is reworded in this change ("after its attempts"). The owner reviews
  it in this PR.

## R6 — Isolation

- **Decision:**
  - **Namespace:** `pilot-a-<target>-<item>`, labelled `specfoundry.io/role: author`, from a new
    `author-ns.yaml.tmpl`. It has a default deny, DNS plus its own proxy only, a quota of 1 pod,
    and no service-account token.
  - **Proxy:** `pilot-egress-a` allows exactly the model hosts, for clients with role `author`.
    It is created by `pilot up` with the other proxies, and recreated when the hosts change
    (#101).
  - **Secret:** only the model key, as for workers.
  - **Cleanup:** the namespace is deleted at the end of the draft. `pilot down` deletes any
    leftovers and records teardown row T06.
- **Checks:** from a probe twin, as P06 does. They are recorded in the draft's
  `checks.jsonl`:

  | Check | What it proves |
  |---|---|
  | A01 | 1 pod, no token |
  | A02 | only the model-key secret |
  | A03 | the model is reachable through `pilot-egress-a`; github.com and direct internet are blocked |
  | A04 | no route to the grader stand-in or a worker proxy |
  | A05 | no workers' namespace can reach the author proxy |
- **Rationale:** SC-002. It uses the same probe pattern as P06 and P07.

## R7 — Identity and submit

- **Decision:** the Job holds no repository credential. The watcher, a host process, resolves
  the identity exactly as `author start` and `author submit` do: the minimal level uses the
  operator's access, and the recommended level uses the companion-only token. It then submits.
  The draft records `trigger: auto` and the Job's spend.
- **Rationale:** the PR comes from the same identity as a manual draft, and the unattended agent
  never holds a credential.

## R8 — The driving session

- **Decision:** the pilot skill's existing rule (it never runs the `author` commands) covers
  `author watch`. The skill and the docs add one sentence: the operator starts the watcher in
  their own terminal.
