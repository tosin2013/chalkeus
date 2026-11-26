// Package ghloop is the GitHub side of the pilot: the operator's read client, the publisher's
// write client, the 004 GitHub loop (sync, drift, approvals) and the write probe
// (016 research R6).
package ghloop

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/go-github/v85/github"

	"github.com/tosin2013/chalkeus/internal/secret"
)

// ErrNoToken means no GitHub token could be found for reads.
var ErrNoToken = errors.New("no GitHub token for reads: set GH_TOKEN or GITHUB_TOKEN, or sign in with `gh auth login`")

// ReadToken resolves the operator's token: GH_TOKEN, GITHUB_TOKEN, then `gh auth token`.
func ReadToken() (secret.Value, error) {
	for _, k := range []string{"GH_TOKEN", "GITHUB_TOKEN"} {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return secret.New(v), nil
		}
	}
	if _, err := exec.LookPath("gh"); err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if out, err := exec.CommandContext(ctx, "gh", "auth", "token").Output(); err == nil {
			if v := strings.TrimSpace(string(out)); v != "" {
				return secret.New(v), nil
			}
		}
	}
	return secret.Value{}, ErrNoToken
}

// NewClient returns a go-github client authenticated with token.
func NewClient(token secret.Value) *github.Client {
	hc := &http.Client{Timeout: 60 * time.Second}
	c := github.NewClient(hc).WithAuthToken(token.Reveal())
	if API != "https://api.github.com" { // tests point API at an httptest server
		if u, err := url.Parse(strings.TrimSuffix(API, "/") + "/"); err == nil {
			c.BaseURL = u
		}
	}
	return c
}

// OperatorClient is the operator's own client (reads, and operator-approved writes such as
// reset-fixture).
func OperatorClient() (*github.Client, error) {
	tok, err := ReadToken()
	if err != nil {
		return nil, err
	}
	return NewClient(tok), nil
}

// SplitRepo splits owner/name.
func SplitRepo(repo string) (string, string) {
	o, n, _ := strings.Cut(repo, "/")
	return o, n
}
