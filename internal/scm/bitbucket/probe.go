package bitbucket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/tosin2013/chalkeus/internal/scm"
)

// ProbePublisher checks the publisher's credential is a repository access token for repo
// (research R4). Such a token can't read its own identity (GET /user refuses access tokens), so
// the check is by elimination:
//   - GET /user must be refused, else the token is a user's API token (SCM_TOKEN_WRONG_KIND);
//   - the target repository must read (SCM_TOKEN_WRONG_REPO);
//   - listing the workspace's repositories must show no other private repository, else it is a
//     project or workspace token (SCM_TOKEN_WIDER_THAN_REPO). Public repositories are listed for
//     anyone, so they don't count (found in the proof, 2026-11-28).
//
// The bot's account id is read back later from the author of the first PR it opens.
func (c *Client) ProbePublisher(ctx context.Context, repo string) scm.TokenProbeResult {
	res := scm.TokenProbeResult{Kind: "repository_token", Repo: repo}
	add := func(name string, ok bool, detail, code string) {
		res.Lines = append(res.Lines, scm.ProbeLine{Name: name, OK: ok, Detail: detail})
		if !ok && res.Code == "" {
			res.Code = code
		}
	}
	var u user
	_, err := c.h.Do(ctx, http.MethodGet, "user", nil, "", &u)
	switch {
	case err == nil:
		add("access token", false, fmt.Sprintf("the token acts as user %s: use a repository access token for %s only", u.account().Name, repo), scm.TokenWrongKind)
		return res
	case scm.IsCode(err, scm.ErrUnauthorized) && !strings.Contains(err.Error(), "not supported"):
		add("access token", false, "the token is invalid or expired", scm.TokenExpired)
		return res
	default:
		add("access token", true, "GET /user is refused, as for repository access tokens", "")
	}
	if _, err := c.h.Do(ctx, http.MethodGet, repoPath(repo), nil, "", nil); err != nil {
		add("reads "+repo, false, err.Error(), scm.TokenWrongRepo)
		return res
	}
	add("reads "+repo, true, "", "")
	ws, _, _ := strings.Cut(repo, "/")
	var others []string
	_, err = c.h.List(ctx, "repositories/"+url.PathEscape(ws)+"?pagelen=100", func(raw []byte) (int, string, error) {
		var page struct {
			Values []struct {
				FullName  string `json:"full_name"`
				IsPrivate bool   `json:"is_private"`
			} `json:"values"`
			Next string `json:"next"`
		}
		if err := json.Unmarshal(raw, &page); err != nil {
			return 0, "", err
		}
		for _, v := range page.Values {
			if v.IsPrivate && !strings.EqualFold(v.FullName, repo) {
				others = append(others, v.FullName)
			}
		}
		return len(page.Values), page.Next, nil
	})
	switch {
	case err != nil && (scm.IsCode(err, scm.ErrForbidden) || scm.IsCode(err, scm.ErrNotFound)):
		add("one repository", true, "the workspace's repository list is refused", "")
	case err != nil:
		add("one repository", false, "could not list the workspace's repositories: "+err.Error(), scm.TokenWiderThanRepo)
	case len(others) > 0:
		add("one repository", false, fmt.Sprintf("the token also reaches %d other private repositories (e.g. %s): a project or workspace token; use a repository access token", len(others), others[0]), scm.TokenWiderThanRepo)
	default:
		add("one repository", true, "no other private repository visible", "")
	}
	return res
}
