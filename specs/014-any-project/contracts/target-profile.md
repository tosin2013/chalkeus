# Contract: target profile

## Files

```text
~/.config/specfoundry/
├── pilot.env                       # shared: model, approvers, publisher mode, defaults (unchanged)
└── targets/
    ├── <name>.env                  # one per target repository (mode 0600)
    └── <name>.publisher.env        # token mode only: PILOT_SCM_TOKEN for that repository (mode 0600)
```

- The directory `targets/` must be mode 0700.
- A profile overrides `pilot.env` only for the keys listed below. Every other key comes from
  `pilot.env`.

## Keys

```sh
PILOT_TARGET_REPO=owner/greeting-service
PILOT_TARGET_BRANCH=main
PILOT_TOOLCHAIN=jdk21-maven
# or your own toolchain image (contracts/toolchain-image.md):
# PILOT_TOOLCHAIN_IMAGE=quay.io/you/toolchain-rust@sha256:<64 hex>   PILOT_TOOLCHAIN=rust-1.83
PILOT_TEST_COMMAND=mvn -B test
# optional: installs dependencies (needs PILOT_REGISTRY_HOSTS); runs in the worker and the prefetch
# PILOT_INSTALL_COMMAND=pip install -r requirements.txt
PILOT_REGISTRY_HOSTS=repo.maven.apache.org
PILOT_COMPANION_REPO=owner/greeting-service-acceptance
PILOT_GITHUB_ISSUES=off
PILOT_GITHUB_APPROVALS=on
# optional, may only raise the toolchain defaults:
# PILOT_GRADER_MEMORY=3Gi  PILOT_GRADER_DEADLINE_S=1200  PILOT_CACHE_SIZE=4Gi  PILOT_MAX_INPUT_MB=80
```

Field rules are in [data-model.md](../data-model.md#target-profile).

## Validation errors (exit 2)

| Code | When |
|---|---|
| `TARGET_UNKNOWN` | `--target <name>`, but `targets/<name>.env` doesn't exist |
| `TARGET_MODE` | the profile or `targets/` is group/world-readable |
| `TARGET_INVALID` | a required key is missing, or a value breaks its rule (the message names the key) |
| `TARGET_IS_CANARY` | the repository is this checkout's origin, or another target's companion repository |
| `TOOLCHAIN_UNAVAILABLE` | the toolchain is not in the catalog, or its image has no lock digest; the message links the toolchain request form |

The repository guard (exit 3) applies to the fixture as today. For other targets it applies
`TARGET_IS_CANARY`.

## Mapping to cluster mode (011, FR-004b)

Each key becomes a field of the same meaning in a namespaced resource, for example
`Project.spec.repository`, `.toolchain`, `.testCommand`, `.registryHosts`,
`.companionRepository`. `<name>.publisher.env` becomes a Secret reference. Only people with
RBAC on that resource can change it, and worker and grader service accounts can't.
