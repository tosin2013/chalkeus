# Contract: grading for item runs

`grade.py` keeps its frame output and check ids (G1–G5). New `meta.json` fields switch the
behaviour; the fixture sets none of them, so its grading is unchanged.

| `meta.json` field | Values | Effect |
|---|---|---|
| `mode` | `attempt` (default) \| `baseline` | baseline: no change bundle. G1 applies the base only, G2 and G3 are skipped, G4 must pass, and G5 must **fail**. The result is `pass` when both hold. |
| `test_command` | string | G4 and G5 run it through `sh -c`, from the project root, with the toolchain environment. Absent means pytest as today. |
| `hidden_mode` | `overlay` | G5 copies `/grading/overlay/` over the project, then runs `test_command`. Absent means the single-file pytest as today. |
| `toolchain` | name | recorded in the report |
| `spec_sha256` | hex | the SHA-256 of the spec text given to the worker (AU11 item variant) |
| `command_timeout_s` | int | default 180; from the catalog (600 for each catalog toolchain) |

## Grader pod (item runs)

- **Init containers:** the toolchain copy, then the input receiver (it waits for `/in/.ready`).
- **Mounts:** `/in` holds the snapshot and change bundles and `meta.json`; `/grading/overlay`
  is staged; `/deps` is the cache PVC, mounted read-only.
- **Maven arguments:** `-o -s /in/settings.xml -Dmaven.repo.local=/tmp/m2
  -Dmaven.repo.local.tail=/deps`.
- **Network:** none (namespace default deny).
- **Resources and deadline:** from the catalog, or the profile.

## Prefetch pod

The same image and toolchain as the grader.
- **Inputs:** the base snapshot plus `/grading/overlay` applied.
- **Command:** runs `test_command` with `-Dmaven.repo.local=/deps` (writable). Its exit code is
  recorded, not judged.
- **Network:** DNS and the target's registry proxy only.
