# Spec 002: Line count

## User Scenarios

### User Story 1 — Count the lines in a text

**Acceptance Scenarios**:

1. **Given** the text `"a\nb\n"`, **When** `line_count` is called, **Then** it returns 2.

## Requirements

- **FR-001**: `line_count(text: str) -> int` is importable as `from notes import line_count`.

## Allowed paths

- `src/notes/*`
- `tests/*`
