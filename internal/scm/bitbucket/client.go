package bitbucket

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/tosin2013/chalkeus/internal/scm"
	"github.com/tosin2013/chalkeus/internal/secret"
)

// Client is scm.Client for Bitbucket Cloud over /2.0 (research R3–R7, R11). Authentication is a
// Bearer token: the operator's Atlassian API token, the publisher's repository access token, or
// the test author's token.
type Client struct {
	host scm.Host
	h    *scm.HTTP
}

// New returns a client for h acting with token (sent as Bearer).
func New(h scm.Host, token secret.Value) (*Client, error) { return NewBasic(h, "", token) }

// NewBasic returns a client that sends Basic auth with email when email isn't empty: an
// Atlassian API token without scopes is accepted only as email:token (research R8). The
// operator sets BITBUCKET_EMAIL for that; repository access tokens always use Bearer.
func NewBasic(h scm.Host, email string, token secret.Value) (*Client, error) {
	if h.Kind == "" {
		h = scm.NewHost(scm.Bitbucket, "", "")
	}
	hc, err := scm.NewHTTP(h, token)
	if err != nil {
		return nil, err
	}
	hc.Basic = email
	return &Client{host: h, h: hc}, nil
}

var _ scm.Client = (*Client)(nil)

// accountIDRE accepts a Bitbucket account {uuid} or an Atlassian account_id (research R7).
var accountIDRE = regexp.MustCompile(`^(\{[0-9a-fA-F-]{36}\}|[0-9]{1,10}:[0-9a-fA-F-]{36}|[0-9a-f]{24})$`)

// ValidAccountID reports whether s is a Bitbucket account id: usernames can't be resolved through
// the API, so FORGE_APPROVERS must list ids for Bitbucket.
func ValidAccountID(s string) bool { return accountIDRE.MatchString(strings.TrimSpace(s)) }

func repoPath(repo string) string {
	ws, slug, _ := strings.Cut(repo, "/")
	return "repositories/" + url.PathEscape(ws) + "/" + url.PathEscape(slug)
}

type user struct {
	UUID        string `json:"uuid"`
	AccountID   string `json:"account_id"`
	DisplayName string `json:"display_name"`
	Nickname    string `json:"nickname"`
	Type        string `json:"type"`
}

func (u *user) account() scm.Account {
	if u == nil || u.UUID == "" {
		return scm.Account{}
	}
	name := u.Nickname
	if name == "" {
		name = u.DisplayName
	}
	return scm.Account{ID: u.UUID, Name: name}
}

type pr struct {
	ID          int    `json:"id"`
	State       string `json:"state"`
	Draft       bool   `json:"draft"`
	Title       string `json:"title"`
	Source      side   `json:"source"`
	Destination side   `json:"destination"`
	Author      *user  `json:"author"`
	ClosedBy    *user  `json:"closed_by"`
	MergeCommit *struct {
		Hash string `json:"hash"`
	} `json:"merge_commit"`
	Links struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
}

type side struct {
	Branch struct {
		Name string `json:"name"`
	} `json:"branch"`
	Commit *struct {
		Hash string `json:"hash"`
	} `json:"commit"`
}

func (p pr) view() scm.ChangeRequest {
	cr := scm.ChangeRequest{ID: p.ID, URL: p.Links.HTML.Href, State: scm.StateOpen, Draft: p.Draft,
		HeadRef: p.Source.Branch.Name, BaseRef: p.Destination.Branch.Name, Author: p.Author.account()}
	if p.Source.Commit != nil {
		cr.HeadSHA = p.Source.Commit.Hash
	}
	switch p.State {
	case "MERGED":
		cr.State, cr.MergedBy = scm.StateMerged, p.ClosedBy.account()
		if p.MergeCommit != nil {
			cr.LandedCommit = p.MergeCommit.Hash
		}
	case "DECLINED", "SUPERSEDED":
		cr.State = scm.StateClosed
	}
	return cr
}

func (c *Client) Host() scm.Host { return c.host }

func (c *Client) Whoami(ctx context.Context) (scm.Account, error) {
	var u user
	if _, err := c.h.Do(ctx, http.MethodGet, "user", nil, "", &u); err != nil {
		return scm.Account{}, err
	}
	return u.account(), nil
}

func (c *Client) Repository(ctx context.Context, repo string) (scm.RepoInfo, error) {
	var r struct {
		MainBranch struct {
			Name string `json:"name"`
		} `json:"mainbranch"`
	}
	if _, err := c.h.Do(ctx, http.MethodGet, repoPath(repo), nil, "", &r); err != nil {
		return scm.RepoInfo{}, err
	}
	info := scm.RepoInfo{DefaultBranch: r.MainBranch.Name}
	var perms struct {
		Values []struct {
			Permission string `json:"permission"`
		} `json:"values"`
	}
	q := url.QueryEscape(fmt.Sprintf(`repository.full_name="%s"`, repo))
	if _, err := c.h.Do(ctx, http.MethodGet, "user/permissions/repositories?q="+q, nil, "", &perms); err == nil && len(perms.Values) > 0 {
		info.CanPush = scm.Permission(perms.Values[0].Permission).CanWrite()
	}
	return info, nil
}

