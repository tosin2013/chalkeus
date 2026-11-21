# notes — Spec Roadmap

Throwaway target for specfoundry's integration tests. Two independent items.

Status legend (lifecycle): **undecided** · **needs-info** · **planned** ·
**specced** · **in-progress** · **implemented** · **verified** · **deferred** ·
**abandoned**.

---

## Planned Specs

### 001 — Word count  [status: planned]

- **Description:** A `word_count` function.
- **Outcome:** `word_count(text)` counts whitespace-separated words.
- **Scope (in):** `src/notes/words.py`, its tests, and this spec's tasks.
- **Scope (out):** line counting (002).
- **Depends on:** none
- **Spec dir:** specs/001-word-count

### 002 — Line count  [status: planned]

- **Description:** A `line_count` function.
- **Outcome:** `line_count(text)` counts lines.
- **Scope (in):** `src/notes/lines.py`, its tests, and this spec's tasks.
- **Scope (out):** word counting (001).
- **Depends on:** none
- **Spec dir:** specs/002-line-count
