# Implementation Plan: Pluggable SCM, phase 1: GitLab and Bitbucket Cloud beside GitHub

**Branch**: `spec-037-pluggable-scm` | **Date**: 2026-11-26 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/037-pluggable-scm/spec.md`

## Summary

Every host operation forge performs on change requests, repositories and accounts moves behind
one interface, `internal/scm.Client`, with GitHub as its first implementation built from today's
code (R1). GitLab and Bitbucket Cloud get plain-HTTP implementations (R2). The work is:

1. **The host interface and GitHub behind it (US1).**
   - `internal/scm`: types (Host, Account, ChangeRequest), the `Client` interface, operator
     credentials and host-side git environment (R8).
   - `internal/scm/github` takes today's calls from `ghloop` (OpenPilotPRs, ClosePR, PR
     read-back), `author` (Identity, openPR), `publisher`, `reconcile.ForFixture` and
     `cli/doctor`.
   - The GitHub loop, the App and the debt report keep their code and stay GitHub-only.
   - Config gains `FORGE_SCM`, `FORGE_SCM_HOST`, `FORGE_SCM_CA_BUNDLE`, per-host repository forms
     and the GitHub-only refusals (contracts/config.md).
2. **Bitbucket Cloud first (US5, US3, US6; owner decision 2026-11-26).**
   - `internal/scm/bitbucket`: the interface over `/2.0` (R3–R7, R11).
   - The repository-token check, the required `restrict_merges` restriction, approvers by
     account UUID (R4, R7).
   - `publish.py` per host (contracts/publisher.md).
   - Egress per host and the new live check P13 (R10); AU03's URL form per host; secret
     heuristics (R12).
   - The companion change request on the target's host (R11).
3. **gitlab.com (US2).** `internal/scm/gitlab` over `/api/v4`; the token check for project
   tokens and dedicated bot accounts (R4, owner decision 2026-11-26); GitLab publisher steps.
4. **Self-managed GitLab (US4).** Host name, CA bundle (Go client, host-side git, publisher pod),
   minimum version 18.0 (R9); proof on a throwaway GitLab CE container (owner decision
   2026-11-26).
5. **doctor and docs (US7).** The rows in contracts/checks.md; `docs/targets.md` per host,
   troubleshooting entries, operator guide, release notes.
6. **Proof.** Bitbucket Cloud, gitlab.com and GitLab CE, in that order: the fixture walkthrough
   and one target item run each, with the refusal cases (quickstart.md).

## Technical Context

**Language/Version**: Go 1.27 (`cmd/chalkeus`, `internal/`); Python 3.13 stdlib for
`pilot/images/tools/publish.py`

**Primary Dependencies**: go-github v85 (unchanged, GitHub only); `net/http` for GitLab
`/api/v4` and Bitbucket `/2.0`; no new Go module (R2); git on the host and in the tools image
(unchanged)

**Storage**: target profiles and `forge.env` (new keys); `targets/<name>.publisher.env` and
`.author.env` (unchanged names, per-host tokens); an optional CA bundle file in the config
folder; run evidence (`run.json`, `prs.json`, `reconcile/*.json`, `checks.json` fields in
data-model.md)

**Testing**: `go test ./...`: a shared contract suite (`internal/scm/scmtest`) run against
fakegh (GitHub) and httptest fakes (GitLab, Bitbucket); config refusals; the no-merge guard for
all three packages; doctor rows; AU03 per host; P13 rendering. `python3 -m unittest`:
`test_publish_hosts.py`. Integration suite (fakegh, kind): unchanged, GitHub only. Live proof:
quickstart.md scenarios 1–4.

**Target Platform**: unchanged hosts (Linux amd64, macOS arm64) and clusters; Git hosts github.com,
gitlab.com, self-managed GitLab ≥ 18.0, Bitbucket Cloud

**Project Type**: CLI and in-cluster Job scripts (single Go module plus `pilot/images`)

**Performance Goals**: no new waits in a run; doctor's host rows add at most one request per row
(under 5 s in total on a normal connection)

**Constraints**:
- With no `FORGE_SCM`, GitHub behaviour and evidence unchanged apart from `scm`, `author_id`,
  `merged_by_id` and P13 (SC-001).
- The publisher stays the only holder of a write credential, scoped to one repository (C-03,
  FR-009); operator credentials never reach a pod (FR-013).
- No merge, approve or accept call anywhere (C-04; no-merge guard).
- The publisher reaches only its run's host (C-07, P13).
- Images unchanged except `publish.py` (tools image), so the image workflow and digest PR run
  once.

**Scale/Scope**: 3 hosts; about 12 host endpoints each; 48 go-github call sites of which about 25
move behind the interface (the loop, App and debt calls stay); 1 new live check; 0 new audit
checks

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | How 037 complies |
|---|---|
| I. The roadmap is the source of truth | Scheduling is unchanged; only where change requests live changes |
| II. Gates fail closed | Unknown host, wrong repository form, a credential wider than one repository, missing scopes, an unreadable approver permission, an old GitLab or an untrusted certificate all refuse before a run; a probe that can't be evaluated fails |
| III. Evidence, not CI, defines done | Each new host is proven by a fixture walkthrough and a target item run with the audit at 11/11 and the refusal cases, not by unit tests; evidence records the host and mergers by id |
| IV. Least privilege (non-negotiable) | Workers and the grader still get no SCM credential (P04, P09, AU04); the publisher's token is checked to reach one repository on every host; Bitbucket's merge-capable token is fenced by a required `restrict_merges` restriction; the operator's credential is used only on the operator's machine; the publisher's egress is limited to its host and proven by P13 |
| V. Humans approve, humans merge | Drafts on every host (native); no merge, approve or accept call (guard extended); the status change counts only when a listed human approver merged it, compared by stable id |
| VI. Build on, don't rebuild | The constitution's "SCMs are pluggable backends behind adapters"; GitHub's implementation reuses today's code; the loop, App and debt report are left as they are; no new dependency |
| VII. Repeatable and portable | Self-managed GitLab with a corporate CA (enterprise); hosts pinned by name in the profile; the proof instance is a pinned GitLab CE version; host-agnostic docs per host |

**Result:** pass, no violations. **Re-check after Phase 1:** still pass. Two trade-offs are
recorded:
- R4: on gitlab.com Free a dedicated bot account's personal token is accepted (owner decision);
  its one-project reach rests on membership and is proven by the associations API before every
  run, not by the token type.
- R4: a Bitbucket repository token can't describe itself, so its "one repository" is proven by
  the token kind plus probes; the probe heuristic is confirmed in the proof.

## Project Structure

### Documentation (this feature)

```text
specs/037-pluggable-scm/
├── plan.md              # this file
├── research.md          # R1–R13
├── data-model.md        # Host, Repo path, Account, ChangeRequest, credentials, evidence
├── quickstart.md        # validation scenarios 0–5
├── contracts/
│   ├── config.md        # profile and forge.env keys, refusals, GitHub-only commands
│   ├── host-interface.md# scm.Client and the per-host endpoint mapping
│   ├── publisher.md     # publish.py per host, read-back
│   └── checks.md        # doctor rows, P13, AU03, secret heuristics
└── tasks.md             # /speckit-tasks
```

### Source Code (repository root)

```text
internal/
├── scm/                       # NEW: types, Client interface, errors, OperatorCredential, GitEnv, URLs
│   ├── scmtest/               # NEW: shared contract suite
│   ├── github/                # NEW: from ghloop (OpenPilotPRs, ClosePR, PR), author (openPR), probe
│   ├── gitlab/                # NEW: /api/v4 client, fake_test.go
│   └── bitbucket/             # NEW: /2.0 client, fake_test.go
├── ghloop/                    # keeps the loop (issues, labels, comments, approvals); PR calls delegate to scm/github
├── config/                    # FORGE_SCM, FORGE_SCM_HOST, FORGE_SCM_CA_BUNDLE; per-host repo forms; refusals
├── fixture/                   # URL and git credentials from scm (no hard-coded github.com)
├── publisher/                 # pr.json scm block; read-back through scm.Client
├── reconcile/                 # PRView and status check through scm.Client; merger by id
├── pipeline/                  # status head via FileAt for non-GitHub; open-PR checks via scm; GitHub-only loop gating
├── author/                    # Identity and Submit through scm.Client (CommitFiles, OpenDraft)
├── cluster/infra.go           # SCMHosts(host); scm-ca ConfigMap
├── checks/                    # P13; scmRE
├── audit/                     # AU03 URL form per host
├── cli/                       # doctor rows; GitHub-only command refusals; askpass subcommand
└── evidence/                  # scm, author_id, merged_by_id
pilot/
├── images/tools/publish.py    # per-host steps (contracts/publisher.md)
├── images/tests/test_publish_hosts.py  # NEW
└── manifests/publisher-job.yaml.tmpl   # optional scm-ca mount
docs/
├── targets.md                 # a section per host
├── troubleshooting.md         # one entry per new refusal
├── operator-guide.md          # hosts supported; section 13 row
└── release-notes.md           # 037 section
specs/037-pluggable-scm/evidence/  # gitlab.com, GitLab CE, Bitbucket Cloud runs
```

**Structure Decision**: one new package tree, `internal/scm`, with one subpackage per host.
`ghloop` keeps the GitHub loop and delegates its change-request calls to `scm/github`, so the loop
code and its tests don't move. Callers that take a raw `*github.Client` today take an
`scm.Client` instead.

## Delivery order

1. **Foundations (US1):** `internal/scm` types and interface, the contract suite, `scm/github`
   from today's code, callers switched, config keys and refusals, evidence fields. Proof: the full
   test suite, and quickstart scenario 1 on GitHub (fixture 11/11; evidence diff limited to the
   new fields). P13 lands here too, so GitHub proves it first.
2. **Bitbucket client and publisher (US5, US3):** `scm/bitbucket` with its fake, the token
   probe, the `restrict_merges` row, host git credentials, `publish.py` Bitbucket steps, egress
   per host, AU03 per host, secret heuristics. The tools image rebuilds once (digest PR, owner
   merges).
3. **Companion on Bitbucket (US6)**, so the owner's test-author session can open hidden-test PRs.
4. **Bitbucket Cloud proof (scenario 2).** Throwaway repositories only. A private target is run
   only after the proof, with `FORGE_EVIDENCE_FEATURE` set to an absolute path outside this
   repository, so its evidence is never committed here (existing behaviour of
   `paths.EvidenceRoot`).
5. **gitlab.com (US2):** `scm/gitlab`, the token probe (project token or dedicated bot account),
   `publish.py` GitLab steps (a second tools-image rebuild), the companion on GitLab; scenario 3.
6. **Self-managed (US4):** host name, CA bundle in Go, git and the publisher pod, version check;
   the GitLab CE container; scenario 4.
7. **doctor and docs (US7)**, release notes; then the debrief and roadmap status.

## Complexity Tracking

No constitution violations to justify.
