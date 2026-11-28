package readiness

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tosin2013/chalkeus/internal/config"
	"github.com/tosin2013/chalkeus/internal/proof"
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
	companionPath     string
	skipHidden        bool
	allRules          bool
	slice             string                // 026: the slice to check
	prior             map[string]PriorSlice // 026: locally recorded slicing
	// 029: a bump commit on a Dependabot branch (files it changes), and the item's spec, written
	// on main after it with the bump's commit id.
	bump     map[string]string
	bumpSpec func(sha string) string
	// 041: by default every hidden-test folder gets a matching proof; noProof leaves them out.
	noProof bool
}

// withProofs adds a proven proof for every item and slice folder of the companion (041), as
// author submit or author prove would.
func withProofs(t *testing.T, files map[string]string, companionPath string) map[string]string {
	t.Helper()
	tmp := t.TempDir()
	for p, b := range files {
		_ = os.MkdirAll(filepath.Dir(filepath.Join(tmp, p)), 0o755)
		_ = os.WriteFile(filepath.Join(tmp, p), []byte(b), 0o644)
	}
	tg := config.Target{CompanionPath: companionPath}
	out := map[string]string{}
	for p, b := range files {
		out[p] = b
	}
	items, _ := os.ReadDir(filepath.Join(tmp, filepath.FromSlash(tg.CompanionItemDir(""))))
	for _, it := range items {
		if !it.IsDir() {
			continue
		}
		dirs := map[string]string{"": tg.CompanionItemDir(it.Name())}
		subs, _ := os.ReadDir(filepath.Join(tmp, filepath.FromSlash(tg.CompanionItemDir(it.Name()))))
		for _, s := range subs {
			if s.IsDir() {
				dirs[s.Name()] = tg.CompanionSliceDir(it.Name(), s.Name())
			}
		}
		for sl, d := range dirs {
			tree, _ := proof.TreeSHA256(filepath.Join(tmp, filepath.FromSlash(d)))
			pf := proof.Proof{Version: 1, Item: it.Name(), Slice: sl, Reference: "patch", TestsTreeSHA256: tree, Result: proof.Proven}
			out[tg.CompanionProofPath(it.Name(), sl)] = string(pf.Marshal())
		}
	}
	return out
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
	if w.bump != nil {
		g := func(args ...string) string {
			out, err := exec.Command("git", append([]string{"-C", tgt, "-c", "user.name=t", "-c", "user.email=t@invalid"}, args...)...).CombinedOutput()
			if err != nil {
				t.Fatalf("git %v: %s", args, out)
			}
			return strings.TrimSpace(string(out))
		}
		g("checkout", "-q", "-b", "dependabot/maven")
		for p, body := range w.bump {
			_ = os.WriteFile(filepath.Join(tgt, p), []byte(body), 0o644)
		}
		g("add", "-A")
		g("commit", "-qm", "chore(deps): bump junit")
		sha := g("rev-parse", "HEAD")
		g("checkout", "-q", "main")
		_ = os.WriteFile(filepath.Join(tgt, "specs/002-feature/spec.md"), []byte(w.bumpSpec(sha)), 0o644)
		g("commit", "-qam", "spec names the bump")
	}
	companion := w.companion
	if !w.noProof {
		companion = withProofs(t, w.companion, w.companionPath)
	}
	comp := gitRepo(t, companion)
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
		SkipHidden:  w.skipHidden,
		AllRules:    w.allRules,
		Slice:       w.slice,
		PriorSlices: w.prior,
	}
	tg := config.Target{Name: "t", Repo: "me/target", Branch: "main", Toolchain: "jdk21-maven", CompanionRepo: "me/target-acceptance", CompanionPath: w.companionPath, MaxInputMB: maxMB}
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

// One shared companion repository serves several targets: each target's items sit under its
// own folder (PILOT_COMPANION_PATH), and only that folder counts.
func TestSharedCompanionPath(t *testing.T) {
	w := newWorld()
	w.companionPath = "target"
	w.companion = map[string]string{"target/items/002/tests/test_x.py": "x", "other/items/002/tests/test_y.py": "y", "items/002/z.py": "z"}
	if res, _ := w.check(t, "002", 50); res.Decision != "allow" || res.HiddenFiles != 1 {
		t.Fatalf("%+v", res)
	}
	w.companion = map[string]string{"other/items/002/tests/test_y.py": "y", "items/002/z.py": "z"}
	if res, _ := w.check(t, "002", 50); res.Decision != "refuse" || !contains(res.Reasons, "ITEM_NO_HIDDEN_TESTS") {
		t.Fatalf("another target's tests must not count: %+v", res)
	}
}

// 018: the leak path. A hidden test's content already in the target refuses the run.
func TestHiddenTestsInTarget(t *testing.T) {
	w := newWorld()
	w.target["src/test/java/Copied.java"] = w.companion["items/002/src/test/java/AcceptanceTest.java"]
	if res, _ := w.check(t, "002", 50); res.Decision != "refuse" || !contains(res.Reasons, "HIDDEN_TESTS_IN_TARGET") {
		t.Fatalf("%+v", res)
	}
}

