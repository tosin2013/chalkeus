// Package pipeline is `./chalkeus forge run` / `./chalkeus forge resume`: one governed cycle for fixture tasks A then
// B (reference: the former Python pilot's pipeline.py). Every human decision fails closed:
// anything but "y" means no; without a terminal the run pauses at the decision instead.
//
//	mode confirmation (FR-023)   -> exit 5, no worker staged (no terminal: paused(awaiting-mode))
//	retry confirmation (FR-017)  -> paused(awaiting-retry), exit 6
//	merge of A's PR (FR-018a)    -> paused(awaiting-merge), exit 0 / resume exit 4
//	reconcile approval (FR-018)  -> exit 5, roadmap unchanged
package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/tosin2013/chalkeus/internal/scm"
	"github.com/tosin2013/chalkeus/internal/scm/hosts"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/tosin2013/chalkeus/internal/audit"
	"github.com/tosin2013/chalkeus/internal/checks"
	"github.com/tosin2013/chalkeus/internal/cluster"
	"github.com/tosin2013/chalkeus/internal/cnitoken"
	"github.com/tosin2013/chalkeus/internal/config"
	"github.com/tosin2013/chalkeus/internal/decisions"
	"github.com/tosin2013/chalkeus/internal/evidence"
	"github.com/tosin2013/chalkeus/internal/fixture"
	"github.com/tosin2013/chalkeus/internal/gate"
	"github.com/tosin2013/chalkeus/internal/ghloop"
	"github.com/tosin2013/chalkeus/internal/grader"
	"github.com/tosin2013/chalkeus/internal/images"
	"github.com/tosin2013/chalkeus/internal/k8s"
	"github.com/tosin2013/chalkeus/internal/paths"
	"github.com/tosin2013/chalkeus/internal/publisher"
	"github.com/tosin2013/chalkeus/internal/pyrepr"
	"github.com/tosin2013/chalkeus/internal/reconcile"
	"github.com/tosin2013/chalkeus/internal/runner"
	"github.com/tosin2013/chalkeus/internal/verbose"
	"github.com/tosin2013/chalkeus/internal/watcher"
	"github.com/tosin2013/chalkeus/internal/worker"
)

// Exit codes (identical to the reference).
const (
	ExitOK, ExitFail, ExitError, ExitAwaitMerge, ExitDeclined, ExitAwaitRetry, ExitRefuse = 0, 1, 2, 4, 5, 6, 10
)

// P holds one invocation's collaborators.
type P struct {
	L      paths.Layout
	Cfg    *config.Config
	C      *k8s.Client
	Runner runner.Runner
	Imgs   map[string]string
	Out    io.Writer
	Status gate.StatusSource
	// Creds is the publisher credential (token or App token), resolved when used.
	Creds Creds
	// GH is the 004 GitHub loop (built on first use).
	GH *ghloop.Loop
	// EnsureWatcher starts the run's detached watcher (nil: watcher.Ensure; replaced in tests).
	EnsureWatcher func(run *evidence.Run) (pid int, started bool, err error)
	// Target is the repository this invocation acts on (014): the fixture unless an item run.
	Target config.Target
	// budget is the run's resolved resource budget (030), set by runBudget.
	budget *images.Budget
	// Network checks Calico's CNI token before each attempt and refreshes it when too short
	// (031 T034; nil: skipped, as in tests).
	Network func(need time.Duration) (string, error)
}

// CNINetwork is P.Network for the kind cluster: restart calico-node when the token on the node
// won't last the attempt (docs/troubleshooting.md#calico-token-expired).
func CNINetwork(c *k8s.Client) func(time.Duration) (string, error) {
	return func(need time.Duration) (string, error) {
		return cnitoken.Ensure(paths.Cluster+"-control-plane", need, 2*time.Minute, func() error {
			return c.RestartDaemonSet("kube-system", "calico-node", 180*time.Second)
		})
	}
}

// item reports whether this invocation is an item run of a real target (014).
func (p *P) item() bool { return p.Target.Repo != "" && !p.Target.Fixture }

// repo and branch are the target's for an item run, else the fixture's.
func (p *P) repo() string {
	if p.item() {
		return p.Target.Repo
	}
	return p.Cfg.Repo()
}

