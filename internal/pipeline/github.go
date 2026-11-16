package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/tosin2013/specfoundry/internal/decisions"
	"github.com/tosin2013/specfoundry/internal/evidence"
	"github.com/tosin2013/specfoundry/internal/fixture"
	"github.com/tosin2013/specfoundry/internal/ghloop"
	"github.com/tosin2013/specfoundry/internal/paths"
	"github.com/tosin2013/specfoundry/internal/pyrepr"
	"github.com/tosin2013/specfoundry/internal/reconcile"
)

// The 004 GitHub loop inside a run (reference: the github_* functions, reconcile_github,
// status_pr_merged, ready_to_resume and watch in pipeline.py).

// Creds is the publisher credential, resolved when used (an App token is minted or refreshed).
type Creds struct {
	Env      func() (string, error) // env file with PILOT_SCM_TOKEN
	App      bool
	BotLogin func() (string, error) // App mode: the bot login passed to the publisher Job
}

func (p *P) approvalsOn() bool { return p.Cfg.GithubApprovals() }
func (p *P) issuesOn() bool    { return p.Cfg.GithubIssues() }

// loop returns the run's GitHub loop (operator reads, publisher-credential writes).
func (p *P) loop() (*ghloop.Loop, error) {
	if p.GH != nil {
		return p.GH, nil
	}
	r, err := ghloop.NewReader()
	if err != nil {
		return nil, err
	}
	p.GH = &ghloop.Loop{Repo: p.Cfg.Repo(), Branch: p.Cfg.Branch(), Approvers: p.Cfg.Approvers(), R: r,
		W: func() (ghloop.Writer, error) {
			env, err := p.Creds.Env()
			if err != nil {
				return nil, err
			}
			return ghloop.NewWriter(env)
		},
		Snapshot: ghloop.DefaultSnapshot(p.Cfg.Repo(), p.Cfg.Branch())}
	return p.GH, nil
}

// githubPost posts the pending mode/retry decision on the run's tracking issue.
func (p *P) githubPost(run *evidence.Run) {
	if !p.approvalsOn() {
		return
	}
	pend := decisions.PendingStep(run)
	if pend == nil || (pend.Step != "mode" && pend.Step != "retry") {
		return
	}
	l, err := p.loop()
	if err == nil {
		_, err = l.PostDecision(run, pend)
	}
	if err != nil {
		p.say("  (could not post the decision on GitHub: %v)", err)
		return
	}
	p.say("  or approve on GitHub: https://github.com/%s/issues/%d (reply `/approve %s`; allowed: %s)",
		p.Cfg.Repo(), ghloop.TrackingIssue(run), pend.Step, strings.Join(p.Cfg.Approvers(), ", "))
}

// githubTerminal notes under the tracking issue that the terminal answered.
func (p *P) githubTerminal(run *evidence.Run, step string, ok bool, ch string) {
	if p.approvalsOn() && ch == "tty" {
		if l, err := p.loop(); err == nil {
			l.MarkAnswered(run, step, ok)
		}
	}
}

