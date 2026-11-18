You are a coding agent working in a git repository (the current directory).
Implement the specification below completely, then mark every task in the task list as
done by changing `- [ ]` to `- [X]` in that tasks.md file.

Rules:
- Change only files matching these paths (`*` also matches `/`):
  - `cli_schema_mcp/src/tools/validate_command_tool.py`
  - `tests/test_tools.py`
  - `specs/006-bundled-short-flags/tasks.md`
- The project uses the python3.13 toolchain. Check your work with: `PYTHONPATH=. python -m pytest -q -p no:cacheprovider --cov=cli_schema_mcp --cov-fail-under=80 tests && bandit -q -r cli_schema_mcp -ll`
- Add or extend tests for your change in the project's test files (`tests/*`). Keep every existing
  test passing unchanged: the grader re-runs the original versions of any test file you edit or delete.
- Only the project's existing dependencies are available to the grader; do not add new ones.
- Do not run git commands; the harness commits your work.
- When finished, stop.

# Specification

# Feature Specification: validate_command accepts bundled short flags

**Feature Branch**: `fix/006-bundled-short-flags`

**Created**: 2026-11-17

**Status**: Draft

**Input**: Roadmap entry 006. `validate_command` reports `podman run -it fedora bash` as invalid
("`-it` is not a valid flag") although the podman schema defines both `-i` (`--interactive`)
and `-t` (`--tty`). Bundled short flags are common (`-it`, `-xzf`, `-ef`), so agents that trust
the tool rewrite correct commands.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - A correct command with bundled short flags validates (Priority: P1)

As an AI agent that validates a command before running it, I want `podman run -it fedora bash`
to be reported valid, so that I don't rewrite a correct command into a worse one.

**Why this priority**: This is the whole item. A false "invalid" on a correct command is the
opposite of what the tool is for.

**Independent Test**: Call `validate_command("podman", "5.4", "podman run -it fedora bash")`
and check `valid` is `true` and `errors` is empty.

**Acceptance Scenarios**:

1. **Given** a subcommand whose schema defines the shorthands `-i` and `-t`, **When** the
   command contains the token `-it`, **Then** the result is `valid: true` with no errors.
