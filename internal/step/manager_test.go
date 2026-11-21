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

// While a run is in flight, step mode relays its watcher's log, only the new lines each pass.
func TestManagerRelaysTheRunLog(t *testing.T) {
	w := newWorld(t)
	dir := filepath.Join(t.TempDir(), "R1")
	_ = os.MkdirAll(filepath.Join(dir, "github"), 0o755)
	logp := filepath.Join(dir, "github", "watch.log")
	_ = os.WriteFile(logp, []byte("Item 009 draft PR: https://x/pull/50\n"), 0o644)
	w.labels[41] = ghloop.Proposal{Labelled: true, AddedBy: "owner"}
	w.runs["009"] = RunInfo{ID: "R1", Dir: dir, State: "paused(awaiting-merge)", Watching: true}
	w.pass(t)
	f, _ := os.OpenFile(logp, os.O_APPEND|os.O_WRONLY, 0o644)
	_, _ = f.WriteString("Status-change PR: https://x/pull/51\n")
	_ = f.Close()
	w.pass(t)
	out := w.m.Out.(*strings.Builder).String()
	if strings.Count(out, "009 run: Item 009 draft PR") != 1 || !strings.Contains(out, "009 run: Status-change PR") {
		t.Fatalf("%s", out)
	}
}

// The wait line says what the owner does and where (owner feedback, 2026-11-19).
func TestWaitingFor(t *testing.T) {
	d := t.TempDir()
	_ = os.MkdirAll(filepath.Join(d, "github"), 0o755)
	_ = os.WriteFile(filepath.Join(d, "github", "tracking.json"), []byte(`{"issue": 45}`), 0o644)
	if got := waitingFor(d, "paused(awaiting-mode)", "me/proj"); !strings.Contains(got, "`/approve mode` on https://github.com/me/proj/issues/45") {
		t.Fatal(got)
	}
	_ = os.WriteFile(filepath.Join(d, "prs.json"), []byte(`[{"kind":"task","url":"https://x/pull/44"}]`), 0o644)
	if got := waitingFor(d, "paused(awaiting-merge)", "me/proj"); got != "waiting for you: review and merge the item's PR: https://x/pull/44" {
		t.Fatal(got)
	}
	_ = os.WriteFile(filepath.Join(d, "prs.json"), []byte(`[{"kind":"task","url":"https://x/pull/44"},{"kind":"status","url":"https://x/pull/45"}]`), 0o644)
	if got := waitingFor(d, "paused(awaiting-merge)", "me/proj"); !strings.Contains(got, "status-change PR (it records the item verified): https://x/pull/45") {
		t.Fatal(got)
	}
	if got := waitingFor(d, "in-progress", "me/proj"); !strings.Contains(got, "nothing needed from you") {
		t.Fatal(got)
	}
	s := ItemState{ID: "009", Proposed: true, Run: RunActive, RunID: "R1", RunWaiting: "waiting for you: review and merge the item's PR: https://x/pull/44"}
	if got := Next(s).String(); got != "009: wait (run R1: waiting for you: review and merge the item's PR: https://x/pull/44)" {
		t.Fatal(got)
	}
}

