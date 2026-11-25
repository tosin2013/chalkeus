# T030 / SC-005: the guide's macOS install steps, scripted (2026-11-25)

**Owner decision (2026-11-25):** the full fresh-Mac walkthrough (SC-005) moves to roadmap item
034, for a community contributor. In its place, 031 records this scripted check of the operator
guide's "Install the tools on a Mac" section (PR #211). It is partial evidence for SC-005, not the
walkthrough itself.

**Method:**
- An empty home folder (`HOME` set to a new temporary folder), with the shell `PATH` reset to
  macOS's default (`/etc/paths`) plus Homebrew.
- Every command of the section was run exactly as written, in order.
- Host: the maintainer's Apple Silicon Mac (macOS 27.0.1). Docker Desktop and `gh` were already
  installed.

| Guide step | Result |
|---|---|
| `mkdir -p ~/.local/bin` and the `~/.zprofile` `PATH` line | `~/.local/bin` is on `PATH` in a new shell |
| spec 0.4.1: `gh release download …darwin_arm64…`, `shasum -a 256 -c --ignore-missing`, `install` | `spec_0.4.1_darwin_arm64.tar.gz: OK`; `spec --version` → `spec 0.4.1`, found in the new `~/.local/bin` |
| Go 1.27.4: `curl …darwin-arm64.tar.gz`, the guide's checksum through `shasum -a 256 -c -`, `tar` | `go1.27.4.darwin-arm64.tar.gz: OK`; `go version go1.27.4 darwin/arm64` |
| `git clone https://github.com/tosin2013/chalkeus.git` | cloned at `c8de29e` |
| The final check: `docker run --rm hello-world` | `Hello from Docker!` (the `docker` command is in `/usr/local/bin`, on macOS's default `PATH`) |
| The final check: `gh auth status` | not tested: sign-in can't be reproduced in a throwaway home |

**What this doesn't cover**, and 034 does:
- installing Docker Desktop and setting its resources;
- `brew install gh` and `gh auth login` on a machine without them;
- a reader's view of the guide;
- the fixture run that follows.

The checksums in the guide come from go.dev's official release list (whose Linux checksum
matches the one already in the guide) and from spec-status's `spec_0.4.1_checksums.txt`.
