package ghloop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/go-github/v85/github"

	"github.com/tosin2013/specfoundry/internal/evidence"
)

// The GitHub reads and writes of the 004 loop (reference: ghsync.py / ghapprove.py reads with the
// operator's gh, and pilot/images/tools/ghops.py writes with the publisher credential). Reads
// use the operator's token; writes use the publisher credential only, so "made with the
// publisher credential" stays a distinct identity (016 R6). There is deliberately no merge.

// Error is any GitHub problem in the loop. Callers fail closed (constitution II).
type Error struct{ Msg string }

func (e *Error) Error() string { return e.Msg }

func loopErr(format string, a ...any) error { return &Error{fmt.Sprintf(format, a...)} }

// Issue is the subset of an issue the loop reads.
type Issue struct {
	Number int
	Title  string
	Body   string
	Labels []string
	State  string // open | closed
}

// Comment is an issue comment as the approval rules see it.
type Comment struct {
	ID        int64
	Body      string
	Author    string
	CreatedAt string
	UpdatedAt string
	App       bool // made through a GitHub App, or by a Bot account
}

// Label is a label to ensure.
type Label struct{ Name, Color string }

// IssueEdit holds the fields to change; nil means unchanged.
type IssueEdit struct {
	Title  *string
	Body   *string
	Labels *[]string
}

// PRView is the subset of a pull request the pipeline reads.
type PRView struct {
	State       string // OPEN | CLOSED | MERGED (gh's spelling)
	MergedAt    string
	MergedBy    string
	HeadSHA     string
	HeadRef     string
	MergeCommit string
	Draft       bool
	Author      string
	URL         string
}

// Reader is the operator-side read access.
type Reader interface {
	Issues(ctx context.Context, repo string) ([]Issue, error)
	Labels(ctx context.Context, repo string) (map[string]bool, error)
	Comments(ctx context.Context, repo string, number int) ([]Comment, error)
	// Permission is the login's permission on repo: admin, maintain, write, read or none
	// (any error reads as none).
	Permission(ctx context.Context, repo, login string) string
	PR(ctx context.Context, repo string, number int) (PRView, error)
}

// Writer is the publisher-credential write access (ghops.py's ops).
type Writer interface {
	Whoami(ctx context.Context) (string, error)
	EnsureLabels(ctx context.Context, repo string, labels []Label) ([]string, error)
	CreateIssue(ctx context.Context, repo, title, body string, labels []string) (int, error)
	EditIssue(ctx context.Context, repo string, number int, e IssueEdit) error
	Comment(ctx context.Context, repo string, number int, body string) (int64, error)
	CloseIssue(ctx context.Context, repo string, number int) error
}

// GHReader implements Reader with go-github.
type GHReader struct{ C *github.Client }

// NewReader returns the operator's reader (GH_TOKEN, GITHUB_TOKEN, then `gh auth token`).
func NewReader() (GHReader, error) {
	c, err := OperatorClient()
	if err != nil {
		return GHReader{}, err
	}
	return GHReader{C: c}, nil
}

func tail(err error, n int) string {
	s := strings.TrimSpace(err.Error())
	if len(s) > n {
		s = s[len(s)-n:]
	}
	return s
}

func (r GHReader) Issues(ctx context.Context, repo string) ([]Issue, error) {
	owner, name := SplitRepo(repo)
	opts := &github.IssueListByRepoOptions{State: "all", ListOptions: github.ListOptions{PerPage: 100}}
	var out []Issue
	for len(out) < 500 {
		page, resp, err := r.C.Issues.ListByRepo(ctx, owner, name, opts)
		if err != nil {
			return nil, loopErr("cannot list issues on %s: %s", repo, tail(err, 200))
		}
		for _, i := range page {
			if i.IsPullRequest() { // `gh issue list` lists issues only
				continue
			}
			var labels []string
			for _, l := range i.Labels {
				labels = append(labels, l.GetName())
			}
			out = append(out, Issue{Number: i.GetNumber(), Title: i.GetTitle(), Body: i.GetBody(), Labels: labels,
				State: strings.ToLower(i.GetState())})
		}
		if resp.NextPage == 0 {
			break
		}
		opts.ListOptions.Page = resp.NextPage
	}
	return out, nil
}

