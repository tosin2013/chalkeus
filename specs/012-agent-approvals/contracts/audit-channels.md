# Contract: approval channels in the evidence and audit (amends 002 `contracts/evidence-record.md`)

## Events

Every decision appends one `approval_channel` event (stage `approval`):
`step=<mode|retry|reconcile> approved=<true|false> channel=<tty|harness|none>`.
`pilot approve` appends `approval_recorded` (`step=<s> channel=harness`).

## Audit

| Check | Pass rule | Detail |
|---|---|---|
| AU09 | unchanged: every attempt with n>1 is preceded by a `retry_confirmed` for the same task and n | adds `channels=[…]`: the channel of each approved retry, in order |
| AU10 | unchanged: a `mode_confirmed` comes before the first attempt | adds `channels=[…]`: the channel of the approved mode |

- Records made before 012 have no `approval_channel` events. They pass as before and show
  `channels=[]`; no channel is inferred.
- `harness` means a human approved through the agent harness's permission prompt. It's
  weaker than `tty` (see research R3). A reviewer reads the detail to tell them apart.
- 004 will add `github` as a third channel value.
