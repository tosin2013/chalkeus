# Image platforms (031 T012, quickstart §3)

Run 2026-11-25 on the Linux amd64 machine, against `pilot/images.lock` at `bb41d85`. Every image is an index with `linux/amd64` and `linux/arm64`; the `unknown/unknown` entries are the build attestations that BuildKit attaches to each platform, not images.

```text
WORKER_IMAGE=ghcr.io/tosin2013/chalkeus-worker@sha256:7f88c9652600739f47aa508fd1e0fb690050843a14ab8fbe3c7b606e29d52ba3
  Platform:    linux/amd64
  Platform:    unknown/unknown
  Platform:    linux/arm64
  Platform:    unknown/unknown
TOOLS_IMAGE=ghcr.io/tosin2013/chalkeus-tools@sha256:12754e069b7e5ffa011cc67a257c7cf8628f3e88a7cd6458811bd11236e5f7bc
  Platform:    linux/amd64
  Platform:    unknown/unknown
  Platform:    linux/arm64
  Platform:    unknown/unknown
EGRESS_IMAGE=ghcr.io/tosin2013/chalkeus-egress@sha256:4733b89c1eb912cfe4884eda79bef214ddd6055ac54eb47a9e611088561e97bc
  Platform:    linux/amd64
  Platform:    unknown/unknown
  Platform:    linux/arm64
  Platform:    unknown/unknown
TOOLCHAIN_PYTHON313_IMAGE=ghcr.io/tosin2013/chalkeus-toolchain-python3.13@sha256:fcde0f539ead2ae023f0161d3f7232d9af9735505f42cbb0526ddf7d7e8eb2a8
  Platform:    linux/amd64
  Platform:    unknown/unknown
  Platform:    linux/arm64
  Platform:    unknown/unknown
TOOLCHAIN_GO_IMAGE=ghcr.io/tosin2013/chalkeus-toolchain-go@sha256:9261a7edde135609ea9b44ad6b322236ff47579a7af213bf0d466fa62c2c9649
  Platform:    linux/amd64
  Platform:    unknown/unknown
  Platform:    linux/arm64
  Platform:    unknown/unknown
TOOLCHAIN_JDK21_MAVEN_IMAGE=ghcr.io/tosin2013/chalkeus-toolchain-jdk21-maven@sha256:4af221e41f30c57cfdfd7a88f8dfe0b52ee6f500e3aaeade47ebe36522d322bc
  Platform:    linux/amd64
  Platform:    unknown/unknown
  Platform:    linux/arm64
  Platform:    unknown/unknown
TOOLCHAIN_NODE24_IMAGE=ghcr.io/tosin2013/chalkeus-toolchain-node24@sha256:bf46eef6eab621c6c38e87e3b130f878490b333ca2d394b6aab212e63f2c25af
  Platform:    linux/amd64
  Platform:    unknown/unknown
  Platform:    linux/arm64
  Platform:    unknown/unknown
```
