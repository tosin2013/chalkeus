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

// 014: an item run's merge step and terminal status approval use its item, not A.
func TestItemRunPendingSteps(t *testing.T) {
	r := makeRun(t, "paused(awaiting-merge)", map[string]any{"task": "002"})
	r.State["kind"], r.State["item"] = "item", "002"
	if p := PendingStep(r); p == nil || p.Step != "merge" || p.Task != "002" {
		t.Fatalf("%+v", p)
	}
	if code, msg := Record(r, "mode", "harness", "", nil); code != 4 || !strings.Contains(msg, "merge 002's PR") {
		t.Fatalf("%d %s", code, msg)
	}
	_ = os.MkdirAll(filepath.Join(r.Dir, "reconcile"), 0o755)
	_ = evidence.WriteJSON(filepath.Join(r.Dir, "reconcile", "002.json"), map[string]any{"reason": "DECLINED",
		"proposed": "specced→verified", "merge_commit": "abc", "item_id": "002"})
	p := PendingStep(r)
	if p == nil || p.Step != "reconcile" || p.Task != "002" || !strings.Contains(Describe(p), "to the target's roadmap") {
		t.Fatalf("%+v %s", p, Describe(p))
	}
	m := makeRun(t, "paused(awaiting-mode)", nil)
	m.State["proposed_mode"] = map[string]any{"name": "step-by-step"}
	m.State["proposed_target"] = "target g (me/g@main), item 002, toolchain node24, registries: registry.npmjs.org"
	if d := Describe(PendingStep(m)); !strings.HasSuffix(d, "; target g (me/g@main), item 002, toolchain node24, registries: registry.npmjs.org") {
		t.Fatal(d)
	}
}

// With PILOT_NO_TTY=1 (step mode's subprocesses, 021) a terminal is never asked, even one
// that is there: the decision waits for an approval recorded from GitHub or the harness.
func TestDecideNoTTY(t *testing.T) {
	oh, oa := HasTTY, AskTTY
	asked := false
	HasTTY, AskTTY = func() bool { return true }, func(string) string { asked = true; return "y\n" }
	defer func() { HasTTY, AskTTY = oh, oa }()
	t.Setenv("PILOT_NO_TTY", "1")
	run := makeRun(t, "paused(awaiting-mode)", nil)
	ok, ch := Decide(run, "mode", "Confirm? ", nil, nil, func(string) {})
	if ok || ch == "tty" || asked {
		t.Fatalf("approved=%t channel=%s asked=%t", ok, ch, asked)
	}
}
