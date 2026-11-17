package grader

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/tosin2013/specfoundry/internal/k8s"
)

func TestItemGradingTemplates(t *testing.T) {
	root, _ := filepath.Abs("../..")
	ns, err := k8s.Render(filepath.Join(root, "pilot/manifests/grading-ns.yaml.tmpl"), map[string]string{
		"NS": NS("002"), "RUN_ID": "r", "QUOTA_CPU": "2", "QUOTA_MEMORY": "2Gi"})
	if err != nil || !strings.Contains(ns, "name: pilot-g-002") || !strings.Contains(ns, `count/secrets: "0"`) ||
		!strings.Contains(ns, "limits.memory: 2Gi") || !strings.Contains(ns, "policyTypes: [Ingress, Egress]") {
		t.Fatalf("%v\n%s", err, ns)
	}
	job, err := k8s.Render(filepath.Join(root, "pilot/manifests/grader-job-item.yaml.tmpl"), map[string]string{
		"NS": "pilot-g-002", "JOB": "grader-1", "DEADLINE": "600", "TOOLS_IMAGE": "t@sha256:0", "GRADER_CPU": "1",
		"GRADER_MEMORY": "1Gi", "INPUT_LIMIT": "64Mi"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"name: grader-1", "activeDeadlineSeconds: 600", "name: receiver",
		"{name: stage, mountPath: /in, subPath: in, readOnly: true}", "{name: stage, mountPath: /grading, subPath: grading, readOnly: true}",
		"pilot/role: grader", "automountServiceAccountToken: false"} {
		if !strings.Contains(job, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(job, "configMap:") || strings.Contains(job, "secret:") || strings.Contains(job, "secretName") {
		t.Error("item grading takes no ConfigMap or Secret")
	}
}

func TestBaselineHelpers(t *testing.T) {
	m := BaselineMeta(map[string]any{"attempt": 3, "mode": "attempt", "test_command": "x"})
	if m["mode"] != "baseline" || m["attempt"] != 0 || m["test_command"] != "x" {
		t.Fatal(m)
	}
	rep := func(g4, g5 string) map[string]any {
		return map[string]any{"checks": []any{map[string]any{"id": "G4_visible_tests", "result": g4}, map[string]any{"id": "G5_hidden_tests", "result": g5}}}
	}
	for want, r := range map[string]map[string]any{
		"BASELINE_SUITE_FAILS":   rep("fail", "skipped"),
		"BASELINE_HIDDEN_PASSES": rep("pass", "fail"),
		"BASELINE_GRADER_ERROR":  rep("skipped", "skipped"),
	} {
		if got := BaselineReason(r); got != want {
			t.Errorf("%s: got %s", want, got)
		}
	}
}
