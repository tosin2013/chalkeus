# Quickstart: validating the GitHub loop

The prerequisites are a machine set up per `docs/operator-guide.md` and the fixture repository.

## 1. Automated

```bash
PYTHONPATH=pilot python3 -m unittest discover -s pilot/tests -t pilot
python3 pilot/tools/docs_check.py
```

These use fake GitHub responses: sync idempotence, drift cases, every approval rejection
reason, status-PR diff check, AU03 rules, App JWT shape (with a test key).

## 2. Sync and links (US1, SC-001, SC-002)

1. `pilot/pilot reset-fixture`, then `pilot/pilot github sync`. Check: 2 issues, each with one
   `status:planned` label and its task checklist.
2. Run `pilot/pilot github sync` again. It should report 0 created, 0 changed.
3. Change one issue's label by hand, then `pilot/pilot github drift` (exit 1), then
   `github sync`. The label is restored.

## 3. GitHub approvals, token mode (US2, US3, SC-003, SC-004)

Set `PILOT_GITHUB_APPROVALS=on` and `PILOT_APPROVERS=<your login>`. The publisher token must
come from a second account, not yours: your own token makes your approvals `publisher_identity`
(docs/github-loop.md §2). Start the run with no terminal:

```bash
setsid pilot/pilot run </dev/null
pilot/pilot github watch --run <id>
```

1. On the tracking issue, reply `/approve mode`. The run starts.
2. At the retry post, check the refusal cases before approving:
   - reply `/approve mode` (wrong step);
   - reply from a second account not in the list;
   - edit an old comment into `/approve retry`;
   - approve, then `/withdraw retry`;
   - use a comment posted with the publisher token (`ghops` test command).

   The run must stay paused for each. Then reply `/approve retry`.
3. Merge A's PR on GitHub. A status-change PR appears. Push an extra change to it, merge it:
   the pilot refuses (`STATUS_PR_EDITED`). Reset and repeat, then merge it unedited. B proceeds.
4. Finish, then `pilot/pilot down` and `pilot/pilot audit`. Expect 11/11, with AU09 and AU10
   showing `channels=['github']` and your login. `github drift` should be empty.

## 4. GitHub App (US4, SC-005, SC-007)

Follow the App setup in the docs only, timed and with no outside lookups. Then
`pilot/pilot app-token --check` should pass the probe on the fixture and fail on the canary.
Repeat §3 with `PILOT_PUBLISHER_CREDENTIAL=app`: PRs and posts are by `<app>[bot]`, and the
audit is 11/11.

## 5. Terminal run unchanged

One terminal run with GitHub approvals off. Same prompts as the guide, and AU09 and AU10
show `tty`.
