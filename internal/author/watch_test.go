package author

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tosin2013/specfoundry/internal/config"
	"github.com/tosin2013/specfoundry/internal/readiness"
)

const statusJSON = `{"items":[
 {"id":"001","title":"done","status":"verified","active":true,"dependencies":[]},
 {"id":"005","title":"blocked","status":"specced","active":true,"dependencies":[{"id":"004","satisfied":false,"state":"unfinished"}]},
 {"id":"006","title":"bundled flags","status":"specced","active":true,"dependencies":[{"id":"001","satisfied":true,"state":"satisfied"}]},
 {"id":"007","title":"old","status":"specced","active":false,"dependencies":[]}]}`

func TestCandidates(t *testing.T) {
	c, err := Candidates(statusJSON)
	if err != nil || len(c) != 1 || c[0].ID != "006" {
		t.Fatalf("%+v %v", c, err)
	}
	if _, err := Candidates("not json"); err == nil {
		t.Fatal("an unreadable roadmap is an error")
	}
}

// watchWorld is the author test world plus a fake author Job and a scripted baseline.
func watchWorld(t *testing.T, baselines ...string) (*world, WatchEnv, *[]string) {
	w := newWorld(t, config.LevelMinimal)
	w.env.Target.AuthorAttempts = 2
	var prompts []string
	n := 0
	we := WatchEnv{Env: w.env, Status: func() (string, error) { return statusJSON, nil }}
	we.Job = func(ctx context.Context, d *Draft, attempt int, prompt string) JobRecord {
		prompts = append(prompts, prompt)
		write(t, filepath.Join(d.Workspace, "draft"), "tests/test_bundled.py", "def test_it():\n    assert False, '"+marker+"'\n")
		m, _ := json.Marshal(Map{Version: 1, SpecSHA256: "spec-sha", Criteria: map[string][]string{"US1-AS1": {"tests/test_bundled.py::test_it"}}, Uncovered: []string{"FR-001"}})
		write(t, filepath.Join(d.Workspace, "draft"), MapFile, string(m))
		return JobRecord{Attempt: attempt, Outcome: "Complete", DraftFiles: 2, Checks: []AuthorCheck{{ID: "A01", Result: "pass"}}}
	}
	we.Baseline = func(dir, id, item string, res readiness.Result, overlay []byte) (map[string]any, []byte, error) {
		b := baselines[n]
		n++
		checks := []any{map[string]any{"id": "G4_visible_tests", "result": "pass"}, map[string]any{"id": "G5_hidden_tests", "result": "pass"}}
		if b == "hidden_passes" {
			checks[1] = map[string]any{"id": "G5_hidden_tests", "result": "fail"}
		}
		r := map[string]any{"result": "pass", "checks": checks}
		if b != "pass" {
			r["result"] = "fail"
		}
		return r, []byte("Tests run: 1, Failures: 0 " + marker), nil
	}
	return w, we, &prompts
}

func TestPassOpensAPROnlyAfterAPassingCheck(t *testing.T) {
	w, we, prompts := watchWorld(t, "pass")
	out, err := Pass(context.Background(), we)
	if err != nil || out != PassOpened {
		t.Fatalf("%s %v\n%s", out, err, w.out)
	}
	if !strings.Contains(w.out.String(), "draft 006: DRAFT_OK, opened https://x/pr/9") {
		t.Fatalf("%s", w.out)
	}
	if len(*prompts) != 1 || strings.Contains((*prompts)[0], "failed its check") {
		t.Fatal("one Job, no feedback")
	}
	d, _ := LoadDraft(Root(w.env.EvidenceRoot), "", "proj", "006")
	if d.Trigger != "auto" || d.State != "submitted" || len(d.Jobs) != 1 {
		t.Fatalf("%+v", d)
	}
	// The PR's draft is never drafted again: the companion now has an open PR.
	w.fake.openHeads = []string{"author/006-x"}
	if out, _ := Pass(context.Background(), we); out != PassIdle || !strings.Contains(w.out.String(), "skip 006: a draft pull request is open") {
		t.Fatalf("%s\n%s", out, w.out)
	}
}

