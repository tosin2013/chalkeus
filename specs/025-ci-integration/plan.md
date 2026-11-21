# Implementation Plan: Integration tests in CI

**Branch**: `025-ci-integration` | **Date**: 2026-11-21 | **Spec**: [spec.md](spec.md)

## Summary

The job runs the real pilot binary against two stand-ins:
- a **fake GitHub** (`cmd/fakegithub`: a REST subset plus git over `git http-backend`);
- a **stub harness** (a canned `opencode` in a test-only worker build).

It runs on kind on a GitHub-hosted runner. **Test endpoints** point the pilot and its images at
the fake. They're refused without `PILOT_TEST_MODE=1`, and a test run's audit marks it as not
evidence (AU12). Go tests in `test/integration/` drive four scenario groups and act as the owner
through the fake. A coverage floor holds the glue packages. Decisions: [research.md](research.md).

## Technical Context

- **Language:** Go for the pilot, the fake server and the harness; Python for the in-image
  publisher scripts (reading `GITHUB_API` and `GIT_BASE`); a shell script for the stub harness.
- **Dependencies:** go-github types (already a dependency), `git http-backend` (part of git),
  kind and Docker on the runner.
- **Testing:** `go test -tags integration ./test/integration/` in CI; unit tests for the new
  config rules, AU12, the stub and the fake's handlers.
- **Constraints:**
  - no secrets;
  - published images unchanged;
  - test endpoints refused without test mode;
  - under 45 minutes on a GitHub-hosted runner.

## Constitution Check

- **C-09 (images):** published images never contain the stub; test builds are local only ✓
- **C-10 (no telemetry):** nothing leaves the runner; the fake GitHub is local ✓
- **C-14 (hidden tests):** the scenarios' hidden tests are test fixtures; P11 still asserts the
  worker never sees them ✓
- **C-04:** the test acts as the owner through the fake, and the pilot still never merges ✓

## Project Structure

```text
cmd/fakegithub/                      the fake GitHub server (test only)
internal/config/                     test mode and endpoints (refused without test mode)
internal/audit/                      AU12 for test runs
pilot/images/worker/stub-opencode    the stub harness; stub-scenarios/<name>/ its canned outputs
pilot/images/tools/*.py              GITHUB_API and GIT_BASE
internal/images/                     test builds copy the stub
test/integration/                    scenarios (build tag integration) and fixtures
.github/workflows/integration.yml    weekly and on PRs touching the pilot
.github/coverage-floor.txt           the glue packages' floors
```
