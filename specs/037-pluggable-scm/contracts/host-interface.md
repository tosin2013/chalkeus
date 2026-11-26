# Contract: the host interface (`internal/scm`)

One interface, three implementations (`internal/scm/github`, `internal/scm/gitlab`,
`internal/scm/bitbucket`). Types are in data-model.md. Every method takes a `context.Context`.
Errors are `*scm.Error{Code, Msg, Status}`; `Code` is one of `not_found`, `forbidden`,
`unauthorized`, `rate_limited`, `conflict`, `invalid`, `unavailable`, and `Msg` never contains a
token.

```go
type Client interface {
    Host() Host
    Whoami(ctx) (Account, error)
    Repository(ctx, repo string) (RepoInfo, error)          // default branch, whether the caller can push
    ResolveAccount(ctx, given string) (Account, error)      // username or id → Account
    Permission(ctx, repo string, a Account) (Permission, error) // none|read|write|admin, effective
    BranchProtection(ctx, repo, branch string) (Protection, error)

    OpenChangeRequests(ctx, repo, headPrefix string) ([]ChangeRequest, error)
    ChangeRequest(ctx, repo string, id int) (ChangeRequest, error)
    ChangedFiles(ctx, repo string, cr ChangeRequest) ([]string, error)
    FileAt(ctx, repo, rev, path string) ([]byte, error)

    CommitFiles(ctx, repo, base, branch string, files []File, message string) (string, error) // companion only
    OpenDraft(ctx, repo, head, base, title, body string) (ChangeRequest, error)               // companion only
    CloseChangeRequest(ctx, repo string, id int, comment string) error                        // reset-fixture only

    ProbePublisher(ctx, repo string) TokenProbeResult // the publisher credential's check (R4)
}
```

`Protection` reports: protected or not, who may merge (roles or ids), whether force-push is
allowed, and for Bitbucket whether a `restrict_merges` restriction exists.

## Per-host mapping

| Method | GitHub | GitLab (`/api/v4`) | Bitbucket (`/2.0`) |
|---|---|---|---|
| Whoami | `GET /user` | `GET /user` | `GET /user` (operator API token only) |
| Repository | `GET /repos/{o}/{n}` | `GET /projects/:path` | `GET /repositories/{ws}/{repo}` |
| ResolveAccount | login | `GET /users?username=` or `GET /users/:id` | `GET /users/{uuid}` |
| Permission | `collaborators/{u}/permission` | `GET /projects/:id/members/all/:user_id` | `GET /workspaces/{ws}/permissions/repositories/{repo}?q=user.uuid=` (needs repo admin) |
| BranchProtection | `branches/{b}/protection` | `GET /projects/:id/protected_branches/:name` | `GET /repositories/{ws}/{repo}/branch-restrictions` |
| OpenChangeRequests | `GET pulls?state=open` | `GET merge_requests?state=opened&source_branch=` | `GET pullrequests?q=state="OPEN"` (with and without `draft=true`, R3) |
| ChangeRequest | `GET pulls/{n}` | `GET merge_requests/:iid` | `GET pullrequests/{id}` |
| ChangedFiles | `GET pulls/{n}/files` | `GET merge_requests/:iid/diffs` | `GET diffstat/{dest}..{src}` |
| FileAt | `GET contents/{path}?ref=` | `GET repository/files/:path/raw?ref=` | `GET src/{rev}/{path}` |
| CommitFiles | Git Data API | `POST repository/commits` (actions) | `POST src` (multipart) |
| OpenDraft | `POST pulls` `draft:true` | `POST merge_requests` title `Draft: …` | `POST pullrequests` `draft:true` |
| CloseChangeRequest | comment, `PATCH state=closed`, delete ref | note, `PUT state_event=close`, delete branch | `POST pullrequests/{id}/decline` |
| ProbePublisher | `ghloop.Probe` (unchanged) | `personal_access_tokens/self`, `/self/associations`, `/user` | `/user` refused, repo read, workspace listing |

## Rules

- No implementation calls a merge, approve or accept endpoint (no-merge guard, R12).
- Reads on the operator's machine use the operator credential (R8); `ProbePublisher` uses the
  publisher credential and only reads (GitHub's blob probe creates an unreferenced blob, as today).
- Pagination is followed to the end or a cap (500 items), and a capped result says so.
- A host's rate-limit response returns `rate_limited`; callers stop, they don't retry in a loop.
- GitHub's implementation reuses today's code and keeps today's evidence values.

## Credential helpers

- `scm.OperatorCredential(Host) (secret.Value, error)`: R8 order; error names the env var to set.
- `scm.GitEnv(Host, secret.Value) []string`: environment for host-side git: GitHub the `gh`
  credential helper (unchanged); GitLab and Bitbucket `GIT_ASKPASS=<chalkeus> askpass` with the
  username and token in the child environment only, plus `GIT_SSL_CAINFO` when a CA bundle is set.
- `scm.RepoURL(Host, repo)`: `https://<host>/<repo>.git`.
- `scm.ChangeRequestURLPrefix(Host, repo)`: for AU03 (R12).
