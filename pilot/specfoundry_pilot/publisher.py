"""Publish a passing attempt as one draft PR through the publisher Job (FR-012-FR-016, R8)."""
from __future__ import annotations

import json
import tempfile
from pathlib import Path

from . import evidence, fixture, frame, k8s, paths

TIMEOUT = 300


class PublishError(Exception):
    pass


def prs(run) -> list:
    p = run.dir / "prs.json"
    return json.loads(p.read_text()) if p.exists() else []


def _body(run, task: str, n: int, closes: int | None = None) -> str:
    st = run.task(task)
    rows = "\n".join(f"| {a['n']} | {a.get('induction') or '-'} | {a.get('result', '-')} | "
                     f"{(a.get('spend') or {}).get('cost_usd', '-')} | {(a.get('spend') or {}).get('tokens', '-')} |"
                     for a in st.get("attempts", []))
    gates = ", ".join(str(g) for g in st.get("gate_seqs", []))
    imgs = json.loads(paths.IMAGES_JSON.read_text())
    return (f"Opened by the specfoundry M1 pilot **publisher** (run `{run.id}`, task {task}, "
            f"fixture item {paths.TASKS[task]}).\n\n"
            f"Passing attempt: **{n}**. Gate decisions: {gates or '-'}.\n\n"
            f"| Attempt | Induction | Grade | Cost (USD) | Tokens |\n|---|---|---|---|---|\n{rows}\n\n"
            f"Images: worker `{imgs['worker']}`, grader/publisher `{imgs['tools']}`.\n\n"
            f"Evidence: `specs/002-m1-gated-pilot/evidence/{run.id}/` in the specfoundry repository.\n\n"
            f"This is a **draft**. A human reviews and merges it; the pilot never merges.\n"
            + (f"\nCloses #{closes}\n" if closes else ""))


def publish(run, cfg, imgs: dict, runner, task: str, n: int) -> dict:
    adir = run.attempt_dir(task, n)
    g = json.loads((adir / "grade.json").read_text())
    if g.get("result") != "pass":
        raise PublishError(f"attempt {task}/{n} did not pass grading (FR-012)")
    existing = [p for p in prs(run) if p["task"] == task and p.get("kind", "task") == "task"]
    if existing:
        return existing[0]
    closes = None
    if getattr(cfg, "github_issues", False):  # 004 FR-003: the PR names the item issue it closes
        from . import ghsync
        closes = ghsync.issue_for(run, task)
        if closes is None:
            closes = ghsync.sync(cfg, run)["issues"].get(paths.TASKS[task])
    head = f"pilot/{run.id}/{task.lower()}"
    spec = {"repo": cfg.repo, "head": head, "base": cfg.branch, "base_revision": g_base(adir),
            "title": f"pilot {run.id} task {task}", "body": _body(run, task, n, closes), "run_id": run.id, "task": task,
            **_login_hint(cfg)}
    run.event("publish", task, n, "publish_started", head)
    res = _submit(run, imgs, runner, task.lower(), spec, adir / "change.bundle")
    view = json.loads(fixture.gh("pr", "view", str(res["number"]), "-R", cfg.repo, "--json",
                                 "isDraft,author,headRefName,url").stdout)
    operator = fixture.gh("api", "user", "--jq", ".login").stdout.strip()
    entry = {"task": task, "number": res["number"], "url": view["url"], "head": view["headRefName"],
             "draft": bool(view["isDraft"]), "author": view["author"]["login"],
             "publisher_identity": res.get("token_login"), "operator_login": operator,
             "attempt": n, "created": bool(res.get("created")), "kind": "task", "closes": closes}
    if not entry["draft"] or entry["head"] != head:
        raise PublishError(f"PR #{entry['number']} is not the expected draft for {head}")
    allp = prs(run) + [entry]
    evidence.write_json(run.dir / "prs.json", allp)
    run.event("publish", task, n, "pr_opened" if entry["created"] else "pr_found", entry["url"])
    return entry


def g_base(adir: Path) -> str:
    return json.loads((adir / "meta.json").read_text())["base_revision"]


def _login_hint(cfg) -> dict:
    """App installation tokens cannot call GET /user; the host passes the bot login (004 US4)."""
    if getattr(cfg, "credential", "token") == "app":
        from . import ghapp
        return {"publisher_login": ghapp.bot_login(cfg)}
    return {}


