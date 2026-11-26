// Package scm is forge's one interface to Git hosts: change requests, repositories and accounts
// (037). GitHub, GitLab (gitlab.com and self-managed) and Bitbucket Cloud implement Client in the
// subpackages github, gitlab and bitbucket. The GitHub loop (issues, labels, comment approvals),
// the GitHub App and the debt report stay GitHub-only and are not part of this interface.
package scm

import (
	"fmt"
	"strings"
)

// Kind is a Git host kind. The empty Kind means GitHub.
type Kind string

const (
	GitHub    Kind = "github"
	GitLab    Kind = "gitlab"
	Bitbucket Kind = "bitbucket"
)

// ParseKind reads a FORGE_SCM value; "" is GitHub.
func ParseKind(s string) (Kind, error) {
	switch k := Kind(strings.ToLower(strings.TrimSpace(s))); k {
	case "", GitHub:
		return GitHub, nil
	case GitLab, Bitbucket:
		return k, nil
	}
	return "", fmt.Errorf("must be github, gitlab or bitbucket, got %q", s)
}

// Default host names and API bases. GitHubAPI and GitHubGitBase are variables so test mode can
// point them at the fake GitHub (FORGE_TEST_GITHUB_API, FORGE_TEST_GIT_BASE).
var (
	GitHubAPI     = "https://api.github.com"
	GitHubGitBase = "https://github.com/"
)

const (
	GitLabDotCom     = "gitlab.com"
	BitbucketHost    = "bitbucket.org"
	BitbucketAPIHost = "api.bitbucket.org"
)

// Host is the Git host a repository lives on (data-model.md).
type Host struct {
	Kind     Kind
	Name     string // github.com, gitlab.com or a self-managed instance, bitbucket.org
	APIBase  string // without a trailing slash
	GitBase  string // with a trailing slash
	CABundle string // absolute path of a PEM bundle (self-managed GitLab only), or ""
}

// NewHost builds a Host. name is FORGE_SCM_HOST (GitLab only; "" means gitlab.com).
func NewHost(kind Kind, name, caBundle string) Host {
	switch kind {
	case GitLab:
		if name == "" {
			name = GitLabDotCom
		}
		return Host{Kind: GitLab, Name: name, APIBase: "https://" + name + "/api/v4", GitBase: "https://" + name + "/", CABundle: caBundle}
	case Bitbucket:
		return Host{Kind: Bitbucket, Name: BitbucketHost, APIBase: "https://" + BitbucketAPIHost + "/2.0", GitBase: "https://" + BitbucketHost + "/"}
	}
	return Host{Kind: GitHub, Name: "github.com", APIBase: strings.TrimRight(GitHubAPI, "/"), GitBase: GitHubGitBase}
}

// IsGitHub reports whether the host is GitHub (the zero Host counts as GitHub).
func (h Host) IsGitHub() bool { return h.Kind == "" || h.Kind == GitHub }

// KindOrGitHub returns the kind, with the zero value read as GitHub.
func (h Host) KindOrGitHub() Kind {
	if h.Kind == "" {
		return GitHub
	}
	return h.Kind
}

// EgressHosts are the host names the publisher's proxy allows for this host (research R10).
func (h Host) EgressHosts() []string {
	switch h.KindOrGitHub() {
	case GitLab:
		return []string{hostOnly(h.Name)}
	case Bitbucket:
		return []string{BitbucketHost, BitbucketAPIHost}
	}
	return []string{"github.com", "api.github.com"}
}

// OtherProbeHost is the Git host P13 expects the publisher to be refused (contracts/checks.md).
func (h Host) OtherProbeHost() string {
	if h.IsGitHub() {
		return GitLabDotCom
	}
	return "github.com"
}

func hostOnly(name string) string {
	if i := strings.LastIndex(name, ":"); i > 0 && !strings.Contains(name[i:], "]") {
		return name[:i]
	}
	return name
}

// Account is a host account. Comparison is by ID only: GitHub logins (lower-cased), GitLab
// numeric user ids, Bitbucket {uuid}s.
type Account struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
	Bot  bool   `json:"bot,omitempty"`
}

