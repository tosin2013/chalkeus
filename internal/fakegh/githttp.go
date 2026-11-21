package fakegh

import (
	"net/http"
	"net/http/cgi"
	"os/exec"
	"strings"
)

// serveGit is git's smart HTTP for /git/<owner>/<name>.git/..., through `git http-backend`
// over the bare repositories (025 R3). Fetching needs no credentials (a test simplification:
// the host's git credentials are for github.com); pushing needs a token with write permission,
// sent as Basic auth (user x-access-token, as the publisher's GIT_ASKPASS does).
func (s *Server) serveGit(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/git")
	full := strings.TrimPrefix(path, "/")
	if i := strings.Index(full, ".git/"); i > 0 {
		full = full[:i]
	} else {
		http.NotFound(w, r)
		return
	}
	s.mu.Lock()
	rp := s.repos[full]
	login := ""
	if _, pass, ok := r.BasicAuth(); ok {
		login = s.login(pass)
	}
	canWrite := rp != nil && rp.canWrite(login)
	s.mu.Unlock()
	if rp == nil {
		http.NotFound(w, r)
		return
	}
	push := r.URL.Query().Get("service") == "git-receive-pack" || strings.HasSuffix(path, "/git-receive-pack")
	if push && !canWrite {
		w.Header().Set("WWW-Authenticate", `Basic realm="fakegithub"`)
		http.Error(w, "push needs a token with write permission", http.StatusUnauthorized)
		return
	}
	bin, err := exec.LookPath("git")
	if err != nil {
		http.Error(w, "git not found", http.StatusInternalServerError)
		return
	}
	h := &cgi.Handler{Path: bin, Args: []string{"http-backend"}, Dir: s.Dir,
		Env: []string{"GIT_PROJECT_ROOT=" + s.Dir, "GIT_HTTP_EXPORT_ALL=1", "REMOTE_USER=" + login, "GIT_CONFIG_NOSYSTEM=1"}}
	r2 := r.Clone(r.Context())
	r2.URL.Path = path // PATH_INFO is the path below /git
	h.ServeHTTP(w, r2)
}