def _submit(run, imgs: dict, runner, key: str, spec: dict, bundle: Path) -> dict:
    """Run the publisher Job for one PR spec + bundle; return its result.json (raises PublishError)."""
    cm = f"pub-{key}"
    jobname = f"publish-{key}-{run.id.lower().replace('t', '').replace('z', '')[-8:]}"
    k8s.kubectl("-n", "pilot-publish", "delete", "configmap", cm, "--ignore-not-found", check=False)
    k8s.kubectl("-n", "pilot-publish", "delete", "job", jobname, "--ignore-not-found", "--wait=true", check=False)
    with tempfile.TemporaryDirectory() as t:
        Path(t, "pr.json").write_text(json.dumps(spec))
        k8s.create_configmap("pilot-publish", cm, {"pr.json": Path(t, "pr.json"), "change.bundle": bundle})
    job = k8s.render(paths.MANIFESTS / "publisher-job.yaml.tmpl", JOB=jobname, CONFIGMAP=cm, TOOLS_IMAGE=imgs["tools"])
    handle = runner.submit("publisher", "pilot-publish", jobname, job, own_namespace=False)
    try:
        outcome = runner.wait(handle, TIMEOUT + 60)
        files, err = frame.decode(runner.collect(handle), required=("result.json",))
        status = runner.status(handle)
    finally:
        runner.delete(handle)
        k8s.kubectl("-n", "pilot-publish", "delete", "configmap", cm, "--ignore-not-found", check=False)
    if err or outcome != "Complete":
        detail = (json.loads(files["result.json"]).get("error") if files else None) or err or outcome
        raise PublishError(f"publisher {outcome}: {detail} ({status.get('pod_reason')})")
    res = json.loads(files["result.json"])
    if res.get("error"):
        raise PublishError(res["error"])
    return res


def status_body(run, task: str, closes: int | None) -> str:
    item = paths.TASKS[task]
    return (f"Opened by the specfoundry pilot **publisher** (run `{run.id}`) after task {task}'s PR was merged "
            f"and checked (merged, tasks 100%, merged tree matches the graded attempt).\n\n"
            f"It changes exactly one line: item {item} `planned` → `verified` in `.specify/memory/roadmap.md`."
            + (f" Item issue: #{closes}." if closes else "") + "\n\n"
            "**Merging this PR approves the status change.** The pilot checks that the merged change is exactly "
            "this edit, and that an allowed approver merged it. The pilot never merges.\n")


def publish_status(run, cfg, imgs: dict, runner, task: str, base_rev: str, bundle: Path) -> dict:
    """004 R6: open the status-change draft PR for `task`'s item. Idempotent like task PRs."""
    existing = [p for p in prs(run) if p.get("kind") == "status" and p["task"] == task]
    if existing:
        return existing[0]
    from . import ghsync
    closes = ghsync.issue_for(run, task)
    item = paths.TASKS[task]
    head = f"pilot/{run.id}/status-{item}"
    spec = {"repo": cfg.repo, "head": head, "base": cfg.branch, "base_revision": base_rev,
            "title": f"pilot {run.id}: {item} verified", "body": status_body(run, task, closes),
            "run_id": run.id, "task": task, **_login_hint(cfg)}
    run.event("publish", task, None, "status_pr_started", head)
    res = _submit(run, imgs, runner, f"status-{task.lower()}", spec, bundle)
    view = json.loads(fixture.gh("pr", "view", str(res["number"]), "-R", cfg.repo, "--json",
                                 "isDraft,author,headRefName,url").stdout)
    entry = {"task": task, "kind": "status", "number": res["number"], "url": view["url"], "head": view["headRefName"],
             "draft": bool(view["isDraft"]), "author": view["author"]["login"],
             "publisher_identity": res.get("token_login"), "base_revision": base_rev, "created": bool(res.get("created"))}
    if not entry["draft"] or entry["head"] != head:
        raise PublishError(f"PR #{entry['number']} is not the expected draft for {head}")
    evidence.write_json(run.dir / "prs.json", prs(run) + [entry])
    run.event("publish", task, None, "status_pr_opened", entry["url"])
    return entry
