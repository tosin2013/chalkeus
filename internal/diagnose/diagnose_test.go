package diagnose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func replay(t *testing.T, name, task string, deadline int) Diagnosis {
	t.Helper()
	return Diagnose(replayInput(t, name, task, deadline))
}

func replayInput(t *testing.T, name, task string, deadline int) Input {
	t.Helper()
	dir := filepath.Join("testdata", name)
	var rec map[string]any
	b, err := os.ReadFile(filepath.Join(dir, "attempt.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &rec); err != nil {
		t.Fatal(err)
	}
	in, err := FromDir(dir, task, rec, nil, "", 0, deadline)
	if err != nil {
		t.Fatal(err)
	}
	return in
}

func kinds(d Diagnosis) map[string][]string {
	out := map[string][]string{}
	for _, s := range d.Signals {
		out[s.Kind] = append(out[s.Kind], s.Detail)
	}
	return out
}

// SC-001: documcp run 20261125T163307Z, attempt 1 (US1, 2,400 s deadline).
func TestReplayDocumcpDeadline(t *testing.T) {
	d := replay(t, "documcp-001-1", "001", 2400)
	if d.Recommend != BackToPlanning {
		t.Fatalf("recommend = %s, want back_to_planning\n%s", d.Recommend, d.Text)
	}
	if !reflect.DeepEqual(d.Triggers, []string{EnvironmentNeed, UnknownWork}) {
		t.Errorf("triggers = %v", d.Triggers)
	}
	if !reflect.DeepEqual(d.Routes, []string{RoutePlanResearch, RoutePlanningPR}) {
		t.Errorf("routes = %v", d.Routes)
	}
	k := kinds(d)
	if len(k[KindResearchTask]) != 2 || !strings.Contains(k[KindResearchTask][0], "T003") || !strings.Contains(k[KindResearchTask][1], "T004a") {
		t.Errorf("research tasks = %v", k[KindResearchTask])
	}
	if len(k[KindManifestOutside]) != 1 || !strings.HasPrefix(k[KindManifestOutside][0], "package-lock.json") {
		t.Errorf("manifest outside = %v", k[KindManifestOutside])
	}
	if len(k[KindEgressRefused]) != 1 || k[KindEgressRefused][0] != "github.com refused 1× by the target proxy" {
		t.Errorf("egress = %v (the probes' refusals must be dropped)", k[KindEgressRefused])
	}
	for _, want := range []string{
		"diagnosis of attempt 1 (001): stopped at the 2400 s deadline, 277364 tokens, $0",
		"points to: an environment need, unknown work",
		"recommended: back to planning",
		"resolve T003, T004a in the plan's research (/speckit-plan), then /speckit-tasks",
		"change package-lock.json in your planning PR, not in an attempt",
	} {
		if !strings.Contains(d.Text, want) {
			t.Errorf("text lacks %q:\n%s", want, d.Text)
		}
	}
}

// SC-002: the fixture's induced first failure recommends retry, with no egress signal.
func TestReplayFixtureInduced(t *testing.T) {
	d := replay(t, "fixture-A-1", "A", 1200)
	if d.Recommend != Retry {
		t.Fatalf("recommend = %s, want retry\n%s", d.Recommend, d.Text)
	}
	k := kinds(d)
	if len(k[KindEgressRefused]) != 0 || len(d.Triggers) != 0 {
		t.Errorf("signals = %+v", d.Signals)
	}
	if len(k[KindFailedChecks]) != 1 || !strings.Contains(k[KindFailedChecks][0], "G5_hidden_tests") {
		t.Errorf("failed checks = %v", k[KindFailedChecks])
	}
}

// FR-006: same evidence, byte-identical diagnosis.
func TestDeterministic(t *testing.T) {
	a, _ := json.Marshal(replay(t, "documcp-001-1", "001", 2400))
	b, _ := json.Marshal(replay(t, "documcp-001-1", "001", 2400))
	if string(a) != string(b) {
		t.Fatal("two diagnoses of the same evidence differ")
	}
}

func TestRules(t *testing.T) {
	base := Input{Task: "001", Attempt: 1, Toolchain: "node24", AllowedPaths: []string{"src/*", "package-lock.json"}}
	cases := []struct {
		name string
		mod  func(*Input)
		want string
		trig []string
	}{
		{"deadline, low allowance", func(i *Input) { i.Reason = "deadline"; i.DeadlineS = 1200; i.TestSeconds = 300 }, RaiseBudget, []string{}},
		{"deadline, allowance unknown", func(i *Input) { i.Reason = "deadline"; i.DeadlineS = 1200 }, Retry, []string{}},
		{"deadline, ample allowance", func(i *Input) { i.Reason = "deadline"; i.DeadlineS = 2400; i.TestSeconds = 60 }, Retry, []string{}},
		{"unknown reason", func(i *Input) { i.Reason = "kaboom" }, BackToPlanning, []string{UnknownWork}},
		{"egress alone", func(i *Input) {
			i.Reason = "grade_fail"
			i.Denials = `[ns] NOTICE x: Proxying refused on filtered domain "pypi.org"`
		}, Retry, []string{}},
		{"egress and clone failure", func(i *Input) {
			i.Reason = "harness_error"
			i.ReportError = "clone_failed"
			i.Denials = `[ns] NOTICE x: Proxying refused on filtered domain "codeload.github.com"`
		}, BackToPlanning, []string{EnvironmentNeed}},
		{"lock file inside the allowed paths", func(i *Input) { i.Reason = "grade_fail"; i.ChangedPaths = []string{"package-lock.json", "src/a.ts"} }, Retry, []string{}},
		{"manifest outside", func(i *Input) { i.Reason = "grade_fail"; i.ChangedPaths = []string{"sub/package.json"} }, BackToPlanning, []string{EnvironmentNeed}},
		{"checked research task", func(i *Input) {
			i.Reason = "grade_fail"
			i.TaskList = "## Phase 1: x\n- [x] T001 Locate the bug\n"
		}, Retry, []string{}},
		{"research task in another slice", func(i *Input) {
			i.Reason = "grade_fail"
			i.TaskList = "## Phase 1: a\n- [ ] T001 Add it\n## Phase 2: b\n- [ ] T002 Investigate it\n"
			i.SlicePhases = []string{"Phase 1: a"}
		}, Retry, []string{}},
	}
	for _, c := range cases {
		in := base
		c.mod(&in)
		d := Diagnose(in)
		if d.Recommend != c.want || !reflect.DeepEqual(d.Triggers, c.trig) {
			t.Errorf("%s: recommend %s triggers %v, want %s %v\n%s", c.name, d.Recommend, d.Triggers, c.want, c.trig, d.Text)
		}
	}
}

func TestResearchWording(t *testing.T) {
	yes := []string{
		"[US1] Locate the suites that leave handles open",
		"[US1] Find the test that writes `.github/workflows/x.yml`",
		"Investigate why it hangs", "[P] [US2] Figure out the cause", "Determine why X",
	}
	no := []string{
		"[P] [US1] Set `.nvmrc` to `24`", "[US1] Remove `--forceExit` from `test:ci`",
		"Set X; find and replace Y", "Findings go in R3", "Explorer view: add a column",
	}
	for _, s := range yes {
		if _, ok := ResearchTask(s); !ok {
			t.Errorf("%q should read as research", s)
		}
	}
	for _, s := range no {
		if p, ok := ResearchTask(s); ok {
			t.Errorf("%q matched %q", s, p)
		}
	}
}

func TestGlobs(t *testing.T) {
	if !allowed("tests/memory/a.test.ts", []string{"tests/*"}) {
		t.Error("tests/* must match nested paths, as fnmatch does")
	}
	if allowed("package-lock.json", []string{"package.json"}) {
		t.Error("package.json must not match package-lock.json")
	}
}
