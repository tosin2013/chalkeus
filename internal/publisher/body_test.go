package publisher

import (
	"strings"
	"testing"

	"github.com/tosin2013/chalkeus/internal/evidence"
	"github.com/tosin2013/chalkeus/internal/paths"
)

func bodyRun(kind string) *evidence.Run {
	r := &evidence.Run{ID: "20990101T000000Z", State: map[string]any{"run_id": "20990101T000000Z",
		"tasks": map[string]any{"A": map[string]any{"attempts": []any{map[string]any{"n": 1, "result": "pass"}}, "gate_seqs": []any{1}},
			"002": map[string]any{"attempts": []any{map[string]any{"n": 1, "result": "pass"}}, "gate_seqs": []any{1}}}}}
	if kind == evidence.KindItem {
		r.SetItemRun(map[string]any{"name": "g", "repo": "me/greeting-service"}, "002")
	}
	return r
}

func TestFixtureBodyUnchanged(t *testing.T) {
	b := body(paths.At(t.TempDir()), bodyRun(evidence.KindFixture), "A", 1, 18)
	for _, want := range []string{"Opened by the specfoundry M1 pilot **publisher** (run `20990101T000000Z`, task A, fixture item 001).",
		"Evidence: `specs/002-m1-gated-pilot/evidence/20990101T000000Z/` in the specfoundry repository.", "\nCloses #18\n"} {
		if !strings.Contains(b, want) {
			t.Errorf("missing %q", want)
		}
	}
	if s := StatusBody(bodyRun(evidence.KindFixture), "A", 18); !strings.Contains(s, "item 001 `planned` → `verified`") {
		t.Error(s)
	}
}

func TestItemBodies(t *testing.T) {
	r := bodyRun(evidence.KindItem)
	b := body(paths.At(t.TempDir()), r, "002", 1, 0)
	if !strings.Contains(b, "(run `20990101T000000Z`, item 002 of `me/greeting-service`)") || strings.Contains(b, "fixture") ||
		strings.Contains(b, "specs/002-m1-gated-pilot") || strings.Contains(b, "Closes") {
		t.Fatal(b)
	}
	r.State["status_from"] = "specced"
	if s := StatusBody(r, "002", 7); !strings.Contains(s, "item 002 `specced` → `verified`") || !strings.Contains(s, "Item issue: #7.") {
		t.Fatal(s)
	}
}

func TestConventionalTitles(t *testing.T) {
	r := bodyRun(evidence.KindItem)
	if got := ItemTitle(r, "002"); got != "feat(002): roadmap item 002" {
		t.Fatal(got)
	}
	r.State["readiness"] = map[string]any{"title": "Greeting endpoint", "change_type": "fix"}
	if got := ItemTitle(r, "002"); got != "fix(002): Greeting endpoint" {
		t.Fatal(got)
	}
	if got := statusTitle(r, "002"); got != "docs(roadmap): 002 verified" {
		t.Fatal(got)
	}
	if got := statusTitle(bodyRun(""), "001"); got != "pilot 20990101T000000Z: 001 verified" {
		t.Fatal("fixture wording changed: " + got)
	}
}

// 026 T027: a slice run's PR has its own head, says which slice it is, and only the last slice
// closes the item's issue (FR-016a). A whole-item run is unchanged.
func TestSliceBodiesHeadsAndTitles(t *testing.T) {
	r := bodyRun(evidence.KindItem)
	r.State["readiness"] = map[string]any{"title": "Stock page", "change_type": "feat"}
	if got := ItemTitle(r, "002"); got != "feat(002): Stock page" || headName(r, "002") != "002" {
		t.Fatalf("whole item: %q %q", got, headName(r, "002"))
	}
	if b := body(paths.At(t.TempDir()), r, "002", 1, 7); !strings.Contains(b, "\nCloses #7\n") || strings.Contains(b, "slice") {
		t.Fatal(b)
	}
	r.SetSlice("US1")
	r.State["slices_last"] = false
	if got := ItemTitle(r, "002"); got != "feat(002): Stock page (US1)" {
		t.Fatalf("slice title %q", got)
	}
	if headName(r, "002") != "002-us1" {
		t.Fatalf("slice head %q", headName(r, "002"))
	}
	b := body(paths.At(t.TempDir()), r, "002", 1, 7)
	if !strings.Contains(b, "\nPart of #7\n") || strings.Contains(b, "Closes") || !strings.Contains(b, "slice **US1** of item 002") {
		t.Fatalf("non-last slice body:\n%s", b)
	}
	r.SetSlice("US2")
	r.State["slices_last"] = true
	if b := body(paths.At(t.TempDir()), r, "002", 1, 7); !strings.Contains(b, "\nCloses #7\n") || strings.Contains(b, "Part of") {
		t.Fatalf("last slice body:\n%s", b)
	}
}
