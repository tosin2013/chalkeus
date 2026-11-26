# Contract: doctor rows, the P13 live check and audit changes

## doctor (targets on any host)

New or changed rows in the target area. Status `ok`, `warn` or `fail`; any `fail` makes doctor
exit 1.

| Row | ok when | otherwise |
|---|---|---|
| `scm host` | host kind and name shown; self-managed GitLab version ≥ 18.0 (`GET /version`) | fail: "GitLab <v> is older than 18.0" |
| `scm ca bundle` | (GitLab with a bundle) a TLS connection to the host succeeds with it | fail, names `FORGE_SCM_CA_BUNDLE`; a certificate error without a bundle names the setting too |
| `operator credential` | resolved per R8 and `Whoami` succeeds | fail: names `GH_TOKEN`/`gh`, `GITLAB_TOKEN`/`glab`, or `BITBUCKET_TOKEN` |
| `publisher credential` | `ProbePublisher` passes: right kind, one repository (the target), scopes, not expired | fail with the probe's code (data-model.md) |
| `approvers` | every approver resolves to an id and has write access to the target | fail per approver; Bitbucket without repo admin on the operator credential: fail "can't verify approvers' write access: the operator credential needs repository admin" |
| `branch protection` | the target branch is protected; Bitbucket: a `restrict_merges` restriction exists | GitHub and GitLab: warn (as today); Bitbucket without `restrict_merges`: fail "the publisher's token can merge without a restrict_merges restriction" |
| `commit author` | the publisher's commit email is a valid address | warn: "some hosting and CI integrations refuse commits from an invalid address (<email>)" |
| `github-only settings` | none requested on a non-GitHub host | fail (contracts/config.md) |

GitHub targets show the same rows as today plus `scm host` and `commit author`.

## P13: the publisher reaches only its host

Runs with the other live checks (P01–P12), from a short-lived pod in `pilot-publish` with the
publisher's NetworkPolicy and proxy settings, holding no credential.

- **Pass:** an HTTPS request to the run's host (`https://<host>/`) through the SCM proxy gets an
  HTTP response, and a request to the other probe host (`gitlab.com` on a GitHub run,
  `github.com` otherwise) is refused by the proxy (403 or connect failure), and a direct
  connection without the proxy fails.
- **Fail:** names which request behaved wrongly.
- Recorded in `checks.json` like P01–P12; diagnosis (`-vv`) shows the proxy's refusal line.

**Amended 2026-11-26 (implementation):** AU04 also requires P13 to pass whenever the run recorded
it (runs from before 037 have none), so a publisher that could reach another Git host fails the
audit. AU04's description text is unchanged (the golden records compare it). The audit stays at
11 checks.

## Audit

- **AU03** (item and slice): the PR URL must match the run's host form
  (`scm.ChangeRequestURLPrefix`): `github.com/<repo>/pull/`, `<host>/<path>/-/merge_requests/`,
  `bitbucket.org/<ws>/<repo>/pull-requests/`. Every PR is a draft and its author id equals the
  recorded publisher id, on every host.
- Fixture AU03: as today, plus the author id rule.
- Other audit checks: unchanged.

## Secret heuristics

`scmRE` (P04, P09) and `scmNameRE` (model folder) also match `GITLAB`, `BITBUCKET` and `GLPAT`.
