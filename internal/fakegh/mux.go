package fakegh

import (
	"fmt"
	"net/http"
	"strings"
)

// Mux is the fake GitHub's HTTP entry point: git's smart HTTP under /git/, the owner API under
// /_test/, a stand-in model provider under /model/, and the REST API at the root.
type Mux struct{ s *Server }

// NewMux serves s.
func NewMux(s *Server) *Mux { return &Mux{s: s} }

func (m *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case strings.HasPrefix(r.URL.Path, "/git/"):
		m.s.serveGit(w, r)
	case strings.HasPrefix(r.URL.Path, "/_test/"):
		m.s.serveTest(w, r)
	case strings.HasPrefix(r.URL.Path, "/model/"):
		serveModel(w, r)
	default:
		m.s.ServeREST(w, r)
	}
}

// Label adds (or removes) a label on an issue as login, recording the issue event the pilot reads
// to tell who proposed an item (the owner acting through the fake).
func (s *Server) Label(repo string, issue int, login, label string, remove bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rp := s.repos[repo]
	if rp == nil || rp.Issues[issue] == nil {
		return fmt.Errorf("no issue %s#%d", repo, issue)
	}
	i := rp.Issues[issue]
	var names []string
	found := false
	for _, l := range i.Labels {
		if l == label {
			found = true
			if remove {
				continue
			}
		}
		names = append(names, l)
	}
	if !found && !remove {
		names = append(names, label)
	}
	(&call{s: s, repo: rp}).setLabels(i, names, login)
	return nil
}

// Comment posts a comment on an issue as login (the owner's `/approve`, for example).
func (s *Server) Comment(repo string, issue int, login, body string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rp := s.repos[repo]
	if rp == nil || rp.Issues[issue] == nil {
		return fmt.Errorf("no issue %s#%d", repo, issue)
	}
	s.addComment(rp.Issues[issue], login, body)
	return nil
}

// Merge merges a PR into its base as login, closes it, and records the merge.
func (s *Server) Merge(repo string, pr int, login string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rp := s.repos[repo]
	if rp == nil || rp.Issues[pr] == nil || rp.Issues[pr].PR == nil {
		return "", fmt.Errorf("no pull request %s#%d", repo, pr)
	}
	i := rp.Issues[pr]
	if i.PR.Merged || i.State != "open" {
		return "", fmt.Errorf("%s#%d is not open", repo, pr)
	}
	sha, err := merge(rp.Bare, i.PR.Base, i.PR.Head, login, fmt.Sprintf("Merge pull request #%d from %s", pr, i.PR.Head))
	if err != nil {
		return "", err
	}
	now := s.now()
	i.PR.Merged, i.PR.MergedBy, i.PR.MergedAt, i.PR.MergeSHA, i.State, i.UpdatedAt = true, login, now, sha, "closed", now
	i.Events = append(i.Events, &Event{ID: s.id(), Kind: "merged", Actor: login, CreatedAt: now},
		&Event{ID: s.id(), Kind: "closed", Actor: login, CreatedAt: now})
	return sha, nil
}

// serveModel answers `./chalkeus forge doctor`'s one-token model check (an OpenAI-compatible
// POST <base>/chat/completions with a bearer key). The stub harness never calls a model; this only
// lets doctor's check pass in the integration tests without a real provider.
func serveModel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/chat/completions") {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"missing key"}}`))
		return
	}
	_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"OK"}}]}`))
}
