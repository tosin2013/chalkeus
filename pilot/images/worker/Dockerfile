# Thin worker image (C-09): OpenCode from the verified upstream release, never an operator's
# local binary. Non-root, no sudo, no secrets; runs under an arbitrary UID with group 0
# (OpenShift restricted SCC friendly).
ARG DEBIAN_BASE

FROM ${DEBIAN_BASE} AS fetch
ARG OPENCODE_REPO
ARG OPENCODE_TAG
# 031: per-architecture pins; BuildKit sets TARGETARCH. An empty or other value fails (no
# default to amd64), so an image can't silently carry the wrong platform's binary.
ARG TARGETARCH
ARG OPENCODE_ASSET_AMD64
ARG OPENCODE_ASSET_ARM64
ARG OPENCODE_ASSET_SHA256_AMD64
ARG OPENCODE_ASSET_SHA256_ARM64
ARG OPENCODE_SHA256_AMD64
ARG OPENCODE_SHA256_ARM64
RUN apt-get update \
 && apt-get install -y --no-install-recommends curl=7.88.1-10+deb12u15 ca-certificates=20250419~deb12u2 \
 && rm -rf /var/lib/apt/lists/*
WORKDIR /fetch
RUN set -eu; case "${TARGETARCH:-}" in \
      amd64) asset="${OPENCODE_ASSET_AMD64}"; sum="${OPENCODE_ASSET_SHA256_AMD64}"; bin="${OPENCODE_SHA256_AMD64}";; \
      arm64) asset="${OPENCODE_ASSET_ARM64}"; sum="${OPENCODE_ASSET_SHA256_ARM64}"; bin="${OPENCODE_SHA256_ARM64}";; \
      *) echo "unsupported TARGETARCH '${TARGETARCH:-}'"; exit 1;; esac; \
    base="https://github.com/${OPENCODE_REPO}/releases/download/${OPENCODE_TAG}"; \
    curl -fsSLO "${base}/${asset}"; curl -fsSLO "${base}/SHA256SUMS"; \
    grep " ${asset}\$" SHA256SUMS | sha256sum -c -; \
    echo "${sum}  ${asset}" | sha256sum -c -; \
    tar -xzf "${asset}"; \
    echo "${bin}  opencode" | sha256sum -c -

FROM ${DEBIAN_BASE}
# Exact versions resolved 2026-11-13 with `apt-cache policy` on the pinned base.
RUN apt-get update \
 && apt-get install -y --no-install-recommends \
      git=1:2.39.5-0+deb12u3 ca-certificates=20250419~deb12u2 python3=3.11.2-1+b1 \
 && rm -rf /var/lib/apt/lists/* \
 && useradd --uid 10001 --gid 0 --create-home --home-dir /home/worker --shell /usr/sbin/nologin worker \
 && mkdir -p /work /home/worker/.config \
 && chgrp -R 0 /home/worker /work && chmod -R g=u /home/worker /work \
 && ! command -v sudo
COPY --from=fetch /fetch/opencode /usr/local/bin/opencode
COPY entrypoint.sh author.sh watchdog.py frame.py /usr/local/lib/pilot/
# Kelos-compatible entrypoint shim (FR-010): lets Kelos users run this hardened image.
# It adds no dependency on Kelos.
COPY kelos_entrypoint.sh /kelos_entrypoint.sh
RUN chmod 0755 /usr/local/bin/opencode /usr/local/lib/pilot/entrypoint.sh /usr/local/lib/pilot/author.sh /kelos_entrypoint.sh
ENV HOME=/home/worker PYTHONDONTWRITEBYTECODE=1 \
    OPENCODE_DISABLE_AUTOUPDATE=1 \
    OPENCODE_DISABLE_MODELS_FETCH=1 \
    OPENCODE_DISABLE_FREE_MODELS=1 \
    OPENCODE_DISABLE_CLAUDE_CODE=1
USER 10001:0
WORKDIR /work
ENTRYPOINT ["/usr/local/lib/pilot/entrypoint.sh"]
