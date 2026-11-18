# Contract: CLI changes

Every existing command, flag, exit code and message is unchanged when `--target` is not given
(FR-015).

| Command | Change |
|---|---|
| `pilot run --target <name> --item NNN` | starts an item run. `--item` requires `--target`. With `--target fixture` (or none), `--item` is refused: the fixture always runs A→B. |
| `pilot status [--run <id>]` | prints `kind`, `target` and `item` for item runs |
| `pilot resume --run <id>` / `approve --run <id> <step>` | unchanged flags; the target comes from `run.json` |
| `pilot audit [--run <id>]` | dispatches on the run's `kind` |
| `pilot gate --target <name> --item NNN` | readiness only (no cluster), with the same reason codes; `gate {A\|B}` is unchanged for the fixture |
| `pilot doctor --target <name> [--item NNN] [--baseline]` | added 2026-11-18: read-only preflight. Profile, cluster (including a one-request model key check), GitHub access, the target's roadmap (progress, `spec status` findings, what each item needs next), every readiness rule (no stop at the first failure), coverage; `--baseline` runs the item's baseline on the cluster without showing hidden-test output. Exit 0, or 1 if any check fails |
| `pilot github sync\|drift\|watch [--target <name>]` | acts on the target's repository and roadmap |
| `pilot app-token --check [--target <name>]` | probes the App on the target's repository (canary: this checkout) |
| `pilot targets` | lists configured targets: name, repository, toolchain, registries, companion; no secrets |
| `pilot reset-fixture` | refuses with `--target` other than `fixture` (exit 2, `NOT_A_FIXTURE`) |
| `pilot down` | also deletes registry proxies and leftover run-scoped grading namespaces, and verifies their absence (teardown rows T04, T05) |

## Mode approval text (item runs)

```text
Run mode for this pilot: {"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, ...}
  target greeting (owner/greeting-service@main), item 003, toolchain jdk21-maven
  egress: model host(s) 1 (from pilot.env); registries: repo.maven.apache.org
  one attempt at a time; every retry needs your confirmation; risk rating is out of scope in M1.
```

The GitHub decision post shows the same lines without the model host count.
