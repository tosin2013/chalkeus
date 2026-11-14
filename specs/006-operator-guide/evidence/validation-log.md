# 006 validation log: fresh-machine run (owner)

Round 1: 2026-11-14. Cells marked `<…>` are still for the owner to fill from the validation machine.

Follow only `docs/operator-guide.md` from GitHub. Record everything below. Never paste secrets,
hostnames, IPs or personal data.

## Machine

- OS: Ubuntu 24.04.5 LTS (Noble Numbat), architecture: amd64
- CPU: 8 × Intel Xeon (Cascadelake), RAM: 15 GiB, free disk: 90 GiB (7 % used on /)
- Fresh? No prior specfoundry, kind, `~/.config/specfoundry`, `~/.config/opencode`: yes

## Sections

| Section | Start (UTC) | Duration | Hesitations / deviations / outside lookups | Guide fix (commit) |
|---|---|---|---|---|
| 3 Before you start | 2026-11-14 ~15:50 UTC | ~5 min | none | — |
| 4 Install the tools | 2026-11-14 ~15:55 UTC | ~8 min | none | — |
| 5 Model provider (was 6) | 2026-11-14 ~16:03 UTC | ~4 min | **F1** `publisher-token` failed: `pilot.env` didn't exist yet. **F2** (owner): reuse an existing OpenCode setup | b4868d6 (sections swapped, errors name the fix); a33427c, 0b04856 (`import-opencode`, `docs/providers.md`) |
| 6 Fixture repo + token (was 5) | 2026-11-14 ~16:04 UTC | ~1 min | **F3** first token was too broad (`canary_push: True`); the check caught it, but the failing token was still saved to `publisher.env` | this branch: only a token that passes the check is saved |
| 7 Stand up | 2026-11-14 ~16:05 UTC | ~3 min | none | — |
| 8 Run the pilot | 2026-11-14 ~16:05 UTC | ~10 min (excl. model work) | A/1 grade_fail → retry confirmed (AU11 induction) | — |
| 9 Merge pause + resume | 2026-11-14 ~16:09 UTC | ~6 min (349 s merge wait) | Owner merged fixture PR #5 (A) and the B PR | — |
| 10 Read the evidence | 2026-11-14 ~16:24 UTC | ~5 min | none | — |
| 11 Tear down | 2026-11-14 ~16:30 UTC | ~2 min | none | — |

**Total hands-on time** (excluding downloads and model work): ~44 min
**Outside lookups** (source code, prior notes, help): 0
**Run id / audit / teardown**: 20261114T160532Z / 11/11 / 3/3

## Troubleshooting spot checks

> **Status: not yet run.** Rows below are placeholders only. Run `pilot/pilot reset-fixture` first,
> then trigger each failure on purpose and resolve it using only `docs/troubleshooting.md`.

| Failure triggered | Entry used | Resolved with the doc only? |
|---|---|---|
| tag-only image reference | | |
| wrong CNI hash | | |
| `pilot run` without a terminal | | |
| pilot PR left open before `run` | | |
| `resume` before merging | | |

## Privacy check

`grep -rniF '<model host>' docs/ specs/006-operator-guide/`: no output (2026-11-14; also the host's first label, the READMEs, and provider names: 0 matches)
`pilot/pilot audit --run 20261114T160532Z` AU08: pass
