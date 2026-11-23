# Contract: the name map

Every name 027 changes. "Old still read" means the old name keeps working, with a one-time notice,
until a later change removes it.

| What | Old | New | Old still read? | Stage |
|---|---|---|---|---|
| Go module | `github.com/tosin2013/specfoundry` | `github.com/tosin2013/chalkeus` | no (GitHub redirects the repository) | 3 |
| Repository | tosin2013/specfoundry | tosin2013/chalkeus (owner, same day) | GitHub redirect | 3 |
| In-repo command | `pilot/pilot <cmd>` | `./chalkeus forge <cmd>` | **no** (hard switch) | 3 |
| Binary built | `pilot/.build/bin/pilot` | `pilot/.build/bin/chalkeus` | no | 3 |
| Skill | `.claude/skills/pilot` | `.claude/skills/forge` | no | 3 |
| Config folder | `~/.config/specfoundry` | `~/.config/chalkeus` | **yes**, if the new one doesn't exist | 2 |
| Folder setting | `SPECFOUNDRY_CONFIG` | `CHALKEUS_CONFIG` | **yes** | 2 |
| Main settings file | `pilot.env` | `forge.env` | **yes** (new wins) | 2 |
| Settings and CLI variables | `PILOT_*` (for example `PILOT_MODEL`, `PILOT_ROOT`) | `FORGE_*` | **yes** (new wins) | 2 |
| Variables inside the pilot's pods | `PILOT_*` set by the pilot | unchanged (internal, R3) | — | — |
| Proposal label default | `pilot/go` | `chalkeus:proposed` | no (a profile can set any label) | 3 |
| Kubernetes labels | `specfoundry.io/*` | `chalkeus.io/*` | **yes, in the audit** (evidence) | 4 |
| kind cluster | `specfoundry-m1` | `chalkeus-m1` | detected, removal command shown | 4 |
| Local registry | `specfoundry-registry` | `chalkeus-registry` | detected | 4 |
| Images | `ghcr.io/tosin2013/specfoundry-pilot-*` | `ghcr.io/tosin2013/chalkeus-*` | old images stay published | 4 |
| Product name in docs | specfoundry, the pilot | Chalkeus, forge | history only | 3 |

Unchanged: the `pilot/` folder layout, the `targets/` folder and `<name>.publisher.env`, evidence
file names, and past items' specs and evidence.
