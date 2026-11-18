package worker

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tosin2013/specfoundry/internal/evidence"
	"github.com/tosin2013/specfoundry/internal/k8s"
	"github.com/tosin2013/specfoundry/internal/paths"
	"github.com/tosin2013/specfoundry/internal/prompt"
	"github.com/tosin2013/specfoundry/internal/toolchain"
	"sigs.k8s.io/yaml"
)

func TestMakeItemInputs(t *testing.T) {
	clone := t.TempDir()
	spec := "# Shout\n\n## Allowed paths\n- `app.py`\n"
	_ = os.MkdirAll(filepath.Join(clone, "specs/002-shout"), 0o755)
	_ = os.WriteFile(filepath.Join(clone, "specs/002-shout/spec.md"), []byte(spec), 0o644)
	_ = os.WriteFile(filepath.Join(clone, "specs/002-shout/tasks.md"), []byte("- [ ] T001 shout\n"), 0o644)
	rd := filepath.Join(t.TempDir(), "20990101T000000Z")
	_ = os.MkdirAll(filepath.Join(rd, "attempts"), 0o755)
	run := &evidence.Run{Dir: rd, ID: filepath.Base(rd), State: map[string]any{"kind": "item", "item": "002"}}
	d := Deps{L: paths.At(t.TempDir()), Item: &Item{SnapshotCommit: "snap123", SpecDir: "specs/002-shout",
		AllowedPaths: []string{"app.py", "specs/002-shout/tasks.md"}, Toolchain: "python", TestCommand: "python3 -m pytest -q", CommandTimeout: 180}}
	idir, err := MakeItemInputs(d, run, "002", 1, clone)
	if err != nil {
		t.Fatal(err)
	}
	adir, _ := run.AttemptDir("002", 1)
	var meta map[string]any
	raw, _ := os.ReadFile(filepath.Join(adir, "meta.json"))
	_ = json.Unmarshal(raw, &meta)
	if meta["base_revision"] != "snap123" || meta["item_id"] != "002" || meta["mode"] != "attempt" || meta["hidden_mode"] != "overlay" ||
		meta["test_command"] != "python3 -m pytest -q" || meta["induction"] != nil || meta["spec_sha256"] != prompt.SpecSHA256(spec) {
		t.Fatalf("%v", meta)
	}
	p, _ := os.ReadFile(filepath.Join(idir, "prompt.md"))
	if !strings.Contains(string(p), "  - `app.py`\n") || strings.Contains(string(p), "slugkit") {
		t.Fatal(string(p))
	}
	if _, err := os.Stat(filepath.Join(adir, "induction.json")); err == nil {
		t.Error("an item run has no induction")
	}
}

// With PILOT_TEST_PATHS, meta.json switches on the grader's G6/G7; without it, nothing changes.
func TestItemTestRulesInMeta(t *testing.T) {
	for _, withRules := range []bool{false, true} {
		clone := t.TempDir()
		_ = os.MkdirAll(filepath.Join(clone, "specs/002-shout"), 0o755)
		_ = os.WriteFile(filepath.Join(clone, "specs/002-shout/spec.md"), []byte("# Shout\n"), 0o644)
		_ = os.WriteFile(filepath.Join(clone, "specs/002-shout/tasks.md"), []byte("- [ ] T001 shout\n"), 0o644)
		rd := filepath.Join(t.TempDir(), "20990101T000000Z")
		_ = os.MkdirAll(filepath.Join(rd, "attempts"), 0o755)
		run := &evidence.Run{Dir: rd, ID: filepath.Base(rd), State: map[string]any{"kind": "item", "item": "002"}}
		it := &Item{SnapshotCommit: "s", SpecDir: "specs/002-shout", AllowedPaths: []string{"app.py", "tests/*"},
			Toolchain: "python", TestCommand: "pytest", CommandTimeout: 180}
		if withRules {
			it.TestPaths, it.TestsMayChange = []string{"tests/*"}, []string{"tests/test_old.py"}
		}
		idir, err := MakeItemInputs(Deps{L: paths.At(t.TempDir()), Item: it}, run, "002", 1, clone)
		if err != nil {
			t.Fatal(err)
		}
		adir, _ := run.AttemptDir("002", 1)
		var meta map[string]any
		raw, _ := os.ReadFile(filepath.Join(adir, "meta.json"))
		_ = json.Unmarshal(raw, &meta)
		p, _ := os.ReadFile(filepath.Join(idir, "prompt.md"))
		_, has := meta["test_paths"]
		if has != withRules || strings.Contains(string(p), "original versions") != withRules {
			t.Fatalf("rules=%t meta=%v\n%s", withRules, meta, p)
		}
		if withRules && (fmt.Sprint(meta["tests_may_change"]) != "[tests/test_old.py]" || !strings.Contains(string(p), "`tests/test_old.py`")) {
			t.Fatalf("%v\n%s", meta, p)
		}
	}
}

func TestItemWorkerJobRenders(t *testing.T) {
	root, _ := filepath.Abs("../..")
	render := func(b toolchain.Blocks, inits string) string {
		y, err := k8s.Render(filepath.Join(root, "pilot/manifests/worker-job-item.yaml.tmpl"), map[string]string{
			"NS": "pilot-w-002-1", "DEADLINE": "1200", "WORKER_IMAGE": "w@sha256:0", "TOOLS_IMAGE": "t@sha256:0", "PILOT_MODEL": "p/m",
			"SPEND_CAP": "1.00", "TOKEN_CAP": "2000000", "INPUT_LIMIT": "64Mi", "PROXY_URL": "http://proxy.pilot-egress-w-g.svc.cluster.local:8888",
			"HOME_LIMIT": "1Gi", "TOOLCHAIN_INITS": inits, "TOOLCHAIN_ENV": b.Env, "TOOLCHAIN_MOUNTS": b.Mounts, "TOOLCHAIN_VOLUMES": b.Volumes})
		if err != nil {
			t.Fatal(err)
		}
		var v any
		if err := yaml.Unmarshal([]byte(y), &v); err != nil {
			t.Fatalf("not YAML: %v\n%s", err, y)
		}
		return y
	}
	y := render(toolchain.Blocks{}, "")
	for _, want := range []string{"name: receiver", "image: t@sha256:0", "{name: inputs, emptyDir: {sizeLimit: 64Mi}}",
		"{name: inputs, mountPath: /in, readOnly: true}", "automountServiceAccountToken: false", "pilot-egress-w-g"} {
		if !strings.Contains(y, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(y, "configMap: {name: inputs}") {
		t.Error("item-run inputs must not come from a ConfigMap")
	}
	raw, _ := os.ReadFile(filepath.Join(root, "pilot/images/toolchain-python3.13/toolchain.json"))
	spec, err := toolchain.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	b := toolchain.PodBlocks("tc@sha256:1", &spec, WorkerDeps, false)
	y = render(b, b.Inits+"\n"+toolchain.SetupInit("t@sha256:0", "http://p:8888", &spec, WorkerDeps))
	for _, want := range []string{"name: toolchain", "image: tc@sha256:1", "name: setup", "sh /in/setup.sh",
		`{name: VIRTUAL_ENV, value: "/home/worker/deps/venv"}`, "{name: toolchain, mountPath: /opt/toolchain, readOnly: true}"} {
		if !strings.Contains(y, want) {
			t.Errorf("toolchain render lacks %q", want)
		}
	}
}
