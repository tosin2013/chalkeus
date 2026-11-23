package pipeline

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tosin2013/chalkeus/internal/paths"
)

// Ported from TestWatch in the former Python pilot's tests/test_ghapprove.py: `github watch` stops when the run is
// no longer paused and returns the last resume's code.
func TestWatchStopsWhenRunIsDone(t *testing.T) {
	root := t.TempDir()
	l := paths.At(root)
	dir := filepath.Join(l.EvidenceRoot(), "X")
	_ = os.MkdirAll(dir, 0o755)
	raw, _ := json.Marshal(map[string]any{"run_id": "X", "state": "succeeded"})
	_ = os.WriteFile(filepath.Join(dir, "run.json"), raw, 0o644)
	var out bytes.Buffer
	p := &P{L: l, Out: &out}
	var slept []time.Duration
	if code := p.Watch("X", time.Second, func(d time.Duration) { slept = append(slept, d) }); code != ExitOK {
		t.Fatal(code, out.String())
	}
	if out.String() != "run X is succeeded; stopped watching\n" || len(slept) != 0 {
		t.Fatalf("%q %v", out.String(), slept)
	}
	if code := p.Watch("nope", time.Second, nil); code != ExitError {
		t.Fatal(code)
	}
}
