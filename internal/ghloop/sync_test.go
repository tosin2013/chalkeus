package ghloop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/tosin2013/chalkeus/internal/evidence"
	"github.com/tosin2013/chalkeus/internal/pyrepr"
)

// Cases ported from the former Python pilot's tests/test_ghsync.py (004 US1: roadmap -> item issues, labels,
// checklists; idempotent sync; drift).

const docJSON = `{"items": [
 {"id": "001", "title": "Slug function", "status": "planned", "active": true, "description": "A slugify.",
  "outcome": "It works.", "spec_dir": "specs/001-slugify", "tasks": {"source": "specs/001-slugify/tasks.md"}},
 {"id": "002", "title": "Unique slugs", "status": "planned", "active": true, "description": "Unique.",
  "outcome": "No collisions.", "spec_dir": "specs/002-unique-slug", "tasks": {"source": "specs/002-unique-slug/tasks.md"}},
 {"id": "000", "title": "Old", "status": "abandoned", "active": false}]}`

func doc(t *testing.T, over map[string]any) map[string]any {
	t.Helper()
	var d map[string]any
	if err := evidence.Decode([]byte(docJSON), &d); err != nil {
		t.Fatal(err)
	}
	for k, v := range over {
		d["items"].([]any)[0].(map[string]any)[k] = v
	}
	return d
}

func texts() map[string]*string {
	s := "# Tasks\n\n- [x] T001 Implement slugify\n- [ ] T002 Export it\nnot a task\n"
	return map[string]*string{"001": &s, "002": nil}
}

func snap(t *testing.T) func() (map[string]any, map[string]*string, error) {
	return func() (map[string]any, map[string]*string, error) { return doc(t, nil), texts(), nil }
}

func TestBodyHasMarkerStatusAndChecklist(t *testing.T) {
	d := DesiredState(doc(t, nil), texts())
	if len(d) != 2 || d[0].Item != "001" || d[1].Item != "002" {
		t.Fatalf("inactive items are history: %+v", d)
	}
	b := d[0].Body
	for _, want := range []string{"status `planned`", "- [x] T001 Implement slugify", "- [ ] T002 Export it"} {
		if !strings.Contains(b, want) {
			t.Errorf("body lacks %q", want)
		}
	}
	if !strings.HasPrefix(b, "<!-- specfoundry:item=001 -->") || strings.Contains(b, "not a task") {
		t.Errorf("body: %q", b)
	}
	if d[0].Label != "status:planned" || strings.Contains(d[1].Body, "### Tasks") {
		t.Errorf("label %q / body2 %q", d[0].Label, d[1].Body)
	}
}

// The exact body the reference writes (ghsync.issue_body), so a switch of implementation does
// not rewrite every issue.
func TestBodyMatchesReference(t *testing.T) {
	got := DesiredState(doc(t, nil), texts())[0].Body
	want := "<!-- specfoundry:item=001 -->\n**001 — Slug function** · status `planned` · spec `specs/001-slugify`\n\nA slugify.\n\n" +
		"**Outcome:** It works.\n\n### Tasks (from specs/001-slugify/tasks.md)\n- [x] T001 Implement slugify\n- [ ] T002 Export it\n\n" +
		"_Generated from the roadmap by specfoundry. Edits here are overwritten by `./chalkeus forge github sync`._\n"
	if got != want {
		t.Fatalf("got\n%q\nwant\n%q", got, want)
	}
}

func TestUnknownStatusMapsToUndecided(t *testing.T) {
	d := doc(t, map[string]any{"status": "done"})
	if l := DesiredState(d, nil)[0].Label; l != "status:undecided" {
		t.Fatal(l)
	}
}

func TestIssuesMatchedByMarkerNotTitle(t *testing.T) {
	issues := []Issue{{Number: 5, Title: "001 — Slug function", Body: "no marker", State: "open"}}
	n := 0
	for _, op := range PlanOps(DesiredState(doc(t, nil), texts()), issues, nil) {
		if op.Kind == "create-issue" {
			n++
		}
	}
	if n != 2 {
		t.Fatal(n)
	}
}