func (c *Client) ResolveAccount(ctx context.Context, given string) (scm.Account, error) {
	given = strings.TrimSpace(given)
	if !ValidAccountID(given) {
		return scm.Account{}, &scm.Error{Code: scm.ErrInvalid, Msg: fmt.Sprintf("%q is not a Bitbucket account id ({uuid} or account_id); usernames can't be resolved", given)}
	}
	var u user
	if _, err := c.h.Do(ctx, http.MethodGet, "users/"+url.PathEscape(given), nil, "", &u); err != nil {
		return scm.Account{}, err
	}
	return u.account(), nil
}

func (c *Client) Permission(ctx context.Context, repo string, a scm.Account) (scm.Permission, error) {
	ws, slug, _ := strings.Cut(repo, "/")
	q := url.QueryEscape(fmt.Sprintf(`user.uuid="%s"`, a.ID))
	var out struct {
		Values []struct {
			Permission string `json:"permission"`
		} `json:"values"`
	}
	p := "workspaces/" + url.PathEscape(ws) + "/permissions/repositories/" + url.PathEscape(slug) + "?q=" + q
	if _, err := c.h.Do(ctx, http.MethodGet, p, nil, "", &out); err != nil {
		if scm.IsCode(err, scm.ErrForbidden) {
			return scm.PermNone, &scm.Error{Code: scm.ErrForbidden, Msg: "can't read who may write " + repo + ": the operator credential needs repository admin"}
		}
		return scm.PermNone, err
	}
	if len(out.Values) == 0 {
		return scm.PermNone, nil
	}
	return scm.Permission(out.Values[0].Permission), nil
}

func (c *Client) BranchProtection(ctx context.Context, repo, branch string) (scm.Protection, error) {
	var prot scm.Protection
	_, err := c.h.List(ctx, repoPath(repo)+"/branch-restrictions?pagelen=100", func(raw []byte) (int, string, error) {
		var page struct {
			Values []struct {
				Kind      string `json:"kind"`
				Pattern   string `json:"pattern"`
				MatchKind string `json:"branch_match_kind"`
			} `json:"values"`
			Next string `json:"next"`
		}
		if err := json.Unmarshal(raw, &page); err != nil {
			return 0, "", err
		}
		for _, r := range page.Values {
			if r.MatchKind != "" && r.MatchKind != "glob" {
				continue
			}
			if ok, _ := path.Match(r.Pattern, branch); !ok {
				continue
			}
			prot.Protected = true
			prot.MergeRoles = append(prot.MergeRoles, r.Kind)
			if r.Kind == "restrict_merges" {
				prot.RestrictMerges = true
			}
			if r.Kind == "force" {
				prot.ForcePush = false
			}
		}
		return len(page.Values), page.Next, nil
	})
	return prot, err
}

// prList reads one query of pull requests, following "next".
func (c *Client) prList(ctx context.Context, repo, q string) ([]pr, error) {
	var out []pr
	_, err := c.h.List(ctx, repoPath(repo)+"/pullrequests?pagelen=50&q="+url.QueryEscape(q), func(raw []byte) (int, string, error) {
		var page struct {
			Values []pr   `json:"values"`
			Next   string `json:"next"`
		}
		if err := json.Unmarshal(raw, &page); err != nil {
			return 0, "", err
		}
		out = append(out, page.Values...)
		return len(page.Values), page.Next, nil
	})
	return out, err
}

