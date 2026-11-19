# Contract: `pilot author watch`

```text
pilot author watch --target <name> [--once]
```

- **Refuses unless:**
  - the target sets `PILOT_AUTHOR_AUTO=on`;
  - `pilot up` has completed (the author proxy exists).

  When it refuses, it exits 2 with the reason.
- **Polls** every `PILOT_AUTHOR_INTERVAL_S` (default 120). `--once` does one pass and exits:
  - 0 when nothing was eligible or a PR opened;
  - 1 when a draft failed its check.
- **Prints one line per decision:** `skip NNN: <reason>`, `draft NNN: started <draft id>`,
  `draft NNN: DRAFT_OK, opened <url>`, or `draft NNN: <code>, no PR`.
- **Never** merges, approves a run, or writes the target repository.

## Profile keys (target profile)

| Key | Default | Meaning |
|---|---|---|
| `PILOT_AUTHOR_AUTO` | `off` | `on` lets `author watch` draft this target's items |
| `PILOT_AUTHOR_INTERVAL_S` | `120` | seconds between polls |
| `PILOT_AUTHOR_AUTO_ATTEMPTS` | `2` | author Jobs per draft (the second gets the first check's failure) |

## Eligibility (research R2)

All of these must hold:
- the author's readiness says `allow`;
- `hidden_files` is 0;
- no open companion PR has the head prefix `author/<NNN>-`;
- no earlier automatic draft exists for the same item and spec SHA-256.

## Author Job I/O (research R3)

- **In:** `/in/meta.json` (`role: author`, `draft_id`, `item`, `attempt`), `/in/prompt.md`, and
  `/in/workspace.tar` (`spec.md`, `plan.md`, `tasks.md`, `criteria.json`, `base/`, an empty
  `draft/`).
- **Out (frame):**
  - `report.json`: version, role, draft id, attempt, harness exit, spend, error, log tail
    (redacted);
  - `draft.tar`: the `draft/` tree only.

  Anything else is ignored.

## Draft evidence additions

`draft.json` gains:
- `trigger` (`manual` | `auto`);
- `jobs`: per attempt, its spend, exit and error;
- `watch_decision`.

`checks.jsonl` holds the A01–A05 rows for each Job.
