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

	"github.com/tosin2013/specfoundry/internal/audit"
	"github.com/tosin2013/specfoundry/internal/evidence"
)

var issueRE = regexp.MustCompile(`/issues/(\d+)`)

// TestFixtureRun is the fixture scenario (T010): the walk-away run with GitHub approvals. The
// owner approves the mode and the retry on the tracking issue and merges A's PR and the status
// PR, all through the fake; the run's watcher does the rest. A/1 fails by induction, A/2 passes,
// B runs and opens its PR.
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

	// Act as the owner at each pause, once, and let the watcher resume.
	var seen []string
	acted := map[string]bool{}
	deadline := time.Now().Add(30 * time.Minute)
	for time.Now().Before(deadline) {
		st := w.status(t, id)
		state := fmt.Sprint(st["state"])
		if len(seen) == 0 || seen[len(seen)-1] != state {
			seen = append(seen, state)
			say("run " + id + ": " + state)
		}
		if state == "succeeded" || state == "failed" {
			break
		}
		key := state + fmt.Sprint(st["pending"])
		if strings.HasPrefix(state, "paused") && !acted[key] {
			acted[key] = true
			w.actAsOwner(t, state, issue)
		}
		if strings.HasPrefix(state, "paused") && st["watcher"] == nil && acted[key] {
			// The watcher stopped (or never started): resume as the operator would.
			if out, code := w.pilot("resume", "--run", id); code != 0 && code != 4 && code != 5 && code != 6 {
				t.Logf("resume exited %d:\n%s", code, tail(out, 30))
			}
		}
		time.Sleep(10 * time.Second)
	}
	t.Logf("states: %s", strings.Join(seen, " → "))
	if last := seen[len(seen)-1]; last != "succeeded" {
		t.Fatalf("run %s ended %s; watch.log:\n%s", id, last, readTail(filepath.Join(w.runDir(id), "github", "watch.log"), 60))
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
	for _, cid := range []string{"P01", "P02", "P03", "P04", "P05", "P06", "P07", "P08", "P09", "P10", "GM01", "GM02", "GM03", "GM04"} {
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

// actAsOwner does what the owner would at a pause, through the fake.
func (x *world) actAsOwner(t *testing.T, state string, issue int) {
	t.Helper()
	switch state {
	case "paused(awaiting-mode)":
		must(t, x.fake.Comment(fixtureRepo, issue, ownerLogin, "/approve mode"))
	case "paused(awaiting-retry)":
		must(t, x.fake.Comment(fixtureRepo, issue, ownerLogin, "/approve retry"))
	case "paused(awaiting-merge)":
		for _, it := range x.fake.Issues(fixtureRepo) {
			if it.PR != nil && it.State == "open" {
				if _, err := x.fake.Merge(fixtureRepo, it.Number, ownerLogin); err != nil {
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
