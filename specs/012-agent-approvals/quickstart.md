# Quickstart: validating 012

## 1. Automated checks (any machine)

```sh
PYTHONPATH=pilot python3 -m unittest discover -s pilot/tests -t pilot   # all pass, incl. approvals and committed evidence
python3 pilot/tools/docs_check.py                                        # 0 problems
```

`test_committed_evidence` audits every committed successful run record and requires it to pass
with no channels (SC-004).

## 2. Refusal cases without an agent (set-up machine, SC-002)

Start a run from a shell with no terminal, for example `setsid pilot/pilot run < /dev/null`.
Then check each case. The run must not advance in any of them.

| Case | Command | Expect |
|---|---|---|
| no approval | `setsid pilot/pilot resume --run <id> </dev/null` | pauses again, exit 5 |
| wrong step | `pilot/pilot approve --run <id> retry` while waiting for mode | exit 2, names the pending step |
| merge | `pilot/pilot approve --run <id> reconcile` before merging A | exit 4, "merge … on GitHub" |
| used once | approve mode, resume, then at the retry pause resume again without approving | pauses at retry |
| not paused | `pilot/pilot approve --run <finished-id> mode` | exit 2 |
| terminal overrides | approve retry, then `pilot/pilot resume` in a terminal and answer `N` | paused; a later headless resume pauses again |
| direct edit | ask the agent to write `pilot/runs/<id>/approvals/pending.json` | the harness refuses |

## 3. Agent-driven run (owner, SC-001 and SC-005)

In each harness (Claude Code, then OpenCode), from the repo root:

1. `pilot/pilot reset-fixture` (in your own terminal), then start the agent.
2. Ask: "Run the pilot using the pilot skill."
3. At each pause, the agent explains and asks you. Say yes. **Check that the harness then asks
   you before `pilot/pilot approve` runs** (SC-005).
4. When A's PR is published, merge it on GitHub and tell the agent.
5. When the run succeeds: `pilot/pilot down`, then `pilot/pilot audit --run <id>`.

Expected:
- the audit passes 11/11;
- AU09 shows `channels=['harness']` and AU10 shows `channels=['harness']`;
- `events.jsonl` has an `approval_channel … channel=harness` event for mode, retry and
  reconcile.

Copy the run's record to `specs/012-agent-approvals/evidence/<run-id>/`.

## 4. Terminal run unchanged (SC-006)

Run `pilot/pilot run` and `resume` in a real terminal. The prompts and exit codes should
match `docs/operator-guide.md` §8–9, and AU09/AU10 should show `channels=['tty']`.
