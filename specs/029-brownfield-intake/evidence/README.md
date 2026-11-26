# 029 evidence folder

Runs land in the active feature's `evidence/` folder, so fixture runs made while 029 was the
active feature were written here. They aren't 029 evidence, and their records aren't committed.

| Run | What | Outcome |
|---|---|---|
| 20261124T210910Z | Fixture run on the Linux amd64 machine (Go pilot, `step-by-step` mode) | **Abandoned.** A/2 passed and draft PR #67 opened; the run paused for its merge. On 2026-11-25 the owner asked for `reset-fixture --yes` before 031's T013 regression run, which closed #67. The run can't continue, and no watcher follows it. Its record (state `paused(awaiting-merge)`) is left as written, untracked |

031's T013 runs (20261125T162546Z, 20261125T170145Z) were also written here first and were moved
to [specs/031-macos-multiarch/evidence/](../../031-macos-multiarch/evidence/) by the owner.
