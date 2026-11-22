package readiness

import (
	"strings"
	"testing"
)

// 026 readiness for slices (contracts/slices.md). Item 002 gets three phases: Setup and User Story 1
// (slice US1), and User Story 2 (slice US2). ticks sets each phase's checked count.
func slicedWorld(setup, us1, us2 int) *world {
	w := newWorld()
	phases := []any{
		map[string]any{"heading": "Phase 1: Setup", "line": 3, "story": nil, "checked": setup, "total": 1},
		map[string]any{"heading": "Phase 2: User Story 1 - a", "line": 7, "story": 1, "checked": us1, "total": 2},
		map[string]any{"heading": "Phase 3: User Story 2 - b", "line": 12, "story": 2, "checked": us2, "total": 2},
	}
	setPhases(w, phases)
	w.companion = map[string]string{
		"items/002/US1/src/test/java/A1Test.java": "class A1Test {}\n",
		"items/002/US2/src/test/java/A2Test.java": "class A2Test {}\n",
	}
	w.target["specs/002-feature/spec.md"] = goodSpec + "\n## Allowed paths (US2)\n\n- `src/main/java/b/*`\n\n## Item acceptance\n\n- a human checks the stock page\n"
	return w
}

func setPhases(w *world, phases any) {
	for _, x := range w.status.doc["items"].([]any) {
		m := x.(map[string]any)
		if m["id"] == "002" {
			tasks := map[string]any{"checked": 0, "total": 5, "percent": 0, "state": "read"}
			if phases != "absent" {
				tasks["phases"] = phases
			}
			m["tasks"] = tasks
		}
	}
}

func codes(res Result) string { return strings.Join(res.Reasons, ",") }

func detail(res Result, code string) string {
	for _, r := range res.Rules {
		if r.ID == code {
			return r.Detail
		}
	}
	return ""
}

func TestSliceUS1OfAFreshItemAllows(t *testing.T) {
	w := slicedWorld(0, 0, 0)
	w.slice = "us1"
	res, _ := w.check(t, "002", 50)
	if res.Decision != "allow" {
		t.Fatalf("%s %+v", codes(res), res.Rules)
	}
	s := res.Slice
	if s == nil || s.ID != "US1" || s.Index != 0 || s.Of != 2 || len(s.Earlier) != 0 || s.AllowedPathsSource != "item" ||
		s.DefaultRevision == "" || res.SlicesLast {
		t.Fatalf("slice info %+v last=%v", s, res.SlicesLast)
	}
	if strings.Join(res.Overlay, ",") != "items/002/US1" || len(res.Earlier) != 0 || res.HiddenFiles != 1 {
		t.Fatalf("overlay %v earlier %v hidden %d", res.Overlay, res.Earlier, res.HiddenFiles)
	}
	if len(res.Acceptance) != 1 || res.Acceptance[0] != "a human checks the stock page" {
		t.Fatalf("acceptance %v", res.Acceptance)
	}
}

func TestSliceUS2AfterUS1MergedAllows(t *testing.T) {
	w := slicedWorld(1, 2, 0)
	w.slice = "US2"
	res, _ := w.check(t, "002", 50)
	if res.Decision != "allow" {
		t.Fatalf("%s %+v", codes(res), res.Rules)
	}
	if !res.SlicesLast || res.Slice.AllowedPathsSource != "slice" || strings.Join(res.Slice.Earlier, ",") != "US1" ||
		strings.Join(res.AllowedPaths, ",") != "src/main/java/b/*,specs/002-feature/tasks.md" {
		t.Fatalf("%+v %v", res.Slice, res.AllowedPaths)
	}
	if strings.Join(res.Overlay, ",") != "items/002/US2" || strings.Join(res.Earlier, ",") != "items/002/US1" {
		t.Fatalf("overlay %v earlier %v", res.Overlay, res.Earlier)
	}
}

