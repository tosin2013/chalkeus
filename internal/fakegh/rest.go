package fakegh

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var colorRE = regexp.MustCompile(`^[0-9a-fA-F]{6}$`)

type route struct {
	method string
	re     *regexp.Regexp
	h      func(c *call)
}

// call is one request in progress.
type call struct {
	s     *Server
	w     http.ResponseWriter
	r     *http.Request
	m     []string // the route's regexp groups (m[1] is owner/name for repo routes)
	login string
	repo  *Repo
	body  map[string]any
}

func (c *call) json(status int, v any) {
	c.w.Header().Set("Content-Type", "application/json")
	c.w.WriteHeader(status)
	_ = json.NewEncoder(c.w).Encode(v)
}

func (c *call) fail(status int, msg string) { c.json(status, map[string]any{"message": msg}) }

func (c *call) base() string { return "http://" + c.r.Host }

func ts(t time.Time) string { return t.UTC().Format(time.RFC3339) }

func (s *Server) routes() []route {
	R := func(m, p string, h func(*call)) route {
		return route{m, regexp.MustCompile("^" + p + "$"), h}
	}
	repo := `/repos/([^/]+/[^/]+)`
	return []route{
		R("GET", `/user`, getUser),
		R("GET", repo, getRepo),
		R("GET", repo+`/collaborators/([^/]+)/permission`, getPermission),
		R("GET", repo+`/branches/([^/]+)/protection`, getProtection),
		R("GET", repo+`/pulls`, listPulls),
		R("POST", repo+`/pulls`, createPull),
		R("GET", repo+`/pulls/(\d+)`, getPull),
		R("PATCH", repo+`/pulls/(\d+)`, editPull),
		R("GET", repo+`/pulls/(\d+)/files`, pullFiles),
		R("GET", repo+`/issues`, listIssues),
		R("POST", repo+`/issues`, createIssue),
		R("GET", repo+`/issues/(\d+)`, getIssue),
		R("PATCH", repo+`/issues/(\d+)`, editIssue),
		R("GET", repo+`/issues/(\d+)/comments`, listComments),
		R("POST", repo+`/issues/(\d+)/comments`, createComment),
		R("GET", repo+`/issues/(\d+)/events`, listEvents),
		R("GET", repo+`/labels`, listLabels),
		R("POST", repo+`/labels`, createLabel),
		R("GET", repo+`/labels/([^/]+)`, getLabel),
		R("POST", repo+`/git/blobs`, createBlob),
		R("POST", repo+`/git/trees`, createTree),
		R("GET", repo+`/git/trees/([0-9a-f]+)`, getTree),
		R("POST", repo+`/git/commits`, createCommit),
		R("GET", repo+`/git/commits/([0-9a-f]+)`, getCommit),
		R("GET", repo+`/git/ref/heads/(.+)`, getRef),
		R("POST", repo+`/git/refs`, createRef),
		R("PATCH", repo+`/git/refs/heads/(.+)`, updateRef),
		R("DELETE", repo+`/git/refs/heads/(.+)`, deleteRef),
	}
}

// ServeREST answers the REST API (mounted at the API root).
func (s *Server) ServeREST(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.URL.Path, "/")
	if p, err := url.PathUnescape(path); err == nil {
		path = p
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	tok := r.Header.Get("Authorization")
	tok = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(tok, "token "), "Bearer "))
	c := &call{s: s, w: w, r: r, login: s.login(tok)}
	for _, rt := range s.routes() {
		if rt.method != r.Method {
			continue
		}
		if m := rt.re.FindStringSubmatch(path); m != nil {
			c.m = m
			if strings.HasPrefix(path, "/repos/") {
				c.repo = s.repos[m[1]]
				if c.repo == nil || c.login == "" || c.repo.perm(c.login) == "" {
					c.fail(404, "Not Found") // private: unknown and unreadable look the same, as on GitHub
					return
				}
				if r.Method != "GET" && !c.repo.canWrite(c.login) {
					c.fail(403, "Resource not accessible by integration")
					return
				}
			} else if c.login == "" {
				c.fail(401, "Bad credentials")
				return
			}
			if r.Body != nil && (r.Method == "POST" || r.Method == "PATCH" || r.Method == "PUT") {
				_ = json.NewDecoder(r.Body).Decode(&c.body)
			}
			rt.h(c)
			return
		}
	}
	c.fail(404, "Not Found (fakegithub has no route for "+r.Method+" "+path+")")
}