// host is the Git host of the repository this invocation acts on (037).
func (p *P) host() scm.Host { return p.Target.Host() }

func (p *P) branch() string {
	if p.item() {
		return p.Target.Branch
	}
	return p.Cfg.Branch()
}

// mergeTask is the task whose PR a paused run awaits: A for the fixture, the item for an item run.
func mergeTask(run *evidence.Run) string {
	if run.Kind() == evidence.KindItem {
		item, _ := run.State["item"].(string)
		return item
	}
	return "A"
}

func (p *P) say(format string, a ...any) { fmt.Fprintf(p.Out, format+"\n", a...) }

// New wires the real collaborators.
func New(l paths.Layout, cfg *config.Config, out io.Writer, creds Creds) (*P, error) {
	c, err := k8s.New(paths.Context)
	if err != nil {
		return nil, err
	}
	return &P{L: l, Cfg: cfg, C: c, Runner: runner.Jobs{C: c}, Out: out, Status: gate.ExecStatus{}, Creds: creds, Network: CNINetwork(c)}, nil
}

// ModeOf is the run mode shown to the operator for confirmation.
func (p *P) ModeOf() map[string]any {
	cap, _ := strconv.ParseFloat(p.Cfg.Get("PILOT_SPEND_CAP_USD"), 64)
	tok, _ := strconv.Atoi(p.Cfg.Get("PILOT_TOKEN_CAP"))
	deadline := worker.Deadline
	if p.budget != nil && p.budget.Worker.DeadlineS > 0 {
		deadline = p.budget.Worker.DeadlineS
	}
	return map[string]any{"name": "step-by-step", "retry_limit": p.Cfg.RetryLimit(), "spend_cap_usd": evidence.PyFloat(cap),
		"token_cap": tok, "worker_deadline_s": deadline}
}

// runBudget is the run's resource budget (030): resolved from the toolchain catalog and the
// target profile the first time, stored in run.json, and read back from there afterwards, so a
// resumed run keeps the budget it started with even if the profile changed (FR-006).
func (p *P) runBudget(run *evidence.Run) images.Budget {
	if p.budget != nil {
		return *p.budget
	}
	var b images.Budget
	if raw, ok := run.State["budget"]; ok && raw != nil {
		if j, err := json.Marshal(raw); err == nil && json.Unmarshal(j, &b) == nil && b.Worker.DeadlineS > 0 {
			p.budget = &b
			return b
		}
	}
	if p.item() {
		b = images.Resolve(p.catalogEntry(), p.Target.Toolchain, p.Target)
	} else {
		b = images.Fixture()
	}
	var stored map[string]any
	if j, err := json.Marshal(b); err == nil {
		_ = json.Unmarshal(j, &stored)
	}
	run.State["budget"] = stored
	_ = run.Save()
	_ = run.Event("prepare", nil, nil, "budget_resolved", mustJSON(b))
	p.budget = &b
	return b
}

// workerLimits is the worker's part of the run's budget.
func (p *P) workerLimits(run *evidence.Run) worker.Limits {
	w := p.runBudget(run).Worker
	return worker.Limits{CPU: w.CPU, Memory: w.Memory, DeadlineS: w.DeadlineS, SoftDeadlineS: w.SoftDeadlineS()}
}

func modeJSON(m map[string]any) string { return decisions.PyJSON(mustJSON(m)) }

func mustJSON(v any) string { raw, _ := json.Marshal(v); return string(raw) }

func (p *P) confirmMode(run *evidence.Run) (bool, string) {
	p.runBudget(run)
	mode := p.ModeOf()
	run.State["proposed_mode"] = mode
	_ = run.Save()
	p.say("Run mode for this pilot: %s", modeJSON(mode))
	if extra := p.modeExtra(run); extra != "" { // an item run's target, toolchain and egress (014)
		p.say("%s", extra)
		run.State["proposed_target"] = p.modeExtraGitHub(run) // for the GitHub decision post (no model host count)
		_ = run.Save()
	}
	p.say("  one attempt at a time; every retry needs your confirmation; risk rating is out of scope in M1.")
	ok, ch := decisions.Decide(run, "mode", "Confirm this mode and start staging workers? [y/N] ", nil, nil, func(s string) { p.say("%s", s) })
	decisions.Note(run, "mode", nil, nil, ok, ch, "", "")
	p.githubTerminal(run, "mode", ok, ch)
	if !ok {
		_ = run.Event("mode", nil, nil, "mode_declined", modeJSON(mode))
		return false, ch
	}
	run.State["mode"] = mode
	_ = run.Event("mode", nil, nil, "mode_confirmed", modeJSON(mode))
	return true, ch
}

