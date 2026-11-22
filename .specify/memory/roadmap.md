<!--
SYNC IMPACT REPORT
==================
Version change: 1.29.1 → 1.29.2 (this revision)
Bump rationale: PATCH — a status transition (025 specced → verified) and notes. No scope,
dependency or constraint changes.
Changes this revision (owner decision 2026-11-22, Tosin Akinosho):
  - 025 Integration tests in CI: verified, after the debrief (PROCEED WITH UPDATES, no
    must-address) and its follow-ups (F1 fixed, F2 asserted, F3 kept as policy). Notes record
    what was delivered beyond the outcome's wording, SC-001/002/004 and the pilot fixes the
    scenarios found.
Specs affected: 025
Open questions added/resolved: none

Prior: Version change: 1.29.0 → 1.29.1
Bump rationale: PATCH — a note update. Nothing changes in scope, status or dependencies.
Changes this revision (2026-11-22):
  - 026 note: spec-status 0.4.0 shipped per-phase task counts (spec-status#66); PR #154 pins it in
    `pilot/pins.env`. 026's external dependency is met, and its plan and tasks are written.
Specs affected: 026
Open questions added/resolved: none

Prior: Version change: 1.28.0 → 1.29.0
Bump rationale: MINOR — a new planned item (026) and a new dependency for 013. Nothing verified
is invalidated.
Changes this revision (owner decisions 2026-11-21, Tosin Akinosho; issue #142):
  - 026 (new, planned): slice runs. A run builds one phase of an item's approved `tasks.md`
    (by default the whole item). The item stays the unit that is verified and that unblocks
    dependents, and `verified` is proposed once, after the last slice merges. Per-slice tick
    state comes from spec-status per-phase counts (a future release), so C-01 is unchanged.
  - 013 (the laptop release) depends on 026.
  - spec-status pin bumped to 0.3.0 (`pilot/pins.env`); 0.3.0 has no per-phase counts yet.
Specs affected: 013, 026 (new)
Open questions added/resolved: Q4 decided (one run per slice, verified per item); Q12 notes
026 (slices are sequential; parallel `[P]` tasks stay open)

Prior: Version change: 1.27.0 → 1.28.0
Bump rationale: MINOR — Q7 re-resolved (brand and binary become Chalkeus / `chalkeus`, and
`pilot` is renamed to `forge`), a scope change to 013 (alias and brand story), the 021
label default, a planned repo rename, and a Q8 command-path note. Nothing verified is
invalidated.
Changes this revision (owner decision 2026-11-21, Tosin Akinosho):
  - Q7 re-resolved, superseding 1.27.0: the brand and installed binary are `chalkeus`
    (Greek χαλκεύς, "smith"); `pilot` is renamed to `forge` to fit the smith theme
    (`chalkeus forge …`). 1.27.0 had chosen SpecFoundry / `specfoundry` and the `sfy`
    alias. Name check: no US trademark and
    no npm, PyPI, crates, Go, Homebrew, Docker Hub or Artifact Hub clash; chalkeus.com has
    been parked since 2015; a Polish metalworking firm trades as Chalkeus; EU and WIPO were
    unchecked. The owner accepts that risk and will revisit if the project gets very popular.
    The short alias is `chk`, offered by the installer, not a second binary. `chk` is not a
    common Unix command and is not an exact Debian, Ubuntu or Homebrew-core package, and
    the npm package `chk` has no bin; the uses found (a small third-party Homebrew kubectl
    version switcher, and a Windows GUI hash tool) were small and unrelated.
  - 013 scope: the opt-in alias becomes `chk` in place of `sfy`; tab completion stays. The
    README and install docs include the smith brand story: the spec is the raw metal, the
    worker build is the forging, the grader is the testing and tempering, and the draft PR
    is the proven piece a human inspects.
  - 021: the proposal label's default becomes `chalkeus:proposed` (still a profile setting).
  - Q8 notes that harness rules match `chalkeus forge …`.
  - Q1 planned: after 1.28.0 merges, a single follow-up change updates the Go module path,
    imports, docs, CI badges and GitHub App references, then the GitHub repo is renamed to
    tosin2013/chalkeus in settings (the owner does that, or gives an explicit go-ahead).
    Both happen before 013 ships. GitHub redirects the old URLs. This revision does not
    rename the repo or change the code. The chalkeus.dev domain stays a separate, undecided
    money call.
Specs affected: 013, 021
Open questions added/resolved: Q7 re-resolved; Q1 planned (repo rename to tosin2013/chalkeus);
chalkeus.dev still open (money call)

Prior: Version change: 1.25.0 → 1.26.0
Bump rationale: MINOR — status transitions (021, 024 verified) and a scope addition to 025 from
their debriefs. Nothing verified is invalidated.
Changes this revision (owner decisions 2026-11-21, Tosin Akinosho):
  - 021 → verified: Stockroom 010 carried end to end by step mode with only GitHub answers,
    audit 11/11 after `down` (specs/021-step-mode/evidence/). Outcome wording corrected:
    doctor findings go to the item's issue (the run's tracking issue doesn't exist yet).
    Debrief: specs/021-step-mode/roadmap-reviews/debrief-20261121T144618Z.md.
  - 024 → verified: automatic drafts for Stockroom 009 and 010 in the sandboxed author Job,
    010 verified with them (specs/024-test-author-automation/evidence/). Only the laptop trigger
    is built; the GitHub Actions trigger is later work (operator trigger: 011). Debrief:
    specs/024-test-author-automation/roadmap-reviews/debrief-20261121T144618Z.md.
  - 025 gains the debriefs' gaps: live cases for a non-approver's label and a failing doctor
    (unit-tested only), the A01-A05 rows asserted in CI, and the known limits (a crashed
    in-progress run holding the slot; a changed active feature hiding earlier drafts).
Specs affected: 021, 024, 025
Open questions added/resolved: none

Prior: Version change: 1.24.0 → 1.25.0
Bump rationale: MINOR — a new planned item (025). Nothing existing changes.
Changes this revision (owner decision 2026-11-19, Tosin Akinosho):
  - 025 (new, planned): integration tests in CI on a kind cluster (weekly, and on PRs that touch
    the pilot), after the seam audit of 021
    and 024 (specs/021-step-mode/audit-20261119.md): six bugs found one live step at a time
    were all in the glue that unit tests with fakes can't see.
Specs affected: 025 (new)
Open questions added/resolved: none

Prior: Version change: 1.23.0 → 1.24.0
Bump rationale: MINOR — material scope changes to 010 and 021, and a new dependency for 013.
Nothing verified is invalidated.
Changes this revision (owner decisions 2026-11-19, Tosin Akinosho):
  - 021 becomes the laptop **manager loop**: a reconcile loop that reads the roadmap, item
    labels and PRs, and hands each item to the drafting loop (024) and the run loop, waiting at
    every human gate. It takes 010's label half (a proposal label, honoured only from an
    approver) and polls GitHub; it no longer depends on 010. No standing mode approval in its
    first version.
  - 010 keeps signed webhooks, for cluster mode: the operator (011) receives them, with TLS
    from cert-manager. It now depends on 011.
  - 013 (the laptop release) ships with 021 as well as 024.
  - The proposal label's name follows the product name (Q7, still open); until then it is a
    profile setting with a placeholder default.
Specs affected: 010, 013, 021
Open questions added/resolved: Q7 gains the label name

Prior: Version change: 1.22.0 → 1.23.0
Bump rationale: MINOR — a new planned item (024) split from 018, and status transitions (014,
017 and 018 verified). Nothing verified is invalidated.
Changes this revision (owner decisions 2026-11-19, Tosin Akinosho):
  - 014 → verified. Debrief: specs/014-any-project/roadmap-reviews/debrief-20261119T171222Z.md
    (proceed). Every success criterion met except SC-008 (moved to 022); fixture 11/11 after
    `down` (20261119T164920Z); readiness refusals proven live.
  - 017 → verified (Stockroom, 8 of 8 items). Its "Runs anywhere" OpenShift part moves to 023.
    Debrief: specs/017-greenfield/roadmap-reviews/debrief-20261119T172429Z.md.
  - 018 → verified on its spec'd scope. The two scope lines added 2026-11-18 (opt-in auto-draft,
    the test author as its own Job on laptop clusters) move to 024. Debrief:
    specs/018-test-author/roadmap-reviews/debrief-20261119T172429Z.md.
  - 024 (new, planned): test-author automation. 021 now depends on 024 (it uses auto-draft).
  - 013: the laptop release ships with auto-draft, so it depends on 024 (owner, 2026-11-19).
Specs affected: 013, 014, 017, 018, 021, 024 (new)
Open questions added/resolved: none

Prior: Version change: 1.21.0 → 1.22.0
Bump rationale: MINOR — a new planned item (023) taking 014's OpenShift EAP validation, and
014 scope and dependency corrections from its debrief. Nothing verified is invalidated.
Changes this revision (owner decisions 2026-11-19, Tosin Akinosho):
  - 023 (new, planned): a Quarkus reference project built from an empty repository like
    Stockroom, run by the operator on OpenShift with models served in the cluster (OpenShift
    AI); depends on 011. The legacy Java EE → JBoss EAP upgrade is its stretch goal.
  - 014: the OpenShift EAP validation moves to 023, and 005 leaves Depends on. The toolchain
    scope line is corrected: the repository declares nothing; built-in images plus operator
    profiles (own images in 022). Debrief:
    specs/014-any-project/roadmap-reviews/debrief-20261119T143806Z.md.
  - 011: a note that its cluster resource takes the target profile fields (014 FR-004b).
