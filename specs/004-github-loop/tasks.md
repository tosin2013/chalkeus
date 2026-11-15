# Tasks: GitHub loop — issues, labels, linked PRs and GitHub approvals

**Input**: design documents in `specs/004-github-loop/`: plan.md, spec.md, research.md (R1–R8),
data-model.md, contracts/ (cli-github, github-objects, audit-github), quickstart.md

**Tests**: included. The project's convention is unit tests with fakes for every gate and
approval rule (no network), and SC-004 requires every refusal case to be proven.

**Organization**: by user story. US1 and US2 are P1; US3 and US4 are P2.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel (different files, no dependency on unfinished tasks)
- Paths are relative to the repository root

---

## Phase 1: Setup

- [X] T001 Add config keys to `pilot/specfoundry_pilot/config.py` (data-model "Config additions"):
  - `PILOT_GITHUB_APPROVALS` (`on`/`off`, default `off`);
  - `PILOT_APPROVERS` (comma-separated logins; "required when approvals are on");
  - `PILOT_PUBLISHER_CREDENTIAL` (`token` default or `app`; "both configured and no choice
    means refuse");
  - `PILOT_APP_ID`, `PILOT_APP_INSTALLATION_ID`;
  - the `github-app.pem` path (mode 600, never read into Python).

  Validate in `validate_values`, and add unit tests in `pilot/tests/test_config.py`.
- [X] T002 [P] Add fake-GitHub test helpers (canned issues, comments, PRs and permission
  responses; no network) in `pilot/tests/ghfake.py`

---

## Phase 2: Foundational (blocking for all stories)

- [X] T003 Create `pilot/images/tools/ghops.py`: GitHub write operations run with the token
  only in the child's environment (the `check_token.py` pattern):
  - `ensure-labels`, `create-issue`, `edit-issue` (body and labels), `comment`,
    `close-issue`;
  - input from a JSON file, output one JSON line;
  - never prints the token.
- [X] T004 Add a host-side runner `run_ghops(cfg, op, payload)` in
  `pilot/specfoundry_pilot/ghsync.py`. It sources the publisher credential env file (the
  token's `publisher.env`, or the App token file from T030) into the child only, and uses the
  same pattern as `publisher_token.verify`.
- [X] T005 [P] Extend the write probe in `pilot/images/tools/check_token.py` to also test
  `issues: write` on the fixture repo (side-effect-free: label lookup plus a dry-run label
  create on a name that already exists, so no write happens) and its absence on the canary.
  Update `pilot/tests/test_publisher_token.py`.
- [X] T006 [P] Write contract notes into the 002 and 012 contracts: one-line "Amended by 004"
  pointers in `specs/002-m1-gated-pilot/contracts/cli.md`,
  `specs/002-m1-gated-pilot/contracts/evidence-record.md` and
  `specs/012-agent-approvals/contracts/audit-channels.md`.

**Checkpoint:** the GitHub write path exists and is tested with fakes; the probe covers issues.

---

## Phase 3: User Story 1 — Follow a run on GitHub (P1) 🎯 MVP

**Goal:** item issues with status labels and task checklists; PRs close their issue; the drift
report is empty after reconcile.

**Independent test:** quickstart §2 and the end of §3 (`github drift` empty).

- [X] T007 [P] [US1] Unit tests in `pilot/tests/test_ghsync.py`:
  - building an item issue body from roadmap and `spec status` data (marker
    `<!-- specfoundry:item=<id> -->`, checklist copied from `tasks.md`);
  - exactly one `status:<status>` label;
  - idempotence (a second sync produces 0 operations);
  - drift cases from research R8.
- [X] T008 [US1] Implement the pure planning functions `desired_state(roadmap, status_json)` and
  `plan_ops(desired, current)` in `pilot/specfoundry_pilot/ghsync.py`. Issues are matched by
  marker, never by title.
- [X] T009 [US1] Implement `sync(cfg, run=None)` in `pilot/specfoundry_pilot/ghsync.py`:
  - read current issues with the operator's `gh`;
  - apply `plan_ops` through `run_ghops`;
  - never delete, close or reopen;
  - write `github/issues.json` and a `github_sync` event when a run is given.
- [X] T010 [US1] Implement `drift(cfg, run=None)` in `pilot/specfoundry_pilot/ghsync.py`: the
  research R8 report; with a run, write `github/drift.json`.
- [X] T011 [US1] Add `pilot github sync` and `pilot github drift [--run <id>]` to
  `pilot/specfoundry_pilot/cli.py`, with exit codes per `contracts/cli-github.md`.
- [X] T012 [US1] In `pilot/specfoundry_pilot/publisher.py`:
  - end the PR body with `Closes #<item issue>`, looked up from `github/issues.json`, or by
    syncing first if it's missing;
  - record `kind: "task"` in `prs.json`.
- [X] T013 [US1] Call `ghsync.sync` at run start and after reconcile, and `ghsync.drift` at the
  end of a successful run, in `pilot/specfoundry_pilot/pipeline.py`. Any GitHub error stops
  with exit 2 and leaves the state unchanged.

**Checkpoint:** a terminal-approved run shows issues, labels and linked PRs, and an empty drift
report.

---

## Phase 4: User Story 2 — Approve a decision on GitHub (P1)

**Goal:** mode and retry approved by `/approve <step>` on the run's tracking issue, recorded with
`channel=github` and the approver.

**Independent test:** quickstart §3 steps 1–2 (every refusal case), and AU09/AU10 showing
`github` with the login.

- [ ] T014 [P] [US2] Unit tests in `pilot/tests/test_ghapprove.py`, one per research R3 rule:
  - accept: an exact first line from a listed approver, posted after the decision post;
  - reject:
    - wrong step;
    - wrong run or attempt (marker mismatch);
    - author not listed;
    - author lacks write now;
    - author is the publisher identity;
    - app-authored;
    - edited;
    - posted before the decision post;
    - withdrawn;
    - duplicate (only the earliest is used);
    - GitHub error (fail closed).
- [ ] T015 [US2] Implement the tracking issue and decision posts in
  `pilot/specfoundry_pilot/ghapprove.py`:
  - `ensure_tracking(cfg, run)` (title `pilot run <run-id>`, marker
    `<!-- specfoundry:run=<run-id> -->`);
  - `post_decision(cfg, run, pending)` (marker
    `<!-- specfoundry:decision run=<id> step=<s> task=<t> n=<n> -->`, text from
    `approvals.describe`, the exact reply command, the allowed approvers);
  - write `github/tracking.json` and a `decision_posted` event.
- [ ] T016 [US2] Implement `find_approval(cfg, run, pending)` in
  `pilot/specfoundry_pilot/ghapprove.py`:
  - read comments with the operator's `gh`;
  - apply the research R3 checks, including the collaborator permission read (`write`,
    `maintain` or `admin`);
  - log every candidate with accept or reject and its reason to `github/approvals.jsonl`, and
    an `approval_rejected` event for each rejection.
- [ ] T017 [US2] In `pilot/specfoundry_pilot/approvals.py`:
  - allow `record(..., channel="github", approver=<login>, comment_id=<id>)`;
  - make `take()` return the channel stored in the record (`harness` or `github`);
  - make `note()` add `approver=` and `comment=` to the `approval_channel` detail for
    `github`.

  Keep the existing single-use and binding rules. Extend `pilot/tests/test_approvals.py`.
- [ ] T018 [US2] Wire into `pilot/specfoundry_pilot/pipeline.py`, when
  `PILOT_GITHUB_APPROVALS=on`:
  - on a pause at mode or retry, call `post_decision` and print the issue URL;
  - in `resume` with no terminal, call `find_approval` before `approvals.decide`, and record
    a found approval;
  - with a terminal, discard any GitHub approval (012 R5) and mark the post answered at the
    terminal.
- [ ] T019 [US2] Add `pilot github watch --run <id>` to `pilot/specfoundry_pilot/cli.py`: poll
  every 15 s and call `resume` when `find_approval` succeeds or the status PR is merged.
  Ctrl-C leaves the run paused.
- [ ] T020 [US2] In `pilot/specfoundry_pilot/audit.py`, add `approvers=[…]` to the AU09 and
  AU10 detail for `github` entries; the pass rules are unchanged. Make AU08 also scan
  `github/`. Add tests in `pilot/tests/test_audit_github.py`.
- [ ] T021 [US2] In `.claude/skills/pilot/SKILL.md` and `AGENTS.md`: agents never post
  `/approve` or `/withdraw` comments, and explain the tracking issue. Add an `ask` rule for
  `Bash(gh issue comment:*)` and `Bash(gh api:*comments*)` in `.claude/settings.json`
  (research R4).

**Checkpoint:** a headless run proceeds only on valid GitHub approvals; every refusal is
recorded.

---

## Phase 5: User Story 3 — Approve a status change on GitHub (P2)

**Goal:** after A's merge, a status-change PR. The human's merge is the approval, and an edited
PR is refused.

**Independent test:** quickstart §3 step 3.

- [ ] T022 [P] [US3] Unit tests in `pilot/tests/test_status_pr.py`:
  - the bundle contains exactly the one heading edit;
  - merge check: accept the exact diff, reject any extra change with `STATUS_PR_EDITED`;
  - not merged means exit 4;
  - the merger is recorded as the approver.
- [ ] T023 [US3] In `pilot/specfoundry_pilot/reconcile.py`, add `status_pr_path`. After the
  existing checks (merged, tasks 100%, tree matches graded), build a one-commit bundle with
  `status_edit` on top of the merge commit. Hand it to the publisher Job as kind `status`
  (head `pilot/<run-id>/status-<item>`, title `pilot <run-id>: <item> verified`, draft, body
  per `contracts/github-objects.md`).
- [ ] T024 [US3] In `pilot/specfoundry_pilot/publisher.py` and `pilot/images/tools/publish.py`,
  accept kind `status`; record `kind: "status"` in `prs.json`. Still no merge code path.
- [ ] T025 [US3] In `pilot/specfoundry_pilot/pipeline.py`, add the `awaiting-status-merge`
  sub-state (`run.json` `pending.status_pr`). On resume:
  - check the status PR (merged, diff equals the proposal, default branch shows `verified`);
  - then record `approval_channel step=reconcile approved=true channel=github approver=<merger> pr=<n>`
    and `task_verified`, and continue to B.

  This path is used when GitHub approvals are on; today's terminal/harness push path stays
  otherwise.
- [ ] T026 [US3] In `pilot/specfoundry_pilot/audit.py`, change AU03 per research R7 and
  `contracts/audit-github.md`: "exactly 2 `task` PRs (A, B), plus 0 or 1 `status` PR per
  verified item; all drafts by the publisher identity; a missing `kind` means `task`". Add
  tests to `pilot/tests/test_audit_github.py`, and confirm `test_committed_evidence.py` still
  passes.

**Checkpoint:** a full headless run completes with all three decisions made on GitHub.

---

## Phase 6: User Story 4 — Choose the publisher credential per install (P2)

**Goal:** an optional GitHub App: a 1-hour token per run, target repo only, PRs by the App.

**Independent test:** quickstart §4.

- [ ] T027 [P] [US4] Unit tests in `pilot/tests/test_ghapp.py` (a throwaway test key generated
  with `openssl` in setUp):
  - JWT header and claims (iat −60 s, exp +9 min, iss = App id);
  - the token request body (`repositories: [<repo>]`, permissions
    `{contents: write, pull_requests: write, issues: write}`);
  - the key never read into Python (signed through a child process);
  - the token file has mode 600 and is deleted at teardown.
- [ ] T028 [US4] Implement `pilot/specfoundry_pilot/ghapp.py`:
  - `jwt(cfg)`, signing with `openssl dgst -sha256 -sign` in a child process;
  - `mint(cfg, run)`, which calls `POST /app/installations/{id}/access_tokens` in a child
    and writes the token to a mode-600 env file under the run's private temp dir, never into
    the evidence;
  - re-mint when the token has expired.
- [ ] T029 [US4] Add `pilot app-key` (store the key from stdin as `github-app.pem`, mode 600,
  kept only if `app-token --check` passes; 006 F3 rule) and `pilot app-token --check` to
  `pilot/specfoundry_pilot/cli.py`.
- [ ] T030 [US4] Make the credential selectable in `pilot/specfoundry_pilot/pipeline.py`,
  `pilot/specfoundry_pilot/infra.py` (the `scm-token` Secret gets only the short-lived token)
  and `ghsync.run_ghops`, from `PILOT_PUBLISHER_CREDENTIAL`. If both are configured and there
  is no choice, refuse. Check that P09 is unchanged.
- [ ] T031 [P] [US4] Write a new `docs/github-app.md`: create the App (permissions contents,
  pull requests and issues: write; no others), install it on the fixture repo only, store the
  key, check it, switch the credential. Link it from `docs/operator-guide.md` and
  `docs/providers.md` if relevant, and add troubleshooting entries in
  `docs/troubleshooting.md` (token missing `issues: write`; App not installed on the repo;
  key rejected).

**Checkpoint:** an App-mode run publishes as `<app>[bot]`, and the audit passes.

---

## Phase 7: Polish and validation

- [ ] T032 [P] Add a "GitHub loop" section to `docs/operator-guide.md`: sync, approvals on
  GitHub, the status-change PR, and the identity limit (spec FR-009a). Add troubleshooting
  entries for each `approval_rejected` reason. Then run `pilot/tools/docs_check.py`.
- [ ] T033 Update `pilot/README.md` with the new commands and config keys.
- [ ] T034 Run the full unit suite and docs check; record the counts in this file's Notes.
- [ ] T035 **Owner action:** quickstart §2–§3 on the fixture in token mode. Record results in
  `specs/004-github-loop/evidence/README.md` and copy the run record into `evidence/<run-id>/`.
- [ ] T036 **Owner action:** quickstart §4 in App mode (SC-005), timing the App setup from the
  docs only (SC-007).
- [ ] T037 **Owner action:** quickstart §5, a terminal run with GitHub approvals off (no
  change).
- [ ] T038 Privacy check: AU08 passes for every 004 run, and no token, key or model host
  appears in `specs/004-github-loop/`.
- [ ] T039 Add an Evidence section to `specs/004-github-loop/spec.md`, then
  `/speckit-roadmap-debrief`, and with the owner's approval `/speckit-roadmap-write`: 004
  spec dir and status; Q4 resolved ("per item").

---

## Dependencies & Execution Order

- **Setup (T001–T002)** comes first. **Foundational (T003–T006)** blocks every story.
- **US1 (T007–T013)** is the MVP. **US2 (T014–T021)** needs T003/T004 and the decision posts;
  it is independent of US1 except that the tracking issue uses the same write runner.
- **US3 (T022–T026)** needs US2's `github` channel (T017) and the publisher changes (T012).
- **US4 (T027–T031)** is independent of US2 and US3; it only needs Foundational. It can run in
  parallel with US2.
- **Validation (T035–T037)** needs all stories. T038 → T039 come last.

### Parallel Opportunities

- T002 alongside T001; T005 and T006 alongside T003/T004.
- Test tasks T007, T014, T022 and T027 can be written in parallel (different files).
- US4 (T027–T031) runs in parallel with US2 once Foundational is done.
- T031 and T032 (docs) in parallel with the code tasks of their stories.

## Implementation Strategy

### MVP

Setup, Foundational and US1: the roadmap mirrored on GitHub, with PRs closing their issues and
an empty drift report, all with terminal approvals as today.

### Then

1. US2 (GitHub approvals for mode and retry).
2. US3 (status-change PR), which completes a fully headless run.
3. US4 (GitHub App).
4. Owner validation in both credential modes.

### Owner actions (not automatable)

T035–T037 need the fixture repo, a cluster, real GitHub accounts (a second account for the
non-approver case), and a GitHub App created by the owner. The agent must not post `/approve`
comments or merge PRs on the owner's behalf.

## Notes

- Do not weaken 012's rules: one pending step, bound, used once; the terminal decides.
- Never read a token or key into the orchestrator. Writes run in child processes or the
  publisher Job only.
- Commit after each phase. Open a PR from `004-github-loop`; never push to `main`.
