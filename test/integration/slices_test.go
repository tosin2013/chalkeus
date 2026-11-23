//go:build integration

package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tosin2013/chalkeus/internal/evidence"
	"github.com/tosin2013/chalkeus/internal/ghloop"
)

// TestSliceRuns is 026's two-slice scenario (quickstart §3) on the slices target, whose item 001
// has two user stories, each with its own allowed paths and hidden tests (items/001/US1, US2):
//  1. `./chalkeus forge slices` lists US1 and US2 untouched, current US1;
//  2. `run --slice US2` is refused before anything starts (SLICE_EARLIER_OPEN);
//  3. `run --slice US1` builds only US1: its PR is titled "(US1)"; after the owner merges it the run
//     records slice_merged and opens no status PR;
//  4. `run --slice US2` grades with US1's hidden tests too (G8, a summary only); after its merge,
//     exactly one status PR is opened and merged;
//  5. `./chalkeus forge audit --item 001` lists both slice runs.
//
// Every run ends failed only because AU12 marks it a test run, as in the other scenarios.
func TestSliceRuns(t *testing.T) {
	const repo = "owner/slices"
	out, code := w.pilot("slices", "--target", "slices", "--item", "001")
	t.Logf("slices (exit %d):\n%s", code, out)
	if code != 0 || !strings.Contains(out, "US1  untouched") || !strings.Contains(out, "US2  untouched") || !strings.Contains(out, "current: US1") {
		t.Fatalf("pilot slices should list US1 and US2 untouched, current US1")
	}

	out, code = w.pilot("run", "--target", "slices", "--item", "001", "--slice", "US2")
	if code != 10 || !strings.Contains(out, "SLICE_EARLIER_OPEN") {
		t.Fatalf("US2 before US1 must be refused with SLICE_EARLIER_OPEN (exit 10), got exit %d:\n%s", code, tail(out, 30))
	}

	us1 := w.sliceRun(t, repo, "US1")
	if !hasEvent(t, us1, "slice_merged") || hasEvent(t, us1, "task_verified") {
		t.Errorf("US1's run should record slice_merged and no verification")
	}
	if n := statusPRs(w, repo); n != 0 {
		t.Errorf("no status PR may open before the last slice; found %d", n)
	}
	title := ""
	for _, it := range w.fake.Issues(repo) {
		if it.PR != nil && strings.HasSuffix(it.PR.Head, "/001-us1") {
			title = it.Title
		}
	}
	if !strings.HasSuffix(title, "(US1)") {
		t.Errorf("US1's PR title %q should end with (US1)", title)
	}

	out, _ = w.pilot("slices", "--target", "slices", "--item", "001")
	if !strings.Contains(out, "US1  merged") || !strings.Contains(out, "current: US2") {
		t.Errorf("after US1 merged, US2 should be current:\n%s", out)
	}

	us2 := w.sliceRun(t, repo, "US2")
	if n := statusPRs(w, repo); n != 1 {
		t.Errorf("exactly one status PR after the last slice; found %d", n)
	}
	g8 := ""
	matches, _ := filepath.Glob(filepath.Join(w.runDir(us2), "attempts", "001-*", "grade.json"))
	for _, m := range matches {
		raw, _ := os.ReadFile(m)
		var g struct {
			Result string `json:"result"`
			Checks []struct{ ID, Result, Detail string }
		}
		if json.Unmarshal(raw, &g) == nil && g.Result == "pass" {
			for _, c := range g.Checks {
				if c.ID == "G8_earlier_slices_kept" {
					g8 = c.Result + ": " + c.Detail
				}
			}
		}
	}
	if !strings.HasPrefix(g8, "pass") || !strings.Contains(g8, "output withheld") {
		t.Errorf("US2's passing grade should have run US1's hidden tests (G8, summary only): %q", g8)
	}

	out, _ = w.pilot("audit", "--target", "slices", "--item", "001")
	t.Logf("item audit:\n%s", out)
	if !strings.Contains(out, us1+"  US1") || !strings.Contains(out, us2+"  US2") || !strings.Contains(out, "AU13") {
		t.Errorf("the item audit should list both slice runs and AU13")
	}
}

// sliceRun starts a slice run, drives it as the owner through the fake (mode approval, merges),
// and returns its id.
func (x *world) sliceRun(t *testing.T, repo, slice string) string {
	t.Helper()
	out, code := x.pilot("run", "--target", "slices", "--item", "001", "--slice", slice)
	if code != 5 {
		t.Fatalf("run --slice %s should pause for the mode (exit 5), got %d:\n%s", slice, code, tail(out, 40))
	}
	id := runID(t, out)
	run, err := evidence.LoadRun(filepath.Join(x.root, "pilot", "runs"), id)
	if err != nil || run == nil {
		t.Fatalf("load run %s: %v", id, err)
	}
	if run.Slice() != slice {
		t.Errorf("run %s records slice %q, want %s", id, run.Slice(), slice)
	}
	seen, reason := x.drive(t, repo, id, ghloop.TrackingIssue(run))
	if last := seen[len(seen)-1]; last != "failed" || reason != "AUDIT" {
		t.Fatalf("slice %s run %s ended %s (reason %s), want failed (reason AUDIT, from AU12 only)", slice, id, last, reason)
	}
	return id
}

func hasEvent(t *testing.T, id, name string) bool {
	t.Helper()
	rows, err := evidence.ReadJSONL(filepath.Join(w.runDir(id), "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range rows {
		if e["event"] == name {
			return true
		}
	}
	return false
}

func statusPRs(x *world, repo string) int {
	n := 0
	for _, it := range x.fake.Issues(repo) {
		if it.PR != nil && strings.Contains(it.PR.Head, "/status-001") {
			n++
		}
	}
	return n
}
