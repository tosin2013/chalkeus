# Quickstart: validating the Go pilot

## 0. Prerequisites

- A machine set up per `docs/operator-guide.md`.
- The Go toolchain (version per `go.mod`); `go version` should print it.

## 1. Automated parity (US1, shared vectors)

```bash
go test ./...                                           # includes the golden audit and vectors
PYTHONPATH=pilot python3 -m unittest discover -s pilot/tests -t pilot   # reference still green
python3 pilot/tools/docs_check.py
```

Expect:
- the golden audit test passes 86/86 rows on 8 records (contracts/parity.md);
- every vector case passes in both languages.

## 2. Gate (US1)

```bash
PILOT_IMPL=go pilot/pilot reset-fixture --yes
PILOT_IMPL=go pilot/pilot gate A ; echo $?   # 0, allow
PILOT_IMPL=go pilot/pilot gate B ; echo $?   # 10, refuse (DEP_UNSATISFIED:001)
```

Compare with `PILOT_IMPL=python`; the verdicts and reasons should be identical.

## 3. Fixture run (US2)

Follow the operator guide §7–§11 with `PILOT_IMPL=go`, at a terminal. Expect:
- the same prompts and pauses;
- the same exit codes;
- 11/11 after `pilot/pilot down`.

Then repeat with an agent and harness approvals (012 quickstart §3). Expect `channels=['harness']`.

## 4. GitHub loop (US3)

Run `specs/004-github-loop/quickstart.md` §2–§5 with `PILOT_IMPL=go`, in token mode and App
mode. Record the results in `specs/004-github-loop/evidence/README.md`.

## 5. Walk away (US4)

1. With `PILOT_GITHUB_APPROVALS=on`, run `pilot/pilot run` and close the terminal at the first
   pause.
2. Approve and merge on GitHub only.
3. Check that the run completes, and that `pilot/pilot status` shows no watcher afterwards
   (contracts/watcher.md).

## 6. Retirement (US5)

After the Python pilot is removed:
- `go test ./...` and `docs_check` both pass;
- `grep -rn specfoundry_pilot --exclude-dir=evidence .` finds only history notes;
- one more guide walkthrough completes with the audit at 11/11.
