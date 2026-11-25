package diagnose

import (
	"strings"
	"testing"
)

// 033 US5 scenarios 1–2: unknown work names the Research Harness, installed or not.
func TestToolsResearchHarness(t *testing.T) {
	d := replay(t, "documcp-001-1", "001", 2400)
	if len(d.Tools) != 1 || d.Tools[0].ID != "harness" || d.Tools[0].Installed {
		t.Fatalf("%+v", d.Tools)
	}
	if !strings.Contains(d.Text, "the harness extension, not installed (specify extension add harness --from") ||
		!strings.Contains(d.Text, "not vetted by Spec Kit: review it first") {
		t.Errorf("install hint:\n%s", d.Text)
	}

	in := replayInput(t, "documcp-001-1", "001", 2400)
	in.Extensions = []string{"harness", "roadmap", "spec-status"}
	d = Diagnose(in)
	if !d.Tools[0].Installed || !strings.Contains(d.Text, "tool for the research: /speckit.harness.explore, then /speckit.harness.report") {
		t.Errorf("installed:\n%s", d.Text)
	}
}

// Scenario 3: an unrecognised failure names Bug Triage.
func TestToolsBugTriage(t *testing.T) {
	d := Diagnose(Input{Task: "001", Attempt: 1, Reason: "kaboom", Extensions: []string{"bug"}})
	var ids []string
	for _, x := range d.Tools {
		ids = append(ids, x.ID)
	}
	if strings.Join(ids, ",") != "harness,bug" ||
		!strings.Contains(d.Text, "tool for the defect: /speckit.bug.assess with this diagnosis as the report, then /speckit.bug.fix, then /speckit.bug.test (installed)") {
		t.Fatalf("%v\n%s", ids, d.Text)
	}
	d = Diagnose(Input{Task: "001", Attempt: 1, Reason: "kaboom"})
	if !strings.Contains(d.Text, "the bug extension, not installed (specify extension add bug)") {
		t.Errorf("bug is vetted:\n%s", d.Text)
	}
	// A plain retry names no tool.
	if d := Diagnose(Input{Task: "A", Attempt: 1, Reason: "grade_fail", FailedChecks: []string{"G5_hidden_tests"}}); len(d.Tools) != 0 {
		t.Errorf("%+v", d.Tools)
	}
}

// Scenario 4: a second deadline in a row with nothing else is unknown work.
func TestRepeatedDeadline(t *testing.T) {
	in := Input{Task: "001", Attempt: 2, Reason: "deadline", DeadlineS: 1200}
	if d := Diagnose(in); d.Recommend != Retry {
		t.Fatalf("one deadline: %s", d.Recommend)
	}
	in.PrevDeadlineNoTrig = true
	d := Diagnose(in)
	if d.Recommend != BackToPlanning || !strings.Contains(d.Text, "attempts 1 and 2 both stopped at the deadline") {
		t.Fatalf("%s\n%s", d.Recommend, d.Text)
	}
}

func TestEnabledExtensions(t *testing.T) {
	reg := `{"schema_version":"1.0","extensions":{"spec-status":{"enabled":true},"harness":{"enabled":false},"bug":{}}}`
	if got := strings.Join(EnabledExtensions([]byte(reg)), ","); got != "bug,spec-status" {
		t.Fatal(got)
	}
	if EnabledExtensions([]byte("not json")) != nil {
		t.Fatal("unreadable registry")
	}
}