Specs affected: 011, 014, 023 (new)
Open questions added/resolved: none

Prior: Version change: 1.20.0 → 1.21.0
Bump rationale: MINOR — a new planned item (022) taking 014's operator-supplied toolchain
images, and owner decisions that change 014's and 017's validation. Nothing verified is
invalidated.
Changes this revision (owner decisions 2026-11-18, Tosin Akinosho):
  - 014: Stockroom replaces greeting-service as the Java/Maven validation (T052–T054).
    Operator-supplied toolchain images (T042, T043, T055, SC-008) move to 022, so 014 closes on
    the four built-in toolchains.
  - 017: the Node.js greenfield is dropped; Stockroom (7 of 7 items verified) is the reference.
    SC-005 is reworded: the run that `down` follows audits 11/11 (met).
  - 022 (new, planned): operator-supplied toolchain images.
  - Footer corrected: the 1.20.0 revision below left it at 1.19.1.
Specs affected: 014, 017, 022 (new)
Open questions added/resolved: none

Prior: Version change: 1.19.1 → 1.20.0
Bump rationale: MINOR — a scope addition to 018 (opt-in auto-draft, the test author as its own
Job on kind and minikube), a note on 010, and a new planned item (021 Step mode). Nothing
existing is invalidated: humans still review and merge every hidden-tests PR and approve every
run, and build-worker isolation is unchanged.
Changes this revision (owner discussion 2026-11-18, Tosin Akinosho):
  - 018: auto-draft, opt-in and off by default (`PILOT_AUTHOR_AUTO`). When an item becomes
    ready (its spec and tasks are merged and its companion folder is empty), the test author
    is started automatically instead of by hand. Only drafting is automatic: the companion PR
    is still reviewed and merged by a human, and runs are still approved by a human. This
    replaces the 018 spec assumption "drafting is never triggered automatically" for targets
    that opt in.
  - 018: the test author may run as its own Job in its own namespace on kind and minikube in
    laptop mode, not only in cluster mode (011). It shares no storage with workers (C-08) and
    has its own egress role (C-07). The separate headless session stays as the no-cluster
    option.
  - 010: note that a merged spec-and-tasks PR is a natural event to start auto-draft.
  - 021 (new, planned): step mode. The pilot works through an item's automatic steps on its
    own and waits at each human gate (tests PR merge, mode approval, item PR merge, status PR
    merge), so the owner can step away and answer from GitHub. It runs `doctor --baseline` and
    posts the result to the tracking issue before asking for the mode.
  - 021: an opt-in standing mode approval per target (attempts, retries, spend cap, deadline
    within set limits). When set, a clean `doctor` result starts the run without a
    `/approve mode` comment. This amends "a human approves every run" for targets that opt in;
    the three merges stay human (C-04).
Specs affected: 018 (scope), 010 (note), 021 (new)
Open questions added/resolved: none

