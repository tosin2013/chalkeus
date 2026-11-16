package decisions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tosin2013/specfoundry/internal/evidence"
)

// Port of the record/take/decide parts of the former Python pilot's tests/test_approvals.py.

func makeRun(t *testing.T, state string, pending map[string]any) *evidence.Run {
	t.Helper()
	d := filepath.Join(t.TempDir(), "20990101T000000Z")
	_ = os.MkdirAll(filepath.Join(d, "reconcile"), 0o755)
	r := &evidence.Run{Dir: d, ID: filepath.Base(d), State: map[string]any{"run_id": filepath.Base(d), "state": state,
		"pending": pending, "event_seq": 0, "proposed_mode": map[string]any{"name": "step-by-step"}}}
	if err := r.Save(); err != nil {
		t.Fatal(err)
	}
	HasTTY = func() bool { return false }
	return r
}

func TestRecord(t *testing.T) {
	if code, _ := Record(makeRun(t, "in-progress", nil), "mode", "", "", nil); code != 2 {
		t.Error("not paused must be refused")
	}
	if code, msg := Record(makeRun(t, "paused(awaiting-retry)", map[string]any{"task": "A", "n": 2}), "mode", "", "", nil); code != 2 || !strings.Contains(msg, "retry") {
		t.Errorf("wrong step: %d %s", code, msg)
	}
	r := makeRun(t, "paused(awaiting-merge)", map[string]any{"task": "A"})
	if code, msg := Record(r, "reconcile", "", "", nil); code != 4 || !strings.Contains(msg, "GitHub") {
		t.Errorf("merge: %d %s", code, msg)
	}
	if _, err := os.Stat(file(r)); err == nil {
		t.Error("nothing may be recorded for a merge")
	}
	_ = evidence.WriteJSON(filepath.Join(r.Dir, "reconcile", "A.json"), map[string]any{"reason": "DECLINED", "proposed": "planned→verified", "merge_commit": "abc", "item_id": "001"})
	if code, msg := Record(r, "reconcile", "", "", nil); code != 0 || !strings.Contains(msg, "abc") {
		t.Errorf("reconcile after proposal: %d %s", code, msg)
	}
}

func TestTake(t *testing.T) {
	r := makeRun(t, "paused(awaiting-mode)", nil)
	Record(r, "mode", "", "", nil)
	if TakeRecord(r, "mode", nil, nil) == nil || TakeRecord(r, "mode", nil, nil) != nil {
		t.Error("used once")
	}
	r = makeRun(t, "paused(awaiting-retry)", map[string]any{"task": "A", "n": 2})
	Record(r, "retry", "", "", nil)
	if TakeRecord(r, "retry", "A", 3) != nil || TakeRecord(r, "retry", "A", 2) != nil {
		t.Error("bound to attempt, consumed by the mismatch")
	}
	r = makeRun(t, "paused(awaiting-merge)", map[string]any{"task": "A"})
	rec := filepath.Join(r.Dir, "reconcile", "A.json")
	_ = evidence.WriteJSON(rec, map[string]any{"reason": "DECLINED", "proposed": "planned→verified", "merge_commit": "abc"})
	Record(r, "reconcile", "", "", nil)
	_ = evidence.WriteJSON(rec, map[string]any{"reason": "DECLINED", "proposed": "planned→verified", "merge_commit": "other"})
	if TakeRecord(r, "reconcile", "A", nil) != nil {
		t.Error("bound to merge commit")
	}
}

func TestDecide(t *testing.T) {
	say := func(string) {}
	r := makeRun(t, "paused(awaiting-mode)", nil)
	if ok, ch := Decide(r, "mode", "Q? ", nil, nil, say); ok || ch != "none" {
		t.Error("no tty, no approval is no")
	}
	Record(r, "mode", "", "", nil)
	if ok, ch := Decide(r, "mode", "Q? ", nil, nil, say); !ok || ch != "harness" {
		t.Errorf("with approval: %v %s", ok, ch)
	}
	Record(r, "mode", "", "", nil)
	HasTTY = func() bool { return true }
	AskTTY = func(string) string { return "n\n" }
	if ok, ch := Decide(r, "mode", "Q? ", nil, nil, say); ok || ch != "tty" {
		t.Error("terminal decides")
	}
	if _, err := os.Stat(file(r)); err == nil {
		t.Error("a terminal answer discards the pending approval")
	}
	HasTTY = func() bool { return false }
	if ok, _ := Decide(r, "mode", "Q? ", nil, nil, say); ok {
		t.Error("a later headless decision must not use the overridden approval")
	}
}

func TestGitHubChannelNote(t *testing.T) {
	r := makeRun(t, "paused(awaiting-retry)", map[string]any{"task": "A", "n": 2})
	if code, msg := Record(r, "retry", "github", "alice", 12); code != 0 || !strings.Contains(msg, "by @alice on GitHub") {
		t.Fatal(msg)
	}
	ok, ch := Decide(r, "retry", "Retry? ", "A", 2, func(string) {})
	if !ok || ch != "github" {
		t.Fatalf("%v %s", ok, ch)
	}
	Note(r, "retry", "A", 2, ok, ch, "", "")
	ev, _ := r.Events()
	if d := ev[len(ev)-1]["detail"]; d != "step=retry approved=true channel=github approver=alice comment=12" {
		t.Fatalf("%v", d)
	}
	r.LastApproval = map[string]any{"approver": "old", "comment_id": 5}
	Note(r, "reconcile", "A", nil, true, "github", "bob", "pr=9")
	ev, _ = r.Events()
	if d := ev[len(ev)-1]["detail"]; d != "step=reconcile approved=true channel=github approver=bob pr=9" {
		t.Fatalf("%v", d)
	}
}
