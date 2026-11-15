"""Generate the 016 golden audit expectations from the reference Python audit.

    PYTHONPATH=pilot python3 pilot/tools/golden_audit.py

Audits every committed pilot evidence record (`specs/*/evidence/<run-id>/run.json`, any state)
in the same mode as pilot/tests/test_committed_evidence.py (final iff teardown.jsonl exists),
and writes specs/016-go-pilot/golden/audit-expected.json (contracts/parity.md). Run by hand and
commit the output; tests only read it.
"""
import json
import subprocess
from pathlib import Path

from specfoundry_pilot import audit

ROOT = Path(__file__).resolve().parents[2]
OUT = ROOT / "specs/016-go-pilot/golden/audit-expected.json"


def main() -> int:
    sha = subprocess.run(["git", "rev-parse", "--short", "HEAD"], cwd=ROOT, capture_output=True, text=True).stdout.strip()
    records = []
    for rj in sorted(ROOT.glob("specs/*/evidence/*/run.json")):
        d = rj.parent
        final = (d / "teardown.jsonl").exists()
        rows = audit.audit_dir(d, final=final)
        records.append({"path": str(d.relative_to(ROOT)), "final": final,
                        "rows": [{k: r[k] for k in ("id", "result", "description", "detail")} for r in rows]})
    OUT.parent.mkdir(parents=True, exist_ok=True)
    OUT.write_text(json.dumps({"generated_by": f"python reference {sha}", "records": records}, indent=2) + "\n")
    print(f"{OUT.relative_to(ROOT)}: {len(records)} records, {sum(len(r['rows']) for r in records)} rows")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