func (p *P) confirmRetry(run *evidence.Run, task string, n, total int) (bool, string) {
	note := deadlineNote(run, task, n-1)               // 030 FR-017
	if d, ok := attemptDiagnosis(run, task, n-1); ok { // 033 FR-005
		text := strings.TrimRight(d.Text, "\n")
		if note != "" {
			text = note + "\n" + text
		}
		note = text
	}
	if note != "" {
		p.say("%s", note)
		run.State["decision_note"] = map[string]any{"step": "retry", "text": note}
	} else {
		delete(run.State, "decision_note")
	}
	_ = run.Save()
	ok, ch := decisions.Decide(run, "retry", fmt.Sprintf("Retry %s (attempt %d of %d)? [y/N] ", task, n, total), task, n, func(s string) { p.say("%s", s) })
	decisions.Note(run, "retry", task, n, ok, ch, "", "")
	p.githubTerminal(run, "retry", ok, ch)
	ev := "retry_declined"
	if ok {
		ev = "retry_confirmed"
	}
	_ = run.Event("retry", task, n, ev, "")
	return ok, ch
}

func (p *P) modeNotConfirmed(run *evidence.Run, ch string) int {
	if ch == "none" {
		p.pause(run, "awaiting-mode", nil)
		p.say("paused: mode not confirmed. %s", decisions.Hint(run, "mode"))
		p.githubPost(run)
		return ExitDeclined
	}
	_ = run.SetRunState("failed", "MODE_NOT_CONFIRMED")
	_, _ = run.WriteSummary()
	return ExitDeclined
}

func (p *P) gateTask(run *evidence.Run, task string) gate.Decision {
	seq, _ := run.NextGateSeq()
	start := time.Now()
	defer func() { verbose.V(1, "gate %s (seq %d) took %s", task, seq, time.Since(start).Round(time.Millisecond)) }()
	d, _ := gate.EvaluateFresh(p.Status, p.Cfg.Repo(), p.Cfg.Branch(), task, paths.Tasks[task], seq, filepath.Join(run.Dir, "gate"),
		func(repo, branch string) (string, string, error) {
			return fixture.FreshClone(p.host(), repo, branch, 1)
		})
	t := run.Task(task)
	t["gate_seqs"] = append(anyList(t["gate_seqs"]), seq)
	if d.Decision == "allow" && d.Input.FixtureCommit != nil {
		t["base_revision"] = *d.Input.FixtureCommit
	}
	_ = run.Save()
	_ = run.Event("gate", task, nil, "gate_"+d.Decision, fmt.Sprintf("seq=%d %s", seq, strings.Join(d.Reasons, ",")))
	p.say("  gate %s: %s %s", task, d.Decision, pyrepr.Strings(d.Reasons))
	return d
}

func anyList(v any) []any { x, _ := v.([]any); return x }

func (p *P) cloneAt(rev string) (string, error) {
	dir, _, err := fixture.FreshClone(p.host(), p.repo(), p.branch(), 0)
	if err != nil {
		return "", err
	}
	if _, err := fixture.Git(p.host(), dir, "checkout", "--quiet", rev); err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	return dir, nil
}

func (p *P) pause(run *evidence.Run, state string, pending map[string]any) {
	if pending == nil {
		pending = map[string]any{}
	}
	run.State["pending"] = pending
	run.State["paused_at"] = evidence.Now()
	_ = run.SetRunState("paused("+state+")", nil)
	_, _ = run.WriteSummary()
}

func (p *P) fail(run *evidence.Run, reason string) int {
	p.dropGrading(run)
	_ = run.SetRunState("failed", reason)
	_, _ = run.WriteSummary()
	p.githubClose(run, "failed ("+reason+")")
	p.say("run %s FAILED: %s", run.ID, reason)
	return ExitFail
}