func (r GHReader) Labels(ctx context.Context, repo string) (map[string]bool, error) {
	owner, name := SplitRepo(repo)
	opts := &github.ListOptions{PerPage: 100}
	out := map[string]bool{}
	for {
		page, resp, err := r.C.Issues.ListLabels(ctx, owner, name, opts)
		if err != nil {
			return nil, loopErr("cannot list labels on %s: %s", repo, tail(err, 200))
		}
		for _, l := range page {
			out[l.GetName()] = true
		}
		if resp.NextPage == 0 {
			return out, nil
		}
		opts.Page = resp.NextPage
	}
}

// rawComment adds performed_via_github_app, which go-github's IssueComment does not carry.
type rawComment struct {
	ID        int64           `json:"id"`
	Body      string          `json:"body"`
	User      *github.User    `json:"user"`
	CreatedAt string          `json:"created_at"`
	UpdatedAt string          `json:"updated_at"`
	ViaApp    json.RawMessage `json:"performed_via_github_app"`
}

func (r GHReader) Comments(ctx context.Context, repo string, number int) ([]Comment, error) {
	var out []Comment
	page := 1
	for {
		u := fmt.Sprintf("repos/%s/issues/%d/comments?per_page=100&page=%d", repo, number, page)
		req, err := r.C.NewRequest("GET", u, nil)
		if err != nil {
			return nil, loopErr("cannot read comments on #%d: %s", number, tail(err, 200))
		}
		var raw []rawComment
		resp, err := r.C.Do(ctx, req, &raw)
		if err != nil {
			return nil, loopErr("cannot read comments on #%d: %s", number, tail(err, 200))
		}
		for _, c := range raw {
			via := len(c.ViaApp) > 0 && string(c.ViaApp) != "null"
			out = append(out, Comment{ID: c.ID, Body: c.Body, Author: c.User.GetLogin(), CreatedAt: c.CreatedAt,
				UpdatedAt: c.UpdatedAt, App: via || c.User.GetType() == "Bot"})
		}
		if resp.NextPage == 0 {
			return out, nil
		}
		page = resp.NextPage
	}
}

func (r GHReader) Permission(ctx context.Context, repo, login string) string {
	owner, name := SplitRepo(repo)
	p, _, err := r.C.Repositories.GetPermissionLevel(ctx, owner, name, login)
	if err != nil || p == nil {
		return "none"
	}
	if p.GetRoleName() == "maintain" {
		return "maintain"
	}
	if s := p.GetPermission(); s != "" {
		return s
	}
	return "none"
}

func (r GHReader) PR(ctx context.Context, repo string, number int) (PRView, error) {
	owner, name := SplitRepo(repo)
	pr, _, err := r.C.PullRequests.Get(ctx, owner, name, number)
	if err != nil {
		return PRView{}, err
	}
	v := PRView{State: strings.ToUpper(pr.GetState()), HeadSHA: pr.GetHead().GetSHA(), HeadRef: pr.GetHead().GetRef(),
		Draft: pr.GetDraft(), Author: pr.GetUser().GetLogin(), URL: pr.GetHTMLURL()}
	if pr.GetMerged() {
		v.State = "MERGED"
		v.MergedAt = pr.GetMergedAt().UTC().Format(evidence.TimeFormat)
		v.MergedBy = pr.GetMergedBy().GetLogin()
		v.MergeCommit = pr.GetMergeCommitSHA()
	}
	return v, nil
}

// GHWriter implements Writer with the publisher credential.
type GHWriter struct {
	C    *github.Client
	Cred Credential
}

// NewWriter loads the publisher credential from envFile (publisher.env or the App token file).
func NewWriter(envFile string) (GHWriter, error) {
	cred, err := LoadCredential(envFile)
	if err != nil {
		return GHWriter{}, loopErr("publisher credential: %v", err)
	}
	return GHWriter{C: NewClient(cred.Token), Cred: cred}, nil
}

