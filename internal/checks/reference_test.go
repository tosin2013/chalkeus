package checks

import (
	"strings"
	"testing"
)

func TestReferenceAbsent(t *testing.T) {
	state := map[string]any{"readiness": map[string]any{"references": []any{
		map[string]any{"path": "documcp/references/004.patch", "sha256": "abc123def456"}}}}
	staged := []string{"base.bundle", "meta.json", "prompt.md"}
	if ok, d := ReferenceAbsent(state, "cm data", "the prompt", staged); !ok || d != "references=1 leaks=[]" {
		t.Fatalf("%t %s", ok, d)
	}
	for name, c := range map[string][3]string{
		"path in prompt":    {"", "see documcp/references/004.patch", "path:documcp/references/004.patch"},
		"hash in configmap": {"x abc123def456", "", "sha256:abc123def456"},
	} {
		if ok, d := ReferenceAbsent(state, c[0], c[1], staged); ok || !strings.Contains(d, c[2]) {
			t.Errorf("%s: %t %s", name, ok, d)
		}
	}
	if ok, d := ReferenceAbsent(state, "", "", append(staged, "reference.patch")); ok || !strings.Contains(d, "staged:reference.patch") {
		t.Errorf("an extra staged input: %t %s", ok, d)
	}
	// A run without references (before 041, or reference: none) passes on the staged inputs alone.
	if ok, _ := ReferenceAbsent(map[string]any{}, "", "", staged); !ok {
		t.Error("no references recorded")
	}
}