func clip(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// githubRefresh re-syncs labels and bodies after a status change; at the end it records the
// drift report. A GitHub error is recorded, never treated as success.
func (p *P) githubRefresh(run *evidence.Run, final bool) {
	if !p.issuesOn() {
		return
	}
	l, err := p.loop()
	if err == nil {
		_, err = l.Sync(run)
	}
	if err == nil && final {
		var rep []pyrepr.OrderedMap
		rep, err = l.Drift(run)
		if err == nil {
			msg := fmt.Sprintf("github drift: %d difference(s)", len(rep))
			if len(rep) > 0 {
				msg += ": " + pyrepr.Value(rep)
			}
			p.say("%s", msg)
		}
	}
	if err != nil {
		_ = run.Event("github", nil, nil, "github_sync_failed", clip(err.Error(), 300))
		p.say("warning: github sync failed: %v", err)
	}
}

// collectGitHub records a valid /approve as the pending approval (resume without a terminal).
func (p *P) collectGitHub(run *evidence.Run) {
	if !p.approvalsOn() || decisions.HasTTY() {
		return
	}
	l, err := p.loop()
	if err != nil {
		p.say("could not read GitHub approvals (%v); the run stays paused", err)
		return
	}
	if msg := l.Collect(run); msg != "" {
		p.say("%s", msg)
	}
}

// closes is the item issue a task's PR closes: the recorded number, or a sync's.
func (p *P) closes(run *evidence.Run) func(task string) (int, error) {
	if !p.issuesOn() {
		return nil
	}
	return func(task string) (int, error) {
		if n := ghloop.IssueFor(run, task); n > 0 {
			return n, nil
		}
		l, err := p.loop()
		if err != nil {
			return 0, err
		}
		s, err := l.Sync(run)
		if err != nil {
			return 0, err
		}
		n, _ := s.Issues.Get(paths.Tasks[task]).(int)
		return n, nil
	}
}

// refreshSecret puts a fresh App token in pilot-publish/scm-token (the 1-hour token may have
// expired since `pilot up`).
func (p *P) refreshSecret() error {
	env, err := p.Creds.Env()
	if err != nil {
		return err
	}
	if err := p.C.DeleteSecret("pilot-publish", "scm-token"); err != nil {
		return err
	}
	return p.C.CreateSecretFromEnvFile("pilot-publish", "scm-token", env)
}

// statusPending is the awaiting-merge pending record while a status-change PR is open.
func statusPending(num int, base string) map[string]any {
	return map[string]any{"task": "A", "status_pr": num, "status_base": base}
}

// reconcileGitHub (004 US3): after A's merge, check reality, open a status-change PR and pause.
// After the human merges it, check it is exactly the proposed edit, merged by an allowed
// approver, then verify A.
func (p *P) reconcileGitHub(run *evidence.Run, pend map[string]any) int {
	if _, ok := pend["status_pr"]; ok && pend["status_pr"] != nil {
		return p.statusPRMerged(run, pend)
	}
	tA := run.Task("A")
	n := asInt(tA["passing_attempt"])
	adir, _ := run.AttemptDir("A", n)
	baseRev, _ := tA["base_revision"].(string)
	rec, cleanup, err := reconcile.ForFixture(p.Cfg.Repo(), p.Cfg.Branch(), run.Dir, run.ID, "A", adir, baseRev,
		func(string) string { return "" }, func(s string) { p.say("%s", s) })
	if err != nil {
		return p.fail(run, "RECONCILE_SETUP")
	}
	pr, _ := tA["pr"].(map[string]any)
	code, r, edited := rec.Check("A", paths.Tasks["A"], asInt(pr["number"]), changedPaths(tA, n))
	var base, bundle string
	if code == 0 {
		base, bundle, err = rec.StatusBundle(edited)
		if err != nil {
			cleanup()
			return p.fail(run, "RECONCILE_STATUS_BUNDLE")
		}
	}
	cleanup()
	r["channel"] = "github"
	_ = evidence.WriteJSON(filepath.Join(run.Dir, "reconcile", "A.json"), r)
	reason, _ := r["reason"].(string)
	ev := "reconcile_proposed"
	if reason != "" {
		ev = "reconcile_" + reason
	}
	mc, _ := r["merge_commit"].(string)
	_ = run.Event("reconcile", "A", n, ev, mc)
	if code == ExitAwaitMerge {
		p.pause(run, "awaiting-merge", map[string]any{"task": "A"})
		p.say("A's PR is not merged yet. Merge it on GitHub, then run `pilot resume` (or `pilot github watch`).")
		return ExitAwaitMerge
	}
	if code != 0 {
		return p.fail(run, "RECONCILE_"+reason)
	}
	deps, err := p.publisherDeps(run)
	var spr map[string]any
	if err == nil {
		spr, err = deps.PublishStatus(run, p.Imgs, "A", base, bundle, ghloop.IssueFor(run, "A"))
	}
	if err != nil {
		_ = run.Event("publish", "A", nil, "status_pr_failed", fmt.Sprintf("%s: %s", errName(err), clip(err.Error(), 300)))
		p.pause(run, "awaiting-merge", map[string]any{"task": "A"})
		p.say("could not open the status-change PR: %v\n`pilot resume` tries again.", err)
		return ExitFail
	}
	p.pause(run, "awaiting-merge", statusPending(asInt(spr["number"]), base))
	p.say("\nStatus-change PR: %v\nIt changes only item %s to `verified`. "+
		"Merging it on GitHub approves the status change; then run `pilot resume` (or `pilot github watch`).", spr["url"], paths.Tasks["A"])
	return ExitAwaitMerge
}

func changedPaths(t map[string]any, n int) []string {
	var out []string
	for _, x := range anyList(t["attempts"]) {
		a, _ := x.(map[string]any)
		if numEq(a["n"], n) {
			for _, c := range anyList(a["changed_paths"]) {
				out = append(out, fmt.Sprint(c))
			}
		}
	}
	return out
}

func (p *P) statusPRMerged(run *evidence.Run, pend map[string]any) int {
	num := asInt(pend["status_pr"])
	base, _ := pend["status_base"].(string)
	item := paths.Tasks["A"]
	l, err := p.loop()
	var view ghloop.PRView
	if err == nil {
		view, err = l.R.PR(l.Context(), p.Cfg.Repo(), num)
	}
	if err != nil {
		p.pause(run, "awaiting-merge", statusPending(num, base))
		p.say("could not read status-change PR #%d: %v; the run stays paused", num, err)
		return ExitError
	}
	if view.State != "MERGED" {
		p.pause(run, "awaiting-merge", statusPending(num, base))
		p.say("status-change PR #%d is not merged yet. Merge it on GitHub to approve, then `pilot resume`.", num)
		return ExitAwaitMerge
	}
	clone, _, err := fixture.FreshClone(p.Cfg.Repo(), p.Cfg.Branch(), 0)
	if err != nil {
		p.pause(run, "awaiting-merge", statusPending(num, base))
		p.say("could not read status-change PR #%d: %v; the run stays paused", num, err)
		return ExitError
	}
	head := view.HeadSHA
	_, ferr := fixture.Git(clone, "fetch", "--quiet", "origin", fmt.Sprintf("pull/%d/head", num))
	diff, derr := fixture.Git(clone, "diff", "--name-only", base, head)
	show := func(rev string) string {
		out, err := fixture.Git(clone, "show", rev+":"+reconcile.RoadmapRel)
		if err != nil {
			return ""
		}
		return out
	}
	baseText, headText := show(base), show(head)
	onDefault := regexp.MustCompile(`(?m)^### ` + regexp.QuoteMeta(item) + ` — .*\[status: verified\]`).MatchString(show("HEAD"))
	_ = os.RemoveAll(clone)
	if ferr != nil || derr != nil {
		p.pause(run, "awaiting-merge", statusPending(num, base))
		p.say("could not read status-change PR #%d: %v; the run stays paused", num, firstErr(ferr, derr))
		return ExitError
	}
	changed := strings.Fields(diff)
	pub, err := l.PublisherLogin(run)
	if err != nil {
		p.pause(run, "awaiting-merge", statusPending(num, base))
		p.say("could not read status-change PR #%d: %v; the run stays paused", num, err)
		return ExitError
	}
	ok, reason := reconcile.CheckStatusPR(view.State, view.MergedBy, baseText, headText, changed, item, p.Cfg.Approvers(), pub)
	if ok && !onDefault {
		ok, reason = false, "STATUS_NOT_ON_DEFAULT"
	}
	changedAny := make([]any, len(changed))
	for i, c := range changed {
		changedAny[i] = c
	}
	_ = evidence.WriteJSON(filepath.Join(run.Dir, "reconcile", "A-status.json"), map[string]any{"pr": num, "base": base,
		"head": nullable(view.HeadSHA), "merge_commit": nullable(view.MergeCommit), "merged_by": nullable(view.MergedBy),
		"changed": changedAny, "ok": ok, "reason": nullable(reason), "at": evidence.Now()})
	rs := reason
	if rs == "" {
		rs = "None"
	}
	_ = run.Event("reconcile", "A", nil, "status_pr_checked", fmt.Sprintf("pr=%d ok=%t reason=%s", num, ok, rs))
	if !ok {
		decisions.Note(run, "reconcile", "A", nil, false, "github", view.MergedBy, fmt.Sprintf("pr=%d reason=%s", num, reason))
		return p.fail(run, reason)
	}
	decisions.Note(run, "reconcile", "A", nil, true, "github", view.MergedBy, fmt.Sprintf("pr=%d", num))
	tA := run.Task("A")
	tA["state"] = "verified"
	_ = run.Save()
	_ = run.Event("status", "A", nil, "task_verified", view.MergeCommit)
	p.githubRefresh(run, false)
	if d := p.gateTask(run, "B"); d.Decision != "allow" {
		return p.fail(run, "B_STILL_BLOCKED")
	}
	return p.runTask(run, "B", 1)
}

func firstErr(errs ...error) error {
	for _, e := range errs {
		if e != nil {
			return e
		}
	}
	return nil
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (p *P) prMerged(num int) bool {
	l, err := p.loop()
	if err != nil {
		return false
	}
	v, err := l.R.PR(l.Context(), p.Cfg.Repo(), num)
	return err == nil && v.State == "MERGED"
}

// ReadyToResume reports whether resume would make progress now (`github watch`): a valid
// GitHub approval was found and recorded, or the awaited PR is merged.
func (p *P) ReadyToResume(run *evidence.Run) bool {
	st, _ := run.State["state"].(string)
	pend, _ := run.State["pending"].(map[string]any)
	switch st {
	case "paused(awaiting-mode)", "paused(awaiting-retry)":
		if l, err := p.loop(); err == nil {
			if msg := l.Collect(run); msg != "" {
				p.say("%s", msg)
			}
		}
		_, err := os.Stat(filepath.Join(run.Dir, "approvals", "pending.json"))
		return err == nil
	case "paused(awaiting-merge)":
		if spr := asInt(pend["status_pr"]); spr > 0 {
			return p.prMerged(spr)
		}
		pr, _ := run.Task("A")["pr"].(map[string]any)
		n := asInt(pr["number"])
		return n > 0 && p.prMerged(n)
	}
	return false
}

// Watch polls GitHub and resumes when a valid approval or a merge appears (004 `github watch`).
// Ctrl-C leaves the run paused.
func (p *P) Watch(runID string, interval time.Duration, sleep func(time.Duration)) int {
	if sleep == nil {
		sleep = time.Sleep
	}
	code := ExitError
	for {
		run, _ := evidence.LoadRun(p.L.EvidenceRoot(), runID)
		if run == nil {
			p.say("no run %s", runID)
			return ExitError
		}
		st := fmt.Sprint(run.State["state"])
		if !strings.HasPrefix(st, "paused") {
			p.say("run %s is %s; stopped watching", runID, st)
			if code != ExitError {
				return code
			}
			if st == "succeeded" {
				return ExitOK
			}
			return ExitError
		}
		if p.ReadyToResume(run) {
			code = p.Resume(runID)
			after, _ := evidence.LoadRun(p.L.EvidenceRoot(), runID)
			if after != nil && after.State["state"] == run.State["state"] && reflect.DeepEqual(after.State["pending"], run.State["pending"]) {
				sleep(interval) // no progress (e.g. declined): don't spin
			}
			continue
		}
		sleep(interval)
	}
}
