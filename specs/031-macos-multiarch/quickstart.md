# Quickstart: validating 031

Each scenario names the spec items it proves. Evidence goes in
`specs/031-macos-multiarch/evidence/<run>/`.

## 1. Pins and build args (FR-002, SC-004)

```sh
go test ./internal/images/ -run BuildArgs     # Go twin matches build_args.py for every image
python3 -m pytest pilot/images/tests/ -k build_args
```

Expected: both pass. Delete one `_ARM64` key locally; both fail naming the key.

## 2. Local arm64 build on the Mac (US1, before CI)

```sh
for i in worker tools toolchain-python3.13; do
  docker buildx build --platform linux/arm64 --load -t chk-$i:arm64 \
    $(python3 pilot/images/build_args.py $i | sed 's/^/--build-arg /') pilot/images/$i
done
```

Expected: each builds; `docker run --rm chk-worker:arm64 opencode --version` prints 2.2.7. Change
one arm64 checksum in `pins.env`: that build fails at the checksum step.

## 3. Two-platform workflow (FR-001, FR-003, FR-004, SC-003)

Open the PR: both `build` legs pass and `merge` is skipped. After merge to `main`, the digest PR
lists index digests, and `image-digests.json` in its body shows both platforms per image. On
any machine:

```sh
for k in $(grep -o '^[A-Z_0-9]*_IMAGE=' pilot/images.lock | tr -d =); do
  docker buildx imagetools inspect "$(grep "^$k=" pilot/images.lock | cut -d= -f2)" | grep Platform
done
```

Expected: every image lists `linux/amd64` and `linux/arm64`.

## 4. Linux regression (FR-005, SC-006)

On the Linux amd64 machine, after the digest PR merges:
`./chalkeus forge up`, then the fixture walkthrough, then `down`, then audit.
Expected: same results as run 20261123T042536Z (026), with the audit at 11/11.
`versions.txt` differs only in the new lines and the spec-status 0.4.1 line.

## 5. doctor on the Mac (US4, FR-010, FR-011, FR-013)

```sh
./chalkeus forge doctor            # Docker Desktop running
# quit Docker Desktop
./chalkeus forge doctor            # expect: fail "Docker isn't running: start Docker Desktop"
./chalkeus forge up                # expect: refuses with the same message, installs nothing
```

## 6. `up`, isolation suite and fixture run on the Mac (US2, US3, SC-001)

Docker Desktop: 8 GB memory, 4+ CPUs (operator guide macOS path).

```sh
./chalkeus forge up
bash test/isolation/run.sh         # every check passes; evidence written
# fixture walkthrough (operator guide section 6), owner approves and merges
./chalkeus forge down && ./chalkeus forge audit
```

Expected: all isolation checks pass with no edits to the suite; audit 11/11; `versions.txt`
shows `host_os: darwin …`, `host_arch: arm64`, `node_platform: linux/arm64`.

## 7. Target run on the Mac (SC-002)

One prepared item of a target with a catalog toolchain (for example the Python 3.13 or Node 24
fixture target): `./chalkeus forge run --target <t> --item <NNN>`. Expected: grades and
publishes as on Linux; audit 11/11 after `down`.

## 8. Docs from a fresh Mac (US5, SC-005)

A Mac without forge's prerequisites follows only the operator guide's macOS path to scenario 6's
result.