func (p *P) checks() checks.Deps { return checks.Deps{L: p.L, C: p.C, Out: p.Out} }

// workerChecks adds an item run's target proxies, so the worker checks include P06r (014 R6).
func (p *P) workerChecks(run *evidence.Run) checks.Deps {
	d := p.checks()
	if run.Kind() == evidence.KindItem && len(p.Target.RegistryHosts) > 0 {
		d.Net = &checks.ItemNet{WorkerProxyNS: cluster.WorkerProxyNS(p.Target.Name),
			PrefetchProxyNS: cluster.PrefetchProxyNS(p.Target.Name), Registries: p.Target.RegistryHosts}
	}
	return d
}

func (p *P) runTask(run *evidence.Run, task string, startN int) int {
	t := run.Task(task)
	total := 1 + p.Cfg.RetryLimit()
	n := startN
	for {
		t["state"] = "attempting"
		_ = run.Save()
		base, _ := t["base_revision"].(string)
		var att map[string]any
		clone, err := p.cloneAt(base)
		if err != nil {
			att = map[string]any{"n": n, "state": "failed", "reason": "harness_error"}
			_ = run.Event("attempt", task, n, "attempt_error", err.Error())
		} else {
			wd := worker.Deps{L: p.L, C: p.C, Runner: p.Runner, Limits: p.workerLimits(run),
				Checks:  func(task string, n int, ns string) { p.workerChecks(run).RunWorkerChecks(run, p.Imgs, task, n, ns) },
				Network: p.Network}
			if run.Kind() == evidence.KindItem {
				wd.Item = p.workerItem(run)
			}
			verbose.V(1, "attempt %s/%d: starting the worker", task, n)
			started := time.Now()
			att = worker.RunAttempt(wd, run, p.Cfg, p.Imgs, task, n, clone, base)
			_ = os.RemoveAll(clone)
			verbose.V(1, "attempt %s/%d: worker %v (%v) after %s", task, n, att["state"], att["reason"], time.Since(started).Round(time.Second))
		}
		if att["state"] == "collected" {
			started := time.Now()
			var g map[string]any
			if run.Kind() == evidence.KindItem {
				adir, _ := run.AttemptDir(task, n)
				g = grader.GradeItem(p.L, p.C, p.Runner, run, p.Imgs, task, n, p.itemGrading(run), filepath.Join(adir, "meta.json"), adir)
			} else {
				g = grader.Grade(p.L, p.C, p.Runner, run, p.Imgs, task, n)
			}
			logGrade(task, n, g, time.Since(started))
			att["result"] = g["result"]
			if g["result"] == "pass" {
				att["state"] = "passed"
			} else {
				att["state"] = "failed"
				if f, ok := g["failure"].(string); ok && f != "" {
					att["reason"] = f
				} else {
					att["reason"] = "grade_fail"
				}
			}
		} else {
			att["result"] = "fail"
		}
		var kept []any
		for _, x := range anyList(t["attempts"]) {
			if a, _ := x.(map[string]any); !numEq(a["n"], n) {
				kept = append(kept, a)
			}
		}
		t["attempts"] = append(kept, att)
		_ = run.Save()
		if att["state"] != "passed" {
			p.diagnoseAttempt(run, task, n, att) // 033 FR-001
		}
		if att["state"] == "passed" {
			t["passing_attempt"] = n
			break
		}
		if task == "A" {
			if d := p.gateTask(run, "B"); d.Decision != "refuse" {
				return p.fail(run, "B_NOT_BLOCKED")
			}
		}
		if n >= total {
			return p.fail(run, task+"_RETRIES_EXHAUSTED")
		}
		ok, ch := p.confirmRetry(run, task, n+1, total)
		if !ok {
			p.pause(run, "awaiting-retry", map[string]any{"task": task, "n": n + 1})
			msg := "`./chalkeus forge resume` asks again."
			if ch == "none" {
				msg = decisions.Hint(run, "retry")
			}
			p.say("paused: retry of %s not confirmed. %s", task, msg)
			if ch == "none" {
				p.githubPost(run)
			}
			return ExitAwaitRetry
		}
		n++
	}
	return p.publishTask(run, task)
}