2. **Given** the same subcommand, **When** the token is `-ti`, **Then** the result is the same
   as for `-it` (letter order doesn't matter).
3. **Given** a subcommand with shorthands `-i` and `-t`, **When** the command contains `-it`
   and also a separate unknown long flag `--bogus`, **Then** the result is `valid: false` with
   exactly one error, naming `--bogus`.

---

### User Story 2 - A bundle with an unknown letter is still caught (Priority: P1)

As an agent, I want a bundle with a letter the subcommand doesn't define to stay invalid, and
to be told which letter, so that I can fix just that part.

**Why this priority**: Accepting bundles must not open a hole for hallucinated flags.

**Independent Test**: Call `validate_command("podman", "5.4", "podman run -iZ fedora bash")`,
where `-Z` isn't a shorthand of `podman run`, and check `valid` is `false`.

**Acceptance Scenarios**:

1. **Given** a subcommand with shorthand `-i` but no `-Z`, **When** the command contains
   `-iZ`, **Then** the result is `valid: false` with exactly one error, and that error contains
   both the original token `-iZ` and the unknown letter `-Z`.
2. **Given** a bundle where several letters are unknown (for example `-iZQ`), **When** it is
   validated, **Then** there is still exactly one error for that token, and it names every
   unknown letter (`-Z` and `-Q`).

---

### Edge Cases

- A single-dash token that is itself a defined flag name or shorthand (for example a CLI that
  defines `-version` as a flag) is accepted as that flag and never split.
- A two-character token such as `-i` is a plain shorthand, not a bundle; it is checked as today.
- Long flags (`--tty`), `--flag=value`, the bare `-`, and everything after `--` behave exactly
  as today.
- A bundle is only expanded when every letter is a single-letter shorthand of the matched
  subcommand. Attached values (`-p8080`, `-ojson`) are out of scope: they are reported invalid
  as today.
- The rest of the response (`status`, `tool`, `version`, `matched_version`, `command_string`,
  `subcommand_context`, `message`) keeps its current shape and meaning (C-04).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: A single-dash token longer than two characters that is not itself a defined flag
  name or shorthand MUST be treated as a bundle of single-letter shorthands.
- **FR-002**: A bundle MUST be valid when every letter is a shorthand defined for the matched
  subcommand.
- **FR-003**: A bundle with one or more unknown letters MUST produce exactly one error. The
  error MUST contain the original token and each unknown letter written as `-<letter>`.
- **FR-004**: All other validation behaviour MUST stay unchanged; the existing tests in
  `tests/test_tools.py` MUST keep passing.
- **FR-005**: The tool MUST stay binary-free and model-agnostic (C-02, C-05): only the schema is
  used.

### Key Entities

- **Bundle**: a token such as `-it` standing for several single-letter shorthands (`-i -t`).

## Allowed paths

- `cli_schema_mcp/src/tools/validate_command_tool.py`
- `tests/test_tools.py`

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: `podman run -it fedora bash` validates as `valid: true` for every podman schema
  version in the registry (5.2, 5.3, 5.4).
- **SC-002**: 100% of bundles containing an unknown letter are reported invalid, naming the
  unknown letters.
- **SC-003**: The full existing test suite passes unchanged, with coverage still at or above 80%.

## Assumptions

- Shorthands in the schema are stored as `-<letter>` (for example `"shorthand": "-i"`), as in
  every schema in `schemas/` today.
- Only the matched subcommand's flags are used. Inherited global flags are already merged into
  each node by the extractor.

# Plan

# Implementation Plan: validate_command accepts bundled short flags

**Branch**: `fix/006-bundled-short-flags` | **Date**: 2026-11-17 | **Spec**: [spec.md](spec.md)

## Summary

`validate_command` compares each flag token with the matched subcommand's flag names and
shorthands. A bundled token like `-it` matches neither. After a token fails the exact match,
check whether it is a single-dash bundle whose letters are all known shorthands; if so it is
valid, otherwise report one error naming the unknown letters.

## Technical Context

**Language/Version**: Python 3.12+ (C-01)
**Primary Dependencies**: none new
**Testing**: pytest (`tests/test_tools.py`, the existing `_patched(SAMPLE_SCHEMA)` helper)
**Constraints**: binary-free (C-02); structured response shape unchanged (C-04)

## Constitution Check

- C-02 binary-free: the change reads the schema only. ✅
- C-04 typed responses: same keys; error strings stay a list of str. ✅
- C-05 model-agnostic: no model-specific logic. ✅

## Design

- Keep `_parse_flags_from_string` as is (it returns tokens like `-it`).
- In `validate_command`, build `short = {shorthand letters}` from the node's flags.
- For each used flag not in `valid_flag_names`:
  - if it starts with a single `-`, has length > 2, and is not `--…`: compute the letters not in
    `short`. None → valid (no error). Some → one error:
    `"<token> is not a valid flag for '<context>' in version <v> (unknown: -Z, -Q)"`.
  - otherwise: the existing error.

## Project Structure

```text
cli_schema_mcp/src/tools/validate_command_tool.py   # change
tests/test_tools.py                                 # new tests
```

# Task list (`specs/006-bundled-short-flags/tasks.md`)

---
description: "Task list for 006: validate_command accepts bundled short flags"
---

# Tasks: validate_command accepts bundled short flags

**Input**: `specs/006-bundled-short-flags/` (spec.md, plan.md)

**Tests**: Yes. Unit tests in `tests/test_tools.py`, using the existing `_patched` helper with a
sample schema that defines shorthands `-i` and `-t`.

## Phase 1: User Story 1 - bundled known shorthands validate (P1)

- [ ] T001 [US1] Add tests in `tests/test_tools.py`: `-it` and `-ti` validate as `valid: true`
  with no errors; `-it --bogus` gives exactly one error naming `--bogus`
- [ ] T002 [US1] In `cli_schema_mcp/src/tools/validate_command_tool.py`, accept a single-dash
  token longer than two characters when every letter is a known single-letter shorthand of the
  matched subcommand (FR-001, FR-002); a token that is itself a defined flag is never split

## Phase 2: User Story 2 - unknown letters are still caught (P1)

- [ ] T003 [US2] Add tests in `tests/test_tools.py`: `-iZ` gives exactly one error containing
  `-iZ` and `-Z`; `-iZQ` gives one error naming `-Z` and `-Q`
- [ ] T004 [US2] In `cli_schema_mcp/src/tools/validate_command_tool.py`, report one error per
  bundle with unknown letters, naming the original token and each unknown letter (FR-003)

## Phase 3: Polish

- [ ] T005 Run the full test suite (`python -m pytest`) and confirm every existing test still
  passes (FR-004)
