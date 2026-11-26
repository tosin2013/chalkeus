//go:build integration

package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/tosin2013/chalkeus/internal/audit"
	"github.com/tosin2013/chalkeus/internal/evidence"
)

var issueRE = regexp.MustCompile(`/issues/(\d+)`)

// TestFixtureRun is the fixture scenario (T010): the walk-away run with GitHub approvals. The
// owner approves the mode and the retry on the tracking issue and merges A's PR and the status
// PR, all through the fake; the run's watcher does the rest. A/1 fails by induction, A/2 passes,
// B runs and opens its PR. The run then ends failed, only because AU12 marks it a test run.
func TestFixtureRun(t *testing.T) {
	out, code := w.pilot("run")
	id := runID(t, out)
	if code != 5 {
		t.Fatalf("run should pause for the mode approval (exit 5), got %d:\n%s", code, tail(out, 40))
	}
	m := issueRE.FindStringSubmatch(out)
	if m == nil {
		t.Fatalf("no tracking issue link in:\n%s", tail(out, 40))
	}
	var issue int
	_, _ = fmt.Sscan(m[1], &issue)

	seen, reason := w.drive(t, fixtureRepo, id, issue)
	// A test run never succeeds: its end-of-run audit fails AU12 "not a test run" (025 R1), so
	// the expected end is failed with reason AUDIT. The audit below checks AU12 is the only cause.
	if last := seen[len(seen)-1]; last != "failed" || reason != "AUDIT" {
		t.Fatalf("run %s ended %s (reason %s), want failed (reason AUDIT, from AU12 only); watch.log:\n%s",
			id, last, reason, readTail(filepath.Join(w.runDir(id), "github", "watch.log"), 60))
	}
	for _, want := range []string{"paused(awaiting-mode)", "paused(awaiting-retry)", "paused(awaiting-merge)"} {
		if !contains(seen, want) {
			t.Errorf("the run never paused at %s: %v", want, seen)
		}
	}

	dir := w.runDir(id)
	rows, err := evidence.ReadJSONL(filepath.Join(dir, "checks", "results.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, r := range rows {
		cid, _ := r["id"].(string)
		res, _ := r["result"].(string)
		if prev, ok := got[cid]; !ok || prev == "pass" {
			got[cid] = res // one failing row for an id is a failure
		}
	}
	for _, cid := range []string{"P01", "P02", "P03", "P04", "P05", "P06", "P07", "P08", "P09", "P10", "P13", "GM01", "GM02", "GM03", "GM04"} {
		if got[cid] != "pass" {
			t.Errorf("check %s: %q (want pass)", cid, got[cid])
		}
	}

	auditRows, err := audit.Dir(dir, []string{filepath.Join(w.config, "opencode", "stub_api_key")},
		[]string{filepath.Join(w.config, "publisher.env")}, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range auditRows {
		switch {
		case r.ID == "AU12":
			if r.Result != "fail" {
				t.Errorf("AU12 must fail for a test run: %+v", r)
			}
		case r.ID == "AU07": // the teardown check; the cluster is still up
		case r.Result != "pass":
			t.Errorf("audit %s: %s (%s)", r.ID, r.Result, r.Detail)
		}
	}

	// On the fake: A's PR and the status PR merged by the owner, B's PR open.
	var merged, open int
	for _, it := range w.fake.Issues(fixtureRepo) {
		if it.PR == nil {
			continue
		}
		switch {
		case it.PR.Merged:
			merged++
		case it.State == "open":
			open++
		}
	}
	if merged < 2 || open < 1 {
		t.Errorf("want A's PR and the status PR merged and B's PR open; got merged=%d open=%d", merged, open)
	}
}

// drive acts as the owner at each pause of a run, once, and lets the run's watcher resume it (or
// resumes it as the operator would when no watcher runs). It returns the states seen and the
// run's end reason. It fails the test when the run stops moving for 10 minutes.
func (x *world) drive(t *testing.T, repo, id string, issue int) ([]string, string) {
	t.Helper()
	var seen []string
	acted := map[string]bool{}
	failures := map[string]int{}
	changed := time.Now()
	deadline := time.Now().Add(30 * time.Minute)
	for time.Now().Before(deadline) {
		st := x.status(t, id)
		state := fmt.Sprint(st["state"])
		if len(seen) == 0 || seen[len(seen)-1] != state {
			seen = append(seen, state)
			say("run " + id + ": " + state)
			changed = time.Now()
		}
		if time.Since(changed) > 10*time.Minute { // the longest step (an attempt) takes a few minutes
			t.Fatalf("run %s has been %s for 10 minutes; watch.log:\n%s", id, state,
				readTail(filepath.Join(x.runDir(id), "github", "watch.log"), 40))
		}
		if state == "succeeded" || state == "failed" {
			t.Logf("states: %s", strings.Join(seen, " → "))
			return seen, fmt.Sprint(st["reason"])
		}
		key := state + fmt.Sprint(st["pending"])
		if strings.HasPrefix(state, "paused") && !acted[key] {
			acted[key] = true
			x.actAsOwner(t, repo, state, issue)
		}
		if strings.HasPrefix(state, "paused") && st["watcher"] == nil && acted[key] {
			if out, code := x.pilot("resume", "--run", id); code != 0 && code != 4 && code != 5 && code != 6 {
				failures[key]++
				t.Logf("resume exited %d:\n%s", code, tail(out, 30))
				if failures[key] >= 3 {
					t.Fatalf("run %s: resume failed 3 times at %s", id, state)
				}
			}
		}
		time.Sleep(10 * time.Second)
	}
	t.Fatalf("run %s did not end in 30 minutes: %v", id, seen)
	return nil, ""
}

// actAsOwner does what the owner would at a pause, through the fake.
func (x *world) actAsOwner(t *testing.T, repo, state string, issue int) {
	t.Helper()
	switch state {
	case "paused(awaiting-mode)":
		must(t, x.fake.Comment(repo, issue, ownerLogin, "/approve mode"))
	case "paused(awaiting-retry)":
		must(t, x.fake.Comment(repo, issue, ownerLogin, "/approve retry"))
	case "paused(awaiting-merge)":
		for _, it := range x.fake.Issues(repo) {
			if it.PR != nil && it.State == "open" {
				if _, err := x.fake.Merge(repo, it.Number, ownerLogin); err != nil {
					t.Fatalf("merge #%d: %v", it.Number, err)
				}
				say(fmt.Sprintf("owner merged #%d (%s)", it.Number, it.Title))
			}
		}
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
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

func readTail(p string, n int) string {
	b, err := os.ReadFile(p)
	if err != nil {
		return "(" + err.Error() + ")"
	}
	return tail(string(b), n)
}
