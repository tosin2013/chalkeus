# Quickstart: validating pluggable SCM (037)

Validation scenarios, in delivery order. Each live scenario writes its evidence to
`specs/037-pluggable-scm/evidence/`. Commands run from the repository root as `./chalkeus forge …`;
every approval and merge is the owner's.

## 0. Unit and contract tests (every PR)

```sh
go vet ./... && go test ./...
python3 -m unittest discover -s pilot/images/tests
```

Expect: the shared contract suite (`internal/scm/scmtest`) passes for GitHub (fakegh), GitLab and
Bitbucket (httptest fakes); the no-merge guard covers all three packages; config refusals
(contracts/config.md) each have a test; `test_publish_hosts.py` passes.

## 1. GitHub unchanged (US1, SC-001)

On the Ubuntu box, with no `FORGE_SCM` anywhere:
1. `./chalkeus forge doctor` and `./chalkeus forge doctor --target stockroom`: same rows as before
   plus `scm host: github (github.com)`, `commit author`.
2. The isolation suite: P01–P12 as before, plus P13 passing (publisher reaches github.com,
   refused gitlab.com).
3. The fixture walkthrough: audit 11/11.
4. Compare the evidence with the last GitHub fixture run: only `scm`, `author_id`,
   `merged_by_id` and P13 differ.

## 2. gitlab.com (US2, US3, US6; SC-002, SC-003, SC-004, SC-005)

Prerequisites (owner):
- A throwaway gitlab.com project for the fixture (`<group>/<name>-pilot-fixture`) and one for a
  target, plus its companion, all on gitlab.com.
- For each publisher: a project access token (Premium) or a personal token of a dedicated bot
  account that is a member of only that project (Free), scopes `api` and `write_repository`.
- A test-author token for the companion.
- `GITLAB_TOKEN` set to the owner's own token.

Steps:
1. Profile: `FORGE_SCM=gitlab`, `FORGE_TARGET_REPO=<group>/<project>`, approvers as GitLab user
   ids or usernames. `./chalkeus forge doctor --target <t>`: every row ok.
2. Refusals: put a group access token in the publisher file → doctor fails with
   `SCM_TOKEN_WIDER_THAN_REPO`; add the bot account to a second project → same. Restore.
3. The fixture walkthrough on gitlab.com: draft MRs opened by the bot, the owner merges, audit
   11/11, P13 proves github.com is refused.
4. Hidden tests: the owner's test-author session submits a draft; a draft MR opens on the
   companion from the test-author identity.
5. One target item run: draft MR, owner merges (try squash), reconcile records the squash
   commit and the tree match passes; the status-change MR is merged by an approver; audit 11/11.
6. Status-change refusals (one run each, or crafted MRs against the check): merged by the
   publisher's bot, merged by a non-approver, an extra changed line. Each refused with its
   reason code.
7. Grep the run's evidence for the token prefix (`glpat-`): no match in worker or grader
   evidence.

## 3. Self-managed GitLab CE (US4; SC-002, SC-003)

Setup (on the Ubuntu box, torn down afterwards):
1. Make a CA and a server certificate for a host name that resolves both on the host and inside
   the kind cluster (for example `gitlab.<host-ip>.nip.io`).
2. Run GitLab CE at a pinned version (≥ 18.0) in a container with that certificate; about 4 GB
   of memory.
3. Create the fixture, target and companion projects, a project access token (any license) and
   the owner's user token.

Steps:
1. Profile: `FORGE_SCM=gitlab`, `FORGE_SCM_HOST=<host>`, `FORGE_SCM_CA_BUNDLE=gitlab-ca.pem`.
   doctor: `scm host` shows the version, `scm ca bundle` ok.
2. Remove the bundle setting → doctor fails naming `FORGE_SCM_CA_BUNDLE`. Restore.
3. The fixture walkthrough and one target item run, as in scenario 2 (fast-forward merge this
   time); P13 proves gitlab.com and github.com are refused.
4. Tear down the container; record its version and the certificate fingerprints (not keys) in
   the evidence.

## 4. Bitbucket Cloud (US5, US6; SC-002 to SC-005)

Prerequisites (owner): a throwaway workspace or repositories for fixture, target and companion;
a repository access token per publisher (scopes `repository:write`, `pullrequest:write`); a
`restrict_merges` branch restriction on each target branch; `BITBUCKET_TOKEN` (the owner's
Atlassian API token, with repository admin on those repositories); approvers as account UUIDs.

Steps:
1. doctor: every row ok. Remove the `restrict_merges` restriction → `branch protection` fails.
   Restore.
2. Put a workspace access token in the publisher file → `SCM_TOKEN_WIDER_THAN_REPO` (confirms the
   listing probe, research R4).
3. The fixture walkthrough: draft PRs (confirm the open-PR lookup finds drafts, research R3),
   audit 11/11.
4. Hidden tests by a draft PR on the companion; one target item run with the status change;
   audit 11/11.
5. The status-change refusals, as in scenario 2.

## 5. Docs (US7, SC-006)

Follow only `docs/targets.md` to set up a fresh GitLab target and a fresh Bitbucket target;
doctor passes before the first run. Each refusal in contracts/config.md and contracts/checks.md
has a troubleshooting entry.
