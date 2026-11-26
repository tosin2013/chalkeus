# Contract: host settings in target profiles and forge.env

## Keys

| Key | Where | Values | Default |
|---|---|---|---|
| `FORGE_SCM` | target profile, `forge.env` (fixture) | `github`, `gitlab`, `bitbucket` | `github` |
| `FORGE_SCM_HOST` | same | a host name (`gitlab.example.com`), GitLab only | `gitlab.com` for GitLab |
| `FORGE_SCM_CA_BUNDLE` | same | file name in the config folder (`gitlab-ca.pem`), GitLab only | none |
| `FORGE_TARGET_REPO`, `FORGE_COMPANION_REPO`, `FORGE_FIXTURE_REPO` | as today | per-host form (data-model.md) | — |
| `FORGE_APPROVERS` | as today | per-host ids (data-model.md) | — |

Names are read as `PILOT_*` internally (`internal/names`), as for every setting since 027.

## Refusals at load (exit 2, message names the key)

- `FORGE_SCM` not one of the three values.
- `FORGE_SCM_HOST` or `FORGE_SCM_CA_BUNDLE` set with `FORGE_SCM` other than `gitlab`.
- `FORGE_SCM_HOST` not a bare host name (no scheme, path or port; port allowed as `host:port`).
- `FORGE_SCM_CA_BUNDLE` missing, not mode 0600, not a PEM certificate, or outside the config folder.
- A repository not in the host's form.
- `FORGE_COMPANION_REPO` set and the companion on a different host (one host per target).
- With `FORGE_SCM` other than `github`: `FORGE_GITHUB_ISSUES=on`, `FORGE_GITHUB_APPROVALS=on`,
  `FORGE_PUBLISHER_CREDENTIAL=app`. Message: "<setting> is GitHub-only in this release; the
  loop on other hosts is roadmap 038".
- Bitbucket: an approver that is neither `{uuid}` nor an `account_id`.

## Commands refused on a non-GitHub target (exit 2)

`forge debt` (reads Dependabot), `forge github sync|drift|watch`, `forge step` (labels and
issues), `forge app-key`, `forge app-token`. Message: "<command> is GitHub-only in this release".

## Files

- `targets/<name>.publisher.env`: `PILOT_SCM_TOKEN=<token for the target's host>` (unchanged name).
- `targets/<name>.author.env`: `PILOT_AUTHOR_TOKEN=<test-author token for the host>` (unchanged).
- The CA bundle file named by `FORGE_SCM_CA_BUNDLE`.

Operator credentials are never stored by forge: `GH_TOKEN`/`gh`, `GITLAB_TOKEN`/`glab`,
`BITBUCKET_TOKEN` (research R8).

## Unchanged

With no `FORGE_SCM` anywhere, every existing profile loads exactly as today, and every existing
refusal keeps its wording.
