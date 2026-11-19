package step

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tosin2013/specfoundry/internal/ghloop"
	"github.com/tosin2013/specfoundry/internal/readiness"
)

const roadmap = `{"items":[
 {"id":"008","title":"done","status":"verified","active":true,"dependencies":[]},
 {"id":"009","title":"Delete a product","status":"specced","active":true,"dependencies":[{"id":"004","satisfied":true,"state":"satisfied"}]},
 {"id":"010","title":"Next","status":"specced","active":true,"dependencies":[]}]}`

type world struct {
	labels   map[int]ghloop.Proposal
	ready    map[string]readiness.Result
	prs      map[string]bool
	runs     map[string]RunInfo
	calls    []string
	comments []string
	doctor   int // the doctor's exit code
	m        *Manager
}

func newWorld(t *testing.T) *world {
	t.Helper()
	w := &world{labels: map[int]ghloop.Proposal{}, ready: map[string]readiness.Result{}, prs: map[string]bool{}, runs: map[string]RunInfo{}}
	for _, i := range []string{"009", "010"} {
		w.ready[i] = readiness.Result{Decision: "allow", HiddenFiles: 1, Commit: "t1", CompanionCommit: "c1"}
	}
	root := t.TempDir()
	dir := filepath.Join(root, "step", "stockroom")
	log, _ := OpenLog(dir)
	issues := func(context.Context) (map[string]int, error) { return map[string]int{"009": 41, "010": 44}, nil }
	src := Source{Status: func() (string, error) { return roadmap, nil }, Issues: issues,
		Proposal:  func(_ context.Context, n int) (ghloop.Proposal, error) { return w.labels[n], nil },
		Ready:     func(i string) (readiness.Result, error) { return w.ready[i], nil },
		TestsPR:   func(_ context.Context, i string) (bool, error) { return w.prs[i], nil },
		Run:       func(i string) (RunInfo, bool) { r, ok := w.runs[i]; return r, ok },
		Approvers: []string{"owner"}, Log: log}
	w.m = &Manager{Target: "stockroom", Src: src, Log: log, Dir: dir, Out: &strings.Builder{}, Issues: issues,
		Sync: func(context.Context) error { w.calls = append(w.calls, "sync"); return nil },
		Draft: func(_ context.Context, i string) (string, error) {
			w.calls = append(w.calls, "draft "+i)
			return "opened", nil
		},
		Pilot: func(_ context.Context, args []string, logPath string) (int, error) {
			w.calls = append(w.calls, strings.Join(args[:1], " ")+" "+args[4])
			_ = os.MkdirAll(filepath.Dir(logPath), 0o755)
			if args[0] == "doctor" {
				_ = os.WriteFile(logPath, []byte("  PASS x\n  FAIL ITEM_NO_HIDDEN_TESTS: none\ndoctor: 1 fail\n"), 0o644)
				return w.doctor, nil
			}
			_ = os.WriteFile(logPath, []byte("== run 20261120T010203Z (target stockroom, item 009)\npaused\n"), 0o644)
			return 5, nil
		},
		Post: func(_ context.Context, n int, body string) error { w.comments = append(w.comments, body); return nil }}
	return w
}

