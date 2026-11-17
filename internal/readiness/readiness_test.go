package readiness

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tosin2013/specfoundry/internal/config"
)

// A fake spec-status: the JSON a prepared target would produce, adjustable per test.
type fakeStatus struct {
	exit *int
	doc  map[string]any
}

func (f fakeStatus) Status(string) (*int, string) {
	if f.exit == nil {
		return nil, ""
	}
	raw, _ := json.Marshal(f.doc)
	return f.exit, string(raw)
}
func (f fakeStatus) Version() string { return "spec 0.0.0-test" }

func zero() *int { z := 0; return &z }

func statusDoc() map[string]any {
	item := func(id, status, specDir string, deps ...map[string]any) map[string]any {
		if deps == nil {
			deps = []map[string]any{}
		}
		return map[string]any{"id": id, "status": status, "status_known": true, "active": true, "spec_dir": specDir, "dependencies": deps}
	}
	return map[string]any{"schema_version": "1.0", "complete": true, "error": nil,
		"source":  map[string]any{"path": ".specify/memory/roadmap.md"},
		"summary": map[string]any{"blocked_count": 1},
		"items": []any{
			item("001", "verified", "specs/001-base"),
			item("002", "planned", "specs/002-feature", map[string]any{"id": "001", "satisfied": true}),
			item("003", "planned", "specs/003-next", map[string]any{"id": "002", "satisfied": false}),
		},
		"diagnostics": []any{}}
}

const goodSpec = "# 002\n\n## Allowed paths\n\n- `src/main/java/*`\n- `src/test/java/*`\n"

func gitRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	d := t.TempDir()
	for p, body := range files {
		_ = os.MkdirAll(filepath.Join(d, filepath.Dir(p)), 0o755)
		if err := os.WriteFile(filepath.Join(d, p), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, a := range [][]string{{"init", "-q", "-b", "main"}, {"add", "-A"},
		{"-c", "user.name=t", "-c", "user.email=t@invalid", "commit", "-qm", "base", "--allow-empty"}} {
		if out, err := exec.Command("git", append([]string{"-C", d}, a...)...).CombinedOutput(); err != nil {
			t.Fatalf("%v: %s", a, out)
		}
	}
	return d
}

type world struct {
	target, companion map[string]string
	status            fakeStatus
	heads             []string
	headsErr          error
	cloneFails        map[string]bool
}

func newWorld() *world {
	return &world{
		target: map[string]string{
			".specify/memory/roadmap.md": "# Roadmap\n\n## Open Questions\n\n- Q1 — naming (affects 003)\n- ~~Q2 — old question about 002~~\n- Q3 — resolved: 002 uses JSON\n",
			"specs/002-feature/spec.md":  goodSpec,
			"specs/002-feature/tasks.md": "- [ ] T001 do it\n",
			"src/main/java/App.java":     "class App {}\n",
		},
		companion: map[string]string{"items/002/src/test/java/AcceptanceTest.java": "class AcceptanceTest {}\n"},
		status:    fakeStatus{exit: zero(), doc: statusDoc()},
	}
}

func (w *world) check(t *testing.T, item string, maxMB int) (Result, string) {
	t.Helper()
	tgt := gitRepo(t, w.target)
	comp := gitRepo(t, w.companion)
	out := t.TempDir()
	d := Deps{Src: w.status,
		Clone: func(repo, branch string) (string, string, error) {
			if w.cloneFails[repo] {
				return "", "", errors.New("clone failed")
			}
			src := map[string]string{"me/target": tgt, "me/target-acceptance": comp}[repo]
			dst := filepath.Join(t.TempDir(), "c")
			if out, err := exec.Command("git", "clone", "-q", src, dst).CombinedOutput(); err != nil {
				t.Fatalf("%s", out)
			}
			head, _ := exec.Command("git", "-C", dst, "rev-parse", "HEAD").Output()
			return dst, strings.TrimSpace(string(head)), nil
		},
		OpenPRHeads: func(string) ([]string, error) { return w.heads, w.headsErr },
	}
	tg := config.Target{Name: "t", Repo: "me/target", Branch: "main", CompanionRepo: "me/target-acceptance", MaxInputMB: maxMB}
	res, err := Check(d, tg, item, 3, out)
	if err != nil {
		t.Fatal(err)
	}
	return res, out
}

func TestPreparedItemAllows(t *testing.T) {
	res, out := newWorld().check(t, "002", 50)
	if res.Decision != "allow" || res.SpecDir != "specs/002-feature" || res.HiddenFiles != 1 || res.CompanionCommit == "" ||
		strings.Join(res.AllowedPaths, ",") != "src/main/java/*,src/test/java/*,specs/002-feature/tasks.md" {
		t.Fatalf("%+v", res)
	}
	for _, f := range []string{"3-002.json", "3-002.status.json", "3-002.readiness.json"} {
		if _, err := os.Stat(filepath.Join(out, f)); err != nil {
			t.Errorf("missing %s", f)
		}
	}
	raw, _ := os.ReadFile(filepath.Join(out, "3-002.readiness.json"))
	if strings.Contains(string(raw), "AcceptanceTest") {
		t.Error("readiness record carries hidden test names")
	}
}

// Each rule, broken alone, refuses with its own code (contracts/readiness.md).
func TestEachRuleRefusesAlone(t *testing.T) {
	cases := []struct {
		name, item, code string
		mb               int
		breakIt          func(w *world)
	}{
		{"bad id", "2", "ITEM_NOT_ON_ROADMAP", 50, func(*world) {}},
		{"not on roadmap", "009", "ITEM_NOT_ON_ROADMAP", 50, func(*world) {}},
		{"dependency", "003", "DEP_UNSATISFIED:002", 50, func(*world) {}},
		{"already verified", "001", "ALREADY_VERIFIED", 50, func(*world) {}},
		{"unreadable status", "002", "UNEVALUABLE:spawn_failed", 50, func(w *world) { w.status.exit = nil }},
		{"target clone", "002", "UNEVALUABLE:clone_failed", 50, func(w *world) { w.cloneFails = map[string]bool{"me/target": true} }},
		{"no spec dir", "002", "ITEM_SPEC_MISSING", 50, func(w *world) {
			w.status.doc["items"].([]any)[1].(map[string]any)["spec_dir"] = nil
		}},
		{"spec missing", "002", "ITEM_SPEC_MISSING", 50, func(w *world) { delete(w.target, "specs/002-feature/spec.md") }},
		{"tasks missing", "002", "ITEM_TASKS_MISSING", 50, func(w *world) { delete(w.target, "specs/002-feature/tasks.md") }},
		{"clarification", "002", "ITEM_OPEN_QUESTIONS", 50, func(w *world) {
			w.target["specs/002-feature/spec.md"] = goodSpec + "\n[NEEDS CLARIFICATION: which format?]\n"
		}},
		{"roadmap question", "002", "ITEM_OPEN_QUESTIONS", 50, func(w *world) {
			w.target[".specify/memory/roadmap.md"] += "- Q4 — does 002 need auth?\n"
		}},
		{"error diagnostic", "002", "ITEM_ERROR_DIAGNOSTIC:SS_X", 50, func(w *world) {
			w.status.doc["diagnostics"] = []any{map[string]any{"item_id": "002", "severity": "error", "code": "SS_X"}}
		}},
		{"no allowed paths", "002", "ITEM_NO_ALLOWED_PATHS", 50, func(w *world) { w.target["specs/002-feature/spec.md"] = "# 002\n" }},
		{"paths too wide", "002", "ITEM_PATHS_TOO_WIDE", 50, func(w *world) {
			w.target["specs/002-feature/spec.md"] = "## Allowed paths\n- `specs/*`\n"
		}},
		{"no hidden tests", "002", "ITEM_NO_HIDDEN_TESTS", 50, func(w *world) {
			w.companion = map[string]string{"items/003/x.java": "x"}
		}},
		{"companion unreadable", "002", "ITEM_NO_HIDDEN_TESTS", 50, func(w *world) {
			w.cloneFails = map[string]bool{"me/target-acceptance": true}
		}},
		{"too large", "002", "INPUT_TOO_LARGE", 0, func(*world) {}},
		{"open item PR", "002", "OPEN_ITEM_PR", 50, func(w *world) { w.heads = []string{"feature/x", "pilot/20990101T000000Z/002"} }},
		{"PR list fails", "002", "OPEN_ITEM_PR", 50, func(w *world) { w.headsErr = errors.New("rate limited") }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := newWorld()
			c.breakIt(w)
			res, _ := w.check(t, c.item, c.mb)
			if res.Decision != "refuse" || !contains(res.Reasons, c.code) {
				t.Fatalf("want refuse %s, got %s %v", c.code, res.Decision, res.Reasons)
			}
		})
	}
}

// An open PR for another item, and struck or resolved questions naming the item, don't refuse.
func TestNonMatchesDontRefuse(t *testing.T) {
	w := newWorld()
	w.heads = []string{"pilot/20990101T000000Z/003", "pilot/20990101T000000Z/0021"}
	if res, _ := w.check(t, "002", 50); res.Decision != "allow" {
		t.Fatal(res.Reasons)
	}
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
