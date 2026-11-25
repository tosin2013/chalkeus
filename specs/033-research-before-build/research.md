# Research: Research before build (033)

Phase 0 of the plan. Each decision names the code it builds on (mapped 2026-11-25) and what else
was considered.

## R1 — Where the diagnosis is built

- **Decision:** a new package `internal/diagnose` with one pure function: attempt evidence in,
  diagnosis out. The pipeline calls it in `runTask` right after the attempt record is saved
  (`internal/pipeline/pipeline.go`, after `run.Save()` at ~:333), before the retries-exhausted
  branch, so an exhausted run gets one too. It writes `attempts/<task>-<n>/diagnosis.json`, adds
  a short `diagnosis` object to the attempt record, and replaces `deadlineNote` as the text of
  `run.State["decision_note"]` (`confirmRetry`, :184-201).
- **Rationale:** the retry question (`confirmRetry`), the GitHub decision comment
  (`ghloop.PostDecision` → `decisionNote`, `internal/ghloop/approve.go:158-160`) and the summary
  all read from the run, so one write reaches all three (FR-005). A pure function over files is
  replayable on kept evidence (SC-001) and deterministic (FR-006).
- **Alternatives:** building it in the worker image (Python): rejected, the worker is untrusted
  and the diagnosis must not depend on what it reports about itself beyond its kept files;
  building it at display time: rejected, the GitHub comment and summary would need it again.

## R2 — Which task text the diagnosis and doctor read

- **Decision:** the diagnosis reads the slice's task list from the attempt's own `prompt.md`
  (`# Task list` section, written by `worker.MakeInputs`), which is what the worker saw and is
  kept in evidence. Doctor reads the slice's lines of `tasks.md` with the existing task-line
  parser `ghloop.Checklist` (`internal/ghloop/sync.go:94-111`), restricted to the slice's phases
  (`readiness.Result.Slice.Phases`).
- **Rationale:** C-01 and `internal/slice` say slices are *scheduled* only from `spec status`
  per-phase counts, never from `tasks.md`. Reading task wording for a warning or a diagnosis
  schedules nothing, and `ghloop` already parses task lines for issue checklists; no new parser.
- **Alternatives:** asking spec-status for task text: it reports counts and lines, not wording.

## R3 — Research wording

- **Decision:** one exported list in `internal/diagnose`, used by doctor and the diagnosis
  (FR-016), matched case-insensitively at the start of an unchecked task's description (after
  the ID and `[P]`/`[USn]` markers): `locate`, `find out`, `find the`, `find which`,
  `investigate`, `identify the cause`, `identify which`, `figure out`, `determine why`,
  `determine which`, `diagnose`, `research`, `explore`, `spike`. Documented in
  `docs/targets.md`.
- **Rationale:** deterministic (no model, FR-006), and anchored at the start of the task to cut
  false positives like "set X; find and replace Y". documcp's T003 ("Locate the suites …") and
  T004a ("Find the test that writes …") match; T001/T002/T005/T006 don't.
- **Alternatives:** a model judgement: rejected (C-16, FR-006); matching anywhere in the line:
  too many false positives ("…, then find").

## R4 — The catalogue of known failure causes

- **Decision:** a Go list in `internal/diagnose` of every attempt failure reason the code can
  produce: `deadline`, `spend_cap`, `grade_fail`, `harness_error`, `no_result`, `bad_frame`,
  `oversize`, `grader_error`, `grader_timeout`, and the worker report errors (`clone_failed`,
  `checkout_failed`, `workspace_failed`, `bundle_failed`, `bad_meta`, `induction_failed`). A
  docs test fails if any is missing from `docs/troubleshooting.md`; the missing ones get entries
  (today the page has none for `grade_fail`, `no_result`, `bad_frame`, `oversize`,
  `grader_timeout`, `grader_error`, `no_frame`). A reason outside the list is "not recognised"
  (FR-002.6).
- **Rationale:** there is no Go code list checked against the docs today (`docs_test.go` checks
  D1–D5 only); this keeps the diagnosis and the page in step, as the spec's assumption asks.
- **Alternatives:** parsing the troubleshooting page at run time: brittle, and the run host
  needn't have the docs.

## R5 — Egress denials: the probes' own refusals

- **Decision:** drop the probes' refusals: `github.com` through the model proxy (P06,
  `internal/checks/checks.go:242`) and `example.invalid` through the target proxy (P06r,
  `undeclaredHost`). Everything else is a signal, grouped by host and proxy with counts. Parse
  tinyproxy's `Proxying refused on filtered domain "<host>"` lines from `proxy-denials.log`
  (collected by `worker.ProxyDenials`, `worker.go:212-235`; nothing parses it today).
