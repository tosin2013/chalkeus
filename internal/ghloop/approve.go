package ghloop

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/tosin2013/chalkeus/internal/decisions"
	"github.com/tosin2013/chalkeus/internal/evidence"
)

// 004 GitHub approvals (reference: ghapprove.py; 004 research R3, R4). The pilot posts each
// pending decision (mode, retry) on the run's tracking issue. An allowed approver replies
// `/approve <step>`. On resume without a terminal, the pilot reads the comments and records the
// first valid approval with channel github, so 012's single-use, bound rules apply unchanged.
//
// A comment counts only if all of these hold:
//   - its first line is exactly `/approve <step>`;
//   - it was posted after the matching decision post, and never edited;
//   - its author is in PILOT_APPROVERS and still has write, maintain or admin on the target repo;
//   - it was not made with the publisher credential, nor through any GitHub App;
//   - its author did not post a later `/withdraw <step>`.
//
// Every candidate is logged with its reason. Any GitHub error means no approval (fail closed).
// Agents must never post these comments.

var allowedPermissions = map[string]bool{"admin": true, "maintain": true, "write": true}

// DecisionMarker matches a decision post's first line, in the new form or the old
// `<!-- specfoundry:… -->` one (posts and tracking records from before 027).
var DecisionMarker = regexp.MustCompile(`<!-- (?:chalkeus|specfoundry):decision run=(\S+) step=(\S+) task=(\S+) n=(\S+) -->`)

// currentMarker rewrites a marker from before the rename (`<!-- specfoundry:`) to today's form, so a
// run paused before the upgrade finds the decision it already posted instead of posting it again.
func currentMarker(m string) string {
	if rest, ok := strings.CutPrefix(m, "<!-- specfoundry:"); ok {
		return "<!-- chalkeus:" + rest
	}
	return m
}

// Tracking is github/tracking.json.
type Tracking struct {
	Issue     any              `json:"issue"` // nil until created
	Decisions []map[string]any `json:"decisions"`
	Closed    bool             `json:"closed,omitempty"` // the run ended and the issue was closed
}

func trackingFile(run *evidence.Run) string { return filepath.Join(run.Dir, "github", "tracking.json") }

// LoadTracking reads the run's tracking record.
func LoadTracking(run *evidence.Run) Tracking {
	t := Tracking{Decisions: []map[string]any{}}
	if raw, err := os.ReadFile(trackingFile(run)); err == nil {
		_ = evidence.Decode(raw, &t)
		if t.Decisions == nil {
			t.Decisions = []map[string]any{}
		}
	}
	return t
}

func saveTracking(run *evidence.Run, t Tracking) error {
	m := map[string]any{"issue": t.Issue, "decisions": t.Decisions}
	if t.Closed {
		m["closed"] = true
	}
	return evidence.WriteJSON(trackingFile(run), m)
}

// TrackingIssue is the tracking issue number, or 0.
func TrackingIssue(run *evidence.Run) int {
	n, _ := num(LoadTracking(run).Issue)
	return n
}

// PublisherLogin is the publisher credential's login, cached in run.json.
func (l *Loop) PublisherLogin(run *evidence.Run) (string, error) {
	if s, _ := run.State["publisher_login"].(string); s != "" {
		return s, nil
	}
	w, err := l.W()
	if err != nil {
		return "", err
	}
	login, err := w.Whoami(l.ctx())
	if err != nil {
		return "", err
	}
	run.State["publisher_login"] = login
	_ = run.Save()
	return login, nil
}

// EnsureTracking creates the run's tracking issue once.
func (l *Loop) EnsureTracking(run *evidence.Run) (int, error) {
	t := LoadTracking(run)
	if n, ok := num(t.Issue); ok && n > 0 {
		return n, nil
	}
	body := fmt.Sprintf("<!-- chalkeus:run=%s -->\nDecisions for pilot run `%s` are posted here. "+
		"Allowed approvers reply with the exact command shown in each post.\n\n"+
		"_The pilot never merges, and a merge is never approved here._\n", run.ID, run.ID)
	w, err := l.W()
	if err != nil {
		return 0, err
	}
	n, err := w.CreateIssue(l.ctx(), l.Repo, "pilot run "+run.ID, body, nil)
	if err != nil {
		return 0, err
	}
	t.Issue = n
	return n, saveTracking(run, t)
}

