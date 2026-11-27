package ghloop

import (
	"strings"
	"testing"

	"github.com/tosin2013/chalkeus/internal/decisions"
)

func TestStoryName(t *testing.T) {
	for in, want := range map[string]string{
		"Phase 1: User Story 1 - Node 24 LTS with a clean test run (Priority: P1) 🎯 MVP": "Node 24 LTS with a clean test run",
		"Phase 2: User Story 2 - Flagship documentation-analysis skill (Priority: P2)":   "Flagship documentation-analysis skill",
		"Phase 3: Polish": "Polish",
		"Setup":           "Setup",
	} {
		if got := storyName(in); got != want {
			t.Errorf("%q → %q, want %q", in, got, want)
		}
	}
}

// sliceState is a slice run's state as readiness records it (documcp 001 US2).
func sliceState(state string) map[string]any {
	return map[string]any{"state": state, "item": "001", "slice": "US2",
		"target": map[string]any{"branch": "main"},
		"mode":   map[string]any{"name": "step-by-step"},
		"readiness": map[string]any{"title": "v2.0 foundation", "spec_dir": "specs/001-v2-foundation",
			"slice": map[string]any{"index": 1, "of": 3,
				"phases": []any{"Phase 2: User Story 2 - Flagship documentation-analysis skill (Priority: P2)"}}},
		"pending": map[string]any{"task": "001", "n": 2}, "tasks": map[string]any{}}
}

func TestTrackingIssueNamesTheWorkForVisitors(t *testing.T) {
	f := newFake("pub-bot", map[string]string{"alice": "write"})
	l := f.loop("alice")
	l.Repo = "o/documcp"
	run := testRun(t, "20261127T173819Z", sliceState("paused(awaiting-retry)"))
	if _, err := l.PostDecision(run, decisions.PendingStep(run)); err != nil {
		t.Fatal(err)
	}
	i := f.issue(TrackingIssue(run))
	if i.Title != "Chalkeus: Flagship documentation-analysis skill (001 · US2)" {
		t.Errorf("title %q", i.Title)
	}
	if len(i.Labels) != 1 || i.Labels[0] != TrackingLabel {
		t.Errorf("labels %v", i.Labels)
	}
	for _, want := range []string{
		"<!-- chalkeus:run=20261127T173819Z -->", // the hidden marker stays first
		"[001: v2.0 foundation](https://github.com/o/documcp/blob/main/specs/001-v2-foundation/spec.md)",
		"US2: Flagship documentation-analysis skill (part 2 of 3)",
		"**Just visiting?** You don't need to do anything",
		"@alice",
		"never merges",
		"run `20261127T173819Z` · step-by-step mode",
		// the post updated the Status row
		"| **Status** | ⏸ attempt 1 didn't pass; waiting for a maintainer to approve another attempt or a replan |",
	} {
		if !strings.Contains(i.Body, want) {
			t.Errorf("body lacks %q:\n%s", want, i.Body)
		}
	}
	if !strings.HasPrefix(i.Body, "<!-- chalkeus:run=20261127T173819Z -->\n") {
		t.Errorf("marker not first: %q", i.Body[:60])
	}
}

func TestDiagnosisIsFoldedInThePost(t *testing.T) {
	f := newFake("pub-bot", map[string]string{"alice": "write"})
	st := sliceState("paused(awaiting-retry)")
	st["decision_note"] = map[string]any{"step": "retry", "text": "  diagnosis of attempt 1 (001): grade_fail\n  recommended: retry\n"}
	run := testRun(t, "20261127T173819Z", st)
	if _, err := f.loop("alice").PostDecision(run, decisions.PendingStep(run)); err != nil {
		t.Fatal(err)
	}
	body := f.comments[TrackingIssue(run)][0].Body
	if !strings.Contains(body, "<details><summary>Diagnosis</summary>\n\n```text\n  diagnosis of attempt 1 (001): grade_fail\n  recommended: retry\n```\n</details>") {
		t.Fatalf("post %q", body)
	}
}

func TestCloseShowsTheOutcomeInTheIssue(t *testing.T) {
	for outcome, want := range map[string]string{
		"succeeded":                 "✅ finished: pull request #226 merged",
		"stopped: back to planning": "⏹ ended: back to planning",
		"failed (AUDIT)":            "❌ ended without a result: failed (AUDIT)",
	} {
		f := newFake("pub-bot", map[string]string{"alice": "write"})
		l := f.loop("alice")
		st := sliceState("paused(awaiting-retry)")
		st["tasks"] = map[string]any{"001": map[string]any{"pr": map[string]any{"number": 226}}}
		run := testRun(t, "20261127T173819Z", st)
		if _, err := l.PostDecision(run, decisions.PendingStep(run)); err != nil {
			t.Fatal(err)
		}
		if err := l.CloseTracking(run, outcome, "x"); err != nil {
			t.Fatal(err)
		}
		if b := f.issue(TrackingIssue(run)).Body; !strings.Contains(b, "| **Status** | "+want) {
			t.Errorf("%s: body lacks %q:\n%s", outcome, want, b)
		}
	}
}

func TestTitleFallsBackWithoutReadiness(t *testing.T) {
	run := testRun(t, "20261116T101500Z", map[string]any{"item": "007"})
	if got := issueTitle(run); got != "Chalkeus: item 007 (run 20261116T101500Z)" {
		t.Errorf("%q", got)
	}
	run = testRun(t, "20261116T101501Z", map[string]any{})
	if got := issueTitle(run); got != "Chalkeus run 20261116T101501Z" {
		t.Errorf("%q", got)
	}
}

func TestLabelFailureNeverBlocksTheIssue(t *testing.T) {
	f := newFake("pub-bot", map[string]string{"alice": "write"})
	f.fail = "ensure-labels"
	run := testRun(t, "20261127T173819Z", sliceState("paused(awaiting-retry)"))
	if _, err := f.loop("alice").PostDecision(run, decisions.PendingStep(run)); err != nil {
		t.Fatal(err)
	}
	if i := f.issue(TrackingIssue(run)); len(i.Labels) != 0 {
		t.Fatalf("labels %v", i.Labels)
	}
}
