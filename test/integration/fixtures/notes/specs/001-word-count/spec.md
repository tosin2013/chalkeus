# Spec 001: Word count

## User Scenarios

### User Story 1 — Count the words in a text

**Acceptance Scenarios**:

1. **Given** the text `"a b  c"`, **When** `word_count` is called, **Then** it returns 3.
2. **Given** an empty text, **When** `word_count` is called, **Then** it returns 0.

## Requirements

- **FR-001**: `word_count(text: str) -> int` is importable as `from notes import word_count`.

## Allowed paths

- `src/notes/*`
- `tests/*`
