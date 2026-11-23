package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tosin2013/chalkeus/internal/decisions"
	"github.com/tosin2013/chalkeus/internal/evidence"
	"github.com/tosin2013/chalkeus/internal/readiness"
	"github.com/tosin2013/chalkeus/internal/reconcile"
)

// slicedRun is item 006's slice run of US1 (of US1, US2), with a merged-looking task and PR.
func slicedRun(t *testing.T, last bool) (*P, *evidence.Run) {
	t.Helper()
	p, r := itemP(t)
	r.SetItemRun(map[string]any{"name": "greeting", "repo": "me/greeting-service"}, "006")
	r.SetSlice("US1")
	r.State["slices_last"] = last
	r.State["readiness"] = map[string]any{"spec_dir": "specs/006-x", "allowed_paths": []any{"app.py"},
		"slice": map[string]any{"id": "US1", "index": 0, "of": 2, "phases": []any{"Phase 1: Setup", "Phase 2: User Story 1"},
			"earlier_phases": []any{}, "all": []any{"US1", "US2"}}}
	tk := r.Task("006")
	tk["passing_attempt"] = 1
	tk["pr"] = map[string]any{"number": 9}
	_ = r.Save()
	return p, r
}

func fakeRec(merged bool, ticked int) reconcile.Reconciler {
	return reconcile.Reconciler{
		PRView: func(int) (map[string]any, error) {
			if !merged {
				return map[string]any{"state": "OPEN"}, nil
			}
			return map[string]any{"state": "MERGED", "mergedAt": "2026-11-22T00:00:00Z", "mergeCommit": map[string]any{"oid": "m1"}}, nil
		},
		ItemStatus: func(string) map[string]any {
			return map[string]any{"tasks": map[string]any{"phases": []any{
				map[string]any{"heading": "Phase 1: Setup", "story": nil, "checked": 1, "total": 1},
				map[string]any{"heading": "Phase 2: User Story 1", "story": 1, "checked": ticked, "total": 2},
				map[string]any{"heading": "Phase 3: User Story 2", "story": 2, "checked": 0, "total": 2},
			}}}
		},
		DefaultFile: func(string) []byte { return []byte("x") },
		GradedFile:  func(string) []byte { return []byte("x") },
		Say:         func(string) {},
	}
}

func events(t *testing.T, r *evidence.Run) string {
	rows, _ := evidence.ReadJSONL(filepath.Join(r.Dir, "events.jsonl"))
	var names []string
	for _, e := range rows {
		names = append(names, e["event"].(string))
	}
	return strings.Join(names, ",")
}

// T032: a US1 merge records slice_merged, sets the task merged, opens no status PR and edits no
// roadmap; it says which slice is next.
func TestSliceMergedNoStatusChange(t *testing.T) {
	p, r := slicedRun(t, false)
	_ = p.sliceMerged(r, fakeRec(true, 2), "006", 1, []string{"app.py"}, "terminal")
	if r.Task("006")["state"] != "merged" {
		t.Fatalf("task state %v", r.Task("006")["state"])
	}
	ev := events(t, r)
	if !strings.Contains(ev, "slice_merged") || strings.Contains(ev, "task_verified") || strings.Contains(ev, "status_pr") {
		t.Fatalf("events %s", ev)
	}
	raw, _ := os.ReadFile(filepath.Join(r.Dir, "reconcile", "006.json"))
	if !strings.Contains(string(raw), `"slice_merged"`) || strings.Contains(string(raw), "applied_commit\": \"") {
		t.Fatalf("reconcile record %s", raw)
	}
	if !strings.Contains(p.Out.(interface{ String() string }).String(), "slice US1 merged; next: --slice US2") {
		t.Fatal(p.Out)
	}
}

func TestSliceMergedWaitsForTheMerge(t *testing.T) {
	p, r := slicedRun(t, false)
	if code := p.sliceMerged(r, fakeRec(false, 2), "006", 1, nil, "terminal"); code != ExitAwaitMerge || r.State["state"] != "paused(awaiting-merge)" {
		t.Fatalf("%d %v", code, r.State["state"])
	}
	p, r = slicedRun(t, false)
	if code := p.sliceMerged(r, fakeRec(true, 1), "006", 1, nil, "terminal"); code == ExitOK || r.State["reason"] != "RECONCILE_SLICE_TASKS_INCOMPLETE" {
		t.Fatalf("%d %v", code, r.State["reason"])
	}
}