// Each slice rule, broken alone, refuses with its own code.
func TestSliceRulesRefuse(t *testing.T) {
	cases := []struct {
		name, slice, code, want string
		w                       func() *world
	}{
		{"no phases (old spec-status)", "US1", "SLICE_UNEVALUABLE", "0.4.0", func() *world { w := slicedWorld(0, 0, 0); setPhases(w, "absent"); return w }},
		{"null phases", "US1", "SLICE_UNEVALUABLE", "tasks.phases", func() *world { w := slicedWorld(0, 0, 0); setPhases(w, nil); return w }},
		{"unknown id", "US7", "SLICE_UNKNOWN", "US1, US2", slicedWorld0},
		{"ALL is not asked for", "ALL", "SLICE_UNKNOWN", "without --slice", slicedWorld0},
		{"repeated story", "US1", "SLICE_UNKNOWN", "story 1 appears in two phases", func() *world {
			w := slicedWorld(0, 0, 0)
			setPhases(w, []any{
				map[string]any{"heading": "Phase 1: User Story 1 - a", "line": 1, "story": 1, "checked": 0, "total": 1},
				map[string]any{"heading": "Phase 2: User Story 1 - again", "line": 5, "story": 1, "checked": 0, "total": 1},
			})
			return w
		}},
		{"earlier not merged", "US2", "SLICE_EARLIER_OPEN", "earlier slice(s) US1", func() *world { return slicedWorld(1, 1, 0) }},
		{"earlier PR still open", "US2", "SLICE_EARLIER_OPEN", "US1", func() *world {
			w := slicedWorld(1, 2, 0)
			w.heads = []string{"pilot/20261122T000000Z/002-us1"}
			return w
		}},
		{"slice already started", "US1", "SLICE_STARTED", "1 of 3", func() *world { return slicedWorld(1, 0, 0) }},
		{"map changed", "US2", "SLICE_MAP_CHANGED", "slice US1 was built with phases", func() *world {
			w := slicedWorld(1, 2, 0)
			w.prior = map[string]PriorSlice{"US1": {Phases: []string{"Phase 2: User Story 1 - a"}, Of: 2}}
			return w
		}},
		{"slice count changed", "US2", "SLICE_MAP_CHANGED", "had 3 slices", func() *world {
			w := slicedWorld(1, 2, 0)
			w.prior = map[string]PriorSlice{"US1": {Phases: []string{"Phase 1: Setup", "Phase 2: User Story 1 - a"}, Of: 3}}
			return w
		}},
		{"this slice's PR open", "US1", "OPEN_ITEM_PR", "002-us1", func() *world {
			w := slicedWorld(0, 0, 0)
			w.heads = []string{"pilot/20261122T000000Z/002-us1"}
			return w
		}},
		{"whole-item PR open", "US1", "OPEN_ITEM_PR", "pilot/x/002", func() *world {
			w := slicedWorld(0, 0, 0)
			w.heads = []string{"pilot/x/002"}
			return w
		}},
		{"slice folder empty", "US2", "ITEM_NO_HIDDEN_TESTS", "items/002/US2", func() *world {
			w := slicedWorld(1, 2, 0)
			delete(w.companion, "items/002/US2/src/test/java/A2Test.java")
			w.companion["items/002/US1/README"] = "x"
			return w
		}},
		{"earlier slice has no tests", "US2", "ITEM_NO_HIDDEN_TESTS", "earlier slice(s) US1", func() *world {
			w := slicedWorld(1, 2, 0)
			delete(w.companion, "items/002/US1/src/test/java/A1Test.java")
			w.companion["items/002/US1/.acceptance-map.json"] = "{}"
			return w
		}},
		{"mixed layout", "US1", "ITEM_NO_HIDDEN_TESTS", "mixes slice folders", func() *world {
			w := slicedWorld(0, 0, 0)
			w.companion["items/002/src/test/java/Loose.java"] = "class Loose {}\n"
			return w
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := c.w()
			w.slice = c.slice
			res, _ := w.check(t, "002", 50)
			if res.Decision != "refuse" || res.Reasons[0] != c.code {
				t.Fatalf("want %s, got %s %+v", c.code, codes(res), res.Rules)
			}
			if d := detail(res, c.code); !strings.Contains(d, c.want) {
				t.Fatalf("%s detail %q, want it to mention %q", c.code, d, c.want)
			}
		})
	}
}

