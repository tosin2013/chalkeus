# Feature Specification: Measure a target before building, and review its CI (036)

**Feature Branch**: `036-ci-review`
**Created**: 2026-11-27
**Status**: In progress (US1 built)
**Input**: roadmap item 036 (1.43.0): `doctor --measure` and a CI review, GitHub first.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Doctor reviews the target's CI (Priority: P1) 🎯 MVP

An operator onboarding a repository runs `./chalkeus forge doctor --target <t>` and, without any
flag, sees what the project's CI runs next to what forge grades with: jobs red on the default
branch and why, whether the test commands agree, the checks grading never runs, and CI's runtime
versions. They fix the profile, the tests or CI before the first run, instead of finding out from
a red pull request.

**Why this priority**: documcp found its coverage gate (001 US1), `--forceExit` and a Security
Audit job red on `main` only after the fact. Reading CI takes seconds and needs no cluster.

**Independent Test**: run doctor on a target with GitHub Actions and compare the `ci` rows with
its workflow files and its latest default-branch runs.

**Acceptance Scenarios**:

1. **Given** a target with `.github/workflows/*.yml`, **When** doctor runs, **Then** it lists the
   workflows that check changes (push or pull request) and names the others (tag-only pushes,
   schedules, manual) without comparing them.
2. **Given** a job whose latest default-branch run failed, **When** doctor runs, **Then** it warns
   with the workflow and job, the failed step, the step's command and the most telling log line
   (an advisory's package and severity, a failing test, a compiler error); a failure in a step
   grading doesn't relate to (deployment, credentials) is information.
3. **Given** CI's test steps and the profile's `FORGE_TEST_COMMAND`, **When** doctor resolves both
   through `package.json` scripts, **Then** it shows both and warns when they disagree on
   coverage, force-exit, the race detector or open-handle detection.
4. **Given** CI steps that typecheck, lint, build, audit, lint commits or scan code, **When** the
   profile's command runs none of that kind, **Then** doctor warns, listing each with its command
   and where CI runs it.
5. **Given** `actions/setup-node`, `setup-go` or `setup-java` versions (including matrix values),
   **When** they differ from the target's toolchain, **Then** doctor warns.
6. **Given** no CI configuration, no access to CI runs, or a host without a CI reader, **Then**
   doctor says so in one information line (and still reviews the files when only runs are
   unreadable); the CI review never fails doctor.

### User Story 2 - `doctor --measure` (Priority: P2)

The measurements in roadmap item 036 (full-run time, peak memory, clean exit, flakiness over
three runs, goals, kept results and trend). Not specced yet; planned after US1.

## Requirements *(mandatory)*

- **FR-001** (US1): The CI review runs on every `doctor --target`, read-only, through a `ci.Reader`
  (GitHub Actions now; other hosts later, 037).
- **FR-002** (US1): The analysis (workflow parsing, script resolution, classification, the
  findings) is pure and tested without network.
- **FR-003** (US1): Every CI review row is `WARN` or `INFO` or `PASS`, never `FAIL`.
- **FR-004** (US1): Log lines are read only from the failed step's section, at most 8 MiB, with no
  credential sent to the log URL.

## Allowed paths

- `internal/ci/*`
- `internal/cli/*`
- `docs/*`

## Success Criteria *(mandatory)*

- **SC-001**: On documcp (2026-11-27) the review reports the Security Audit job red on `main`
  with `zod <4.6.6 (severity: high)`, the Vercel deploy failure as information, and typecheck,
  lint, build, audit and commit lint as checks grading doesn't run.
