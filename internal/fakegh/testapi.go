package fakegh

import (
	"encoding/json"
	"net/http"
	"sort"
)

// serveTest is the owner API under /_test/ (contracts/test-mode.md): the integration tests set
// up users and repositories and act as the owner (comment, label, merge) through it.
func (s *Server) serveTest(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token, Login, Repo, From, Label, Body string
		Perms                                 map[string]string
		Issue, PR                             int
		Remove                                bool
	}
	if r.Method == http.MethodPost {
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	reply := func(v any, err error) {
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(v)
	}
	switch r.Method + " " + r.URL.Path {
	case "POST /_test/users":
		s.AddUser(in.Token, in.Login)
		reply(map[string]string{"login": in.Login}, nil)
	case "POST /_test/repos":
		reply(map[string]string{"repo": in.Repo}, s.AddRepo(in.Repo, in.From, in.Perms))
	case "POST /_test/comment":
		reply(map[string]any{"issue": in.Issue}, s.Comment(in.Repo, in.Issue, in.Login, in.Body))
	case "POST /_test/label":
		reply(map[string]any{"issue": in.Issue}, s.Label(in.Repo, in.Issue, in.Login, in.Label, in.Remove))
	case "POST /_test/merge":
		sha, err := s.Merge(in.Repo, in.PR, in.Login)
		reply(map[string]string{"sha": sha}, err)
	case "GET /_test/state":
		reply(s.State(), nil)
	default:
		http.NotFound(w, r)
	}
}

// State is every repository's issues and PRs, for the tests' assertions.
func (s *Server) State() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]any{}
	for full, rp := range s.repos {
		var issues []map[string]any
		for _, i := range rp.Issues {
			it := map[string]any{"number": i.Number, "title": i.Title, "state": i.State, "labels": i.Labels, "user": i.User,
				"comments": len(i.Comments)}
			if i.PR != nil {
				it["pr"] = map[string]any{"head": i.PR.Head, "base": i.PR.Base, "draft": i.PR.Draft, "merged": i.PR.Merged}
			}
			var bodies []string
			for _, c := range i.Comments {
				bodies = append(bodies, c.User+": "+c.Body)
			}
			it["comment_bodies"] = bodies
			issues = append(issues, it)
		}
		sort.Slice(issues, func(a, b int) bool { return issues[a]["number"].(int) < issues[b]["number"].(int) })
		out[full] = map[string]any{"issues": issues}
	}
	return out
}

// Issues is a snapshot of a repository's issues and PRs, by number (for the integration tests).
func (s *Server) Issues(repo string) []Issue {
	s.mu.Lock()
	defer s.mu.Unlock()
	rp := s.repos[repo]
	if rp == nil {
		return nil
	}
	out := make([]Issue, 0, len(rp.Issues))
	for n := 1; n < rp.Next; n++ {
		if i := rp.Issues[n]; i != nil {
			c := *i
			if i.PR != nil {
				pr := *i.PR
				c.PR = &pr
			}
			c.Labels = append([]string(nil), i.Labels...)
			out = append(out, c)
		}
	}
	return out
}
