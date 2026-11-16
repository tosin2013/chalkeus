<!--
SYNC IMPACT REPORT
==================
Version change: 1.12.0 → 1.13.0
Bump rationale: MINOR — a material change to 014's validation, and notes on 014, 015 and 017.

Changes this revision (owner direction 2026-11-16, proposed):
  - 014: the brownfield validation changes from a Quarkus repository to a legacy Java EE
    upgrade on OpenShift (older EAP/WildFly app to EAP 8 or Quarkus), using Red Hat's migration
    tools inside the process, with an in-cluster model; graded as behaviour unchanged
  - 014 note: toolchain images are UBI-based on OpenShift and built on demand elsewhere
  - 017 note: the greenfield validation runs on every cluster profile
  - 015 note: waves scale the same way on kind, OpenShift and plain Kubernetes; only capacity
    (the concurrency limit) differs
  - 015 note: unblocked tasks inside an item may run in parallel; the pilot owns the schedule
  - New Q12: parallel tasks inside an item
  - New Q13: how planning scales
  - Delivery order (owner direction 2026-11-16): laptop first, end to end (016, 014 on kind,
    017, 013 release), then OpenShift (005, then 014's EAP validation); see Cross-Cutting Notes
  - 014 note: kind and plain-Kubernetes users can ask for a new language toolchain image by
    opening a GitHub issue

Specs affected: 014, 015, 017
Open questions added/resolved: Q12, Q13 added

Previous revision 1.12.0 (2026-11-15):
Version change: 1.11.0 → 1.12.0
Bump rationale: MINOR — a new planned spec (017) and a material change to 014's validation.
Q11 is resolved by 017.

Changes this revision (owner decisions 2026-11-15, after the Go pilot's first fixture run):
  - 014: validated on a real Quarkus (Java) brownfield repository, chosen when 014 is specified;
    the fixture stays the regression test
  - New entry: 017 Greenfield bootstrap, validated on a new Node.js repository whose first
    roadmap item is a human-specified scaffold; depends on 014
  - Q11 resolved: greenfield is supported through 017, after 014

Specs affected: 014, 017 (new)
Open questions added/resolved: Q11 resolved

Previous revision 1.11.0 (2026-11-15):
Version change: 1.10.0 → 1.11.0
Bump rationale: MINOR — a material scope change to 014 and a new Open Question (Q11). 016 gets
a note only; no constraint is reversed.

Changes this revision (owner decisions 2026-11-15, after the 016 US2 merge):
  - 014 scope (in): per-repo worker and grader toolchains; an egress allowlist for the repo's
    package registries; hidden acceptance tests written by a human, with the repo's own test
    suite as a visible check. Note: 014 targets brownfield repos (existing code and tests)
  - New Q11: greenfield bootstrap (an empty repo has nothing to grade against yet)
  - 016 note: stderr diagnostics (-v, -vv, -vvv) are allowed as a debugging aid outside parity

Specs affected: 014, 016
Open questions added/resolved: Q11 added

Previous revision 1.10.0 (2026-11-15):
Version change: 1.9.0 → 1.10.0
Bump rationale: MINOR — a new constraint (C-13), a new planned spec (016), dependency
rewiring (011, 013, 014), 013 moved from needs-info to planned, and notes on 004 and Q9.
No existing constraint is reversed (the roadmap never mandated Python).

Changes this revision (owner decision 2026-11-15, prompted by issue #25 and the Go operator):
  - New C-13: one implementation language, Go, for the CLI and the operator, in one Go module;
    the Python pilot is the reference implementation until the Go CLI reaches parity
  - New entry: 016 Go pilot at parity
  - 013: form decided (one Go binary per platform); status needs-info → planned; depends on 016
  - 014 depends on 004 and 016; 011 depends on 004, 005 and 016
  - 004: spec dir linked; note that the Python implementation is the reference, its validation
    was stopped, and it is validated on the Go pilot in 016
  - Q9: operator language decided (Go, same module); split question narrowed
  - Cross-cutting: the CLI/operator split rule and the operator inventory (from issue #25)

Specs affected: 004, 011, 013, 014, 016 (new)
Open questions added/resolved: Q9 partly resolved (language)

Previous revision 1.9.0 (2026-11-15):
Version change: 1.8.0 → 1.9.0
Bump rationale: MINOR — a new constraint (C-12) and two new planned specs (014, 015).

Changes this revision (owner decisions 2026-11-15):
  - New C-12: humans plan, agents build. Humans write, research, unblock and release roadmap
    items; agents implement only items a human prepared (spec + tasks) and released; every
    unblock passes through a human merge and an approved status change
  - New entry: 014 Any project, any item (run a chosen, prepared item of a real repo)
  - New entry: 015 Released waves (a human releases a set of ready items; the pilot runs them
    in parallel in dependency order; blocked items wait until their dependencies are verified)
  - 010 and 011 notes: proposals still need a human release; cluster mode reuses the wave
    scheduler

Specs affected: 010, 011, 014 (new), 015 (new)
Open questions added/resolved: none (PRD F6 "who launches workers" answered for now: the human)

Previous revision 1.8.0 (2026-11-15):
Version change: 1.7.2 → 1.8.0
Bump rationale: MINOR — a new planned spec (013), new open questions (Q8–Q10), a status
transition, scope/notes corrections on 012, and material scope changes to 005 and 011.

Changes this revision (owner approval 2026-11-15, 012 debrief
specs/012-agent-approvals/roadmap-reviews/debrief-20261115.md):
  - Status: 012 planned → verified (runs 20261114T225256Z and 20261115T020750Z in OpenCode,
    terminal run 20261115T023426Z, all audit 11/11; 7 refusal cases; Claude Code harness
    check; T022 privacy check; no must-address findings); spec dir linked
  - 012 scope: the OpenCode ask rule moves from a committed opencode.json to each operator's
    own OpenCode config (owner decision 2026-11-15; old text struck through)
  - 012 notes: the harness channel also depends on the permission mode (auto/bypass skip the
    prompt); the approvals/ deny rule exists in Claude Code only
  - New entry: 013 Installable pilot release (needs-info)
  - New Q8: using the pilot across projects; harness rules per user; agents that run in a
    real terminal; deny rules beyond Claude Code
  - 005 scope and title: in-cluster planning moves to 011; 005 is the OpenShift target in laptop mode
    (old text struck through)
  - 011: develop on kind, then OpenShift as the release check; lives in operator/ in this
    repo until Q9's split trigger; takes in-cluster planning from 005
  - New Q9: when the operator moves to its own repo
  - New Q10: distribution channels and timing (CLI releases, images, Helm chart on Artifact
    Hub, operator on OperatorHub.io)

Specs affected: 005, 011, 012, 013 (new)
Open questions added/resolved: Q8, Q9, Q10 added

Previous revision 1.7.2 (2026-11-14):
Version change: 1.7.1 → 1.7.2
Bump rationale: PATCH — status transition.

Changes this revision (owner approval 2026-11-14):
  - Status: 006 planned → verified (fresh-machine run 20261114T160532Z: audit 11/11,
    teardown 3/3, 0 outside lookups, ~44 min hands-on; troubleshooting spot checks 5/5;
    tasks 32/32; debrief specs/006-operator-guide/roadmap-reviews/debrief-20261114.md with no
    must-address findings; closes issue #12)

Specs affected: 006
Open questions added/resolved: none

Previous revision 1.7.1 (2026-11-14):
Version change: 1.7.0 → 1.7.1
Bump rationale: PATCH — wording corrections and a spec dir link on 006 (issue #12).

Changes this revision (006 debrief 2026-11-14, issue #12):
  - 006 scope: stale "003 (Kelos)" wording struck through and corrected
  - 006 notes: stale worked-example run id struck through; validation runs recorded
  - 006: spec dir linked (specs/006-operator-guide)

Specs affected: 006
Open questions added/resolved: none

Previous revision 1.7.0 (2026-11-14):
Version change: 1.6.0 → 1.7.0
Bump rationale: MINOR — a new planned spec (012) and a note on 004.

Changes this revision (owner decision 2026-11-14, 006 validation finding F4):
  - New entry: 012 Working with an agent: pilot skill and harness approvals. A first cut
    landed early (branch pilot-agent-approvals) so the 006 validation could continue with an
    agent; status stays planned until a spec formalises the audit-contract change
  - 004: GitHub approvals reuse 012's pending-step approval and channel record

Specs affected: 004, 012 (new)
Open questions added/resolved: none

Previous revision 1.6.0 (2026-11-14):
Version change: 1.5.0 → 1.6.0
Bump rationale: MINOR — a new planned spec (011), a new constraint (C-11), a new open
question (Q7), and material scope changes to 004 and 005.

Changes this revision (owner decisions 2026-11-14, from the 006 fresh-machine validation
discussion):
  - New C-11: two modes, one set of guarantees. Laptop mode (operator off): the CLI
    orchestrates. Cluster mode (operator on): the CLI is a client of an in-cluster operator.
    Human approvals can be given on GitHub in both modes; publisher credentials may be a
    fine-grained token or a GitHub App in either mode
  - New entry: 011 Cluster mode: specfoundry Kubernetes operator
  - 004: GitHub approvals for the human decisions (both modes) added to scope; the publisher
    credential note changes from "replace the token with a GitHub App" to "support both,
    chosen per install" (old text struck through)
  - 005: note that a remote OpenShift cluster is first driven by the laptop CLI (configurable
    kube context, preflight instead of cluster creation, namespace permissions); in-cluster
    orchestration is 011
  - New Q7: product name and CLI interface ("pilot" is a prototype name)
  - Cross-cutting: 007, 009 and 010 run in the operator in cluster mode

Specs affected: 004, 005, 011 (new); 007, 009, 010 (note only)
Open questions added/resolved: Q7 added

Previous revision 1.5.0 (2026-11-13):
Version change: 1.4.0 → 1.5.0
Bump rationale: MINOR — four new planned specs (007–010) and an open question resolved,
plus a status transition (003 → verified) and wording updates.

Changes this revision (from debrief specs/003-m2-kelos-backend/roadmap-reviews/
debrief-20261113T182000Z.md; owner decisions 2026-11-13):
  - Status: 003 planned → verified (validation run 20261113T180203Z on GHCR images: checks
    32/32, audit 11/11, teardown 3/3; CI and enterprise evidence; evidence merged to main via
    PR #3); spec dir linked
  - 003 scope wording and C-09: digests live in pilot/images.lock (outputs, never a workflow
    trigger); pilot/pins.env holds inputs
  - New entries: 007 Operator observability (closes Q6), 008 More harnesses, 009 Period
    spend budget at the gate, 010 Event-proposed runs
  - Per-run provider keys: revisited in 003 with no change; carried to 005
  - Notes: 004 (GitHub Actions PR permission; public-repo permissions field), 005
    (per-run keys; explicit podFailurePolicy candidate), Q1 (public, Apache-2.0)

Specs affected: 003, 004, 005, 007, 008, 009, 010
Open questions added/resolved: Q6 resolved (→ 007)

Previous revision 1.4.0 (2026-11-13):
Version change: 1.3.0 → 1.4.0
Bump rationale: MINOR — material scope change to 003 (renamed: Kelos runner work dropped),
C-05 reworded, a new constraint (C-10 telemetry), a new end state, and a new open question.

Changes this revision (owner decision 2026-11-13; constitution amended to 1.1.0 the same day):
  - Positioning: Kelos is a competitor and design reference, not a dependency or runner
    (constitution VI). Target environments: laptops and enterprise (constitution VII)
  - C-05 reworded accordingly; Kelos runner adoption removed
  - 003 renamed "M2 hardened images and Kelos competitive review"; scope: GitHub-built
    digest-pinned images (laptop and enterprise ready) plus a factual competitive review
    (security bar comparison and a gap list of Kelos features enterprises expect)
  - New C-10: first-party telemetry is operator observability only; no phone-home
  - New end state: runs on laptops and in enterprise environments
  - New Q6: which spec delivers operator observability (C-10)
  - Cross-cutting: PRD F10 drift noted, then resolved the same day (PRD updated by the owner's approval)

Specs affected: 003 (and 005 via C-10)
Open questions added/resolved: Q6 added

Previous revision 1.3.0 (2026-11-13):
Version change: 1.2.0 → 1.3.0
Bump rationale: MINOR — material scope change to 003, an amended constraint (C-05) and a new
constraint (C-09), plus a note on 005.

Changes this revision (owner decision 2026-11-13, after reading Kelos v0.64.0's source during
003 planning: security first):
  - C-05 amended: wrap Kelos, never fork; if Kelos cannot meet the security bar by
    configuration alone, do not adopt it (plain Jobs stays the runner; Kelos is a reference)
  - New C-09: pilot images built by GitHub Actions from verified upstream sources, published
    to a public registry per target platform (GHCR for kind; possibly Quay variants for
    OpenShift), referenced only by digest
  - Scope changed on 003: assess Kelos against a security bar with a possible `do-not-adopt`
    outcome; forking Kelos is out of scope; GitHub-built images added; outcome reworded
  - Note added on 005: decide OpenShift image variants on Quay vs mirroring the C-09 images

Specs affected: 003, 005
Open questions added/resolved: Q2 resolved (wrap or do not adopt; never fork)

Previous revision 1.2.0 (2026-11-13):
Version change: 1.1.0 → 1.2.0
Bump rationale: MINOR — new constraints (C-07, C-08), a new planned spec (006) and a
material scope change on 003, plus a status transition on 002.

Changes this revision (from debrief specs/002-m1-gated-pilot/roadmap-reviews/
debrief-20261113T032500Z.md, approved by the owner 2026-11-13):
  - Status: 002 planned → verified (pilot run 20261113T030655Z: checks 32/32, audit
    11/11, teardown 3/3, mutation test caught all; spec dir linked)
  - New constraints: C-07 (egress through per-role domain-allowlist proxies), C-08 (no
    worker-writable storage shared with the grader; results via framed logs)
  - Scope changed on 003: adds the Kelos adapter implementing runner/base.py and keeping
    C-07/C-08, and pinning plus checking the Calico manifest sha256
  - Note added on 004: GitHub App with per-run installation tokens for the publisher
  - New entry: 006 Operator guide (planned, depends on 002)
  - Open questions: Q3 decided for M1 (open for M2); Q4 decided for the pilot; Q5 partly
    decided
  - Cross-cutting notes: accepted per-run-key deviation; 002 workarounds for the
    spec-status asks

Specs affected: 002, 003, 004, 006
Open questions added/resolved: Q3 partly resolved (M1), Q4 resolved for M1, Q5 partly resolved

Previous revision 1.1.0 (2026-11-13):
Version change: 1.0.0 → 1.1.0
Bump rationale: MINOR — material scope change (Kelos read-only review moved from 001 to 003)
plus a status transition.

Changes this revision:
  - Status: 001 implemented → verified (rebuilt suite test/isolation/run.sh, 37/37 pass,
    evidence specs/001-m0-isolation-stub/evidence/20261113T014128Z, teardown verified)
  - Scope changed on 001: Kelos read-only review removed (no written review in repo)
  - Scope changed on 003: Kelos read-only review added as a required input to C-05

Specs affected: 001, 003
Open questions added/resolved: none

Notes: the original 2026-11-12 M0 run (32/33) was on another machine and is not
reproducible here; the rebuilt suite replaces it. Owner approved the move on 2026-11-13.

Previous revision 1.0.0 (2026-11-13): initial creation from docs/prd.md, entries 001–005,
decisions C-01 … C-06, open questions Q1–Q5.
-->

# specfoundry — Spec Roadmap

Living, non-binding map of the specs planned for specfoundry. It is **not a
commitment to order or scope** — it captures the spec-specific discussion,
decisions, technology choices, outcomes, and constraints surfaced during the
constitution and grilling phases so they are not lost before the spec that needs
them is written. Specs are scoped and clarified when they are actually started.
Foundations: the project [constitution](constitution.md) and the
[PRD](../../docs/prd.md).

Status legend (lifecycle): **undecided** · **needs-info** · **planned** ·
**specced** · **in-progress** · **implemented** · **verified** · **deferred** ·
**abandoned**.

---

## Vision & End States

- A human-approved Spec Kit roadmap is carried end to end: spec, plan, tasks,
  issues, isolated build, independent verification, draft PR, human merge.
- A roadmap entry becomes `verified` only when the evidence supports it, and that
  is what unblocks the next entry.
- No worker ever starts on a blocked item, and no worker ever holds an SCM
  credential (auditable from evidence records).
- From a fresh kind cluster to the first draft PR in one documented command
  sequence, with verified teardown.
- The same governed pilot runs on a laptop (kind) and in an enterprise environment
  (OpenShift restricted SCC, arbitrary UID, mirrored or air-gapped registries, proxies),
  with a small trust base and audit evidence a security team can review.

## Constraints & Decisions

- **C-01 — Scheduling inputs:** only the roadmap and `spec status` JSON (schema
  1.0) schedule work (PRD F1, constitution I).
- **C-02 — Gate fails closed:** blocked or unevaluable means no start. Until
  spec-status ships a per-item readiness exit code, read `summary.blocked_count`
  and `dependencies[].satisfied` (PRD F4, constitution II).
- **C-03 — Isolation and credentials:** one worker per pod and namespace; only the
  publisher holds SCM write credentials; grader is separate and not writable by
  workers (PRD S1–S6, constitution IV).
- **C-04 — Human merge only:** draft PRs only, one per item or task, never
  auto-merge (PRD F8, constitution V).
- **C-05 — ~~Wrap Kelos, don't fork~~ Kelos is a competitor, not a dependency:**
  ~~fork only if token split, bundle-out or stage retry cannot be worked around (PRD Q2
  default, constitution VI).~~ ~~**Amended 2026-11-13:** never fork. Kelos is adopted as a
  runner only if it meets the security bar (spec 003) through configuration alone.
  Otherwise it is not adopted: plain Jobs remains the runner and Kelos is used as a
  reference only (constitution II, IV, VI).~~ **Amended again 2026-11-13 (constitution
  1.1.0):** Kelos is a competitor and design reference. It is not adopted as a runner or
  forked. Its ideas may be borrowed, with Apache-2.0 licence and NOTICE kept for any copied
  code. Public comparisons stay factual. Plain Jobs is the runner now and OpenShift next.
- **C-06 — Enforcing CNI:** isolation tests run on Calico or Cilium; kind's
  default CNI does not enforce NetworkPolicy (PRD S4).
- **C-07 — Egress through per-role allowlist proxies:** worker and publisher egress
  goes only through a domain-allowlisting proxy for their role (model hosts for
  workers, the SCM for the publisher). NetworkPolicy decides which namespace may reach
  which proxy, because Calico OSS cannot match domain names. A worker can never reach
  the SCM proxy (PRD S2, S4; decided in 002, research R4).
- **C-08 — No storage shared between worker and grader:** nothing writable is shared
  between a worker and the grader. Worker results leave only as a framed stdout result
  collected by the orchestrator; grading material lives only in grader namespaces (PRD
  S5; decided in 002, research R5).
- **C-09 — Images built in GitHub, pinned by digest:** pilot images are built by GitHub
  Actions from verified upstream sources (never an operator's local binary), contain no
  secrets, are published to a public registry chosen per target platform, and are
  referenced only by digest. Digests live in `pilot/images.lock` (written only by the
  image workflow's PR and never a workflow trigger); build inputs live in `pilot/pins.env`.
  kind and local runs use GHCR (`ghcr.io/tosin2013/
  specfoundry-*`). The OpenShift target may use different images published to Quay (for
  example UBI-based variants for OpenShift's arbitrary-UID restricted SCC); 005 decides
  (PRD S6, constitution VII; decided 2026-11-13).
- **C-10 — Telemetry is operator observability only:** no third-party telemetry. First-party
  telemetry means metrics, traces and events (e.g. OpenTelemetry or Prometheus) sent only to
  a collector the operator runs. Nothing leaves the operator's environment unless they
  configure it; no phone-home usage telemetry; never secrets, prompts or source code
  (constitution Security & Isolation Constraints, 1.1.0).
- **C-11 — Two modes, one set of guarantees:** *laptop mode* (operator off): the CLI runs the
  orchestration itself against a local or remote cluster. *Cluster mode* (operator on): the
  CLI is a client of the specfoundry Kubernetes operator running in the cluster. The CLI
  detects which applies, or the operator chooses it in configuration. Both modes keep C-01
  to C-10, produce the same evidence and pass the same audit; cluster mode adds features and
  never relaxes a guarantee. The human decisions (mode confirmation, retries, merge, approving
  a status change) can be given on GitHub in both modes; terminal prompts stay available in
  laptop mode. The publisher's SCM credential may be a fine-grained token or a GitHub App
  (per-run installation tokens) in either mode; the same checks apply to both (decided
  2026-11-14).
- **C-12 — Humans plan, agents build:** humans write the roadmap, do the research, unblock
  items and release them. Agents may assist a human with planning, but they implement only
  items a human prepared and released:
  - a spec and a task list exist;
  - every dependency is verified, or is in the same release;
  - no open questions remain.

  The pilot never picks up an item by itself, never writes specs, and changes the roadmap only
  through the status change a human approves. Every unblock passes through a human merge and an
  approved status change. In PRD F6 ("who launches workers"), the human launches for now
  (decided 2026-11-15).
- **C-13 — One implementation language, Go:** the CLI and the operator are written in Go, in
  one Go module in this repository (decided 2026-11-15). The Python pilot
  (`pilot/specfoundry_pilot/`) is the reference implementation until the Go CLI reaches
  parity, then it is retired. Parity means all of:
  - the same commands and exit codes;
  - the same evidence format;
  - the same audit results on every committed evidence record (the golden data);
  - a passing fixture run.

  The in-cluster image scripts (grader, publisher, token check, GitHub writes) may stay in
  Python behind their contracts until there is a reason to port them.

## Planned Specs

### 001 — M0 stub isolation test on kind with Calico  [status: verified]

- **Description:** Stub worker isolation test suite on kind with Calico.
- **Outcome:** All isolation checks pass on kind with Calico, the evidence is
  recorded in the repo, and teardown is verified.
- **Scope (in):** kind cluster with Calico, namespace/NetworkPolicy/ResourceQuota/
  service-account checks for a stub worker.
- **Scope (out):** real harness images, grader, publisher, GitHub integration, Kelos
  review (moved to 003).
- **Depends on:** none
- **Governed by:** C-03, C-06
- **Addresses:** docs/prd.md §9 M0
- **Spec dir:** specs/001-m0-isolation-stub
- **Notes:** First run 2026-11-12 on another machine passed 32 of 33. Rebuilt
  here as test/isolation/run.sh; verified 2026-11-13 with 37 of 37 checks passing
  (including teardown), control probes for every blocked path, and a mutation test.
  Evidence: specs/001-m0-isolation-stub/evidence/20261113T014128Z.

### 002 — M1 pilot: two-task gate on kind with plain Jobs  [status: verified]

- **Description:** Codex's two-task pilot run with plain Kubernetes Jobs.
- **Outcome:** Task A's grade fails so task B stays blocked; A is retried and
  passes, then B runs; exactly one draft PR per task, opened by the publisher.
- **Scope (in):** gate stage using spec status, plain-Jobs runner, isolated worker
  namespace, separate grader, publisher Job, evidence record per run.
- **Scope (out):** Kelos, GitHub issue sync, in-cluster planning, OpenShift.
- **Depends on:** 001
- **Governed by:** C-01, C-02, C-03, C-04
- **Addresses:** docs/prd.md §9 M1, §10 first two metrics
- **Spec dir:** specs/002-m1-gated-pilot
- **Notes:** Proves the "no worker starts on a blocked item" metric. ~~Machine
  choice is open (Q3).~~ Ran on the Ubuntu host (31 GB, 8 CPUs), kind + Calico; one
  worker run per roadmap item (Q4 for M1). Verified 2026-11-13, run
  20261113T030655Z: A/1 failed grading (induced), B stayed refused, A/2 passed, draft
  PR #1 merged by the owner, reconcile approved, B passed, draft PR #2. Checks 32/32,
  audit 11/11, teardown 3/3, mutation test caught all. Evidence:
  specs/002-m1-gated-pilot/evidence/20261113T030655Z. Known limits are in the 002 plan.

### 003 — M2 hardened images and Kelos competitive review  [status: verified]

- **Description:** Build the pilot's images in GitHub so they run on laptops and in
  enterprise environments, and write a factual competitive review of Kelos.
- **Outcome:** GitHub Actions builds the worker (OpenCode from a checksum-verified upstream
  release), tools and egress images, publishes them as public digest-pinned images, and
  `pilot up` pulls them by digest; a plain-Jobs pilot run on those images passes its full
  audit. A written review compares Kelos v0.64.0 with specfoundry against the security bar,
  and lists the Kelos features enterprises expect, as candidate roadmap items.
- **Scope (in):** image workflow (registry as a parameter; GHCR now); arbitrary-UID-friendly,
  non-root, no-sudo images with no secrets; digest PRs to ~~`pilot/pins.env`~~ `pilot/images.lock`; pull-by-digest
  `pilot up` with `--local-build` fallback; Calico manifest hash pinned and checked;
  competitive review (security bar SB-1–SB-8 comparison, enterprise gap list); optional
  Kelos-compatible entrypoint in the worker image so Kelos users can run it.
- **Scope (out):** Kelos runner adapter or install; forking Kelos; other harness images;
  OpenShift (005); operator observability (see Q6).
- **Depends on:** 002
- **Governed by:** C-03, C-05, C-07, C-08, C-09, C-10
- **Addresses:** docs/prd.md §9 M2 (re-scoped), S6, N1, N3
- **Spec dir:** specs/003-m2-kelos-backend
- **Notes:** Renamed from "M2 Kelos backend" on 2026-11-13. Revisit per-run provider keys
  here (see Cross-Cutting Notes): revisited 2026-11-13, no change (Kelos uses a static
  key Secret too); carried to 005. Verified 2026-11-13: validation run 20261113T180203Z on
  the GHCR images (checks 32/32, audit 11/11, teardown 3/3, `image_source: pinned`);
  images public with provenance and SBOM; no rebuild loop; arbitrary-UID and mirror checks
  pass. Evidence: specs/003-m2-kelos-backend/evidence/. Superseded entry text (roadmap 1.3.0), kept for history:
  ~~(former description) ~~Run the M1 pilot through Kelos behind an adapter.~~ Assess Kelos~~
  ~~against the security bar and adopt it as the worker runner only if it meets the bar by~~
  ~~configuration; build the pilot's images in GitHub.~~
  ~~(former outcome) ~~Same pilot result as 002 with Kelos as the runner: namespace,~~
  ~~policies, thin image, gate stage, publisher and separate grader all preserved.~~ A~~
  ~~written review scores Kelos against the security bar and records `wrap`,~~
  ~~`wrap-with-workarounds` or `do-not-adopt`. If adopted, the 002 pilot result is~~
  ~~reproduced with Kelos running the workers (gate, publisher and separate grader~~
  ~~preserved). If not, plain Jobs stays the runner. Either way, the pilot's images are~~
  ~~built by GitHub Actions and pulled by digest (C-09).~~
  ~~(former scope (in)) written read-only Kelos review (security model, token handling,~~
  ~~bundle-out, stage retry) as the input to C-05; Kelos adapter; pinned Kelos version~~
  ~~with telemetry off; thin OpenCode reference image pinned by digest; the Kelos~~
  ~~adapter implements the 002 runner interface (`pilot/specfoundry_pilot/runner/base.py`)~~
  ~~and keeps C-07 and C-08; pin and check the Calico manifest sha256~~
  ~~(`CALICO_SHA256` in `pilot/pins.env`, checked by `pilot up`). Added 2026-11-13: the~~
  ~~security bar (no SCM credential in workers, no sudo, no automatic retries, telemetry off,~~
  ~~digest pins, effective token off, C-07/C-08 kept, controller privileges reviewed and~~
  ~~accepted); the Kelos adapter only if the review adopts; GitHub Actions building~~
  ~~`pilot-worker` (OpenCode from a verified upstream release), `pilot-tools` and~~
  ~~`pilot-egress` to public GHCR, with `pilot up` pulling by digest.~~
  ~~(former scope (out)) other harness images, OpenShift, **forking Kelos** (C-05).~~
  ~~(former dependency) 002~~
  ~~(former governed by) C-03, C-05, C-07, C-08, C-09~~
  ~~(former notes) Revisit per-run provider keys here (see Cross-Cutting Notes).~~
  ~~(former addresses) docs/prd.md §9 M2, F10~~

### 004 — M3 GitHub loop  [status: planned]

- **Description:** Tie roadmap entries, issues and PRs together on GitHub.
- **Outcome:** taskstoissues creates one issue per task, labels mirror spec status
  state, PRs carry Closes #N, and reconcile after merge shows no drift.
- **Scope (in):** issue creation, label sync, PR linking, post-merge reconcile. Added
  2026-11-14: GitHub approvals for the human decisions (mode, retry, approving a status
  change; merge is already on GitHub), usable in laptop and cluster mode (C-11); GitHub App
  as a second publisher credential beside the fine-grained token, with setup docs and checks.
- **Scope (out):** Bitbucket and other SCMs.
- **Depends on:** 003
- **Governed by:** C-01, C-04, C-11
- **Addresses:** docs/prd.md §9 M3, F3, F11
- **Notes:** ~~Replace the publisher's fine-grained token with a GitHub App that mints a
  1-hour installation token per run, scoped to the target repo, so PRs show
  `<app>[bot]` rather than the owner's account.~~ **Amended 2026-11-14: support both,
  chosen per install.** The fine-grained token stays the default (simplest; PRs show the
  owner's account). The GitHub App mints a 1-hour installation token per run, requested for
  the target repo only with contents and pull-request write, so PRs show `<app>[bot]`; it is
  recommended for teams and cluster mode, not required. In laptop mode the App's private key
  stays on the host and only the short-lived token enters the cluster. Both pass the same
  write probe (fixture writable, canary not) and P09. In 002 the token owner is the operator,
  which weakens PRD §10 metric 2 as seen from GitHub (the evidence audit still shows the
  publisher Job opened every PR). Learned in 003: new repositories block GitHub Actions
  from opening PRs until *Allow GitHub Actions to create and approve pull requests* is
  enabled; on a public repository the `permissions` field of `GET /repos` reports the
  account's rights, not a fine-grained token's (use a write probe instead). GitHub approvals
  reuse 012's pending-step approval and channel record, adding a `github` channel
  (2026-11-14). **2026-11-15:** implemented in the Python pilot (PR #21), which is now the
  reference implementation (C-13). Its owner validation was stopped (owner decision): 016 ports
  the behaviour to Go, and 004's validation (quickstart §2–§5) runs against the Go pilot, which
  also produces the golden evidence for the `github` channel. Status stays `planned` until then. Issue #25 (watching needs a terminal) is
  solved in the Go design: a laptop daemon in 016, and the operator in 011.
- **Spec dir:** specs/004-github-loop/

### 005 — M4 OpenShift target (laptop mode)  [status: planned]

- **Description:** ~~Optional in-cluster planning Job, and OpenShift as a target
  platform.~~ OpenShift as a target platform, driven by the laptop CLI (C-11 laptop mode).
  *(Amended 2026-11-15: in-cluster planning moved to 011.)*
- **Outcome:** ~~Planning location is configurable per project (local or
  in-cluster) with the same handoff, and the pilot runs on OpenShift.~~ The pilot runs on a
  remote OpenShift cluster from the laptop CLI and passes the same full audit as on kind.
- **Scope (in):** ~~planning Job, per-project planning setting, OpenShift runner
  validation.~~ OpenShift runner validation; a configurable kube context; preflight checks
  instead of creating a cluster; namespaces the user may not be able to create; image
  variants or mirrors for OpenShift (Quay).
- **Scope (out):** hosted SaaS or UI.
- **Depends on:** 003
- **Addresses:** docs/prd.md §9 M4, F2, N1
- **Notes:** In-cluster planning is optional per the PRD; OpenShift is the
  required part. Decide whether OpenShift needs its own image variants (for example UBI
  base, arbitrary-UID support) published to Quay, or the C-09 images mirrored there; the
  003 image workflow takes the registry as a parameter (003 showed the images already run
  as an arbitrary UID and mirror by digest). Revisit per-run provider keys here (carried
  from 003; enterprise secret managers may make them practical). Candidate: an explicit
  `podFailurePolicy` on worker Jobs, so infrastructure disruptions are recorded distinctly
  from worker failures (from the 003 Kelos review). Added 2026-11-14: first prove the laptop
  CLI against a remote OpenShift cluster (C-11 laptop mode): a configurable kube context
  instead of the hard-coded kind context, preflight checks instead of creating a cluster,
  and enterprise namespace permissions (users often cannot create namespaces).
  Orchestration inside the cluster is 011.

### 006 — Operator guide: repeat the pilot from zero  [status: verified]

- **Description:** End-user documentation so someone new can repeat the governed
  pilot on their own machine.
- **Outcome:** Someone new, starting from a fresh machine and using only the guide,
  stands up the stack, runs the pilot through both human gates, tears down, and gets a
  passing audit.
- **Scope (in):** prerequisites and machine sizing; provider setup (OpenCode config
  directory, `{file:}` key reference); publisher token creation and
  `pilot publisher-token`; the command sequence with every prompt; the merge pause;
  reading the evidence and audit; teardown; troubleshooting from 002's real failures;
  ~~where 003 (Kelos) and 004 (GitHub App) will change the steps~~ where 004 (GitHub App,
  GitHub approvals) and later items will change the steps (003 delivered hardened images, not
  Kelos; corrected 2026-11-14).
- **Scope (out):** hosted docs site or UI; naming specific private model providers.
- **Depends on:** 002
- **Governed by:** C-07
- **Addresses:** constitution VII ("one documented command sequence"), docs/prd.md N2, N3
- **Notes:** Requested by the owner 2026-11-13. ~~Use run 20261113T030655Z as the
  worked example.~~ The guide's worked example is 003's validation run 20261113T180203Z; the
  fresh-machine validation run is 20261114T160532Z (audit 11/11, teardown 3/3, 0 outside
  lookups, ~44 min hands-on), and run 20261114T213046Z repeated it agent-driven with
  harness approvals (012). Debrief: specs/006-operator-guide/roadmap-reviews/debrief-20261114.md
  (corrected 2026-11-14).
- **Spec dir:** specs/006-operator-guide

### 007 — Operator observability  [status: planned]

- **Description:** Metrics, traces and events that operators can use to watch and audit
  pilot runs in their own environment.
- **Outcome:** Gate decisions, attempts, spend and run duration are emitted as metrics,
  traces and events (OpenTelemetry/Prometheus) to an operator-run collector. Nothing leaves
  the environment unless configured, and evidence and telemetry carry no secrets, prompts
  or code.
- **Scope (in):** instrumentation of the gate, attempts, grading, publishing and spend; an
  export to an operator-configured collector; documentation.
- **Scope (out):** phone-home or vendor-hosted telemetry; dashboards as a product.
- **Depends on:** 002
- **Governed by:** C-10
- **Addresses:** constitution Security & Isolation Constraints (telemetry); docs/prd.md N4
- **Notes:** Proposed in the 003 Kelos review; closes Q6 (owner decision 2026-11-13).

### 008 — More harnesses (Claude Code, Codex)  [status: planned]

- **Description:** Additional worker harnesses behind the same worker contract.
- **Outcome:** Each harness ships as a thin, digest-pinned, no-sudo worker image built by
  the `pilot-images` workflow, behind the same worker contract; a pilot run with each
  passes its full audit.
- **Scope (in):** Claude Code and Codex worker images; per-harness configuration docs;
  spend accounting from each harness's usage output.
- **Scope (out):** other harnesses; harness-specific features outside the worker contract.
- **Depends on:** 003
- **Governed by:** C-03, C-07, C-09
- **Addresses:** docs/prd.md §2 (harness-agnostic), N2
- **Notes:** Proposed in the 003 Kelos review (owner decision 2026-11-13). Under C-12 an
  event can only *propose*; a human still releases the item (015).

### 009 — Period spend budget at the gate  [status: planned]

- **Description:** A spend limit per day or per project, enforced before work starts.
- **Outcome:** A daily or project spend limit is checked at the gate before any worker
  starts, fails closed when exceeded or unevaluable, and is recorded in the evidence; it
  adds to the per-attempt cap rather than replacing it.
- **Scope (in):** budget configuration; spend accounting across runs; gate integration;
  evidence fields.
- **Scope (out):** provider billing integrations.
- **Depends on:** 002
- **Governed by:** C-01, C-02
- **Addresses:** docs/prd.md F7
- **Notes:** Idea borrowed from Kelos's admission-time TaskBudget (attributed in the 003
  review; owner decision 2026-11-13).

### 010 — Event-proposed runs  [status: planned]

- **Description:** Let authenticated external events propose work without bypassing
  governance.
- **Outcome:** Authenticated events (for example signed webhooks or an issue label) may
  *propose* a roadmap item for a run; the spec-status gate and a human mode confirmation
  still decide.
- **Scope (in):** event authentication; proposal recording; handoff to the gate and the
  mode confirmation.
- **Scope (out):** automatic runs without human confirmation; unauthenticated endpoints.
- **Depends on:** 004
- **Governed by:** C-01, C-02, C-04
- **Addresses:** docs/prd.md F6
- **Notes:** Proposed in the 003 Kelos review (owner decision 2026-11-13).

### 011 — Cluster mode: specfoundry Kubernetes operator  [status: planned]

- **Description:** An in-cluster operator runs governed runs, with the CLI as its client
  (operator on); the CLI still works standalone (operator off).
- **Outcome:** The operator runs governed runs inside the cluster under its own scoped
  service account, with human approvals on GitHub. The CLI works with it (operator on) or
  without it (operator off). A cluster-mode run passes the same full audit as a laptop run.
- **Scope (in):** the operator and a run resource it acts on; the gate as a container in the
  cluster; run state and evidence stored in the cluster; model keys and the publisher
  credential (token or GitHub App key) from cluster secrets or an enterprise secret store,
  with the operator minting per-run App tokens; parallel runs; permissions through
  Kubernetes RBAC; the CLI's operator on/off detection; GitHub approvals (from 004). Added
  2026-11-15: the optional in-cluster planning Job and per-project planning setting (moved
  from 005).
- **Scope (out):** a hosted SaaS or UI; multi-cluster; any weakening of a guarantee.
- **Depends on:** 004, 005, 016
- **Governed by:** C-01, C-02, C-03, C-04, C-07, C-08, C-10, C-11
- **Addresses:** docs/prd.md F2, N1, §9 M4
- **Notes:** Owner decision 2026-11-14. 007 (observability), 009 (period budgets) and 010
  (event-proposed runs) gain the most in cluster mode. Decide here how runs get namespaces
  when users cannot create them. Depends on spec-status shipping a CLI container image for
  the in-cluster gate (PRD §8). The GitHub App is the recommended publisher credential here.
  **Owner decision 2026-11-15:** develop and test on kind first, then make the same full
  audit on OpenShift the release check. 005 (OpenShift in laptop mode) comes first, so the
  operator is designed around OpenShift's constraints (SCC, arbitrary UIDs, namespace
  permissions, egress). The operator lives in `operator/` in this repo until Q9's split
  trigger is met. It runs released waves (015) in the cluster, reusing the same wave scheduler.

### 012 — Working with an agent: pilot skill and harness approvals  [status: verified]

- **Description:** Let a human and an LLM agent run the pilot together: the agent runs the
  commands and explains each pause; the human makes every decision.
- **Outcome:** With no terminal, every decision pauses instead of being assumed. The human
  approves the one pending step through `pilot approve`, which the agent harness always asks
  the human about. Each approval is bound to that step and used once. The audit (AU09, AU10)
  records each approval's channel (`tty` or `harness`), and a project `pilot` skill teaches
  agents the commands, exit codes and the rule never to approve for the human.
- **Scope (in):** `pilot approve` (mode, retry, reconcile; never merge); paused-for-mode
  state; channel recording in events and the audit; harness ask rules ~~(`.claude/settings.json`,
  `opencode.json`)~~ (`.claude/settings.json`, committed; for OpenCode, a rule each operator
  adds once to `~/.config/opencode/opencode.json`, documented in the project; owner decision
  2026-11-15); the `pilot` skill and `AGENTS.md`; troubleshooting and README updates.
- **Scope (out):** a `--yes` flag or any self-approval by an agent; approving merges outside
  GitHub; GitHub approvals themselves (004).
- **Governed by:** C-04, C-11
- **Notes:** Owner decision 2026-11-14 (006 finding F4). A `--yes` flag that records a
  confirmation was rejected: AU09 and AU10 attest a human decision. The `harness` channel is
  only as strong as the harness permission rule and trusts the agent not to forge the
  approval file (the harness denies edits to `approvals/`); GitHub approvals (004) are the
  stronger channel. First cut implemented early to unblock the 006 validation; the spec
  records the audit-contract change and validates it with an agent-driven run.
  **Validation 2026-11-15:** the harness channel also depends on the harness's permission
  *mode*: Claude Code's auto mode (and any bypass or auto-accept mode) runs `approve` with no
  prompt, so approvals through an agent need normal permission mode. The `approvals/` deny
  rule exists in Claude Code only; in OpenCode the agent's skill rules are what stop a direct
  edit (see Q8).
- **Spec dir:** specs/012-agent-approvals/

### 013 — Installable pilot release  [status: planned]

- **Description:** Ship the pilot as an installable, versioned release, so operators can
  install it and run it against any project without cloning this repository.
- **Outcome:** ~~_to be defined_.~~ A versioned release of the Go `pilot` binary for Linux and
  macOS (amd64 and arm64), plus Windows where the pilot supports it, with published SHA-256
  checksums. It installs without the source tree and keeps every guarantee and the full audit.
- **Scope (in):** ~~_to be defined_. Candidates: the release form (single binary, zipapp, or a
  Python package); …~~ A GitHub Actions release workflow (checksums, like the pilot images);
  how the CLI finds the target project and its config; the command name used in harness rules
  (Q8); install docs.
- **Scope (out):** OS package managers; signing beyond checksums (revisit with Q10).
- **Depends on:** 016
- **Notes:** Raised by the owner 2026-11-15 during the 012 debrief. Tied to Q7 (product name
  and CLI surface) and Q8 (multi-project use; harness rules today match the `pilot/pilot`
  path). Decide the form and name before speccing. **2026-11-15:** form decided by C-13 (one Go
  binary); the name is still Q7. Dependency changed from 012 to 016 (2026-11-15).

### 014 — Any project, any item  [status: planned]

- **Description:** Run one chosen, prepared roadmap item of a real repository, not just the
  fixture's two items.
- **Outcome:** `pilot run --item NNN` against a configured repository runs that item under
  every existing guarantee: gate, isolation, grading, draft PR, human merge, status change and
  the full audit. It refuses an item that is not prepared (C-12).
- **Scope (in):**
  - the item to run comes from the roadmap and `spec status`, replacing the hard-coded A/B
    mapping;
  - the worker prompt and the grading come from the item's spec, tasks and the repo's own tests;
  - a readiness check: a spec and tasks exist, dependencies are verified, no open questions;
  - per-item evidence, and an audit that no longer assumes exactly items A and B;
  - per-repo toolchains for the worker and the grader (an image per language, or declared in
    the repo's configuration), still digest-pinned and hardened (C-08, C-09);
  - an egress allowlist for the repo's package registries, so dependencies can be installed,
    declared per repo and shown at the mode approval;
  - hidden acceptance tests written by a human and kept out of the worker's view, with the
    repo's own test suite run as a visible check.
- **Scope (out):** parallel runs (015); writing specs or plans; choosing items automatically.
- **Depends on:** 004, 016 (and 005 for the OpenShift EAP validation)
- **Governed by:** C-01, C-02, C-04, C-09, C-12
- **Addresses:** docs/prd.md §4 loop, F1, F8, F9
- **Notes:** Owner decision 2026-11-15. The fixture stays the regression test. Pairs with 013
  (installable CLI). Grading a real repo needs the item's acceptance criteria to be runnable;
  the spec decides how that is expressed. 014 targets **brownfield** repos (existing code and
  tests); greenfield bootstrap is Q11 (owner decision 2026-11-15).

  ~~**Validation (owner decision 2026-11-15):** a real **Quarkus** (Java) brownfield repository,~~
  ~~chosen when 014 is specified, with a roadmap of a few items where one depends on another.~~
  **Validation (owner decision 2026-11-16, replaces the Quarkus validation):** a legacy Java EE
  upgrade on **OpenShift**, the enterprise brownfield case:
  - the starting point is a public application on the `javax` namespace with real tests, on an
    older JBoss EAP or WildFly release (candidates: the Java EE 7 Petstore, Cargo Tracker's
    javaee7 branch; chosen when 014 is specified), or a small app built for the purpose if
    neither fits;
  - the target is a current release (EAP 8 or Quarkus, decided when 014 is specified);
  - the owner prepares the repository (constitution, roadmap, specs, tasks, and hidden
    JUnit/RestAssured acceptance tests that pin the application's behaviour before the
    upgrade) and splits the upgrade into a few dependent items (for example: build and
    dependencies; `javax` to `jakarta`; server configuration; tests and cleanup);
  - Red Hat's migration tools are used inside the process: the owner runs the Migration
    Toolkit for Applications analysis when planning, and the worker image carries its CLI
    (`kantra`, rule sets baked in) and may use Kai against the in-cluster model;
  - the grader never trusts the migration tools' own reports; each item passes only if the
    hidden tests and the app's own suite show behaviour is unchanged;
  - it runs on OpenShift with the model served in the cluster, so worker model traffic stays
    in the cluster; worker and grader images are Red Hat UBI-based (C-09 already allows
    Quay/UBI images for OpenShift).

  This exercises a JDK/Maven toolchain, Maven egress (plus the Red Hat Maven repository if EAP
  artifacts are used), and OpenShift's restricted SCC. The story is that specfoundry governs the
  upgrade (plan, isolation, independent grading, draft PR, audit) and the Red Hat tools do the
  analysis inside it. The EAP validation therefore also needs 005 (OpenShift target) and an
  in-cluster model endpoint. The fixture stays the fast, deterministic regression test (its A/1
  induction is the only reliable retry exercise).

  **Toolchain images (owner direction 2026-11-16):** on OpenShift, worker and grader toolchain
  images are built on Red Hat UBI images (JDK, Node.js). Elsewhere (kind, minikube, plain
  Kubernetes), toolchain images are built on demand, one per language a project needs, still by
  GitHub Actions and pinned by digest (C-09).
  On kind, minikube and plain Kubernetes, a user who needs a language toolchain that has no
  image yet asks for one by opening a GitHub issue on the specfoundry repository (the issue
  form and the image build are defined when 014 is specified).

### 015 — Released waves  [status: planned]

- **Description:** A human releases a set of prepared items, and the pilot builds them in
  parallel in dependency order.
- **Outcome:** One human release starts every released item whose dependencies are verified,
  each in its own isolated unit with its own draft PR. Released items that are still blocked
  wait, and start only when their dependencies become verified, which always passes through a
  human merge and an approved status change. Each item passes the full audit, and the wave has
  a summary.
- **Scope (in):**
  - a release command (and later a GitHub release action), refused unless every item is
    prepared and every dependency is verified or released in the same set (C-12);
  - one approval for the wave's mode and total budget, with per-item retries still approved
    one by one;
  - a concurrency limit, and running items with overlapping scopes one after the other;
  - a queue for blocked items, released as their dependencies become verified;
  - per-item evidence and a wave summary.
- **Scope (out):**
  - starting items nobody released;
  - auto-merging, and unblocking without a human merge;
  - releasing an item that depends on something not yet specced.
- **Depends on:** 014, 009
- **Governed by:** C-01, C-02, C-04, C-07, C-12
- **Addresses:** docs/prd.md §4 step 8, F6, F7
- **Notes:** Owner decisions 2026-11-15:
  - humans plan, research and unblock, and agents build released items;
  - blocked items in a release follow the dependency order (the pilot's A→B gate, scaled up).

  Human review is the expected bottleneck: every PR still needs a human merge. Laptop waves are
  small (kind capacity); large waves run in cluster mode (011).

  **Scaling goal (owner direction 2026-11-16):** a released roadmap scales its jobs the same way
  on kind, OpenShift and plain Kubernetes. Only capacity differs: the concurrency limit is set
  per cluster, and the guarantees, evidence and audit are the same everywhere (C-11).

  **Parallel tasks inside an item (owner direction 2026-11-16, proposed):** blockers still block,
  and that is fine. Where tasks have no blockers between them (Spec Kit's `[P]` marker in
  tasks.md, checked against spec-status), they may run at the same time, each in its own
  isolated unit; dependent tasks keep their order. The pilot owns the schedule: the human sees
  one plan at release (what runs together, what waits, and why), and each worker sees only its
  own task. See Q12.

### 016 — Go pilot at parity  [status: planned]

- **Description:** Rewrite the pilot CLI in Go (C-13), reproducing the reference Python pilot's
  behaviour exactly, before any new feature is built.
- **Outcome:** `pilot` is one Go binary. It runs the fixture end to end with the audit at
  11/11. Its audit gives the same results as the reference on every committed evidence record.
  Then the Python pilot is retired.
- **Scope (in):**
  - every current command: `up`, `down`, `run`, `resume`, `approve`, `status`, `audit`,
    `gate`, `publisher-token`, `import-opencode`, `reset-fixture`, `github sync|drift|watch`,
    `app-key`, `app-token`;
  - the same evidence format, exit codes and approval channels;
  - the gate through spec-status as a Go library (spec-status needs a public package; its
    logic is in `internal/` today);
  - the Kubernetes client where it simplifies (client-go), with a minimal dependency surface;
  - the watch design for issue #25: a laptop daemon mode started by the run, built so the
    operator (011) can reuse it;
  - conformance vectors (approval rules, status-PR check) and a golden-evidence audit suite;
  - 004's owner validation (its quickstart §2–§5), run against the Go pilot;
  - switching `pilot/pilot` to the Go binary, and retiring the Python pilot at parity.
- **Scope (out):** new features (014, 015, 011 follow); porting the in-cluster image scripts;
  release packaging (013).
- **Depends on:** 012
- **Governed by:** C-01, C-02, C-03, C-04, C-07, C-08, C-11, C-12, C-13
- **Addresses:** docs/prd.md N3, F9, F10
- **Notes:** Owner decision 2026-11-15, prompted by issue #25 and by the operator being Go:
  - one codebase for the CLI and the operator;
  - spec-status used as a library;
  - static binaries across operating systems;
  - the Kubernetes ecosystem is Go.

  The committed evidence of 002, 003, 006 and 012 is the golden data (004 was never validated in
  Python; its validation runs on the Go pilot).
  Port order: audit and gate first (pure, proven on golden data), then run and resume on the
  fixture, then the GitHub loop, then setup and teardown.

  Stderr diagnostics (`-v` step timings, `-vv` pod phases and events, `-vvv` live agent
  activity) are allowed as a debugging aid outside parity: they never change stdout, exit codes
  or evidence, and they are redacted (owner decision 2026-11-15).

### 017 — Greenfield bootstrap  [status: planned]

- **Description:** Start a project from an empty repository: the roadmap's first item is a
  scaffold, and every later item grows the project like a brownfield one.
- **Outcome:** On a new, empty **Node.js** repository whose constitution and roadmap a human
  wrote, the pilot runs item 001 (a human-specified scaffold: layout, package manifest, test
  runner, CI), grades it against acceptance tests the human wrote, and publishes a draft PR.
  After the human merges it and approves `verified`, the next items run as in 014.
- **Scope (in):**
  - grading an item when the repository has no tests or build of its own yet (only the hidden
    acceptance tests, and the scaffold's own tests once it exists);
  - a Node.js toolchain for the worker and grader, and npm registry egress, through 014's
    per-repo mechanisms;
  - guidance for writing a scaffold item's spec and acceptance tests;
  - the validation on a new Node.js repository with at least the scaffold and one dependent
    item.
- **Scope (out):** generating the constitution, roadmap or specs (humans plan, C-12); project
  templates as a product; languages beyond the ones 014 and 017 validate.
- **Depends on:** 014
- **Governed by:** C-01, C-04, C-07, C-08, C-12
- **Addresses:** docs/prd.md §4 loop, F1
- **Notes:** Owner decision 2026-11-15; resolves Q11. The Node.js repository is chosen when 017
  is specified. Planning still comes first: an empty repository gets its constitution, roadmap,
  and the scaffold's spec, tasks and acceptance tests from a human before any agent runs.
  **Runs anywhere (owner direction 2026-11-16):** the greenfield validation must pass on every
  cluster profile (kind, minikube, plain Kubernetes and OpenShift), using an on-demand Node.js
  image off OpenShift and a UBI Node.js image on OpenShift.

## Open Questions

- Q1 — Final project name and repo location (new repo vs. inside the spec-status
  org). Interim home: GitHub repo tosin2013/specfoundry (public since 2026-11-13, Apache-2.0,
  with CONTRIBUTING and a Code of Conduct).
- Q2 — Kelos as wrapped dependency vs. fork (default: wrap, see C-05). **Resolved
  2026-11-13:** never fork; wrap only if the security bar is met by configuration,
  otherwise do not adopt (C-05 amended).
- Q3 — Which machine runs M1 and M2 (Ubuntu box, Mac, another lab machine). **M1 decided
  2026-11-13:** the Ubuntu host (31 GB, 8 CPUs). Still open for M2.
- Q4 — Granularity: one worker run per roadmap entry, or per task or issue. **Decided
  for M1:** one worker run per roadmap item. Revisit in 004, where issues are per task.
- Q5 — Worker model provider and spend cap defaults. **Partly decided 2026-11-13:**
  harness OpenCode 2.2.7; provider is operator configuration (not named in docs); caps
  default to $1 per attempt with a 2M-token fallback when the provider reports no cost.
  Still open: provider choice for shared use, and USD accounting (declaring model
  prices).
- Q6 — Which spec delivers operator observability (C-10): metrics, traces and events for
  gate decisions, attempts, spend and run duration to an operator-run collector? Candidates:
  a new entry after 003, or folded into 004/005. **Resolved 2026-11-13:** entry 007.
- Q7 — Product name and CLI interface: `pilot` is a prototype name, and the CLI becomes
  the interface for humans and LLM agents in both modes (C-11). Decide the name and the
  stable command surface before 011 (raised 2026-11-14).
- Q8 — Using the pilot across projects and harnesses (raised 2026-11-15, 012 debrief).
  - Do harness ask rules live per user, or does a `pilot` command install them?
  - Today's rules match the `pilot/pilot` path; an installed CLI (013) changes the pattern.
  - How are agents that run in a real terminal handled (Warp, for example)? Such an agent
    could answer a `[y/N]` prompt that the audit records as `tty`.
  - Do harnesses other than Claude Code get a deny rule for `approvals/`?

  Relevant to 008, 011 and 013.
- Q9 — When does the operator move to its own repo (raised 2026-11-15)? Default: it stays
  in `operator/` here. Split when it has its own toolchain and release cadence (for example
  a Go operator with an OLM bundle) **and** the evidence and audit contract is versioned
  and language-neutral, so the CLI and the operator can each check it independently. Name
  per Q7. **Partly resolved 2026-11-15:** the operator is written in Go, in the same Go module
  as the CLI (C-13). They share code, so the remaining question is only when `operator/`
  needs its own release cadence or OLM bundle repository.
- Q10 — Distribution channels and timing (raised 2026-11-15). Proposed, not yet decided:
  - CLI on GitHub Releases with checksums (013); images on GHCR or Quay by digest (C-09, 005);
  - after 011 is verified: a **Helm chart**, hosted as an OCI artifact on GHCR and listed on
    **Artifact Hub** (artifacthub.io, an index that links to where charts are hosted), and the
    operator as an OLM bundle on **OperatorHub.io** (submitted by PR to the community
    operators repository). Artifact Hub can index the operator too;
  - a certified catalog listing later, on enterprise demand (needs certified base images);
  - everything stays mirrorable for companies with internal registries: digest-only
    references, checksums, a configurable registry prefix, and no runtime internet fetch
    during install.
- Q11 — Greenfield bootstrap (raised 2026-11-15). An empty repository has no build, tests or
  code to grade against. Options: item 001 is a human-specified scaffold (project layout, test
  harness, CI) whose acceptance tests the human writes, graded like any other item; or
  greenfield stays out of scope until 014 is verified on brownfield repos. Planning still comes
  first either way (constitution, roadmap, spec and tasks, C-12). **Resolved 2026-11-15:**
  entry 017, after 014; item 001 of a new project is a human-specified scaffold, validated on
  a Node.js repository.
- Q12 — Parallel tasks inside an item (raised 2026-11-16). Run unblocked `[P]` tasks of one
  item at the same time? Open: whether each task gets its own draft PR or they combine into the
  item's PR; how overlapping file scopes are detected; how much the human and the worker need to
  see of the schedule (proposed: the human sees the plan at release, the worker sees only its
  task). Relates to Q4 (granularity), 004 (issues per task) and 015.
- Q13 — How planning scales (raised 2026-11-16). Under C-12 the limit on throughput is how fast
  humans prepare items (spec, tasks, hidden tests), not how many workers run. Open: whether
  agents may draft specs, tasks or tests for a human to approve, and what tooling helps (for
  example migration analysis reports in brownfield upgrades).

## Cross-Cutting Notes

- **Delivery order (owner direction 2026-11-16):** laptop first, then OpenShift, in stages, so
  someone on a laptop can trust that the whole loop works end to end before the enterprise case.
  Dependencies are unchanged; this is the order of work among items that are ready.
  1. 016 Go pilot at parity (its T047 owner run also closes 004's superseded validation).
  2. 014 Any project, on kind with the fixture and a real repository; the EAP validation waits.
  3. 017 Greenfield bootstrap, end to end on a laptop (kind or minikube).
  4. 013 Installable pilot release, if needed at that point.
  5. 005 OpenShift target (laptop mode).
  6. 014's EAP validation on OpenShift with an in-cluster model.
  7. 009 and 015 (released waves), then 007, 008, 010 and 011.
- spec-status asks drafted in PRD §8 (report-only skills, per-item readiness exit
  code, CLI container image) gate 002 and 003 in practice; track them as upstream
  issues. **002 workarounds in place:** the gate runs on the host (no CLI image); the
  grader uses `spec status --json` (no report-only `verify-readiness`); reconcile is a
  deterministic pilot step (the skill needs an agent). See the 002 plan, Known limits.
- ~~**PRD drift (2026-11-13):** docs/prd.md F10 ("First backend is Kelos") and §9 M2 still
  describe Kelos as a backend. The owner has been sent replacement text; until the PRD is
  edited, the roadmap and constitution 1.1.0 take precedence.~~ Resolved 2026-11-13: the
  PRD was updated (F10, §6, §7, §9 M2, Q2, N1, new N4).
- **Accepted deviation (002):** constitution IV asks for per-run worker keys with spend
  caps. 002 copies a long-lived provider key into a per-attempt Secret (deleted with the
  attempt) and enforces the cap with the pilot watchdog and deadline. Accepted by the
  owner 2026-11-13; revisit in 003. Revisited in 003 (no better mechanism found, including
  in Kelos); carried to 005.
- **Cluster mode (2026-11-14):** in cluster mode, 007 (observability), 009 (period budgets)
  and 010 (event proposals) run in the operator (011); in laptop mode they run in the CLI,
  where they apply at all. Both modes keep C-11's guarantees.
- **CLI/operator split (2026-11-15, issue #25):** long-running, event-driven or
  credential-holding work belongs to the operator in cluster mode; human-started actions stay
  in the CLI.
  - **Moves to the operator:**
    - the run state machine;
    - watching for approvals and merges, and resuming;
    - decision posts and approval reading;
    - the status-change PR;
    - GitHub sync and drift on events;
    - App token minting and `scm-token` refresh;
    - publisher Jobs;
    - the gate container;
    - the P01–P10 checks;
    - evidence storage;
    - the wave scheduler (015), budgets (009), event intake (010) and telemetry (007).
  - **Stays in the CLI:**
    - `run` / `release`;
    - `approve` (harness channel);
    - `status` and `audit`;
    - on-demand `github sync|drift`;
    - `watch` (laptop mode);
    - credential setup and `reset-fixture`;
    - laptop `up` and `down`.

---

**Version**: 1.13.0 | **Ratified**: 2026-11-13 | **Last Amended**: 2026-11-16