func numEq(v any, n int) bool {
	switch x := v.(type) {
	case float64:
		return int(x) == n
	case int:
		return x == n
	case json.Number:
		i, err := x.Int64()
		return err == nil && int(i) == n
	}
	return false
}

func asInt(v any) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	case json.Number:
		i, _ := x.Int64()
		return int(i)
	}
	return 0
}

func (p *P) publisherDeps(run *evidence.Run) (publisher.Deps, error) {
	// 037: the publisher's proxy allows the run's Git host only (P13 proves it after the publish).
	if p.C != nil {
		if recreated, err := cluster.EnsureSCMProxy(p.L, p.C, p.host(), p.Imgs["egress"]); err != nil {
			return publisher.Deps{}, err
		} else if recreated {
			p.say("publisher egress now allows %s only", p.host().Name)
		}
	}
	d := publisher.Deps{L: p.L, C: p.C, Runner: p.Runner, Repo: p.repo(), Branch: p.branch(), Host: p.host(), Closes: p.closes(run)}
	if p.item() { // the cluster's scm-token was set at `up` for the fixture: put the target's in (014)
		d.RefreshSecret = p.refreshSecret
	}
	if p.Creds.App { // App installation tokens cannot call GET /user; the host passes the bot login (004 US4)
		login, err := p.Creds.BotLogin()
		if err != nil {
			return d, err
		}
		d.LoginHint, d.RefreshSecret = login, p.refreshSecret
	}
	return d, nil
}

func (p *P) publishTask(run *evidence.Run, task string) int {
	t := run.Task(task)
	n := asInt(t["passing_attempt"])
	t["state"] = "publishing"
	_ = run.Save()
	deps, err := p.publisherDeps(run)
	if err == nil && run.Kind() == evidence.KindItem {
		err = p.replay(run, task, n)
	}
	var pr map[string]any
	if err == nil {
		pr, err = deps.Publish(run, p.Imgs, task, n)
	}
	if err != nil {
		t["state"] = "publish-failed"
		msg := err.Error()
		if len(msg) > 300 {
			msg = msg[:300]
		}
		_ = run.Event("publish", task, n, "publish_failed", fmt.Sprintf("%s: %s", errName(err), msg))
		p.pause(run, "awaiting-publish", map[string]any{"task": task, "n": n})
		p.say("publisher failed for %s: %v\n`./chalkeus forge resume` re-runs it (idempotent).", task, err)
		return ExitFail
	}
	t["state"] = "published"
	t["pr"] = pr
	_ = run.Save()
	p.checks().RunClusterChecks(run)
	p.checks().RunPublisherEgressCheck(run, p.Imgs["tools"], p.host()) // 037 P13
	if task == "A" {
		if d := p.gateTask(run, "B"); d.Decision != "refuse" {
			return p.fail(run, "B_NOT_BLOCKED")
		}
		t["state"] = "awaiting-merge"
		p.pause(run, "awaiting-merge", map[string]any{"task": "A"})
		p.say("\nTask A draft PR: %v\nReview and merge it on GitHub yourself, then run: ./chalkeus forge resume", pr["url"])
		p.ensureWatcher(run)
		return ExitOK
	}
	if run.Kind() == evidence.KindItem {
		p.dropGrading(run) // grading is over once the item's PR is open
		t["state"] = "awaiting-merge"
		p.pause(run, "awaiting-merge", map[string]any{"task": task})
		p.say("\nItem %s draft PR: %v\nReview and merge it on GitHub yourself, then run: ./chalkeus forge resume --run %s", task, pr["url"], run.ID)
		p.ensureWatcher(run)
		return ExitOK
	}
	return p.finish(run)
}

func errName(err error) string {
	if _, ok := err.(*publisher.Error); ok {
		return "PublishError"
	}
	return "Error"
}

func (p *P) finish(run *evidence.Run) int {
	p.githubRefresh(run, true)
	ok, err := audit.Run(run, false, p.Cfg, p.Out)
	if err != nil || !ok {
		return p.fail(run, "AUDIT")
	}
	_ = run.SetRunState("succeeded", nil)
	_, _ = run.WriteSummary()
	p.githubClose(run, "succeeded")
	rel, _ := filepath.Rel(p.L.Root, run.Dir)
	p.say("run %s succeeded. Evidence: %s", run.ID, rel)
	return ExitOK
}

