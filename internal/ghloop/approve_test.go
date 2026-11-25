package ghloop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/tosin2013/chalkeus/internal/decisions"
	"github.com/tosin2013/chalkeus/internal/evidence"
)

// Cases ported from the former Python pilot's tests/test_ghapprove.py (004 US2): every rejection rule, posting,
// recording with the github channel, and the terminal overriding a GitHub approval.

type approveFixture struct {
	f     *fakeGH
	l     *Loop
	run   *evidence.Run
	p     *decisions.Pending
	d     map[string]any
	issue int
}

func setupApprove(t *testing.T) *approveFixture {
	t.Helper()
	f := newFake("pub-bot", map[string]string{"alice": "write", "bob": "write", "carol": "read"})
	l := f.loop("alice")
	run := testRun(t, "20261116T101500Z", map[string]any{"state": "paused(awaiting-retry)",
		"pending": map[string]any{"task": "A", "n": 2}, "tasks": map[string]any{}})
	p := decisions.PendingStep(run)
	d, err := l.PostDecision(run, p)
	if err != nil {
		t.Fatal(err)
	}
	return &approveFixture{f: f, l: l, run: run, p: p, d: d, issue: TrackingIssue(run)}
}

func (a *approveFixture) say(body, author string) int64 {
	return a.f.addComment(a.issue, body, author, false, false)
}

