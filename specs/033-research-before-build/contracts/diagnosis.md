# Contract: the diagnosis (033)

## `attempts/<task>-<n>/diagnosis.json`

Shape in [data-model.md](../data-model.md#diagnosis). Written once per failed attempt; never
rewritten. A run started before 033 has none, and nothing requires it.

## Text shown with the retry question, in the summary and on the tracking issue

Plain text, at most ~20 lines, same text in all three places:

```text
  diagnosis of attempt <n> (<task>): <ended> after <seconds> s, <tokens> tokens, $<cost>
  signals:
    - <detail> (<source>)
    ...
  points to: <unknown work | an environment need | nothing outside the attempt>
  recommended: <retry | raise the budget | back to planning>
    <one line per route, e.g.:>
    back to planning: resolve <T003> in the plan's research (/speckit-plan), then /speckit-tasks
    back to planning: change <package-lock.json> in your planning PR, not in an attempt
    raise the budget: FORGE_WORKER_DEADLINE_S in the target profile, then a new run
  your choice: approve retry, or end the run with `replan` (you decide; forge does neither)
```

Example (documcp run `20261125T163307Z`, attempt 1; SC-001):

```text
  diagnosis of attempt 1 (001): deadline after 2400 s, 277364 tokens, $0.0
  signals:
    - task T003 reads as research: "Locate the suites that leave handles open" (prompt.md)
    - task T004a reads as research: "Find the test that writes …" (prompt.md)
    - package-lock.json changed, outside the allowed paths (dependency lock file)
    - github.com refused 1× by the target proxy (proxy-denials.log)
  points to: unknown work, an environment need
  recommended: back to planning
    back to planning: resolve T003, T004a in the plan's research (/speckit-plan), then /speckit-tasks
    back to planning: change package-lock.json in your planning PR, not in an attempt
  your choice: approve retry, or end the run with `replan` (you decide; forge does neither)
```

Example (fixture A/1, induced failure; SC-002):

```text
  diagnosis of attempt 1 (A): grade_fail after <s> s, <tokens> tokens, $<cost>
  signals:
    - grading failed: G5_hidden_tests (grade.json)
  points to: nothing outside the attempt
  recommended: retry
  your choice: approve retry, or end the run with `replan` (you decide; forge does neither)
```

## Summary section

`summary.md` gains `## Diagnosis` after the Attempts table, one sub-block per diagnosed attempt
with the same text. The section is omitted when no attempt has a diagnosis, so committed
summaries regenerate unchanged (`TestFixtureModeAndSummaryUnchanged`).

## Determinism

Same evidence files → byte-identical `diagnosis.json` and text. No clock, no network, no model.
