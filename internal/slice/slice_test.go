package slice

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func load(t *testing.T, name string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestFrom014(t *testing.T) {
	phases, ok := Parse(load(t, "phases-014.json"))
	if !ok || len(phases) != 7 {
		t.Fatalf("ok=%v phases=%d, want 7", ok, len(phases))
	}
	got, err := FromPhases(phases)
	if err != nil {
		t.Fatal(err)
	}
	if ids := IDs(got); !reflect.DeepEqual(ids, []string{"US2", "US1", "US3", "US4"}) {
		t.Fatalf("ids %v", ids)
	}
	prefixes := [][]string{
		{"Phase 1: Setup", "Phase 2: Foundational", "Phase 3: User Story 2"},
		{"Phase 4: User Story 1"},
		{"Phase 5: User Story 3"},
		{"Phase 6: User Story 4", "Phase 7: Polish"},
	}
	for i, s := range got {
		if s.Index != i || len(s.Phases) != len(prefixes[i]) {
			t.Fatalf("slice %s: index %d phases %v", s.ID, s.Index, s.Phases)
		}
		for j, p := range prefixes[i] {
			if !strings.HasPrefix(s.Phases[j], p) {
				t.Errorf("slice %s phase %d = %q, want prefix %q", s.ID, j, s.Phases[j], p)
			}
		}
		if State(s) != Merged {
			t.Errorf("014 is all ticked; %s is %s", s.ID, State(s))
		}
	}
	if got[0].Total != 3+9+5 {
		t.Errorf("US2 total %d", got[0].Total)
	}
	if _, ok := Current(got); ok {
		t.Error("no current slice expected when all are merged")
	}
	if !Last(got, got[3]) || Last(got, got[0]) {
		t.Error("Last")
	}
	if e := Earlier(got, got[2]); !reflect.DeepEqual(IDs(e), []string{"US2", "US1"}) {
		t.Errorf("Earlier %v", IDs(e))
	}
}

func TestParseCases(t *testing.T) {
	if _, ok := Parse(load(t, "phases-null.json")); ok {
		t.Error("null phases must be unevaluable")
	}
	if _, ok := Parse(load(t, "phases-absent.json")); ok {
		t.Error("absent phases must be unevaluable")
	}
	phases, ok := Parse(load(t, "phases-none.json"))
	if !ok || len(phases) != 0 {
		t.Fatalf("[] phases: ok=%v n=%d", ok, len(phases))
	}
	got, err := FromPhases(phases)
	if err != nil || len(got) != 1 || got[0].ID != All || Sliced(got) {
		t.Fatalf("[] gives one ALL slice, got %+v %v", got, err)
	}
}

func TestDuplicateStory(t *testing.T) {
	phases, _ := Parse(load(t, "phases-dup-story.json"))
	_, err := FromPhases(phases)
	if err == nil || !strings.Contains(err.Error(), "User Story 2") || !strings.Contains(err.Error(), "User Story 3") {
		t.Fatalf("want an error naming both headings, got %v", err)
	}
}

func TestStatesAndFind(t *testing.T) {
	two, three := 1, 2
	ph := []Phase{
		{Heading: "Phase 1: Setup", Checked: 2, Total: 2},
		{Heading: "Phase 2: User Story 1", Story: &two, Checked: 3, Total: 3},
		{Heading: "Phase 3: User Story 2", Story: &three, Checked: 1, Total: 4},
		{Heading: "Phase 4: Polish", Checked: 0, Total: 2},
	}
	got, _ := FromPhases(ph)
	if State(got[0]) != Merged || State(got[1]) != Started {
		t.Fatalf("states %s %s", State(got[0]), State(got[1]))
	}
	if c, ok := Current(got); !ok || c.ID != "US2" {
		t.Fatalf("current %v", c)
	}
	if s, ok := Find(got, "us2"); !ok || s.ID != "US2" {
		t.Error("find us2")
	}
	for _, bad := range []string{"", "ALL", "all", "US9"} {
		if _, ok := Find(got, bad); ok {
			t.Errorf("Find(%q) must fail", bad)
		}
	}
	none := []Slice{{ID: "US1", Checked: 0, Total: 3}}
	if State(none[0]) != Untouched {
		t.Error("untouched")
	}
}