func TestFirstSyncCreatesIssuesAndLabels(t *testing.T) {
	f := newFake("pub-bot", nil)
	l := f.loop()
	l.Snapshot = snap(t)
	s, err := l.Sync(nil)
	if err != nil {
		t.Fatal(err)
	}
	if s.Created != 2 || s.Changed != 0 || pyrepr.Value(s.Issues) != "{'001': 1, '002': 2}" {
		t.Fatalf("%+v %s", s, pyrepr.Value(s.Issues))
	}
	if !f.labels["status:verified"] {
		t.Error("labels not ensured")
	}
	for _, i := range f.issues {
		if !reflect.DeepEqual(i.Labels, []string{"status:planned"}) {
			t.Errorf("labels %v", i.Labels)
		}
	}
}

func TestSecondSyncIsANoOp(t *testing.T) {
	f := newFake("pub-bot", nil)
	l := f.loop()
	l.Snapshot = snap(t)
	_, _ = l.Sync(nil)
	n := len(f.ops)
	s, _ := l.Sync(nil)
	if s.Created != 0 || s.Changed != 0 || len(f.ops) != n {
		t.Fatalf("%+v ops %d -> %d", s, n, len(f.ops))
	}
}

func TestHandEditedLabelRestoredOtherLabelsKept(t *testing.T) {
	f := newFake("pub-bot", nil)
	l := f.loop()
	l.Snapshot = snap(t)
	_, _ = l.Sync(nil)
	f.issues[0].Labels = []string{"status:verified", "good first issue"}
	if rep := DriftReport(DesiredState(doc(t, nil), texts()), f.issues, nil, nil); rep[0].Get("kind") != "label_mismatch" {
		t.Fatal(rep)
	}
	s, _ := l.Sync(nil)
	got := append([]string(nil), f.issues[0].Labels...)
	sort.Strings(got)
	if s.Changed != 1 || !reflect.DeepEqual(got, []string{"good first issue", "status:planned"}) {
		t.Fatalf("%+v %v", s, got)
	}
}

func TestSyncNeverClosesOrDeletes(t *testing.T) {
	f := newFake("pub-bot", nil)
	l := f.loop()
	l.Snapshot = snap(t)
	_, _ = l.Sync(nil)
	f.issues = append(f.issues, Issue{Number: 99, Title: "x", Body: "<!-- specfoundry:item=777 -->", State: "open"})
	_, _ = l.Sync(nil)
	if f.count("close-issue") != 0 {
		t.Fatal("closed an issue")
	}
}

func TestSyncGitHubErrorFails(t *testing.T) {
	f := newFake("pub-bot", nil)
	f.fail = "read"
	l := f.loop()
	l.Snapshot = snap(t)
	if _, err := l.Sync(nil); err == nil {
		t.Fatal("no error")
	}
}

