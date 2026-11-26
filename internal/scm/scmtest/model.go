package scmtest

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/tosin2013/chalkeus/internal/scm"
)

// Model is an in-memory host: repositories with commits, branches and change requests. It
// implements Fake (the owner's side), and each host's HTTP fake translates its API onto it, so
// every implementation is tested against the same behaviour.
type Model struct {
	mu       sync.Mutex
	Repos    map[string]*Repo
	Owner    Account
	Other    Account
	Kinds    []string
	nextID   int
	seq      int
	Approved bool // never set by any client: the no-merge contract
}

// Account is an alias so fakes don't need two imports.
type Account = scm.Account

// Repo is one repository.
type Repo struct {
	Name      string
	Default   string
	Commits   map[string]*Commit
	Branches  map[string]string
	Requests  []*Request
	Perms     map[string]scm.Permission
	Protected map[string]bool
}

// Commit is a snapshot of files.
type Commit struct {
	SHA     string
	Parents []string
	Files   map[string]string
	Message string
}

// Request is a change request.
type Request struct {
	ID       int
	Head     string
	Base     string
	Draft    bool
	Title    string
	State    string // open, merged, closed
	Author   Account
	MergedBy Account
	Landed   string
	HeadSHA  string
	BaseSHA  string
}

// NewModel makes a model whose client acts as owner.
func NewModel(owner, other Account, kinds ...string) *Model {
	if len(kinds) == 0 {
		kinds = []string{MergeCommit, Squash, FastForward}
	}
	return &Model{Repos: map[string]*Repo{}, Owner: owner, Other: other, Kinds: kinds}
}

// Lock and Unlock guard the model for HTTP fakes.
func (m *Model) Lock()   { m.mu.Lock() }
func (m *Model) Unlock() { m.mu.Unlock() }

func (m *Model) sha(seed string) string {
	m.seq++
	s := sha1.Sum([]byte(fmt.Sprintf("%s-%d", seed, m.seq)))
	return hex.EncodeToString(s[:])
}

// Repo returns a repository or nil (call with the lock held).
func (m *Model) Repo(name string) *Repo { return m.Repos[name] }

// Resolve turns a branch name or commit sha into a commit (lock held).
func (r *Repo) Resolve(rev string) *Commit {
	if sha, ok := r.Branches[rev]; ok {
		return r.Commits[sha]
	}
	if c, ok := r.Commits[rev]; ok {
		return c
	}
	for sha, c := range r.Commits { // short shas
		if len(rev) >= 7 && strings.HasPrefix(sha, rev) {
			return c
		}
	}
	return nil
}

// Request returns a request by id or nil (lock held).
func (r *Repo) Request(id int) *Request {
	for _, q := range r.Requests {
		if q.ID == id {
			return q
		}
	}
	return nil
}

// Changed lists the paths that differ between two commits (lock held).
func Changed(a, b *Commit) []string {
	var out []string
	seen := map[string]bool{}
	for p, v := range b.Files {
		seen[p] = true
		if a == nil || a.Files[p] != v {
			out = append(out, p)
		}
	}
	if a != nil {
		for p := range a.Files {
			if !seen[p] {
				out = append(out, p)
			}
		}
	}
	sort.Strings(out)
	return out
}

// RequestChanged is what a request changes: its head against the base it was opened on.
func (r *Repo) RequestChanged(q *Request) []string {
	return Changed(r.Commits[q.BaseSHA], r.Commits[q.HeadSHA])
}

// NewCommit adds a commit with files on top of parent (lock held); delete removes paths.
func (m *Model) NewCommit(r *Repo, parent string, files map[string]string, msg string) *Commit {
	c := &Commit{SHA: m.sha(r.Name + msg), Files: map[string]string{}, Message: msg}
	if p := r.Commits[parent]; p != nil {
		for k, v := range p.Files {
			c.Files[k] = v
		}
		c.Parents = []string{parent}
	}
	for k, v := range files {
		c.Files[k] = v
	}
	r.Commits[c.SHA] = c
	return c
}