// Lines written after the last pass while the run was active (the status PR, "succeeded") are
// relayed once the run has ended (found live, 2026-11-19: a status PR merged in 31 s).
func TestManagerRelaysTheEndOfARun(t *testing.T) {
	w := newWorld(t)
	dir := filepath.Join(t.TempDir(), "R1")
	_ = os.MkdirAll(filepath.Join(dir, "github"), 0o755)
	logp := filepath.Join(dir, "github", "watch.log")
	_ = os.WriteFile(logp, []byte("Item 009 draft PR: https://x/pull/44\n"), 0o644)
	w.labels[41] = ghloop.Proposal{Labelled: true, AddedBy: "owner"}
	w.runs["009"] = RunInfo{ID: "R1", Dir: dir, State: "paused(awaiting-merge)", Watching: true}
	w.pass(t)
	if !w.m.Busy {
		t.Fatal("a run in flight makes the loop busy")
	}
	f, _ := os.OpenFile(logp, os.O_APPEND|os.O_WRONLY, 0o644)
	_, _ = f.WriteString("Status-change PR: https://x/pull/45\nrun R1 succeeded.\n")
	_ = f.Close()
	w.runs["009"] = RunInfo{ID: "R1", Dir: dir, State: "succeeded"}
	w.pass(t)
	out := w.m.Out.(*strings.Builder).String()
	if !strings.Contains(out, "009 run: Status-change PR: https://x/pull/45") || !strings.Contains(out, "009 run: run R1 succeeded.") || w.m.Busy {
		t.Fatalf("%s", out)
	}
}

// Audit 2026-11-19: step mode says what to label and where, and why nothing can be proposed.
func TestManagerTellsWhatToLabel(t *testing.T) {
	w := newWorld(t)
	w.m.Src.Repo, w.m.Src.Label = "me/proj", "pilot/go"
	w.pass(t)
	out := w.m.Out.(*strings.Builder).String()
	if !strings.Contains(out, `009: skip (not proposed: to start it, add the label "pilot/go" to https://github.com/me/proj/issues/41)`) {
		t.Fatalf("%s", out)
	}
	w.m.Src.Status = func() (string, error) { return `{"items":[{"id":"008","status":"verified","active":true}]}`, nil }
	w.pass(t)
	w.pass(t)
	if n := strings.Count(w.m.Out.(*strings.Builder).String(), "no item can be proposed now"); n != 1 {
		t.Fatalf("said once, got %d", n)
	}
}

// Audit: an automatic draft that already failed for this spec isn't retried every pass; the
// owner is asked once instead.
func TestManagerNoRedraftLoop(t *testing.T) {
	w := newWorld(t)
	w.labels[41] = ghloop.Proposal{Labelled: true, AddedBy: "owner"}
	w.ready["009"] = readiness.Result{Decision: "allow", Commit: "t1", CompanionCommit: "c1", SpecSHA256: "s1"}
	w.m.Src.AutoDraft = true
	w.m.Src.Drafted = func(item, sha string) bool { return item == "009" && sha == "s1" }
	w.pass(t)
	w.pass(t)
	if strings.Contains(strings.Join(w.calls, ","), "draft") || len(w.comments) != 1 || !strings.Contains(w.comments[0], "needs-tests:009") {
		t.Fatalf("%v %v", w.calls, w.comments)
	}
}

// Audit: a run in flight needs no readiness check (no clones every 15 s).
func TestGatherSkipsReadinessForARunInFlight(t *testing.T) {
	w := newWorld(t)
	w.labels[41] = ghloop.Proposal{Labelled: true, AddedBy: "owner"}
	w.runs["009"] = RunInfo{ID: "R1", State: "paused(awaiting-merge)", Watching: true}
	ready := w.m.Src.Ready
	w.m.Src.Ready = func(i string) (readiness.Result, error) {
		if i == "009" {
			t.Fatal("readiness for a run in flight")
		}
		return ready(i)
	}
	w.pass(t)
}

// With nothing to propose, step mode points at the open planning PR (owner feedback: "but what pr").
func TestManagerPointsAtThePlanningPR(t *testing.T) {
	w := newWorld(t)
	w.m.Src.Status = func() (string, error) { return `{"items":[{"id":"009","status":"verified","active":true}]}`, nil }
	w.m.Src.Planning = func(context.Context) ([]string, error) {
		return []string{"https://github.com/me/proj/pull/46 (docs(010): plan updating a product's stock)"}, nil
	}
	w.pass(t)
	if !strings.HasPrefix(w.m.Next, "review and merge the planning PR https://github.com/me/proj/pull/46") {
		t.Fatalf("%q", w.m.Next)
	}
}