func testRun(t *testing.T, id string, state map[string]any) *evidence.Run {
	t.Helper()
	dir := filepath.Join(t.TempDir(), id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if state == nil {
		state = map[string]any{}
	}
	state["run_id"] = id
	raw, _ := json.Marshal(state)
	if err := os.WriteFile(filepath.Join(dir, "run.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := evidence.OpenRun(dir)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestSyncRecordsEvidenceWithRun(t *testing.T) {
	f := newFake("pub-bot", nil)
	l := f.loop()
	l.Snapshot = snap(t)
	run := testRun(t, "20261116T101500Z", nil)
	if _, err := l.Sync(run); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(run.Dir, "github", "issues.json"))
	var got map[string]any
	_ = json.Unmarshal(raw, &got)
	if !reflect.DeepEqual(got["issues"], map[string]any{"001": 1.0, "002": 2.0}) {
		t.Fatal(got)
	}
	ev, _ := run.Events()
	if len(ev) != 1 || ev[0]["event"] != "github_sync" || ev[0]["detail"] != "created=2 changed=0" {
		t.Fatal(ev)
	}
	if IssueFor(run, "A") != 1 || IssueFor(run, "B") != 2 {
		t.Fatal("IssueFor")
	}
}

func driftIssues(state1, label1 string) []Issue {
	return []Issue{{Number: 1, Body: "<!-- specfoundry:item=001 -->", Labels: []string{label1}, State: state1},
		{Number: 2, Body: "<!-- specfoundry:item=002 -->", Labels: []string{"status:planned"}, State: "open"}}
}

func kinds(rep []pyrepr.OrderedMap) []string {
	var out []string
	for _, r := range rep {
		out = append(out, r.Get("kind").(string))
	}
	sort.Strings(out)
	return out
}

func TestDrift(t *testing.T) {
	d := DesiredState(doc(t, nil), texts())
	if rep := DriftReport(d, driftIssues("open", "status:planned"), nil, nil); len(rep) != 0 {
		t.Fatal(rep)
	}
	iss := append(driftIssues("open", "status:planned")[:1],
		Issue{Number: 3, Body: "<!-- specfoundry:item=009 -->", State: "open"},
		Issue{Number: 4, Body: "<!-- specfoundry:item=001 -->", State: "open"})
	if k := kinds(DriftReport(d, iss, nil, nil)); !reflect.DeepEqual(k, []string{"duplicate_issue", "missing_issue", "orphan_issue"}) {
		t.Fatal(k)
	}
	ver := DesiredState(doc(t, map[string]any{"status": "verified"}), texts())
	if k := kinds(DriftReport(ver, driftIssues("open", "status:verified"), nil, nil)); !reflect.DeepEqual(k, []string{"open_but_verified"}) {
		t.Fatal(k)
	}
	if k := kinds(DriftReport(d, driftIssues("closed", "status:planned"), nil, nil)); !reflect.DeepEqual(k, []string{"closed_but_not_verified"}) {
		t.Fatal(k)
	}
}

func TestDriftReprMatchesReference(t *testing.T) {
	d := DesiredState(doc(t, nil), texts())
	rep := DriftReport(d, driftIssues("open", "status:verified"), nil, nil)
	want := "[{'kind': 'label_mismatch', 'item': '001', 'issue': 1, 'want': 'status:planned', 'have': ['status:verified']}]"
	if got := pyrepr.Value(rep); got != want {
		t.Fatalf("got %s", got)
	}
	if got := pyrepr.SortedJSON(rep[0].Values); got != `{"have": ["status:verified"], "issue": 1, "item": "001", "kind": "label_mismatch", "want": "status:planned"}` {
		t.Fatalf("json %s", got)
	}
}

func TestTaskPRMustCloseItsIssue(t *testing.T) {
	d := DesiredState(doc(t, nil), texts())
	prs := []map[string]any{{"task": "A", "number": json.Number("7"), "kind": "task", "closes": nil},
		{"task": "A", "number": json.Number("8"), "kind": "status"}}
	nums := map[string]any{"001": json.Number("1")}
	if k := kinds(DriftReport(d, driftIssues("open", "status:planned"), prs, nums)); !reflect.DeepEqual(k, []string{"pr_without_closes"}) {
		t.Fatal(k)
	}
	prs[0]["closes"] = json.Number("1")
	if rep := DriftReport(d, driftIssues("open", "status:planned"), prs, nums); len(rep) != 0 {
		t.Fatal(rep)
	}
}

// 014: an item run's task label is the item id, for both the PR-closes drift check and IssueFor.
func TestItemRunMapsByItemID(t *testing.T) {
	d := DesiredState(doc(t, nil), texts())
	prs := []map[string]any{{"task": "002", "number": json.Number("9"), "kind": "task", "closes": nil}}
	nums := map[string]any{"002": json.Number("2")}
	if k := kinds(DriftReport(d, driftIssues("open", "status:planned"), prs, nums)); !reflect.DeepEqual(k, []string{"pr_without_closes"}) {
		t.Fatal(k)
	}
	run := testRun(t, "20261117T101500Z", map[string]any{"kind": "item", "item": "002"})
	_ = os.MkdirAll(filepath.Join(run.Dir, "github"), 0o755)
	_ = os.WriteFile(filepath.Join(run.Dir, "github", "issues.json"), []byte(`{"issues": {"001": 1, "002": 2}}`), 0o644)
	if IssueFor(run, "002") != 2 {
		t.Fatal("IssueFor by item id")
	}
}
