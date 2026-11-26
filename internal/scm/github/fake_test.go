package github

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/tosin2013/chalkeus/internal/scm"
	"github.com/tosin2013/chalkeus/internal/scm/scmtest"
	"github.com/tosin2013/chalkeus/internal/secret"
)

// fakeGitHub serves the GitHub REST endpoints the client calls, over scmtest.Model.
type fakeGitHub struct {
	m     *scmtest.Model
	blobs map[string]string
	trees map[string]map[string]string
	n     int
}

func (f *fakeGitHub) id(p string) string { f.n++; return fmt.Sprintf("%s%038d", p, f.n) }

var (
	repoRE = regexp.MustCompile(`^/repos/([^/]+/[^/]+)(/.*)?$`)
	numRE  = regexp.MustCompile(`^/pulls/(\d+)(/files)?$`)
)

func (f *fakeGitHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.m.Lock()
	defer f.m.Unlock()
	js := func(status int, v any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(v)
	}
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)
	if r.URL.Path == "/user" {
		js(200, map[string]any{"login": f.m.Owner.Name, "type": "User"})
		return
	}
	mm := repoRE.FindStringSubmatch(r.URL.Path)
	if mm == nil {
		js(404, map[string]any{"message": "Not Found"})
		return
	}
	repo, rest := f.m.Repo(mm[1]), mm[2]
	if repo == nil {
		js(404, map[string]any{"message": "Not Found"})
		return
	}
	user := func(a scm.Account) map[string]any { return map[string]any{"login": a.Name, "type": "User"} }
	pr := func(q *scmtest.Request) map[string]any {
		state := "open"
		if q.State != scm.StateOpen {
			state = "closed"
		}
		v := map[string]any{"number": q.ID, "state": state, "draft": q.Draft, "html_url": fmt.Sprintf("https://github.com/%s/pull/%d", repo.Name, q.ID),
			"head": map[string]any{"ref": q.Head, "sha": q.HeadSHA}, "base": map[string]any{"ref": q.Base}, "user": user(q.Author),
			"merged": q.State == scm.StateMerged}
		if q.State == scm.StateMerged {
			v["merge_commit_sha"], v["merged_by"] = q.Landed, user(q.MergedBy)
		}
		return v
	}
	switch {
	case r.Method == "GET" && rest == "":
		js(200, map[string]any{"default_branch": repo.Default, "permissions": map[string]any{"push": true}})
	case r.Method == "GET" && strings.HasPrefix(rest, "/collaborators/"):
		login := strings.Split(rest, "/")[2]
		p := repo.Perms[scm.NormalID(login)]
		if p == "" {
			p = scm.PermNone
		}
		js(200, map[string]any{"permission": string(p), "role_name": string(p)})
	case r.Method == "GET" && strings.HasPrefix(rest, "/branches/"):
		if !repo.Protected[strings.Split(rest, "/")[2]] {
			js(404, map[string]any{"message": "Branch not protected"})
			return
		}
		js(200, map[string]any{"allow_force_pushes": map[string]any{"enabled": false}})
	case r.Method == "GET" && rest == "/pulls":
		var out []any
		for _, q := range repo.Requests {
			if q.State == scm.StateOpen {
				out = append(out, pr(q))
			}
		}
		js(200, out)
	case r.Method == "POST" && rest == "/pulls":
		draft, _ := body["draft"].(bool)
		q := f.m.OpenLocked(repo, body["head"].(string), body["base"].(string), body["title"].(string), draft, f.m.Owner)
		js(201, pr(q))
	case numRE.MatchString(rest):
		g := numRE.FindStringSubmatch(rest)
		n, _ := strconv.Atoi(g[1])
		q := repo.Request(n)
		if q == nil {
			js(404, map[string]any{"message": "Not Found"})
			return
		}
		switch {
		case g[2] == "/files":
			var out []any
			for _, p := range repo.RequestChanged(q) {
				out = append(out, map[string]any{"filename": p})
			}
			js(200, out)
		case r.Method == "PATCH":
			if body["state"] == "closed" {
				q.State = scm.StateClosed
			}
			js(200, pr(q))
		default:
			js(200, pr(q))
		}
	case r.Method == "POST" && strings.HasPrefix(rest, "/issues/"):
		js(201, map[string]any{"id": 1})
	case r.Method == "GET" && strings.HasPrefix(rest, "/contents/"):
		c := repo.Resolve(r.URL.Query().Get("ref"))
		path := strings.TrimPrefix(rest, "/contents/")
		v, ok := "", false
		if c != nil {
			v, ok = c.Files[path]
		}
		if !ok {
			js(404, map[string]any{"message": "Not Found"})
			return
		}
		js(200, map[string]any{"type": "file", "encoding": "base64", "path": path, "content": base64.StdEncoding.EncodeToString([]byte(v))})
	case r.Method == "GET" && strings.HasPrefix(rest, "/git/ref/heads/"):
		sha, ok := repo.Branches[strings.TrimPrefix(rest, "/git/ref/heads/")]
		if !ok {
			js(404, map[string]any{"message": "Not Found"})
			return
		}
		js(200, map[string]any{"ref": "refs/heads/x", "object": map[string]any{"sha": sha, "type": "commit"}})
	case r.Method == "GET" && strings.HasPrefix(rest, "/git/commits/"):
		sha := strings.TrimPrefix(rest, "/git/commits/")
		c := repo.Commits[sha]
		if c == nil {
			js(404, map[string]any{"message": "Not Found"})
			return
		}
		f.trees["t"+sha] = c.Files
		js(200, map[string]any{"sha": sha, "tree": map[string]any{"sha": "t" + sha}})
	case r.Method == "POST" && rest == "/git/blobs":
		id := f.id("b")
		f.blobs[id] = body["content"].(string)
		js(201, map[string]any{"sha": id})
	case r.Method == "POST" && rest == "/git/trees":
		files := map[string]string{}
		for k, v := range f.trees[fmt.Sprint(body["base_tree"])] {
			files[k] = v
		}
		for _, e := range body["tree"].([]any) {
			te := e.(map[string]any)
			if sha, ok := te["sha"].(string); ok {
				files[te["path"].(string)] = f.blobs[sha]
			} else {
				delete(files, te["path"].(string))
			}
		}
		id := f.id("t")
		f.trees[id] = files
		js(201, map[string]any{"sha": id})
	case r.Method == "POST" && rest == "/git/commits":
		tree := body["tree"]
		if t, ok := tree.(map[string]any); ok {
			tree = t["sha"]
		}
		parent := ""
		if ps, _ := body["parents"].([]any); len(ps) > 0 {
			parent = fmt.Sprint(ps[0])
			if p, ok := ps[0].(map[string]any); ok {
				parent = fmt.Sprint(p["sha"])
			}
		}
		c := f.m.NewCommit(repo, "", f.trees[fmt.Sprint(tree)], fmt.Sprint(body["message"]))
		c.Parents = []string{parent}
		js(201, map[string]any{"sha": c.SHA})
	case r.Method == "POST" && rest == "/git/refs":
		repo.Branches[strings.TrimPrefix(body["ref"].(string), "refs/heads/")] = body["sha"].(string)
		js(201, map[string]any{"ref": body["ref"]})
	case r.Method == "DELETE" && strings.HasPrefix(rest, "/git/refs/heads/"):
		delete(repo.Branches, strings.TrimPrefix(rest, "/git/refs/heads/"))
		w.WriteHeader(204)
	default:
		js(404, map[string]any{"message": "no route " + r.Method + " " + rest})
	}
}

func TestContract(t *testing.T) {
	scmtest.Run(t, func(t *testing.T) (scm.Client, scmtest.Fake) {
		m := scmtest.NewModel(scm.Account{ID: "owner", Name: "Owner"}, scm.Account{ID: "approver", Name: "Approver"})
		srv := httptest.NewServer(&fakeGitHub{m: m, blobs: map[string]string{}, trees: map[string]map[string]string{}})
		t.Cleanup(srv.Close)
		h := scm.NewHost(scm.GitHub, "", "")
		h.APIBase = srv.URL
		return New(h, secret.New("t")), m
	})
}