func status(resp *github.Response, err error) int {
	if resp != nil && resp.Response != nil {
		return resp.StatusCode
	}
	var ge *github.ErrorResponse
	if errors.As(err, &ge) && ge.Response != nil {
		return ge.Response.StatusCode
	}
	return 0
}

// opErr matches ghops.py's message: "ghops <op>: <what> failed (HTTP <n>)".
func opErr(op, what string, resp *github.Response, err error) error {
	st := status(resp, err)
	if st == 0 {
		return loopErr("ghops %s: %s", op, tail(err, 300))
	}
	return loopErr("ghops %s: %s failed (HTTP %d)", op, what, st)
}

func timeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, 30*time.Second)
}

func (w GHWriter) Whoami(ctx context.Context) (string, error) {
	if w.Cred.Kind == "app" { // App installation tokens cannot call GET /user
		return w.Cred.Login, nil
	}
	ctx, cancel := timeout(ctx)
	defer cancel()
	u, resp, err := w.C.Users.Get(ctx, "")
	if err != nil {
		return "", opErr("whoami", "whoami", resp, err)
	}
	return u.GetLogin(), nil
}

func (w GHWriter) EnsureLabels(ctx context.Context, repo string, labels []Label) ([]string, error) {
	owner, name := SplitRepo(repo)
	var created []string
	for _, l := range labels {
		c, cancel := timeout(ctx)
		_, _, err := w.C.Issues.GetLabel(c, owner, name, url.PathEscape(l.Name))
		cancel()
		if err == nil {
			continue
		}
		color := l.Color
		if color == "" {
			color = "ededed"
		}
		c, cancel = timeout(ctx)
		n := l.Name
		_, resp, err := w.C.Issues.CreateLabel(c, owner, name, &github.Label{Name: &n, Color: &color})
		cancel()
		if err != nil && status(resp, err) != 422 { // 422: created concurrently
			return nil, opErr("ensure-labels", "create label "+l.Name, resp, err)
		}
		created = append(created, l.Name)
	}
	return created, nil
}

func (w GHWriter) CreateIssue(ctx context.Context, repo, title, body string, labels []string) (int, error) {
	owner, name := SplitRepo(repo)
	ctx, cancel := timeout(ctx)
	defer cancel()
	if labels == nil {
		labels = []string{}
	}
	i, resp, err := w.C.Issues.Create(ctx, owner, name, &github.IssueRequest{Title: &title, Body: &body, Labels: &labels})
	if err != nil {
		return 0, opErr("create-issue", "create issue", resp, err)
	}
	return i.GetNumber(), nil
}

func (w GHWriter) EditIssue(ctx context.Context, repo string, number int, e IssueEdit) error {
	owner, name := SplitRepo(repo)
	ctx, cancel := timeout(ctx)
	defer cancel()
	_, resp, err := w.C.Issues.Edit(ctx, owner, name, number, &github.IssueRequest{Title: e.Title, Body: e.Body, Labels: e.Labels})
	if err != nil {
		return opErr("edit-issue", "edit issue", resp, err)
	}
	return nil
}

func (w GHWriter) Comment(ctx context.Context, repo string, number int, body string) (int64, error) {
	owner, name := SplitRepo(repo)
	ctx, cancel := timeout(ctx)
	defer cancel()
	c, resp, err := w.C.Issues.CreateComment(ctx, owner, name, number, &github.IssueComment{Body: &body})
	if err != nil {
		return 0, opErr("comment", "comment", resp, err)
	}
	return c.GetID(), nil
}

func (w GHWriter) CloseIssue(ctx context.Context, repo string, number int) error {
	owner, name := SplitRepo(repo)
	ctx, cancel := timeout(ctx)
	defer cancel()
	closed := "closed"
	_, resp, err := w.C.Issues.Edit(ctx, owner, name, number, &github.IssueRequest{State: &closed})
	if err != nil {
		return opErr("close-issue", "close issue", resp, err)
	}
	return nil
}
