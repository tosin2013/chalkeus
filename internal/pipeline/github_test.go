package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/tosin2013/specfoundry/internal/config"
	"github.com/tosin2013/specfoundry/internal/evidence"
	"github.com/tosin2013/specfoundry/internal/ghloop"
	"github.com/tosin2013/specfoundry/internal/paths"
	"github.com/tosin2013/specfoundry/internal/watcher"
)

// A minimal in-memory GitHub for the pipeline's 004 hooks.
type miniGH struct {
	comments map[int][]ghloop.Comment
	closed   []int
	next     int
	cid      int64
	fail     bool
}

func (m *miniGH) Issues(context.Context, string) ([]ghloop.Issue, error) { return nil, m.err() }
func (m *miniGH) Labels(context.Context, string) (map[string]bool, error) {
	return map[string]bool{}, m.err()
}
func (m *miniGH) Comments(_ context.Context, _ string, n int) ([]ghloop.Comment, error) {
	return m.comments[n], m.err()
}
func (m *miniGH) Permission(context.Context, string, string) string { return "write" }
func (m *miniGH) PR(context.Context, string, int) (ghloop.PRView, error) {
	return ghloop.PRView{}, m.err()
}
func (m *miniGH) err() error {
	if m.fail {
		return errors.New("GitHub unreachable")
	}
	return nil
}
func (m *miniGH) Whoami(context.Context) (string, error) { return "pub-bot", nil }
func (m *miniGH) EnsureLabels(context.Context, string, []ghloop.Label) ([]string, error) {
	return nil, nil
}
func (m *miniGH) CreateIssue(context.Context, string, string, string, []string) (int, error) {
	m.next++
	return m.next, nil
}
func (m *miniGH) EditIssue(context.Context, string, int, ghloop.IssueEdit) error { return nil }
func (m *miniGH) Comment(_ context.Context, _ string, n int, body string) (int64, error) {
	m.cid++
	m.comments[n] = append(m.comments[n], ghloop.Comment{ID: m.cid, Body: body, Author: "pub-bot", CreatedAt: "t", UpdatedAt: "t"})
	return m.cid, nil
}
func (m *miniGH) CloseIssue(_ context.Context, _ string, n int) error {
	m.closed = append(m.closed, n)
	return nil
}

func withGitHub(t *testing.T, p *P) *miniGH {
	t.Helper()
	v, _ := config.ValidateValues(map[string]string{"PILOT_FIXTURE_REPO": "o/x-pilot-fixture", "PILOT_MODEL": "p/m",
		"PILOT_MODEL_EGRESS_HOSTS": "h", "PILOT_GITHUB_APPROVALS": "on", "PILOT_APPROVERS": "alice"})
	p.Cfg = &config.Config{Values: v}
	m := &miniGH{comments: map[int][]ghloop.Comment{}}
	p.GH = &ghloop.Loop{Repo: "o/x-pilot-fixture", Approvers: []string{"alice"}, R: m,
		W:        func() (ghloop.Writer, error) { return m, nil },
		Snapshot: func() (map[string]any, map[string]*string, error) { return map[string]any{}, nil, m.err() }}
	return m
}

func TestPauseWithGitHubApprovalsPostsTheDecision(t *testing.T) {
	p, r, out := newP(t)
	m := withGitHub(t, p)
	p.modeNotConfirmed(r, "none")
	if !strings.Contains(out.String(), "  or approve on GitHub: https://github.com/o/x-pilot-fixture/issues/1 (reply `/approve mode`; allowed: alice)") {
		t.Fatalf("%q", out.String())
	}
	if len(m.comments[1]) != 1 || !strings.Contains(m.comments[1][0].Body, "step=mode task=- n=- -->") {
		t.Fatal(m.comments)
	}
	// a valid /approve then becomes the pending approval on resume
	m.comments[1] = append(m.comments[1], ghloop.Comment{ID: 50, Body: "/approve mode", Author: "alice", CreatedAt: "t2", UpdatedAt: "t2"})
	out.Reset()
	p.collectGitHub(r)
	if !strings.Contains(out.String(), "approved by @alice on GitHub: start the run in this mode") {
		t.Fatalf("%q", out.String())
	}
}

func TestGitHubPostFailureIsReportedNotFatal(t *testing.T) {
	p, r, out := newP(t)
	withGitHub(t, p)
	p.GH.W = func() (ghloop.Writer, error) { return nil, errors.New("no credential") }
	if code := p.modeNotConfirmed(r, "none"); code != ExitDeclined {
		t.Fatal(code)
	}
	if !strings.Contains(out.String(), "  (could not post the decision on GitHub: no credential)") {
		t.Fatalf("%q", out.String())
	}
}

