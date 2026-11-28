package author

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tosin2013/chalkeus/internal/config"
	"github.com/tosin2013/chalkeus/internal/fixture"
	"github.com/tosin2013/chalkeus/internal/readiness"
)

func caseJSON(c EvalCase) string { b, _ := json.Marshal(c); return string(b) }

// evalWorld: the target from newWorld, a companion with eval cases, and a fake author Job that
// writes one test file.
func evalWorld(t *testing.T, refReportFor func(n int) map[string]any) (*world, EvalEnv, *int) {
	w := newWorld(t, config.LevelMinimal)
	ready := EvalCase{ID: "proj-006", Target: "proj", BaseRevision: w.res.Commit, Item: "006", SpecDir: "specs/006-x",
		Title: "bundled flags", AllowedPaths: []string{"app.py"}, KnownFailure: "a test that doesn't compile", Status: "ready"}
	pending := EvalCase{ID: "proj-007", Target: "proj", Item: "007", Status: "pending"}
	gitRepo(t, fixture.BaseURL, "me/acc.git", map[string]string{
		"evals/proj-006/case.json": caseJSON(ready), "evals/proj-006/reference.patch": "diff\n" + refMarker + "\n",
		"evals/proj-007/case.json": caseJSON(pending)})
	jobs := 0
	job := func(ctx context.Context, d *Draft, attempt int, prompt string) JobRecord {
		jobs++
		write(t, filepath.Join(d.Workspace, "draft"), "tests/test_bundled.py", "def test_it():\n    assert False, '"+marker+"'\n")
		m, _ := json.Marshal(Map{Version: 1, SpecSHA256: d.SpecSHA256, Criteria: map[string][]string{"US1-AS1": {"tests/test_bundled.py::test_it"}}, Uncovered: []string{"FR-001"}})
		write(t, filepath.Join(d.Workspace, "draft"), MapFile, string(m))
		return JobRecord{Attempt: attempt, Outcome: "Complete", DraftFiles: 1}
	}
	baseline := func(dir, id, item string, res readiness.Result, overlay []byte) (map[string]any, []byte, error) {
		if res.Commit != w.res.Commit {
			t.Fatalf("eval drafts against the case's base: %s", res.Commit)
		}
		return map[string]any{"result": "pass", "checks": []any{}}, nil, nil
	}
	n := 0
	w.env.Reference = func(dir, id, item string, res readiness.Result, overlay, ref []byte) (map[string]any, []byte, error) {
		n++
		if !bytes.Contains(ref, []byte(refMarker)) {
			t.Fatal("eval stages the case's reference")
		}
		return refReportFor(n), []byte("out " + marker), nil
	}
	w.env.Target.AuthorAttempts = 2
	ee := EvalEnv{StateDir: t.TempDir(), Companion: "me/acc", Host: w.env.Target.Host(), Out: w.out,
		Setup: func(target string) (Env, func(context.Context, *Draft, int, string) JobRecord, Baseline, string, error) {
			return w.env, job, baseline, "opencode-go/test-model", nil
		}}
	return w, ee, &jobs
}

