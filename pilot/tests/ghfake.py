"""A fake GitHub for 004 tests: in-memory issues, labels, comments and permissions; no network.

Install with `fake.patch(self)` in setUp. It replaces the I/O seams of `ghsync` and `ghapprove`
(`read_issues`, `read_labels`, `run_ghops`, `read_comments`, `read_permission`).
"""
from __future__ import annotations

from unittest import mock


class FakeGitHub:
    def __init__(self, publisher="pub-bot", permissions=None):
        self.issues: list[dict] = []
        self.labels: set[str] = set()
        self.comments: dict[int, list[dict]] = {}
        self.permissions = dict(permissions or {})
        self.publisher = publisher
        self.ops: list[tuple[str, dict]] = []
        self.fail: str | None = None  # set to an op name (or "read") to simulate a GitHub error
        self._next = 1
        self._cid = 1000

    # --- reads (operator's gh) ---
    def read_issues(self, cfg):
        if self.fail == "read":
            raise RuntimeError("GitHub unreachable")
        return [dict(i, labels=list(i["labels"])) for i in self.issues]

    def read_labels(self, cfg):
        if self.fail == "read":
            raise RuntimeError("GitHub unreachable")
        return set(self.labels)

    def read_comments(self, cfg, number):
        if self.fail == "read":
            raise RuntimeError("GitHub unreachable")
        return [dict(c) for c in self.comments.get(number, [])]

    def read_permission(self, cfg, login):
        if self.fail == "read":
            raise RuntimeError("GitHub unreachable")
        return self.permissions.get(login, "none")

    # --- writes (publisher credential via ghops) ---
    def run_ghops(self, cfg, op, payload):
        if self.fail in (op, "write"):
            raise RuntimeError(f"ghops {op} failed")
        self.ops.append((op, payload))
        if op == "whoami":
            return {"login": self.publisher}
        if op == "ensure-labels":
            self.labels |= {lab["name"] for lab in payload["labels"]}
            return {"created": [lab["name"] for lab in payload["labels"]]}
        if op == "create-issue":
            n = self._next
            self._next += 1
            self.issues.append({"number": n, "title": payload["title"], "body": payload["body"],
                                "labels": list(payload.get("labels", [])), "state": "open"})
            return {"number": n}
        if op == "edit-issue":
            i = self._issue(payload["number"])
            if "body" in payload:
                i["body"] = payload["body"]
            if "labels" in payload:
                i["labels"] = list(payload["labels"])
            if "title" in payload:
                i["title"] = payload["title"]
            return {"number": i["number"]}
        if op == "comment":
            return {"id": self.add_comment(payload["number"], payload["body"], self.publisher)}
        if op == "close-issue":
            self._issue(payload["number"])["state"] = "closed"
            return {"number": payload["number"]}
        raise AssertionError(op)

    # --- helpers for tests ---
    def add_comment(self, number, body, author, created_at=None, edited=False, app=False):
        self._cid += 1
        ts = created_at or f"2026-11-16T10:{self._cid % 60:02d}:00Z"
        self.comments.setdefault(number, []).append({
            "id": self._cid, "body": body, "author": author, "created_at": ts,
            "updated_at": ("2026-11-16T23:59:00Z" if edited else ts), "app": app})
        return self._cid

    def _issue(self, n):
        return next(i for i in self.issues if i["number"] == n)

    def patch(self, test):
        from specfoundry_pilot import ghsync
        for name in ("read_issues", "read_labels", "run_ghops"):
            p = mock.patch.object(ghsync, name, getattr(self, name))
            p.start()
            test.addCleanup(p.stop)
        try:
            from specfoundry_pilot import ghapprove
        except ImportError:
            return self
        for name in ("read_comments", "read_permission"):
            p = mock.patch.object(ghapprove, name, getattr(self, name))
            p.start()
            test.addCleanup(p.stop)
        return self
