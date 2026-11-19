# Contract: audit for item runs

- **Same check ids, AU01–AU11.** `kind=item` selects the variants below; the other checks are
  unchanged.
- **Older records:** fixture runs and records without `kind` keep today's checks, and the
  golden data is unchanged.

| Check | Item-run rule | Detail |
|---|---|---|
| AU01 | Only `NNN` was gated `allow`. Its readiness rows are all `pass`. No other item has an attempt event. | `item=NNN readiness=k/k other_attempts=0` |
| AU02 | The baseline passed before the first attempt. The grade results are `fail*` then exactly one `pass`, and that pass is the last attempt. | `baseline=pass order=[...]` |
| AU03 | Exactly 1 task PR and at most 1 status PR, all drafts authored by the publisher identity, on the target repository | as today |
| AU11 | No attempt was induced. Every attempt's prompt contains the spec whose SHA-256 is `meta.spec_sha256`, and that equals the SHA-256 of `spec.md` at the base revision. | `attempts=n induced=0` |

## New check rows (`checks/results.jsonl`, item runs)

| Row | Rule |
|---|---|
| P06r | From a probe in the item worker's policy, a declared registry host is reachable through the target's worker proxy and an undeclared host (`example.invalid`) is refused; the prefetch proxy's allowlist is exactly the declared registries |
| P11 | No overlay file name or content hash appears in the worker namespace's ConfigMaps, staged-input list or prompt |
| P12 | The grading namespace's grader pods have no egress rule, and the prefetch pod's rule allows only DNS and the registry proxy |

- **AU04** additionally requires P11 and P12 for item runs.
- **AU07** additionally requires teardown rows T04 (no registry proxy left) and T05 (no grading
  namespace or PVC left).
