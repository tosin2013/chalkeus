package bitbucket

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/tosin2013/chalkeus/internal/scm"
	"github.com/tosin2013/chalkeus/internal/scm/scmtest"
	"github.com/tosin2013/chalkeus/internal/secret"
)

// fakeBitbucket serves the Bitbucket /2.0 endpoints the client calls, over scmtest.Model. It
// reproduces two quirks: the pull request list leaves drafts out unless the query asks for
// draft=true (research R3), and diffstat answers with a redirect.
type fakeBitbucket struct {
	m           *scmtest.Model
	accessToken bool     // GET /user refused, as for repository access tokens
	extraRepos  []string // other repositories the token can list (a workspace token)
}

var (
	repoRE = regexp.MustCompile(`^/2\.0/repositories/([^/]+)/([^/]+)(/.*)?$`)
	prRE   = regexp.MustCompile(`^/pullrequests/(\d+)(/[a-z]+)?$`)
	wsRE   = regexp.MustCompile(`^/2\.0/repositories/([^/]+)$`)
	permRE = regexp.MustCompile(`^/2\.0/workspaces/([^/]+)/permissions/repositories/([^/]+)$`)
	uuidRE = regexp.MustCompile(`user\.uuid="([^"]+)"`)
)

func (f *fakeBitbucket) user(a scm.Account) map[string]any {
	return map[string]any{"uuid": a.ID, "nickname": a.Name, "display_name": a.Name, "type": "user"}
}

func (f *fakeBitbucket) pr(r *scmtest.Repo, q *scmtest.Request) map[string]any {
	state := map[string]string{scm.StateOpen: "OPEN", scm.StateMerged: "MERGED", scm.StateClosed: "DECLINED"}[q.State]
	v := map[string]any{"id": q.ID, "state": state, "draft": q.Draft, "title": q.Title,
		"source":      map[string]any{"branch": map[string]any{"name": q.Head}, "commit": map[string]any{"hash": q.HeadSHA}},
		"destination": map[string]any{"branch": map[string]any{"name": q.Base}},
		"author":      f.user(q.Author),
		"links":       map[string]any{"html": map[string]any{"href": fmt.Sprintf("https://bitbucket.org/%s/pull-requests/%d", r.Name, q.ID)}}}
	if q.State == scm.StateMerged {
		v["merge_commit"], v["closed_by"] = map[string]any{"hash": q.Landed}, f.user(q.MergedBy)
	}
	return v
}

