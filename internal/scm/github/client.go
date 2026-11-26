package github

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	gh "github.com/google/go-github/v85/github"

	"github.com/tosin2013/chalkeus/internal/scm"
	"github.com/tosin2013/chalkeus/internal/secret"
)

// Client is scm.Client for GitHub, over go-github. It reproduces the calls forge made before the
// host interface (ghloop.OpenPilotPRs, ClosePR, GHReader.PR, author.openPR, doctor's reads).
type Client struct {
	host  scm.Host
	GH    *gh.Client
	Probe func(ctx context.Context, repo string) scm.TokenProbeResult // the publisher probe (ghloop.Probe), injected
}

// New returns a client for h authenticated with token.
func New(h scm.Host, token secret.Value) *Client {
	if h.Kind == "" {
		h = scm.NewHost(scm.GitHub, "", "")
	}
	c := gh.NewClient(&http.Client{Timeout: 60 * time.Second}).WithAuthToken(token.Reveal())
	if base := strings.TrimRight(h.APIBase, "/"); base != "" && base != "https://api.github.com" {
		if u, err := url.Parse(base + "/"); err == nil {
			c.BaseURL = u
		}
	}
	return &Client{host: h, GH: c}
}

// Wrap makes a client from an existing go-github client (callers that already hold one).
func Wrap(h scm.Host, c *gh.Client) *Client {
	if h.Kind == "" {
		h = scm.NewHost(scm.GitHub, "", "")
	}
	return &Client{host: h, GH: c}
}

var _ scm.Client = (*Client)(nil)

func split(repo string) (string, string) {
	o, n, _ := strings.Cut(repo, "/")
	return o, n
}

// mapErr turns a go-github error into an *scm.Error.
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	var rl *gh.RateLimitError
	var al *gh.AbuseRateLimitError
	if errors.As(err, &rl) || errors.As(err, &al) {
		return &scm.Error{Code: scm.ErrRateLimited, Msg: err.Error(), Status: 403}
	}
	var ge *gh.ErrorResponse
	if errors.As(err, &ge) && ge.Response != nil {
		code := scm.ErrUnavailable
		switch ge.Response.StatusCode {
		case 401:
			code = scm.ErrUnauthorized
		case 403:
			code = scm.ErrForbidden
		case 404:
			code = scm.ErrNotFound
		case 409:
			code = scm.ErrConflict
		case 422:
			code = scm.ErrInvalid
		case 429:
			code = scm.ErrRateLimited
		}
		return &scm.Error{Code: code, Msg: err.Error(), Status: ge.Response.StatusCode}
	}
	return &scm.Error{Code: scm.ErrUnavailable, Msg: err.Error()}
}

func account(u *gh.User) scm.Account {
	if u == nil || u.GetLogin() == "" {
		return scm.Account{}
	}
	return scm.Account{ID: strings.ToLower(u.GetLogin()), Name: u.GetLogin(), Bot: u.GetType() == "Bot"}
}

func (c *Client) Host() scm.Host { return c.host }

func (c *Client) Whoami(ctx context.Context) (scm.Account, error) {
	u, _, err := c.GH.Users.Get(ctx, "")
	if err != nil {
		return scm.Account{}, mapErr(err)
	}
	return account(u), nil
}

func (c *Client) Repository(ctx context.Context, repo string) (scm.RepoInfo, error) {
	o, n := split(repo)
	r, _, err := c.GH.Repositories.Get(ctx, o, n)
	if err != nil {
		return scm.RepoInfo{}, mapErr(err)
	}
	return scm.RepoInfo{DefaultBranch: r.GetDefaultBranch(), CanPush: r.GetPermissions().GetPush()}, nil
}

func (c *Client) ResolveAccount(_ context.Context, given string) (scm.Account, error) {
	login := strings.TrimLeft(strings.TrimSpace(given), "@")
	if login == "" {
		return scm.Account{}, &scm.Error{Code: scm.ErrInvalid, Msg: "empty GitHub login"}
	}
	return scm.Account{ID: strings.ToLower(login), Name: login}, nil
}

