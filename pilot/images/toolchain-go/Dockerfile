# Toolchain image: Go (014 T038, contracts/toolchain-image.md). Only /opt/toolchain; an init
# container copies it into the worker, prefetch and grader pods. The archive is verified against
# the pin (go.dev publishes the SHA-256 in its download list). Non-root, group 0, no setuid files.
ARG DEBIAN_BASE

FROM ${DEBIAN_BASE} AS fetch
ARG TARGETARCH
ARG GO_VERSION
ARG GO_SHA256_AMD64
ARG GO_SHA256_ARM64
RUN apt-get update \
 && apt-get install -y --no-install-recommends curl=7.88.1-10+deb12u15 ca-certificates=20250419~deb12u2 xz-utils=5.4.1-1+deb12u2 \
 && rm -rf /var/lib/apt/lists/*
WORKDIR /fetch
RUN set -eu; case "${TARGETARCH:-}" in amd64) sum="${GO_SHA256_AMD64}";; arm64) sum="${GO_SHA256_ARM64}";; *) echo "unsupported TARGETARCH '${TARGETARCH:-}'"; exit 1;; esac; \
    f="${GO_VERSION}.linux-${TARGETARCH:-}.tar.gz"; \
    curl -fsSLO "https://go.dev/dl/${f}"; echo "${sum}  ${f}" | sha256sum -c -; \
    mkdir -p /opt/toolchain && tar -xzf "${f}" -C /opt/toolchain; \
    rm -rf /opt/toolchain/go/test /opt/toolchain/go/api /opt/toolchain/go/doc
COPY toolchain.json /opt/toolchain/toolchain.json
RUN find /opt/toolchain -perm /6000 -type f -exec chmod ug-s {} + \
 && chgrp -R 0 /opt/toolchain && chmod -R g=u /opt/toolchain

FROM ${DEBIAN_BASE}
COPY --from=fetch /opt/toolchain /opt/toolchain
USER 10004:0
# The init container runs this copy (-R keeps modes and links; no -a, which fails on a mount
# point the arbitrary UID doesn't own).
CMD ["cp", "-R", "/opt/toolchain/.", "/dest/"]
