# Dependencies of the Go pilot

The pilot keeps its trust base small (constitution IV): everything not listed here is the Go
standard library. Each dependency below is pinned in `go.sum`, checked by `govulncheck` in CI,
and updated through Dependabot PRs that a human reviews and merges.

| Module | Why it's needed |
|---|---|
| `github.com/spf13/cobra` (with `github.com/spf13/pflag`) | the command line: subcommands, flags, help |
| `k8s.io/client-go`, `k8s.io/api`, `k8s.io/apimachinery` | talking to the cluster through your kubeconfig: apply, jobs, logs, exec, secrets, access reviews (replaces `kubectl` subprocesses) |
| `sigs.k8s.io/yaml` | reading the pilot's YAML manifests |

More entries are added as the port reaches them (spec 016): kind's library, the GitHub API client and
`golang.org/x/term`. Each one is listed here with its reason, in
the same change that adds it.

## Programs the pilot runs

| Program | Why |
|---|---|
| `git` | clone, bundle and push the fixture repository (bundles aren't well supported by Go git libraries) |
| `docker` | the local registry and teardown checks; kind itself needs it |
| `spec` ([spec-status](https://github.com/tosin2013/spec-status)) | the gate, until spec-status offers a public Go package |
