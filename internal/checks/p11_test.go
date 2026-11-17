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

func netpol(name string, labels map[string]any, egress []any) map[string]any {
	spec := map[string]any{"podSelector": map[string]any{}, "policyTypes": []any{"Egress"}}
	if labels != nil {
		spec["podSelector"] = map[string]any{"matchLabels": labels}
	}
	if egress != nil {
		spec["egress"] = egress
	}
	return map[string]any{"metadata": map[string]any{"name": name}, "spec": spec}
}

func to(ns string) any {
	return map[string]any{"to": []any{map[string]any{"namespaceSelector": map[string]any{"matchLabels": map[string]any{"kubernetes.io/metadata.name": ns}}}}}
}

func TestGradingPolicies(t *testing.T) {
	deny := netpol("default-deny-all", nil, nil)
	prefetch := netpol("allow-prefetch-egress", map[string]any{"pilot/role": "prefetch"}, []any{to("kube-system"), to("pilot-egress-r-p")})
	if ok, d := GradingPolicies([]map[string]any{deny, prefetch}, "pilot-egress-r-p"); !ok {
		t.Fatal(d)
	}
	if ok, _ := GradingPolicies([]map[string]any{deny}, "pilot-egress-r-p"); !ok {
		t.Fatal("no cache: the default deny alone is fine")
	}
	for name, nps := range map[string][]map[string]any{
		"no default deny":       {prefetch},
		"grader egress":         {deny, netpol("allow-grader", map[string]any{"pilot/role": "grader"}, []any{to("pilot-egress-model")})},
		"prefetch to the model": {deny, netpol("allow-prefetch-egress", map[string]any{"pilot/role": "prefetch"}, []any{to("pilot-egress-model")})},
		"prefetch for all pods": {deny, netpol("allow-prefetch-egress", nil, []any{to("pilot-egress-r-p")})},
	} {
		if ok, d := GradingPolicies(nps, "pilot-egress-r-p"); ok {
			t.Errorf("%s: passed (%s)", name, d)
		}
	}
}
