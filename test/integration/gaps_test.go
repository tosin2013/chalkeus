//go:build integration

package integration

import (
	"strings"
	"testing"
)

// TestStepGaps covers the step-mode debrief's gaps (T013) on the gaps target, whose hidden tests
// are already merged and whose own tests fail:
//  1. a label added by someone who isn't an approver starts nothing (no doctor, no run, no
//     comment), and the pass says an approver must add it;
//  2. once an approver adds it, doctor --baseline fails (the project's tests fail): exactly one
//     comment on the item's issue, and no run;
//  3. another pass at the same commits neither comments again nor runs.
func TestStepGaps(t *testing.T) {
	const repo = "owner/gaps"
	step := func(what string) string {
		out, code := w.pilotTTY("step", "--target", "gaps", "--once")
		t.Logf("step pass, %s (exit %d):\n%s", what, code, tail(out, 60))
		return out
	}
	step("before any label") // mirrors the roadmap to issues
	issue := 0
	for _, it := range w.fake.Issues(repo) {
		if it.PR == nil && strings.HasPrefix(it.Title, "001") {
			issue = it.Number
		}
	}
	if issue == 0 {
		t.Fatalf("no issue for item 001")
	}

	must(t, w.fake.Label(repo, issue, "stranger", "chalkeus:proposed", false))
	out := step("after a non-approver's label")
	if !strings.Contains(out, "not in FORGE_APPROVERS") || !strings.Contains(out, "next: 001: waiting for an approver") {
		t.Errorf("the pass should say the label's adder is not an approver, and that an approver must relabel")
	}
	if acts := actionsOf(t, "gaps"); len(acts) != 0 {
		t.Errorf("a non-approver's label must start nothing; decisions: %v", acts)
	}

	must(t, w.fake.Label(repo, issue, "stranger", "chalkeus:proposed", true))
	must(t, w.fake.Label(repo, issue, ownerLogin, "chalkeus:proposed", false))
	out = step("after the owner's label")
	if !strings.Contains(out, "next: 001: waiting for you: doctor found problems") {
		t.Errorf("after the failed doctor, the next line should point at its findings")
	}
	if strings.Contains(out, "already pass") || !strings.Contains(out, "not checked: the project's own tests fail first") {
		t.Errorf("doctor must not claim the hidden tests pass when they were never graded")
	}
	out = step("again, at the same commits")
	if !strings.Contains(out, "next: 001: waiting for you: doctor found problems") && strings.Contains(out, "nothing is waiting for you") {
		t.Errorf("a standing doctor failure is still the owner's move")
	}

	acts := actionsOf(t, "gaps")
	if count(acts, "doctor-failed") != 1 || count(acts, "comment") != 1 || count(acts, "run") != 0 {
		t.Errorf("want one failed doctor, one comment and no run; decisions: %v", acts)
	}
	var comments []string
	for _, it := range w.fake.Issues(repo) {
		if it.Number == issue {
			for _, c := range it.Comments {
				if strings.Contains(c.Body, "pilot-step:doctor-failed") {
					comments = append(comments, c.Body)
				}
			}
		}
	}
	if len(comments) != 1 {
		t.Fatalf("want exactly one doctor-failed comment on #%d, got %d", issue, len(comments))
	}
	if !strings.Contains(comments[0], "FAIL") {
		t.Errorf("the comment should list doctor's findings:\n%s", comments[0])
	}
	for _, it := range w.fake.Issues(repo) {
		if it.PR != nil {
			t.Errorf("no PR may open on the target; found #%d", it.Number)
		}
	}
}

// actionsOf lists the actions in a target's step decisions.jsonl ("" when there is none yet).
func actionsOf(t *testing.T, target string) []string {
	t.Helper()
	rows, _ := readRows(filepathJoin(w.root, "pilot", "runs", "step", target, "decisions.jsonl"))
	var out []string
	for _, r := range rows {
		if a, _ := r["action"].(string); a != "" {
			out = append(out, a)
		}
	}
	return out
}

func count(xs []string, s string) int {
	n := 0
	for _, x := range xs {
		if x == s {
			n++
		}
	}
	return n
}
