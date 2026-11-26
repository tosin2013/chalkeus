"""Reads the greetlib version pinned in requirements.txt (greetlib itself is never installed)."""
import re
from pathlib import Path

REQUIREMENTS = Path(__file__).resolve().parents[2] / "requirements.txt"


def pinned_major():
    m = re.search(r"^greetlib==(\d+)", REQUIREMENTS.read_text(), re.M)
    return int(m.group(1)) if m else 0
