//go:build integration

package integration

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tosin2013/specfoundry/internal/evidence"
)

// TestStepMode is the step-mode scenario (T012) on the tally target, one `step --once` pass at a
// time, with the owner acting through the fake between passes:
//  1. the first pass mirrors the roadmap to issues; nothing is proposed yet;
//  2. the owner labels item 001's issue `pilot/go`; the next pass drafts its hidden tests and
//     opens the companion PR;
//  3. the owner merges it; the next pass runs doctor and starts the run, which pauses for the
//     mode approval on GitHub (no terminal: PILOT_NO_TTY);
//  4. the owner approves the mode and merges the item PR and the status PR as the run asks;
//  5. the run ends failed, only because AU12 marks it a test run, and the next pass reports it.
func TestStepMode(t *testing.T) {
	const repo = "owner/tally"
	step := func(what string) string {
		out, code := w.pilot("step", "--target", "tally", "--once")
		t.Logf("step pass, %s (exit %d):\n%s", what, code, tail(out, 25))
		return out
	}

	step("before the label")
	issue := 0
	for _, it := range w.fake.Issues(repo) {
		if it.PR == nil && strings.HasPrefix(it.Title, "001") {
			issue = it.Number
		}
	}
	if issue == 0 {
		t.Fatalf("no issue for item 001 after the first pass")
	}
	must(t, w.fake.Label(repo, issue, ownerLogin, "pilot/go", false))

	out := step("after the label")
	if !strings.Contains(out, "DRAFT_OK") {
		t.Fatalf("the pass after the label should draft item 001's hidden tests")
	}
	companion := 0
	for _, it := range w.fake.Issues("owner/tally-acceptance") {
		if it.PR != nil && it.State == "open" && strings.HasPrefix(it.PR.Head, "author/001-") {
			companion = it.Number
		}
	}
	if companion == 0 {
		t.Fatalf("no companion PR for 001")
	}
	if _, err := w.fake.Merge("owner/tally-acceptance", companion, ownerLogin); err != nil {
		t.Fatal(err)
	}

	step("after the tests merged")
	entries := decisions(t)
	runDir := ""
	for _, e := range entries {
		if e["action"] == "run" {
			runDir, _ = e["run"].(string)
		}
	}
	if runDir == "" {
		t.Fatalf("the pass after the merge should run doctor and start the run: %v", entries)
	}
	id := filepath.Base(runDir)
	if st := w.status(t, id); st["state"] != "paused(awaiting-mode)" {
		t.Errorf("the run should pause for the mode approval: state=%v", st["state"])
	}
	evs, err := evidence.ReadJSONL(filepath.Join(w.runDir(id), "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	by := ""
	for _, e := range evs {
		if e["event"] == "run_started_by" {
			by = fmt.Sprint(e["detail"])
		}
	}
	if by != "step" {
		t.Errorf("run_started_by is %q, want step", by)
	}

	seen, reason := w.drive(t, repo, id, issue)
	if last := seen[len(seen)-1]; last != "failed" || reason != "AUDIT" {
		t.Fatalf("run %s ended %s (reason %s), want failed (reason AUDIT, from AU12 only)", id, last, reason)
	}

	out = step("after the run")
	if !strings.Contains(out, "run-failed") && !strings.Contains(out, "relabel") {
		t.Errorf("the pass after the run should report it failed")
	}
	var actions []string
	for _, e := range decisions(t) {
		actions = append(actions, fmt.Sprint(e["action"]))
	}
	for _, want := range []string{"draft", "doctor-ok", "run"} {
		if !contains(actions, want) {
			t.Errorf("decisions.jsonl has no %q: %v", want, actions)
		}
	}
}

// decisions is step mode's log for the tally target.
func decisions(t *testing.T) []evidence.Row {
	t.Helper()
	rows, err := evidence.ReadJSONL(filepath.Join(w.root, "pilot", "runs", "step", "tally", "decisions.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	return rows
}
