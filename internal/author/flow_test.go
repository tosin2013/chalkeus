package author

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/tosin2013/specfoundry/internal/config"
	"github.com/tosin2013/specfoundry/internal/fixture"
	"github.com/tosin2013/specfoundry/internal/ghloop"
	"github.com/tosin2013/specfoundry/internal/readiness"
)

const marker = "SECRET_HIDDEN_EXPECTATION_77"

// fakeGitHub is the slice of the GitHub API the test author uses.
type fakeGitHub struct {
	mu        sync.Mutex
	push      map[string]bool // repo → can push
	blobs     []string
	tree      []map[string]any
	prBody    string
	prTitle   string
	openHeads []string
}

func (f *fakeGitHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p := r.URL.Path
	body, _ := io.ReadAll(r.Body)
	var in map[string]any
	_ = json.Unmarshal(body, &in)
	out := func(v any) { w.Header().Set("Content-Type", "application/json"); _ = json.NewEncoder(w).Encode(v) }
	switch {
	case p == "/user":
		out(map[string]any{"login": "op"})
	case strings.HasSuffix(p, "/pulls") && r.Method == http.MethodGet:
		var prs []any
		for _, h := range f.openHeads {
			prs = append(prs, map[string]any{"html_url": "https://x/pr/1", "head": map[string]any{"ref": h}})
		}
		out(prs)
	case strings.HasSuffix(p, "/pulls"):
		f.prBody, f.prTitle = fmt.Sprint(in["body"]), fmt.Sprint(in["title"])
		out(map[string]any{"html_url": "https://x/pr/9", "number": 9, "head": map[string]any{"ref": in["head"]}})
	case strings.Contains(p, "/git/ref"):
		if r.Method == http.MethodPost {
			out(map[string]any{"ref": in["ref"]})
		} else {
			out(map[string]any{"object": map[string]any{"sha": "parent1"}})
		}
	case strings.HasSuffix(p, "/git/blobs"):
		f.blobs = append(f.blobs, fmt.Sprint(in["content"]))
		out(map[string]any{"sha": fmt.Sprintf("blob%d", len(f.blobs))})
	case strings.HasSuffix(p, "/git/trees"):
		for _, e := range in["tree"].([]any) {
			f.tree = append(f.tree, e.(map[string]any))
		}
		out(map[string]any{"sha": "tree2"})
	case strings.Contains(p, "/git/commits/"):
		out(map[string]any{"sha": "parent1", "tree": map[string]any{"sha": "tree1"}})
	case strings.HasSuffix(p, "/git/commits"):
		out(map[string]any{"sha": "commit3"})
	case strings.HasPrefix(p, "/repos/"):
		repo := strings.TrimPrefix(p, "/repos/")
		out(map[string]any{"full_name": repo, "default_branch": "main", "permissions": map[string]any{"push": f.push[repo]}})
	default:
		http.NotFound(w, r)
	}
}

func gitRepo(t *testing.T, root, name string, files map[string]string) string {
	t.Helper()
	d := filepath.Join(root, name)
	_ = os.MkdirAll(d, 0o755)
	for p, b := range files {
		write(t, d, p, b)
	}
	for _, a := range [][]string{{"init", "-q", "-b", "main"}, {"add", "-A"}, {"-c", "user.name=t", "-c", "user.email=t@invalid", "commit", "-qm", "base"}} {
		if out, err := exec.Command("git", append([]string{"-C", d}, a...)...).CombinedOutput(); err != nil {
			t.Fatalf("%v %s", a, out)
		}
	}
	head, _ := exec.Command("git", "-C", d, "rev-parse", "HEAD").Output()
	return strings.TrimSpace(string(head))
}

type world struct {
	env  Env
	fake *fakeGitHub
	out  *bytes.Buffer
	res  readiness.Result
}