func TestSlicesLast(t *testing.T) {
	_, r := slicedRun(t, false)
	if SlicesLast(r) {
		t.Fatal("US1 of 2 is not last")
	}
	r.State["slices_last"] = true
	if !SlicesLast(r) {
		t.Fatal("last slice")
	}
	r.SetSlice("")
	r.State["slices_last"] = false
	if !SlicesLast(r) {
		t.Fatal("a whole-item run ends with the status proposal")
	}
}

// T030/T032: with "## Item acceptance" the status proposal waits for the acceptance decision;
// a recorded approval lets it go on, once.
func TestAcceptanceGate(t *testing.T) {
	p, r := slicedRun(t, true)
	if ok, _ := p.acceptanceGate(r, "006"); !ok {
		t.Fatal("no acceptance checks: go on")
	}
	r.State["readiness"].(map[string]any)["acceptance"] = []any{"a human checks the stock page"}
	ok, code := p.acceptanceGate(r, "006")
	if ok || code != ExitAwaitMerge || r.State["state"] != "paused(awaiting-acceptance)" {
		t.Fatalf("%v %d %v", ok, code, r.State["state"])
	}
	pend := decisions.PendingStep(r)
	if pend == nil || pend.Step != "acceptance" || !strings.Contains(decisions.Describe(pend), "acceptance checks") {
		t.Fatalf("pending %+v", pend)
	}
	if c, msg := decisions.Record(r, "acceptance", "harness", "", nil); c != 0 {
		t.Fatalf("%d %s", c, msg)
	}
	if ok, _ := p.acceptanceGate(r, "006"); !ok || r.State["acceptance_recorded"] != true {
		t.Fatal("a recorded acceptance lets the proposal go on")
	}
	if !strings.Contains(events(t, r), "acceptance_wait") || !strings.Contains(events(t, r), "acceptance_recorded") {
		t.Fatal(events(t, r))
	}
}

func TestOverlayFoldersAndSliceOf(t *testing.T) {
	_, r := slicedRun(t, false)
	d, e := overlayFolders(r, "items/006")
	if strings.Join(d, ",") != "items/006" || len(e) != 0 {
		t.Fatalf("a run without readiness overlay uses the item folder: %v %v", d, e)
	}
	rd := r.State["readiness"].(map[string]any)
	rd["overlay"], rd["earlier"] = []any{"items/006/US2"}, []any{"items/006/US1"}
	d, e = overlayFolders(r, "items/006")
	if strings.Join(d, ",") != "items/006/US2" || strings.Join(e, ",") != "items/006/US1" {
		t.Fatalf("%v %v", d, e)
	}
	id, phases, _ := sliceOf(r)
	if id != "US1" || len(phases) != 2 {
		t.Fatalf("%s %v", id, phases)
	}
	rd["slice"] = &readiness.SliceInfo{ID: "US2", Phases: []string{"Phase 3"}, EarlierPhases: [][]string{{"Phase 1", "Phase 2"}}, All: []string{"US1", "US2"}}
	id, phases, earlier := sliceOf(r)
	if id != "US2" || len(phases) != 1 || len(earlier) != 1 {
		t.Fatalf("%s %v %v", id, phases, earlier)
	}
	p, _ := itemP(t)
	if w := p.workerItem(r).SliceMeta(); w == nil || w["id"] != "US2" || len(w["earlier"].([]any)) != 1 {
		t.Fatalf("worker slice meta %v", w)
	}
}

func TestPriorSlices(t *testing.T) {
	root := t.TempDir()
	mk := func(id, state, sl string, of int) {
		d := filepath.Join(root, id)
		_ = os.MkdirAll(d, 0o755)
		r := &evidence.Run{Dir: d, ID: id, State: map[string]any{"run_id": id, "state": state}}
		r.SetItemRun(map[string]any{"name": "t"}, "006")
		r.SetSlice(sl)
		r.State["readiness"] = map[string]any{"slice": map[string]any{"id": sl, "of": of, "phases": []any{"P-" + id}}}
		_ = r.Save()
	}
	mk("20990101T000000Z", "succeeded", "US1", 2)
	mk("20990102T000000Z", "succeeded", "US1", 3) // newer wins
	mk("20990103T000000Z", "failed", "US2", 3)    // not succeeded: ignored
	got := PriorSlices(root, "t", "006", "")
	if len(got) != 1 || got["US1"].Of != 3 || got["US1"].Phases[0] != "P-20990102T000000Z" {
		t.Fatalf("%+v", got)
	}
	if PriorSlices(root, "other", "006", "") != nil {
		t.Fatal("another target's runs don't count")
	}
}