Prior: Version change: 1.19.0 → 1.19.1
Bump rationale: PATCH — a status transition and a Spec dir link.
Changes this revision (2026-11-18):
  - 017: planned → specced; Spec dir specs/017-greenfield/ (written guide-first, after
    docs/greenfield.md and Stockroom's first run, 20261118T161914Z: item 001 verified).
Specs affected: 017
Open questions added/resolved: none

Prior: Version change: 1.18.2 → 1.19.0
Bump rationale: MINOR — Q13 resolved (specs half: the controller plans with the human), a new
decision (C-16, deterministic grading only), and a material scope change to 017 (Java/Maven
first, Node later).
Changes this revision (owner decisions 2026-11-18):
  - Q13: humans and the controller (the agent/harness driving the work) plan together; the
    human approves and merges. Build workers never plan; the test author stays separate.
  - C-16 (new): specfoundry grades deterministically; evaluating live model behaviour is not
    its lane (AI apps are graded on the code around the model, with a fake model).
  - 017: the first greenfield validation is a new Java/Maven repository (a small Spring Boot
    REST service: 001 scaffold, 002 a real endpoint); Node.js follows. Spec-first: the
    planning PR (constitution, roadmap, specs, the approved pom.xml and CI) comes before any
    build; workers never change .github/.
Specs affected: 017 (scope)
Open questions added/resolved: Q13 resolved (specs half)

Prior: Version change: 1.18.1 → 1.18.2 (this revision)
Bump rationale: PATCH — status transitions and a missing Spec dir link, after the first real
item run succeeded (run 20261118T030513Z, cli-schema-mcp item 006).
Changes this revision (owner, 2026-11-18):
  - 014: planned → in-progress, and its Spec dir (specs/014-any-project/) linked so spec status
    reads its tasks; the Python path is validated on a real target, the Java/Maven path and the
    remaining US3 tasks are not.
  - 018: specced → implemented (25/25 tasks; first use on cli-schema-mcp 006). `verified` is the
    owner's call after a debrief.
  - 011 note: the spec-status Go library (tosin2013/spec-status#65) is worked on first when 011
    starts; `pilot doctor` and the CLI gate move to it afterwards.
Specs affected: 011 (note), 014 (status, spec dir), 018 (status)

Prior: Version change: 1.18.0 → 1.18.1 (this revision)
Bump rationale: PATCH — a backlog note on 020 (token use ideas); no scope or status change.
Specs affected: 020 (note)

Prior: Version change: 1.17.1 → 1.18.0 (this revision)
Bump rationale: MINOR — a new planned spec (020 Model profiles and usage), a scope addition to
008 (each harness's login import), and Q5 partly amended (OpenCode's own services may be named
as the quick start).
Changes this revision (owner decisions 2026-11-18, after the first cli-schema-mcp run's model
key was blocked by the provider for heavy use):
  - Q5: OpenCode's own services (for example OpenCode Go) may be named in the docs as the quick
    start; other providers stay operator configuration.
  - 008: scope gains `import-<harness>` for each harness's login (Claude Code, Codex), API keys
    first; subscription logins only where the vendor's terms allow automated use.
  - Delivered now (no new entry): `import-opencode --from-login` and a `model key` check in
    `pilot doctor` (one tiny request with the worker's key and model).
  - 020 (new): Model profiles and usage, planned.
Specs affected: 008 (scope), 020 (new)
Open questions added/resolved: Q5 partly amended

Prior: Version change: 1.17.0 → 1.17.1
Bump rationale: PATCH — a clarification of 014's scope: a read-only preflight (`pilot doctor`)
that reports every readiness problem at once. No decision or entry changes.
Specs affected: 014 (note)
Open questions added/resolved: none

Prior: Version change: 1.16.1 → 1.17.0 (this revision)
Bump rationale: MINOR — a new decision (C-15 developer mode), a new planned spec (019), and
material scope changes to C-14 (security levels) and 018 (skill, laptop session). Nothing
existing is invalidated: build workers' isolation is unchanged at every level and in dev mode.

Changes this revision (owner decisions 2026-11-17):
  - C-14: three security levels for the test author and publisher (minimal, the laptop default;
    recommended; strict, the cluster default), with the reasoning: the one realistic way a build
    worker could learn hidden tests is if they land in the target repository, so the levels
    differ in who can write the target and who can read the companion. Build-worker isolation is
    the same at every level.
  - C-15 (new): developer mode, an explicit flag for developing specfoundry itself. It may relax
    test authoring, the companion and approvals, never build-worker isolation; it is stamped
    everywhere and can never change a roadmap status.
  - 018: status specced; the test author is a skill; on a laptop it runs as a separate agent
    session (not the session that drives runs), in cluster mode as its own Job; its identity
    follows the C-14 level.
  - 019 (new): Developer mode, specced.
  - 011: cluster mode defaults to the strict level.

Specs affected: 011 (note), 018 (status, scope), 019 (new)
Open questions added/resolved: none

Prior: Version change: 1.16.0 → 1.16.1 (this revision)
Bump rationale: PATCH — C-14 wording: humans may write hidden tests too (as C-12 and 014 say);
the test author's write access is limited to the companion. Grader evidence now withholds
hidden-test output (014 contract grading.md), so C-14's "never expose test contents" holds.

Prior: Version change: 1.15.0 → 1.16.0
Bump rationale: MINOR — a new planned spec (018) and amendments to C-07, C-12 and C-14. No
existing entry is invalidated: humans still plan and approve, and build workers still never
see hidden tests.

Changes this revision (owner decision 2026-11-17, Tosin Akinosho):
  - C-12: humans still plan and approve. A separate test-author agent may write hidden
    acceptance tests; a human reviews and merges them into the private companion repository.
    Build workers never write or see them.
  - C-14: hidden tests stay out of reach of every build worker, controller-launched worker
    session and publisher. Only the test-author identity may write, and only the companion
    repository (`PILOT_COMPANION_REPO` / `PILOT_COMPANION_PATH`). The test author never
    launches, steers or reviews workers. Worker and grader evidence never expose test contents.
  - C-07: a test-author egress role reaches model hosts and the companion repository's SCM
    host only. It has no route to the worker or publisher proxies, and workers cannot reach it.
  - New entry: 018 Test author (planned, not started); depends on 014.
  - 011, 014 and 017: C-14 added to Governed by. 014's "written by a human" scope line marked
    as no longer the only path.
  - Q13: the tests half is resolved (agents may draft hidden tests only via the test-author
    role; a human merges). The specs and tasks half stays open.

Specs affected: 011, 014, 017 (governed by); 018 (new)
Open questions added/resolved: Q13 partly resolved (tests half)

Prior: Version change: 1.14.0 → 1.15.0
Bump rationale: MINOR — a new decision (hidden tests across contexts) and a scope change on
014 (one companion for many targets); 011 gains a matching scope note. No entry is invalidated.

Changes this revision (owner decisions 2026-11-17):
  - New decision C-14: hidden acceptance tests come from a private git repository the agent's
    side can't read, in every context (laptop, server, team, cluster); each run records the
    repository, commit and folder.
  - 014: one companion repository may serve many targets in any languages
    (`PILOT_COMPANION_PATH`); the companion is storage only, never built as a whole.
  - 011: the operator reads the companion read-only (GitHub App token for that repository only);
    graders still hold no SCM credential.
  - 014 note: the first real target is tosin2013/cli-schema-mcp (Python 3.13), item 006.

Prior: Version change: 1.13.1 → 1.14.0
Bump rationale: MINOR — a constraint gains an exception (C-09: operator-supplied toolchain
images); it doesn't invalidate existing entries. Plus a material scope clarification on 014.

Changes this revision (owner decisions 2026-11-17, during 014's implementation):
  - C-09: exception for a target's toolchain image, which may be operator-supplied: pinned by
    digest, checked by the toolchain contract probe, shown at the mode approval and recorded
    as operator-supplied. A GitHub build attestation is verified when present but not
    required. The worker image and the grader/publisher tools image always stay
    specfoundry-built.
  - 014 note: "any repo" means any programming language. specfoundry builds JDK 21 + Maven,
    Node.js 24 + npm, Python 3.13 and Go first, replacing the JDK 17 plan. Other languages use
    an own toolchain image from a template (UBI base for Red Hat/OpenShift, any base
    elsewhere) or the request form. Target settings live in operator-side profiles
    (targets/<name>.env), which become a cluster resource in 011.

Specs affected: 014 (scope), 011 (note via 014), 017 (Node.js 24 toolchain available)
Open questions added/resolved: none

Previous revision 1.13.1 (2026-11-17):
Version change: 1.13.0 → 1.13.1
Bump rationale: PATCH — two status transitions, an outcome clarification and notes; no new
entry, constraint or open question.

Changes this revision (owner decisions 2026-11-17, after the 016 and 004 debriefs):
  - 016: planned → verified; spec dir linked; notes: the spec-status library is deferred to
    tosin2013/spec-status#65 (the exec gate meets FR-015), approval replies (FR-019) added on
    the owner's request, App mode first proven end to end on the Go pilot
  - 004: planned → verified (SC-007, the App setup time, was not measured; the owner accepted
    that); outcome clarified: one issue per roadmap item with its task checklist, as the spec
    decided; notes: validated on the Go pilot (016 T047), refusal cases partly in CI by owner
    decision, approval replies and the walk-away watcher delivered with 016
  - Unblocked: 013 (depends on 016); 010 and 014 (depend on 004; 014 also on 016)

Specs affected: 004, 016 (status); 010, 013, 014 (dependencies now satisfied)
Open questions added/resolved: none

Previous revision 1.13.0 (2026-11-16):
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
    017, 009, 015, then the 013 release), then OpenShift (005, then 014's EAP validation); see
    Cross-Cutting Notes
  - 015 note: scaling is in the first release, with a small default concurrency limit on
    laptops (for example 2)
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
  **Test-author role (owner decision 2026-11-17, Tosin Akinosho):** a test-author egress role
  reaches model hosts and the companion repository's SCM host only. It has no route to the
  worker proxy or the publisher proxy, and workers cannot reach it. NetworkPolicy still decides
  which namespace may reach which proxy.
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
  **Exception (owner decision 2026-11-17, 014):** a target's *toolchain* image, which only
  supplies a language's tools under `/opt/toolchain`, may be operator-supplied. It must be
  pinned by digest and pass the toolchain contract probe. It is shown at the mode approval and
  recorded in the evidence as operator-supplied. A GitHub build attestation is verified when
  present but not required, so enterprise CI and UBI bases work. The worker image and the
  grader/publisher tools image are always built here, under the rule above.
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
  **Amended 2026-11-17 (owner decision, Tosin Akinosho):** humans still plan and approve. A
  separate test-author agent may write the hidden acceptance tests. A human reviews that work
  and merges it into the private companion repository. Build workers never write those tests
  and never see them. The test author does not implement roadmap items, and the pilot still
  never writes specs or picks up an item by itself.
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

- **C-14 — ~~Hidden tests stay out of the agent's reach in every context~~ Hidden tests stay out of every build worker's reach** (owner decision
  2026-11-17; **amended 2026-11-17, Tosin Akinosho**): hidden acceptance tests come from a
  private git repository (the companion) ~~that the agent's side can't read~~. They stay out of
  reach of every build worker, every controller-launched worker session and the publisher,
  whether the pilot runs on a laptop, a server, for a team or in a cluster. Only humans and the
  test-author identity have write access, the test author only to that companion repository
  (`PILOT_COMPANION_REPO` / `PILOT_COMPANION_PATH`). That write is not a credential for the
  target repository: the publisher remains the only identity that may write the target (C-03).
  The test author never launches, steers or reviews workers. Worker evidence and grader
  evidence never expose test contents. ~~Each context reads it with its own credential (your
  git access; a test-authors team on GitHub; the operator's read-only App token), and~~ each
  run records the repository, commit and folder. Humans and the operator still read it with
  their own credentials (your git access; the operator's read-only App token). One companion
  may serve many targets. A local folder is acceptable only as a laptop experiment and must be
  labelled so in the evidence.

  **Security levels (owner decision 2026-11-17).** Why the rules exist: a build worker sees only
  the item's spec and tasks and a snapshot of the target repository, and has no route to GitHub.
  The one realistic way it could learn hidden tests is if they land **in the target
  repository** (an agent copies them there by mistake or through a prompt injection, or a
  publisher bug carries them over); the next run's snapshot would then contain them. So the
  levels differ only in who can write the target and who can read the companion. Build-worker
  isolation is the same at every level.

  | Level | For | Test author's identity | Publisher on the companion | Test author's egress |
  |---|---|---|---|---|
  | **minimal** (laptop default) | a solo developer | the operator's own git access | allowed | not enforced |
  | **recommended** | teams | a token or App that can write **only** the companion | not allowed | not enforced |
  | **strict** (cluster default) | enterprise, cluster mode | a separate App, companion only, its own Job | not allowed | enforced (C-07) |

  At the minimal level the protection is branch protection on the target, human review of
  every PR, the test-author skill's rules, and P11. Separate identities are **recommended**
  for teams and **expected** in enterprise environments, where review alone doesn't scale and
  separation of duties (who wrote the tests vs. who wrote the code) is often required. Evidence
  records the level; the audit reports it and never fails a run for using a lower level.

- **C-16 — specfoundry grades deterministically; evaluating model behaviour is not its lane**
  (owner decision 2026-11-18): an item is `verified` when its deterministic acceptance tests pass,
  offline. For AI-driven apps that covers everything around the model (prompt construction,
  tool and function wiring, parsing structured output, retries, guardrails), tested with a fake
  model or recorded responses. Judging live model behaviour ("is the answer good?", "does
  retrieval find the right document?", "does the model call the right tool?") is evaluation: it is
  probabilistic and model-dependent, and belongs to the project's own evaluation tools and CI,
  which a human can read before merging. specfoundry neither runs live-model evaluations nor gives
  graders model access for them.

- **C-15 — Developer mode is a labelled sandbox** (owner decision 2026-11-17): while
  specfoundry itself is being developed, an explicit flag (`--dev`, never a default) may relax
  steps that exist only to protect real results: hidden tests from a local folder, the same
  agent session drafting them, approvals in the terminal only. It never relaxes build-worker
  isolation (no secrets, model-only egress, no hidden tests in worker inputs). A dev-mode run is
  stamped in its evidence, at its start, in its PR titles (`[dev]`) and in the audit
  ("development run: does not count"), and it can never propose or change a roadmap status.
  The first real run of an item (for example `cli-schema-mcp` 006) uses a real level, not dev
  mode, so that its result counts.

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

### 004 — M3 GitHub loop  [status: verified]

- **Description:** Tie roadmap entries, issues and PRs together on GitHub.
- **Outcome:** ~~taskstoissues creates one issue per task,~~ one issue per roadmap item with its
  task checklist *(clarified 2026-11-17: what the spec decided and what was built)*, labels mirror
  spec status state, PRs carry Closes #N, and reconcile after merge shows no drift.
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
  **2026-11-17: verified** on the Go pilot (016 T047; evidence in `specs/004-github-loop/spec.md`
  §Evidence and the debrief). Live: App mode 11/11 with every decision on GitHub, the
  walk-away run, `publisher_identity` and `STATUS_PR_MERGED_BY_PUBLISHER` refusals, and the
  App's write probe (fixture writable, canary not). By owner decision, `wrong_step`, `edited`,
  `withdrawn` and `not_an_approver` are covered in CI rather than live (one maintainer account).
  SC-007 (App setup under 30 minutes) was not timed; the owner accepted that. Delivered with 016:
  approval replies on GitHub (FR-019) and the walk-away watcher (issue #25).
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
  **Amended 2026-11-18 (owner):** scope gains each harness's own login import
  (`import-claude-code`, `import-codex`), like `import-opencode --from-login`: one credential
  copied at mode 0600, the worker's only egress the provider's API host, `doctor` checking the
  key. API keys first; subscription logins only where the vendor's terms allow automated use.

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
- **Depends on:** 004, 011 (2026-11-19: the webhook receiver is the operator's)
- **Governed by:** C-01, C-02, C-04
- **Addresses:** docs/prd.md F6
- **Notes:** Proposed in the 003 Kelos review (owner decision 2026-11-13). A merged
  spec-and-tasks PR is a natural event to start 018's opt-in auto-draft of hidden tests
  (2026-11-18); it proposes a draft, not a run, and a human still merges the tests.
  **Amended 2026-11-19 (owner): signed webhooks, cluster mode.** Label proposals moved to 021
  (laptop, polled). 010 keeps signed webhooks: the operator (011) receives GitHub's webhooks
  over TLS with certificates from cert-manager, verifies their signatures, and turns a proposal
  into the same decisions 021 makes, instead of polling.

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
- **Governed by:** C-01, C-02, C-03, C-04, C-07, C-08, C-10, C-11, C-14
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
  **Hidden tests (C-14, 2026-11-17):** the operator reads the companion read-only, with a
  GitHub App token scoped to that repository's contents; graders still hold no SCM credential,
  and runs record the companion's repository, commit and folder exactly as in laptop mode.
  The test-author Job (018) is the only cluster identity that may write the companion, and it
  runs in its own namespace; the operator's grader path stays read-only.
  **spec-status library (owner decision 2026-11-18):** when 011 starts, the spec-status
  developers work on the Go library (tosin2013/spec-status#65) first, so the operator evaluates
  the gate in-process instead of running the `spec` CLI in a container; the CLI container image
  is the fallback. Afterwards, `pilot doctor` and the CLI's gate move to the same library. The
  grader keeps the CLI inside its isolated image. The CLI and the library come from one pinned
  spec-status release and must give the same answer for the same roadmap (the audit relies on
  it). Cluster mode defaults
  to the **strict** level (C-14).
  **Target profiles (014 FR-004b, 2026-11-19):** the run resource takes the fields of 014's
  operator-side target profiles (`targets/<name>.env`: repository, branch, toolchain, install
  and test commands, registry hosts, companion repository and path, test paths), so a target is
  configured the same way in both modes and the repository still declares nothing.

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
  (Q8); install docs; **tab completion set up on install** for bash, zsh, fish and PowerShell
  (owner condition for the long binary name, 2026-11-21); an opt-in `chk` shell alias
  offered on install, with completion wired to it as well; the smith brand story in the
  README and install docs (the spec is the raw metal, the worker build is the forging, the
  grader is the testing and tempering, and the draft PR is the proven piece a human inspects).
- **Scope (out):** OS package managers; signing beyond checksums (revisit with Q10).
- **Depends on:** 016, 024, 021, 026 (2026-11-19: the laptop release includes auto-draft and the
  manager loop. 2026-11-21: it also ships slice runs)
- **Notes:** **Owner decision 2026-11-21 (issue #142):** the laptop release ships with slice
  runs (026), so items with several stories of different risk can be built and reviewed one
  story at a time. **Owner decision 2026-11-19:** the laptop-mode release ships with the test
  author's auto-draft (024), so a released pilot drafts an item's hidden tests on its own once
  the item is ready; a human still reviews and merges them.
  Raised by the owner 2026-11-15 during the 012 debrief. Tied to Q7 (product name
  and CLI surface) and Q8 (multi-project use; harness rules today match the `pilot/pilot`
  path). Decide the form and name before speccing. **2026-11-15:** form decided by C-13 (one Go
  binary); the name is still Q7. Dependency changed from 012 to 016 (2026-11-15).
  **2026-11-21 (owner):** Q7 resolved: the binary is `specfoundry`, with `pilot` as a
  subcommand (`specfoundry pilot …`), and completion ships with the install.
  **2026-11-21 (owner, later):** Q7 re-resolved (1.28.0, superseding 1.27.0): the binary is
  `chalkeus`, with `forge` as the main subcommand (`chalkeus forge …`); `pilot` is renamed
  to `forge` to fit the smith theme. The opt-in alias is `chk`.
  The repo rename to tosin2013/chalkeus is planned after this revision merges and before 013
  ships (see Q1); this item's docs also carry the smith brand story.

### 014 — Any project, any item  [status: verified]

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
  - per-repo toolchains for the worker and the grader (an image per language~~, or declared in
    the repo's configuration~~), still digest-pinned and hardened (C-08, C-09). *(Corrected
    2026-11-19: the repository declares nothing; the operator's target profile names one of the
    built-in images. Operator-supplied images are 022.)*
  - an egress allowlist for the repo's package registries, so dependencies can be installed,
    declared per repo and shown at the mode approval;
  - hidden acceptance tests ~~written by a human and~~ kept out of the worker's view, with the
    repo's own test suite run as a visible check. A human may write them, or the test author
    (018) may draft them for a human to review and merge (C-12, owner decision 2026-11-17).
- **Scope (out):** parallel runs (015); writing specs or plans; choosing items automatically.
- **Depends on:** 004, 016 (2026-11-19: no longer 005; the OpenShift EAP validation moved to
  023)
- **Spec dir:** specs/014-any-project/
- **Governed by:** C-01, C-02, C-04, C-09, C-12, C-14
- **Addresses:** docs/prd.md §4 loop, F1, F8, F9
- **Notes:** Owner decision 2026-11-15. The fixture stays the regression test. Pairs with 013
  (installable CLI). Grading a real repo needs the item's acceptance criteria to be runnable;
  the spec decides how that is expressed. 014 targets **brownfield** repos (existing code and
  tests); greenfield bootstrap is Q11 (owner decision 2026-11-15).

  ~~**Validation (owner decision 2026-11-15):** a real **Quarkus** (Java) brownfield repository,~~
  ~~chosen when 014 is specified, with a roadmap of a few items where one depends on another.~~
  **Validation (owner decision 2026-11-16, replaces the Quarkus validation):** *(Moved to 023
  on 2026-11-19, owner decision; kept here as history. 014 was validated on kind with
  cli-schema-mcp and Stockroom.)* A legacy Java EE
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

  **Any language (owner decision 2026-11-17):** "any project" means any programming language.
  - **specfoundry builds** JDK 21 + Maven, Node.js 24 + npm, Python 3.13 and Go first. JDK 21
    replaces the earlier JDK 17 plan.
  - **Any other language** uses an own toolchain image built from a template (a UBI base for
    Red Hat and OpenShift, any base elsewhere; C-09 exception), or the request form.
  - **Target settings** (repository, toolchain, test command, registry hosts, companion
    repository with the hidden tests) live in operator-side profiles,
    `~/.config/specfoundry/targets/<name>.env`. The target repository declares nothing. In
    cluster mode (011) the same fields become a cluster resource.
  - **Preflight (owner decision 2026-11-18):** `pilot doctor --target <name> [--item NNN]
    [--baseline]` reports every problem with a target and item in one pass (profile, cluster,
    GitHub access, every readiness rule, coverage; optionally the project's tests offline on the
    cluster), so problems surface before anyone prepares tests. It changes nothing; runs still
    fail closed at their own checks.
  - **Hidden tests (owner decision 2026-11-17, C-14):** one private companion repository
    may serve all targets, in any languages; each target names its folder
    (`PILOT_COMPANION_PATH`). The companion is storage only; the pilot copies one item's folder
    over the project in the grader. Who drafts them is 018: a human, or the test author with a
    human merge. First real target: `tosin2013/cli-schema-mcp`, item 006.
  - See `specs/014-any-project/`.
  - **Validation done on Stockroom (owner decision 2026-11-18):** `tosin2013/stockroom`, the
    Java/Maven greenfield of 017, replaces greeting-service for T052–T054: 7 items ran end to
    end on `jdk21-maven`, with dependent items and no configuration edit between runs; the
    last run audits 11/11 after `down`. **Operator-supplied toolchain images** (T042, T043, the
    T055 run, SC-008) move to 022, so 014 closes on the four toolchains specfoundry builds.
  - **Verified 2026-11-19 (owner):** debrief proceed
    (`specs/014-any-project/roadmap-reviews/debrief-20261119T171222Z.md`). Every success
    criterion met except SC-008 (022): cli-schema-mcp and Stockroom 001–008 end to end, the
    readiness refusals live, P06r and an undeclared request refused live (Stockroom 008), and the
    fixture 11/11 after `down` (20261119T164920Z) with no configuration edit after Stockroom.

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

  **In the first release (owner direction 2026-11-16):** scaling ships in the first release
  (013), even if a laptop runs only a few items at once. The default concurrency limit on a
  laptop is small (for example 2), and the value is set when 015 is specified.

  **Parallel tasks inside an item (owner direction 2026-11-16, proposed):** blockers still block,
  and that is fine. Where tasks have no blockers between them (Spec Kit's `[P]` marker in
  tasks.md, checked against spec-status), they may run at the same time, each in its own
  isolated unit; dependent tasks keep their order. The pilot owns the schedule: the human sees
  one plan at release (what runs together, what waits, and why), and each worker sees only its
  own task. See Q12.

### 016 — Go pilot at parity  [status: verified]

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

  **2026-11-17: verified** (evidence in `specs/016-go-pilot/spec.md` §Evidence and the debrief):
  the Go-only pilot ran the guide's walkthrough at 11/11 after the Python pilot was retired, and
  its audit matches the reference on all golden records. The gate still runs the `spec` binary:
  the library form waits for tosin2013/spec-status#65, and the exec form meets FR-015. Added on
  the owner's request: approval replies on GitHub (FR-019). App mode was first proven end to end
  on the Go pilot; the parity runs also fixed a publisher-namespace quota race and an AU08 false
  positive that the reference shared. 004's refusal cases are partly covered in CI by owner
  decision (see 004).
- **Spec dir:** specs/016-go-pilot/

### 017 — Greenfield bootstrap  [status: verified]

- **Description:** Start a project from an empty repository: the roadmap's first item is a
  scaffold, and every later item grows the project like a brownfield one.
- **Outcome:** On a new, empty ~~**Node.js**~~ **Java/Maven** repository (Node.js follows;
  owner decision 2026-11-18) whose constitution and roadmap a human wrote ~~,~~ (with the
  controller, Q13), the pilot runs item 001 (a human-specified scaffold: layout, package manifest, test
  runner, CI), grades it against acceptance tests the human wrote, and publishes a draft PR.
  After the human merges it and approves `verified`, the next items run as in 014.
- **Scope (in):**
  - grading an item when the repository has no tests or build of its own yet (only the hidden
    acceptance tests, and the scaffold's own tests once it exists);
  - ~~a Node.js toolchain for the worker and grader, and npm registry egress,~~ the JDK 21 +
    Maven toolchain (Maven Central egress) first, then Node.js 24 (npm), through 014's per-repo
    mechanisms;
  - guidance for writing a scaffold item's spec and acceptance tests;
  - the validation on a new ~~Node.js~~ Java/Maven repository (a small Spring Boot REST
    service) with at least the scaffold and one dependent item; a Node.js greenfield follows.
- **Scope (out):** generating the constitution, roadmap or specs (humans plan, C-12); project
  templates as a product; languages beyond the ones 014 and 017 validate.
- **Depends on:** 014
- **Governed by:** C-01, C-04, C-07, C-08, C-12, C-14
- **Addresses:** docs/prd.md §4 loop, F1
- **Spec dir:** specs/017-greenfield/
- **Notes:** Owner decision 2026-11-15; resolves Q11. ~~The Node.js repository is chosen when
  017 is specified.~~ **Amended 2026-11-18 (owner):** Java first, Node later; the first
  repository is a new, small Spring Boot REST service (001 scaffold, 002 a real endpoint),
  planned by the owner with the controller (Q13). It also gives the JDK 21 + Maven toolchain its
  first real end-to-end run.
  **Spec-first (owner, 2026-11-18):** the greenfield repository is planned before anything is
  built: its constitution, roadmap and item specs are the guide and reference. The scaffold's
  dependencies are defined in its approved spec, and the planning PR commits a matching
  `pom.xml`, so the offline cache is filled only from human-approved inputs (014 R7). CI also
  comes in the planning PR: `.github/` stays protected, and workers never change it. Planning still comes first: an empty repository gets its constitution, roadmap,
  ~~and the scaffold's spec, tasks and acceptance tests from a human before any agent runs.~~
  and the scaffold's spec and tasks from a human before any build worker runs. Acceptance
  tests may be written by that human or drafted by the test author (018); a human merges them
  (owner decision 2026-11-17).
  **Runs anywhere (owner direction 2026-11-16):** *(2026-11-19: verified on kind; the OpenShift
  part moved to 023.)* The greenfield validation must pass on every
  cluster profile (kind, minikube, plain Kubernetes and OpenShift), using an on-demand Node.js
  image off OpenShift and a UBI Node.js image on OpenShift.
  **Verified 2026-11-19 (owner):** Stockroom, 8 of 8 items, each on its first attempt; the run
  that `down` followed audits 11/11 (SC-005).
  **Node.js dropped (owner decision 2026-11-18):** Stockroom is the reference greenfield
  (https://github.com/tosin2013/stockroom, 7 of 7 items verified, each on its first attempt);
  no Node.js greenfield is planned. 017's spec drops US5, FR-008 and SC-004.

### 018 — Test author  [status: verified]

- **Description:** A separate test-author agent drafts an item's hidden acceptance tests.
  Humans still plan and approve. Build workers never write or see those tests.
- **Outcome:** For a prepared item, the test author opens a pull request on the private
  companion repository. A human reviews and merges it. The evidence records the test-author
  identity and the companion commit. Every hidden test maps to an acceptance criterion in the
  item's spec, and any criterion with no test is flagged. Build workers, controller-launched
  worker sessions and the publisher still cannot read the tests.
- **Scope (in):**
  - a test-author skill and job, with its own agent session, separate from every build worker;
  - a credential that can write only the companion repository (`PILOT_COMPANION_REPO` /
    `PILOT_COMPANION_PATH`);
  - its own egress role (C-07): model hosts and the companion repository's SCM host only, with
    no route to the worker or publisher proxies, and no route from workers to it;
  - laptop mode uses a separate identity; cluster mode runs the test author as its own Job in
    its own namespace (C-11);
  - **amended 2026-11-17:** the test author is a **skill**. On a laptop it runs as a separate
    agent session (never the session that drives runs); in cluster mode the Job runs an agent
    with the same skill. Its identity follows the C-14 level: the operator's own access at
    minimal, a companion-only token or App at recommended, a separate App at strict. Helper
    commands in `pilot/pilot` check a draft against the unchanged base, check the mapping and
    open the companion PR;
  - the output is a pull request to the companion repository, which a human merges (C-04);
  - evidence records the author identity and the companion commit;
  - a new grader check: every hidden test maps to an acceptance criterion in the item's spec,
    and criteria with no test are flagged;
  - ~~**amended 2026-11-18 (auto-draft, opt-in):** with `PILOT_AUTHOR_AUTO=on` for a target, the
    test author starts on its own when an item becomes ready (spec and tasks merged, companion
    folder empty). It is off by default. Only drafting is automatic: a human still reviews and
    merges the companion PR (C-04) and still approves every run. Triggers: a GitHub Actions
    workflow on the target's merge (a companion-only token, the recommended level), or the
    pilot on the laptop, or the operator in cluster mode (011). In every case the author is a
    separate process with its own identity, never the driving session;~~ *(Moved to 024 on
    2026-11-19, owner decision.)*
  - ~~**amended 2026-11-18 (Job on laptop clusters):** in laptop mode the test author may also
    run as its own Job in its own namespace on kind or minikube, with its own egress role
    (C-07) and no storage shared with workers (C-08), so strict-style isolation is available
    before 011. A separate headless agent session stays as the no-cluster option.~~ *(Moved to
    024 on 2026-11-19, owner decision.)*
- **Scope (out):** build workers writing or seeing hidden tests; the test author launching,
  steering or reviewing workers; the test author touching worker branches; the test author
  writing specs, tasks or the roadmap (the specs and tasks half of Q13 stays open).
- **Depends on:** 014
- **Governed by:** C-01, C-03, C-04, C-07, C-08, C-11, C-12, C-14, C-15
- **Spec dir:** specs/018-test-author/
- **Notes:** **Verified 2026-11-19 (owner):** the test author drafted the hidden tests for
  cli-schema-mcp 006 and Stockroom's items in a separate session at the minimal level; every
  draft failed on the base and was merged by the owner, and every run using them passed P11. The
  recommended level has no live run yet. Auto-draft and the laptop Job are 024.
  Owner decision 2026-11-17 (Tosin Akinosho). Specced 2026-11-17. The publisher remains the
  only writer of the target repository (C-03); the test author's credential is companion-only.
  The test author never launches, steers or reviews workers, and worker evidence and grader
  evidence never expose test contents (C-14).

  These checks from 014 stay as they are. This item does not redefine them:
  - **G5** still means the hidden acceptance tests pass on the worker's change, and that the
    worker never saw them. The test author produces those tests. G5 is not the mapping check.
  - **P11** still fails if an overlay file name or content hash appears in the worker's
    ConfigMaps, staged input or prompt. The test-author session must not put test contents on
    any worker path.
  - **BASELINE_HIDDEN_PASSES** still refuses a run whose hidden tests already pass on the
    unchanged base. Tests the author drafts have to fail on that base, or the run has no signal.
  - **ITEM_NO_HIDDEN_TESTS** still refuses a run when `items/NNN/` is missing, empty or
    unreadable. The human-merged companion pull request is what fills that folder. The test
    author does not bypass the readiness check.

  The mapping check is new and sits beside those four. It reads the merged companion folder
  and the item's spec, and it does not show test contents to workers.

### 019 — Developer mode  [status: specced]

- **Description:** An explicit `--dev` flag for developing specfoundry itself, which relaxes
  steps that only protect real results and stamps the run as a development run (C-15).
- **Outcome:** A maintainer can exercise the pilot end to end on a throwaway item without a
  private companion, a separate test-author session or GitHub approvals. Every dev-mode run is
  unmistakably labelled and can never change a roadmap status. Build-worker isolation is
  identical to a real run.
- **Scope (in):** the `--dev` flag on run and resume (and the test author's commands); hidden
  tests from a local folder; the same session drafting tests; terminal-only approvals; stamps in
  evidence, run start, PR titles and audit; no status PR and no status change; a check that dev
  mode never changes worker isolation (the same P-checks pass).
- **Scope (out):** any relaxation of build-worker isolation, secrets handling or human merge;
  making dev mode a default or a profile-wide setting; counting dev-mode runs as evidence.
- **Depends on:** 014
- **Governed by:** C-04, C-12, C-14, C-15
- **Spec dir:** specs/019-dev-mode/
- **Notes:** Owner decision 2026-11-17: "a devmode option that is a flag to the project when
  developing".

### 020 — Model profiles and usage  [status: planned]

- **Description:** Named model profiles and usage tracking, so a run can balance cost and
  capability while coding.
- **Outcome:** The mode the human approves says which model profile each attempt uses (for
  example a cheap default first, a stronger model for retries). Each run reports tokens and cost
  by model, attempt and item, and a subscription's usage limit is tracked like a budget; before an
  attempt, the pilot checks what is left (with 009) and pauses rather than start work it can't
  finish.
- **Scope (in):** model profiles in `pilot.env` (each a provider, model and key file); a
  per-attempt profile in the mode; usage in evidence and `pilot status`; subscription limits as
  token budgets; integration with 009's budget check and 007's telemetry.
- **Scope (out):** switching models inside an attempt (the harness's job); provider billing
  integrations; automatic model choice without the human seeing it in the mode.
- **Depends on:** 009, 014
- **Governed by:** C-01, C-02, C-10
- **Notes:** Owner decision 2026-11-18, after the first `cli-schema-mcp` run's key was blocked
  by its provider for heavy use.
  **Backlog (owner, 2026-11-18), best done once clusters are up (011):** the first real attempt
  on `cli-schema-mcp` used ~326k tokens. Ideas to cut that: prompt guidance per item (which
  files not to open, such as large data folders; run only the relevant tests while working, the
  full suite once at the end); a per-attempt breakdown of tokens by step (files read, test runs,
  largest inputs) in the evidence; a lower token cap for small items in the mode. The grader
  itself spends no tokens.

### 021 — Step mode: the laptop manager loop  [status: verified]

- **Description:** Let the pilot carry an item, and then the next ready item, through every
  automatic step on its own, stopping at each human decision, so the owner can leave it running
  and answer from GitHub.
- **Outcome:** After the owner merges an item's spec and tasks, the pilot drafts hidden tests
  (018 auto-draft) and waits for the tests PR merge; runs `doctor --baseline` ~~and posts the
  result to the tracking issue~~ (findings are posted on the item's issue when it fails,
  corrected 2026-11-21); waits for `/approve mode` (or uses a standing approval); builds
  and grades; waits for the item PR merge; opens the status PR and waits for its merge; then
  moves to the next ready item. Every wait is a comment or a merge on GitHub, nothing runs while
  it waits, and it resumes when the owner answers. A failing `doctor` stops the item and says
  what to fix.
- **Scope (in):**
  - a step-mode loop for laptop mode, started per target, that survives the owner being away
    (no terminal input needed at any gate);
  - `doctor --baseline` run automatically before the mode gate, with its findings on the
    tracking issue;
  - an opt-in **standing mode approval** per target: a bounded mode (attempts, retries, spend
    cap, deadline) the owner sets once; when it is set and `doctor` is clean, the run starts
    without a `/approve mode` comment. Off by default. Every run started this way is recorded as
    started under the standing approval;
  - picking the next item only from `spec status` readiness (C-01, C-02);
  - the same loop in cluster mode through the operator (011).
- **Scope (out):** auto-merging any PR (C-04): the tests PR, the item PR and the status PR stay
  human merges; a standing approval without a spend cap or deadline; skipping the gate or
  `doctor`; step mode in developer mode counting as evidence (C-15).
- **Depends on:** 024 (2026-11-19: 010 is no longer needed; the label half moved here)
- **Spec dir:** specs/021-step-mode/
- **Governed by:** C-01, C-02, C-04, C-11, C-12
- **Notes:** Owner decision 2026-11-18 (Tosin Akinosho), after the Stockroom runs: "if I wanted
  to step away from the computer and leave it running it could do it by steps." Stockroom shows
  why the merges stay human: item 005 passed every check, and only reading the PR found the
  stale stock figure (now item 007). The spend budget (009) should land before standing
  approvals are used beyond a single item.
  **Amended 2026-11-19 (owner): the manager loop.** On the laptop, step mode is one manager
  process per target, built like an operator's reconcile loop:
  - each pass reads the roadmap (`spec status`), the item issues' labels and open PRs, and
    decides each item's next step;
  - it delegates: drafting to the test author's loop (024), runs to `run` and `github watch`;
  - it holds no state of its own beyond GitHub and the run evidence, so it can stop and
    restart at any point;
  - an item is **proposed** by a label on its issue (004 mirrors items as issues), honoured only
    when an approver (`PILOT_APPROVERS`) added it, read from the issue's timeline. The label
    proposes; the gate, `doctor` and `/approve mode` still decide (010's rule). Its name follows
    Q7: default `chalkeus:proposed` (Q7 re-resolved 2026-11-21, 1.28.0, superseding the
    1.27.0 default `specfoundry:proposed`), still a profile setting;
  - it polls GitHub, as the other watchers do; signed webhooks are 010, in cluster mode;
  - the first version always waits for `/approve mode`; the standing approval follows 009.
  The same decisions run in the operator (011) later, on a different host.
  **Verified 2026-11-21 (owner):** Stockroom 010 went from the owner's label to `verified` with
  only GitHub answers (drafting, doctor, the run's `/approve mode`, item and status PRs); audit
  11/11 after `down`. Evidence: specs/021-step-mode/evidence/. Fixes found live: #121-#128 and the
  seam audit (#125).

### 022 — Operator-supplied toolchain images  [status: planned]

- **Description:** A target may use a toolchain image the operator built, for a language
  specfoundry has no image for.
- **Outcome:** with `PILOT_TOOLCHAIN_IMAGE` set to a digest, a prepared item in such a language
  runs end to end under every guarantee; the mode approval shows the image as
  operator-supplied, a GitHub build attestation is verified when present, and the audit is
  11/11.
- **Scope (in):** 014's T042 (the image in the toolchain init container only; attestation:
  verified, absent and recorded, or failing and refused with `TOOLCHAIN_ATTESTATION_FAILED`),
  T043 (the own-image template: UBI and any-base Dockerfiles, a `toolchain.json` example, a
  build-and-attest workflow, a README) and T055 (one run in a language with no specfoundry
  image, 014 SC-008).
- **Scope (out):** replacing the worker or tools images (never; C-09); building images for
  users.
- **Depends on:** 014
- **Governed by:** C-08, C-09 (operator-supplied exception), C-14
- **Notes:** Owner decision 2026-11-18: split from 014 so 014 closes on the four built-in
  toolchains (JDK 21 + Maven, Node.js 24, Python 3.13, Go). 014's tasks.md keeps T042, T043 and
  T055 as the starting point.

### 023 — Quarkus reference project on OpenShift  [status: planned]

- **Description:** A second reference project built roadmap-driven from an empty repository, as
  Stockroom was, on Quarkus, run by the operator on OpenShift with models served in the cluster.
- **Outcome:** a Quarkus service grows from an empty repository through a few dependent
  roadmap items. The operator (011) runs each item on OpenShift, code generation uses models
  served in the cluster (OpenShift AI), and each item is graded offline with hidden tests and
  merged by the owner, every run auditing 11/11. Stretch goal: a legacy Java EE application
  upgraded to a current JBoss EAP release, the same way, one item per upgrade step.
- **Scope (in):**
  - the Quarkus project: planning PR (constitution, roadmap, first specs, every dependency
    declared), hidden tests in the companion, items on `jdk21-maven`, as in
    [docs/greenfield.md](../../docs/greenfield.md);
  - runs by the operator on OpenShift, with the model endpoint in the cluster, so model traffic
    stays in the cluster;
  - stretch: the EAP upgrade validation moved from 014 (its notes keep the detail: a `javax`
    application with real tests, Migration Toolkit for Applications analysis when planning, the
    grader trusting only hidden tests and the app's suite, UBI images, the model in the
    cluster).
- **Scope (out):** new pilot features beyond what 005 brings; a Node.js greenfield (dropped in
  017).
- **Depends on:** 014, 017, 011
- **Governed by:** C-04, C-09, C-12, C-14
- **Notes:** Owner decision 2026-11-19, from 014's debrief: "something similar to what we did
  with Stockroom, but with Quarkus; the Java EAP could be a stretch goal when running in
  OpenShift". Owner decision 2026-11-19 (later the same day): 023 starts only once the operator
  runs in OpenShift (011, which itself needs 005), because its code generation uses models
  served in the cluster (OpenShift AI); it does not start on kind first. The Quarkus version,
  the in-cluster model, and the target release for EAP are decided when 023 is specified.

### 024 — Test-author automation  [status: verified]

- **Description:** The test author starts on its own when an item becomes ready, and can run as
  its own Job on a laptop cluster.
- **Outcome:** with `PILOT_AUTHOR_AUTO=on` for a target, a ready item (spec and tasks merged,
  companion folder empty) gets a drafted companion PR without the operator starting the test
  author; a human still reviews and merges it and approves every run. On kind or minikube the
  test author can run as its own Job in its own namespace, so strict-style isolation is
  available before 011.
- **Scope (in):** the two lines moved from 018 on 2026-11-19 (their text stays in 018, struck
  through): auto-draft, off by default, triggered by a GitHub Actions workflow on the target's
  merge (companion-only token), by the pilot on the laptop, or by the operator (011); and the
  test-author Job with its own egress role (C-07) and no storage shared with workers (C-08).
- **Scope (out):** auto-merging any PR (C-04); the test author launching or steering workers.
- **Depends on:** 018
- **Governed by:** C-04, C-07, C-08, C-11, C-12, C-14
- **Spec dir:** specs/024-test-author-automation/
- **Notes:** Owner decision 2026-11-19, from 018's debrief: split out so 018 is verified on what
  its spec delivered. 021 (step mode) depends on it. **Laptop mode first (owner, 2026-11-19):**
  auto-draft must work in laptop mode, because the laptop release (013) ships with it; the
  operator trigger waits for 011. **Specced 2026-11-19 (owner):** the pilot on the laptop is the
  trigger (`PILOT_AUTHOR_AUTO=on`), the author runs as its own Job on the local cluster with
  model-only egress and no repository credential, and its harness is OpenCode; the GitHub
  Actions trigger comes later.
  **Verified 2026-11-21 (owner):** drafts for Stockroom 009 (`author watch`) and 010 (step
  mode) in the sandboxed author Job; each passed `author check` and was merged by the owner; 010
  verified with them, audit 11/11 after `down`. Only the laptop trigger is built; the GitHub
  Actions trigger is later work, and the operator trigger is 011's. Evidence:
  specs/024-test-author-automation/evidence/.

### 025 — Integration tests in CI  [status: verified]

- **Description:** Run the pilot's real pieces together in CI, on a kind cluster, so seams
  between commands, templates, checks, processes and GitHub are tested before a live run finds
  them.
- **Outcome:** a CI job (weekly, and on PRs that touch the pilot's Go code, manifests or images)
  stands up kind with `pilot up --local-build` and runs, with a stub harness instead of a model:
  - the fixture run to its first PR;
  - `author watch --once`;
  - `step --once` against a test target, with their isolation checks, evidence and audit rows.

  A broken seam fails the job. The glue packages get a coverage floor that only rises.
- **Scope (in):**
  - a stub harness: canned agent output in place of OpenCode, no model key and no cost;
  - GitHub for the tests: a sandbox repository and App, or a fake GitHub API server (decided
    when specified);
  - the scenarios above, and the audit checklist's seams as assertions (template labels found
    by their checks, no terminal in step mode's subprocesses, waits shown between polls);
  - a coverage floor per package for `cli`, `pipeline`, `worker`, `checks`, `cluster` and
    `grader`;
  - from the 021 and 024 debriefs (2026-11-21): live cases for a non-approver's label and a
    failing `doctor` in step mode; the author Job's A01-A05 rows asserted; the known limits (a
    crashed `in-progress` run holding the slot; a changed active feature hiding earlier drafts).
- **Scope (out):** testing models or prompts; OpenShift in CI (with 005); replacing the live
  validation runs, which still close each item.
- **Depends on:** 016, 021, 024
- **Spec dir:** specs/025-ci-integration/
- **Governed by:** C-09, C-10, C-14
- **Notes:** Owner decision 2026-11-19, after the seam audit
  (specs/021-step-mode/audit-20261119.md): decision logic had 84-93% unit coverage and none of the
  six live bugs; the glue had 15-22% and all six. The audit's known limits (a crashed
  `in-progress` run holding the slot) are tested here first.
  **Specced 2026-11-21 (owner):** a fake GitHub API server in this repository (no secrets, also
  for fork PRs), a stub `opencode` in a locally built test worker image, and a GitHub-hosted
  runner. Test endpoints are refused unless `PILOT_TEST_MODE=1`, and a test run's audit says
  it is not evidence.
  **Verified 2026-11-22 (owner, after the debrief and its follow-ups):** evidence
  [specs/025-ci-integration/evidence/README.md](../../specs/025-ci-integration/evidence/README.md),
  debrief
  [roadmap-reviews/debrief-20261122T185418Z.md](../../specs/025-ci-integration/roadmap-reviews/debrief-20261122T185418Z.md)
  (PROCEED WITH UPDATES, no must-address findings).
  - **Delivered (beyond the outcome's wording):** the fixture run goes to its end (status PR,
    B, audit), not only its first PR; there are four scenarios (fixture, automatic drafting,
    step mode, and step mode's gaps); every test run ends `failed` because AU12 marks it a test
    run, by design.
  - **SC-001:** five of the six live bugs, each put back, fail their scenario. #124 (a run's
    last lines after it ends) is covered by a unit test; the scenario's relay assertion found a
    related bug instead (see below).
  - **SC-002:** `integration` passed on GitHub-hosted runners on #140, #146, #147, #155 and on
    `main` (workflow_dispatch, 2026-11-22), in under 20 minutes, with no secrets.
  - **SC-004:** unit-coverage floors in `.github/coverage-floor.txt`; the scenarios run 33-79%
    of the glue packages (15-23% by unit tests), reported in every integration run.
  - **Debrief follow-ups:** F1 fixed (automatic drafts made under an earlier Spec Kit feature
    still count); F2 asserted (a long-running `pilot step` must relay a run's end); F3 kept as
    policy (a floor is never lowered without saying why in the PR; not a check).
  - **Pilot fixes the scenarios found:** `pilot up` records the model host (A03 failed before
    any run); doctor accepts a local build and no longer claims ungraded hidden tests pass;
    step's `next:` line names the owner's move in four more cases; step keeps relaying a run
    after its item is verified; the status-change commit says why it failed without a git
    identity; the 024 docs match the output.

### 026 — Slice runs  [status: planned]

- **Description:** Let a run build one slice of a roadmap item (one user-story phase of its
  approved `tasks.md`) instead of the whole item. The roadmap item stays the unit that is
  verified and that unblocks dependents.
- **Outcome:** `pilot run --target <t> --item NNN --slice <phase-id>` builds, grades and
  publishes one slice under every existing guarantee. Running without `--slice` builds the whole
  item, exactly as today. The item is proposed `verified` once, after its last slice merges.
- **Scope (in):**
  - **one model, not two modes:** an item has one or more slices, and by default one slice is
    the whole item;
  - **where slices come from:** the phase headings of the item's approved `tasks.md`. The human
    who approves the tasks decides where to split. The pilot and agents never re-slice. The
    plan's risk level may require slicing (for example "high risk: one story per slice");
  - **gate for a slice (fails closed):**
    - the item's dependencies are verified (unchanged);
    - earlier slices are merged. Slices run in `tasks.md` order unless the tasks mark them
      independent;
    - this slice's tasks are unticked;
    - an item with open slices is not `ALREADY_VERIFIED`;
  - **allowed paths:** per slice when the spec gives them (for example
    `## Allowed paths (US2)`), otherwise the item's;
  - **grading:**
    - only this slice's tasks may be ticked;
    - it runs this slice's hidden tests (`items/NNN/<slice>/` in the companion), every
      earlier slice's hidden tests as a regression check, and the project's test command;
    - G6 and G7 apply per slice;
  - **publishing:** one draft PR per slice, merged by a human, titled
    `feat(NNN): <title> (USn)`;
  - **status:**
    - the `verified` proposal (status-change PR or terminal approval) comes only after the
      last slice merges;
    - an item may declare an item-level acceptance check (for example a human live check)
      that must be recorded first;
    - the pilot writes no intermediate roadmap status;
  - **after verification:** gaps found after an item is verified never add slices to it. They
    need a new item, or an explicit reopen that a human approves;
  - **evidence and audit:** the run evidence records the item and the slice, and the item's
    audit covers all its slice runs (today AU01–AU03 assume one run per item);
  - **interactions:**
    - 024 drafts hidden tests per slice, and is triggered when a slice is ready but has no
      tests;
    - in 021, a proposed item runs its slices in order, waiting for the human merge between
      them, and each slice needs its own mode approval;
    - in 015, slices of one item stay sequential inside its unit.
- **Scope (out):**
  - running slices or `[P]` tasks of one item in parallel (Q12, 015);
  - the pilot choosing or changing slice boundaries;
  - a roadmap status between `specced` and `verified` written by the pilot.
- **Depends on:** 014, 021, 024
- **Spec dir:** specs/026-slice-runs/
- **Governed by:** C-01, C-02, C-04, C-12, C-14
- **Addresses:** docs/prd.md F1, F8
- **Notes:** Owner decision 2026-11-21 (Tosin Akinosho), from issue #142. It ships with the
  laptop release (013 depends on it), because real projects have items with several stories of
  different risk. Without slices, operators accept large PRs that are hard to review, or split
  stories into roadmap items and so change what "done" means for dependents.
  - **Tick state from spec-status (owner decision 2026-11-21):** per-slice tick state comes from
    spec-status, which will report per-phase task counts. 026 also depends on that spec-status
    release (not a roadmap item, so it isn't listed in `Depends on`). The pilot doesn't parse `tasks.md`
    phases itself, so C-01 is unchanged. ~~spec-status 0.3.0 (pinned 2026-11-21) still reports
    only item-level `tasks.percent`.~~ **2026-11-22:** spec-status 0.4.0 shipped
    `tasks.phases` (spec-status#66 closed); PR #154 pins it, so 026 is no longer waiting on it.
    Plan: `specs/026-slice-runs/plan.md`.
  - **Code today assumes one run per item** (found 2026-11-21):
    - `pilot run` flags;
    - G3 and reconcile need `tasks.percent == 100`;
    - the companion folder is `items/NNN`;
    - readiness `OPEN_ITEM_PR`;
    - AU01–AU03;
    - step mode's item keys and the test author's item keys.
  - ~~**Open:**~~ **Decided in the spec's clarifications (owner, 2026-11-21):**
    - ~~slice ids for phases that aren't user stories (Setup, Foundational, Polish)~~ they join
      story slices: earlier phases join the first story's slice, Polish joins the last; ids stay
      `US1`, `US2`, …;
    - ~~how `tasks.md` marks slices independent~~ not in this version: slices are always
      sequential;
    - ~~exactly how G6 and G7 apply per slice~~ per slice, against the base the slice starts from
      (spec default, FR-014).

## Open Questions

- Q1 — Final project name and repo location (new repo vs. inside the spec-status
  org). Interim home: GitHub repo tosin2013/specfoundry (public since 2026-11-13, Apache-2.0,
  with CONTRIBUTING and a Code of Conduct).
  **Planned 2026-11-21 (owner):** the product name is Chalkeus (Q7). After 1.28.0 merges, a
  single follow-up change updates the Go module path, imports, docs, CI badges and GitHub
  App references, then the GitHub repo is renamed to tosin2013/chalkeus in settings (the
  owner does that, or gives an explicit go-ahead). Both happen before 013 ships. GitHub
  redirects the old URLs. This revision does not rename the repo or change the code. The
  "new repo vs. inside the spec-status org" options are not the plan; the repo stays under
  tosin2013 and is renamed in place.
- Q2 — Kelos as wrapped dependency vs. fork (default: wrap, see C-05). **Resolved
  2026-11-13:** never fork; wrap only if the security bar is met by configuration,
  otherwise do not adopt (C-05 amended).
- Q3 — Which machine runs M1 and M2 (Ubuntu box, Mac, another lab machine). **M1 decided
  2026-11-13:** the Ubuntu host (31 GB, 8 CPUs). Still open for M2.
- Q4 — Granularity: one worker run per roadmap entry, or per task or issue. **Decided
  for M1:** one worker run per roadmap item. Revisit in 004, where issues are per task.
  **Decided 2026-11-21 (owner decision, Tosin Akinosho; issue #142):** a run builds one
  *slice* of an item (one phase of its approved `tasks.md`, by default the whole item), and the
  item stays the unit that is verified and unblocks dependents. See 026.
- Q5 — Worker model provider and spend cap defaults. **Partly decided 2026-11-13:**
  harness OpenCode 2.2.7; provider is operator configuration (not named in docs); caps
  default to $1 per attempt with a 2M-token fallback when the provider reports no cost.
  Still open: provider choice for shared use, and USD accounting (declaring model
  prices).
  **Amended 2026-11-18 (owner):** OpenCode's own services (for example OpenCode Go) may be
  named in the docs as a quick start, because OpenCode is already the harness; other providers
  stay operator configuration. `pilot import-opencode --from-login` imports an OpenCode API-key
  login (never a subscription OAuth login), and `pilot doctor` checks the model key with one tiny
  request before any run.
- Q6 — Which spec delivers operator observability (C-10): metrics, traces and events for
  gate decisions, attempts, spend and run duration to an operator-run collector? Candidates:
  a new entry after 003, or folded into 004/005. **Resolved 2026-11-13:** entry 007.
- Q7 — Product name and CLI interface: `pilot` is a prototype name, and the CLI becomes
  the interface for humans and LLM agents in both modes (C-11). Decide the name and the
  stable command surface before 011 (raised 2026-11-14).
  **2026-11-19:** the name also sets 021's proposal label (a profile setting with a placeholder
  until then). The `specfoundry` GitHub organization name is taken.
  **Resolved 2026-11-21 (owner, 1.27.0):** brand SpecFoundry; installed binary `specfoundry`,
  with `pilot` as a subcommand (`specfoundry pilot …`); tab completion on install (013);
  proposal label default `specfoundry:proposed` (021). Why not bare `pilot`: an existing AI
  ticket-to-PR tool already ships a `pilot` CLI; why not `sf`: the Salesforce CLI's binary
  (research return MSG-2115). Not a trademark clearance: the `specfoundry` GitHub org is a
  public org of unknown control, and another project uses the exact name SpecFoundry. Releases
  come from tosin2013/specfoundry for now. Short alias: `sfy`, an opt-in shell alias the
  installer offers, not a binary. Docs, harness rules and agents always use `specfoundry`.
  `sfy` over `sfd`: an existing `sfd` CLI wraps the Salesforce CLI (next to the `sf`
  clash), while the `sfy` uses found were small and unrelated.
  **Re-resolved 2026-11-21 (owner, 1.28.0, superseding 1.27.0):** brand Chalkeus (Greek
  χαλκεύς, "smith"); installed binary `chalkeus`, with `forge` as the main subcommand
  (`chalkeus forge …`); `pilot` is renamed to `forge` to fit the smith theme; tab
  completion on install (013); proposal label default
  `chalkeus:proposed` (021). Name check: no US trademark and no npm, PyPI, crates, Go,
  Homebrew, Docker Hub or Artifact Hub clash; chalkeus.com has been parked since 2015; a
  Polish metalworking firm trades as Chalkeus; EU and WIPO were unchecked. The owner
  accepts that risk and will revisit if the project gets very popular. Short alias: `chk`,
  an opt-in shell alias the installer offers, not a binary. `chk` is not a common Unix
  command, not an exact Debian, Ubuntu or Homebrew-core package, and the npm package `chk`
  has no bin; the uses found (a small third-party Homebrew kubectl version switcher, and a
  Windows GUI hash tool) were small and unrelated. Docs, harness rules and agents always
  use `chalkeus`. The repo stays tosin2013/specfoundry until the planned rename in Q1.
  The chalkeus.dev domain is a separate, undecided money call.
- Q8 — Using the pilot across projects and harnesses (raised 2026-11-15, 012 debrief).
  - Do harness ask rules live per user, or does a `pilot` command install them?
  - Today's rules match the `pilot/pilot` path; an installed CLI (013) changes the pattern.
    Per Q7 (re-resolved 2026-11-21, 1.28.0, superseding 1.27.0) the installed binary is
    `chalkeus`, so rules match `chalkeus forge …`, not a bare `pilot`.
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
  **2026-11-21:** 026 (slice runs) splits an item into sequential slices, one per `tasks.md`
  phase. Running unblocked `[P]` tasks at the same time is still this question (and 015).
- Q13 — How planning scales (raised 2026-11-16). Under C-12 the limit on throughput is how fast
  humans prepare items (spec, tasks, hidden tests), not how many workers run.
  ~~Open: whether agents may draft specs, tasks or tests for a human to approve, and what
  tooling helps (for example migration analysis reports in brownfield upgrades).~~
  **Tests half resolved 2026-11-17 (owner decision, Tosin Akinosho):** agents may draft hidden
  acceptance tests only through the separate test-author role (018, C-12, C-14). A human reviews
  and merges them into the companion repository. Build workers never write or see them.
  ~~**Still open:** whether agents may draft specs or tasks for a human to approve, and what
  tooling helps (for example migration analysis reports in brownfield upgrades).~~
  **Specs half resolved 2026-11-18 (owner decision):** humans and the **controller** (the
  agent or harness driving the work, for example the session running `pilot` commands) plan
  together: the controller may draft a project's constitution, roadmap, specs and tasks with the
  human, and the human approves and merges them. Build workers still never plan, and the test
  author (018) stays a separate session from the controller (C-14). Still open: what tooling
  helps (for example migration analysis reports in brownfield upgrades).

## Cross-Cutting Notes

- **Delivery order (owner direction 2026-11-16):** laptop first, then OpenShift, in stages, so
  someone on a laptop can trust that the whole loop works end to end before the enterprise case.
  Dependencies are unchanged; this is the order of work among items that are ready.
  1. 016 Go pilot at parity (its T047 owner run also closes 004's superseded validation).
  2. 014 Any project, on kind with the fixture and a real repository; the EAP validation waits.
  3. 017 Greenfield bootstrap, end to end on a laptop (kind or minikube).
  4. 009 Period spend budget (015 depends on it).
  5. 015 Released waves, with a small laptop concurrency limit (for example 2).
  6. 013 Installable pilot release: the first release includes scaling.
  7. 005 OpenShift target (laptop mode).
  8. 014's EAP validation on OpenShift with an in-cluster model.
  9. Then 007, 008, 010 and 011.
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

**Version**: 1.29.2 | **Ratified**: 2026-11-13 | **Last Amended**: 2026-11-22