- **Rationale:** documcp attempt 1 has three refusals; two are probes, the third (`github.com`
  through the target proxy, 16:43:00) is the worker's own. The fixture's A/1 has only the P06
  probe refusal, so its diagnosis has no egress signal (SC-002).
- **Rule:** an egress signal alone points to an environment need only when the attempt also
  failed with an install or download error, or changed a manifest; otherwise it's listed as
  information (spec edge case).

## R6 — Environment signals

- **Decision:**
  - files changed outside the allowed paths (`changed_paths` against
    `run.State["readiness"]["allowed_paths"]`, also in the attempt's `meta.json`), marking
    dependency manifests and lock files with the toolchain's patterns (`images.IsManifest`,
    `internal/images/manifests.go:9-22`);
  - report errors that mean install or clone trouble (`clone_failed`, `bundle_failed`) and
    install/download wording in the grader's G4 detail;
  - egress refusals per R5.
- `PREFETCH_FAILED` and `TOOLCHAIN_*` stop a run before any attempt; their run-level failure
  message gains the same "back to planning: planning PR" route, without a diagnosis file.

## R7 — Test-run allowance: timing the test command

- **Decision:** `pilot/images/tools/grade.py` records the seconds the project test command took in
  G4's detail and in its report (`test_seconds`). Doctor `--baseline` reads it from the baseline
  grade and reports `allowance = floor((worker deadline − soft margin) / test_seconds)`, warning
  below 5. A run's own baseline grade (event `baseline_pass`) gives the same number to the
  diagnosis. Without a recorded time (older grader image), the allowance is "unknown".
- **Rationale:** the baseline grade runs the command twice (G4, then G5 with the overlay), so
  timing the Job would overstate one run; only the grader sees one run's time. The worker
  deadline is `images.Resolve(...).Worker.DeadlineS` minus the 120 s soft margin (030).
- **Cost:** a grader image change goes through the image workflow's PR and `images.lock`
  (C-09); the Go side tolerates the old image.

## R8 — "Back to planning": the decision and the final state

- **Decision:**
  - **Word:** a new step `replan` for `forge approve` (`./chalkeus forge approve --run <id>
    replan`) and on GitHub (`/approve replan`), accepted only while the run is
    `paused(awaiting-retry)`. Reusing `approve` keeps the harness rule
    (`Bash(./chalkeus forge approve:*)` asks) and `command_audit.go` unchanged; the approver
    allow-list and permission checks in `ghloop.Evaluate` apply as for every step.
  - **State:** a new final state `stopped(back-to-planning)`. `Resume`'s awaiting-retry branch
    checks for a `replan` record before `confirmRetry`; then it sets the state, drops the
    grading namespace (`dropGrading`), writes the summary, closes the tracking issue's decision
    and stops the watcher (`watcher.Stop/Release`). The watcher's `ReadyToResume` treats a
    `replan` record like an approval so it resumes the run to end it.
  - **Step mode:** `runEnded` and `runState` (`internal/step/manager.go:203-213`,
    `gather.go:136-147`) count `stopped(...)` as ended, so the target's slot frees.
  - **Audit:** on a `stopped(back-to-planning)` run, AU02 and AU03 (which need a final pass and
    one task PR) report `n/a` with "run ended by the owner for planning", and the audit's
    result counts only the other checks.
- **Rationale:** a paused run today can only be retried or left paused with its watcher (and
  step mode's slot) held forever. `failed(BACK_TO_PLANNING)` was considered: simplest, but it
  reads as a failure of forge (FR-011).
- **Alternatives:** a new top-level command (`forge replan`): needs a command-audit class and a
  settings rule; nothing gained over an `approve` step.

## R9 — External practice (2026-11-25 search)

- Agents that over-explore burn their timeout without converging on a plan (Sourcegraph,
  "Why coding agents fail in large codebases"); limiting each session to one feature was
  critical in Anthropic's long-running-agent harness work. A time-boxed spike separates
  investigation from building. For Jest leaks: isolate by folder, then by file
  (`why-is-node-running`, `leaked-handles`), fix the origin; `--detectOpenHandles` implies
  `--runInBand` and can hide leaks that share a stack.
- Applied: research is planning work (C-12), not an attempt; forge's job is to notice it and
  say so with evidence.