func (c *Client) Permission(ctx context.Context, repo string, a scm.Account) (scm.Permission, error) {
	o, n := split(repo)
	login := a.Name
	if login == "" {
		login = a.ID
	}
	p, _, err := c.GH.Repositories.GetPermissionLevel(ctx, o, n, login)
	if err != nil {
		if scm.IsCode(mapErr(err), scm.ErrNotFound) {
			return scm.PermNone, nil
		}
		return scm.PermNone, mapErr(err)
	}
	switch {
	case p.GetRoleName() == "maintain":
		return scm.PermWrite, nil
	case p.GetPermission() == "admin":
		return scm.PermAdmin, nil
	case p.GetPermission() == "write":
		return scm.PermWrite, nil
	case p.GetPermission() == "read":
		return scm.PermRead, nil
	}
	return scm.PermNone, nil
}

func (c *Client) BranchProtection(ctx context.Context, repo, branch string) (scm.Protection, error) {
	o, n := split(repo)
	p, _, err := c.GH.Repositories.GetBranchProtection(ctx, o, n, branch)
	if err != nil {
		if errors.Is(err, gh.ErrBranchNotProtected) {
			return scm.Protection{}, nil
		}
		if e := mapErr(err); scm.IsCode(e, scm.ErrNotFound) {
			return scm.Protection{}, nil
		} else {
			return scm.Protection{}, e
		}
	}
	return scm.Protection{Protected: true, ForcePush: p.GetAllowForcePushes().Enabled}, nil
}

func (c *Client) view(pr *gh.PullRequest) scm.ChangeRequest {
	cr := scm.ChangeRequest{ID: pr.GetNumber(), URL: pr.GetHTMLURL(), State: scm.StateOpen, Draft: pr.GetDraft(),
		HeadRef: pr.GetHead().GetRef(), HeadSHA: pr.GetHead().GetSHA(), BaseRef: pr.GetBase().GetRef(), Author: account(pr.GetUser())}
	switch {
	case pr.GetMerged():
		cr.State, cr.LandedCommit, cr.MergedBy = scm.StateMerged, pr.GetMergeCommitSHA(), account(pr.GetMergedBy())
	case pr.GetState() == "closed":
		cr.State = scm.StateClosed
	}
	return cr
}

func (c *Client) OpenChangeRequests(ctx context.Context, repo, headPrefix string) ([]scm.ChangeRequest, error) {
	o, n := split(repo)
	var out []scm.ChangeRequest
	opts := &gh.PullRequestListOptions{State: "open", ListOptions: gh.ListOptions{PerPage: 100}}
	for seen := 0; ; {
		prs, resp, err := c.GH.PullRequests.List(ctx, o, n, opts)
		if err != nil {
			return nil, mapErr(err)
		}
		for _, p := range prs {
			if strings.HasPrefix(p.GetHead().GetRef(), headPrefix) {
				out = append(out, c.view(p))
			}
		}
		seen += len(prs)
		if resp.NextPage == 0 || seen >= scm.PageCap {
			return out, nil
		}
		opts.Page = resp.NextPage
	}
}

func (c *Client) ChangeRequest(ctx context.Context, repo string, id int) (scm.ChangeRequest, error) {
	o, n := split(repo)
	pr, _, err := c.GH.PullRequests.Get(ctx, o, n, id)
	if err != nil {
		return scm.ChangeRequest{}, mapErr(err)
	}
	return c.view(pr), nil
}

func (c *Client) ChangedFiles(ctx context.Context, repo string, cr scm.ChangeRequest) ([]string, error) {
	o, n := split(repo)
	var out []string
	opts := &gh.ListOptions{PerPage: 100}
	for {
		files, resp, err := c.GH.PullRequests.ListFiles(ctx, o, n, cr.ID, opts)
		if err != nil {
			return nil, mapErr(err)
		}
		for _, f := range files {
			out = append(out, f.GetFilename())
		}
		if resp.NextPage == 0 || len(out) >= scm.PageCap {
			sort.Strings(out)
			return out, nil
		}
		opts.Page = resp.NextPage
	}
}

func (c *Client) FileAt(ctx context.Context, repo, rev, path string) ([]byte, error) {
	o, n := split(repo)
	f, _, _, err := c.GH.Repositories.GetContents(ctx, o, n, path, &gh.RepositoryContentGetOptions{Ref: rev})
	if err != nil {
		return nil, mapErr(err)
	}
	if f == nil {
		return nil, &scm.Error{Code: scm.ErrNotFound, Msg: path + " is a folder"}
	}
	s, err := f.GetContent()
	if err != nil {
		return nil, &scm.Error{Code: scm.ErrInvalid, Msg: err.Error()}
	}
	return []byte(s), nil
}