func newWorld(t *testing.T, level string) *world {
	t.Helper()
	fake := &fakeGitHub{push: map[string]bool{"me/acc": true, "me/proj": true}}
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	oldAPI, oldBase := ghloop.API, fixture.BaseURL
	ghloop.API, fixture.BaseURL = srv.URL, t.TempDir()+"/"
	t.Cleanup(func() { ghloop.API, fixture.BaseURL = oldAPI, oldBase })
	spec := "# 006\\n\\n### User Story 1 - x (Priority: P1)\\n\\n**Acceptance Scenarios**:\\n\\n1. **Given** a, **Then** b.\\n\\n## Requirements\\n\\n- **FR-001**: do it.\\n"
	spec = strings.ReplaceAll(spec, "\\n", "\n")
	commit := gitRepo(t, fixture.BaseURL, "me/proj.git", map[string]string{"app.py": "x = 1\n", "specs/006-x/spec.md": spec,
		"specs/006-x/tasks.md": "- [ ] T001\n", "specs/006-x/plan.md": "plan\n"})
	_ = os.Setenv("GH_TOKEN", "operator-token")
	t.Cleanup(func() { _ = os.Unsetenv("GH_TOKEN") })
	cfgDir := t.TempDir()
	tg := config.Target{Name: "proj", Repo: "me/proj", Branch: "main", CompanionRepo: "me/acc", CompanionPath: "proj",
		SecurityLevel: level, AuthorEnv: filepath.Join(cfgDir, "proj.author.env")}
	w := &world{fake: fake, out: &bytes.Buffer{}, res: readiness.Result{Decision: "allow", Commit: commit, SpecDir: "specs/006-x",
		Title: "bundled flags", SpecSHA256: "spec-sha", AllowedPaths: []string{"app.py"}}}
	w.env = Env{Target: tg, EvidenceRoot: t.TempDir(), BuildDir: t.TempDir(), Out: w.out,
		Ready:         func(config.Target, string) (readiness.Result, error) { return w.res, nil },
		Identity:      ResolveIdentity,
		OpenAuthorPRs: OpenAuthorPRs}
	return w
}

func TestStartMinimalRecordsThatTheIdentityCanWriteTheTarget(t *testing.T) {
	w := newWorld(t, config.LevelMinimal)
	d, err := Start(context.Background(), w.env, "006", false)
	if err != nil {
		t.Fatal(err)
	}
	if d.Level != "minimal" || d.Identity != "op" || !d.CanWriteTarget || !strings.Contains(w.out.String(), "Protect its main branch") {
		t.Fatalf("%+v\n%s", d, w.out)
	}
	for _, f := range []string{"spec.md", "plan.md", "tasks.md", "criteria.json", "base/app.py"} {
		if _, err := os.Stat(filepath.Join(d.Workspace, f)); err != nil {
			t.Errorf("workspace lacks %s", f)
		}
	}
	if err := os.WriteFile(filepath.Join(d.Workspace, "base", "app.py"), []byte("y"), 0o644); err == nil {
		t.Error("base/ must be read-only")
	}
	raw, _ := os.ReadFile(filepath.Join(d.Workspace, "criteria.json"))
	if !strings.Contains(string(raw), `"US1-AS1"`) || !strings.Contains(string(raw), `"FR-001"`) {
		t.Fatal(string(raw))
	}
}

func TestStartRefusals(t *testing.T) {
	w := newWorld(t, config.LevelRecommended)
	_ = os.WriteFile(w.env.Target.AuthorEnv, []byte("PILOT_AUTHOR_TOKEN=author-token\n"), 0o600)
	if _, err := Start(context.Background(), w.env, "006", false); code(err) != CodeCanWriteTarget {
		t.Fatalf("recommended with a token that can push to the target: %v", err)
	}
	w.fake.push["me/proj"] = false
	if _, err := Start(context.Background(), w.env, "006", false); err != nil {
		t.Fatalf("recommended, companion-only token: %v", err)
	}
	w.res.HiddenFiles = 2
	if _, err := Start(context.Background(), w.env, "006", false); code(err) != CodeCompanionNotEmpty {
		t.Fatalf("%v", err)
	}
	w.res.HiddenFiles = 0
	w.fake.openHeads = []string{"author/006-20990101t000000z"}
	if _, err := Start(context.Background(), w.env, "006", false); code(err) != CodeDraftPROpen {
		t.Fatalf("%v", err)
	}
	w.env.Target.SecurityLevel = config.LevelStrict
	if _, err := Start(context.Background(), w.env, "006", false); code(err) != CodeNeedsClusterMode {
		t.Fatalf("%v", err)
	}
	w.env.Target.SecurityLevel = config.LevelMinimal
	w.fake.openHeads = nil
	w.res = readiness.Result{Decision: "refuse", Reasons: []string{"ITEM_OPEN_QUESTIONS"}}
	if _, err := Start(context.Background(), w.env, "006", false); code(err) != "ITEM_OPEN_QUESTIONS" {
		t.Fatalf("%v", err)
	}
}

