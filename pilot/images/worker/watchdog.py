"""Spend-cap watchdog for one OpenCode attempt (research R2).

    opencode run --format json ... | watchdog.py <harness_pid>

Tees every event line to /tmp/harness.jsonl, sums step_finish cost and tokens (keyed by
part id), and SIGTERMs the harness when the cap is reached. Writes /tmp/spend.json at EOF.
Cap modes: "cost" while any non-zero cost has been reported; otherwise "token-fallback".

030 (soft deadline): with PILOT_SOFT_DEADLINE_S set (seconds from the watchdog's start), the
harness is SIGTERMed when it passes, and SIGKILLed 15 s later if it is still alive, so the
entrypoint can still bundle the change and report usage before the Job's own deadline.
spend.json then has "stopped_by_deadline": true. Unset or empty: no soft deadline (as before).
"""
from __future__ import annotations

import json
import os
import signal
import sys
import threading


def tokens_of(t) -> int:
    if not isinstance(t, dict):
        return 0
    return int(t.get("input") or 0) + int(t.get("output") or 0) + int(t.get("reasoning") or 0)


class Meter:
    def __init__(self, cap_usd: float, token_cap: int):
        self.cap_usd, self.token_cap = cap_usd, token_cap
        self.seen: dict[str, tuple[float, int]] = {}
        self.stopped_by_cap = False
        self.stopped_by_deadline = False

    def feed(self, line: str) -> bool:
        """Account one event line; return True if the cap is now exceeded."""
        try:
            e = json.loads(line)
        except ValueError:
            return False
        if not isinstance(e, dict) or e.get("type") != "step_finish":
            return False
        p = e.get("part") or {}
        key = str(p.get("id") or len(self.seen))
        self.seen[key] = (float(p.get("cost") or 0.0), tokens_of(p.get("tokens")))
        return self.exceeded()

    @property
    def cost(self) -> float:
        return round(sum(c for c, _ in self.seen.values()), 6)

    @property
    def tokens(self) -> int:
        return sum(t for _, t in self.seen.values())

    @property
    def mode(self) -> str:
        return "cost" if self.cost > 0 else "token-fallback"

    def exceeded(self) -> bool:
        return self.cost >= self.cap_usd if self.mode == "cost" else self.tokens >= self.token_cap

    def report(self) -> dict:
        return {"cap_usd": self.cap_usd, "token_cap": self.token_cap, "cap_mode": self.mode,
                "cost_usd": self.cost, "tokens": self.tokens, "stopped_by_cap": self.stopped_by_cap,
                "stopped_by_deadline": self.stopped_by_deadline}


def _signal(pid: int, sig) -> bool:
    """Send sig to pid; False if it no longer exists."""
    try:
        os.kill(pid, sig)
        return True
    except ProcessLookupError:
        return False


def soft_deadline(env) -> float:
    """PILOT_SOFT_DEADLINE_S as seconds (0 = none)."""
    try:
        return max(0.0, float(env.get("PILOT_SOFT_DEADLINE_S") or 0))
    except ValueError:
        return 0.0


def arm_deadline(m: Meter, pid: int, seconds: float, kill_after: float = 15.0):
    """Start the soft-deadline timer; returns it (or None) so callers and tests can cancel it."""
    if not pid or seconds <= 0:
        return None

    def fire():
        if m.stopped_by_cap:
            return
        m.stopped_by_deadline = True
        if _signal(pid, signal.SIGTERM):
            k = threading.Timer(kill_after, lambda: _signal(pid, signal.SIGKILL))
            k.daemon = True
            k.start()

    t = threading.Timer(seconds, fire)
    t.daemon = True
    t.start()
    return t


def main(argv) -> int:
    pid = int(argv[1]) if len(argv) > 1 else 0
    m = Meter(float(os.environ.get("PILOT_SPEND_CAP_USD", "1.00")), int(os.environ.get("PILOT_TOKEN_CAP", "2000000")))
    timer = arm_deadline(m, pid, soft_deadline(os.environ))
    tmp = os.environ.get("PILOT_TMP", "/tmp")  # overridable for tests (024 author role)
    with open(os.path.join(tmp, "harness.jsonl"), "a") as log:
        for line in sys.stdin:
            log.write(line)
            log.flush()
            if m.feed(line) and not m.stopped_by_cap and not m.stopped_by_deadline:
                m.stopped_by_cap = True
                if pid:
                    _signal(pid, signal.SIGTERM)
    if timer is not None:
        timer.cancel()
    with open(os.path.join(tmp, "spend.json"), "w") as f:
        json.dump(m.report(), f)
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
