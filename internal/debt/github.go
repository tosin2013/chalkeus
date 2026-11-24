package debt

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/go-github/v85/github"
)

// ErrNoAccess means the credential can't read something (HTTP 403/404 on an optional read):
// the report shows it as unknown, never as a pass (spec edge cases).
var ErrNoAccess = errors.New("not readable with this credential")

// PR is the part of an open pull request the report reads.
type PR struct {
	Number     int
	URL        string
	Title      string
	Body       string
	Author     string
	AuthorType string // "Bot" for apps
	HeadRef    string
	HeadSHA    string
}

// Check states.
const (
	CheckFailing = "failing"
	CheckPending = "pending"
	CheckPassing = "passing"
)

// Check is one check run or commit status on a commit.
type Check struct {
	Name  string
	State string
}

// Alert is an open Dependabot alert.
type Alert struct {
	GHSA    string
	Package string
	Patched string // the first patched version, "" when none
}

// Reader is everything the report reads; GitHub implements it with GETs only (FR-007).
type Reader interface {
	DefaultBranch(ctx context.Context, repo string) (branch, sha string, err error)
	OpenPRs(ctx context.Context, repo string) ([]PR, error)
	Checks(ctx context.Context, repo, sha string) ([]Check, error)
	// File returns a file's content at ref, or nil when it doesn't exist.
	File(ctx context.Context, repo, path, ref string) ([]byte, error)
	Alerts(ctx context.Context, repo string) ([]Alert, error)
}

// GitHub reads through go-github. Every request it makes is a GET.
type GitHub struct{ C *github.Client }

func split(repo string) (string, string) {
	o, n, _ := strings.Cut(repo, "/")
	return o, n
}

func access(resp *github.Response, err error) error {
	if err == nil {
		return nil
	}
	if resp != nil && (resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound) {
		return ErrNoAccess
	}
	return err
}

func (g GitHub) DefaultBranch(ctx context.Context, repo string) (string, string, error) {
	o, n := split(repo)
	r, _, err := g.C.Repositories.Get(ctx, o, n)
	if err != nil {
		return "", "", fmt.Errorf("cannot read %s: %v", repo, err)
	}
	b, _, err := g.C.Repositories.GetBranch(ctx, o, n, r.GetDefaultBranch(), 0)
	if err != nil {
		return "", "", fmt.Errorf("cannot read %s's default branch: %v", repo, err)
	}
	return r.GetDefaultBranch(), b.GetCommit().GetSHA(), nil
}

func (g GitHub) OpenPRs(ctx context.Context, repo string) ([]PR, error) {
	o, n := split(repo)
	opts := &github.PullRequestListOptions{State: "open", ListOptions: github.ListOptions{PerPage: 100}}
	var out []PR
	for page := 0; page < 3; page++ { // up to 300, so the report can count what's over its limit
		prs, resp, err := g.C.PullRequests.List(ctx, o, n, opts)
		if err != nil {
			return nil, fmt.Errorf("cannot list %s's open pull requests: %v", repo, err)
		}
		for _, p := range prs {
			out = append(out, PR{Number: p.GetNumber(), URL: p.GetHTMLURL(), Title: p.GetTitle(), Body: p.GetBody(),
				Author: p.GetUser().GetLogin(), AuthorType: p.GetUser().GetType(), HeadRef: p.GetHead().GetRef(), HeadSHA: p.GetHead().GetSHA()})
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return out, nil
}

func (g GitHub) Checks(ctx context.Context, repo, sha string) ([]Check, error) {
	o, n := split(repo)
	var out []Check
	runs, resp, err := g.C.Checks.ListCheckRunsForRef(ctx, o, n, sha, &github.ListCheckRunsOptions{ListOptions: github.ListOptions{PerPage: 100}})
	if err := access(resp, err); err != nil {
		return nil, err
	}
	for _, r := range runs.CheckRuns {
		out = append(out, Check{Name: r.GetName(), State: runState(r.GetStatus(), r.GetConclusion())})
	}
	st, resp, err := g.C.Repositories.GetCombinedStatus(ctx, o, n, sha, &github.ListOptions{PerPage: 100})
	if err := access(resp, err); err != nil {
		return nil, err
	}
	for _, s := range st.Statuses {
		out = append(out, Check{Name: s.GetContext(), State: statusState(s.GetState())})
	}
	return out, nil
}

func runState(status, conclusion string) string {
	if status != "completed" {
		return CheckPending
	}
	switch conclusion {
	case "success", "neutral", "skipped":
		return CheckPassing
	}
	return CheckFailing // failure, timed_out, cancelled, action_required, startup_failure, stale
}

func statusState(s string) string {
	switch s {
	case "success":
		return CheckPassing
	case "pending":
		return CheckPending
	}
	return CheckFailing
}

func (g GitHub) File(ctx context.Context, repo, path, ref string) ([]byte, error) {
	o, n := split(repo)
	u := fmt.Sprintf("repos/%s/%s/contents/%s?ref=%s", o, n, escapePath(path), url.QueryEscape(ref))
	req, err := g.C.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.raw+json") // raw: no 1 MB limit
	var buf bytes.Buffer
	resp, err := g.C.Do(ctx, req, &buf)
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if err := access(resp, err); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func escapePath(p string) string {
	parts := strings.Split(p, "/")
	for i, s := range parts {
		parts[i] = url.PathEscape(s)
	}
	return strings.Join(parts, "/")
}

func (g GitHub) Alerts(ctx context.Context, repo string) ([]Alert, error) {
	o, n := split(repo)
	state := "open"
	alerts, resp, err := g.C.Dependabot.ListRepoAlerts(ctx, o, n, &github.ListAlertsOptions{State: &state, ListOptions: github.ListOptions{PerPage: 100}})
	if err := access(resp, err); err != nil {
		return nil, err
	}
	var out []Alert
	for _, a := range alerts {
		v := a.GetSecurityVulnerability()
		out = append(out, Alert{GHSA: a.GetSecurityAdvisory().GetGHSAID(), Package: v.GetPackage().GetName(),
			Patched: v.GetFirstPatchedVersion().GetIdentifier()})
	}
	return out, nil
}
