// Package publisher publishes a passing attempt as one draft PR through the publisher Job
// (reference: the former Python pilot's publisher.py; FR-012–FR-016). There is no merge path.
package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tosin2013/specfoundry/internal/evidence"
	"github.com/tosin2013/specfoundry/internal/frame"
	"github.com/tosin2013/specfoundry/internal/ghloop"
	"github.com/tosin2013/specfoundry/internal/k8s"
	"github.com/tosin2013/specfoundry/internal/paths"
	"github.com/tosin2013/specfoundry/internal/runner"
)

// Timeout is the publisher Job's budget in seconds.
const Timeout = 300

// Error is a publisher problem (resumable; never "verified").
type Error struct{ Msg string }

func (e *Error) Error() string { return e.Msg }

// Deps are the publisher's collaborators.
type Deps struct {
	L      paths.Layout
	C      *k8s.Client
	Runner runner.Runner
	Repo   string
	Branch string
	// RefreshSecret puts a fresh credential in scm-token before each publish (App mode); may be nil.
	RefreshSecret func() error
	// LoginHint is the bot login passed to the Job for App tokens ("" for fine-grained tokens).
	LoginHint string
	// Closes returns the item issue a task's PR closes (004), or 0.
	Closes func(task string) (int, error)
}

// PRs reads prs.json.
func PRs(run *evidence.Run) []map[string]any {
	var out []map[string]any
	if raw, err := os.ReadFile(filepath.Join(run.Dir, "prs.json")); err == nil {
		_ = evidence.Decode(raw, &out)
	}
	return out
}

func orDash(v any) string {
	if v == nil {
		return "-"
	}
	if s, ok := v.(string); ok && s == "" {
		return "-"
	}
	return fmt.Sprint(v)
}

func body(l paths.Layout, run *evidence.Run, task string, n int, closes int) string {
	st := run.Task(task)
	var rows []string
	for _, x := range anyList(st["attempts"]) {
		a, _ := x.(map[string]any)
		sp, _ := a["spend"].(map[string]any)
		res := "-"
		if r, ok := a["result"]; ok {
			res = fmt.Sprint(r)
		}
		rows = append(rows, fmt.Sprintf("| %v | %s | %s | %s | %s |", a["n"], orDash(a["induction"]), res,
			orDashKey(sp, "cost_usd"), orDashKey(sp, "tokens")))
	}
	var gates []string
	for _, g := range anyList(st["gate_seqs"]) {
		gates = append(gates, fmt.Sprint(g))
	}
	gs := strings.Join(gates, ", ")
	if gs == "" {
		gs = "-"
	}
	var imgs map[string]string
	if raw, err := os.ReadFile(l.ImagesJSON); err == nil {
		_ = json.Unmarshal(raw, &imgs)
	}
	if run.Kind() == evidence.KindItem {
		return itemBody(run, task, n, gs, rows, imgs, closes)
	}
	b := fmt.Sprintf("Opened by the specfoundry M1 pilot **publisher** (run `%s`, task %s, fixture item %s).\n\n"+
		"Passing attempt: **%d**. Gate decisions: %s.\n\n"+
		"| Attempt | Induction | Grade | Cost (USD) | Tokens |\n|---|---|---|---|---|\n%s\n\n"+
		"Images: worker `%s`, grader/publisher `%s`.\n\n"+
		"Evidence: `specs/002-m1-gated-pilot/evidence/%s/` in the specfoundry repository.\n\n"+
		"This is a **draft**. A human reviews and merges it; the pilot never merges.\n",
		run.ID, task, paths.Tasks[task], n, gs, strings.Join(rows, "\n"), imgs["worker"], imgs["tools"], run.ID)
	if closes > 0 {
		b += fmt.Sprintf("\nCloses #%d\n", closes)
	}
	return b
}

func orDashKey(m map[string]any, k string) string {
	if m == nil {
		return "-"
	}
	v, ok := m[k]
	if !ok {
		return "-"
	}
	return fmt.Sprint(v)
}

func anyList(v any) []any { x, _ := v.([]any); return x }

