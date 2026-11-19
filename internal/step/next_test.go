package step

import (
	"strings"
	"testing"
)

func base(id string) ItemState {
	return ItemState{ID: id, Proposed: true, Ready: true, Hidden: 1, Commits: "t1:c1"}
}

// Every row of contracts/step.md's decision table.
func TestNext(t *testing.T) {
	cases := []struct {
		name string
		mod  func(*ItemState)
		want string
	}{
		{"not proposed", func(s *ItemState) { s.Proposed, s.NotWhy = false, "no label" }, Skip},
		{"not ready", func(s *ItemState) { s.Ready, s.ReadyWhy = false, "ITEM_OPEN_QUESTIONS" }, Skip},
		{"tests PR open", func(s *ItemState) { s.Hidden, s.TestsPR = 0, true }, Wait},
		{"no tests, auto-draft on", func(s *ItemState) { s.Hidden, s.AutoDraft = 0, true }, Draft},
		{"no tests, auto-draft off", func(s *ItemState) { s.Hidden = 0 }, NeedsTests},
		{"tests merged", func(s *ItemState) {}, Doctor},
		{"doctor failed, same commits", func(s *ItemState) { s.DoctorFailedAt = "t1:c1" }, DoctorFailed},
		{"doctor failed, target changed", func(s *ItemState) { s.DoctorFailedAt = "t0:c1" }, Doctor},
		{"run active", func(s *ItemState) { s.Run, s.RunID = RunActive, "R1" }, Wait},
		{"run active, label removed", func(s *ItemState) { s.Run, s.Proposed = RunActive, false }, Wait},
		{"run succeeded", func(s *ItemState) { s.Run = RunSucceeded }, Done},
		{"run failed", func(s *ItemState) { s.Run = RunFailed }, RunFailedAct},
		{"run failed, relabelled", func(s *ItemState) { s.Run, s.RelabelledAfterRun = RunFailed, true }, Doctor},
	}
	for _, c := range cases {
		s := base("009")
		c.mod(&s)
		if got := Next(s); got.Kind != c.want || got.Item != "009" || got.Reason == "" {
			t.Errorf("%s: got %v, want %s", c.name, got, c.want)
		}
	}
}

func TestPlanOneItemInFlight(t *testing.T) {
	a, b, c := base("007"), base("008"), base("009")
	a.Run = RunSucceeded
	p := Plan([]ItemState{a, b, c})
	if p[0].Kind != Done || p[1].Kind != Doctor || p[2].Kind != Skip || !strings.Contains(p[2].Reason, "008 is in flight") {
		t.Fatalf("%v", p)
	}
	// A run started by hand on a later, unproposed item holds the slot.
	c.Proposed, c.Run, c.RunID = false, RunActive, "R9"
	p = Plan([]ItemState{a, b, c})
	if p[1].Kind != Skip || !strings.Contains(p[1].Reason, "009 is in flight") || p[2].Kind != Wait {
		t.Fatalf("%v", p)
	}
	// Failures don't hold the slot: the next proposed item goes on.
	b.Run, c = RunFailed, base("009")
	p = Plan([]ItemState{a, b, c})
	if p[1].Kind != RunFailedAct || p[2].Kind != Doctor {
		t.Fatalf("%v", p)
	}
}