// draftAndCheck starts a draft, writes a test and a map, and runs check with a fake baseline.
func draftAndCheck(t *testing.T, w *world, baselineResult string) (*Draft, error) {
	t.Helper()
	d, err := Start(context.Background(), w.env, "006", false)
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(d.Workspace, "draft"), "tests/test_bundled.py", "def test_it():\n    assert False, '"+marker+"'\n")
	m, _ := json.Marshal(Map{Version: 1, SpecSHA256: "spec-sha", Criteria: map[string][]string{"US1-AS1": {"tests/test_bundled.py::test_it"}}, Uncovered: []string{"FR-001"}})
	write(t, filepath.Join(d.Workspace, "draft"), MapFile, string(m))
	fake := func(dir, id, item string, res readiness.Result, overlay []byte) (map[string]any, []byte, error) {
		if res.Commit != w.res.Commit || !bytes.Contains(overlay, []byte(marker)) || bytes.Contains(overlay, []byte("spec_sha256")) {
			t.Fatalf("baseline got the wrong inputs: commit=%s", res.Commit)
		}
		checks := []any{map[string]any{"id": "G4_visible_tests", "result": "pass"}, map[string]any{"id": "G5_hidden_tests", "result": "pass"}}
		if baselineResult == "hidden_passes" {
			checks[1] = map[string]any{"id": "G5_hidden_tests", "result": "fail"}
		}
		r := map[string]any{"result": "pass", "checks": checks}
		if baselineResult != "pass" {
			r["result"] = "fail"
		}
		return r, []byte("AssertionError: " + marker), nil
	}
	return Check(context.Background(), w.env, "006", fake)
}

func TestCheckAndSubmit(t *testing.T) {
	w := newWorld(t, config.LevelMinimal)
	if _, err := Submit(context.Background(), w.env, "006"); err == nil {
		t.Fatal("submit without a draft must fail")
	}
	d, err := draftAndCheck(t, w, "hidden_passes")
	if code(err) != CodeDraftHiddenPasses {
		t.Fatalf("%v", err)
	}
	if _, err := Submit(context.Background(), w.env, "006"); code(err) != CodeDraftNotChecked {
		t.Fatalf("a failed check can't be submitted: %v", err)
	}
	d, err = draftAndCheck(t, w, "pass")
	if err != nil || d.State != "checked(pass)" || !strings.Contains(w.out.String(), marker) {
		t.Fatalf("the author sees its own output: %v\n%s", err, w.out)
	}
	write(t, filepath.Join(d.Workspace, "draft"), "tests/test_more.py", "x")
	if _, err := Submit(context.Background(), w.env, "006"); code(err) != CodeDraftNotChecked {
		t.Fatalf("a changed draft must be checked again: %v", err)
	}
	_ = os.Remove(filepath.Join(d.Workspace, "draft", "tests", "test_more.py"))
	d, err = Submit(context.Background(), w.env, "006")
	if err != nil || d.State != "submitted" || d.PR["url"] != "https://x/pr/9" {
		t.Fatalf("%+v %v", d, err)
	}
	var paths []string
	for _, e := range w.fake.tree {
		paths = append(paths, fmt.Sprint(e["path"]))
	}
	if strings.Join(paths, ",") != "proj/items/006/"+MapFile+",proj/items/006/tests/test_bundled.py" {
		t.Fatalf("files outside the item folder: %v", paths)
	}
	if !strings.HasPrefix(w.fake.prTitle, "test(006): hidden acceptance tests for bundled flags") ||
		!strings.Contains(w.fake.prBody, "| US1-AS1 |") || !strings.Contains(w.fake.prBody, "**uncovered**") {
		t.Fatalf("%s\n%s", w.fake.prTitle, w.fake.prBody)
	}
	// 018 US2: no test contents in the PR body or in any evidence file.
	if strings.Contains(w.fake.prBody, marker) {
		t.Fatal("the PR body carries test contents")
	}
	_ = filepath.Walk(d.Dir(), func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			raw, _ := os.ReadFile(p)
			if bytes.Contains(raw, []byte(marker)) {
				t.Errorf("%s carries test contents", p)
			}
		}
		return nil
	})
}
