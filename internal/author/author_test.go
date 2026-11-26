package author

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func ids(cs []Criterion) string {
	var out []string
	for _, c := range cs {
		out = append(out, c.ID)
	}
	return strings.Join(out, ",")
}

func TestCriteriaOfItem006(t *testing.T) {
	raw, err := os.ReadFile("testdata/spec-006.md")
	if err != nil {
		t.Fatal(err)
	}
	got := ids(Criteria(string(raw)))
	if got != "US1-AS1,US1-AS2,US1-AS3,US2-AS1,US2-AS2,FR-001,FR-002,FR-003,FR-004,FR-005" {
		t.Fatal(got)
	}
}

func TestCriteriaOf018(t *testing.T) {
	raw, _ := os.ReadFile("../../specs/018-test-author/spec.md")
	cs := Criteria(string(raw))
	got := ids(cs)
	for _, want := range []string{"US1-AS1", "US1-AS4", "US2-AS4", "US3-AS3", "FR-001", "FR-006a", "FR-013"} {
		if !strings.Contains(","+got+",", ","+want+",") {
			t.Errorf("missing %s in %s", want, got)
		}
	}
	if strings.Contains(got, "US0") || cs[0].Title == "" || strings.Contains(cs[0].Title, "**") {
		t.Fatalf("%+v", cs[0])
	}
}

func write(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, rel)
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func code(err error) string {
	if e, ok := err.(*Error); ok {
		return e.Code
	}
	return ""
}

func TestDraftFiles(t *testing.T) {
	d := t.TempDir()
	if _, _, err := DraftFiles(d); code(err) != CodeDraftEmpty {
		t.Fatalf("empty: %v", err)
	}
	write(t, d, MapFile, "{}")
	if _, _, err := DraftFiles(d); code(err) != CodeDraftEmpty {
		t.Fatal("the map alone is not a test")
	}
	write(t, d, "tests/test_b.py", "b")
	write(t, d, "tests/test_a.py", "a")
	files, h1, err := DraftFiles(d)
	if err != nil || len(files) != 2 || files[0].Path != "tests/test_a.py" || len(h1) != 64 {
		t.Fatalf("%v %v", files, err)
	}
	write(t, d, MapFile, `{"version":1}`)
	if _, h2, _ := DraftFiles(d); h2 == h1 {
		t.Fatal("the draft hash must change when the map changes")
	}
	if err := os.Symlink("/etc/passwd", filepath.Join(d, "tests", "link.py")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := DraftFiles(d); code(err) != CodeDraftOutsideFolder {
		t.Fatalf("symlink: %v", err)
	}
}

func TestCheckMap(t *testing.T) {
	cs := []Criterion{{ID: "US1-AS1"}, {ID: "US2-AS1"}, {ID: "FR-001"}}
	files := []File{{Path: "tests/test_x.py"}}
	d := t.TempDir()
	put := func(m Map) {
		raw, _ := json.Marshal(m)
		write(t, d, MapFile, string(raw))
	}
	if _, err := CheckMap(d, cs, "s", files); code(err) != CodeMapInvalid {
		t.Fatal("a missing map must be MAP_INVALID")
	}
	good := Map{Version: 1, SpecSHA256: "s", Criteria: map[string][]string{"US1-AS1": {"tests/test_x.py::test_a"}, "FR-001": {"tests/test_x.py"}},
		Uncovered: []string{"US2-AS1"}}
	put(good)
	sum, err := CheckMap(d, cs, "s", files)
	if err != nil || sum["covered"] != 2 || strings.Join(sum["uncovered"].([]string), ",") != "US2-AS1" {
		t.Fatalf("%v %v", sum, err)
	}
	for name, m := range map[string]Map{
		"stale spec":     {SpecSHA256: "old", Criteria: good.Criteria, Uncovered: good.Uncovered},
		"unknown id":     {SpecSHA256: "s", Criteria: map[string][]string{"US9-AS9": {"tests/test_x.py"}}, Uncovered: []string{"US1-AS1", "US2-AS1", "FR-001"}},
		"missing file":   {SpecSHA256: "s", Criteria: map[string][]string{"US1-AS1": {"tests/nope.py"}}, Uncovered: []string{"US2-AS1", "FR-001"}},
		"uncovered lies": {SpecSHA256: "s", Criteria: good.Criteria, Uncovered: nil},
		"empty refs":     {SpecSHA256: "s", Criteria: map[string][]string{"US1-AS1": {}}, Uncovered: []string{"US2-AS1", "FR-001"}},
	} {
		put(m)
		if _, err := CheckMap(d, cs, "s", files); code(err) != CodeMapInvalid {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestDraftEvidenceRoundTrip(t *testing.T) {
	root := t.TempDir()
	d, err := NewDraft(root)
	if err != nil {
		t.Fatal(err)
	}
	d.Target, d.Item, d.Files = "t", "006", []File{{Path: "tests/test_x.py", SHA256: "ab"}}
	if err := d.Save(); err != nil {
		t.Fatal(err)
	}
	_ = d.Event("started", "level=minimal")
	got, err := LoadDraft(root, "", "t", "006")
	if err != nil || got.ID != d.ID || got.Files[0].Path != "tests/test_x.py" {
		t.Fatalf("%+v %v", got, err)
	}
	if _, err := LoadDraft(root, "", "t", "007"); err == nil {
		t.Fatal("another item's draft must not be found")
	}
}

func TestExitCodes(t *testing.T) {
	if (&Error{Code: CodeDraftHiddenPasses}).ExitCode() != 1 || (&Error{Code: CodeCompanionNotEmpty}).ExitCode() != 10 {
		t.Fatal("results exit 1, refusals exit 10")
	}
}

// F12: a slice draft's map lists only its slice's criteria; check validates it against those
// (SliceCriteria), as start writes them, not against every criterion of the item.
func TestCheckMapSliceCriteria(t *testing.T) {
	all := []Criterion{{ID: "US1-AS1"}, {ID: "US2-AS1"}, {ID: "US2-AS2"}, {ID: "FR-001"}}
	files := []File{{Path: "tests/skill.test.ts"}}
	d := t.TempDir()
	raw, _ := json.Marshal(Map{Version: 1, SpecSHA256: "s",
		Criteria:  map[string][]string{"US2-AS1": {"tests/skill.test.ts"}, "US2-AS2": {"tests/skill.test.ts"}},
		Uncovered: []string{"FR-001"}})
	write(t, d, MapFile, string(raw))
	if _, err := CheckMap(d, SliceCriteria(all, "US2"), "s", files); err != nil {
		t.Fatalf("a US2 map is valid against US2's criteria: %v", err)
	}
	if _, err := CheckMap(d, all, "s", files); code(err) != CodeMapInvalid {
		t.Fatal("against every criterion it lacks US1-AS1: the F12 failure")
	}
}
