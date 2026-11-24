# Quickstart: validating 030

Prerequisites: `./chalkeus forge up` done; `go vet ./... && go test ./...` and
`python3 -m unittest discover pilot/images/tests` green.

## 1. Deadline-stopped attempt keeps its work (US6, MVP)

1. In a scratch target profile (or the fixture with a test override), set
   `FORGE_WORKER_DEADLINE_S=300` and give it an item that takes longer.
2. Run it and approve the mode. Expect: attempt `failed (deadline)`; in
   `attempts/<task>-1/`: `report.json` with `stopped_by: "deadline"` and non-null `spend`;
   `change.bundle` when the agent changed files; the run summary's Cost/Tokens filled; the retry
   decision text states time, tokens, cost and changed files.

## 2. Raise and take back a budget (US1)

1. Add `FORGE_WORKER_DEADLINE_S=2400`, `FORGE_GRADER_CPU=3`, `FORGE_COMMAND_TIMEOUT_S=900`.
   `./chalkeus forge targets` lists them as raised. A value below the floor → `TARGET_INVALID`.
2. Start a run: the mode approval shows each role's budget with the raised keys marked; the
   worker/prefetch/grader Jobs carry the limits (`attempts/*/manifests.yaml`); `run.json.budget`
   matches.
3. Remove the lines, resume the paused run: it keeps the raised budget. A new run uses floors.

## 3. Capacity (US2, US3)

1. `FORGE_GRADER_CPU=64` on a laptop cluster: `forge doctor --target <t>` → `FAIL
   CAPACITY_INSUFFICIENT` with the local message; `forge run` refuses to start.
2. With a kube context that can't list nodes: doctor warns; a run whose pod can't be scheduled
   stops within 2 minutes with `CAPACITY_UNSCHEDULABLE`, no attempt counted.

## 4. documcp (SC-004, live)

`FORGE_WORKER_DEADLINE_S=2400` in `targets/documcp.env`, then a new US1 run (026 T048). Expect
either a draft PR, or a deadline stop whose evidence shows how far the agent got.
