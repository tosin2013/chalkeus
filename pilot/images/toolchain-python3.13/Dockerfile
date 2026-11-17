# Toolchain image: Python 3.13 (014, contracts/toolchain-image.md). It only carries
# /opt/toolchain; an init container copies it into the worker, prefetch and grader pods. The
# interpreter is python-build-standalone's relocatable build, verified against the release's
# SHA256SUMS and the pin. Non-root, group 0, no setuid files.
ARG DEBIAN_BASE

FROM ${DEBIAN_BASE} AS fetch
ARG PYTHON313_REPO
ARG PYTHON313_TAG
ARG PYTHON313_ASSET
ARG PYTHON313_ASSET_SHA256
RUN apt-get update \
 && apt-get install -y --no-install-recommends curl=7.88.1-10+deb12u15 ca-certificates=20250419~deb12u2 \
 && rm -rf /var/lib/apt/lists/*
WORKDIR /fetch
RUN set -eu; base="https://github.com/${PYTHON313_REPO}/releases/download/${PYTHON313_TAG}"; \
    curl -fsSLO "${base}/${PYTHON313_ASSET}"; curl -fsSLO "${base}/SHA256SUMS"; \
    grep "  ${PYTHON313_ASSET}\$" SHA256SUMS | sha256sum -c -; \
    echo "${PYTHON313_ASSET_SHA256}  ${PYTHON313_ASSET}" | sha256sum -c -; \
    mkdir -p /opt/toolchain && tar -xzf "${PYTHON313_ASSET}" -C /opt/toolchain; \
    find /opt/toolchain -perm /6000 -type f -exec chmod ug-s {} +; \
    # No strip: binutils' strip breaks this build's versioned symbols (found in testing).
    rm -rf /opt/toolchain/python/lib/python3.13/test /opt/toolchain/python/lib/python3.13/idlelib
COPY toolchain.json /opt/toolchain/toolchain.json
RUN chgrp -R 0 /opt/toolchain && chmod -R g=u /opt/toolchain

FROM ${DEBIAN_BASE}
COPY --from=fetch /opt/toolchain /opt/toolchain
USER 10004:0
# The init container runs this copy (-R keeps modes and links; no -a, which fails on a mount
# point the arbitrary UID doesn't own).
CMD ["cp", "-R", "/opt/toolchain/.", "/dest/"]
