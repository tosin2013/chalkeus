# Contract: `.acceptance-map.json`

Lives at `<path>/items/<NNN>/.acceptance-map.json`. Never copied into the grader overlay and not
counted as a hidden test.

```json
{
  "version": 1,
  "spec_sha256": "<sha256 of the item's spec.md>",
  "criteria": {
    "US1-AS1": ["tests/test_bundled_flags.py::test_it_is_valid"],
    "FR-003":  ["tests/test_bundled_flags.py::test_unknown_letters_named"]
  },
  "uncovered": ["US2-AS2"]
}
```

- **Criteria ids:** `US<n>-AS<m>` is acceptance scenario m (1-based) of user story n; `FR-<nnn>`
  is a functional requirement. `author start` lists them in `criteria.json`.
- **Valid when:** every key is a known id; every file part of a test reference exists in the
  draft; `uncovered` equals the known ids not in `criteria`.
- **At run time:** readiness reports `uncovered` and whether `spec_sha256` still matches; it
  never refuses because of the map.