// Run starts a new run.
func (p *P) Run() int {
	imgs, err := images.Load(p.L)
	if err != nil {
		p.say("chalkeus: %v", err)
		return ExitError
	}
	p.Imgs = imgs
	sc, err := hosts.Operator(p.host())
	if err != nil {
		p.say("chalkeus: %v", err)
		return ExitError
	}
	open, err := sc.OpenChangeRequests(context.Background(), p.Cfg.Repo(), "pilot/")
	if err != nil {
		p.say("chalkeus: %v", err)
		return ExitError
	}
	if len(open) > 0 {
		var urls []string
		for _, o := range open {
			urls = append(urls, o.URL)
		}
		p.say("open pilot PRs exist on %s: %s\nrun `./chalkeus forge reset-fixture` first", p.Cfg.Repo(), pyrepr.Strings(urls))
		return ExitError
	}
	base, err := os.ReadFile(p.L.FixtureBase)
	if err != nil {
		p.say("no fixture base recorded; run `./chalkeus forge reset-fixture` first")
		return ExitError
	}
	run, err := evidence.NewRun(p.L.EvidenceRoot())
	if err != nil {
		p.say("chalkeus: %v", err)
		return ExitError
	}
	p.say("== run %s", run.ID)
	run.State["fixture_base"] = strings.TrimSpace(string(base))
	h := p.Target.Host()
	run.SetSCM(string(h.KindOrGitHub()), h.Name)     // 037
	if hosts := p.Cfg.ModelHosts(); len(hosts) > 0 { // for the P06 probe; never in evidence
		_ = os.WriteFile(p.L.ModelHost, []byte(hosts[0]+"\n"), 0o644)
	}
	_ = evidence.WriteConfigJSON(run, p.Cfg.Public(), p.Cfg.ConfigDir, append(p.Cfg.ModelFiles(), p.Cfg.PublisherEnv))
	pins, _ := paths.ReadEnv(p.L.Pins)
	_ = evidence.WriteVersions(run, pins, p.L.Build, paths.Context, paths.Cluster, imgs)
	_ = run.Event("run", nil, nil, "run_started", p.Cfg.Repo())
	if p.issuesOn() { // 004 US1: mirror the roadmap before anything is staged
		l, err := p.loop()
		if err == nil {
			_, err = l.Sync(run)
		}
		if err != nil {
			p.say("github sync failed: %v", err)
			return p.fail(run, "GITHUB_SYNC")
		}
	}
	p.checks().GateMatrix(run, p.Status)
	rows, _ := evidence.ReadJSONL(filepath.Join(run.Dir, "checks", "results.jsonl"))
	for _, r := range rows {
		if id, _ := r["id"].(string); strings.HasPrefix(id, "GM") && r["result"] != "pass" {
			return p.fail(run, "GATE_MATRIX")
		}
	}
	if d := p.gateTask(run, "A"); d.Decision != "allow" {
		return p.fail(run, "A_REFUSED")
	}
	if d := p.gateTask(run, "B"); d.Decision != "refuse" {
		return p.fail(run, "B_NOT_BLOCKED")
	}
	ok, ch := p.confirmMode(run)
	if !ok {
		return p.modeNotConfirmed(run, ch)
	}
	_ = run.Save()
	return p.runTask(run, "A", 1)
}

