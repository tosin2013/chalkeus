# Toolchain image: JDK 21 (Eclipse Temurin) + Maven (014 T038, contracts/toolchain-image.md). Only
# /opt/toolchain; an init container copies it into the worker, prefetch and grader pods. The JDK
# is verified against Adoptium's .sha256.txt and the pin, Maven against Apache's .sha512 and the
# pin. Non-root, group 0, no setuid files.
ARG DEBIAN_BASE

FROM ${DEBIAN_BASE} AS fetch
ARG TARGETARCH
ARG TEMURIN21_RELEASE
ARG TEMURIN21_FILE_VERSION
ARG TEMURIN21_SHA256_AMD64
ARG TEMURIN21_SHA256_ARM64
ARG MAVEN_VERSION
ARG MAVEN_SHA512
RUN apt-get update \
 && apt-get install -y --no-install-recommends curl=7.88.1-10+deb12u15 ca-certificates=20250419~deb12u2 xz-utils=5.4.1-1+deb12u2 \
 && rm -rf /var/lib/apt/lists/*
WORKDIR /fetch
RUN set -eu; case "${TARGETARCH:-amd64}" in amd64) a=x64; sum="${TEMURIN21_SHA256_AMD64}";; arm64) a=aarch64; sum="${TEMURIN21_SHA256_ARM64}";; *) echo "unsupported $TARGETARCH"; exit 1;; esac; \
    f="OpenJDK21U-jdk_${a}_linux_hotspot_${TEMURIN21_FILE_VERSION}.tar.gz"; \
    rel=$(printf %s "${TEMURIN21_RELEASE}" | sed 's/+/%2B/'); base="https://github.com/adoptium/temurin21-binaries/releases/download/${rel}"; \
    curl -fsSLO "${base}/${f}"; curl -fsSLO "${base}/${f}.sha256.txt"; \
    sha256sum -c "${f}.sha256.txt"; echo "${sum}  ${f}" | sha256sum -c -; \
    mkdir -p /opt/toolchain/jdk && tar -xzf "${f}" -C /opt/toolchain/jdk --strip-components=1; \
    rm -rf /opt/toolchain/jdk/lib/src.zip /opt/toolchain/jdk/man /opt/toolchain/jdk/demo; \
    m="apache-maven-${MAVEN_VERSION}-bin.tar.gz"; \
    curl -fsSLO "https://archive.apache.org/dist/maven/maven-3/${MAVEN_VERSION}/binaries/${m}"; \
    curl -fsSLO "https://archive.apache.org/dist/maven/maven-3/${MAVEN_VERSION}/binaries/${m}.sha512"; \
    echo "$(cut -d' ' -f1 "${m}.sha512")  ${m}" | sha512sum -c -; echo "${MAVEN_SHA512}  ${m}" | sha512sum -c -; \
    mkdir -p /opt/toolchain/maven && tar -xzf "${m}" -C /opt/toolchain/maven --strip-components=1
COPY toolchain.json /opt/toolchain/toolchain.json
# The Maven prefetch helper (PILOT_INSTALL_COMMAND=pilot-maven-prefetch): see the script.
COPY pilot-maven-prefetch /opt/toolchain/bin/pilot-maven-prefetch
RUN find /opt/toolchain -perm /6000 -type f -exec chmod ug-s {} + \
 && chgrp -R 0 /opt/toolchain && chmod -R g=u /opt/toolchain

FROM ${DEBIAN_BASE}
COPY --from=fetch /opt/toolchain /opt/toolchain
USER 10004:0
# The init container runs this copy (-R keeps modes and links; no -a, which fails on a mount
# point the arbitrary UID doesn't own).
CMD ["cp", "-R", "/opt/toolchain/.", "/dest/"]