func (d Deps) submit(run *evidence.Run, imgs map[string]string, key string, spec map[string]any, bundle string) (map[string]any, error) {
	if d.RefreshSecret != nil {
		if err := d.RefreshSecret(); err != nil {
			return nil, err
		}
	}
	cm := "pub-" + key
	short := strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(run.ID), "t", ""), "z", "")
	if len(short) > 8 {
		short = short[len(short)-8:]
	}
	job := fmt.Sprintf("publish-%s-%s", key, short)
	_ = d.C.DeleteConfigMap("pilot-publish", cm)
	_ = d.C.DeleteJob("pilot-publish", job, true)
	tmp, err := os.MkdirTemp("", "pilot-pub-")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	raw, _ := json.Marshal(spec)
	if err := os.WriteFile(filepath.Join(tmp, "pr.json"), raw, 0o644); err != nil {
		return nil, err
	}
	if err := d.C.CreateConfigMap("pilot-publish", cm, map[string]string{"pr.json": filepath.Join(tmp, "pr.json"), "change.bundle": bundle}, nil); err != nil {
		return nil, err
	}
	y, err := k8s.Render(filepath.Join(d.L.Manifests, "publisher-job.yaml.tmpl"), map[string]string{"JOB": job, "CONFIGMAP": cm, "TOOLS_IMAGE": imgs["tools"]})
	if err != nil {
		return nil, err
	}
	h, err := d.Runner.Submit("publisher", "pilot-publish", job, y, false)
	if err != nil {
		return nil, err
	}
	outcome := d.Runner.Wait(h, (Timeout+60)*time.Second)
	files, ferr := frame.Decode(d.Runner.Collect(h), "result.json")
	status := d.Runner.Status(h)
	d.Runner.Delete(h)
	_ = d.C.DeleteConfigMap("pilot-publish", cm)
	if ferr != "" || outcome != "Complete" {
		detail := ferr
		if files != nil {
			var r map[string]any
			_ = json.Unmarshal(files["result.json"], &r)
			if e, ok := r["error"].(string); ok && e != "" {
				detail = e
			}
		}
		if detail == "" {
			detail = outcome
		}
		return nil, &Error{fmt.Sprintf("publisher %s: %s (%v)", outcome, detail, status["pod_reason"])}
	}
	var res map[string]any
	if err := evidence.Decode(files["result.json"], &res); err != nil {
		return nil, &Error{"publisher result unreadable"}
	}
	if e, ok := res["error"].(string); ok && e != "" {
		return nil, &Error{e}
	}
	return res, nil
}

// ItemBase is an item run's real base revision (the snapshot's tree is its tree).
func ItemBase(run *evidence.Run) string {
	snap, _ := run.State["snapshot"].(map[string]any)
	s, _ := snap["base_revision"].(string)
	return s
}