func TestPassRetriesWithFeedbackThenGivesUp(t *testing.T) {
	w, we, prompts := watchWorld(t, "hidden_passes", "pass")
	if out, err := Pass(context.Background(), we); err != nil || out != PassOpened {
		t.Fatalf("%s %v\n%s", out, err, w.out)
	}
	if len(*prompts) != 2 || !strings.Contains((*prompts)[1], "failed its check: DRAFT_HIDDEN_PASSES") || !strings.Contains((*prompts)[1], marker) {
		t.Fatal("the second Job gets the failure and its own tests' output")
	}

	w2, we2, _ := watchWorld(t, "hidden_passes", "hidden_passes")
	if out, err := Pass(context.Background(), we2); err != nil || out != PassFailed {
		t.Fatalf("%s %v\n%s", out, err, w2.out)
	}
	if len(w2.fake.prBody) != 0 {
		t.Fatal("no PR without a passing check (SC-003)")
	}
	// Same spec version: never drafted again automatically (FR-007).
	if out, _ := Pass(context.Background(), we2); out != PassIdle || !strings.Contains(w2.out.String(), "already drafted automatically") {
		t.Fatalf("%s\n%s", out, w2.out)
	}
	// A spec change makes it eligible again.
	w2.res.SpecSHA256 = "spec-sha-2"
	we2.Ready = func(config.Target, string) (readiness.Result, error) { return w2.res, nil }
	if why, _, _ := we2.skip(context.Background(), "006"); why != "" {
		t.Fatalf("a new spec version is eligible: %s", why)
	}
}

func TestPassSkipsAndRefusals(t *testing.T) {
	w, we, prompts := watchWorld(t)
	w.res.HiddenFiles = 3
	if out, _ := Pass(context.Background(), we); out != PassIdle || !strings.Contains(w.out.String(), "hidden tests already merged") {
		t.Fatalf("%s\n%s", out, w.out)
	}
	w.res.HiddenFiles, w.res.Decision, w.res.Reasons = 0, "refuse", []string{"ITEM_OPEN_QUESTIONS"}
	if out, _ := Pass(context.Background(), we); out != PassIdle || !strings.Contains(w.out.String(), "not ready (ITEM_OPEN_QUESTIONS)") {
		t.Fatalf("%s\n%s", out, w.out)
	}
	// A failed isolation check stops the draft before any check or PR.
	w.res.Decision, w.res.Reasons = "allow", nil
	job := we.Job
	we.Job = func(ctx context.Context, d *Draft, a int, p string) JobRecord {
		r := job(ctx, d, a, p)
		r.Checks = []AuthorCheck{{ID: "A03", Result: "fail", Detail: "github_via_proxy=http=200"}}
		return r
	}
	if out, _ := Pass(context.Background(), we); out != PassFailed || !strings.Contains(w.out.String(), "isolation check A03 failed") || len(*prompts) != 1 {
		t.Fatalf("%s\n%s", out, w.out)
	}
}

func TestPassStopsWhenCancelled(t *testing.T) {
	w, we, _ := watchWorld(t, "pass")
	ctx, cancel := context.WithCancel(context.Background())
	we.Job = func(c context.Context, d *Draft, a int, p string) JobRecord {
		cancel()
		return JobRecord{Attempt: a, Outcome: "cancelled", Error: "cancelled"}
	}
	if out, err := Pass(ctx, we); err != nil || out != PassStopped || len(w.fake.prBody) != 0 {
		t.Fatalf("%s %v\n%s", out, err, w.out)
	}
	d, _ := LoadDraft(Root(w.env.EvidenceRoot), "", "proj", "006")
	if !strings.HasPrefix(d.WatchDecision, "stopped") {
		t.Fatalf("%+v", d)
	}
}