func TestGitHubRefreshFailureIsRecorded(t *testing.T) {
	p, r, out := newP(t)
	m := withGitHub(t, p)
	m.fail = true
	p.githubRefresh(r, true)
	if !strings.Contains(out.String(), "warning: github sync failed: GitHub unreachable") {
		t.Fatalf("%q", out.String())
	}
	ev, _ := r.Events()
	if ev[len(ev)-1]["event"] != "github_sync_failed" {
		t.Fatal(ev)
	}
}

func TestWatcherStartsOnlyWithGitHubApprovals(t *testing.T) {
	p, r, out := newP(t)
	calls := 0
	p.EnsureWatcher = func(*evidence.Run) (int, bool, error) { calls++; return 4242, true, nil }
	p.ensureWatcher(r) // approvals off
	if calls != 0 {
		t.Fatal("watcher started with GitHub approvals off")
	}
	withGitHub(t, p)
	p.modeNotConfirmed(r, "none")
	if calls != 1 || !strings.Contains(out.String(), "watcher started (pid 4242); you can close this terminal. Log: ") {
		t.Fatalf("%d %q", calls, out.String())
	}
	out.Reset()
	p.EnsureWatcher = func(*evidence.Run) (int, bool, error) { return 4242, false, nil }
	p.ensureWatcher(r)
	if out.Len() != 0 {
		t.Fatalf("message for an already-running watcher: %q", out.String())
	}
}

func TestWatchLoopRemovesItsOwnPidfile(t *testing.T) {
	p, r, _ := newP(t)
	p.L = paths.At(t.TempDir())
	runDir := filepath.Join(p.L.EvidenceRoot(), r.ID)
	_ = os.MkdirAll(filepath.Join(runDir, "github"), 0o755)
	raw, _ := json.Marshal(map[string]any{"run_id": r.ID, "state": "failed"})
	_ = os.WriteFile(filepath.Join(runDir, "run.json"), raw, 0o644)
	pidfile := watcher.PidFile(runDir)
	_ = os.WriteFile(pidfile, []byte(strconv.Itoa(os.Getpid())), 0o644)
	if code := p.Watch(r.ID, time.Millisecond, func(time.Duration) {}); code != ExitError {
		t.Fatal(code)
	}
	if _, err := os.Stat(pidfile); err == nil {
		t.Fatal("pidfile kept after the run ended")
	}
}

func TestResumeRefusesWhileAnotherWatcherRuns(t *testing.T) {
	p, r, out := newP(t)
	withGitHub(t, p)
	p.L = paths.At(t.TempDir())
	runDir := filepath.Join(p.L.EvidenceRoot(), r.ID)
	_ = os.MkdirAll(filepath.Join(runDir, "github"), 0o755)
	raw, _ := json.Marshal(map[string]any{"run_id": r.ID, "state": "paused(awaiting-mode)", "impl": "go"})
	_ = os.WriteFile(filepath.Join(runDir, "run.json"), raw, 0o644)
	_ = os.WriteFile(watcher.PidFile(runDir), []byte("4242"), 0o644)
	old := watcher.Alive
	watcher.Alive = func(pid int, _ string) bool { return pid == 4242 }
	defer func() { watcher.Alive = old }()
	if code := p.Resume(r.ID); code != ExitError || !strings.Contains(out.String(), "a watcher is resuming run "+r.ID+" on its own (pid 4242)") {
		t.Fatalf("%d %q", code, out.String())
	}
}

func TestWatcherHandsBackAPublishFailure(t *testing.T) {
	p, r, out := newP(t)
	p.L = paths.At(t.TempDir())
	runDir := filepath.Join(p.L.EvidenceRoot(), r.ID)
	_ = os.MkdirAll(filepath.Join(runDir, "github"), 0o755)
	raw, _ := json.Marshal(map[string]any{"run_id": r.ID, "state": "paused(awaiting-publish)", "pending": map[string]any{"task": "A", "n": 2}})
	_ = os.WriteFile(filepath.Join(runDir, "run.json"), raw, 0o644)
	_ = os.WriteFile(watcher.PidFile(runDir), []byte(strconv.Itoa(os.Getpid())), 0o644)
	if code := p.Watch(r.ID, time.Millisecond, func(time.Duration) { t.Fatal("kept waiting") }); code != ExitError {
		t.Fatal(code)
	}
	if _, err := os.Stat(watcher.PidFile(runDir)); err == nil || !strings.Contains(out.String(), "only `pilot/pilot resume --run "+r.ID+"` retries") {
		t.Fatalf("%v %q", err, out.String())
	}
}

