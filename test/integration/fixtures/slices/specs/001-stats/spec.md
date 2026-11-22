# Spec 001: Stats

## User Scenarios

### User Story 1 — Add up a list of numbers

**Acceptance Scenarios**:

1. **Given** `[1, 2, 3]`, **When** `total` is called, **Then** it returns 6.

### User Story 2 — Average a list of numbers

**Acceptance Scenarios**:

1. **Given** `[2, 4]`, **When** `mean` is called, **Then** it returns 3.

## Requirements

- **FR-001**: `from stats.total import total` and `from stats.mean import mean` work.

## Allowed paths

- `src/stats/*`
- `tests/*`

## Allowed paths (US1)

- `src/stats/total.py`
- `tests/test_total.py`

## Allowed paths (US2)

- `src/stats/mean.py`
- `tests/test_mean.py`