func slicedWorld0() *world { return slicedWorld(0, 0, 0) }

// A later slice's open PR doesn't block an earlier slice (it can't exist without the earlier one
// merged, but the rule only looks back).
func TestLaterSlicePRDoesNotBlockUS1(t *testing.T) {
	w := slicedWorld(0, 0, 0)
	w.slice, w.heads = "US1", []string{"pilot/x/002-us2"}
	if res, _ := w.check(t, "002", 50); res.Decision != "allow" {
		t.Fatalf("%s", codes(res))
	}
}

func TestWholeItemOfASlicedItem(t *testing.T) {
	// Untouched: the whole item runs, overlaying every slice folder in order.
	res, _ := slicedWorld(0, 0, 0).check(t, "002", 50)
	if res.Decision != "allow" || strings.Join(res.Overlay, ",") != "items/002/US1,items/002/US2" || res.Slice != nil || res.HiddenFiles != 2 {
		t.Fatalf("%s overlay=%v hidden=%d", codes(res), res.Overlay, res.HiddenFiles)
	}
	// Some slice already begun: refused, the remaining slices go one by one.
	res, _ = slicedWorld(1, 2, 0).check(t, "002", 50)
	if res.Decision != "refuse" || res.Reasons[0] != "ITEM_PARTLY_SLICED" {
		t.Fatalf("%s", codes(res))
	}
	// Any slice PR open blocks the whole item.
	w := slicedWorld(0, 0, 0)
	w.heads = []string{"pilot/x/002-us2"}
	if res, _ := w.check(t, "002", 50); res.Reasons[0] != "OPEN_ITEM_PR" {
		t.Fatalf("%s", codes(res))
	}
}

// US1 guard (T012): without tasks.phases (spec-status 0.3.0) a whole-item run is exactly as before,
// with no slice rule recorded.
func TestWholeItemWithoutPhasesUnchanged(t *testing.T) {
	w := newWorld()
	setPhases(w, "absent")
	res, _ := w.check(t, "002", 50)
	if res.Decision != "allow" || res.Slice != nil || strings.Join(res.Overlay, ",") != "items/002" {
		t.Fatalf("%s %+v", codes(res), res)
	}
	for _, r := range res.Rules {
		if strings.HasPrefix(r.ID, "SLICE_") || r.ID == "slice" || r.ID == "ITEM_PARTLY_SLICED" {
			t.Fatalf("unexpected slice rule %+v", r)
		}
	}
}

// FR-008, FR-019: a verified item is refused by the gate even with --slice, and an item with a
// merged slice (status still planned) still gates as startable.
func TestSliceGateInteraction(t *testing.T) {
	w := slicedWorld(0, 0, 0)
	w.slice = "US1"
	for _, x := range w.status.doc["items"].([]any) {
		if m := x.(map[string]any); m["id"] == "002" {
			m["status"] = "verified"
		}
	}
	if res, _ := w.check(t, "002", 50); res.Decision != "refuse" || !strings.Contains(codes(res), "ALREADY_VERIFIED") {
		t.Fatalf("verified item: %s", codes(res))
	}
	w = slicedWorld(1, 2, 0)
	w.slice = "US2"
	if res, _ := w.check(t, "002", 50); res.Decision != "allow" || detail(res, "gate") != "allow" {
		t.Fatalf("item with a merged slice must gate as startable: %s", codes(res))
	}
}
