package doctor

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func put(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, rel)
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

const statusDoc = `{"items": [
 {"id": "001", "title": "Core", "status": "verified", "active": true, "spec_dir": "specs/001-core", "tasks": {"checked": 13, "total": 13}, "dependencies": []},
 {"id": "003", "title": "OCI", "status": "implemented", "active": true, "spec_dir": "specs/003-oci", "tasks": {"checked": 0, "total": 12}, "dependencies": []},
 {"id": "006", "title": "Bundled flags", "status": "specced", "active": true, "spec_dir": "specs/006-flags", "tasks": {"checked": 0, "total": 5}, "dependencies": [{"id": "001", "satisfied": true}]},
 {"id": "007", "title": "No tests yet", "status": "specced", "active": true, "spec_dir": "specs/007-x", "tasks": {"checked": 0, "total": 1}, "dependencies": []},
 {"id": "004", "title": "Lightspeed", "status": "needs-info", "active": true, "spec_dir": "", "tasks": null, "dependencies": [{"id": "003", "satisfied": false}]},
 {"id": "005", "title": "Community", "status": "planned", "active": true, "spec_dir": "", "tasks": null, "dependencies": [{"id": "003", "satisfied": false}]},
 {"id": "009", "title": "Gone", "status": "abandoned", "active": false}
], "diagnostics": [
 {"code": "SS_STAGE_FILE_MISSING", "severity": "error", "message": "Stage-required file plan.md is absent.", "item_id": "001"},
 {"code": "SS_DEPENDENCY_BLOCKED", "severity": "warning", "message": "blocked", "item_id": "005"},
 {"code": "SS_INFO_THING", "severity": "info", "message": "fyi", "item_id": "006"}
]}`

func TestRoadmapRows(t *testing.T) {
	target, comp := t.TempDir(), t.TempDir()
	allowed := "# x\n\n## Allowed paths\n\n- `src/*`\n"
	put(t, target, "specs/006-flags/spec.md", allowed)
	put(t, target, "specs/006-flags/tasks.md", "- [ ] T001\n")
	put(t, target, "specs/007-x/spec.md", allowed)
	put(t, target, "specs/007-x/tasks.md", "- [ ] T001\n")
	put(t, comp, "items/006/tests/test_a.py", "x")
	put(t, comp, "items/007/.acceptance-map.json", "{}") // a map alone isn't a test
	var r Report
	r.Roadmap(statusDoc, target, func(item string) string { return filepath.Join(comp, "items", item) })
	got := map[string]Row{}
	for _, x := range r.Rows {
		got[strings.SplitN(x.Check, " ", 2)[0]] = x
	}
	want := map[string][2]string{
		"progress": {Info, "1 of 6 active items verified (16%)"},
		"001":      {Info, "done"},
		"003":      {Info, "implemented; verifying it is the owner's call"},
		"006":      {Pass, "ready to run"},
		"007":      {Info, "hidden tests"},
		"004":      {Info, "not startable while needs-info"},
		"005":      {Info, "dependencies verified first (003); a spec"},
	}
	for id, w := range want {
		x, ok := got[id]
		if !ok || x.Status != w[0] || !strings.Contains(x.Detail, w[1]) {
			t.Errorf("%s: got %+v, want %s %q", id, x, w[0], w[1])
		}
	}
	diag := 0
	for _, x := range r.Rows {
		if strings.HasPrefix(x.Check, "spec status ") {
			diag++
			if x.Status != Warn || !strings.Contains(x.Detail, "SS_STAGE_FILE_MISSING") {
				t.Errorf("%+v", x)
			}
		}
	}
	if diag != 1 {
		t.Errorf("only spec status's errors and warnings, minus dependency warnings: %d rows", diag)
	}
	if _, ok := got["009"]; ok {
		t.Error("inactive items are not listed")
	}
	if r.Failed() {
		t.Fatal("the roadmap section never fails doctor")
	}
	var b bytes.Buffer
	r.Print(&b)
	if !strings.Contains(b.String(), "== roadmap") || !strings.Contains(b.String(), "INFO") {
		t.Fatal(b.String())
	}
	var bad Report
	bad.Roadmap("not json", target, func(string) string { return "" })
	if len(bad.Rows) != 1 || bad.Rows[0].Status != Warn {
		t.Fatalf("%+v", bad.Rows)
	}
}

func TestRoadmapUnreadableCompanion(t *testing.T) {
	target := t.TempDir()
	put(t, target, "specs/006-flags/spec.md", "# x\n\n## Allowed paths\n\n- `src/*`\n")
	put(t, target, "specs/006-flags/tasks.md", "- [ ] T001\n")
	var r Report
	r.Roadmap(`{"items":[{"id":"006","title":"t","status":"specced","active":true,"spec_dir":"specs/006-flags","dependencies":[]}]}`, target,
		func(string) string { return "" })
	if d := r.Rows[len(r.Rows)-1].Detail; !strings.Contains(d, "a readable companion") {
		t.Fatal(d)
	}
}
