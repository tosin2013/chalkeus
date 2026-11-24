package worker

import "testing"

// 030 (contracts/worker-soft-deadline.md): how a finished worker Job is classified.
func TestFailure(t *testing.T) {
	spent := map[string]any{"tokens": 10.0, "stopped_by_cap": false}
	for _, c := range []struct {
		name, outcome, frameErr, reportErr string
		frame                              bool
		spend                              map[string]any
		reason, note                       string
	}{
		{"soft deadline keeps evidence, never graded", "Complete", "", "deadline", true, spent, "deadline", ""},
		{"hard deadline without a frame", "DeadlineExceeded", "", "", false, nil, "deadline", "no_frame"},
		{"hard deadline with a frame", "DeadlineExceeded", "", "", true, spent, "deadline", ""},
		{"spend cap", "Complete", "", "spend_cap", true, map[string]any{"stopped_by_cap": true}, "spend_cap", ""},
		{"bad frame", "Complete", "frame_invalid", "", false, nil, "frame_invalid", ""},
		{"collected for grading", "Complete", "", "", true, spent, "", ""},
	} {
		r, n := Failure(c.outcome, c.frameErr, c.reportErr, c.frame, c.spend)
		if r != c.reason || n != c.note {
			t.Errorf("%s: got %q %q", c.name, r, n)
		}
	}
}

func TestLimitsDefaults(t *testing.T) {
	if l := (Limits{}).Or(); l.CPU != "2" || l.Memory != "2Gi" || l.DeadlineS != Deadline || l.SoftDeadlineS != 0 {
		t.Fatalf("%+v", l)
	}
	if l := (Limits{CPU: "4", DeadlineS: 2400, SoftDeadlineS: 2280}).Or(); l.CPU != "4" || l.Memory != "2Gi" || l.DeadlineS != 2400 {
		t.Fatalf("%+v", l)
	}
}
