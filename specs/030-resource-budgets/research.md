# Research: Resource budgets and capacity checks

## R1 — Keep a deadline-stopped attempt's work (US6, FR-016/017)

- **Today:** the worker Job's `activeDeadlineSeconds` is `worker.Deadline` (1,200, a constant in
  `internal/worker/worker.go:25`). When Kubernetes kills the pod, `entrypoint.sh` never reaches
  `finish()`, so no result frame (report.json, change.bundle) is written to stdout;
  `RunAttempt` decodes nothing and records `failed/deadline` with no spend
  (`worker.go:324-357`). That is F15.
- **Decision:** a **soft deadline** inside the worker. `meta.json` gains `soft_deadline_s`
  (the attempt deadline minus a collection margin, default 120 s). `watchdog.py`, which already
  SIGTERMs the harness at the spend cap and writes `/tmp/spend.json` at EOF, also SIGTERMs it when
  the soft deadline passes and records `stopped_by_deadline: true`. `entrypoint.sh` then runs its
  normal path: copy the workspace back, commit, bundle, `finish()` with `ERROR="deadline"`. The
  Job's hard deadline stays as the backstop.
- **Pipeline:** `RunAttempt` treats `report.error == "deadline"` like the hard outcome
  (`failed/deadline`) but keeps report.json, change.bundle and spend; the attempt's spend counts
  toward the caps as for `spend_cap`. The change is stored as evidence only: a failed attempt is
  never graded or published (existing rule: grading runs only for `collected` attempts without a
  failure reason).
- **Alternatives:** (a) read files from the pod after the deadline: the pod is gone; (b) stream
  the diff periodically: more moving parts, and secrets/hygiene risk in a sidecar; (c) a
  `preStop` hook: not run for `activeDeadlineSeconds` kills reliably within the grace period.

## R2 — Where budgets come from (US1, FR-001..006)

- **Today:** grader CPU/memory/job deadline/command timeout come from the toolchain catalog
  (`internal/images/toolchains.go`, `"grader": {"cpu","memory","deadline_s","command_timeout_s"}`)
  with profile raises for memory and job deadline (`internal/pipeline/item.go:486-500`); the
  prefetch Job uses the grader's CPU/memory (`pilot/manifests/prefetch-job.yaml.tmpl:48`); the
  worker and author Jobs hard-code `cpu: "2", memory: 2Gi` in their templates, and their
  deadlines are code constants (`worker.Deadline`, author `je.Deadline`).
- **Decision:** one resolver, `budget.Resolve(catalogEntry, target) (Budget, error)`, returns
  per-role values from the catalog (gaining a `worker` and `author` block with today's values as
  defaults) raised by the profile; every Job template takes `${CPU}`, `${MEMORY}`, `${DEADLINE}`
  from it. The resolved budget is written to `run.json` at run start and read back on resume
  (FR-006), never re-resolved from the profile.
- **Floors:** a value below the catalog's is refused when the profile is read
  (`TARGET_INVALID` naming the key and floor), like the existing memory and deadline checks
  (`toolchains.go:153-157`).
- **Keys:** see contracts/profile-keys.md.

## R3 — Capacity: what to compare (US2, FR-007..010)

- Kubernetes schedules on **requests** (500m/512Mi for every role today), so a pod can land on a
  node that can't give it its **limit**. The check judges limits: the largest role's CPU and
  memory limit must fit the largest schedulable node's allocatable minus the requests of
  Chalkeus's own long-running pods on it (egress proxies, publisher, registry if in-cluster).
- Quotas: `ResourceQuota` (`limits.cpu`, `limits.memory`) and `LimitRange` max in the run
  namespaces. Run namespaces are created per attempt (`pilot-w-*`, `pilot-g-*`), so the check
  reads quotas and limit ranges that apply cluster-wide by policy only where readable; an
  unreadable list warns (FR-009).
- Reading: the existing dynamic client (`internal/k8s` `List(ns, resource, selector)`) lists
  `nodes`, `resourcequotas`, `limitranges` and pods; no new dependency.
- **Local vs remote:** the kube context `kind-<cluster>` (`internal/paths/paths.go:16`) is the
  cluster `forge up` created on this machine → local message; anything else → remote message.
- **Where:** a new doctor area `== capacity`, and the same function called by `forge run` and
  `forge author check` before staging (FR-010), failing closed on a fail.

## R4 — Unschedulable pods at run time (US3, FR-011)

- `runner.Wait` polls the Job until Complete/Failed/DeadlineExceeded. Decision: while waiting,
  also read the Job's pod; if its `PodScheduled` condition is `False` with reason
  `Unschedulable` and a message containing `Insufficient cpu` or `Insufficient memory` for longer
  than 120 s, return a new outcome `Unschedulable`. The caller records `capacity_stop` (event and
  evidence), deletes the namespace and stops the run; the attempt counter isn't advanced and no
  spend is recorded (no harness ran).

## R5 — Visibility (US4, FR-012/013)

- The mode summary (`internal/pipeline/pipeline.go:111`) gains `budget` (per role, with a
  `raised` list naming the profile keys used); the terminal mode text and the GitHub decision
  comment print it. `run.json` and a draft's `draft.json` record the resolved budget and the
  capacity result. The audit compares `run.json.budget` with the approved mode (SC-005), as a new
  check in the item audit for runs recorded after this feature; older evidence has no `budget`
  and is skipped.

## R6 — Worker prompt (FR-005)

- The prompt already states the grader's command timeout from `CommandTimeout`
  (`worker.go:407`). It reads the resolved value instead of the catalog's.

## R7 — Testing

- Go unit tests: resolver (floors, defaults, malformed values, resume keeps budget), template
  rendering per role, capacity math with fake node/quota/pod lists (local and remote messages,
  unreadable → warn), `Wait` returning `Unschedulable`, `RunAttempt` with a soft-deadline frame.
- Python unit tests (`pilot/images/tests/`): watchdog soft deadline → SIGTERM and
  `stopped_by_deadline`; entrypoint frame on a soft stop (existing harness stub).
- Integration (025 harness): a fixture attempt with a short soft deadline yields a deadline
  failure with report.json and spend; a target with an impossible CPU raise fails doctor's
  capacity area.
