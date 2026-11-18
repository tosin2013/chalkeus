# Toolchain offline samples (014 T040)

`smoke.sh` uses one tiny project per catalog toolchain, named like its `toolchain.json`'s `name`.
Each `sample.json` gives two commands, run as an arbitrary UID with the toolchain mounted
read-only, the way the pilot runs them:

1. `install`, with the network, into a dependency directory (the prefetch);
2. `test`, with **no network** and the dependency directory **read-only** (the grader).

The toolchain's `setup_command` runs before `install`. For `maven-settings` toolchains the smoke
test sets `MAVEN_ARGS` as the pilot does per role (`internal/toolchain` `RoleEnv`), without the
proxy settings file (the CI runner has direct network).
