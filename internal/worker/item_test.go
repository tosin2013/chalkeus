package worker

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tosin2013/specfoundry/internal/evidence"
	"github.com/tosin2013/specfoundry/internal/k8s"
	"github.com/tosin2013/specfoundry/internal/paths"
	"github.com/tosin2013/specfoundry/internal/prompt"
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

func TestItemWorkerJobRenders(t *testing.T) {
	root, _ := filepath.Abs("../..")
	y, err := k8s.Render(filepath.Join(root, "pilot/manifests/worker-job-item.yaml.tmpl"), map[string]string{
		"NS": "pilot-w-002-1", "DEADLINE": "1200", "WORKER_IMAGE": "w@sha256:0", "TOOLS_IMAGE": "t@sha256:0", "PILOT_MODEL": "p/m",
		"SPEND_CAP": "1.00", "TOKEN_CAP": "2000000", "INPUT_LIMIT": "64Mi"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"name: receiver", "image: t@sha256:0", "{name: inputs, emptyDir: {sizeLimit: 64Mi}}",
		"{name: inputs, mountPath: /in, readOnly: true}", "automountServiceAccountToken: false"} {
		if !strings.Contains(y, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(y, "configMap: {name: inputs}") {
		t.Error("item-run inputs must not come from a ConfigMap")
	}
}