// T047 in CI: the three single-account refusals the watcher meets on a tracking issue
// (wrong step, an edited comment, a withdrawn approval) each get one ❌ reply, leave the run
// paused, and are not replied to twice; a valid approval then resumes it with one ✅ reply.
func TestWatcherRefusesSingleAccountCasesThenAccepts(t *testing.T) {
	p, r, _ := newP(t)
	m := withGitHub(t, p)
	p.modeNotConfirmed(r, "none") // posts the decision (comment 1) on tracking issue 1
	m.cid = 1000
	say := func(id int64, body, created, updated string) {
		m.comments[1] = append(m.comments[1], ghloop.Comment{ID: id, Body: body, Author: "alice", CreatedAt: created, UpdatedAt: updated})
	}
	say(100, "/approve retry", "t1", "t1") // wrong_step
	say(101, "/approve mode", "t2", "t3")  // edited after posting
	say(102, "/approve mode", "t4", "t4")  // withdrawn by the next comment
	say(103, "/withdraw mode", "t5", "t5")
	replies := func() []string {
		var out []string
		for _, c := range m.comments[1] {
			if c.Author == "pub-bot" && c.ID > 1000 {
				out = append(out, c.Body)
			}
		}
		return out
	}
	for tick := 0; tick < 2; tick++ { // the watcher polls repeatedly; each refusal is answered once
		if p.ReadyToResume(r) {
			t.Fatalf("tick %d: refused approvals resumed the run", tick)
		}
	}
	got := replies()
	want := []string{"`wrong_step`", "`edited`", "`withdrawn`"}
	if len(got) != len(want) {
		t.Fatalf("replies %q", got)
	}
	for i, w := range want {
		if !strings.HasPrefix(got[i], "❌ ") || !strings.Contains(got[i], w) || !strings.Contains(got[i], "The run stays paused.") {
			t.Fatalf("reply %d = %q, want %s", i, got[i], w)
		}
	}
	say(104, "/approve mode", "t6", "t6")
	if !p.ReadyToResume(r) {
		t.Fatal("a valid approval did not make the run ready to resume")
	}
	got = replies()
	if len(got) != 4 || !strings.HasPrefix(got[3], "✅ `/approve mode` from @alice accepted (comment 104)") {
		t.Fatalf("replies %q", got)
	}
	var reasons []string
	raw, _ := os.ReadFile(filepath.Join(r.Dir, "github", "approvals.jsonl"))
	for _, l := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		var e map[string]any
		_ = json.Unmarshal([]byte(l), &e)
		reasons = append(reasons, fmt.Sprint(e["reason"]))
	}
	if strings.Join(reasons, ",") != "wrong_step,edited,withdrawn,<nil>" {
		t.Fatalf("approvals.jsonl reasons %v", reasons)
	}
}

func TestRunEndClosesTheTrackingIssue(t *testing.T) {
	p, r, out := newP(t)
	m := withGitHub(t, p)
	p.modeNotConfirmed(r, "none") // creates tracking issue #1
	p.githubClose(r, "failed (AUDIT)")
	p.githubClose(r, "failed (AUDIT)") // a second end (resume, watcher) closes nothing more
	if fmt.Sprint(m.closed) != "[1]" {
		t.Fatalf("closed %v; %q", m.closed, out.String())
	}
	last := m.comments[1][len(m.comments[1])-1].Body
	if !strings.Contains(last, "failed (AUDIT)") || !strings.Contains(last, r.ID) {
		t.Fatal(last)
	}
	ev, _ := r.Events()
	if ev[len(ev)-1]["event"] != "tracking_closed" {
		t.Fatal(ev[len(ev)-1])
	}
}

func TestRunEndWithoutGitHubApprovalsTouchesNoIssue(t *testing.T) {
	p, r, _ := newP(t)
	p.githubClose(r, "succeeded") // the fixture config has GitHub approvals off
	ev, _ := r.Events()
	for _, e := range ev {
		if strings.HasPrefix(fmt.Sprint(e["event"]), "tracking_") {
			t.Fatal(e)
		}
	}
}