func dash(v any) string {
	if !truthy(v) {
		return "-"
	}
	return pyStr(v)
}

// Marker is the decision post marker for pending step p.
func Marker(run *evidence.Run, p *decisions.Pending) string {
	return fmt.Sprintf("<!-- chalkeus:decision run=%s step=%s task=%s n=%s -->", run.ID, p.Step, dash(p.Task), dash(p.N))
}

func findDecision(t Tracking, marker string) map[string]any {
	for _, d := range t.Decisions {
		if s, _ := d["marker"].(string); currentMarker(s) == currentMarker(marker) {
			return d
		}
	}
	return nil
}

// PostDecision posts the pending decision once (idempotent per run/step/task/n) and returns
// its tracking entry.
func (l *Loop) PostDecision(run *evidence.Run, p *decisions.Pending) (map[string]any, error) {
	key := Marker(run, p)
	if d := findDecision(LoadTracking(run), key); d != nil {
		return d, nil
	}
	issue, err := l.EnsureTracking(run)
	if err != nil {
		return nil, err
	}
	var at []string
	for _, a := range l.Approvers {
		at = append(at, "@"+a)
	}
	body := fmt.Sprintf("%s\n**Decision pending: %s.**\n\nTo approve, reply with exactly:\n\n    /approve %s\n\n"+
		"Allowed approvers: %s. To cancel your approval before the run resumes: `/withdraw %s`.\n",
		key, decisions.Describe(p), p.Step, strings.Join(at, ", "), p.Step)
	if note := decisionNote(run, p.Step); note != "" { // 030: e.g. why the last attempt stopped
		body += "\n" + note + "\n"
	}
	w, err := l.W()
	if err != nil {
		return nil, err
	}
	cid, err := w.Comment(l.ctx(), l.Repo, issue, body)
	if err != nil {
		return nil, err
	}
	d := map[string]any{"marker": key, "step": p.Step, "task": p.Task, "n": p.N, "comment_id": cid, "at": evidence.Now()}
	t := LoadTracking(run)
	t.Decisions = append(t.Decisions, d)
	if err := saveTracking(run, t); err != nil {
		return nil, err
	}
	_ = run.Event("github", p.Task, p.N, "decision_posted", fmt.Sprintf("step=%s issue=%d comment=%d", p.Step, issue, cid))
	return d, nil
}

// MarkAnswered notes under the tracking issue that the terminal decided (012 R5). Best effort.
func (l *Loop) MarkAnswered(run *evidence.Run, step string, approved bool) {
	issue := TrackingIssue(run)
	if issue == 0 {
		return
	}
	w, err := l.W()
	if err != nil {
		return
	}
	yn := "no"
	if approved {
		yn = "yes"
	}
	_, _ = w.Comment(l.ctx(), l.Repo, issue, fmt.Sprintf("Decision `%s` was answered at the operator's terminal (%s); approvals here no longer apply.", step, yn))
}

// CloseTracking posts the run's outcome on its tracking issue and closes it, once. It is called
// only when a run ends (succeeded or failed); a paused run keeps its issue open for approvals.
func (l *Loop) CloseTracking(run *evidence.Run, outcome, evidencePath string) error {
	t := LoadTracking(run)
	issue, _ := num(t.Issue)
	if issue == 0 || t.Closed {
		return nil
	}
	w, err := l.W()
	if err != nil {
		return err
	}
	body := fmt.Sprintf("Run `%s` %s. Evidence: `%s`.\n\nThis run has ended, so this issue is closed; "+
		"approvals here no longer apply.", run.ID, outcome, evidencePath)
	if _, err := w.Comment(l.ctx(), l.Repo, issue, body); err != nil {
		return err
	}
	if err := w.CloseIssue(l.ctx(), l.Repo, issue); err != nil {
		return err
	}
	t.Closed = true
	return saveTracking(run, t)
}