func userJSON(login string) map[string]any {
	return map[string]any{"login": login, "id": len(login), "type": "User"}
}

func getUser(c *call) { c.json(200, userJSON(c.login)) }

func getRepo(c *call) {
	p := c.repo.perm(c.login)
	owner, name, _ := strings.Cut(c.repo.Full, "/")
	c.json(200, map[string]any{"full_name": c.repo.Full, "name": name, "owner": userJSON(owner), "private": true,
		"default_branch": c.repo.Default, "html_url": c.base() + "/" + c.repo.Full,
		"permissions": map[string]bool{"admin": p == "admin", "push": c.repo.canWrite(c.login), "pull": p != ""}})
}

func getPermission(c *call) {
	p := c.repo.perm(c.m[2])
	if p == "" {
		p = "none"
	}
	c.json(200, map[string]any{"permission": p, "user": userJSON(c.m[2])})
}

func getProtection(c *call) {
	if !c.repo.Protected[c.m[2]] {
		c.fail(404, "Branch not protected")
		return
	}
	c.json(200, map[string]any{"required_pull_request_reviews": map[string]any{"required_approving_review_count": 0}})
}

func num(c *call, i int) int { n, _ := strconv.Atoi(c.m[i]); return n }

func str(m map[string]any, k string) string { s, _ := m[k].(string); return s }

func (c *call) issueJSON(i *Issue) map[string]any {
	labels := []map[string]any{}
	for _, l := range i.Labels {
		labels = append(labels, map[string]any{"name": l, "color": c.repo.Labels[l]})
	}
	kind := "issues"
	if i.PR != nil {
		kind = "pull"
	}
	out := map[string]any{"number": i.Number, "title": i.Title, "body": i.Body, "state": i.State, "labels": labels,
		"user": userJSON(i.User), "comments": len(i.Comments), "created_at": ts(i.CreatedAt), "updated_at": ts(i.UpdatedAt),
		"html_url": fmt.Sprintf("%s/%s/%s/%d", c.base(), c.repo.Full, kind, i.Number)}
	if i.PR != nil {
		out["pull_request"] = map[string]any{"url": fmt.Sprintf("%s/repos/%s/pulls/%d", c.base(), c.repo.Full, i.Number)}
	}
	return out
}

func (c *call) pullJSON(i *Issue) map[string]any {
	out := c.issueJSON(i)
	owner, _, _ := strings.Cut(c.repo.Full, "/")
	out["html_url"] = fmt.Sprintf("%s/%s/pull/%d", c.base(), c.repo.Full, i.Number)
	out["draft"], out["merged"] = i.PR.Draft, i.PR.Merged
	out["head"] = map[string]any{"ref": i.PR.Head, "sha": refSHA(c.repo.Bare, i.PR.Head), "label": owner + ":" + i.PR.Head}
	out["base"] = map[string]any{"ref": i.PR.Base, "sha": refSHA(c.repo.Bare, i.PR.Base)}
	if i.PR.Merged {
		out["merged_by"], out["merged_at"], out["merge_commit_sha"] = userJSON(i.PR.MergedBy), ts(i.PR.MergedAt), i.PR.MergeSHA
	}
	return out
}

func (c *call) sorted(pr bool) []*Issue {
	var out []*Issue
	for _, i := range c.repo.Issues {
		if !pr || i.PR != nil {
			out = append(out, i)
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Number < out[b].Number })
	return out
}

func stateMatch(want, got string) bool {
	return want == "all" || (want == "" && got == "open") || want == got
}

func listPulls(c *call) {
	q := c.r.URL.Query()
	head := q.Get("head")
	if _, b, ok := strings.Cut(head, ":"); ok {
		head = b
	}
	out := []map[string]any{}
	for _, i := range c.sorted(true) {
		if stateMatch(q.Get("state"), i.State) && (head == "" || i.PR.Head == head) && (q.Get("base") == "" || i.PR.Base == q.Get("base")) {
			out = append(out, c.pullJSON(i))
		}
	}
	c.json(200, out)
}

