# Quickstart: validating research before build (033)

## Prerequisites

- Go on `PATH` (`$HOME/.local/go/bin`), this repository checked out.
- For the end-to-end checks: `./chalkeus forge up` and a configured fixture or target.

## 1. Replay on kept evidence (no cluster)

```sh
go test ./internal/diagnose/ -run Replay -v
```

Expected:
- documcp `20261125T163307Z` attempt 1 → `back_to_planning`, naming T003, `package-lock.json`
  and one `github.com` refusal by the target proxy (SC-001; text in
  [contracts/diagnosis.md](contracts/diagnosis.md)).
- fixture `specs/016-go-pilot/evidence/20261117T001332Z` A/1 → `retry`, no egress signal
  (SC-002).

## 2. Unit and docs checks

```sh
go vet ./... && go test ./...
```

Expected: pass, including the fixture-summary parity test, the reason-catalogue docs test and
the approvals vectors with `replan`.

## 3. Fixture run end to end

```sh
./chalkeus forge run          # the fixture; A/1 fails by induction
```

Expected: the retry question shows the diagnosis recommending retry; approving the retry
continues as before; `./chalkeus forge audit` results are unchanged (SC-002).

## 4. Back to planning

Pause a fixture run at A/1's retry, then (the human approves through the harness):

```sh
./chalkeus forge approve --run <id> replan
./chalkeus forge resume --run <id>
./chalkeus forge status --run <id>
```

Expected: state `stopped(back-to-planning)`, no watcher running, the grading namespace gone,
audit AU02/AU03 `n/a`; `./chalkeus forge run` starts a new run normally (SC-004).

## 5. Doctor

```sh
./chalkeus forge doctor --target documcp --item 001 --slice US1 --baseline
```

Expected (documcp as of 2026-11-25): a research-task warning for T003 and T004a, the measured
test-run time and the allowance; no check changes from pass to fail (SC-005).