// LogEntry is one candidate comment's verdict.
type LogEntry struct {
	CommentID int64
	Author    string
	CreatedAt string
	Accepted  bool
	Reason    string // "" when accepted
}

func firstLine(body string) string {
	lines := splitLines(strings.TrimSpace(body))
	if len(lines) == 0 {
		return ""
	}
	return strings.TrimSpace(lines[0])
}

// Evaluate applies the approval rules (pure). It returns the accepted comment (nil if none) and
// a log entry for every `/approve` candidate. Shared vectors:
// specs/004-github-loop/contracts/vectors/approvals.json.
func Evaluate(comments []Comment, decisionCommentID int64, step string, approvers []string, publisher string,
	permission func(login string) string) (*Comment, []LogEntry) {
	want := "/approve " + step
	allowed := map[string]bool{}
	for _, a := range approvers {
		allowed[strings.ToLower(a)] = true
	}
	cs := append([]Comment(nil), comments...)
	sort.SliceStable(cs, func(i, j int) bool { return cs[i].ID < cs[j].ID })
	withdrawn := map[string][]int64{}
	for _, c := range cs {
		if firstLine(c.Body) == "/withdraw "+step && c.ID > decisionCommentID {
			a := strings.ToLower(c.Author)
			withdrawn[a] = append(withdrawn[a], c.ID)
		}
	}
	var log []LogEntry
	var valid []Comment
	for _, c := range cs {
		first := firstLine(c.Body)
		if !strings.HasPrefix(first, "/approve") {
			continue
		}
		author := strings.ToLower(c.Author)
		laterWithdraw := false
		for _, w := range withdrawn[author] {
			if w > c.ID {
				laterWithdraw = true
			}
		}
		reason := ""
		switch {
		case c.ID <= decisionCommentID:
			reason = "before_decision"
		case first != want:
			reason = "wrong_step"
		case c.App:
			reason = "app_authored"
		case author == strings.ToLower(publisher):
			reason = "publisher_identity"
		case c.UpdatedAt != c.CreatedAt:
			reason = "edited"
		case !allowed[author]:
			reason = "not_an_approver"
		case laterWithdraw:
			reason = "withdrawn"
		case !allowedPermissions[permission(c.Author)]:
			reason = "no_write_access"
		case len(valid) > 0:
			reason = "duplicate"
		}
		log = append(log, LogEntry{CommentID: c.ID, Author: c.Author, CreatedAt: c.CreatedAt, Accepted: reason == "", Reason: reason})
		if reason == "" {
			valid = append(valid, c)
		}
	}
	if len(valid) == 0 {
		return nil, log
	}
	return &valid[0], log
}

