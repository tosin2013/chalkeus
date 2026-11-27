# Implementation plan: Measure a target, and review its CI (036)

**Spec**: [spec.md](spec.md) · **Tasks**: [tasks.md](tasks.md)

## US1: the CI review (built)

- **`internal/ci`**, no network in the analysis: `ParseWorkflow` (sigs.k8s.io/yaml), `Split`,
  `Resolve` (package scripts, a few levels, extra arguments kept), `Classify` (test, typecheck,
  lint, build, audit, commitlint, install; known actions such as CodeQL and dependency review),
  `Flags` (coverage, `--forceExit`, `-race`, `--detectOpenHandles`), `Review` (the findings),
  `ErrorLine` and `StepSection` (the telling line of a failed step's log).
- **`ci.Reader`**: `Workflows`, `File`, `LatestRuns`. `ci.GitHub` uses the operator's go-github
  client: the contents API for `.github/workflows` and `package.json`, the Actions API for the
  newest completed push run per workflow and its jobs, and the job log (pre-signed URL, no
  credential, 8 MiB cap) for failed jobs.
- **`doctorCI`** runs after the roadmap rows on every `doctor --target`; hosts other than GitHub
  get one information line until 037's hosts add readers.
- **Constitution and roadmap**: read-only (C-12: forge never changes the target); no new command,
  so the command audit is unchanged.

## US2: `--measure`

To be planned from roadmap 036's outcome (grader-like Jobs, kept results, goals).