// ApplyFiles is NewCommit for scm.File lists (deletes included), as a client's CommitFiles does.
func (m *Model) ApplyFiles(r *Repo, parent string, files []scm.File, msg string) *Commit {
	c := m.NewCommit(r, parent, nil, msg)
	for _, f := range files {
		if f.Delete {
			delete(c.Files, f.Path)
		} else {
			c.Files[f.Path] = string(f.Content)
		}
	}
	return c
}

// OpenLocked opens a request (lock held), as a client's OpenDraft does.
func (m *Model) OpenLocked(r *Repo, head, base, title string, draft bool, author Account) *Request {
	m.nextID++
	q := &Request{ID: m.nextID, Head: head, Base: base, Draft: draft, Title: title, State: scm.StateOpen, Author: author,
		HeadSHA: r.Branches[head], BaseSHA: r.Branches[base]}
	r.Requests = append(r.Requests, q)
	return q
}

// --- Fake ---

func (m *Model) CreateRepo(t *testing.T, repo string, files map[string]string) {
	m.Lock()
	defer m.Unlock()
	r := &Repo{Name: repo, Default: "main", Commits: map[string]*Commit{}, Branches: map[string]string{},
		Perms: map[string]scm.Permission{scm.NormalID(m.Owner.ID): scm.PermAdmin}, Protected: map[string]bool{}}
	c := m.NewCommit(r, "", files, "initial")
	r.Branches["main"] = c.SHA
	m.Repos[repo] = r
}

func (m *Model) PushBranch(t *testing.T, repo, branch string, files map[string]string) string {
	m.Lock()
	defer m.Unlock()
	r := m.must(t, repo)
	c := m.NewCommit(r, r.Branches["main"], files, "push "+branch)
	r.Branches[branch] = c.SHA
	return c.SHA
}

func (m *Model) OpenRequest(t *testing.T, repo, head string, draft bool, author scm.Account) int {
	m.Lock()
	defer m.Unlock()
	return m.OpenLocked(m.must(t, repo), head, "main", "change "+head, draft, author).ID
}

func (m *Model) MergeAsOwner(t *testing.T, repo string, id int, how string, by scm.Account) string {
	m.Lock()
	defer m.Unlock()
	r := m.must(t, repo)
	q := r.Request(id)
	head := r.Commits[q.HeadSHA]
	var landed string
	switch how {
	case FastForward:
		landed = q.HeadSHA
	case Squash:
		landed = m.NewCommit(r, r.Branches[q.Base], head.Files, "squash").SHA
	default:
		c := m.NewCommit(r, r.Branches[q.Base], head.Files, "merge")
		c.Parents = append(c.Parents, q.HeadSHA)
		landed = c.SHA
	}
	r.Branches[q.Base] = landed
	q.State, q.MergedBy, q.Landed, q.Draft = scm.StateMerged, by, landed, false
	return landed
}

func (m *Model) Decline(t *testing.T, repo string, id int) {
	m.Lock()
	defer m.Unlock()
	m.must(t, repo).Request(id).State = scm.StateClosed
}

func (m *Model) SetPermission(t *testing.T, repo string, a scm.Account, p scm.Permission) {
	m.Lock()
	defer m.Unlock()
	m.must(t, repo).Perms[scm.NormalID(a.ID)] = p
}

func (m *Model) Protect(t *testing.T, repo, branch string) {
	m.Lock()
	defer m.Unlock()
	m.must(t, repo).Protected[branch] = true
}

func (m *Model) MergeKinds() []string { return m.Kinds }

func (m *Model) Accounts() (scm.Account, scm.Account) { return m.Owner, m.Other }

func (m *Model) must(t *testing.T, repo string) *Repo {
	r := m.Repos[repo]
	if r == nil {
		t.Fatalf("no repo %s", repo)
	}
	return r
}
