# 024 evidence: test-author automation

Live validation on [Stockroom](https://github.com/tosin2013/stockroom), on this machine (kind).
The owner started every watcher in their own terminal.

| When | What | Result |
|---|---|---|
| 1st `author watch --once`, item 009 | both author Jobs refused by the cluster: the command didn't load the image digests | no PR; fixed in #112 |
| 2nd, item 009 | the author Job ran; A01 found no pod (it looked for `app=worker`; the author pod is `app=author`) | **fail closed: no PR**; fixed in #113 |
| 3rd, item 009 | author Job, A01–A05, then the check on the cluster | **DRAFT_OK**; companion PR #10 opened, merged by the owner |
| run [20261119T190457Z](20261119T190457Z/), item 009 | passed grading, but **P11 failed**: the drafted file reused the path 009's tasks tell the worker to create, so the grader's overlay replaced the worker's own test | abandoned (stockroom#43 closed by the owner); fixed in #115 (`DRAFT_PATH_IN_SPEC`); the owner renamed the tests (companion #11) |
| step mode, item 010 | step mode started the drafting itself (021); the check passed | companion PR #12, merged by the owner; 010 then verified, **audit 11/11 after `down`** ([021 evidence](../../021-step-mode/evidence/20261119T223729Z/)) |

## Success criteria

| Criterion | Status | Evidence |
|---|---|---|
| SC-001 | met | 009 (`author watch`) and 010 (step mode): from "spec and tasks merged" to an open companion PR with no operator action, each passing the same check as a manual draft |
| SC-002 | met | every PR above opened only after A01–A05 passed (the watcher refuses otherwise, as the 2nd attempt showed). The check rows are in the draft evidence, which this session doesn't read |
| SC-003 | met | no PR opened without a passing check (attempts 1 and 2); no item drafted twice for one spec version (a fixed redraft loop in step mode, #125) |
| SC-004 | met (by design) | `author watch` refuses to start unless `PILOT_AUTHOR_AUTO=on`, and step mode drafts only when it's on (`TestManagerDraftsAndAsksForTests`); `pilot doctor` shows the setting. No live run with it off was tried |
| SC-005 | met | 010's run with drafted tests: baseline, attempt, P11 pass, audit 11/11 after `down` |

The draft evidence (`drafts/`) is not committed: the session that drives runs never reads it
(024 FR-008). The owner may commit it.
