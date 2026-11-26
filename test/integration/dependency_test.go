//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tosin2013/chalkeus/internal/audit"
	"github.com/tosin2013/chalkeus/internal/evidence"
	"github.com/tosin2013/chalkeus/internal/ghloop"
)

// TestDependencyItem is 029's dependency item (US2, SC-003, SC-004) on the deps target:
//  1. a bot branch bumps greetlib 1.0 → 2.0 in requirements.txt, and item 001's spec names that
//     commit under `## Manifest change` (the owner's preparation, done here on the fake);
//  2. the baseline passes with the visible test failing (the bump broke it: that is the work), and
//     the hidden tests are not measured;
//  3. attempt 1 pins greetlib back to 1.0 instead of fixing the code: it fails G2 with
//     MANIFEST_CHANGED, and the owner approves the retry;
//  4. attempt 2 fixes the code and passes, offline;
//  5. the published branch holds the manifest commit first (its requirements.txt is the approved
//     commit's blob), then the fix, which leaves requirements.txt alone;
//  6. the audit passes (AU12 fails only because this is a test run; AU07 waits for teardown).
func TestDependencyItem(t *testing.T) {
	const repo = "owner/deps"
	bare := filepath.Join(w.fake.Dir, repo+".git")
	bump, specCommit := prepareDependencyItem(t, bare)

	out, code := w.pilot("run", "--target", "deps", "--item", "001")
	if code != 5 {
		t.Fatalf("run should pause for the mode (exit 5), got %d:\n%s", code, tail(out, 40))
	}
	id := runID(t, out)
	run, err := evidence.LoadRun(filepath.Join(w.root, "pilot", "runs"), id)
	if err != nil || run == nil {
		t.Fatalf("load run %s: %v", id, err)
	}
	seen, reason := w.drive(t, repo, id, ghloop.TrackingIssue(run))
	if last := seen[len(seen)-1]; last != "failed" || reason != "AUDIT" {
		t.Fatalf("run %s ended %s (reason %s), want failed (reason AUDIT, from AU12 only)", id, last, reason)
	}
	if !contains(seen, "paused(awaiting-retry)") {
		t.Errorf("the run never paused for the retry after the manifest edit: %v", seen)
	}
	dir := w.runDir(id)

	// Readiness recorded the change.
	var mc map[string]any
	readJSON(t, filepath.Join(dir, "manifest-change.json"), &mc)
	if mc["commit"] != bump || mc["replayed_commit"] == nil {
		t.Errorf("manifest-change.json: commit %v (want %s), replayed_commit %v", mc["commit"], bump, mc["replayed_commit"])
	}

	// The baseline: visible tests fail with the approved change; hidden tests not measured.
	base := grade(t, filepath.Join(dir, "baseline", "grade.json"))
	if base.Result != "pass" || base.check("G4_visible_tests").Result != "fail" || base.check("G5_hidden_tests").Result != "skipped" ||
		!strings.HasPrefix(base.check("G4_visible_tests").Detail, "with the approved manifest change: ") {
		t.Errorf("baseline: %s, G4 %+v, G5 %+v", base.Result, base.check("G4_visible_tests"), base.check("G5_hidden_tests"))
	}

	// Attempt 1 edited the manifest: G2 MANIFEST_CHANGED. Attempt 2 passed.
	a1 := grade(t, attemptGrade(t, dir, 1))
	if g2 := a1.check("G2_allowed_paths"); a1.Result != "fail" || g2.Result != "fail" || !strings.HasPrefix(g2.Detail, "MANIFEST_CHANGED: requirements.txt") {
		t.Errorf("attempt 1: %s, G2 %+v", a1.Result, g2)
	}
	if a2 := grade(t, attemptGrade(t, dir, 2)); a2.Result != "pass" {
		t.Errorf("attempt 2: %s %+v", a2.Result, a2.Checks)
	}

	// The published change, as merged (the fake merges with --no-ff, so the PR's head is the merge
	// commit's second parent): the manifest commit, then the fix.
	head := ""
	for _, it := range w.fake.Issues(repo) {
		if it.PR != nil && it.PR.Merged && strings.HasSuffix(it.PR.Head, "/001") {
			head = it.PR.MergeSHA + "^2"
		}
	}
	if head == "" {
		t.Fatalf("no merged PR for item 001 on %s", repo)
	}
	revs := strings.Fields(gitOut(t, bare, "rev-list", "--reverse", specCommit+".."+head))
	if len(revs) < 2 {
		t.Fatalf("the merged change should hold the manifest commit and the fix, got %v", revs)
	}
	if got, want := gitOut(t, bare, "rev-parse", revs[0]+":requirements.txt"), gitOut(t, bare, "rev-parse", bump+":requirements.txt"); got != want {
		t.Errorf("the first commit's requirements.txt is %s, want the approved commit's %s", got, want)
	}
	if msg := gitOut(t, bare, "log", "-1", "--format=%B", revs[0]); !strings.Contains(msg, bump) || !strings.Contains(msg, "Chalkeus-Run: "+id) {
		t.Errorf("the manifest commit's message should name %s and the run:\n%s", bump, msg)
	}
	if mc["replayed_commit"] != nil && mc["replayed_commit"] != revs[0] {
		t.Errorf("replayed_commit %v is not the branch's first commit %s", mc["replayed_commit"], revs[0])
	}
	if changed := gitOut(t, bare, "diff", "--name-only", revs[0], revs[len(revs)-1]); strings.Contains(changed, "requirements.txt") {
		t.Errorf("the fix changed requirements.txt: %s", changed)
	}

	rows, err := audit.Dir(dir, []string{filepath.Join(w.config, "opencode", "stub_api_key")},
		[]string{filepath.Join(w.config, "targets", "deps.publisher.env")}, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		switch {
		case r.ID == "AU12":
			if r.Result != "fail" {
				t.Errorf("AU12 must fail for a test run: %+v", r)
			}
		case r.ID == "AU07":
		case r.ID == "AU03" && r.Detail == "repo="+repo+" task_prs=1 status_prs=1":
			// The fake's PR links are http://<gateway>:443/<repo>/pull/<n>, not github.com, so item
			// AU03 can't pass here; checkPRs checks the rest of it.
			checkPRs(t, dir, repo)
		case r.Result != "pass":
			t.Errorf("audit %s: %s (%s)", r.ID, r.Result, r.Detail)
		}
	}
}