// Same reports whether two accounts are the same, by ID.
func (a Account) Same(b Account) bool { return a.ID != "" && NormalID(a.ID) == NormalID(b.ID) }

// NormalID is the comparable form of an account id: lower-cased and trimmed (GitHub logins are
// case-insensitive; GitLab ids are digits; Bitbucket UUIDs compare case-insensitively).
func NormalID(id string) string { return strings.ToLower(strings.TrimSpace(id)) }

// Permission is an account's effective access to a repository.
type Permission string

const (
	PermNone  Permission = "none"
	PermRead  Permission = "read"
	PermWrite Permission = "write"
	PermAdmin Permission = "admin"
)

// CanWrite reports write or admin.
func (p Permission) CanWrite() bool { return p == PermWrite || p == PermAdmin }

// Protection describes a branch's protection.
type Protection struct {
	Protected      bool
	MergeRoles     []string // who may merge, as the host names roles or ids
	ForcePush      bool
	RestrictMerges bool // Bitbucket: a restrict_merges restriction matches the branch
}

// RepoInfo is what forge reads about a repository.
type RepoInfo struct {
	DefaultBranch string
	CanPush       bool
}

// File is one file of a commit forge writes (the test author's companion drafts only).
type File struct {
	Path    string
	Content []byte
}

// Change request states, normalised across hosts (research R5).
const (
	StateOpen   = "open"
	StateMerged = "merged"
	StateClosed = "closed"
)

// ChangeRequest is a pull request (GitHub, Bitbucket) or merge request (GitLab).
type ChangeRequest struct {
	ID           int     `json:"number"`
	URL          string  `json:"url"`
	State        string  `json:"state"`
	Draft        bool    `json:"draft"`
	HeadRef      string  `json:"head"`
	HeadSHA      string  `json:"head_sha,omitempty"`
	BaseRef      string  `json:"base,omitempty"`
	LandedCommit string  `json:"merge_commit,omitempty"`
	Author       Account `json:"author"`
	MergedBy     Account `json:"merged_by"`
}

// Merged reports whether the request was merged.
func (c ChangeRequest) Merged() bool { return c.State == StateMerged }

// Publisher-credential refusal codes (data-model.md TokenProbeResult).
const (
	TokenWiderThanRepo = "SCM_TOKEN_WIDER_THAN_REPO"
	TokenWrongKind     = "SCM_TOKEN_WRONG_KIND"
	TokenMissingScope  = "SCM_TOKEN_MISSING_SCOPE"
	TokenExpired       = "SCM_TOKEN_EXPIRED"
	TokenWrongRepo     = "SCM_TOKEN_WRONG_REPO"
)

// ProbeLine is one check of the publisher credential.
type ProbeLine struct {
	Name   string
	OK     bool
	Detail string
}

// TokenProbeResult is the result of checking a publisher credential (research R4).
type TokenProbeResult struct {
	Kind    string // github token|app; gitlab project_token|bot_account_token; bitbucket repository_token
	Repo    string
	BotID   string
	Scopes  []string
	Expires string
	Lines   []ProbeLine
	Code    string // "" when every line passed, else one of the Token* codes
}

// OK reports whether the credential passed every check.
func (r TokenProbeResult) OK() bool {
	if r.Code != "" {
		return false
	}
	for _, l := range r.Lines {
		if !l.OK {
			return false
		}
	}
	return true
}

// Error codes.
const (
	ErrNotFound     = "not_found"
	ErrForbidden    = "forbidden"
	ErrUnauthorized = "unauthorized"
	ErrRateLimited  = "rate_limited"
	ErrConflict     = "conflict"
	ErrInvalid      = "invalid"
	ErrUnavailable  = "unavailable"
)

// Error is a host error. Msg never contains a token.
type Error struct {
	Code   string
	Msg    string
	Status int
}

func (e *Error) Error() string { return e.Msg }

// IsCode reports whether err is an *Error with the given code.
func IsCode(err error, code string) bool {
	e, ok := err.(*Error)
	return ok && e.Code == code
}
