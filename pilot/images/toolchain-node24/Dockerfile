# Toolchain image: Node.js 24 + npm (014 T038, contracts/toolchain-image.md). Only /opt/toolchain;
# an init container copies it into the worker, prefetch and grader pods. The tarball is verified
# against nodejs.org's SHASUMS256.txt and the pin. Non-root, group 0, no setuid files.
ARG DEBIAN_BASE

FROM ${DEBIAN_BASE} AS fetch
ARG TARGETARCH
ARG NODE24_VERSION
ARG NODE24_SHA256_AMD64
ARG NODE24_SHA256_ARM64
RUN apt-get update \
 && apt-get install -y --no-install-recommends curl=7.88.1-10+deb12u15 ca-certificates=20250419~deb12u2 xz-utils=5.4.1-1+deb12u2 \
 && rm -rf /var/lib/apt/lists/*
WORKDIR /fetch
RUN set -eu; case "${TARGETARCH:-amd64}" in amd64) a=x64; sum="${NODE24_SHA256_AMD64}";; arm64) a=arm64; sum="${NODE24_SHA256_ARM64}";; *) echo "unsupported $TARGETARCH"; exit 1;; esac; \
    f="node-${NODE24_VERSION}-linux-${a}.tar.xz"; base="https://nodejs.org/dist/${NODE24_VERSION}"; \
    curl -fsSLO "${base}/${f}"; curl -fsSLO "${base}/SHASUMS256.txt"; \
    grep "  ${f}\$" SHASUMS256.txt | sha256sum -c -; echo "${sum}  ${f}" | sha256sum -c -; \
    mkdir -p /opt/toolchain/node && tar -xJf "${f}" -C /opt/toolchain/node --strip-components=1; \
    rm -rf /opt/toolchain/node/include /opt/toolchain/node/share
COPY toolchain.json /opt/toolchain/toolchain.json
RUN find /opt/toolchain -perm /6000 -type f -exec chmod ug-s {} + \
 && chgrp -R 0 /opt/toolchain && chmod -R g=u /opt/toolchain

FROM ${DEBIAN_BASE}
COPY --from=fetch /opt/toolchain /opt/toolchain
USER 10004:0
# The init container runs this copy (-R keeps modes and links; no -a, which fails on a mount
# point the arbitrary UID doesn't own).
CMD ["cp", "-R", "/opt/toolchain/.", "/dest/"]
