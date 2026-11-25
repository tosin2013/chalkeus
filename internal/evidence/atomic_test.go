package evidence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// WriteJSON is atomic: a concurrent reader always sees a whole file (the 033 #204 integration
// flake: `forge status` read an empty run.json mid-write).
func TestWriteJSONAtomic(t *testing.T) {
	p := filepath.Join(t.TempDir(), "run.json")
	big := map[string]any{}
	for i := 0; i < 2000; i++ {
		big[fmt.Sprintf("k%04d", i)] = strings.Repeat("x", 50)
	}
	if err := WriteJSON(p, big); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 200; i++ {
			_ = WriteJSON(p, big)
		}
	}()
	for {
		select {
		case <-done:
			entries, _ := os.ReadDir(filepath.Dir(p))
			if len(entries) != 1 {
				t.Fatalf("temp files left: %v", entries)
			}
			return
		default:
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil || len(m) != 2000 {
			t.Fatalf("partial read (%d bytes): %v", len(raw), err)
		}
	}
}