// 018: the test author's readiness doesn't need hidden tests, and reports what is there.
func TestSkipHiddenForTheAuthor(t *testing.T) {
	w := newWorld()
	w.skipHidden = true
	w.companion = map[string]string{"items/003/x.java": "x"}
	if res, _ := w.check(t, "002", 50); res.Decision != "allow" || res.HiddenFiles != 0 || res.SpecSHA256 == "" {
		t.Fatalf("%+v", res)
	}
}

// 018: the coverage map is reported, never refused, and not counted as a hidden test.
func TestMappingReport(t *testing.T) {
	w := newWorld()
	w.companion["items/002/"+MapFile] = `{"spec_sha256":"old","uncovered":["US1-AS2"]}`
	res, _ := w.check(t, "002", 50)
	if res.Decision != "allow" || res.HiddenFiles != 1 || res.Mapping["present"] != true || res.Mapping["stale"] != true ||
		fmt.Sprint(res.Mapping["uncovered"]) != "[US1-AS2]" {
		t.Fatalf("%+v", res)
	}
	delete(w.companion, "items/002/"+MapFile)
	if res, _ := w.check(t, "002", 50); res.Mapping["present"] != false {
		t.Fatalf("%+v", res.Mapping)
	}
}

// doctor: every failing rule is reported, not only the first; runs still stop at the first.
func TestAllRulesReportsEveryProblem(t *testing.T) {
	w := newWorld()
	w.companion = map[string]string{"items/003/x.java": "x"}
	w.heads = []string{"pilot/20990101T000000Z/002"}
	res, _ := w.check(t, "002", 0) // no hidden tests, too large, an open item PR
	if res.Decision != "refuse" || len(res.Reasons) != 1 || res.Reasons[0] != "ITEM_NO_HIDDEN_TESTS" {
		t.Fatalf("a run must stop at the first problem: %v", res.Reasons)
	}
	w.allRules = true
	res, _ = w.check(t, "002", 0)
	if res.Decision != "refuse" || strings.Join(res.Reasons, ",") != "ITEM_NO_HIDDEN_TESTS,INPUT_TOO_LARGE,OPEN_ITEM_PR" {
		t.Fatalf("doctor must list every problem: %v", res.Reasons)
	}
	w2 := newWorld()
	w2.allRules = true
	if res, _ := w2.check(t, "002", 50); res.Decision != "allow" {
		t.Fatalf("a prepared item still allows in all-rules mode: %v", res.Reasons)
	}
}

// 029: a dependency item's manifest change is checked in readiness; a bad one refuses with its
// code (gate), and doctor (AllRules) lists every code at once.
func TestManifestChangeReadiness(t *testing.T) {
	pom := func(v string) string { return "<project><properties><junit.version>" + v + "</junit.version></properties></project>\n" }
	spec := func(files string) func(string) string {
		return func(sha string) string {
			return goodSpec + "\n## Manifest change\n\n- Commit: `" + sha + "`\n- Files: " + files + "\n"
		}
	}
	w := newWorld()
	w.target["pom.xml"] = pom("5.10.0")
	w.bump, w.bumpSpec = map[string]string{"pom.xml": pom("5.11.0")}, spec("`pom.xml`")
	res, _ := w.check(t, "002", 50)
	if res.Decision != "allow" || res.ManifestChange == nil || len(res.ManifestChange.Changes) != 1 {
		t.Fatalf("%+v %+v", res.Reasons, res.ManifestChange)
	}

	w = newWorld()
	w.target["pom.xml"] = pom("5.10.0")
	w.bump, w.bumpSpec = map[string]string{"pom.xml": pom("5.11.0")}, spec("`pom.xml`, `src/main/java/App.java`")
	res, _ = w.check(t, "002", 50)
	if res.Decision != "refuse" || res.Reasons[0] != "MANIFEST_CHANGE_NOT_MANIFEST" || res.ManifestChange != nil {
		t.Fatalf("gate: %v", res.Reasons)
	}

	w = newWorld()
	w.target["pom.xml"] = pom("5.10.0")
	w.bump, w.bumpSpec = map[string]string{"pom.xml": pom("5.11.0")}, spec("`pom.xml`, `src/main/java/App.java`")
	w.slice, w.allRules = "", true
	res, _ = w.check(t, "002", 50)
	if strings.Join(res.Reasons, ",") != "MANIFEST_CHANGE_NOT_MANIFEST" {
		t.Fatalf("doctor: %v", res.Reasons)
	}

	w = newWorld()
	w.target["specs/002-feature/spec.md"] = goodSpec + "\n## Manifest change\n\n- Commit: `abc`\n"
	res, _ = w.check(t, "002", 50)
	if res.Decision != "refuse" || res.Reasons[0] != "MANIFEST_CHANGE_UNREADABLE" {
		t.Fatalf("unreadable: %v", res.Reasons)
	}
}