func (c *call) newIssue(title, body string, pr *PR) *Issue {
	now := c.s.now()
	i := &Issue{Number: c.repo.Next, Title: title, Body: body, State: "open", User: c.login, PR: pr, CreatedAt: now, UpdatedAt: now}
	c.repo.Next++
	c.repo.Issues[i.Number] = i
	return i
}

func createPull(c *call) {
	head, base := str(c.body, "head"), str(c.body, "base")
	if _, b, ok := strings.Cut(head, ":"); ok {
		head = b
	}
	if base == "" {
		base = c.repo.Default
	}
	if refSHA(c.repo.Bare, head) == "" || refSHA(c.repo.Bare, base) == "" {
		c.fail(422, "Validation Failed: head or base branch does not exist")
		return
	}
	draft, _ := c.body["draft"].(bool)
	i := c.newIssue(str(c.body, "title"), str(c.body, "body"), &PR{Head: head, Base: base, Draft: draft})
	pullRef(c.repo.Bare, i.Number, head)
	c.json(201, c.pullJSON(i))
}

func (c *call) issue(n int, pr bool) *Issue {
	i := c.repo.Issues[n]
	if i == nil || (pr && i.PR == nil) {
		c.fail(404, "Not Found")
		return nil
	}
	return i
}

func getPull(c *call) {
	if i := c.issue(num(c, 2), true); i != nil {
		c.json(200, c.pullJSON(i))
	}
}

func (c *call) edit(i *Issue) {
	if v, ok := c.body["title"].(string); ok {
		i.Title = v
	}
	if v, ok := c.body["body"].(string); ok {
		i.Body = v
	}
	if v, ok := c.body["state"].(string); ok && (v == "open" || v == "closed") && (i.PR == nil || !i.PR.Merged) {
		if v == "closed" && i.State == "open" {
			i.Events = append(i.Events, &Event{ID: c.s.id(), Kind: "closed", Actor: c.login, CreatedAt: c.s.now()})
		}
		i.State = v
	}
	if ls, ok := c.body["labels"].([]any); ok {
		var names []string
		for _, l := range ls {
			if n, ok := l.(string); ok {
				names = append(names, n)
			} else if m, ok := l.(map[string]any); ok {
				names = append(names, str(m, "name"))
			}
		}
		c.setLabels(i, names, c.login)
	}
	i.UpdatedAt = c.s.now()
}

// setLabels replaces an issue's labels, recording labeled/unlabeled events by actor.
func (c *call) setLabels(i *Issue, names []string, actor string) {
	had := map[string]bool{}
	for _, l := range i.Labels {
		had[l] = true
	}
	want := map[string]bool{}
	for _, n := range names {
		want[n] = true
		if !had[n] {
			i.Events = append(i.Events, &Event{ID: c.s.id(), Kind: "labeled", Label: n, Actor: actor, CreatedAt: c.s.now()})
		}
		if _, ok := c.repo.Labels[n]; !ok {
			c.repo.Labels[n] = "ededed"
		}
	}
	for _, l := range i.Labels {
		if !want[l] {
			i.Events = append(i.Events, &Event{ID: c.s.id(), Kind: "unlabeled", Label: l, Actor: actor, CreatedAt: c.s.now()})
		}
	}
	i.Labels = append([]string(nil), names...)
}

func editPull(c *call) {
	if i := c.issue(num(c, 2), true); i != nil {
		c.edit(i)
		if d, ok := c.body["draft"].(bool); ok {
			i.PR.Draft = d
		}
		c.json(200, c.pullJSON(i))
	}
}

func pullFiles(c *call) {
	i := c.issue(num(c, 2), true)
	if i == nil {
		return
	}
	files, err := changedFiles(c.repo.Bare, i.PR.Base, i.PR.Head)
	if err != nil {
		c.fail(500, err.Error())
		return
	}
	if files == nil {
		files = []map[string]string{}
	}
	c.json(200, files)
}

func listIssues(c *call) {
	q := c.r.URL.Query()
	out := []map[string]any{}
	for _, i := range c.sorted(false) {
		if stateMatch(q.Get("state"), i.State) {
			out = append(out, c.issueJSON(i))
		}
	}
	c.json(200, out)
}

