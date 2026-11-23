# 027 evidence

## The rename day (SC-005, T021)

- **2026-11-23 05:26 UTC:** stage 3 (#168) merged. The owner renamed the repository from
  `tosin2013/specfoundry` to `tosin2013/chalkeus` the same day.
- **The old address redirects:** `git ls-remote https://github.com/tosin2013/specfoundry.git HEAD`
  returns the same commit as `origin/main` (checked 2026-11-23).

Workflows on `main` of the renamed repository:

| Workflow | Run | Trigger | Commit | Result |
|---|---|---|---|---|
| go | [45200562777](https://github.com/tosin2013/chalkeus/actions/runs/45200562777) | push (#176) | 3c90564 | success |
| pilot-images | [45121859470](https://github.com/tosin2013/chalkeus/actions/runs/45121859470) | push (#172, stage 4) | cf036dc | success: published the seven `chalkeus-*` images; digest PR #173 |
| integration | [45125968377](https://github.com/tosin2013/chalkeus/actions/runs/45125968377) | weekly schedule | 35f9710 | success: every scenario on kind, with `chalkeus-m1` and `chalkeus.io` labels |

Stage 4 (#172) also passed every check on its pull request, including integration, before the
merge.

## The new images are public (T019)

Each `chalkeus-*` digest in `pilot/images.lock` pulls with an anonymous GHCR token (HTTP 200 for
all seven, 2026-11-23), and `./chalkeus forge up` on the owner's machine created `chalkeus-m1`
from them; `./chalkeus forge doctor --target stockroom` reported the images match the lock.
