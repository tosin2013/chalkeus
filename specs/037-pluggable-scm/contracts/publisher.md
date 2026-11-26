# Contract: the publisher Job on each host (`publish.py`)

The Job keeps its shape: inputs `/in/pr.json` and `/in/change.bundle`, the token from Secret
`scm-token` (`PILOT_SCM_TOKEN`), egress only through the SCM proxy, output a `result.json` frame.

## `pr.json` additions

```json
{
  "scm": {"kind": "gitlab", "api": "https://gitlab.example.com/api/v4",
          "git_base": "https://gitlab.example.com/", "ca_file": "/etc/chalkeus/scm-ca.pem"}
}
```

Absent `scm` means GitHub with today's values (`GITHUB_API`, `GIT_BASE` test overrides kept).
`ca_file` is present only when a CA bundle is set; the bundle is mounted from ConfigMap `scm-ca`.

## Steps per host

| Step | GitHub (unchanged) | GitLab | Bitbucket |
|---|---|---|---|
| Who am I | `GET /user` (skipped for App) | `GET /user` → bot id | skipped (not supported for access tokens); author read from the PR |
| Push | user `x-access-token` | user `oauth2` | user `x-token-auth` |
| Find open request for head | `GET pulls?head=owner:head&state=open` | `GET merge_requests?source_branch=head&state=opened` | `GET pullrequests?q=source.branch.name="head" AND state="OPEN"` (and `draft=true`) |
| Open draft | `POST pulls` `draft:true` | `POST merge_requests` title `Draft: <title>` | `POST pullrequests` `draft:true` |
| Result | `{number,url,head,draft,author,token_login,created}` | same, `author` = bot username, adds `author_id` | same, `author_id` from the PR's `author.uuid` |

Common rules:
- An existing open request for the head is reused (idempotent), as today.
- Git uses `GIT_ASKPASS`; HTTPS uses an `ssl` context with `ca_file` when given; git uses
  `GIT_SSL_CAINFO`.
- The publisher never merges, approves or closes anything.
- Errors print the host's status and message, never the token.

## Read-back on the operator's machine

After the Job, forge reads the request with the operator credential (`scm.Client.ChangeRequest`)
and fails the publish unless: it is a draft, its head is the expected branch, and its author id
equals the result's `author_id`. The author id is recorded as the run's publisher identity
(`publisher_login` for GitHub, unchanged; `publisher_id` for all hosts).

## Tests (`pilot/images/tests/test_publish_hosts.py`)

A local HTTP stub per host checks: the request paths and bodies above, the draft marker, reuse
of an open request, the push username, the CA-file path, and that the token never appears in
output.
