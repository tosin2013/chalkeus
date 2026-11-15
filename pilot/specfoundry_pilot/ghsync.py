"""004 GitHub loop: mirror the target roadmap as item issues, status labels and task checklists,
and report drift (research R1, R2, R8; contracts/github-objects.md).

The roadmap is the source of truth (constitution I): issues and labels are written *from* it and
never read back as status. Writes run through `run_ghops`, a child process that holds the
publisher credential in its own environment only. Reads use the operator's `gh`.
"""
from __future__ import annotations

import json
import re
import shutil
import subprocess
from pathlib import Path

from . import evidence, fixture, paths

GHOPS = paths.IMAGES / "tools" / "ghops.py"
STATUSES = ("undecided", "needs-info", "planned", "specced", "in-progress", "implemented",
            "verified", "deferred", "abandoned")
LABEL_PREFIX = "status:"
LABEL_COLORS = {"verified": "0e8a16", "implemented": "1d76db", "in-progress": "fbca04", "planned": "c5def5"}
ITEM_MARKER = re.compile(r"<!-- specfoundry:item=([^ ]+) -->")


class GitHubLoopError(Exception):
    """Any GitHub problem in the loop. Callers fail closed (constitution II)."""


# ---------------------------------------------------------------- I/O seams (patched in tests)

def credential_env(cfg) -> Path:
    """The env file holding the publisher credential for child processes (token or App token)."""
    if cfg.credential == "app":
        from . import ghapp
        return ghapp.current_env(cfg)
    return cfg.publisher_env


def run_ghops(cfg, op: str, payload: dict) -> dict:
    """Run one GitHub write in a child process with the credential sourced there only."""
    import tempfile
    env_file = credential_env(cfg)
    with tempfile.NamedTemporaryFile("w", suffix=".json", delete=False) as f:
        json.dump(payload, f)
        pfile = f.name
    try:
        r = subprocess.run(["bash", "-c", 'set -a; . "$1"; set +a; exec python3 "$2" "$3" "$4"',
                            "_", str(env_file), str(GHOPS), op, pfile],
                           capture_output=True, text=True, timeout=120)
    finally:
        Path(pfile).unlink(missing_ok=True)
    try:
        res = json.loads(r.stdout.strip().splitlines()[-1])
    except Exception:  # noqa: BLE001
        raise GitHubLoopError(f"ghops {op} did not run: {(r.stderr or '').strip()[-300:]}")
    if not res.get("ok"):
        raise GitHubLoopError(f"ghops {op}: {res.get('error')}")
    return res["result"]


def read_issues(cfg) -> list[dict]:
    try:
        out = fixture.gh("issue", "list", "-R", cfg.repo, "--state", "all", "--limit", "500",
                         "--json", "number,title,body,labels,state").stdout
    except subprocess.CalledProcessError as e:
        raise GitHubLoopError(f"cannot list issues on {cfg.repo}: {(e.stderr or '').strip()[-200:]}")
    return [{"number": i["number"], "title": i["title"], "body": i.get("body") or "",
             "labels": [lab["name"] for lab in i.get("labels", [])], "state": i["state"].lower()}
            for i in json.loads(out)]


def read_labels(cfg) -> set[str]:
    try:
        out = fixture.gh("label", "list", "-R", cfg.repo, "--limit", "500", "--json", "name").stdout
    except subprocess.CalledProcessError as e:
        raise GitHubLoopError(f"cannot list labels on {cfg.repo}: {(e.stderr or '').strip()[-200:]}")
    return {lab["name"] for lab in json.loads(out)}


# ---------------------------------------------------------------- pure functions

TASK_LINE = re.compile(r"^- \[( |x|X)\] (.+)$")


def checklist(tasks_text: str | None) -> list[str]:
    if not tasks_text:
        return []
    out = []
    for line in tasks_text.splitlines():
        m = TASK_LINE.match(line.strip())
        if m:
            out.append(f"- [{'x' if m.group(1) in 'xX' else ' '}] {m.group(2)}")
    return out


def issue_body(item: dict, tasks_text: str | None) -> str:
    spec = item.get("spec_dir")
    lines = [f"<!-- specfoundry:item={item['id']} -->",
             f"**{item['id']} — {item['title']}** · status `{item['status']}`" + (f" · spec `{spec}`" if spec else ""),
             "", item.get("description") or "", "", f"**Outcome:** {item.get('outcome') or '-'}"]
    tl = checklist(tasks_text)
    if tl:
        src = (item.get("tasks") or {}).get("source") or "tasks.md"
        lines += ["", f"### Tasks (from {src})", *tl]
    lines += ["", "_Generated from the roadmap by specfoundry. Edits here are overwritten by `pilot github sync`._"]
    return "\n".join(lines) + "\n"


def desired_state(status_json: dict, tasks_texts: dict[str, str | None]) -> dict[str, dict]:
    """item id -> {title, body, label} for every active item (superseded items are history)."""
    out = {}
    for it in status_json.get("items", []):
        if not it.get("active", True):
            continue
        status = it["status"] if it.get("status") in STATUSES else "undecided"
        out[it["id"]] = {"title": f"{it['id']} — {it['title']}", "body": issue_body(it, tasks_texts.get(it["id"])),
                         "label": LABEL_PREFIX + status, "status": status}
    return out


def by_item(issues: list[dict]) -> tuple[dict[str, dict], list[dict]]:
    """Index issues by their item marker. Returns (item -> issue, duplicates)."""
    idx, dups = {}, []
    for i in sorted(issues, key=lambda x: x["number"]):
        m = ITEM_MARKER.search(i.get("body") or "")
        if not m:
            continue
        if m.group(1) in idx:
            dups.append(i)
        else:
            idx[m.group(1)] = i
    return idx, dups


