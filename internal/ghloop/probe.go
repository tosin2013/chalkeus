package ghloop

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tosin2013/specfoundry/internal/config"
	"github.com/tosin2013/specfoundry/internal/secret"
)

// The publisher credential's scope probe (reference: pilot/images/tools/check_token.py).
// Write access is tested with side-effect-free probes:
//   - contents: POST an unreferenced git blob (invisible, garbage-collected);
//   - issues (004): POST a label with an invalid colour; GitHub rejects it with 422 *after* the
//     permission check, so 422 proves write access and nothing is created.
// GET /repos `permissions` is not used: for a public repo it shows the account's rights, not a
// fine-grained token's.

// API is the GitHub API base (tests point it at an httptest server).
var API = "https://api.github.com"

var (
	blobProbe   = map[string]string{"content": "specfoundry publisher token scope probe (unreferenced blob)", "encoding": "utf-8"}
	issuesProbe = map[string]string{"name": "specfoundry-probe", "color": "not-a-color"}
)

// ProbeResult is the probe outcome (the reference's JSON line).
type ProbeResult struct {
	OK          bool     `json:"ok"`
	Login       *string  `json:"login"`
	Repo        string   `json:"repo"`
	RepoPush    bool     `json:"repo_push"`
	IssuesWrite *bool    `json:"issues_write"`
	Canary      *string  `json:"canary"`
	CanaryPush  *bool    `json:"canary_push"`
	Expires     *string  `json:"expires"`
	Problems    []string `json:"problems"`
}

// Credential is a publisher credential loaded from an env file.
type Credential struct {
	Token secret.Value
	Kind  string // "" (fine-grained token) or "app"
	Login string // set for App tokens (PILOT_SCM_LOGIN)
}

// LoadCredential reads PILOT_SCM_TOKEN (and, for App tokens, PILOT_SCM_KIND/LOGIN) from an env file.
func LoadCredential(envFile string) (Credential, error) {
	if err := config.CheckMode(envFile); err != nil {
		return Credential{}, err
	}
	raw, err := os.ReadFile(envFile)
	if err != nil {
		return Credential{}, err
	}
	var c Credential
	for _, line := range strings.Split(string(raw), "\n") {
		k, v, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok || strings.HasPrefix(k, "#") {
			continue
		}
		switch k {
		case "PILOT_SCM_TOKEN":
			c.Token = secret.New(v)
		case "PILOT_SCM_KIND":
			c.Kind = v
		case "PILOT_SCM_LOGIN":
			c.Login = v
		}
	}
	if c.Token.Empty() {
		return c, fmt.Errorf("PILOT_SCM_TOKEN missing in %s", filepath.Base(envFile))
	}
	return c, nil
}

func call(token secret.Value, method, path string, body any) (int, map[string]any, http.Header) {
	var r io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		r = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, API+path, r)
	if err != nil {
		return 0, nil, nil
	}
	req.Header.Set("Authorization", "Bearer "+token.Reveal())
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "specfoundry-pilot-token-check")
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return 0, nil, nil
	}
	defer func() { _ = resp.Body.Close() }()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out, resp.Header
}

func sptr(s string) *string { return &s }
func bptr(b bool) *bool     { return &b }

// Probe checks a credential against the fixture repo (and the canary, which must not be
// writable). issues adds the Issues-write probe (004).
func Probe(c Credential, repo, canary string, issues bool) ProbeResult {
	res := ProbeResult{Repo: repo, Problems: []string{}}
	if c.Kind == "app" { // 004: App installation tokens cannot call GET /user
		res.Login = sptr(c.Login)
		res.Expires = sptr("1 hour (GitHub App installation token)")
	} else {
		st, user, hdr := call(c.Token, "GET", "/user", nil)
		if l, ok := user["login"].(string); ok {
			res.Login = sptr(l)
		}
		if st != 200 {
			res.Problems = append(res.Problems, fmt.Sprintf("token rejected by GitHub (HTTP %d)", st))
		}
		if e := hdr.Get("Github-Authentication-Token-Expiration"); e != "" {
			res.Expires = sptr(e)
		}
	}
	st, _, _ := call(c.Token, "GET", "/repos/"+repo, nil)
	if st != 200 {
		res.Problems = append(res.Problems, fmt.Sprintf("cannot read %s (HTTP %d)", repo, st))
	}
	stw, _, _ := call(c.Token, "POST", "/repos/"+repo+"/git/blobs", blobProbe)
	res.RepoPush = stw == 201
	if st == 200 && !res.RepoPush {
		res.Problems = append(res.Problems, fmt.Sprintf("no write access to %s (write probe HTTP %d; needs Contents: read and write)", repo, stw))
	}
	if stp, _, _ := call(c.Token, "GET", "/repos/"+repo+"/pulls?state=all&per_page=1", nil); stp != 200 {
		res.Problems = append(res.Problems, fmt.Sprintf("cannot list pull requests on %s (needs Pull requests: read and write)", repo))
	}
	if issues {
		sti, _, _ := call(c.Token, "POST", "/repos/"+repo+"/labels", issuesProbe)
		res.IssuesWrite = bptr(sti == 422)
		if sti != 422 {
			res.Problems = append(res.Problems, fmt.Sprintf("no issue write access to %s (probe HTTP %d; needs Issues: read and write)", repo, sti))
		}
	}
	if canary != "" {
		res.Canary = sptr(canary)
		stc, _, _ := call(c.Token, "POST", "/repos/"+canary+"/git/blobs", blobProbe)
		res.CanaryPush = bptr(stc == 201)
		if stc == 201 {
			res.Problems = append(res.Problems, fmt.Sprintf("token can write %s: scope it to the fixture repo only", canary))
		}
	}
	res.OK = len(res.Problems) == 0
	return res
}

func pyVal(v any) string {
	switch x := v.(type) {
	case *string:
		if x == nil {
			return "None"
		}
		return *x
	case *bool:
		if x == nil {
			return "None"
		}
		if *x {
			return "True"
		}
		return "False"
	case bool:
		if x {
			return "True"
		}
		return "False"
	}
	return fmt.Sprint(v)
}

// Report prints the result the way the reference publisher_token.report does.
func Report(w io.Writer, r ProbeResult) {
	status := "NOT OK"
	if r.OK {
		status = "OK"
	}
	fmt.Fprintf(w, "publisher token: %s\n", status)
	fmt.Fprintf(w, "  login: %s\n", pyVal(r.Login))
	fmt.Fprintf(w, "  repo: %s\n", r.Repo)
	fmt.Fprintf(w, "  repo_push: %s\n", pyVal(r.RepoPush))
	fmt.Fprintf(w, "  canary: %s\n", pyVal(r.Canary))
	fmt.Fprintf(w, "  canary_push: %s\n", pyVal(r.CanaryPush))
	fmt.Fprintf(w, "  expires: %s\n", pyVal(r.Expires))
	for _, p := range r.Problems {
		fmt.Fprintf(w, "  problem: %s\n", p)
	}
}

// VerifyFile loads the credential in envFile and probes it (the reference publisher_token.verify).
func VerifyFile(envFile, repo, canary string, issues bool) ProbeResult {
	c, err := LoadCredential(envFile)
	if err != nil {
		return ProbeResult{Repo: repo, Problems: []string{err.Error()}}
	}
	return Probe(c, repo, canary, issues)
}