func createIssue(c *call) {
	i := c.newIssue(str(c.body, "title"), str(c.body, "body"), nil)
	if ls, ok := c.body["labels"].([]any); ok {
		var names []string
		for _, l := range ls {
			if n, ok := l.(string); ok {
				names = append(names, n)
			}
		}
		c.setLabels(i, names, c.login)
	}
	c.json(201, c.issueJSON(i))
}

func getIssue(c *call) {
	if i := c.issue(num(c, 2), false); i != nil {
		c.json(200, c.issueJSON(i))
	}
}

func editIssue(c *call) {
	if i := c.issue(num(c, 2), false); i != nil {
		c.edit(i)
		c.json(200, c.issueJSON(i))
	}
}

func (c *call) commentJSON(i *Issue, cm *Comment) map[string]any {
	return map[string]any{"id": cm.ID, "body": cm.Body, "user": userJSON(cm.User), "created_at": ts(cm.CreatedAt),
		"updated_at": ts(cm.UpdatedAt), "html_url": fmt.Sprintf("%s/%s/issues/%d#issuecomment-%d", c.base(), c.repo.Full, i.Number, cm.ID)}
}

func listComments(c *call) {
	i := c.issue(num(c, 2), false)
	if i == nil {
		return
	}
	out := []map[string]any{}
	for _, cm := range i.Comments {
		out = append(out, c.commentJSON(i, cm))
	}
	c.json(200, out)
}

// AddComment adds a comment as login (the REST handler and the owner API share it).
func (s *Server) addComment(i *Issue, login, body string) *Comment {
	now := s.now()
	cm := &Comment{ID: s.id(), User: login, Body: body, CreatedAt: now, UpdatedAt: now}
	i.Comments = append(i.Comments, cm)
	i.UpdatedAt = now
	return cm
}

func createComment(c *call) {
	if i := c.issue(num(c, 2), false); i != nil {
		c.json(201, c.commentJSON(i, c.s.addComment(i, c.login, str(c.body, "body"))))
	}
}

func listEvents(c *call) {
	i := c.issue(num(c, 2), false)
	if i == nil {
		return
	}
	out := []map[string]any{}
	for _, e := range i.Events {
		ev := map[string]any{"id": e.ID, "event": e.Kind, "actor": userJSON(e.Actor), "created_at": ts(e.CreatedAt)}
		if e.Label != "" {
			ev["label"] = map[string]any{"name": e.Label}
		}
		out = append(out, ev)
	}
	c.json(200, out)
}

func listLabels(c *call) {
	names := make([]string, 0, len(c.repo.Labels))
	for n := range c.repo.Labels {
		names = append(names, n)
	}
	sort.Strings(names)
	out := []map[string]any{}
	for _, n := range names {
		out = append(out, map[string]any{"name": n, "color": c.repo.Labels[n]})
	}
	c.json(200, out)
}

func createLabel(c *call) {
	n := str(c.body, "name")
	if n == "" {
		c.fail(422, "Validation Failed: name is missing") // the token probe's expected answer
		return
	}
	if col := str(c.body, "color"); col != "" && !colorRE.MatchString(col) {
		c.fail(422, "Validation Failed: color is invalid") // also the token probe's expected answer
		return
	}
	if _, ok := c.repo.Labels[n]; ok {
		c.fail(422, "Validation Failed: already_exists")
		return
	}
	c.repo.Labels[n] = str(c.body, "color")
	c.json(201, map[string]any{"name": n, "color": c.repo.Labels[n]})
}

func getLabel(c *call) {
	col, ok := c.repo.Labels[c.m[2]]
	if !ok {
		c.fail(404, "Not Found")
		return
	}
	c.json(200, map[string]any{"name": c.m[2], "color": col})
}

func createBlob(c *call) {
	content := str(c.body, "content")
	if str(c.body, "encoding") == "base64" {
		b, err := decodeB64(content)
		if err != nil {
			c.fail(422, "bad base64")
			return
		}
		content = string(b)
	}
	sha, err := git(c.repo.Bare, []byte(content), nil, "hash-object", "-w", "--stdin")
	if err != nil {
		c.fail(500, err.Error())
		return
	}
	c.json(201, map[string]any{"sha": sha})
}

