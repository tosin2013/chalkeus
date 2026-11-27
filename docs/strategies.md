# Strategies: getting items built without wasted attempts

What works when you run forge on a real project, learned from building documcp's first slice
(item 001, slice US1, 2026-11-25). Each strategy says what to do, why, and which command or
setting does it. The [approach](approach.md) explains the principles behind them; this page is
the practice.

**Contents:**
1. [Measure first](#1-measure-first)
2. [Research before build](#2-research-before-build)
3. [Grade the way CI does](#3-grade-the-way-ci-does)
4. [Size budgets from evidence](#4-size-budgets-from-evidence)
5. [Keep slices to known work](#5-keep-slices-to-known-work)
6. [When an attempt fails, read the diagnosis](#6-when-an-attempt-fails-read-the-diagnosis)
7. [Put environment changes in planning](#7-put-environment-changes-in-planning)
8. [Defer what you don't need yet](#8-defer-what-you-dont-need-yet)
9. [Let the test author run on its own](#9-let-the-test-author-run-on-its-own)
10. [Spend the strong model on planning](#10-spend-the-strong-model-on-planning)
11. [Case study: documcp US1](#11-case-study-documcp-us1)

---

## 1. Measure first

Before the first item, find out what the project's test suite costs: how long a full run takes
on the worker's CPUs, how much memory it needs, whether it exits on its own, and whether any
test is flaky. Most failed attempts come from one of these, and each is measurable before any
attempt runs.

- Today: `./chalkeus forge doctor --target <t> --item NNN --baseline` runs the project's tests on
  the cluster, offline, and reports how long one run took and how many fit in one attempt (the
  test-run allowance; a warning below 5).
- Planned (roadmap 036): a `doctor` option that measures the suite without an item (time, peak
  memory, clean exit with and without parallel workers, flakiness over three runs, whether the
  profile's test command matches CI) and keeps the numbers per target.

**Measurements are goals, not only budgets.** If a full test run takes minutes, raising the
deadline works, but so may refactoring slow tests, splitting the suite, or changing a library or
test runner. Record a target (for example "a full run under 60 s") and plan that work as items of
its own.

## 2. Research before build

An attempt builds known changes. A task whose answer isn't known yet ("locate the suites that
leave handles open", "figure out which framework…") is research: its cost isn't known either,
and an agent can spend a whole attempt searching. More time or smaller attempts don't fix that.

- Do the research while planning, yourself or with an agent that assists you, and record the
  answer in the item's `research.md` (`/speckit-plan`); then regenerate the tasks
  (`/speckit-tasks`) so each one names the change to make.
- `./chalkeus forge doctor` warns about research-worded tasks
  ([targets: research tasks](targets.md#research-tasks)).
- The Spec Kit Research Harness extension runs such an investigation within a budget and writes
  the findings into `research.md`; Bug Triage suits a defect such as a flaky test.

## 3. Grade the way CI does

Set `FORGE_TEST_COMMAND` in the target profile to what the project's CI runs, including coverage
thresholds and without workarounds such as force-exit flags. `./chalkeus forge doctor --target <t>`
shows CI's commands next to forge's, the checks grading doesn't run (typecheck, lint, audit) and
any CI job red on the default branch ([targets: doctor's CI review](targets.md#doctors-ci-review)). Otherwise a slice can pass grading
and still turn the project's CI red (documcp: a 78% branch-coverage gate the profile's command
didn't check), or hide a defect the slice was meant to fix.

## 4. Size budgets from evidence

Raise a budget when the evidence says so, not by guessing, and only for a new run (a run keeps
the budget it started with):

| Evidence | Setting |
|---|---|
| the attempt's `job.json` shows `OOMKilled` (the diagnosis says so) | `FORGE_WORKER_MEMORY` |
| few full test runs fit in one attempt (doctor's allowance, the diagnosis) | `FORGE_WORKER_DEADLINE_S` |
| the grader runs out of time on the project's tests | `FORGE_GRADER_DEADLINE_S` |

The worker namespace's quota follows the budget, so a raised budget is admitted
([targets: configure the target](targets.md#2-configure-the-target)).

## 5. Keep slices to known work

One user story per slice, and every task in it a named change in named files. A slice that mixes
quick edits with open-ended work loses the quick edits when the open-ended part times out. If a
story holds both, split the research into planning (strategy 2), or the story into two.

## 6. When an attempt fails, read the diagnosis

The retry question starts with the attempt's diagnosis: what the evidence shows, what it points
to, and one recommendation ([troubleshooting: diagnosis](troubleshooting.md#diagnosis)).

| The diagnosis recommends | Do |
|---|---|
| retry | the cause looks temporary (an ordinary grading failure, a slow model): `approve … retry` |
| raise the budget | the work is known but larger than the budget: end the run, raise the setting, start a new run |
| back to planning | unknown work or an environment need: end the run with `./chalkeus forge approve --run <id> replan`, do the research or the planning change, then start a new run |

A second timeout in a row with nothing else in the evidence counts as unknown work: don't spend
the last retry on the same search.

## 7. Put environment changes in planning

An attempt can't change the runtime, the toolchain, dependencies, registries or the lock file.
Make those changes in your planning PR, or allow the exact file in the slice's allowed paths when
the change is part of the work (documcp: `engines` in `package.json` also changes the lock file's
root `engines`, so US1 allows `package-lock.json` for that entry only).

## 8. Defer what you don't need yet

When research turns up something bigger than the slice (documcp: about 3,200 lines of memory
subsystems behind three stub tools), fix only what the slice needs and record the rest as a
roadmap item with status `deferred`. It isn't lost, and it doesn't grow the slice.

## 9. Let the test author run on its own

Turn on auto-draft (`FORGE_AUTHOR_AUTO=on`) and start
`./chalkeus forge author watch --target <t>` once, in your own plain terminal: it drafts each
ready slice's hidden tests in a sandboxed Job on your cluster and opens the companion pull request
for you to review ([targets: drafting automatically](targets.md#drafting-hidden-tests-automatically)).
The session that drives runs never starts it.

## 10. Spend the strong model on planning

Use the strongest model you have for planning and research (short, high-value work) and a cheaper
model in forge for building known tasks. The expensive case is a cheap model searching an
open-ended problem for an hour; strategies 1, 2 and 6 exist to avoid it.

## 11. Case study: documcp US1

documcp is a TypeScript MCP server: about 60,000 lines of source, 1,706 tests, a 222 s test run on
the grader's 2 CPUs, a coverage gate, and real tech debt. Its first slice moved it to Node 24 with
a clean test run:

| Run | What happened | Strategy it led to |
|---|---|---|
| `20261125T163307Z` | 40 min searching for leaked handles (task "locate the suites…"); changed the lock file | research before build (2); environment changes in planning (7) |
| (planning) | a per-folder scan found two leaks and the test that wrote into the repository; tasks rewritten | research before build (2) |
| `20261125T191444Z` | the worker was OOM-killed at 2Gi | budgets from evidence (4) |
| `20261125T200730Z` | attempt 1: the quota refused the raised budget (a forge bug, fixed); attempt 2: most of the work, but a worker-process leak remained | measure first (1); research before build (2); defer (8) |
| (planning) | a paired scan with parallel workers found the last two leaks; unused subsystems deferred | research before build (2); defer (8) |
| `20261125T223435Z` | passed on the first attempt in 22 minutes; CI then caught a coverage gap, closed with tests on the pull request | grade the way CI does (3) |

Evidence: [specs/026-slice-runs/evidence/](../specs/026-slice-runs/evidence/); friction log:
[documcp onboarding](../specs/026-slice-runs/evidence/documcp-onboarding.md).
