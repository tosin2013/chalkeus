# Spec 001: Total

## User Scenarios

### User Story 1 — Add up a list of numbers

**Acceptance Scenarios**:

1. **Given** `[1, 2, 3]`, **When** `total` is called, **Then** it returns 6.
2. **Given** an empty list, **When** `total` is called, **Then** it returns 0.

## Requirements

- **FR-001**: `total(numbers: list[int]) -> int` is importable as `from gaps import total`.

## Allowed paths

- `src/gaps/*`
- `tests/*`
