package ghapp

import (
	"fmt"
	"strconv"
	"time"

	"github.com/tosin2013/chalkeus/internal/ghloop"
	"github.com/tosin2013/chalkeus/internal/secret"
)

// The 029 triage report's permissions (research R1): read only, the required ones first. An
// optional one the installation wasn't granted is left out of the request (GitHub refuses a
// token request that names a permission the App lacks) and reported as missing.
var (
	ReportRequired = []string{"contents", "pull_requests"}
	ReportOptional = []string{"checks", "statuses", "vulnerability_alerts"}
)

// ReadToken mints an installation token for repo with read permissions only, from the
// installation's granted set (029 FR-008). The token is returned in memory and never written
// anywhere. perms maps each report permission to "granted" or "missing". It fails when a
// required permission isn't granted.
func ReadToken(keyPath, appID, installationID, repo string, now time.Time) (tok secret.Value, perms map[string]string, err error) {
	key, err := LoadKey(keyPath)
	var j string
	if err == nil {
		j, err = JWT(key, appID, now)
	}
	if err != nil {
		return tok, nil, fmt.Errorf("cannot sign with the App key; is it the App's private key?")
	}
	inst, err := strconv.Atoi(installationID)
	if err != nil {
		return tok, nil, fmt.Errorf("FORGE_APP_INSTALLATION_ID must be numeric")
	}
	st, info := call("GET", fmt.Sprintf("/app/installations/%d", inst), j, nil)
	if st != 200 {
		return tok, nil, fmt.Errorf("GitHub rejected the App JWT or installation (HTTP %d): check FORGE_APP_ID, FORGE_APP_INSTALLATION_ID and the key", st)
	}
	has, _ := info["permissions"].(map[string]any)
	perms = map[string]string{}
	req := map[string]string{}
	for _, p := range ReportRequired {
		if has[p] != "read" && has[p] != "write" {
			return tok, nil, fmt.Errorf("the GitHub App lacks %s: read; grant it in the App's settings (Permissions)", p)
		}
		req[p] = "read"
	}
	for _, p := range ReportOptional {
		if has[p] == "read" || has[p] == "write" {
			req[p] = "read"
			perms[p] = "granted"
		} else {
			perms[p] = "missing"
		}
	}
	_, name := ghloop.SplitRepo(repo)
	st, out := call("POST", fmt.Sprintf("/app/installations/%d/access_tokens", inst), j,
		map[string]any{"repositories": []string{name}, "permissions": req})
	if st != 201 {
		return tok, nil, fmt.Errorf("cannot mint a read-only token for %s (HTTP %d): is the App installed on that repository?", repo, st)
	}
	got, _ := out["permissions"].(map[string]any)
	for p := range req {
		if got[p] != "read" {
			return tok, nil, fmt.Errorf("the read-only token for %s came back with %s: %v, expected read", repo, p, got[p])
		}
	}
	for p, v := range got {
		if v != "read" && p != "metadata" {
			return tok, nil, fmt.Errorf("the read-only token for %s carries %s: %v; refusing to use it", repo, p, v)
		}
	}
	value, _ := out["token"].(string)
	if value == "" {
		return tok, nil, fmt.Errorf("GitHub returned no token for %s", repo)
	}
	return secret.New(value), perms, nil
}