// OpenChangeRequests queries open pull requests with and without draft=true and merges the
// results, because the list may leave drafts out unless filtered (research R3).
func (c *Client) OpenChangeRequests(ctx context.Context, repo, headPrefix string) ([]scm.ChangeRequest, error) {
	seen := map[int]bool{}
	var out []scm.ChangeRequest
	for _, q := range []string{`state="OPEN"`, `state="OPEN" AND draft=true`} {
		prs, err := c.prList(ctx, repo, q)
		if err != nil {
			return nil, err
		}
		for _, p := range prs {
			if seen[p.ID] || !strings.HasPrefix(p.Source.Branch.Name, headPrefix) {
				continue
			}
			seen[p.ID] = true
			out = append(out, p.view())
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (c *Client) ChangeRequest(ctx context.Context, repo string, id int) (scm.ChangeRequest, error) {
	var p pr
	if _, err := c.h.Do(ctx, http.MethodGet, fmt.Sprintf("%s/pullrequests/%d", repoPath(repo), id), nil, "", &p); err != nil {
		return scm.ChangeRequest{}, err
	}
	return p.view(), nil
}

func (c *Client) ChangedFiles(ctx context.Context, repo string, cr scm.ChangeRequest) ([]string, error) {
	set := map[string]bool{}
	_, err := c.h.List(ctx, fmt.Sprintf("%s/pullrequests/%d/diffstat?pagelen=100", repoPath(repo), cr.ID), func(raw []byte) (int, string, error) {
		var page struct {
			Values []struct {
				Old *struct {
					Path string `json:"path"`
				} `json:"old"`
				New *struct {
					Path string `json:"path"`
				} `json:"new"`
			} `json:"values"`
			Next string `json:"next"`
		}
		if err := json.Unmarshal(raw, &page); err != nil {
			return 0, "", err
		}
		for _, v := range page.Values {
			if v.New != nil {
				set[v.New.Path] = true
			}
			if v.Old != nil {
				set[v.Old.Path] = true
			}
		}
		return len(page.Values), page.Next, nil
	})
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(set))
	for p := range set {
		out = append(out, p)
	}
	sort.Strings(out)
	return out, nil
}

func (c *Client) FileAt(ctx context.Context, repo, rev, p string) ([]byte, error) {
	var raw []byte
	if _, err := c.h.Do(ctx, http.MethodGet, repoPath(repo)+"/src/"+url.PathEscape(rev)+"/"+escapePath(p), nil, "", &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func escapePath(p string) string {
	parts := strings.Split(p, "/")
	for i, s := range parts {
		parts[i] = url.PathEscape(s)
	}
	return strings.Join(parts, "/")
}

// branchHead is a branch's commit.
func (c *Client) branchHead(ctx context.Context, repo, branch string) (string, error) {
	var ref struct {
		Target struct {
			Hash string `json:"hash"`
		} `json:"target"`
	}
	if _, err := c.h.Do(ctx, http.MethodGet, repoPath(repo)+"/refs/branches/"+url.PathEscape(branch), nil, "", &ref); err != nil {
		return "", err
	}
	return ref.Target.Hash, nil
}

// CommitFiles writes files as one commit on a new branch with POST /src (multipart: message,
// branch, parents, one field per file, "files" for each deleted path).
func (c *Client) CommitFiles(ctx context.Context, repo, base, branch string, files []scm.File, message string) (string, error) {
	parent, err := c.branchHead(ctx, repo, base)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("message", message)
	_ = w.WriteField("branch", branch)
	_ = w.WriteField("parents", parent)
	for _, f := range files {
		if f.Delete {
			_ = w.WriteField("files", f.Path)
			continue
		}
		fw, err := w.CreateFormFile(f.Path, path.Base(f.Path))
		if err != nil {
			return "", err
		}
		_, _ = fw.Write(f.Content)
	}
	_ = w.Close()
	hdr, err := c.h.Do(ctx, http.MethodPost, repoPath(repo)+"/src", &buf, w.FormDataContentType(), nil)
	if err != nil {
		return "", err
	}
	if loc := hdr.Get("Location"); loc != "" {
		if i := strings.LastIndex(loc, "/"); i >= 0 && i < len(loc)-1 {
			return loc[i+1:], nil
		}
	}
	return c.branchHead(ctx, repo, branch)
}

func (c *Client) OpenRequest(ctx context.Context, repo, head, base, title, body string, draft bool) (scm.ChangeRequest, error) {
	req := map[string]any{"title": title, "description": body, "draft": draft, "close_source_branch": false,
		"source": map[string]any{"branch": map[string]any{"name": head}}, "destination": map[string]any{"branch": map[string]any{"name": base}}}
	var p pr
	if _, err := c.h.Do(ctx, http.MethodPost, repoPath(repo)+"/pullrequests", req, "", &p); err != nil {
		return scm.ChangeRequest{}, err
	}
	return p.view(), nil
}

// CloseChangeRequest comments, declines the pull request and deletes its branch (reset-fixture).
func (c *Client) CloseChangeRequest(ctx context.Context, repo string, id int, comment string) error {
	cr, err := c.ChangeRequest(ctx, repo, id)
	if err != nil {
		return err
	}
	base := fmt.Sprintf("%s/pullrequests/%d", repoPath(repo), id)
	if comment != "" {
		if _, err := c.h.Do(ctx, http.MethodPost, base+"/comments", map[string]any{"content": map[string]any{"raw": comment}}, "", nil); err != nil {
			return err
		}
	}
	if _, err := c.h.Do(ctx, http.MethodPost, base+"/decline", nil, "", nil); err != nil {
		return err
	}
	if _, err := c.h.Do(ctx, http.MethodDelete, repoPath(repo)+"/refs/branches/"+url.PathEscape(cr.HeadRef), nil, "", nil); err != nil && !scm.IsCode(err, scm.ErrNotFound) {
		return err
	}
	return nil
}