// "next:" tells a first-time user what to do, or that nothing is needed (owner feedback).
func TestManagerNext(t *testing.T) {
	w := newWorld(t)
	w.m.Src.Repo, w.m.Src.Label = "me/proj", "pilot/go"
	w.pass(t)
	if w.m.Next != `009: add the label "pilot/go" to https://github.com/me/proj/issues/41` {
		t.Fatalf("label: %q", w.m.Next)
	}
	w.labels[41] = ghloop.Proposal{Labelled: true, AddedBy: "owner"}
	w.ready["009"] = readiness.Result{Decision: "allow", Commit: "t1", CompanionCommit: "c1"}
	w.m.Src.AutoDraft = true
	w.pass(t)
	if !strings.HasPrefix(w.m.Next, "009: drafting its hidden tests") {
		t.Fatalf("draft: %q", w.m.Next)
	}
	w.runs["009"] = RunInfo{ID: "R1", State: "paused(awaiting-mode)", Watching: true, Waiting: "waiting for you: reply `/approve mode` on https://x/issues/9"}
	w.pass(t)
	if w.m.Next != "009: waiting for you: reply `/approve mode` on https://x/issues/9" {
		t.Fatalf("approve: %q", w.m.Next)
	}
	w.m.Src.Status = func() (string, error) { return `{"items":[{"id":"009","status":"verified","active":true}]}`, nil }
	w.m.Src.Planning = func(context.Context) ([]string, error) { return []string{"https://x/pull/46 (plan 010)"}, nil }
	w.pass(t)
	if w.m.Next != "review and merge the planning PR https://x/pull/46 (plan 010); then label the new item's issue" {
		t.Fatalf("planning: %q", w.m.Next)
	}
}

// T014: an in-progress run with no watcher and no event for an hour crashed: it counts as failed
// (step mode then waits for a relabel); a recent or watched one is still active.
func TestStaleInProgressRun(t *testing.T) {
	old := now
	t.Cleanup(func() { now = old })
	base := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	now = func() time.Time { return base }
	for _, c := range []struct {
		name string
		r    RunInfo
		want string
	}{
		{"recent", RunInfo{State: "in-progress", EndedAt: base.Add(-10 * time.Minute)}, RunActive},
		{"crashed", RunInfo{State: "in-progress", EndedAt: base.Add(-61 * time.Minute)}, RunFailed},
		{"quiet but watched", RunInfo{State: "in-progress", Watching: true, EndedAt: base.Add(-3 * time.Hour)}, RunActive},
		{"no events yet", RunInfo{State: "in-progress"}, RunActive},
	} {
		if got := runState(c.r); got != c.want {
			t.Errorf("%s: %s, want %s", c.name, got, c.want)
		}
	}
	// From the evidence on disk, as NewestRun reads it.
	root := t.TempDir()
	d := filepath.Join(root, "20261121T100000Z")
	_ = os.MkdirAll(d, 0o755)
	_ = os.WriteFile(filepath.Join(d, "run.json"), []byte(`{"run_id":"20261121T100000Z","kind":"item","item":"001","target":{"name":"t","repo":"o/t"},"state":"in-progress"}`), 0o644)
	_ = os.WriteFile(filepath.Join(d, "events.jsonl"), []byte(`{"seq":1,"at":"2026-11-21T10:30:00Z","event":"attempt_started"}`+"\n"), 0o644)
	r, ok := NewestRun(root, nil, "t", "001", func(string) bool { return false })
	if !ok || runState(r) != RunFailed {
		t.Fatalf("a run quiet since 10:30 with no watcher is stale at 12:00: %+v %s", r, runState(r))
	}
	if a := Next(ItemState{ID: "001", Proposed: true, Run: runState(r), RunID: r.ID}); a.Kind != RunFailedAct {
		t.Errorf("step mode waits for a relabel after a crashed run: %v", a)
	}
}
