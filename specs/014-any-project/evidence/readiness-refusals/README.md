# Readiness refusals on Stockroom (014 T053 step 1, SC-002)

On 2026-11-19, on this machine (pilot at main `b2fc83b` plus the `gate --root` change, `spec`
0.3.0). Each case is a local copy of
[tosin2013/stockroom](https://github.com/tosin2013/stockroom) at `79d337e`, the commit where
item 008 was specced and ready, with one local commit that breaks one rule. Nothing was pushed.
`run.sh` repeats it:

```sh
specs/014-any-project/evidence/readiness-refusals/run.sh <stockroom checkout> <scratch dir>
```

Each case runs `pilot/pilot gate --target stockroom --item NNN --root <copy>`. The hidden tests
are still read from the companion repository, and open PRs from GitHub.
[results.txt](results.txt) has one line per case: the exit code, the expected code and the
refusal reasons.

| Case | Broken rule | Refused with |
|---|---|---|
| control | none | `allow` (exit 0) |
| not-on-roadmap | item 099 doesn't exist | `ITEM_NOT_ON_ROADMAP` |
| spec-missing | `spec.md` removed | `ITEM_ERROR_DIAGNOSTIC:SS_STAGE_FILE_MISSING` |
| no-spec-dir | the roadmap's `Spec dir` line removed | `ITEM_ERROR_DIAGNOSTIC:SS_SPEC_REFERENCE_MISSING` |
| tasks-missing | `tasks.md` removed | `ITEM_TASKS_MISSING` |
| open-q | a `[NEEDS CLARIFICATION` marker | `ITEM_OPEN_QUESTIONS` |
| no-paths | `## Allowed paths` removed | `ITEM_NO_ALLOWED_PATHS` |
| wide-paths | an allowed path of `*` | `ITEM_PATHS_TOO_WIDE` |
| no-hidden | the item renumbered 009, which has no companion folder | `ITEM_NO_HIDDEN_TESTS` |
| too-large | a 60 MB file (limit 50 MB) | `INPUT_TOO_LARGE` |
| dep | dependency 006 set back to `specced` | `DEP_UNSATISFIED:006` |
| unreadable | the roadmap replaced with garbage | `UNEVALUABLE:exit_2` |

Every broken rule refused with exit 10 before any cluster work.

**Not exercised here:**
- **A missing `spec.md` never reaches `ITEM_SPEC_MISSING`.** `spec status` reports an error
  diagnostic for it first, and the gate refuses on that (contract order 0, before rule 2).
  `ITEM_SPEC_MISSING` stays as a backstop with unit tests (`internal/readiness`).
- **`HIDDEN_TESTS_IN_TARGET` and `OPEN_ITEM_PR`** were not run here. The first needs a hidden
  test's content copied into the target, which the session that drives runs must never read.
  The second needs an open pilot PR on the target. Both have unit tests.