func (a *approveFixture) find(t *testing.T) *Found {
	t.Helper()
	got, err := a.l.FindApproval(a.run, a.p)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func (a *approveFixture) reasons(t *testing.T) []any {
	t.Helper()
	rows, _ := evidence.ReadJSONL(filepath.Join(a.run.Dir, "github", "approvals.jsonl"))
	var out []any
	for _, r := range rows {
		out = append(out, r["reason"])
	}
	return out
}

func TestTrackingIssueAndPostAreIdempotent(t *testing.T) {
	a := setupApprove(t)
	again, err := a.l.PostDecision(a.run, a.p)
	if err != nil || fmt.Sprint(again["comment_id"]) != fmt.Sprint(a.d["comment_id"]) {
		t.Fatalf("%v %v", again, err)
	}
	if a.f.count("create-issue") != 1 || a.f.count("comment") != 1 {
		t.Fatal(a.f.ops)
	}
	body := a.f.comments[a.issue][0].Body
	for _, want := range []string{"<!-- chalkeus:decision run=20261116T101500Z step=retry task=A n=2 -->", "/approve retry", "@alice"} {
		if !strings.Contains(body, want) {
			t.Errorf("post lacks %q", want)
		}
	}
	want := "<!-- chalkeus:decision run=20261116T101500Z step=retry task=A n=2 -->\n**Decision pending: retry task A (attempt 2).**\n\n" +
		"To approve, reply with exactly:\n\n    /approve retry\n\nAllowed approvers: @alice. To cancel your approval before the run resumes: `/withdraw retry`.\n" +
		"\nTo end the run instead and go back to planning, reply with exactly:\n\n    /approve replan\n" // 033
	if body != want {
		t.Fatalf("post body\n%q", body)
	}
	ev, _ := a.run.Events()
	if ev[len(ev)-1]["event"] != "decision_posted" || ev[len(ev)-1]["detail"] != "step=retry issue=1 comment=1001" {
		t.Fatal(ev)
	}
}

func TestRules(t *testing.T) {
	cases := []struct {
		name    string
		do      func(a *approveFixture)
		accept  bool
		reasons []any
	}{
		{"accept", func(a *approveFixture) { a.say("/approve retry\nlooks fine", "alice") }, true, []any{nil}},
		{"wrong step", func(a *approveFixture) { a.say("/approve mode", "alice") }, false, []any{"wrong_step"}},
		{"extra words", func(a *approveFixture) { a.say("/approve retry please", "alice") }, false, []any{"wrong_step"}},
		{"not listed", func(a *approveFixture) { a.say("/approve retry", "bob") }, false, []any{"not_an_approver"}},
		{"no write access now", func(a *approveFixture) { a.f.permissions["alice"] = "read"; a.say("/approve retry", "alice") }, false, []any{"no_write_access"}},
		{"publisher identity", func(a *approveFixture) {
			a.l.Approvers = []string{"alice", "pub-bot"}
			a.f.permissions["pub-bot"] = "admin"
			a.say("/approve retry", "pub-bot")
		}, false, []any{"publisher_identity"}},
		{"app authored", func(a *approveFixture) { a.f.addComment(a.issue, "/approve retry", "alice", false, true) }, false, []any{"app_authored"}},
		{"edited", func(a *approveFixture) { a.f.addComment(a.issue, "/approve retry", "alice", true, false) }, false, []any{"edited"}},
		{"before decision", func(a *approveFixture) {
			a.f.comments[a.issue] = append([]Comment{{ID: 1, Body: "/approve retry", Author: "alice", CreatedAt: "t", UpdatedAt: "t"}}, a.f.comments[a.issue]...)
		}, false, []any{"before_decision"}},
		{"withdrawn", func(a *approveFixture) { a.say("/approve retry", "alice"); a.say("/withdraw retry", "alice") }, false, []any{"withdrawn"}},
		{"withdraw by someone else", func(a *approveFixture) { a.say("/approve retry", "alice"); a.say("/withdraw retry", "bob") }, true, []any{nil}},
		{"duplicate earliest wins", func(a *approveFixture) { a.say("/approve retry", "alice"); a.say("/approve retry", "alice") }, true, []any{nil, "duplicate"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a := setupApprove(t)
			c.do(a)
			got := a.find(t)
			if (got != nil) != c.accept {
				t.Fatalf("accepted=%v", got)
			}
			if c.accept && (got.Approver != "alice" || got.CommentID != a.f.comments[a.issue][1].ID) {
				t.Fatalf("accepted %+v", got)
			}
			if r := a.reasons(t); !reflect.DeepEqual(r, c.reasons) {
				t.Fatalf("reasons %v", r)
			}
		})
	}
}

func TestOtherAttemptPostDoesNotCount(t *testing.T) {
	a := setupApprove(t)
	a.say("/approve retry", "alice")
	a.run.State["pending"] = map[string]any{"task": "A", "n": 3}
	p3 := decisions.PendingStep(a.run)
	if _, err := a.l.PostDecision(a.run, p3); err != nil {
		t.Fatal(err)
	}
	if got, _ := a.l.FindApproval(a.run, p3); got != nil {
		t.Fatal(got)
	}
}

func TestGitHubErrorFailsClosed(t *testing.T) {
	a := setupApprove(t)
	a.say("/approve retry", "alice")
	a.f.fail = "read"
	if msg := a.l.Collect(a.run); !strings.Contains(msg, "the run stays paused") {
		t.Fatal(msg)
	}
	if _, err := os.Stat(filepath.Join(a.run.Dir, "approvals", "pending.json")); err == nil {
		t.Fatal("approval recorded despite a GitHub error")
	}
}

func TestLogNotDuplicatedAcrossResumes(t *testing.T) {
	a := setupApprove(t)
	a.say("/approve mode", "alice")
	a.find(t)
	a.find(t)
	if r := a.reasons(t); !reflect.DeepEqual(r, []any{"wrong_step"}) {
		t.Fatal(r)
	}
	ev, _ := a.run.Events()
	last := ev[len(ev)-1]
	if last["event"] != "approval_rejected" || last["detail"] != "step=retry comment=1002 author=alice reason=wrong_step" {
		t.Fatal(last)
	}
}

func TestCollectRecordsGitHubChannelUsedOnce(t *testing.T) {
	a := setupApprove(t)
	cid := a.say("/approve retry", "alice")
	msg := a.l.Collect(a.run)
	if !strings.Contains(msg, "@alice") {
		t.Fatal(msg)
	}
	raw, _ := os.ReadFile(filepath.Join(a.run.Dir, "approvals", "pending.json"))
	var rec map[string]any
	_ = json.Unmarshal(raw, &rec)
	if rec["channel"] != "github" || rec["approver"] != "alice" || rec["comment_id"] != float64(cid) {
		t.Fatal(rec)
	}
	old := decisions.HasTTY
	decisions.HasTTY = func() bool { return false }
	defer func() { decisions.HasTTY = old }()
	ok, ch := decisions.Decide(a.run, "retry", "Retry? ", "A", 2, func(string) {})
	if !ok || ch != "github" {
		t.Fatal(ok, ch)
	}
	decisions.Note(a.run, "retry", "A", 2, ok, ch, "", "")
	if ok, ch := decisions.Decide(a.run, "retry", "Retry? ", "A", 2, func(string) {}); ok || ch != "none" {
		t.Fatal("approval used twice")
	}
	ev, _ := a.run.Events()
	var last map[string]any
	for _, e := range ev {
		if e["event"] == "approval_channel" {
			last = e
		}
	}
	if want := fmt.Sprintf("step=retry approved=true channel=github approver=alice comment=%d", cid); last["detail"] != want {
		t.Fatal(last["detail"])
	}
}

func TestTerminalOverridesGitHubApproval(t *testing.T) {
	a := setupApprove(t)
	a.say("/approve retry", "alice")
	a.l.Collect(a.run)
	oh, oa := decisions.HasTTY, decisions.AskTTY
	decisions.HasTTY = func() bool { return true }
	decisions.AskTTY = func(string) string { return "n" }
	defer func() { decisions.HasTTY, decisions.AskTTY = oh, oa }()
	if ok, ch := decisions.Decide(a.run, "retry", "Retry? ", "A", 2, func(string) {}); ok || ch != "tty" {
		t.Fatal(ok, ch)
	}
	if _, err := os.Stat(filepath.Join(a.run.Dir, "approvals", "pending.json")); err == nil {
		t.Fatal("pending approval kept after the terminal answered")
	}
}

func TestReconcileAndMergeNotCollected(t *testing.T) {
	f := newFake("pub-bot", nil)
	r := testRun(t, "20261116T111111Z", map[string]any{"state": "paused(awaiting-merge)", "pending": map[string]any{"task": "A"}})
	if msg := f.loop("alice").Collect(r); msg != "" {
		t.Fatal(msg)
	}
}

func TestMarkAnswered(t *testing.T) {
	a := setupApprove(t)
	a.l.MarkAnswered(a.run, "retry", true)
	c := a.f.comments[a.issue]
	if got := c[len(c)-1].Body; got != "Decision `retry` was answered at the operator's terminal (yes); approvals here no longer apply." {
		t.Fatal(got)
	}
}

// FR-019: the bot answers each /approve once: accepted, or refused with the reason.
func TestRepliesToApprovals(t *testing.T) {
	a := setupApprove(t)
	a.say("/approve mode", "alice")
	a.say("/approve retry", "bob")
	a.find(t)
	a.find(t) // a second poll must not reply again
	bodies := func() []string {
		var out []string
		for _, c := range a.f.comments[a.issue][1:] {
			if c.Author == "pub-bot" {
				out = append(out, c.Body)
			}
		}
		return out
	}
	want := []string{
		"❌ `/approve mode` from @alice not accepted (`wrong_step`): the pending step is `retry`; reply exactly `/approve retry`. The run stays paused.",
		"❌ `/approve retry` from @bob not accepted (`not_an_approver`): @bob is not in FORGE_APPROVERS. The run stays paused.",
	}
	if got := bodies(); !reflect.DeepEqual(got, want) {
		t.Fatalf("replies\n%q", got)
	}
	cid := a.say("/approve retry", "alice")
	if msg := a.l.Collect(a.run); msg == "" {
		t.Fatal("not collected")
	}
	got := bodies()
	if last := got[len(got)-1]; last != fmt.Sprintf("✅ `/approve retry` from @alice accepted (comment %d); the run is resuming.", cid) {
		t.Fatal(last)
	}
}

// The watcher collects an approval, then resume collects again: recorded once, one reply.
func TestCollectTwiceRecordsOnce(t *testing.T) {
	a := setupApprove(t)
	a.say("/approve retry", "alice")
	if a.l.Collect(a.run) == "" {
		t.Fatal("not collected")
	}
	n := len(a.f.comments[a.issue])
	if msg := a.l.Collect(a.run); msg != "" {
		t.Fatalf("second collect: %q", msg)
	}
	ev, _ := a.run.Events()
	rec := 0
	for _, e := range ev {
		if e["event"] == "approval_recorded" {
			rec++
		}
	}
	if rec != 1 || len(a.f.comments[a.issue]) != n {
		t.Fatalf("recorded %d times, comments %d -> %d", rec, n, len(a.f.comments[a.issue]))
	}
}

func TestEveryRefusalReasonHasAnExplanation(t *testing.T) {
	for _, r := range []string{"before_decision", "wrong_step", "app_authored", "publisher_identity", "edited",
		"not_an_approver", "withdrawn", "no_write_access", "duplicate"} {
		got := RefusalReply(LogEntry{Author: "x", Reason: r}, "/approve retry", "retry")
		if strings.Contains(got, "%!") || strings.Contains(got, "{") || !strings.Contains(got, "(`"+r+"`)") {
			t.Errorf("%s: %q", r, got)
		}
	}
}

func TestCloseTrackingPostsOutcomeAndClosesOnce(t *testing.T) {
	a := setupApprove(t)
	if err := a.l.CloseTracking(a.run, "succeeded", "specs/x/evidence/20261116T101500Z"); err != nil {
		t.Fatal(err)
	}
	if a.f.issue(a.issue).State != "closed" {
		t.Fatalf("issue not closed: %+v", a.f.issue(a.issue))
	}
	last := a.f.comments[a.issue][len(a.f.comments[a.issue])-1].Body
	for _, want := range []string{"Run `20261116T101500Z` succeeded", "specs/x/evidence/20261116T101500Z", "approvals here no longer apply"} {
		if !strings.Contains(last, want) {
			t.Errorf("closing comment lacks %q: %s", want, last)
		}
	}
	if !LoadTracking(a.run).Closed {
		t.Error("tracking.json does not record the close")
	}
	// Idempotent: a second call (a resumed watcher, a re-run of finish) posts nothing more.
	if err := a.l.CloseTracking(a.run, "succeeded", "x"); err != nil {
		t.Fatal(err)
	}
	if a.f.count("close-issue") != 1 || a.f.count("comment") != 2 {
		t.Fatal(a.f.ops)
	}
}

func TestCloseTrackingWithoutIssueDoesNothing(t *testing.T) {
	f := newFake("pub-bot", nil)
	run := testRun(t, "20261116T101600Z", map[string]any{"state": "failed"})
	if err := f.loop("alice").CloseTracking(run, "failed (X)", "x"); err != nil {
		t.Fatal(err)
	}
	if len(f.ops) != 0 {
		t.Fatal(f.ops)
	}
}

func TestCloseTrackingErrorLeavesItOpenForARetry(t *testing.T) {
	a := setupApprove(t)
	a.f.fail = "close-issue"
	if err := a.l.CloseTracking(a.run, "succeeded", "x"); err == nil {
		t.Fatal("want an error")
	}
	if LoadTracking(a.run).Closed {
		t.Fatal("recorded closed although closing failed")
	}
}

// 027: a run paused before the rename recorded its decision post with the old marker; after the
// upgrade it must find that post instead of posting the decision again.
func TestOldMarkerDecisionNotPostedAgain(t *testing.T) {
	a := setupApprove(t)
	tr := LoadTracking(a.run)
	for _, d := range tr.Decisions {
		d["marker"] = strings.Replace(d["marker"].(string), "<!-- chalkeus:", "<!-- specfoundry:", 1)
	}
	if err := saveTracking(a.run, tr); err != nil {
		t.Fatal(err)
	}
	again, err := a.l.PostDecision(a.run, a.p)
	if err != nil || fmt.Sprint(again["comment_id"]) != fmt.Sprint(a.d["comment_id"]) || a.f.count("comment") != 1 {
		t.Fatalf("%v %v %v", again, err, a.f.ops)
	}
	for _, m := range []string{"<!-- specfoundry:decision run=R step=mode task=- n=- -->", "<!-- chalkeus:decision run=R step=mode task=- n=- -->"} {
		if g := DecisionMarker.FindStringSubmatch(m); g == nil || g[1] != "R" || g[2] != "mode" {
			t.Errorf("%s: %v", m, g)
		}
	}
}

// 030: a pending decision carries the pipeline's note for its step only.
func TestDecisionNote(t *testing.T) {
	run := &evidence.Run{State: map[string]any{"decision_note": map[string]any{"step": "retry", "text": "  attempt 1 stopped at the 2400 s deadline"}}}
	if got := decisionNote(run, "retry"); got != "  attempt 1 stopped at the 2400 s deadline" {
		t.Fatal(got)
	}
	if got := decisionNote(run, "mode"); got != "" {
		t.Fatal(got)
	}
}
