# Implementation Plan: Resource budgets and capacity checks

**Branch**: `plan-030-resource-budgets` | **Date**: 2026-11-24 | **Spec**: [spec.md](spec.md)

## Summary

Two problems from documcp's live run, fixed in order of how much they unblock:

1. **An attempt that hits its deadline loses everything** (F15). The worker gets a soft
   deadline: its watchdog, which already stops the agent at the spend cap, also stops it 120 s
   before the Job's deadline, so the entrypoint's normal path still bundles the change and
   reports usage (R1). Together with a raise-only worker deadline, CPU and memory, this is the
   MVP: documcp's US1 can run with, say, 2,400 s and never lose an attempt's work again.
2. **Budgets are fixed and capacity is unchecked.** One resolver turns the catalog plus the
   profile's raises into a per-role budget (worker, grader, prefetch, author) stored in
   `run.json` (R2); templates take it; doctor, `run` and `author check` compare it with the
   cluster (R3); an unschedulable pod stops the run fast (R4); the mode approval and evidence
   show it (R5).

## Technical Context

- **Language:** Go (`cmd/chalkeus`, `internal/`); Python in the worker image
  (`pilot/images/worker/watchdog.py`, `entrypoint.sh`).
- **Dependencies:** existing packages `config`, `images`, `pipeline`, `worker`, `grader`,
  `author`, `runner`, `k8s`, `doctor`, `evidence`, `audit`. No new modules; nodes, quotas and
  limit ranges are read with the existing dynamic client.
- **Storage:** `run.json` and `draft.json` gain `budget` and `capacity`; `report.json` gains
  `stopped_by` (data-model.md).
- **Testing:** `go test ./...` (resolver, templates, capacity math, `Wait` unschedulable,
  `RunAttempt` soft stop); `python3 -m unittest discover pilot/images/tests` (watchdog); 025
  integration scenarios (quickstart §1, §3); one live documcp run (quickstart §4).
- **Target platform:** laptop mode on kind and remote Kubernetes/OpenShift contexts; macOS
  (031) shares the code.
- **Constraints:** without budget keys, behaviour, manifests and evidence are unchanged except
  the new `budget`/`capacity` records; the worker image change ships through the image workflow
  (C-09) and older images (no `soft_deadline_s` support) keep today's behaviour.
- **Scale:** one run at a time per target; a handful of nodes.

## Constitution Check

| Principle | How 030 holds | ✓ |
|---|---|---|
| I. Roadmap is the source of truth | no scheduling input added; budgets come from the owner's profile and the catalog | ✓ |
| II. Gates fail closed | malformed or below-floor values refuse the profile; a capacity fail refuses `run`/`author check`; only an unreadable cluster warns, and the run-time stop backs it up | ✓ |
| III. Evidence defines done | deadline stops now keep change, log and usage; budget and capacity recorded; audit compares budget with the approved mode | ✓ |
| IV. Least privilege | raising CPU/memory/time changes no network policy, egress, credential or mount (FR-014); capacity reads are list-only and optional | ✓ |
| V. Humans approve and merge | the budget is shown in the mode approval; a deadline-stopped attempt is never published; retries still need approval | ✓ |
| VI. Build on, don't rebuild | reuses the watchdog's stop path, the catalog's floors, the existing raise keys, the dynamic client | ✓ |
| VII. Repeatable and portable | no platform-specific code; the local/remote message differs only by kube context | ✓ |

Roadmap constraints: C-02 (fail closed), C-03 (isolation unchanged), C-11 (same guarantees in
both modes), C-16 (grading deterministic: a raised command timeout is recorded and approved).

Post-design re-check: no violations; nothing to track under Complexity.

## Project Structure

### Documentation (this feature)

```text
specs/030-resource-budgets/
├── spec.md
├── plan.md            # this file
├── research.md        # R1–R7
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── profile-keys.md
│   ├── capacity.md
│   └── worker-soft-deadline.md
└── checklists/requirements.md
```

### Source code (repository root)

```text
internal/config/targets.go          # new keys, parse + validate (floors checked in images)
internal/images/toolchains.go       # catalog worker/author blocks; Budget resolver (or new internal/budget)
internal/pipeline/pipeline.go       # mode summary gains budget
internal/pipeline/item.go           # use resolved budget for grader/prefetch; store in run.json
internal/worker/worker.go           # Deadline from budget; soft_deadline_s in meta; soft-stop handling
internal/author/job.go              # author budget
internal/runner/runner.go           # Wait: Unschedulable outcome
internal/doctor/capacity.go         # new area; shared Check() used by run and author check
internal/evidence/                  # budget, capacity, capacity_stop events
internal/audit/                     # budget vs approved mode (new runs only)
pilot/manifests/worker-job*.yaml.tmpl, author-job.yaml.tmpl   # ${CPU} ${MEMORY}
pilot/images/worker/watchdog.py, entrypoint.sh                 # soft deadline
pilot/images/tests/test_watchdog.py
docs/targets.md                     # sizing section (FR-015)
```

## Phases (for tasks.md)

1. **US6 + worker budget (MVP):** soft deadline (worker image, meta, `RunAttempt`), worker keys,
   worker template variables, resolver for the worker role, budget in `run.json`. Ships with an
   image rebuild.
2. **US1 rest:** grader CPU, command timeout, author keys; prefetch follows grader; resume keeps
   budget; prompt states the resolved timeout.
3. **US2:** capacity check (doctor area, `run`/`author check` preflight), local/remote/quota
   messages.
4. **US3:** `Unschedulable` outcome and `capacity_stop`.
5. **US4:** mode approval, GitHub decision text, evidence, audit check.
6. **US5:** sizing docs, including slow vs never-exits (documcp).

## Complexity Tracking

None.
