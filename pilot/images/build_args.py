#!/usr/bin/env python3
"""Single source of image build arguments, from pilot/pins.env (used by `pilot up --local-build`
and by .github/workflows/pilot-images.yml).

    build_args.py <worker|tools|egress> [--format=docker|github]

docker: one `KEY=VALUE` per line (for --build-arg); github: `KEY=VALUE` lines for a multi-line
`build-args` input. Stdlib only.
"""
import sys
from pathlib import Path

PINS = Path(__file__).resolve().parents[1] / "pins.env"


def read_pins(path=PINS) -> dict:
    out = {}
    for line in path.read_text().splitlines():
        line = line.strip()
        if line and not line.startswith("#") and "=" in line:
            k, v = line.split("=", 1)
            out[k] = v
    return out


def split_release(ref: str) -> tuple[str, str]:
    repo, tag = ref.split("@", 1)
    return repo, tag


ARCHES = ("AMD64", "ARM64")


def per_arch(*keys: str) -> list[str]:
    """Each key with an _AMD64 and an _ARM64 suffix (031: the Dockerfiles pick by TARGETARCH)."""
    return [f"{k}_{a}" for k in keys for a in ARCHES]


def pick(p: dict, keys: list[str]) -> dict:
    """The named pins; a missing one fails, naming it (C-09: no fallback to another value)."""
    missing = [k for k in keys if not p.get(k)]
    if missing:
        raise SystemExit(f"pins.env: missing {missing[0]}")
    return {k: p[k] for k in keys}


def build_args(image: str, pins: dict | None = None) -> dict:
    p = pins or read_pins()
    if image == "worker":
        repo, tag = split_release(pick(p, ["OPENCODE_RELEASE"])["OPENCODE_RELEASE"])
        return {**pick(p, ["DEBIAN_BASE"]), "OPENCODE_REPO": repo, "OPENCODE_TAG": tag,
                **pick(p, per_arch("OPENCODE_ASSET", "OPENCODE_ASSET_SHA256", "OPENCODE_SHA256"))}
    if image == "tools":
        repo, tag = split_release(pick(p, ["SPEC_RELEASE"])["SPEC_RELEASE"])
        version = tag.lstrip("v")
        return {**pick(p, ["DEBIAN_BASE"]), "SPEC_REPO": repo, "SPEC_TAG": tag,
                **pick(p, per_arch("SPEC_ASSET", "SPEC_ASSET_SHA256", "SPEC_SHA256")),
                "SPEC_CHECKSUMS": f"spec_{version}_checksums.txt"}
    if image == "egress":
        return pick(p, ["ALPINE_BASE"])
    if image == "toolchain-python3.13":
        repo, tag = split_release(pick(p, ["PYTHON313_RELEASE"])["PYTHON313_RELEASE"])
        return {**pick(p, ["DEBIAN_BASE"]), "PYTHON313_REPO": repo, "PYTHON313_TAG": tag,
                **pick(p, per_arch("PYTHON313_ASSET", "PYTHON313_ASSET_SHA256"))}
    if image == "toolchain-node24":
        return pick(p, ["DEBIAN_BASE", "NODE24_VERSION", *per_arch("NODE24_SHA256")])
    if image == "toolchain-go":
        return pick(p, ["DEBIAN_BASE", "GO_VERSION", *per_arch("GO_SHA256")])
    if image == "toolchain-jdk21-maven":
        return pick(p, ["DEBIAN_BASE", "TEMURIN21_RELEASE", "TEMURIN21_FILE_VERSION",
                        *per_arch("TEMURIN21_SHA256"), "MAVEN_VERSION", "MAVEN_SHA512"])
    raise SystemExit(f"unknown image {image!r}")


if __name__ == "__main__":
    if len(sys.argv) < 2:
        raise SystemExit(__doc__)
    for k, v in build_args(sys.argv[1]).items():
        print(f"{k}={v}")