func (w *world) pass(t *testing.T) bool {
	t.Helper()
	f, err := w.m.Pass(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestManagerCarriesALabelledItem(t *testing.T) {
	w := newWorld(t)
	w.pass(t)
	if strings.Join(w.calls, ",") != "sync" {
		t.Fatalf("nothing is proposed, nothing starts: %v", w.calls)
	}
	w.labels[41] = ghloop.Proposal{Labelled: true, AddedBy: "owner"}
	w.pass(t)
	if strings.Join(w.calls, ",") != "sync,sync,doctor 009,run 009" {
		t.Fatalf("%v", w.calls)
	}
	if runs := w.m.Log.RunDirs(); len(runs) != 1 || !strings.HasSuffix(runs[0], "20261120T010203Z") {
		t.Fatalf("the started run's folder is recorded: %v", runs)
	}
	// While the run waits for the owner, nothing else starts, for this item or the next.
	w.runs["009"] = RunInfo{ID: "20261120T010203Z", State: "paused(awaiting-mode)", Watching: true}
	w.labels[44] = ghloop.Proposal{Labelled: true, AddedBy: "owner"}
	w.calls = nil
	w.pass(t)
	if strings.Join(w.calls, ",") != "sync" {
		t.Fatalf("one item in flight: %v", w.calls)
	}
	// It succeeded: the next proposed item goes.
	w.runs["009"] = RunInfo{ID: "20261120T010203Z", State: "succeeded"}
	w.calls = nil
	w.pass(t)
	if strings.Join(w.calls, ",") != "sync,doctor 010,run 010" {
		t.Fatalf("%v", w.calls)
	}
}

func TestManagerApproverLabelOnly(t *testing.T) {
	w := newWorld(t)
	w.labels[41] = ghloop.Proposal{Labelled: true, AddedBy: "mallory"}
	w.pass(t)
	out := w.m.Out.(*strings.Builder).String()
	if len(w.calls) != 1 || !strings.Contains(out, "@mallory, who is not in PILOT_APPROVERS") {
		t.Fatalf("%v\n%s", w.calls, out)
	}
}

func TestManagerDraftsAndAsksForTests(t *testing.T) {
	w := newWorld(t)
	w.labels[41] = ghloop.Proposal{Labelled: true, AddedBy: "owner"}
	w.ready["009"] = readiness.Result{Decision: "allow", Commit: "t1", CompanionCommit: "c1"}
	w.m.Src.AutoDraft = true
	w.pass(t)
	if strings.Join(w.calls, ",") != "sync,draft 009" {
		t.Fatalf("%v", w.calls)
	}
	w.prs["009"] = true
	w.calls = nil
	w.pass(t)
	if strings.Join(w.calls, ",") != "sync" {
		t.Fatalf("waits for the tests PR: %v", w.calls)
	}
	w.m.Src.AutoDraft, w.prs["009"] = false, false
	w.pass(t)
	w.pass(t)
	if len(w.comments) != 1 || !strings.Contains(w.comments[0], "pilot-step:needs-tests:009") {
		t.Fatalf("one needs-tests comment: %v", w.comments)
	}
}

func TestManagerDoctorFailureCommentsOnceUntilAChange(t *testing.T) {
	w := newWorld(t)
	w.labels[41] = ghloop.Proposal{Labelled: true, AddedBy: "owner"}
	w.doctor = 1
	if !w.pass(t) {
		t.Fatal("a failed doctor is reported")
	}
	w.pass(t)
	if strings.Join(w.calls, ",") != "sync,doctor 009,sync" || len(w.comments) != 1 || !strings.Contains(w.comments[0], "FAIL ITEM_NO_HIDDEN_TESTS") {
		t.Fatalf("%v %v", w.calls, w.comments)
	}
	// The companion changed: doctor runs again, and now passes.
	w.ready["009"] = readiness.Result{Decision: "allow", HiddenFiles: 1, Commit: "t1", CompanionCommit: "c2"}
	w.doctor = 0
	w.calls = nil
	w.pass(t)
	if strings.Join(w.calls, ",") != "sync,doctor 009,run 009" {
		t.Fatalf("%v", w.calls)
	}
}

func TestManagerStaleOrFailedRunWaitsForARelabel(t *testing.T) {
	w := newWorld(t)
	ended := time.Date(2026, 11, 19, 19, 0, 0, 0, time.UTC)
	w.runs["009"] = RunInfo{ID: "R1", State: "paused(awaiting-merge)", Watching: false, EndedAt: ended} // its PR was closed
	w.labels[41] = ghloop.Proposal{Labelled: true, AddedBy: "owner", AddedAt: ended.Add(-time.Hour)}
	w.labels[44] = ghloop.Proposal{Labelled: true, AddedBy: "owner"}
	w.pass(t)
	if strings.Join(w.calls, ",") != "sync,doctor 010,run 010" || len(w.comments) != 1 || !strings.Contains(w.comments[0], "run-failed:R1") {
		t.Fatalf("a stale run doesn't hold the slot; the next item goes: %v %v", w.calls, w.comments)
	}
	delete(w.labels, 44)
	w.runs["010"] = RunInfo{ID: "R2", State: "succeeded"}
	w.labels[41] = ghloop.Proposal{Labelled: true, AddedBy: "owner", AddedAt: ended.Add(time.Hour)} // relabelled
	w.calls = nil
	w.pass(t)
	if strings.Join(w.calls, ",") != "sync,doctor 009,run 009" {
		t.Fatalf("a relabel tries again: %v", w.calls)
	}
}

func TestNewestRun(t *testing.T) {
	root := t.TempDir()
	mk := func(id, target, item, state, at string) {
		d := filepath.Join(root, id)
		_ = os.MkdirAll(d, 0o755)
		_ = os.WriteFile(filepath.Join(d, "run.json"), []byte(`{"run_id":"`+id+`","kind":"item","item":"`+item+`","target":{"name":"`+target+`"},"state":"`+state+`"}`), 0o644)
		_ = os.WriteFile(filepath.Join(d, "events.jsonl"), []byte(`{"at":"2026-11-19T10:00:00Z"}`+"\n"+`{"at":"`+at+`"}`+"\n"), 0o644)
	}
	mk("20261119T100000Z", "stockroom", "009", "failed", "2026-11-19T10:05:00Z")
	mk("20261119T110000Z", "stockroom", "009", "paused(awaiting-mode)", "2026-11-19T11:01:00Z")
	mk("20261119T120000Z", "other", "009", "succeeded", "2026-11-19T12:00:00Z")
	r, ok := NewestRun(root, nil, "stockroom", "009", func(string) bool { return true })
	if !ok || r.ID != "20261119T110000Z" || r.State != "paused(awaiting-mode)" || r.EndedAt.Hour() != 11 || runState(r) != RunActive {
		t.Fatalf("%+v %v", r, ok)
	}
	if _, ok := NewestRun(root, nil, "stockroom", "010", nil); ok {
		t.Fatal("no run for 010")
	}
}

func TestManagerPrintsOnlyChanges(t *testing.T) {
	w := newWorld(t)
	w.labels[41] = ghloop.Proposal{Labelled: true, AddedBy: "mallory"}
	w.pass(t)
	w.pass(t)
	if n := strings.Count(w.m.Out.(*strings.Builder).String(), "009: skip"); n != 1 {
		t.Fatalf("the same decision is printed once, got %d", n)
	}
}

// A stopped manager starts nothing new (T010).
func TestManagerStartsNothingWhenStopped(t *testing.T) {
	w := newWorld(t)
	w.labels[41] = ghloop.Proposal{Labelled: true, AddedBy: "owner"}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := w.m.Pass(ctx); err != nil {
		t.Fatal(err)
	}
	if strings.Join(w.calls, ",") != "sync" {
		t.Fatalf("%v", w.calls)
	}
}
