package pipeline

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// 033 FR-001: a failed attempt gets diagnosis.json, its record the recommendation, and an event.
func TestDiagnoseAttemptRecorded(t *testing.T) {
	p, r, _ := newP(t)
	src := filepath.Join(repoRoot(t), "internal", "diagnose", "testdata", "documcp-001-1")
	dir, err := r.AttemptDir("A", 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"report.json", "meta.json", "proxy-denials.log", "tasklist.md"} {
		b, err := os.ReadFile(filepath.Join(src, f))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, f), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var att map[string]any
	b, _ := os.ReadFile(filepath.Join(src, "attempt.json"))
	_ = json.Unmarshal(b, &att)
	r.State["mode"] = map[string]any{"worker_deadline_s": 2400.0}
	r.State["toolchain"] = map[string]any{"name": "node24"}
	r.State["tasks"] = map[string]any{"A": map[string]any{"attempts": []any{att}}}

	p.diagnoseAttempt(r, "A", 1, att)

	d, ok := attemptDiagnosis(r, "A", 1)
	if !ok || d.Recommend != "back_to_planning" || !strings.Contains(d.Text, "stopped at the 2400 s deadline") {
		t.Fatalf("%v %+v", ok, d)
	}
	if dg, _ := att["diagnosis"].(map[string]any); dg["recommend"] != "back_to_planning" {
		t.Fatalf("attempt record: %v", att["diagnosis"])
	}
	if _, err := r.WriteSummary(); err != nil {
		t.Fatal(err)
	}
	sum, _ := os.ReadFile(filepath.Join(r.Dir, "summary.md"))
	if !strings.Contains(string(sum), "## Diagnosis") || !strings.Contains(string(sum), "recommended: back to planning") {
		t.Fatalf("summary lacks the diagnosis:\n%s", sum)
	}
	ev, _ := r.Events()
	if last := ev[len(ev)-1]; last["event"] != "attempt_diagnosed" || last["detail"] != "back_to_planning" {
		t.Fatalf("%v", last)
	}
}

// 033 FR-022 input: the previous attempt stopped at the deadline with no trigger.
func TestPrevDeadlineNoTrigger(t *testing.T) {
	_, r, _ := newP(t)
	r.State["tasks"] = map[string]any{"001": map[string]any{"attempts": []any{
		map[string]any{"n": 1.0, "reason": "deadline", "diagnosis": map[string]any{"recommend": "retry", "triggers": []any{}}},
		map[string]any{"n": 2.0, "reason": "deadline", "diagnosis": map[string]any{"recommend": "back_to_planning", "triggers": []any{"unknown_work"}}},
		map[string]any{"n": 3.0, "reason": "grade_fail"},
	}}}
	if !prevDeadlineNoTrigger(r, "001", 2) {
		t.Error("attempt 1 stopped at the deadline with nothing")
	}
	if prevDeadlineNoTrigger(r, "001", 3) {
		t.Error("attempt 2 had a trigger")
	}
	if prevDeadlineNoTrigger(r, "001", 4) || prevDeadlineNoTrigger(r, "001", 1) {
		t.Error("attempt 3 was graded; attempt 0 doesn't exist")
	}
}

// 033 FR-019: the target's enabled extensions at the base revision.
func TestRecordExtensions(t *testing.T) {
	_, r, _ := newP(t)
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("%v: %s", args, out)
		}
	}
	run("init", "-q", "-b", "main")
	_ = os.MkdirAll(filepath.Join(dir, ".specify", "extensions"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, ".specify", "extensions", ".registry"), []byte(`{"extensions":{"harness":{"enabled":true},"bug":{"enabled":false}}}`), 0o644)
	run("add", "-A")
	run("-c", "user.email=t@t", "-c", "user.name=t", "commit", "-qm", "x")
	recordExtensions(r, dir, "main")
	if got := strList(r.State["spec_extensions"]); len(got) != 1 || got[0] != "harness" {
		t.Fatalf("%v", r.State["spec_extensions"])
	}
	recordExtensions(r, t.TempDir(), "main")
	if got := strList(r.State["spec_extensions"]); got == nil || len(got) != 0 {
		t.Fatalf("no repository: %v", r.State["spec_extensions"])
	}
}
