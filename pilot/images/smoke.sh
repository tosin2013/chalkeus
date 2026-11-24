#!/usr/bin/env bash
# Arbitrary-UID smoke test for the pilot images (003 US2, T023/T024).
# Runs each image as UID 1000123 / GID 0 with a read-only root filesystem and only the same
# writable mounts the pods get. Used by .github/workflows/pilot-images.yml before pushing, and
# locally. Needs: docker, git, python3.
#
#   WORKER=<ref> TOOLS=<ref> EGRESS=<ref> pilot/images/smoke.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
UID_ARB=1000123
W="$(mktemp -d)"; trap 'docker rm -f pilot-smoke-proxy >/dev/null 2>&1 || true; docker network rm pilot-smoke >/dev/null 2>&1 || true; rm -rf "$W"' EXIT
RUN=(docker run --rm --user "${UID_ARB}:0" --read-only --security-opt no-new-privileges --cap-drop ALL)
frame() { PYTHONPATH="$ROOT/pilot/images/common" python3 -c '
import sys, json
import frame
files, err = frame.decode(sys.stdin.read(), required=(sys.argv[1],))
if err: sys.exit(f"bad frame: {err}")
print(json.dumps(json.loads(files[sys.argv[1]]))[:400])' "$1"; }
fail=0

# --- fixture repo + bundles --------------------------------------------------------------
mkdir -p "$W/repo" "$W/in" "$W/grade" "$W/grading" "$W/proxy"
cp -r "$ROOT/pilot/fixture/." "$W/repo/"
git -C "$W/repo" init -q -b main
git -C "$W/repo" add -A
git -C "$W/repo" -c user.name=smoke -c user.email=smoke@invalid commit -qm base
BASE="$(git -C "$W/repo" rev-parse HEAD)"
git -C "$W/repo" bundle create -q "$W/in/base.bundle" HEAD
cp "$W/in/base.bundle" "$W/grade/base.bundle"
cp "$ROOT/pilot/images/tests/reference/slugkit/slug.py" "$W/repo/src/slugkit/slug.py"
echo 'from .slug import slugify  # noqa: F401' >> "$W/repo/src/slugkit/__init__.py"
# Portable in-place edit (GNU and BSD sed differ on -i; 031 runs this on macOS too).
sed 's/- \[ \]/- [X]/' "$W/repo/specs/001-slugify/tasks.md" > "$W/tasks.md" && mv "$W/tasks.md" "$W/repo/specs/001-slugify/tasks.md"
git -C "$W/repo" add -A
git -C "$W/repo" -c user.name=smoke -c user.email=smoke@invalid commit -qm solution
git -C "$W/repo" bundle create -q "$W/grade/change.bundle" "$BASE..HEAD"
cp "$ROOT/pilot/grading/"*.py "$W/grading/"
META='{"run_id":"smoke","task":"A","attempt":1,"item_id":"001","base_revision":"'"$BASE"'","allowed_paths":["src/slugkit/*","tests/*","specs/001-slugify/tasks.md"],"induction":null}'
echo "$META" > "$W/in/meta.json"; echo "$META" > "$W/grade/meta.json"
echo "Reply with ok." > "$W/in/prompt.md"
chmod -R a+rX "$W"

# --- worker: starts, runs its pipeline, emits a frame (no model config -> harness error) --
echo "== worker ($WORKER) as uid $UID_ARB"
if out="$("${RUN[@]}" --tmpfs /tmp:exec,mode=1777 --tmpfs /work:mode=1777 --tmpfs /home/worker:mode=1777 \
      -v "$W/in:/in:ro" -e PILOT_MODEL=none/none "$WORKER" | frame report.json)"; then
  echo "  PASS frame decoded: $out"
else echo "  FAIL worker produced no valid frame"; fail=1; fi

# --- tools: grades the reference solution, no network -------------------------------------
echo "== tools ($TOOLS) as uid $UID_ARB"
if out="$("${RUN[@]}" --network none --tmpfs /tmp:exec,mode=1777 -v "$W/grade:/in:ro" -v "$W/grading:/grading:ro" \
      "$TOOLS" python3 /usr/local/lib/pilot/grade.py | frame grade.json)" && echo "$out" | grep -q '"result": "pass"'; then
  echo "  PASS grade: pass"
else echo "  FAIL grader: $out"; fail=1; fi

# --- egress: tinyproxy starts and enforces the allowlist ----------------------------------
echo "== egress ($EGRESS) as uid $UID_ARB"
sed -n '1,200p' "$ROOT/pilot/images/egress/tinyproxy.conf.tmpl" > "$W/proxy/tinyproxy.conf"
printf '^allowed\\.invalid$\n' > "$W/proxy/allowlist"
chmod -R a+rX "$W/proxy"
docker network create pilot-smoke >/dev/null
docker run -d --name pilot-smoke-proxy --network pilot-smoke --user "${UID_ARB}:0" --read-only --cap-drop ALL \
  --tmpfs /tmp:mode=1777 -v "$W/proxy:/etc/pilot-proxy:ro" "$EGRESS" >/dev/null
sleep 2
code="$(docker run --rm --network pilot-smoke --user "${UID_ARB}:0" --read-only --tmpfs /tmp:mode=1777 "$TOOLS" \
  curl -s -o /dev/null -w '%{http_code}' --max-time 5 -x http://pilot-smoke-proxy:8888 https://blocked.invalid/ || true)"
if docker logs pilot-smoke-proxy 2>&1 | grep -q 'refused on filtered domain "blocked.invalid"'; then
  echo "  PASS proxy running and refused a non-allowlisted host (curl http=$code)"
else echo "  FAIL proxy did not refuse as expected"; docker logs pilot-smoke-proxy 2>&1 | tail -5; fail=1; fi

# --- toolchain images (014, contracts/toolchain-image.md): the init copy, then the version
# command, both as an arbitrary UID with a read-only root, as in the pods. TOOLCHAINS lists refs.
for tc in ${TOOLCHAINS:-}; do
  vol="pilot-smoke-tc-$RANDOM"; docker volume create "$vol" >/dev/null
  docker run --rm --user 0:0 -v "$vol":/dest "$TOOLS" chmod 0777 /dest >/dev/null 2>&1 || true
  if "${RUN[@]}" -v "$vol":/dest "$tc" >/dev/null 2>&1 \
     && v="$("${RUN[@]}" -v "$vol":/opt/toolchain:ro --tmpfs /tmp:exec,mode=1777 "$TOOLS" python3 -c '
import json, os, subprocess, sys
t = json.load(open("/opt/toolchain/toolchain.json"))
env = dict(os.environ, PILOT_DEPS="/tmp/deps")
path = [p.replace("${PILOT_DEPS}", "/tmp/deps") for p in t["path"]]
env["PATH"] = ":".join(path + [env["PATH"]])
for root, _, files in os.walk("/opt/toolchain"):
    for f in files:
        p = os.path.join(root, f)
        if os.path.isfile(p) and not os.path.islink(p) and os.stat(p).st_mode & 0o6000:
            sys.exit("setuid file: " + p)
print(subprocess.run(t["version_command"], env=env, capture_output=True, text=True, check=True).stdout.strip().splitlines()[0])' 2>&1)"; then
    echo "  PASS toolchain $tc: $v"
  else echo "  FAIL toolchain $tc: ${v:-copy failed}"; fail=1; fi
  # The offline sample (T040): install with the network into a deps volume (the prefetch), then the
  # tests with no network and the deps read-only (the grader), as an arbitrary UID.
  name="$(docker run --rm -v "$vol":/opt/toolchain:ro "$TOOLS" python3 -c 'import json; print(json.load(open("/opt/toolchain/toolchain.json"))["name"])' 2>/dev/null || true)"
  sample="$ROOT/pilot/images/toolchain-samples/$name"
  if [ -n "$name" ] && [ -f "$sample/sample.json" ]; then
    deps="pilot-smoke-deps-$RANDOM"; docker volume create "$deps" >/dev/null
    docker run --rm --user 0:0 -v "$deps":/deps "$TOOLS" chmod 0777 /deps >/dev/null 2>&1 || true
    step() { # step <online|offline> <command>
      local net=() mode=rw; [ "$1" = offline ] && { net=(--network none); mode=ro; }
      # ${net[@]+...}: an empty array under set -u fails in bash < 4.4 (macOS's 3.2; 031).
      "${RUN[@]}" ${net[@]+"${net[@]}"} -v "$vol":/opt/toolchain:ro -v "$deps":/deps:"$mode" -v "$sample":/src:ro \
        --tmpfs /tmp:exec,mode=1777 --tmpfs /work:exec,mode=1777 -e HOME=/tmp -e PILOT_DEPS=/deps -e ROLE="$1" \
        "$TOOLS" bash -c '
set -e
eval "$(python3 -c "
import json, os, shlex
t = json.load(open(\"/opt/toolchain/toolchain.json\"))
print(\"export PATH=\" + shlex.quote(\":\".join(p.replace(\"\${PILOT_DEPS}\", \"/deps\") for p in t[\"path\"])) + \":\$PATH\")
for k, v in t.get(\"env\", {}).items():
    print(\"export \" + k + \"=\" + shlex.quote(v.replace(\"\${PILOT_DEPS}\", \"/deps\")))
if t.get(\"proxy\") == \"maven-settings\":  # as internal/toolchain RoleEnv, minus the settings file
    a = \"-B -o -Dmaven.repo.local=/tmp/m2 -Dmaven.repo.local.tail=/deps/m2\" if os.environ[\"ROLE\"] == \"offline\" else \"-B -Dmaven.repo.local=/deps/m2\"
    print(\"export MAVEN_ARGS=\" + shlex.quote(a))
if os.environ[\"ROLE\"] == \"online\" and t.get(\"setup_command\"):
    print(\"(cd /work && \" + \" \".join(shlex.quote(x) for x in t[\"setup_command\"]) + \")\")
")"
cp -r /src/. /work/ && cd /work
'"$2"
    }
    install="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["install"])' "$sample/sample.json")"
    tests="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["test"])' "$sample/sample.json")"
    phase=install
    if out="$(step online "$install" 2>&1)" && phase=tests && out="$(step offline "$tests" 2>&1)"; then
      echo "  PASS toolchain $name offline sample: install online, tests offline with read-only deps"
    else # name the step, and show the first errors (Maven's last lines are only its help footer)
      echo "  FAIL toolchain $name offline sample, $phase step: $( { printf '%s\n' "$out" | grep -m8 -E 'ERROR|Error|error' || true; printf '%s\n' "$out" | tail -5; } )"; fail=1; fi
    docker volume rm -f "$deps" >/dev/null
  fi
  docker volume rm -f "$vol" >/dev/null
done

[ "$fail" -eq 0 ] && echo "== smoke: all images pass as an arbitrary UID" || { echo "== smoke: FAILED"; exit 1; }
