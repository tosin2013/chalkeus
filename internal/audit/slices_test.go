package audit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tosin2013/chalkeus/internal/evidence"
)

// 026 T035: AU03's slice variant and AU13, on synthetic slice runs of item 006.

func sliceRunDir(t *testing.T, root, id, sliceID, state string, last bool, prs []map[string]any, earlier []string) string {
	t.Helper()
	d := filepath.Join(root, id)
	_ = os.MkdirAll(filepath.Join(d, "gate"), 0o755)
	st := map[string]any{"run_id": id, "kind": "item", "item": "006", "slice": sliceID, "slices_last": last, "state": state,
		"target": map[string]any{"name": "stock", "repo": "me/stockroom"}}
	if err := evidence.WriteJSON(filepath.Join(d, "run.json"), st); err != nil {
		t.Fatal(err)
	}
	all := make([]any, 0, len(prs))
	for _, p := range prs {
		all = append(all, p)
	}
	_ = evidence.WriteJSON(filepath.Join(d, "prs.json"), all)
	_ = evidence.WriteJSON(filepath.Join(d, "gate", "1-006.readiness.json"), map[string]any{"decision": "allow",
		"rules": []any{map[string]any{"id": "gate", "result": "pass"}, map[string]any{"id": "slice", "result": "pass"}},
		"slice": map[string]any{"id": sliceID, "of": 2, "earlier": earlier, "default_revision": "abc1234def"}})
	return d
}

func pr(kind, head string) map[string]any {
	return map[string]any{"kind": kind, "head": head, "draft": true, "author": "bot[bot]", "publisher_identity": "bot[bot]",
		"url": "https://github.com/me/stockroom/pull/1"}
}

func state(d string) map[string]any {
	var st map[string]any
	readJSON(filepath.Join(d, "run.json"), &st)
	return st
}

func TestSliceAU03(t *testing.T) {
	root := t.TempDir()
	us1 := sliceRunDir(t, root, "20990101T000000Z", "US1", "succeeded", false, []map[string]any{pr("task", "pilot/20990101T000000Z/006-us1")}, nil)
	if r := sliceAU03(us1, state(us1), "006", "US1"); r.Result != "pass" {
		t.Fatalf("US1: %+v", r)
	}
	bad := sliceRunDir(t, root, "20990102T000000Z", "US1", "succeeded", false,
		[]map[string]any{pr("task", "pilot/x/006-us1"), pr("status", "pilot/x/status-006")}, nil)
	if r := sliceAU03(bad, state(bad), "006", "US1"); r.Result != "fail" {
		t.Fatalf("a status PR before the last slice must fail: %+v", r)
	}
	wrongHead := sliceRunDir(t, root, "20990103T000000Z", "US1", "succeeded", false, []map[string]any{pr("task", "pilot/x/006")}, nil)
	if r := sliceAU03(wrongHead, state(wrongHead), "006", "US1"); r.Result != "fail" {
		t.Fatalf("the slice's PR must come from its own head: %+v", r)
	}
	last := sliceRunDir(t, root, "20990104T000000Z", "US2", "succeeded", true,
		[]map[string]any{pr("task", "pilot/x/006-us2"), pr("status", "pilot/x/status-006")}, []string{"US1"})
	if r := sliceAU03(last, state(last), "006", "US2"); r.Result != "pass" {
		t.Fatalf("last slice with one status PR: %+v", r)
	}
}

func TestSliceAU13(t *testing.T) {
	root := t.TempDir()
	us1 := sliceRunDir(t, root, "20990101T000000Z", "US1", "succeeded", false, []map[string]any{pr("task", "pilot/a/006-us1")}, nil)
	if r := sliceAU13(us1, state(us1), "006", "US1"); r.Result != "pass" || !strings.Contains(r.Detail, "slice=US1 of 2 earlier=[] status_proposals=0") {
		t.Fatalf("US1: %+v", r)
	}
	us2 := sliceRunDir(t, root, "20990102T000000Z", "US2", "succeeded", true,
		[]map[string]any{pr("task", "pilot/b/006-us2"), pr("status", "pilot/b/status-006")}, []string{"US1"})
	r := sliceAU13(us2, state(us2), "006", "US2")
	if r.Result != "pass" || !strings.Contains(r.Detail, "US1 merged@abc1234 (run 20990101T000000Z)") {
		t.Fatalf("US2 with US1's run local: %+v", r)
	}
	// Another local slice run of the same item also proposed a status: two proposals in all.
	sliceRunDir(t, root, "20990103T000000Z", "US1", "failed", false, []map[string]any{pr("status", "pilot/c/status-006")}, nil)
	if r := sliceAU13(us2, state(us2), "006", "US2"); r.Result != "fail" || !strings.Contains(r.Detail, "more than one status proposal") {
		t.Fatalf("two proposals across runs must fail: %+v", r)
	}
	// US1 built on another machine: reported, not a failure.
	other := t.TempDir()
	alone := sliceRunDir(t, other, "20990105T000000Z", "US2", "succeeded", true, []map[string]any{pr("task", "pilot/d/006-us2")}, []string{"US1"})
	if r := sliceAU13(alone, state(alone), "006", "US2"); r.Result != "pass" || !strings.Contains(r.Detail, "US1 merged@abc1234, evidence elsewhere") {
		t.Fatalf("evidence elsewhere: %+v", r)
	}
	// A non-last slice with a status PR fails AU13 too.
	early := sliceRunDir(t, t.TempDir(), "20990106T000000Z", "US1", "succeeded", false, []map[string]any{pr("status", "pilot/e/status-006")}, nil)
	if r := sliceAU13(early, state(early), "006", "US1"); r.Result != "fail" {
		t.Fatalf("%+v", r)
	}
}

// T036: the item view lists each local run of the item with its slice and writes nothing.
func TestItemView(t *testing.T) {
	root := t.TempDir()
	us1 := sliceRunDir(t, root, "20990101T000000Z", "US1", "succeeded", false, []map[string]any{pr("task", "pilot/a/006-us1")}, nil)
	us2 := sliceRunDir(t, root, "20990102T000000Z", "US2", "succeeded", true, []map[string]any{pr("task", "pilot/b/006-us2")}, []string{"US1"})
	for _, d := range []string{us1, us2} {
		_ = os.WriteFile(filepath.Join(d, "events.jsonl"), nil, 0o644)
		_ = os.MkdirAll(filepath.Join(d, "checks"), 0o755)
		_ = os.WriteFile(filepath.Join(d, "checks", "results.jsonl"), nil, 0o644)
	}
	before := snapshotTree(t, root)
	var b strings.Builder
	if _, err := ItemView(root, "stock", "006", nil, &b); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	for _, want := range []string{"item 006 on target stock: 2 local run(s)", "20990101T000000Z  US1", "20990102T000000Z  US2", "newest run 20990102T000000Z:", "AU13"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in\n%s", want, out)
		}
	}
	if after := snapshotTree(t, root); after != before {
		t.Fatal("the item view must write nothing")
	}
	b.Reset()
	if ok, _ := ItemView(root, "stock", "009", nil, &b); ok || !strings.Contains(b.String(), "no local runs") {
		t.Fatal(b.String())
	}
}

func snapshotTree(t *testing.T, root string) string {
	var parts []string
	_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			raw, _ := os.ReadFile(p)
			parts = append(parts, p+":"+string(raw))
		}
		return nil
	})
	return strings.Join(parts, "\n")
}
