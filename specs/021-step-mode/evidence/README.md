# 021 evidence: step mode, the laptop manager loop

Records of the runs step mode started on [Stockroom](https://github.com/tosin2013/stockroom),
on this machine (kind), and step mode's own decisions log.

| Run | Item | Started | Answers | Result | Notes |
|---|---|---|---|---|---|
| [20261119T202820Z](20261119T202820Z/) | 009 (delete a product) | by step mode after the owner's "pilot/go" label on stockroom#41 (`run_started_by step`) | **mode over `tty`**: the run inherited step mode's terminal and the owner typed `y` there; item PR stockroom#44 and status PR #45 merged on GitHub | succeeded; audit 10/10 (no `down` followed it) | The terminal prompt was a bug: step mode's subprocesses now have no controlling terminal and `PILOT_NO_TTY=1` (#122). 009's hidden tests were drafted by `author watch` (024) and renamed by the owner (companion #11) after the collision fixed in #115. |
| [20261119T223729Z](20261119T223729Z/) | 010 (update a product's stock) | by step mode after the owner's label on stockroom#47 | **every answer on GitHub**: hidden tests drafted by step mode (024) in the sandbox, companion PR #12 merged; `/approve mode` on stockroom#48 (`channel=github`); item PR #49 and status PR #50 merged | succeeded; **audit 11/11 after `down`** | Attempt 1 passed (79k tokens); all 15 isolation checks passed, including P06r and P11. The first item carried end to end by step mode. |

[step/stockroom/decisions.jsonl](step/stockroom/decisions.jsonl) records every action step mode
took, and each `step_stopped` when the owner restarted it. [step/stockroom/](step/stockroom/)
also holds the doctor and run logs of each item.

## Success criteria

| Criterion | Status | Evidence |
|---|---|---|
| SC-001 | met | 010: labelled, drafted, doctored, run and verified with only the owner's GitHub answers; audit 11/11 after `down` |
| SC-002 | met (unit tests) | `TestManagerApproverLabelOnly`, `TestProposedBy` (the last adder decides, across pages); no live non-approver label was tried |
| SC-003 | met | live: step mode was stopped and restarted several times, including while 009's run waited for its merges (`step_stopped` entries), and never started a second run or draft; plus `TestManagerCarriesALabelledItem` |
| SC-004 | met (unit tests) | `TestManagerDoctorFailureCommentsOnceUntilAChange`; no live doctor failure occurred |

## Found live and fixed during validation

| PR | Finding |
|---|---|
| #121 | step mode required `PILOT_GITHUB_ISSUES` although runs keep issues on with approvals |
| #122 | a run step mode started could ask on the owner's terminal |
| #123 | the wait line didn't say what to do, or where |
| #124 | a status PR merged between two passes was never shown |
| #125 | seam audit ([audit-20261119.md](../audit-20261119.md)): which label to add and where, no redraft loop, no clones while a run is in flight |
| #127 | with nothing to propose, it didn't name the open planning PR |
| #128 | it didn't explain itself or say what's next |

The draft evidence of step mode's automatic drafts (`drafts/`) is not committed: the session
that drives runs never reads it (024 FR-008).