func baseRevision(adir string) string {
	var meta map[string]any
	if raw, err := os.ReadFile(filepath.Join(adir, "meta.json")); err == nil {
		_ = json.Unmarshal(raw, &meta)
	}
	s, _ := meta["base_revision"].(string)
	return s
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

// Publish opens (or finds) the task's draft PR. Idempotent: an existing task PR is returned.
func (d Deps) Publish(run *evidence.Run, imgs map[string]string, task string, n int) (map[string]any, error) {
	adir, err := run.AttemptDir(task, n)
	if err != nil {
		return nil, err
	}
	var g map[string]any
	raw, _ := os.ReadFile(filepath.Join(adir, "grade.json"))
	_ = json.Unmarshal(raw, &g)
	if g["result"] != "pass" {
		return nil, &Error{fmt.Sprintf("attempt %s/%d did not pass grading (FR-012)", task, n)}
	}
	for _, p := range PRs(run) {
		kind, _ := p["kind"].(string)
		if p["task"] == task && (kind == "" || kind == "task") {
			return p, nil
		}
	}
	closes := 0
	if d.Closes != nil {
		if closes, err = d.Closes(task); err != nil {
			return nil, err
		}
	}
	head := fmt.Sprintf("pilot/%s/%s", run.ID, strings.ToLower(task))
	base, bundle, title := baseRevision(adir), filepath.Join(adir, "change.bundle"), fmt.Sprintf("pilot %s task %s", run.ID, task)
	if run.Kind() == evidence.KindItem { // the change replayed onto the real base (014 R4)
		base, bundle, title = ItemBase(run), filepath.Join(adir, "publish.bundle"), fmt.Sprintf("pilot %s: item %s", run.ID, task)
	}
	spec := map[string]any{"repo": d.Repo, "head": head, "base": d.Branch, "base_revision": base,
		"title": title, "body": body(d.L, run, task, n, closes), "run_id": run.ID, "task": task}
	if d.LoginHint != "" {
		spec["publisher_login"] = d.LoginHint
	}
	_ = run.Event("publish", task, n, "publish_started", head)
	res, err := d.submit(run, imgs, strings.ToLower(task), spec, bundle)
	if err != nil {
		return nil, err
	}
	gh, err := ghloop.OperatorClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	owner, name := ghloop.SplitRepo(d.Repo)
	number := asInt(res["number"])
	pr, _, err := gh.PullRequests.Get(ctx, owner, name, number)
	if err != nil {
		return nil, err
	}
	me, _, err := gh.Users.Get(ctx, "")
	if err != nil {
		return nil, err
	}
	var closesV any
	if closes > 0 {
		closesV = closes
	}
	entry := map[string]any{"task": task, "number": number, "url": pr.GetHTMLURL(), "head": pr.GetHead().GetRef(),
		"draft": pr.GetDraft(), "author": pr.GetUser().GetLogin(), "publisher_identity": res["token_login"],
		"operator_login": me.GetLogin(), "attempt": n, "created": res["created"] == true, "kind": "task", "closes": closesV}
	if !pr.GetDraft() || pr.GetHead().GetRef() != head {
		return nil, &Error{fmt.Sprintf("PR #%d is not the expected draft for %s", number, head)}
	}
	all := append(PRs(run), entry)
	if err := evidence.WriteJSON(filepath.Join(run.Dir, "prs.json"), all); err != nil {
		return nil, err
	}
	ev := "pr_found"
	if entry["created"] == true {
		ev = "pr_opened"
	}
	_ = run.Event("publish", task, n, ev, pr.GetHTMLURL())
	return entry, nil
}

// itemBody is an item run's PR body (014): the target and item instead of the fixture wording,
// and the run's evidence location on the operator's machine.
func itemBody(run *evidence.Run, task string, n int, gs string, rows []string, imgs map[string]string, closes int) string {
	b := fmt.Sprintf("Opened by the specfoundry pilot **publisher** (run `%s`, item %s of `%s`).\n\n"+
		"Passing attempt: **%d**. Gate and readiness decisions: %s.\n\n"+
		"| Attempt | Induction | Grade | Cost (USD) | Tokens |\n|---|---|---|---|---|\n%s\n\n"+
		"Images: worker `%s`, grader/publisher `%s`.\n\n"+
		"Graded independently: the project's own tests and hidden acceptance tests the worker never saw. "+
		"Evidence: run `%s` on the operator's machine.\n\n"+
		"This is a **draft**. A human reviews and merges it; the pilot never merges.\n",
		run.ID, task, run.TargetRepo(), n, gs, strings.Join(rows, "\n"), imgs["worker"], imgs["tools"], run.ID)
	if closes > 0 {
		b += fmt.Sprintf("\nCloses #%d\n", closes)
	}
	return b
}

// StatusBody is the status-change PR body (004 R6). An item run names the status it starts from
// (014 R12: planned, specced or in-progress).
func StatusBody(run *evidence.Run, task string, closes int) string {
	item := run.ItemOf(task)
	from, _ := run.State["status_from"].(string)
	if from == "" {
		from = "planned"
	}
	issue := ""
	if closes > 0 {
		issue = fmt.Sprintf(" Item issue: #%d.", closes)
	}
	return fmt.Sprintf("Opened by the specfoundry pilot **publisher** (run `%s`) after task %s's PR was merged "+
		"and checked (merged, tasks 100%%, merged tree matches the graded attempt).\n\n"+
		"It changes exactly one line: item %s `"+from+"` → `verified` in `.specify/memory/roadmap.md`.%s\n\n"+
		"**Merging this PR approves the status change.** The pilot checks that the merged change is exactly "+
		"this edit, and that an allowed approver merged it. The pilot never merges.\n", run.ID, task, item, issue)
}

// PublishStatus opens (or finds) the status-change draft PR for task's item (004 R6).
// Idempotent like task PRs. closes is the item issue number (0 if none).
func (d Deps) PublishStatus(run *evidence.Run, imgs map[string]string, task, baseRev, bundle string, closes int) (map[string]any, error) {
	for _, p := range PRs(run) {
		if p["kind"] == "status" && p["task"] == task {
			return p, nil
		}
	}
	item := run.ItemOf(task)
	head := fmt.Sprintf("pilot/%s/status-%s", run.ID, item)
	spec := map[string]any{"repo": d.Repo, "head": head, "base": d.Branch, "base_revision": baseRev,
		"title": fmt.Sprintf("pilot %s: %s verified", run.ID, item), "body": StatusBody(run, task, closes), "run_id": run.ID, "task": task}
	if d.LoginHint != "" {
		spec["publisher_login"] = d.LoginHint
	}
	_ = run.Event("publish", task, nil, "status_pr_started", head)
	res, err := d.submit(run, imgs, "status-"+strings.ToLower(task), spec, bundle)
	if err != nil {
		return nil, err
	}
	gh, err := ghloop.OperatorClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	owner, name := ghloop.SplitRepo(d.Repo)
	number := asInt(res["number"])
	pr, _, err := gh.PullRequests.Get(ctx, owner, name, number)
	if err != nil {
		return nil, err
	}
	entry := map[string]any{"task": task, "kind": "status", "number": number, "url": pr.GetHTMLURL(), "head": pr.GetHead().GetRef(),
		"draft": pr.GetDraft(), "author": pr.GetUser().GetLogin(), "publisher_identity": res["token_login"],
		"base_revision": baseRev, "created": res["created"] == true}
	if !pr.GetDraft() || pr.GetHead().GetRef() != head {
		return nil, &Error{fmt.Sprintf("PR #%d is not the expected draft for %s", number, head)}
	}
	if err := evidence.WriteJSON(filepath.Join(run.Dir, "prs.json"), append(PRs(run), entry)); err != nil {
		return nil, err
	}
	_ = run.Event("publish", task, nil, "status_pr_opened", pr.GetHTMLURL())
	return entry, nil
}
