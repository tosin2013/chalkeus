package grader

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tosin2013/specfoundry/internal/k8s"
	"github.com/tosin2013/specfoundry/internal/toolchain"
	"sigs.k8s.io/yaml"
)

func TestItemGradingTemplates(t *testing.T) {
	root, _ := filepath.Abs("../..")
	ns, err := k8s.Render(filepath.Join(root, "pilot/manifests/grading-ns.yaml.tmpl"), map[string]string{
		"NS": NS("002"), "RUN_ID": "r", "QUOTA_CPU": "2", "QUOTA_MEMORY": "2Gi", "CACHE_BLOCK": ""})
	if err != nil || !strings.Contains(ns, "name: pilot-g-002") || !strings.Contains(ns, `count/secrets: "0"`) ||
		!strings.Contains(ns, "limits.memory: 2Gi") || !strings.Contains(ns, "policyTypes: [Ingress, Egress]") {
		t.Fatalf("%v\n%s", err, ns)
	}
	job, err := k8s.Render(filepath.Join(root, "pilot/manifests/grader-job-item.yaml.tmpl"), map[string]string{
		"NS": "pilot-g-002", "JOB": "grader-1", "DEADLINE": "600", "TOOLS_IMAGE": "t@sha256:0", "GRADER_CPU": "1",
		"GRADER_MEMORY": "1Gi", "INPUT_LIMIT": "64Mi", "TOOLCHAIN_INITS": "", "TOOLCHAIN_ENV": "", "TOOLCHAIN_MOUNTS": "", "TOOLCHAIN_VOLUMES": ""})
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

// 014 R7: with a cache, the namespace gets the PVC and a prefetch-only egress rule; graders mount
// the cache read-only and get the toolchain; the prefetch Job mounts it read-write.
func TestCacheAndToolchainRender(t *testing.T) {
	root, _ := filepath.Abs("../..")
	raw, _ := os.ReadFile(filepath.Join(root, "pilot/images/toolchain-python3.13/toolchain.json"))
	spec, err := toolchain.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	g := ItemGrading{NS: "pilot-g-002", CPU: "2", Memory: "2Gi", DeadlineS: 900, InputLimit: "116Mi", ToolchainImage: "tc@sha256:1",
		ToolchainSpec: &spec, Cache: true, CacheSize: "2Gi", PrefetchProxyNS: "pilot-egress-r-p"}
	parse := func(name, y string) {
		for _, doc := range strings.Split(y, "\n---\n") {
			var v any
			if err := yaml.Unmarshal([]byte(doc), &v); err != nil {
				t.Fatalf("%s not YAML: %v\n%s", name, err, doc)
			}
		}
	}
	ns, err := k8s.Render(filepath.Join(root, "pilot/manifests/grading-ns.yaml.tmpl"), map[string]string{
		"NS": g.NS, "RUN_ID": "r", "QUOTA_CPU": "2", "QUOTA_MEMORY": "2Gi", "CACHE_BLOCK": g.cacheBlock()})
	if err != nil {
		t.Fatal(err)
	}
	parse("grading-ns", ns)
	for _, want := range []string{"kind: PersistentVolumeClaim", "storage: 2Gi", "podSelector: {matchLabels: {pilot/role: prefetch}}",
		"kubernetes.io/metadata.name: pilot-egress-r-p"} {
		if !strings.Contains(ns, want) {
			t.Errorf("grading-ns lacks %q", want)
		}
	}
	vars := g.podVars(true)
	for k, v := range map[string]string{"NS": g.NS, "JOB": "grader-1", "DEADLINE": "900", "TOOLS_IMAGE": "t@sha256:0",
		"GRADER_CPU": "2", "GRADER_MEMORY": "2Gi", "INPUT_LIMIT": "116Mi"} {
		vars[k] = v
	}
	job, err := k8s.Render(filepath.Join(root, "pilot/manifests/grader-job-item.yaml.tmpl"), vars)
	if err != nil {
		t.Fatal(err)
	}
	parse("grader", job)
	for _, want := range []string{"{name: deps, mountPath: /deps, readOnly: true}", "claimName: deps, readOnly: true", "image: tc@sha256:1",
		`{name: VIRTUAL_ENV, value: "/deps/venv"}`} {
		if !strings.Contains(job, want) {
			t.Errorf("grader lacks %q", want)
		}
	}
	pv := g.podVars(false)
	for k, v := range map[string]string{"NS": g.NS, "DEADLINE": "900", "TOOLS_IMAGE": "t@sha256:0", "GRADER_CPU": "2",
		"GRADER_MEMORY": "2Gi", "INPUT_LIMIT": "116Mi", "PROXY_URL": "http://proxy.pilot-egress-r-p.svc.cluster.local:8888"} {
		pv[k] = v
	}
	pf, err := k8s.Render(filepath.Join(root, "pilot/manifests/prefetch-job.yaml.tmpl"), pv)
	if err != nil {
		t.Fatal(err)
	}
	parse("prefetch", pf)
	if !strings.Contains(pf, "{name: deps, mountPath: /deps}") || strings.Contains(pf, "/deps, readOnly: true") ||
		!strings.Contains(pf, "pilot/role: prefetch") || !strings.Contains(pf, `"$(/usr/bin/python3 -c`) {
		t.Errorf("prefetch:\n%s", pf)
	}
}

func TestParseProbe(t *testing.T) {
	good := `{"name":"python3.13","version_command":["python3","--version"],"path":["/opt/toolchain/python/bin"],"env":{},"proxy":"env"}`
	line := func(tj, version string, rc int, setuid []string) string {
		b, _ := json.Marshal(map[string]any{"toolchain_json": tj, "version": version, "version_rc": rc, "setuid": setuid})
		return "noise\nPILOT_PROBE " + string(b) + "\n"
	}
	pr, err := parseProbe(line(good, "Python 3.13.18", 0, nil), "Complete")
	if err != nil || pr.Version != "Python 3.13.18" || pr.Spec.Name != "python3.13" {
		t.Fatalf("%+v %v", pr, err)
	}
	for name, logs := range map[string]string{
		"no result": "cp: cannot stat\n",
		"bad json":  line("{", "", 0, nil),
		"version":   line(good, "not found", 127, nil),
		"setuid":    line(good, "Python 3.13.18", 0, []string{"/opt/toolchain/x"}),
		"bad rules": line(`{"name":"x","version_command":["x"],"proxy":"socks"}`, "", 0, nil),
	} {
		if _, err := parseProbe(logs, "Complete"); err == nil || !strings.HasPrefix(err.Error(), "TOOLCHAIN_CONTRACT:") {
			t.Errorf("%s: %v", name, err)
		}
	}
}
