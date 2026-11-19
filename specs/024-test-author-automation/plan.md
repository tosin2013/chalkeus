# Implementation Plan: Test-author automation

**Branch**: `024-test-author-automation` | **Date**: 2026-11-19 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/024-test-author-automation/spec.md`

## Summary

With `PILOT_AUTHOR_AUTO=on` for a target, a new command, `pilot author watch`, polls the target
for items ready for drafting. For each such item, one at a time, it:
1. creates the draft as `author start` does;
2. runs the test author as a **Job** in its own namespace. The Job uses the worker image in an
   `author` role, with OpenCode and a prompt built from the test-author skill, and model-only
   egress through its own proxy;
3. unpacks the draft from the Job's framed output;
4. runs `author check`, with one bounded retry that feeds back the failure;
5. if the check passes, runs `author submit`.

The human reviews and merges the companion PR, as in 018. Decisions are in
[research.md](research.md).

## Technical Context

- **Language/Version:** Go 1.24 (`cmd/pilot`, `internal/`). The image scripts are Bash and
  Python 3 (`pilot/images/worker/`).
- **Primary dependencies:** the existing pilot packages:
  - `author` (start, check, submit);
  - `readiness`;
  - `worker`, `frame` and `stage` (Job, framed output, staged inputs);
  - `cluster` (egress proxies);
  - `checks`;
  - `prompt`.
- **Storage:** draft evidence under `<evidence root>/drafts/<id>/` (018). Workspaces under
  `pilot/.build/author/` (git-ignored).
- **Testing:** `go test ./...` (unit tests with fakes, as `author/flow_test.go` does), and the
  Python suite for the entrypoint's author role. One live run on kind with Stockroom
  (quickstart).
- **Target platform:** laptop clusters (kind, minikube). Cluster mode stays 011's.
- **Constraints:**
  - the Job holds no repository credential and reaches only the model hosts;
  - no storage is shared with workers;
  - no new image;
  - manual drafting stays unchanged (FR-009).
- **Scale:** one draft at a time per watcher, and at most `PILOT_AUTHOR_AUTO_ATTEMPTS` (2) Jobs
  per draft.

## Constitution Check

- **C-04, human merge only:** the watcher opens a companion PR and never merges it. Runs still
  need the human's mode approval ✓
- **C-07, C-08, isolation:** the author has its own namespace and proxy, with model hosts only.
  It shares no storage, and its draft leaves only through framed logs ✓
- **C-09, images:** it reuses the digest-pinned worker image; nothing new is built ✓
- **C-11, laptop and cluster modes:** this is laptop mode; the operator trigger stays 011's ✓
- **C-12, humans plan:** the author drafts tests only, for items a human specified ✓
- **C-14, hidden tests:** the draft goes only to the companion PR and the author's own
  evidence. Workers and the driving session never see it ✓

No violations.

## Project Structure

### Documentation (this feature)

```text
specs/024-test-author-automation/
├── spec.md
├── plan.md
├── research.md
├── quickstart.md
├── contracts/author-watch.md
└── tasks.md
```

### Source code

```text
internal/author/watch.go                    eligibility, the draft loop, retry with feedback
internal/author/job.go                      the author Job: stage, run, collect, unpack
internal/checks/author.go                   A01–A05 against the author namespace
internal/prompt/author.go                   the author prompt (from the skill's rules)
internal/cli/author.go                      `pilot author watch`
internal/cluster/infra.go                   pilot-egress-a; teardown row T06
internal/config/targets.go                  PILOT_AUTHOR_AUTO, _INTERVAL_S, _AUTO_ATTEMPTS
pilot/manifests/author-ns.yaml.tmpl         the author namespace
pilot/manifests/author-job.yaml.tmpl        the author Job (worker image, role author)
pilot/images/worker/entrypoint.sh           role author: workspace in, draft.tar out
docs/targets.md, .claude/skills/pilot/SKILL.md, .claude/skills/test-author/SKILL.md
```

**Structure decision:** it extends the existing `author` package and the worker image's
entrypoint; there is no new package or image.
