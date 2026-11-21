//go:build integration

package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAutoDraft is the automatic-drafting scenario (T011) on the notes target. The first pass
// drafts item 001 (the stub's good draft): the author Job's isolation checks A01-A05 pass, the
// check says DRAFT_OK and a companion PR opens. The second pass skips 001 (its PR is open) and
// drafts 002, whose file path the item's own plan names: DRAFT_PATH_IN_SPEC and no PR.
func TestAutoDraft(t *testing.T) {
	out, code := w.pilot("author", "watch", "--target", "notes", "--once")
	t.Logf("first pass (exit %d):\n%s", code, tail(out, 30))
	if code != 0 || !strings.Contains(out, "draft 001: DRAFT_OK, opened") {
		t.Fatalf("item 001 should get a checked companion PR")
	}
	out, code = w.pilot("author", "watch", "--target", "notes", "--once")
	t.Logf("second pass (exit %d):\n%s", code, tail(out, 30))
	if !strings.Contains(out, "skip 001: a draft pull request is open") {
		t.Errorf("item 001 should be skipped while its PR is open")
	}
	if code != 1 || !strings.Contains(out, "draft 002: attempt 1 DRAFT_PATH_IN_SPEC") || !strings.Contains(out, "draft 002: failed, no PR") {
		t.Errorf("item 002's draft should be refused for its path, with no PR")
	}

	var prs []string
	for _, it := range w.fake.Issues("owner/notes-acceptance") {
		if it.PR != nil {
			prs = append(prs, it.PR.Head)
		}
	}
	if len(prs) != 1 || !strings.HasPrefix(prs[0], "author/001-") {
		t.Errorf("want exactly one companion PR, for 001; got %v", prs)
	}
	for _, it := range w.fake.Issues("owner/notes") {
		if it.PR != nil {
			t.Errorf("the watcher must never write the target; found PR #%d", it.Number)
		}
	}

	// The author Jobs' isolation checks, from each draft's draft.json (jobs[].checks).
	files, _ := filepath.Glob(filepath.Join(w.root, "pilot", "runs", "drafts", "*", "draft.json"))
	if len(files) == 0 {
		t.Fatal("no draft.json")
	}
	seen := map[string]bool{}
	for _, f := range files {
		var d struct {
			Jobs []struct {
				Checks []struct{ ID, Result, Detail string } `json:"checks"`
			} `json:"jobs"`
		}
		b, err := os.ReadFile(f)
		if err != nil || json.Unmarshal(b, &d) != nil {
			t.Fatalf("%s: %v", f, err)
		}
		for _, j := range d.Jobs {
			for _, c := range j.Checks {
				seen[c.ID] = true
				if c.Result != "pass" {
					t.Errorf("%s: %s %s (%s)", filepath.Base(filepath.Dir(f)), c.ID, c.Result, c.Detail)
				}
			}
		}
	}
	for _, id := range []string{"A01", "A02", "A03", "A04", "A05"} {
		if !seen[id] {
			t.Errorf("no %s row in the drafts' checks", id)
		}
	}
}
