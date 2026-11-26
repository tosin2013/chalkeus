package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tosin2013/chalkeus/internal/config"
	"github.com/tosin2013/chalkeus/internal/evidence"
	"github.com/tosin2013/chalkeus/internal/paths"
)

func itemP(t *testing.T) (*P, *evidence.Run) {
	t.Helper()
	p, r, _ := newP(t)
	root, _ := filepath.Abs("../..")
	p.L = paths.At(root)
	p.L.Build = t.TempDir()
	p.Target = config.Target{Name: "greeting", Repo: "me/greeting-service", Branch: "main", Toolchain: "python",
		TestCommand: "python3 -m pytest -q", RegistryHosts: []string{"pypi.org"}, MaxInputMB: 50, SecurityLevel: "minimal"}
	r.SetItemRun(map[string]any{"name": "greeting", "repo": "me/greeting-service"}, "002")
	r.State["readiness"] = map[string]any{"spec_dir": "specs/002-x", "allowed_paths": []any{"app.py", "specs/002-x/tasks.md"}}
	r.State["snapshot"] = map[string]any{"base_revision": "b0", "snapshot_commit": "s0"}
	return p, r
}

func TestItemModeText(t *testing.T) {
	p, r := itemP(t)
	got := p.modeExtra(r)
	want := "  target greeting (me/greeting-service@main), item 002, toolchain python\n  egress: model host(s) 1 (from forge.env); registries: pypi.org" +
		"\n  security level: minimal; hidden tests: 0 file(s), no coverage map"
	if got != want {
		t.Fatalf("%q", got)
	}
	r.State["readiness"].(map[string]any)["hidden_files"] = json.Number("2") // as read back from run.json on resume
	r.State["readiness"].(map[string]any)["mapping"] = map[string]any{"present": true, "valid": true, "uncovered": []any{"US2-AS1"}, "stale": true}
	if c := coverage(r); c != "2 file(s), uncovered criteria: [US2-AS1] (map written for an older version of the spec)" {
		t.Fatal(c)
	}
	r.State["readiness"].(map[string]any)["mapping"] = map[string]any{"present": true, "valid": true, "uncovered": []any{}}
	if c := coverage(r); c != "2 file(s), every criterion covered" {
		t.Fatal(c)
	}
	delete(r.State["readiness"].(map[string]any), "mapping")
	p.Target.ToolchainImage = "quay.io/me/tc@sha256:" + strings.Repeat("a", 64)
	if !strings.Contains(p.modeExtra(r), "toolchain python (operator-supplied quay.io/me/tc@sha256:") {
		t.Error(p.modeExtra(r))
	}
	fixture, fr, _ := newP(t)
	if fixture.modeExtra(fr) != "" {
		t.Error("the fixture's mode text must not change")
	}
}

func TestItemRunInputs(t *testing.T) {
	p, r := itemP(t)
	w := p.workerItem(r)
	if w.SnapshotCommit != "s0" || w.SpecDir != "specs/002-x" || strings.Join(w.AllowedPaths, ",") != "app.py,specs/002-x/tasks.md" ||
		w.TestCommand != "python3 -m pytest -q" || w.CommandTimeout != 300 || w.InputLimit != "116Mi" {
		t.Fatalf("%+v", w)
	}
	g := p.itemGrading(r)
	if g.NS != "pilot-g-002" || g.CPU != "1" || g.Memory != "1Gi" || g.DeadlineS != 600 {
		t.Fatalf("%+v", g)
	}
	p.Target.Toolchain = "node24"
	p.Target.GraderMemory = "4Gi"
	if g := p.itemGrading(r); g.CPU != "2" || g.Memory != "4Gi" || g.DeadlineS != 900 {
		t.Fatalf("catalog defaults raised by the profile: %+v", g)
	}
	m := p.baselineMeta(r)
	if m["mode"] != "baseline" || m["attempt"] != 0 || m["base_revision"] != "s0" || m["hidden_mode"] != "overlay" {
		t.Fatal(m)
	}
	if mergeTask(r) != "002" {
		t.Fatal(mergeTask(r))
	}
	_, fr, _ := newP(t)
	if mergeTask(fr) != "A" {
		t.Fatal("the fixture's merge task is A")
	}
}

func TestReplayNeedsTheRunsClone(t *testing.T) {
	p, r := itemP(t)
	adir, _ := r.AttemptDir("002", 1)
	_ = os.WriteFile(filepath.Join(adir, "change.bundle"), []byte("x"), 0o644)
	if err := p.replay(r, "002", 1); err == nil || !strings.Contains(err.Error(), "target clone is gone") {
		t.Fatal(err)
	}
}

// 029: a dependency item's manifest change reaches the mode text, the worker and the baseline.
func TestDependencyItemInputs(t *testing.T) {
	p, r := itemP(t)
	if manifestOf(r) != nil || manifestText(r) != "" || p.workerItem(r).FixedPaths != nil {
		t.Fatal("an ordinary item has no manifest change")
	}
	if m := p.baselineMeta(r); m["dependency"] != nil || m["fixed_paths"] != nil {
		t.Fatalf("ordinary baseline meta: %v", m)
	}
	c := "3f9c2d0e8b7a6c5d4e3f2a1b0c9d8e7f6a5b4c3d"
	r.State["manifest_change"] = map[string]any{"commit": c, "files": []any{"requirements.txt"}, "subject": "chore(deps): bump requests",
		"packages": []any{map[string]any{"name": "requests", "from": "2.31.0", "to": "2.32.3", "level": "minor"}}}
	if got := manifestText(r); got != `dependency item: approved manifest change 3f9c2d0 ("chore(deps): bump requests") in requirements.txt, fixed for the worker; bumps requests 2.31.0 → 2.32.3 (minor)` {
		t.Fatal(got)
	}
	if !strings.Contains(p.modeExtra(r), "\n  dependency item: approved manifest change 3f9c2d0") || !strings.Contains(p.modeExtraGitHub(r), "; dependency item:") {
		t.Fatalf("%s\n%s", p.modeExtra(r), p.modeExtraGitHub(r))
	}
	if w := p.workerItem(r); strings.Join(w.FixedPaths, ",") != "requirements.txt" {
		t.Fatalf("%+v", w.FixedPaths)
	}
	m := p.baselineMeta(r)
	if m["dependency"] != true || fmt.Sprint(m["fixed_paths"]) != "[requirements.txt]" || m["mode"] != "baseline" {
		t.Fatalf("dependency baseline meta: %v", m)
	}
	if checkResult(map[string]any{"checks": []any{map[string]any{"id": "G4_visible_tests", "result": "fail"}}}, "G4_visible_tests") != "fail" {
		t.Fatal("checkResult")
	}
}
