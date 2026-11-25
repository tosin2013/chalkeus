package pipeline

import (
	"encoding/json"
	"os"
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