// checkPRs is item AU03 without its github.com link check: one task PR and one status PR, both
// drafts by the publisher identity, on the target.
func checkPRs(t *testing.T, dir, repo string) {
	t.Helper()
	var prs []map[string]any
	readJSON(t, filepath.Join(dir, "prs.json"), &prs)
	kinds := map[string]int{}
	for _, p := range prs {
		kinds[fmt.Sprint(p["kind"])]++
		if p["draft"] != true || p["author"] != botLogin || p["publisher_identity"] != botLogin ||
			!strings.Contains(fmt.Sprint(p["url"]), "/"+repo+"/pull/") {
			t.Errorf("PR record %v: want a draft by %s on %s", p, botLogin, repo)
		}
	}
	if kinds["task"] != 1 || kinds["status"] != 1 || len(prs) != 2 {
		t.Errorf("want one task PR and one status PR, got %v", kinds)
	}
}

// prepareDependencyItem does the owner's preparation on the fake: a bot branch with the approved
// bump on top of main, then a commit on main naming it in item 001's spec. It returns the bump's
// id and the new main.
func prepareDependencyItem(t *testing.T, bare string) (string, string) {
	t.Helper()
	work := t.TempDir()
	gitOut(t, "", "clone", "-q", bare, work)
	gitOut(t, work, "checkout", "-q", "-b", "dependabot/pip/greetlib-2.0")
	must(t, os.WriteFile(filepath.Join(work, "requirements.txt"), []byte("greetlib==2.0\n"), 0o644))
	gitOut(t, work, "-c", "user.name=dependabot[bot]", "-c", "user.email=bot@example.invalid",
		"commit", "-qam", "chore(deps): bump greetlib from 1.0 to 2.0")
	bump := gitOut(t, work, "rev-parse", "HEAD")
	gitOut(t, work, "push", "-q", "origin", "dependabot/pip/greetlib-2.0")
	gitOut(t, work, "checkout", "-q", "main")
	spec := filepath.Join(work, "specs", "001-greetlib-2", "spec.md")
	b, err := os.ReadFile(spec)
	must(t, err)
	must(t, os.WriteFile(spec, []byte(strings.Replace(string(b), "BUMP_COMMIT", bump, 1)), 0o644))
	gitOut(t, work, "-c", "user.name="+ownerLogin, "-c", "user.email=owner@example.invalid",
		"commit", "-qam", "docs(001): name the approved greetlib bump")
	gitOut(t, work, "push", "-q", "origin", "main")
	return bump, gitOut(t, work, "rev-parse", "HEAD")
}

type gradeFile struct {
	Result string `json:"result"`
	Checks []struct {
		ID     string `json:"id"`
		Result string `json:"result"`
		Detail string `json:"detail"`
	} `json:"checks"`
}

type gradeCheck struct{ Result, Detail string }

func (g gradeFile) check(id string) gradeCheck {
	for _, c := range g.Checks {
		if c.ID == id {
			return gradeCheck{c.Result, c.Detail}
		}
	}
	return gradeCheck{}
}

func grade(t *testing.T, p string) gradeFile {
	t.Helper()
	var g gradeFile
	readJSON(t, p, &g)
	return g
}

func attemptGrade(t *testing.T, dir string, n int) string {
	t.Helper()
	m, _ := filepath.Glob(filepath.Join(dir, "attempts", fmt.Sprintf("001-%d", n), "grade.json"))
	if len(m) != 1 {
		all, _ := filepath.Glob(filepath.Join(dir, "attempts", "*"))
		t.Fatalf("no grade for attempt %d; attempts: %v", n, all)
	}
	return m[0]
}

func readJSON(t *testing.T, p string, v any) {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatalf("%s: %v", p, err)
	}
}

func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	c := exec.Command("git", args...)
	c.Dir = dir
	b, err := c.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, b)
	}
	return strings.TrimSpace(string(b))
}