func createTree(c *call) {
	var entries []map[string]any
	if ts, ok := c.body["tree"].([]any); ok {
		for _, e := range ts {
			if m, ok := e.(map[string]any); ok {
				entries = append(entries, m)
			}
		}
	}
	sha, err := writeTree(c.repo.Bare, str(c.body, "base_tree"), entries)
	if err != nil {
		c.fail(422, err.Error())
		return
	}
	c.json(201, map[string]any{"sha": sha})
}

func getTree(c *call) {
	out, err := git(c.repo.Bare, nil, nil, "ls-tree", c.m[2])
	if err != nil {
		c.fail(404, "Not Found")
		return
	}
	tree := []map[string]any{}
	for _, l := range strings.Split(out, "\n") {
		meta, path, ok := strings.Cut(l, "\t")
		f := strings.Fields(meta)
		if ok && len(f) == 3 {
			tree = append(tree, map[string]any{"path": path, "mode": f[0], "type": f[1], "sha": f[2]})
		}
	}
	c.json(200, map[string]any{"sha": c.m[2], "tree": tree})
}

func createCommit(c *call) {
	args := []string{"commit-tree", str(c.body, "tree"), "-m", str(c.body, "message")}
	if ps, ok := c.body["parents"].([]any); ok {
		for _, p := range ps {
			if s, ok := p.(string); ok {
				args = append(args, "-p", s)
			}
		}
	}
	sha, err := git(c.repo.Bare, nil, []string{"GIT_AUTHOR_NAME=" + c.login, "GIT_COMMITTER_NAME=" + c.login}, args...)
	if err != nil {
		c.fail(422, err.Error())
		return
	}
	c.json(201, map[string]any{"sha": sha, "tree": map[string]any{"sha": str(c.body, "tree")}})
}

func getCommit(c *call) {
	tree, err := git(c.repo.Bare, nil, nil, "rev-parse", c.m[2]+"^{tree}")
	if err != nil {
		c.fail(404, "Not Found")
		return
	}
	parents := []map[string]any{}
	if out, err := git(c.repo.Bare, nil, nil, "rev-list", "--parents", "-n", "1", c.m[2]); err == nil {
		for _, p := range strings.Fields(out)[1:] {
			parents = append(parents, map[string]any{"sha": p})
		}
	}
	msg, _ := git(c.repo.Bare, nil, nil, "log", "-1", "--format=%B", c.m[2])
	c.json(200, map[string]any{"sha": c.m[2], "tree": map[string]any{"sha": tree}, "parents": parents, "message": msg})
}

func refJSON(branch, sha string) map[string]any {
	return map[string]any{"ref": "refs/heads/" + branch, "object": map[string]any{"sha": sha, "type": "commit"}}
}

func getRef(c *call) {
	sha := refSHA(c.repo.Bare, c.m[2])
	if sha == "" {
		c.fail(404, "Not Found")
		return
	}
	c.json(200, refJSON(c.m[2], sha))
}

func createRef(c *call) {
	branch := strings.TrimPrefix(str(c.body, "ref"), "refs/heads/")
	if refSHA(c.repo.Bare, branch) != "" {
		c.fail(422, "Reference already exists")
		return
	}
	if _, err := git(c.repo.Bare, nil, nil, "update-ref", "refs/heads/"+branch, str(c.body, "sha")); err != nil {
		c.fail(422, err.Error())
		return
	}
	c.json(201, refJSON(branch, str(c.body, "sha")))
}

func updateRef(c *call) {
	if _, err := git(c.repo.Bare, nil, nil, "update-ref", "refs/heads/"+c.m[2], str(c.body, "sha")); err != nil {
		c.fail(422, err.Error())
		return
	}
	c.json(200, refJSON(c.m[2], str(c.body, "sha")))
}

func deleteRef(c *call) {
	if refSHA(c.repo.Bare, c.m[2]) == "" {
		c.fail(422, "Reference does not exist")
		return
	}
	if _, err := git(c.repo.Bare, nil, nil, "update-ref", "-d", "refs/heads/"+c.m[2]); err != nil {
		c.fail(500, err.Error())
		return
	}
	c.w.WriteHeader(204)
}