// CommitFiles writes files as one commit on a new branch through the Git Data API (the author's
// openPR before 037).
func (c *Client) CommitFiles(ctx context.Context, repo, base, branch string, files []scm.File, message string) (string, error) {
	o, n := split(repo)
	ref, _, err := c.GH.Git.GetRef(ctx, o, n, "refs/heads/"+base)
	if err != nil {
		return "", mapErr(err)
	}
	parent := ref.GetObject().GetSHA()
	pc, _, err := c.GH.Git.GetCommit(ctx, o, n, parent)
	if err != nil {
		return "", mapErr(err)
	}
	var entries []*gh.TreeEntry
	for _, f := range files {
		if f.Delete {
			entries = append(entries, &gh.TreeEntry{Path: gh.Ptr(f.Path), Mode: gh.Ptr("100644"), Type: gh.Ptr("blob")})
			continue
		}
		blob, _, err := c.GH.Git.CreateBlob(ctx, o, n, gh.Blob{Content: gh.Ptr(string(f.Content)), Encoding: gh.Ptr("utf-8")})
		if err != nil {
			return "", mapErr(err)
		}
		entries = append(entries, &gh.TreeEntry{Path: gh.Ptr(f.Path), Mode: gh.Ptr("100644"), Type: gh.Ptr("blob"), SHA: blob.SHA})
	}
	tree, _, err := c.GH.Git.CreateTree(ctx, o, n, pc.GetTree().GetSHA(), entries)
	if err != nil {
		return "", mapErr(err)
	}
	commit, _, err := c.GH.Git.CreateCommit(ctx, o, n, gh.Commit{Message: gh.Ptr(message), Tree: tree,
		Parents: []*gh.Commit{{SHA: gh.Ptr(parent)}}}, nil)
	if err != nil {
		return "", mapErr(err)
	}
	if _, _, err := c.GH.Git.CreateRef(ctx, o, n, gh.CreateRef{Ref: "refs/heads/" + branch, SHA: commit.GetSHA()}); err != nil {
		return "", mapErr(err)
	}
	return commit.GetSHA(), nil
}

func (c *Client) OpenRequest(ctx context.Context, repo, head, base, title, body string, draft bool) (scm.ChangeRequest, error) {
	o, n := split(repo)
	np := &gh.NewPullRequest{Title: gh.Ptr(title), Head: gh.Ptr(head), Base: gh.Ptr(base), Body: gh.Ptr(body)}
	if draft {
		np.Draft = gh.Ptr(true)
	}
	pr, _, err := c.GH.PullRequests.Create(ctx, o, n, np)
	if err != nil {
		return scm.ChangeRequest{}, mapErr(err)
	}
	return c.view(pr), nil
}

// CloseChangeRequest comments, closes the PR and deletes its head branch (`gh pr close
// --delete-branch --comment …`; ghloop.ClosePR before 037).
func (c *Client) CloseChangeRequest(ctx context.Context, repo string, id int, comment string) error {
	o, n := split(repo)
	pr, _, err := c.GH.PullRequests.Get(ctx, o, n, id)
	if err != nil {
		return mapErr(err)
	}
	if comment != "" {
		if _, _, err := c.GH.Issues.CreateComment(ctx, o, n, id, &gh.IssueComment{Body: &comment}); err != nil {
			return mapErr(err)
		}
	}
	closed := "closed"
	if _, _, err := c.GH.PullRequests.Edit(ctx, o, n, id, &gh.PullRequest{State: &closed}); err != nil {
		return mapErr(err)
	}
	if _, err := c.GH.Git.DeleteRef(ctx, o, n, "heads/"+pr.GetHead().GetRef()); err != nil {
		if e := mapErr(err).(*scm.Error); e.Status != 422 && e.Status != 404 {
			return e
		}
	}
	return nil
}

func (c *Client) ProbePublisher(ctx context.Context, repo string) scm.TokenProbeResult {
	if c.Probe == nil {
		return scm.TokenProbeResult{Repo: repo, Code: scm.TokenWrongKind,
			Lines: []scm.ProbeLine{{Name: "probe", Detail: "the GitHub publisher probe isn't configured (ghloop.VerifyFile)"}}}
	}
	return c.Probe(ctx, repo)
}
