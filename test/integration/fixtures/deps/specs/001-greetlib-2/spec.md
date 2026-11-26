# Spec 001: Greetlib 2

## Manifest change

- Commit: `BUMP_COMMIT`
- Files: `requirements.txt`
- From the bot pull request that bumps greetlib from 1.0 to 2.0

## User Scenarios

### User Story 1 — Greet with greetlib 2

**Acceptance Scenarios**:

1. **Given** greetlib 2 is pinned, **When** the code is loaded, **Then** it supports major 2.
2. **Given** the name `Ada`, **When** `greet` is called, **Then** it returns `Hello, Ada!`.

## Requirements

- **FR-001**: `greet.SUPPORTED` includes 2, and `greet(name)` returns `Hello, <name>!`.

## Allowed paths

- `src/greet/*`
- `tests/*`
