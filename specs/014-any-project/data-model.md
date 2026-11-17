# Data model: Any project, any item (014)

The entities and how they change; file formats are in [contracts/](contracts/).

## Target profile

One per target repository, on the operator's machine ([contracts/target-profile.md](contracts/target-profile.md)).

| Field | Key | Rule |
|---|---|---|
| name | file name `targets/<name>.env` | `^[a-z0-9][a-z0-9-]{0,30}$`; `fixture` is reserved for the implicit target |
| repository | `PILOT_TARGET_REPO` | `owner/name`; not this checkout's origin; not any profile's companion repository |
| branch | `PILOT_TARGET_BRANCH` | default `main` |
| toolchain | `PILOT_TOOLCHAIN` | a name in `pilot/images/toolchains.json` whose image is locked; or a free label when `PILOT_TOOLCHAIN_IMAGE` is set |
| own toolchain image | `PILOT_TOOLCHAIN_IMAGE` | optional; `registry/name@sha256:<64 hex>`; must meet contracts/toolchain-image.md |
| test command | `PILOT_TEST_COMMAND` | non-empty; runs from the repository root; exit 0 = pass |
| registry hosts | `PILOT_REGISTRY_HOSTS` | comma-separated hostnames (`^[A-Za-z0-9.-]+$`); may be empty |
| companion repository | `PILOT_COMPANION_REPO` | `owner/name`, private, readable by the operator |
| GitHub issues / approvals | `PILOT_GITHUB_ISSUES`, `PILOT_GITHUB_APPROVALS` | `on`/`off`; issues default `off` for a new target |
| resource overrides | `PILOT_GRADER_MEMORY`, `PILOT_GRADER_DEADLINE_S`, `PILOT_CACHE_SIZE` | may only raise the catalog default, up to caps |
| input limit | `PILOT_MAX_INPUT_MB` | default 50 |
| publisher (token mode) | file `targets/<name>.publisher.env` | mode 0600; fine-grained token for that repository only |

**Implicit `fixture` target:** `pilot.env`'s `PILOT_FIXTURE_REPO`/`PILOT_FIXTURE_BRANCH`,
toolchain `python`, the built-in grading, and today's behaviour.

## Prepared item

Read on a fresh clone of the target at its branch head.

| Attribute | Source |
|---|---|
| id `NNN` | `--item`; must be on the roadmap |
| status, dependencies | `spec status --json` |
| spec dir | the item's `Spec dir` in the status JSON |
| spec, tasks, plan | `<spec dir>/spec.md`, `tasks.md`, `plan.md` (plan optional) |
| allowed paths | the `## Allowed paths` list in `spec.md` ([contracts/item-spec.md](contracts/item-spec.md)) |
| hidden tests | `items/NNN/` in the companion repository |

**Readiness:** passes when all of R3's rules pass. Otherwise the run refuses with a reason
code ([contracts/readiness.md](contracts/readiness.md)).

## Run (changes)

`run.json` gains:
- `kind`: `fixture` | `item` (absent = `fixture`);
- `target`: `{name, repo, branch, toolchain}`;
- `item`: `NNN`;
- `snapshot`: `{base_revision, snapshot_commit, bytes}`;
- `companion`: `{repo, commit, files_sha256, file_count}`;
- `cache`: `{namespace, pvc, size, prefetch: {result, seconds}}`.

`tasks.<label>`: for item runs, the label is `NNN`. The fields are as today, plus
`baseline: {result, checks}`.

**State machine for item runs** (fixture unchanged):

```text
created → gate (readiness) ─refuse→ refused
        → awaiting-mode → (approved) → prefetch → baseline ─fail→ failed(BASELINE_*)
        → attempting(n) ─fail→ awaiting-retry → attempting(n+1) … ─exhausted→ failed
                        ─pass→ publishing → awaiting-merge
        → (merged) → awaiting-status (status PR or terminal approval) → verified → finishing → succeeded
```

`pending` steps and decisions are reused: `mode`, `retry`, `reconcile` (the terminal status
approval), and the status-change PR merge on GitHub.

## Toolchain (catalog entry)

`pilot/images/toolchains.json`:

```json
{"jdk21-maven": {"image_key": "TOOLCHAIN_JDK21_MAVEN_IMAGE",
                 "proxy": "maven-settings",
                 "worker": {"home": "1Gi", "tmp": "512Mi", "env": {"MAVEN_OPTS": "-Xmx768m"}},
                 "grader": {"cpu": "2", "memory": "2Gi", "deadline_s": 900, "command_timeout_s": 600},
                 "cache": {"size": "2Gi"}},
 "node24": {"image_key": "TOOLCHAIN_NODE24_IMAGE", "proxy": "env", "…": "…"},
 "python": {"image_key": null, "proxy": "env", "grader": "default", "cache": null}}
```

The image's own `/opt/toolchain/toolchain.json` supplies `PATH` and environment
(contracts/toolchain-image.md). An operator-supplied image gets the same default resources as a
catalog toolchain.

## Registry proxy

`pilot-egress-r-<target>`: a namespace with a tinyproxy Deployment whose allowlist is the
profile's registry hosts. It is created or re-rendered at run start and deleted by `down`.
Clients: worker pods of that target's runs, and that target's prefetch pods.

## Grading namespace (item runs)

`pilot-g-<label>` per run, holding:
- the PVC `deps`;
- a prefetch Job, with label `pilot/role=prefetch`, allowed egress to the registry proxy only;
- a baseline Job and one grader Job per attempt, with label `pilot/role=grader` and no egress;
- staged inputs, through each Job's init receiver.

It is deleted when the run ends.

## Evidence additions (item runs)

| File | Content |
|---|---|
| `gate/<seq>-<label>.json` | the readiness rows, in addition to the gate result |
| `prefetch.json` | result, duration, cache bytes; no dependency contents |
| `baseline/grade.json` | the baseline grade report |
| `companion.json` | repo, commit, file list with SHA-256s (no contents) |
| `checks/results.jsonl` | P06 registry rows and P11 overlay row |
| `attempts/<NNN>-<n>/meta.json` | `+ test_command, hidden_mode:"overlay", spec_sha256, toolchain` |
