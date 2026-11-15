package gate

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// Port of pilot/tests/test_gate.py (the fail-closed matrix, GM01's unit part).

func fx(t *testing.T, name string) string {
	raw, err := Fixture(name)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestRunMatrix(t *testing.T) {
	if ok, detail := RunMatrix(); !ok || detail != "OK" {
		t.Fatal(detail)
	}
}

func ip(i int) *int { return &i }

func TestMatrix(t *testing.T) {
	cases := []struct {
		file, item, want string
		reasons          []string
	}{
		{"ready_A.json", "001", "allow", []string{"OK"}},
		{"blocked_B.json", "002", "refuse", []string{"DEP_UNSATISFIED:001"}},
		{"ready_B_after_verify.json", "002", "allow", []string{"OK"}},
		{"schema_1_1.json", "001", "refuse", []string{"SCHEMA_MISMATCH"}},
		{"incomplete.json", "001", "refuse", []string{"INCOMPLETE"}},
		{"error_set.json", "001", "refuse", []string{"UNEVALUABLE:tool_error"}},
		{"item_missing.json", "001", "refuse", []string{"ITEM_MISSING"}},
		{"item_duplicate.json", "001", "refuse", []string{"ITEM_DUPLICATE"}},
		{"status_unknown.json", "001", "refuse", []string{"STATUS_UNKNOWN"}},
		{"inactive.json", "001", "refuse", []string{"INACTIVE"}},
		{"already_verified.json", "001", "refuse", []string{"ALREADY_VERIFIED"}},
		{"already_implemented.json", "001", "refuse", []string{"ALREADY_IMPLEMENTED"}},
		{"item_error_diag.json", "001", "refuse", []string{"ITEM_ERROR_DIAGNOSTIC:SS_TASK_FILE_MISSING"}},
		{"inconsistent_count.json", "002", "refuse", []string{"DEP_UNSATISFIED:001", "INCONSISTENT_BLOCKED_COUNT"}},
		{"not_startable.json", "001", "refuse", []string{"STATUS_NOT_STARTABLE:deferred"}},
		{"malformed.txt", "001", "refuse", []string{"UNEVALUABLE:bad_json"}},
	}
	for _, c := range cases {
		d, r := Decide(c.item, ip(0), fx(t, c.file))
		if d != c.want || !reflect.DeepEqual(r, c.reasons) {
			t.Errorf("%s: got %s %v, want %s %v", c.file, d, r, c.want, c.reasons)
		}
	}
}

func TestExitCodesAndShapes(t *testing.T) {
	check := func(name string, exit *int, out string, want []string) {
		d, r := Decide("001", exit, out)
		if d != "refuse" || !reflect.DeepEqual(r, want) {
			t.Errorf("%s: %s %v, want refuse %v", name, d, r, want)
		}
	}
	ready := fx(t, "ready_A.json")
	check("exit 2 with ready json", ip(2), ready, []string{"UNEVALUABLE:exit_2"})
	check("exit 127", ip(127), "", []string{"UNEVALUABLE:exit_127"})
	check("spawn failed", nil, "", []string{"UNEVALUABLE:spawn_failed"})
	check("empty", ip(0), "", []string{"UNEVALUABLE:empty"})
	check("non-object", ip(0), "[]", []string{"SCHEMA_MISMATCH"})
	if d, r := Decide("001", ip(1), ready); d != "allow" || r[0] != "OK" {
		t.Errorf("exit 1 should be evaluable: %s %v", d, r)
	}
	var doc map[string]any
	_ = json.Unmarshal([]byte(ready), &doc)
	doc["items"].([]any)[0].(map[string]any)["dependencies"] = []any{map[string]any{"id": "x"}}
	raw, _ := json.Marshal(doc)
	if d, _ := Decide("001", ip(0), string(raw)); d != "refuse" {
		t.Error("dependency without satisfied must refuse")
	}
}

// validate is a minimal check against contracts/gate-decision.schema.json.
func validate(t *testing.T, d Decision) {
	t.Helper()
	raw, _ := os.ReadFile("../../specs/002-m1-gated-pilot/contracts/gate-decision.schema.json")
	var schema struct {
		Required   []string `json:"required"`
		Properties struct {
			Reasons struct {
				Items struct {
					Pattern string `json:"pattern"`
				} `json:"items"`
			} `json:"reasons"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatal(err)
	}
	enc, _ := json.Marshal(d)
	var m map[string]any
	_ = json.Unmarshal(enc, &m)
	for _, k := range schema.Required {
		if _, ok := m[k]; !ok {
			t.Errorf("missing %s", k)
		}
	}
	pat := regexp.MustCompile(schema.Properties.Reasons.Items.Pattern)
	for _, r := range d.Reasons {
		if !pat.MatchString(r) {
			t.Errorf("bad reason %s", r)
		}
	}
	if (d.Decision == "allow") != (len(d.Reasons) == 1 && d.Reasons[0] == "OK") {
		t.Errorf("allow must mean reasons == [OK]: %v", d)
	}
}

func copyFixture(t *testing.T) string {
	t.Helper()
	dst := t.TempDir()
	if out, err := exec.Command("cp", "-r", "../../pilot/fixture/.", dst).CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	return dst
}

func needSpec(t *testing.T) {
	if _, err := exec.LookPath(ExecStatus{}.bin()); err != nil {
		t.Skip("spec binary not installed")
	}
}

func TestRealSpecStatusOnFixture(t *testing.T) {
	needSpec(t)
	root, out := copyFixture(t), t.TempDir()
	a, err := Evaluate(ExecStatus{}, "A", "001", root, 1, out, "local", nil)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := Evaluate(ExecStatus{}, "B", "002", root, 2, out, "local", nil)
	if a.Decision != "allow" || b.Decision != "refuse" || !reflect.DeepEqual(b.Reasons, []string{"DEP_UNSATISFIED:001"}) {
		t.Fatalf("A %v / B %v", a, b)
	}
	validate(t, a)
	validate(t, b)
	for _, f := range []string{"1-A.json", "2-B.status.json"} {
		if _, err := os.Stat(filepath.Join(out, f)); err != nil {
			t.Error(err)
		}
	}
}

func TestFailClosedVariants(t *testing.T) {
	needSpec(t)
	d, _ := Evaluate(ExecStatus{Bin: "/nonexistent/spec"}, "A", "001", copyFixture(t), 1, "", "local", nil)
	if d.Decision != "refuse" || d.Reasons[0] != "UNEVALUABLE:spawn_failed" {
		t.Errorf("spawn failure: %v", d)
	}
	validate(t, d)
	if d, _ := Evaluate(ExecStatus{Bin: "/bin/false"}, "A", "001", copyFixture(t), 1, "", "local", nil); d.Decision != "refuse" {
		t.Errorf("tool failure: %v", d)
	}
	if d, _ := Evaluate(ExecStatus{}, "A", "001", t.TempDir(), 1, "", "local", nil); d.Decision != "refuse" {
		t.Errorf("missing roadmap: %v", d)
	}
	root := copyFixture(t)
	rm := filepath.Join(root, ".specify/memory/roadmap.md")
	raw, _ := os.ReadFile(rm)
	text := string(raw)
	_ = os.WriteFile(rm, []byte(text[:strings.Index(text, "### 002")]+"### 002 — Unique sl"), 0o644)
	if d, _ := Evaluate(ExecStatus{}, "B", "002", root, 1, "", "local", nil); d.Decision != "refuse" {
		t.Errorf("truncated roadmap: %v", d)
	}
}
