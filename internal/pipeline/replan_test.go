package pipeline

import (
	"strings"
	"testing"

	"github.com/tosin2013/chalkeus/internal/decisions"
)

// 033 US2: `replan` answers a retry pause; the run ends as stopped(back-to-planning), recorded.
func TestReplanStopsRunForPlanning(t *testing.T) {
	p, r, out := newP(t)
	r.State["state"] = "paused(awaiting-retry)"
	r.State["pending"] = map[string]any{"task": "A", "n": 2}
	_ = r.Save()

	if code, msg := decisions.Record(r, decisions.Replan, "harness", "", nil); code != 0 || !strings.Contains(msg, "back to planning") {
		t.Fatalf("%d %s", code, msg)
	}
	if !decisions.Recorded(r, decisions.Replan, "A", 2) || decisions.Recorded(r, "retry", "A", 2) {
		t.Fatal("the record must be a replan, not a retry")
	}
	if code := p.stopForPlanning(r, "A", 2); code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	if r.State["state"] != StoppedForPlanning || r.State["reason"] != "BACK_TO_PLANNING" {
		t.Fatalf("state %v reason %v", r.State["state"], r.State["reason"])
	}
	st, _ := r.State["stopped"].(map[string]any)
	if st["channel"] != "harness" || st["task"] != "A" || st["attempt"] != 1 || st["decision"] != "back-to-planning" {
		t.Fatalf("stopped %v", st)
	}
	if _, ok := r.State["pending"]; ok {
		t.Fatal("pending must be cleared")
	}
	if decisions.PendingStep(r) != nil {
		t.Fatal("a stopped run waits for nothing")
	}
	if !strings.Contains(out.String(), "stopped: back to planning") {
		t.Fatalf("%q", out.String())
	}
	ev, _ := r.Events()
	var seen []string
	for _, e := range ev {
		seen = append(seen, e["event"].(string))
	}
	if j := strings.Join(seen, ","); !strings.Contains(j, "approval_recorded") || !strings.Contains(j, "approval_channel") {
		t.Fatalf("events %s", j)
	}
}

// A replan is refused unless the run waits for a retry.
func TestReplanOnlyAnswersRetry(t *testing.T) {
	_, r, _ := newP(t)
	r.State["state"] = "paused(awaiting-mode)"
	_ = r.Save()
	if code, msg := decisions.Record(r, decisions.Replan, "harness", "", nil); code == 0 || !strings.Contains(msg, "waiting for 'mode'") {
		t.Fatalf("%d %s", code, msg)
	}
}