// Resume continues a paused run.
func (p *P) Resume(runID string) int {
	run, err := evidence.LoadRun(p.L.EvidenceRoot(), runID)
	state := "None"
	if run != nil {
		state = fmt.Sprint(run.State["state"])
	}
	if err != nil || run == nil || !strings.HasPrefix(state, "paused") {
		suffix := ""
		if runID != "" {
			suffix = " " + runID
		}
		p.say("no paused run%s (state=%s)", suffix, state)
		return ExitError
	}
	if p.approvalsOn() { // one resumer at a time: the run's watcher, or the operator after --stop
		if w := watcher.Status(run.Dir, run.ID); w.Running && w.Pid != os.Getpid() {
			p.say("a watcher is resuming run %s on its own (pid %d). To resume it yourself, first run: ./chalkeus forge github watch --run %s --stop",
				run.ID, w.Pid, run.ID)
			return ExitError
		}
	}
	if run.State["impl"] != "go" {
		p.say("this run was started by the Python pilot, which was removed in 016; reset the fixture and start a new run")
		return ExitError
	}
	imgs, err := images.Load(p.L)
	if err != nil {
		p.say("chalkeus: %v", err)
		return ExitError
	}
	p.Imgs = imgs
	p.collectGitHub(run) // 004: a valid /approve on GitHub becomes the approval
	pend, _ := run.State["pending"].(map[string]any)
	run.State["state"] = "in-progress"
	_ = run.Save()
	_ = run.Event("run", nil, nil, "run_resumed", state)
	switch state {
	case "paused(awaiting-mode)":
		ok, ch := p.confirmMode(run)
		if !ok {
			return p.modeNotConfirmed(run, ch)
		}
		_ = run.Save()
		if run.Kind() == evidence.KindItem {
			return p.startItem(run)
		}
		return p.runTask(run, "A", 1)
	case "paused(awaiting-retry)":
		task, _ := pend["task"].(string)
		n := asInt(pend["n"])
		if decisions.Recorded(run, decisions.Replan, task, n) { // 033: the owner chose back to planning
			return p.stopForPlanning(run, task, n)
		}
		ok, ch := p.confirmRetry(run, task, n, 1+p.Cfg.RetryLimit())
		if !ok {
			p.pause(run, "awaiting-retry", map[string]any{"task": task, "n": n})
			if ch == "none" {
				p.say("paused: retry of %s not confirmed. %s", task, decisions.Hint(run, "retry"))
				p.githubPost(run)
			}
			return ExitAwaitRetry
		}
		return p.runTask(run, task, n)
	case "paused(awaiting-publish)":
		task, _ := pend["task"].(string)
		return p.publishTask(run, task)
	case "paused(awaiting-acceptance)": // 026: back to the merge check, then the acceptance decision
		if p.approvalsOn() {
			return p.reconcileGitHub(run, pend)
		}
		return p.reconcileA(run)
	case "paused(awaiting-merge)":
		if p.approvalsOn() { // 004 R6: the status change is approved by merging a status-change PR
			return p.reconcileGitHub(run, pend)
		}
		return p.reconcileA(run)
	}
	p.say("cannot resume from %s", state)
	return ExitError
}

