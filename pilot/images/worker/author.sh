#!/usr/bin/env bash
# The worker image's `author` role (024 R3, contracts/author-watch.md): the test author drafts
# hidden tests for one item. In: /in/meta.json, /in/prompt.md, /in/workspace.tar (spec.md,
# plan.md, tasks.md, criteria.json, base/, an empty draft/). Out, on fd 3: one frame holding
# report.json and draft.tar (only the draft/ tree). No repository credential is present.
# Paths can be overridden for tests (PILOT_IN, PILOT_WORK, PILOT_LIB, PILOT_TMP).
set -uo pipefail
IN="${PILOT_IN:-/in}"; WORK="${PILOT_WORK:-/work}"; LIB="${PILOT_LIB:-/usr/local/lib/pilot}"; TMP="${PILOT_TMP:-/tmp}"
OUT="$TMP/out"
mkdir -p "$OUT"
ERROR=""; HEXIT=""

finish() {
  cd "$TMP" || true
  python3 - "$ERROR" "$HEXIT" "$IN" "$TMP" <<'PY' > "$OUT/report.json"
import json, os, re, subprocess, sys
err, hexit, inp, tmp = sys.argv[1] or None, sys.argv[2], sys.argv[3], sys.argv[4]
meta = json.load(open(os.path.join(inp, "meta.json")))
spend_p = os.path.join(tmp, "spend.json")
spend = json.load(open(spend_p)) if os.path.exists(spend_p) else None
tail = ""
for f in ("entry.log", "harness.stderr", "harness.jsonl"):
    p = os.path.join(tmp, f)
    if os.path.exists(p):
        tail += open(p, errors="replace").read()[-2048:]
tail = re.sub(r"(sk-|ghp_|gho_|ghs_|github_pat_)[A-Za-z0-9_\-]{4,}", r"\1***", tail)
tail = re.sub(r"(?i)(bearer\s+)[A-Za-z0-9._\-]{8,}", r"\1***", tail)
try:
    harness_version = subprocess.run(["opencode", "--version"], capture_output=True, text=True, timeout=20).stdout.strip().lstrip("v") or None
except Exception:  # noqa: BLE001 - the version is informational
    harness_version = None
print(json.dumps({
    "version": 1, "role": "author", "draft_id": meta.get("draft_id"), "item": meta.get("item"),
    "attempt": meta.get("attempt"),
    "harness": {"name": "opencode", "version": harness_version, "exit": int(hexit) if hexit not in ("", None) else None},
    "spend": spend, "error": err, "log_tail": tail[-4096:],
}))
PY
  args=("report.json=$OUT/report.json")
  [ -s "$OUT/draft.tar" ] && args+=("draft.tar=$OUT/draft.tar")
  python3 "$LIB/frame.py" emit "${args[@]}" >&3
  exit 0
}

TASK="$WORK/task"
mkdir -p "$TASK"
tar -xf "$IN/workspace.tar" -C "$TASK" || { ERROR="workspace_failed"; finish; }
mkdir -p "$TASK/draft"
# The project and the planning files are read-only for the author; only draft/ is its output.
chmod -R a-w "$TASK/base" "$TASK"/*.md "$TASK"/criteria.json 2>/dev/null || true
cd "$TASK"

mkfifo "$TMP/events"
opencode run --standalone --format json --auto --model "$PILOT_MODEL" "$(cat "$IN/prompt.md")" \
  > "$TMP/events" 2> "$TMP/harness.stderr" &
HPID=$!
(cd "$TMP" && python3 "$LIB/watchdog.py" "$HPID") < "$TMP/events"
wait "$HPID"; HEXIT=$?
if [ -f "$TMP/spend.json" ] && python3 -c 'import json,sys; sys.exit(0 if json.load(open(sys.argv[1]))["stopped_by_cap"] else 1)' "$TMP/spend.json"; then
  ERROR="spend_cap"
elif [ "$HEXIT" -ne 0 ]; then
  ERROR="harness_error"
fi

# Only regular files under draft/ leave the Job: no links, nothing outside it.
find "$TASK/draft" ! -type f ! -type d -delete 2>/dev/null || true
if [ -n "$(find "$TASK/draft" -type f -print -quit)" ]; then
  tar -cf "$OUT/draft.tar" -C "$TASK" draft || ERROR="${ERROR:-draft_tar_failed}"
fi
finish