func TestEvalScoresAndCompares(t *testing.T) {
	w, ee, jobs := evalWorld(t, func(int) map[string]any { return refReport("", "proven") })
	r, err := Eval(context.Background(), ee, "")
	if err != nil {
		t.Fatal(err)
	}
	if r.Score != 1 || r.Of != 1 || len(r.Cases) != 2 || r.Cases[0].Outcome != "proven" || r.Cases[1].Outcome != "pending" ||
		r.Model != "opencode-go/test-model" || r.PromptSHA256 == "" || r.Previous != nil || *jobs != 1 {
		t.Fatalf("%+v jobs=%d", r, *jobs)
	}
	if !strings.Contains(w.out.String(), "score: 1 of 1 proven") {
		t.Fatal(w.out.String())
	}
	// The second run fails the reference check twice: unproven, and the score dropped.
	w2, ee2, _ := evalWorld(t, func(int) map[string]any { return refReport("DRAFT_FAILS_ON_REFERENCE", "fails_on_reference") })
	ee2.StateDir = ee.StateDir
	r2, err := Eval(context.Background(), ee2, "proj-006")
	if err != nil {
		t.Fatal(err)
	}
	if r2.Score != 0 || !r2.Dropped || r2.Previous == nil || *r2.Previous != 1 || r2.Cases[0].Reason != CodeDraftFailsOnReference ||
		len(r2.Changes) != 1 || r2.Changes[0] != "proj-006: proven -> unproven" {
		t.Fatalf("%+v", r2)
	}
	if !strings.Contains(w2.out.String(), "SCORE DROPPED") {
		t.Fatal(w2.out.String())
	}
	// Nothing opened a pull request, and no test or reference contents reached the result.
	if len(w.fake.tree) != 0 || len(w2.fake.tree) != 0 {
		t.Fatal("eval never opens a pull request")
	}
	entries, _ := os.ReadDir(ee.StateDir)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		raw, _ := os.ReadFile(filepath.Join(ee.StateDir, e.Name()))
		if bytes.Contains(raw, []byte(marker)) || bytes.Contains(raw, []byte(refMarker)) {
			t.Fatalf("%s carries contents", e.Name())
		}
	}
}

func TestEvalRedraftsWithFeedback(t *testing.T) {
	_, ee, jobs := evalWorld(t, func(n int) map[string]any {
		if n == 1 {
			return refReport("DRAFT_FAILS_ON_REFERENCE", "fails_on_reference")
		}
		return refReport("", "proven")
	})
	r, err := Eval(context.Background(), ee, "proj-006")
	if err != nil || r.Score != 1 || *jobs != 2 {
		t.Fatalf("%v %+v jobs=%d", err, r, *jobs)
	}
}

func TestEvalBusyNamespaceIsAnError(t *testing.T) {
	_, ee, jobs := evalWorld(t, func(int) map[string]any { return refReport("", "proven") })
	ee.Busy = func(ns string) bool { return ns == "pilot-a-proj-006" }
	r, _ := Eval(context.Background(), ee, "proj-006")
	if r.Cases[0].Outcome != "error" || !strings.Contains(r.Cases[0].Reason, "author watch") || *jobs != 0 {
		t.Fatalf("%+v", r.Cases[0])
	}
}

func TestLoadEvalCasesValidates(t *testing.T) {
	d := t.TempDir()
	if _, err := LoadEvalCases(d, ""); err == nil || !strings.Contains(err.Error(), "no evals/") {
		t.Fatal(err)
	}
	write(t, d, "evals/a/case.json", `{"status":"ready","target":"t"}`)
	if _, err := LoadEvalCases(d, ""); err == nil || !strings.Contains(err.Error(), "needs target, base_revision, item and spec_dir") {
		t.Fatal(err)
	}
	write(t, d, "evals/a/case.json", `{"target":"t","base_revision":"abc","item":"004","spec_dir":"specs/004-x"}`)
	if _, err := LoadEvalCases(d, ""); err == nil || !strings.Contains(err.Error(), "needs reference.patch") {
		t.Fatal(err)
	}
	write(t, d, "evals/a/reference.patch", "diff\n")
	write(t, d, "evals/b/case.json", `{"status":"maybe"}`)
	if _, err := LoadEvalCases(d, "b"); err == nil || !strings.Contains(err.Error(), "status") {
		t.Fatal(err)
	}
	cs, err := LoadEvalCases(d, "a")
	if err != nil || len(cs) != 1 || cs[0].ID != "a" || cs[0].Status != "ready" {
		t.Fatalf("%v %+v", err, cs)
	}
	if _, err := LoadEvalCases(d, "zzz"); err == nil {
		t.Fatal("an unknown case is an error")
	}
}