func (p *P) reconcileA(run *evidence.Run) int {
	pausedAt, err := time.Parse(evidence.TimeFormat, fmt.Sprint(run.State["paused_at"]))
	if err != nil {
		pausedAt = time.Now().UTC()
	}
	task := mergeTask(run)
	tA := run.Task(task)
	n := asInt(tA["passing_attempt"])
	channel := ""
	ask := func(q string) string {
		ok, ch := decisions.Decide(run, "reconcile", q, task, nil, func(s string) { p.say("%s", s) })
		decisions.Note(run, "reconcile", task, nil, ok, ch, "", "")
		channel = ch
		if ok {
			return "y"
		}
		return ""
	}
	adir, _ := run.AttemptDir(task, n)
	base, _ := tA["base_revision"].(string)
	rec, cleanup, err := reconcile.ForFixture(p.host(), p.repo(), p.branch(), run.Dir, run.ID, task, adir, base, ask, func(s string) { p.say("%s", s) })
	if err != nil {
		return p.fail(run, "RECONCILE_SETUP")
	}
	var changed []string
	for _, x := range anyList(tA["attempts"]) {
		a, _ := x.(map[string]any)
		if numEq(a["n"], n) {
			for _, c := range anyList(a["changed_paths"]) {
				changed = append(changed, fmt.Sprint(c))
			}
		}
	}
	pr, _ := tA["pr"].(map[string]any)
	if p.item() {
		rec.Where = p.repo() + "'s roadmap"
	}
	if !SlicesLast(run) { // 026: a slice before the item's last: no status change
		return p.sliceMerged(run, rec, cleanup, task, n, changed, "terminal")
	}
	if len(acceptanceChecks(run)) > 0 && run.State["acceptance_recorded"] != true { // 026 FR-018
		if c, _, _ := rec.Check(task, run.ItemOf(task), asInt(pr["number"]), changed); c == 0 {
			if goOn, ec := p.acceptanceGate(run, task); !goOn {
				cleanup()
				return ec
			}
		}
	}
	code, r := rec.Run(task, run.ItemOf(task), asInt(pr["number"]), changed)
	cleanup()
	_ = evidence.WriteJSON(filepath.Join(run.Dir, "reconcile", task+".json"), r)
	reason, _ := r["reason"].(string)
	evName := "reconcile_applied"
	if reason != "" {
		evName = "reconcile_" + reason
	}
	applied, _ := r["applied_commit"].(string)
	_ = run.Event("reconcile", task, n, evName, applied)
	switch code {
	case ExitAwaitMerge:
		p.pause(run, "awaiting-merge", map[string]any{"task": task})
		if task == "A" {
			p.say("A's PR is not merged yet. Merge it on GitHub, then run `./chalkeus forge resume` again.") // the guide quotes this
		} else {
			p.say("%s's PR is not merged yet. Merge it on GitHub, then run `./chalkeus forge resume` again.", task)
		}
		return ExitAwaitMerge
	case ExitDeclined:
		p.pause(run, "awaiting-merge", map[string]any{"task": task})
		if channel == "none" {
			p.say("waiting for your approval of the status change; roadmap unchanged. %s", decisions.Hint(run, "reconcile"))
		} else {
			p.say("status change declined; roadmap unchanged; B stays blocked.")
		}
		return ExitDeclined
	case 0:
	default:
		return p.fail(run, "RECONCILE_"+reason)
	}
	wait := int(time.Now().UTC().Sub(pausedAt).Seconds())
	_ = run.Event("reconcile", task, nil, "merge_wait_seconds", strconv.Itoa(wait))
	tA["state"] = "verified"
	_ = run.Save()
	_ = run.Event("status", task, nil, "task_verified", applied)
	p.githubRefresh(run, false)
	if run.Kind() == evidence.KindItem { // one item: nothing follows its verification (014)
		return p.finish(run)
	}
	if d := p.gateTask(run, "B"); d.Decision != "allow" {
		return p.fail(run, "B_STILL_BLOCKED")
	}
	return p.runTask(run, "B", 1)
}

// logGrade prints the grade and, at -v, the first check that did not pass.
func logGrade(task string, n int, g map[string]any, took time.Duration) {
	verbose.V(1, "attempt %s/%d: grade %v after %s", task, n, g["result"], took.Round(time.Second))
	checks, _ := g["checks"].([]any)
	for _, x := range checks {
		c, _ := x.(map[string]any)
		if c["result"] != "pass" {
			d := fmt.Sprint(c["detail"])
			if len(d) > 300 {
				d = d[:300] + "…"
			}
			verbose.V(1, "  %v %v: %s", c["id"], c["result"], d)
			return
		}
	}
}

// deadlineNote describes attempt n of task when it stopped at its deadline (030 FR-017): how long
// it ran, its usage and what it changed; "" for any other attempt.
func deadlineNote(run *evidence.Run, task string, n int) string {
	tasks, _ := run.State["tasks"].(map[string]any)
	t, _ := tasks[task].(map[string]any)
	for _, x := range anyList(t["attempts"]) {
		a, _ := x.(map[string]any)
		if !numEq(a["n"], n) || a["reason"] != "deadline" {
			continue
		}
		mode, _ := run.State["mode"].(map[string]any)
		secs := fmt.Sprint(mode["worker_deadline_s"])
		sp, _ := a["spend"].(map[string]any)
		if sp == nil {
			return fmt.Sprintf("  attempt %d stopped at the %s s deadline; its usage and changes weren't collected.", n, secs)
		}
		changed := len(anyList(a["changed_paths"]))
		return fmt.Sprintf("  attempt %d stopped at the %s s deadline: %v tokens, $%v; changed %d file(s) (kept in the attempt's evidence, not graded).",
			n, secs, sp["tokens"], sp["cost_usd"], changed)
	}
	return ""
}
