// Package fakegh is the integration tests' fake GitHub (025 R3): the REST calls the pilot, its
// publisher and its test author make, backed by real bare git repositories so REST-created
// commits and pushed branches are the same objects. It is a test tool: nothing in the pilot
// imports it, and `cmd/fakegithub` is never released.
package fakegh

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Server holds every repository, user and issue in memory; the git data lives in bare
// repositories under Dir.
type Server struct {
	Dir   string
	mu    sync.Mutex
	users map[string]string // token -> login
	repos map[string]*Repo  // "owner/name" -> repo
	ids   int64
	now   func() time.Time
}

// Repo is one repository.
type Repo struct {
	Full      string
	Bare      string
	Default   string
	Perms     map[string]string // login -> admin | write | read
	Issues    map[int]*Issue    // issues and pull requests share numbers, as on GitHub
	Next      int
	Labels    map[string]string // name -> color
	Protected map[string]bool   // branch -> protected
}

// Issue is an issue or a pull request (PR != nil).
type Issue struct {
	Number    int
	Title     string
	Body      string
	State     string // open | closed
	Labels    []string
	User      string
	Comments  []*Comment
	Events    []*Event
	PR        *PR
	CreatedAt time.Time
	UpdatedAt time.Time
}

// PR is a pull request's own fields.
type PR struct {
	Head, Base string
	Draft      bool
	Merged     bool
	MergedBy   string
	MergedAt   time.Time
	MergeSHA   string
}

// Comment is an issue or PR comment.
type Comment struct {
	ID        int64
	User      string
	Body      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Event is an issue event (labeled, unlabeled, closed, merged).
type Event struct {
	ID        int64
	Kind      string
	Label     string
	Actor     string
	CreatedAt time.Time
}

// New makes a server whose bare repositories live under dir.
func New(dir string) *Server {
	return &Server{Dir: dir, users: map[string]string{}, repos: map[string]*Repo{}, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Server) id() int64 { s.ids++; return s.ids }

// AddUser maps a token to a login (the Authorization header's token).
func (s *Server) AddUser(token, login string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users[token] = login
}

// AddRepo creates a repository whose main branch is one commit holding the files of fromDir
// ("" for an empty main), with login permissions.
func (s *Server) AddRepo(full, fromDir string, perms map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.repos[full]; ok {
		return fmt.Errorf("repository %s exists", full)
	}
	bare := filepath.Join(s.Dir, full+".git")
	if err := os.MkdirAll(filepath.Dir(bare), 0o755); err != nil {
		return err
	}
	if err := initRepo(bare, fromDir); err != nil {
		return err
	}
	if perms == nil {
		perms = map[string]string{}
	}
	s.repos[full] = &Repo{Full: full, Bare: bare, Default: "main", Perms: perms, Issues: map[int]*Issue{}, Next: 1,
		Labels: map[string]string{}, Protected: map[string]bool{}}
	return nil
}

// login is the user behind a token ("" if unknown).
func (s *Server) login(token string) string { return s.users[token] }

// perm is login's permission on repo: admin, write, read or "".
func (r *Repo) perm(login string) string { return r.Perms[login] }

func (r *Repo) canWrite(login string) bool {
	p := r.perm(login)
	return p == "write" || p == "admin"
}