func (f *fakeBitbucket) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.m.Lock()
	defer f.m.Unlock()
	js := func(status int, v any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(v)
	}
	nf := func() { js(404, map[string]any{"type": "error", "error": map[string]any{"message": "Not found"}}) }
	p := r.URL.EscapedPath()
	switch {
	case p == "/2.0/user":
		if f.accessToken {
			js(403, map[string]any{"type": "error", "error": map[string]any{"message": "Token is invalid or not supported for this endpoint."}})
			return
		}
		js(200, f.user(f.m.Owner))
		return
	case p == "/2.0/user/permissions/repositories":
		js(200, map[string]any{"values": []any{map[string]any{"permission": "admin"}}})
		return
	case strings.HasPrefix(p, "/2.0/users/"):
		id, _ := url.PathUnescape(strings.TrimPrefix(p, "/2.0/users/"))
		for _, a := range []scm.Account{f.m.Owner, f.m.Other} {
			if a.ID == id {
				js(200, f.user(a))
				return
			}
		}
		nf()
		return
	case permRE.MatchString(p):
		g := permRE.FindStringSubmatch(p)
		repo := f.m.Repo(g[1] + "/" + g[2])
		if repo == nil {
			nf()
			return
		}
		var vals []any
		if u := uuidRE.FindStringSubmatch(r.URL.Query().Get("q")); u != nil {
			if perm := repo.Perms[scm.NormalID(u[1])]; perm != "" {
				vals = append(vals, map[string]any{"permission": string(perm)})
			}
		}
		js(200, map[string]any{"values": vals})
		return
	case wsRE.MatchString(p):
		ws := wsRE.FindStringSubmatch(p)[1]
		var vals []any
		for name := range f.m.Repos {
			if strings.HasPrefix(name, ws+"/") {
				vals = append(vals, map[string]any{"full_name": name})
			}
		}
		for _, x := range f.extraRepos {
			vals = append(vals, map[string]any{"full_name": x})
		}
		js(200, map[string]any{"values": vals})
		return
	}
	g := repoRE.FindStringSubmatch(p)
	if g == nil {
		nf()
		return
	}
	repo, rest := f.m.Repo(g[1]+"/"+g[2]), g[3]
	if repo == nil {
		nf()
		return
	}
	switch {
	case r.Method == "GET" && rest == "":
		js(200, map[string]any{"full_name": repo.Name, "mainbranch": map[string]any{"name": repo.Default}})
	case r.Method == "GET" && rest == "/branch-restrictions":
		var vals []any
		for b := range repo.Protected {
			vals = append(vals, map[string]any{"kind": "restrict_merges", "pattern": b, "branch_match_kind": "glob"})
		}
		js(200, map[string]any{"values": vals})
	case r.Method == "GET" && rest == "/pullrequests":
		q := r.URL.Query().Get("q")
		wantDraft := strings.Contains(q, "draft=true")
		var vals []any
		for _, x := range repo.Requests {
			if x.State == scm.StateOpen && x.Draft == wantDraft { // the quirk: drafts only when asked
				vals = append(vals, f.pr(repo, x))
			}
		}
		js(200, map[string]any{"values": vals})
	case r.Method == "POST" && rest == "/pullrequests":
		var body struct {
			Title  string `json:"title"`
			Draft  bool   `json:"draft"`
			Source struct {
				Branch struct{ Name string } `json:"branch"`
			} `json:"source"`
			Destination struct {
				Branch struct{ Name string } `json:"branch"`
			} `json:"destination"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		q := f.m.OpenLocked(repo, body.Source.Branch.Name, body.Destination.Branch.Name, body.Title, body.Draft, f.m.Owner)
		js(201, f.pr(repo, q))
	case prRE.MatchString(rest):
		pg := prRE.FindStringSubmatch(rest)
		n, _ := strconv.Atoi(pg[1])
		q := repo.Request(n)
		if q == nil {
			nf()
			return
		}
		switch pg[2] {
		case "":
			js(200, f.pr(repo, q))
		case "/diffstat":
			http.Redirect(w, r, fmt.Sprintf("/2.0/repositories/%s/diffstat/%s..%s?pr=%d", repo.Name, q.HeadSHA, q.BaseSHA, q.ID), http.StatusFound)
		case "/comments":
			js(201, map[string]any{"id": 1})
		case "/decline":
			q.State = scm.StateClosed
			js(200, f.pr(repo, q))
		default:
			nf()
		}
	case r.Method == "GET" && strings.HasPrefix(rest, "/diffstat/"):
		n, _ := strconv.Atoi(r.URL.Query().Get("pr"))
		var vals []any
		for _, path := range repo.RequestChanged(repo.Request(n)) {
			vals = append(vals, map[string]any{"status": "modified", "new": map[string]any{"path": path}})
		}
		js(200, map[string]any{"values": vals})
	case r.Method == "GET" && strings.HasPrefix(rest, "/src/"):
		parts := strings.SplitN(strings.TrimPrefix(rest, "/src/"), "/", 2)
		rev, _ := url.PathUnescape(parts[0])
		path, _ := url.PathUnescape(parts[1])
		c := repo.Resolve(rev)
		if c == nil {
			nf()
			return
		}
		v, ok := c.Files[path]
		if !ok {
			nf()
			return
		}
		_, _ = io.WriteString(w, v)
	case r.Method == "GET" && strings.HasPrefix(rest, "/refs/branches/"):
		b, _ := url.PathUnescape(strings.TrimPrefix(rest, "/refs/branches/"))
		sha, ok := repo.Branches[b]
		if !ok {
			nf()
			return
		}
		js(200, map[string]any{"name": b, "target": map[string]any{"hash": sha}})
	case r.Method == "DELETE" && strings.HasPrefix(rest, "/refs/branches/"):
		b, _ := url.PathUnescape(strings.TrimPrefix(rest, "/refs/branches/"))
		delete(repo.Branches, b)
		w.WriteHeader(204)
	case r.Method == "POST" && rest == "/src":
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			js(400, map[string]any{"error": err.Error()})
			return
		}
		var files []scm.File
		for _, path := range r.MultipartForm.Value["files"] {
			files = append(files, scm.File{Path: path, Delete: true})
		}
		for name, fhs := range r.MultipartForm.File {
			fh, _ := fhs[0].Open()
			raw, _ := io.ReadAll(fh)
			files = append(files, scm.File{Path: name, Content: raw})
		}
		c := f.m.ApplyFiles(repo, r.FormValue("parents"), files, r.FormValue("message"))
		repo.Branches[r.FormValue("branch")] = c.SHA
		w.Header().Set("Location", fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s/commit/%s", repo.Name, c.SHA))
		w.WriteHeader(201)
	default:
		js(404, map[string]any{"error": "no route " + r.Method + " " + rest})
	}
}

var (
	owner = scm.Account{ID: "{11111111-1111-1111-1111-111111111111}", Name: "owner"}
	other = scm.Account{ID: "{22222222-2222-2222-2222-222222222222}", Name: "approver"}
)

func newFake(t *testing.T, f *fakeBitbucket) *Client {
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	h := scm.NewHost(scm.Bitbucket, "", "")
	h.APIBase = srv.URL + "/2.0"
	c, err := New(h, secret.New("t"))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestContract(t *testing.T) {
	scmtest.Run(t, func(t *testing.T) (scm.Client, scmtest.Fake) {
		m := scmtest.NewModel(owner, other, scmtest.MergeCommit, scmtest.Squash, scmtest.FastForward)
		return newFake(t, &fakeBitbucket{m: m}), m
	})
}

func TestProbePublisher(t *testing.T) {
	setup := func(f *fakeBitbucket) scm.TokenProbeResult {
		f.m = scmtest.NewModel(owner, other)
		c := newFake(t, f)
		f.m.CreateRepo(t, "ws/target", map[string]string{"a": "b"})
		return c.ProbePublisher(t.Context(), "ws/target")
	}
	if r := setup(&fakeBitbucket{accessToken: true}); !r.OK() || r.Kind != "repository_token" {
		t.Fatalf("a repository token: %+v", r)
	}
	if r := setup(&fakeBitbucket{accessToken: false}); r.Code != scm.TokenWrongKind {
		t.Fatalf("a user's API token: %+v", r)
	}
	if r := setup(&fakeBitbucket{accessToken: true, extraRepos: []string{"ws/other"}}); r.Code != scm.TokenWiderThanRepo {
		t.Fatalf("a workspace token: %+v", r)
	}
	f := &fakeBitbucket{accessToken: true, m: scmtest.NewModel(owner, other)}
	c := newFake(t, f)
	if r := c.ProbePublisher(t.Context(), "ws/missing"); r.Code != scm.TokenWrongRepo {
		t.Fatalf("another repository: %+v", r)
	}
}

func TestAccountIDs(t *testing.T) {
	for _, ok := range []string{"{11111111-1111-1111-1111-111111111111}", "557058:2a1b3c4d-1234-5678-9abc-def012345678", "5b10ac8d82e05b22cc7d4ef5"} {
		if !ValidAccountID(ok) {
			t.Errorf("%q refused", ok)
		}
	}
	for _, bad := range []string{"alice", "@alice", "{not-a-uuid}", ""} {
		if ValidAccountID(bad) {
			t.Errorf("%q accepted", bad)
		}
	}
}

func TestPermissionNeedsAdmin(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(403) }))
	defer srv.Close()
	h := scm.NewHost(scm.Bitbucket, "", "")
	h.APIBase = srv.URL + "/2.0"
	c, _ := New(h, secret.New("t"))
	if _, err := c.Permission(t.Context(), "ws/r", other); err == nil || !strings.Contains(err.Error(), "needs repository admin") {
		t.Fatalf("%v", err)
	}
}
