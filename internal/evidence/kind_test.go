package evidence

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestKindDefaultsToFixture(t *testing.T) {
	r := &Run{State: map[string]any{"run_id": "x"}}
	if r.Kind() != KindFixture || !reflect.DeepEqual(r.TaskLabels(), []string{"A", "B"}) {
		t.Fatal(r.Kind(), r.TaskLabels())
	}
}

func TestItemRunSummary(t *testing.T) {
	d := filepath.Join(t.TempDir(), "20990101T000000Z")
	_ = os.MkdirAll(d, 0o755)
	r := &Run{Dir: d, ID: filepath.Base(d), State: map[string]any{"run_id": filepath.Base(d), "state": "succeeded",
		"tasks": map[string]any{"003": map[string]any{"attempts": []any{map[string]any{"n": 1, "result": "pass"}}}}}}
	r.SetItemRun(map[string]any{"name": "greeting", "repo": "me/g", "branch": "main", "toolchain": "jdk17-maven"}, "003")
	if r.Kind() != KindItem || !reflect.DeepEqual(r.TaskLabels(), []string{"003"}) {
		t.Fatal(r.Kind(), r.TaskLabels())
	}
	if _, err := r.WriteSummary(); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(d, "summary.md"))
	s := string(raw)
	if !strings.HasPrefix(s, "# pilot item run 20990101T000000Z") || !strings.Contains(s, "| 003 | 1 | - | pass |") {
		t.Fatal(s)
	}
}

func TestSummaryElapsedIgnoresLaterAudits(t *testing.T) {
	d := filepath.Join(t.TempDir(), "20990101T000000Z")
	_ = os.MkdirAll(d, 0o755)
	lines := []string{
		`{"at": "2099-01-01T00:00:00Z", "attempt": null, "detail": "", "event": "run_started", "seq": 1, "stage": "run", "task": null}`,
		`{"at": "2099-01-01T00:10:00Z", "attempt": null, "detail": "", "event": "run_succeeded", "seq": 2, "stage": "run", "task": null}`,
		`{"at": "2099-01-01T00:10:00Z", "attempt": null, "detail": "10/10", "event": "audit_done", "seq": 3, "stage": "audit", "task": null}`,
		`{"at": "2099-01-01T05:00:00Z", "attempt": null, "detail": "10/11", "event": "audit_done", "seq": 4, "stage": "audit", "task": null}`,
	}
	_ = os.WriteFile(filepath.Join(d, "events.jsonl"), []byte(strings.Join(lines, "\n")+"\n"), 0o644)
	r := &Run{Dir: d, ID: filepath.Base(d), State: map[string]any{"run_id": filepath.Base(d), "state": "succeeded"}}
	if _, err := r.WriteSummary(); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(d, "summary.md"))
	if !strings.Contains(string(raw), "**Elapsed:** 600 s total") {
		t.Fatal(string(raw))
	}
}