def plan_ops(repo: str, desired: dict, issues: list[dict], labels: set[str]) -> list[tuple[str, dict]]:
    ops = []
    want_labels = [{"name": LABEL_PREFIX + s, "color": LABEL_COLORS.get(s, "ededed")} for s in STATUSES]
    missing = [lab for lab in want_labels if lab["name"] not in labels]
    if missing:
        ops.append(("ensure-labels", {"repo": repo, "labels": missing}))
    idx, _ = by_item(issues)
    for item, d in desired.items():
        cur = idx.get(item)
        if cur is None:
            ops.append(("create-issue", {"repo": repo, "item": item, "title": d["title"], "body": d["body"],
                                         "labels": [d["label"]]}))
            continue
        other = [lab for lab in cur["labels"] if not lab.startswith(LABEL_PREFIX)]
        new_labels = sorted(other + [d["label"]])
        edit = {}
        if cur["body"].strip() != d["body"].strip():
            edit["body"] = d["body"]
        if sorted(cur["labels"]) != new_labels:
            edit["labels"] = new_labels
        if cur["title"] != d["title"]:
            edit["title"] = d["title"]
        if edit:
            ops.append(("edit-issue", {"repo": repo, "number": cur["number"], "item": item, **edit}))
    return ops


def drift_report(desired: dict, issues: list[dict], prs: list[dict] | None = None,
                 issue_numbers: dict | None = None) -> list[dict]:
    """Differences between the roadmap and GitHub (research R8). Empty means no drift."""
    out = []
    idx, dups = by_item(issues)
    for item, d in desired.items():
        cur = idx.get(item)
        if cur is None:
            out.append({"kind": "missing_issue", "item": item})
            continue
        status_labels = sorted(lab for lab in cur["labels"] if lab.startswith(LABEL_PREFIX))
        if status_labels != [d["label"]]:
            out.append({"kind": "label_mismatch", "item": item, "issue": cur["number"],
                        "want": d["label"], "have": status_labels})
        if d["status"] == "verified" and cur["state"] == "open":
            out.append({"kind": "open_but_verified", "item": item, "issue": cur["number"]})
        if d["status"] != "verified" and cur["state"] == "closed":
            out.append({"kind": "closed_but_not_verified", "item": item, "issue": cur["number"]})
    for item, cur in idx.items():
        if item not in desired:
            out.append({"kind": "orphan_issue", "item": item, "issue": cur["number"]})
    for i in dups:
        out.append({"kind": "duplicate_issue", "issue": i["number"]})
    for p in prs or []:
        if p.get("kind", "task") != "task":
            continue
        want = (issue_numbers or {}).get(paths.TASKS.get(p.get("task"), ""))
        if want and not p.get("closes") == want:
            out.append({"kind": "pr_without_closes", "pr": p.get("number"), "want": want})
    return out


# ---------------------------------------------------------------- orchestration

def _snapshot(cfg) -> tuple[dict, dict[str, str | None]]:
    clone, _ = fixture.fresh_clone(cfg.repo, cfg.branch, depth=1)
    try:
        r = subprocess.run(["spec", "status", "--json", "--root", str(clone)], capture_output=True, text=True)
        if r.returncode not in (0, 1):
            raise GitHubLoopError(f"spec status failed on {cfg.repo}: {r.stderr.strip()[-200:]}")
        doc = json.loads(r.stdout)
        texts = {}
        for it in doc.get("items", []):
            src = (it.get("tasks") or {}).get("source")
            f = clone / src if src else None
            texts[it["id"]] = f.read_text() if f and f.is_file() else None
        return doc, texts
    finally:
        shutil.rmtree(clone, ignore_errors=True)


def sync(cfg, run=None) -> dict:
    """Create missing item issues and labels; update labels, titles and bodies. Idempotent;
    never deletes, closes or reopens. Returns {created, changed, issues: {item: number}}."""
    doc, texts = _snapshot(cfg)
    desired = desired_state(doc, texts)
    issues = read_issues(cfg)
    ops = plan_ops(cfg.repo, desired, issues, read_labels(cfg))
    numbers = {k: v["number"] for k, v in by_item(issues)[0].items()}
    created = changed = 0
    for op, payload in ops:
        item = payload.pop("item", None)
        res = run_ghops(cfg, op, payload)
        if op == "create-issue":
            created += 1
            numbers[item] = res["number"]
        elif op == "edit-issue":
            changed += 1
    summary = {"created": created, "changed": changed, "issues": numbers}
    if run is not None:
        d = run.dir / "github"
        d.mkdir(exist_ok=True)
        evidence.write_json(d / "issues.json", {"repo": cfg.repo, "issues": numbers, "at": evidence.now()})
        run.event("github", None, None, "github_sync", f"created={created} changed={changed}")
    return summary


def drift(cfg, run=None) -> list[dict]:
    doc, texts = _snapshot(cfg)
    desired = desired_state(doc, texts)
    issues = read_issues(cfg)
    prs, numbers = None, None
    if run is not None:
        prs = json.loads((run.dir / "prs.json").read_text()) if (run.dir / "prs.json").exists() else []
        f = run.dir / "github" / "issues.json"
        numbers = json.loads(f.read_text())["issues"] if f.exists() else None
    rep = drift_report(desired, issues, prs, numbers)
    if run is not None:
        (run.dir / "github").mkdir(exist_ok=True)
        evidence.write_json(run.dir / "github" / "drift.json", {"drift": rep, "at": evidence.now()})
    return rep


def issue_for(run, task: str) -> int | None:
    f = run.dir / "github" / "issues.json"
    if not f.exists():
        return None
    return json.loads(f.read_text())["issues"].get(paths.TASKS[task])
