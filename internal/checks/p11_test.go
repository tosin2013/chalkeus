package checks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHiddenTestsAbsent(t *testing.T) {
	d := t.TempDir()
	_ = os.WriteFile(filepath.Join(d, "companion.json"), []byte(`{"files": [{"path": "src/test/java/AcceptanceTest.java", "sha256": "x"}]}`), 0o644)
	staged := []string{"base.bundle", "meta.json", "prompt.md"}
	if ok, detail := HiddenTestsAbsent(d, "kube-root-ca.crt\n...", "# Specification\nadd shout", staged); !ok {
		t.Fatal(detail)
	}
	for name, c := range map[string]struct {
		cm, prompt string
		staged     []string
	}{
		"in a ConfigMap": {"leak\nsrc/test/java/AcceptanceTest.java", "", staged},
		"in the prompt":  {"", "see src/test/java/AcceptanceTest.java", staged},
		"extra staged":   {"", "", append(staged, "overlay.tar")},
	} {
		if ok, detail := HiddenTestsAbsent(d, c.cm, c.prompt, c.staged); ok || !strings.Contains(detail, "leaks=") {
			t.Errorf("%s: %v %s", name, ok, detail)
		}
	}
	if ok, _ := HiddenTestsAbsent(t.TempDir(), "", "", staged); ok {
		t.Error("no companion.json must fail closed")
	}
}