// Found is an accepted GitHub approval.
type Found struct {
	Approver  string
	CommentID int64
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// FindApproval reads the tracking issue for an approval of pending step p, logging every
// candidate to github/approvals.jsonl once. A GitHub problem is returned as an error (the
// caller keeps the run paused).
func (l *Loop) FindApproval(run *evidence.Run, p *decisions.Pending) (*Found, error) {
	t := LoadTracking(run)
	d := findDecision(t, Marker(run, p))
	issue, _ := num(t.Issue)
	if issue == 0 || d == nil {
		return nil, nil
	}
	comments, err := l.R.Comments(l.ctx(), l.Repo, issue)
	if err != nil {
		return nil, err
	}
	pub, err := l.PublisherLogin(run)
	if err != nil {
		return nil, err
	}
	firstLines := map[int64]string{}
	for _, c := range comments {
		firstLines[c.ID] = firstLine(c.Body)
	}
	dc, _ := num(d["comment_id"])
	accepted, log := Evaluate(comments, int64(dc), p.Step, l.Approvers, pub,
		func(login string) string { return l.R.Permission(l.ctx(), l.Repo, login) })
	f := filepath.Join(run.Dir, "github", "approvals.jsonl")
	seen := map[string]bool{}
	if rows, err := evidence.ReadJSONL(f); err == nil {
		for _, r := range rows {
			seen[fmt.Sprint(r["comment_id"])] = true
		}
	}
	for _, e := range log {
		if seen[fmt.Sprint(e.CommentID)] {
			continue
		}
		row := map[string]any{"comment_id": e.CommentID, "author": e.Author, "created_at": e.CreatedAt,
			"accepted": e.Accepted, "reason": nullable(e.Reason), "step": p.Step, "task": p.Task, "n": p.N, "at": evidence.Now()}
		if err := evidence.AppendJSONL(f, row); err != nil {
			return nil, err
		}
		if !e.Accepted {
			_ = run.Event("approval", p.Task, p.N, "approval_rejected",
				fmt.Sprintf("step=%s comment=%d author=%s reason=%s", p.Step, e.CommentID, e.Author, e.Reason))
			l.reply(issue, RefusalReply(e, firstLines[e.CommentID], p.Step))
		}
	}
	if accepted == nil {
		return nil, nil
	}
	return &Found{Approver: accepted.Author, CommentID: accepted.ID}, nil
}

// Collect is used on resume without a terminal: if a valid GitHub approval exists for the
// pending mode or retry step, record it (channel github). It returns a message for the
// operator, or "".
func (l *Loop) Collect(run *evidence.Run) string {
	p := decisions.PendingStep(run)
	if p == nil || (p.Step != "mode" && p.Step != "retry" && p.Step != "acceptance") {
		return ""
	}
	if decisions.Recorded(run, p.Step, p.Task, p.N) { // already collected (the watcher, then resume)
		return ""
	}
	found, err := l.FindApproval(run, p)
	if err != nil {
		return fmt.Sprintf("could not read GitHub approvals (%v); the run stays paused", err)
	}
	if found == nil {
		return ""
	}
	code, msg := decisions.Record(run, p.Step, "github", found.Approver, found.CommentID)
	if code != 0 {
		return ""
	}
	l.reply(TrackingIssue(run), AcceptReply(p.Step, found))
	return msg
}

// reply posts the bot's answer to an /approve comment on the tracking issue (FR-019). Best
// effort: a failed reply never changes the decision.
func (l *Loop) reply(issue int, body string) {
	if issue == 0 {
		return
	}
	if w, err := l.W(); err == nil {
		_, _ = w.Comment(l.ctx(), l.Repo, issue, body)
	}
}

// AcceptReply is the bot's reply to an accepted /approve.
func AcceptReply(step string, f *Found) string {
	return fmt.Sprintf("✅ `/approve %s` from @%s accepted (comment %d); the run is resuming.", step, f.Approver, f.CommentID)
}

var refusalWhy = map[string]string{
	"before_decision":    "it was posted before this decision",
	"wrong_step":         "the pending step is `{step}`; reply exactly `/approve {step}`",
	"app_authored":       "it was made through a GitHub App or by a bot",
	"publisher_identity": "it was made with the publisher's identity",
	"edited":             "the comment was edited; post a new one instead",
	"not_an_approver":    "@{login} is not in FORGE_APPROVERS",
	"withdrawn":          "it was withdrawn by a later `/withdraw {step}`",
	"no_write_access":    "@{login} no longer has write access to the repository",
	"duplicate":          "an earlier approval of this step was already accepted",
}

// RefusalReply is the bot's reply to a refused /approve: the reason code and what it means.
func RefusalReply(e LogEntry, first, step string) string {
	why := strings.NewReplacer("{step}", step, "{login}", e.Author).Replace(refusalWhy[e.Reason])
	tail := " The run stays paused."
	if e.Reason == "duplicate" {
		tail = ""
	}
	return fmt.Sprintf("❌ `%s` from @%s not accepted (`%s`): %s.%s", first, e.Author, e.Reason, why, tail)
}

// decisionNote is extra context the pipeline left for a pending decision (030: a retry after a
// deadline stop states the time used, usage and changed files).
func decisionNote(run *evidence.Run, step string) string {
	n, _ := run.State["decision_note"].(map[string]any)
	if s, _ := n["step"].(string); s != step {
		return ""
	}
	text, _ := n["text"].(string)
	return text
}
