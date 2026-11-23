package pipeline

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/tosin2013/chalkeus/internal/decisions"
	"github.com/tosin2013/chalkeus/internal/evidence"
	"github.com/tosin2013/chalkeus/internal/readiness"
	"github.com/tosin2013/chalkeus/internal/reconcile"
)

// 026: what happens after a slice's PR merges, and the item-level acceptance decision.

// nextSliceID is the slice after this run's, from the readiness record ("" if none or unknown).
func nextSliceID(run *evidence.Run) string {
	r, _ := run.State["readiness"].(map[string]any)
	var all []string
	switch s := r["slice"].(type) {
	case *readiness.SliceInfo:
		all = s.All
	case map[string]any:
		for _, x := range anyList(s["all"]) {
			all = append(all, fmt.Sprint(x))
		}
	}
	for i, id := range all {
		if id == run.Slice() && i+1 < len(all) {
			return all[i+1]
		}
	}
	return ""
}

// sliceMerged handles the merge of a slice that isn't the item's last (026 R7): it checks the
// merge (CheckSlice), records slice_merged, proposes no status, and finishes the run. channel is
// "github" or "terminal" (for the reconcile record).
func (p *P) sliceMerged(run *evidence.Run, rec reconcile.Reconciler, task string, n int, changed []string, channel string) int {
	tA := run.Task(task)
	pr, _ := tA["pr"].(map[string]any)
	id, phases, _ := sliceOf(run)
	code, r := rec.CheckSlice(task, run.ItemOf(task), id, phases, asInt(pr["number"]), changed)
	r["channel"] = channel
	_ = evidence.WriteJSON(filepath.Join(run.Dir, "reconcile", task+".json"), r)
	reason, _ := r["reason"].(string)
	mc, _ := r["merge_commit"].(string)
	ev := "slice_merged"
	if reason != "" {
		ev = "reconcile_" + reason
	}
	_ = run.Event("reconcile", task, n, ev, mc)
	switch code {
	case ExitAwaitMerge:
		p.pause(run, "awaiting-merge", map[string]any{"task": task})
		p.say("%s's slice %s PR is not merged yet. Merge it on GitHub, then run `./chalkeus forge resume` (or `./chalkeus forge github watch`).", task, id)
		p.ensureWatcher(run)
		return ExitAwaitMerge
	case 0:
	default:
		return p.fail(run, "RECONCILE_"+reason)
	}
	tA["state"] = "merged"
	_ = run.Save()
	_ = run.Event("status", task, nil, "slice_merged", id)
	if next := nextSliceID(run); next != "" {
		p.say("slice %s merged; next: --slice %s", id, next)
	} else {
		p.say("slice %s merged", id)
	}
	return p.finish(run)
}

// acceptanceChecks are the item-level acceptance checks the item's spec declares (FR-018).
func acceptanceChecks(run *evidence.Run) []string {
	r, _ := run.State["readiness"].(map[string]any)
	var out []string
	for _, x := range anyList(r["acceptance"]) {
		out = append(out, fmt.Sprint(x))
	}
	return out
}

// acceptanceGate holds the status proposal until a human records the item's acceptance checks
// (026 R7), through the same decision channels as every other decision: the terminal, `pilot
// approve acceptance`, or `/approve acceptance` on GitHub. It reports whether to go on; when not,
// code is the run's exit code (the run is paused at paused(awaiting-acceptance)).
func (p *P) acceptanceGate(run *evidence.Run, task string) (bool, int) {
	checks := acceptanceChecks(run)
	if len(checks) == 0 || run.State["acceptance_recorded"] == true {
		return true, 0
	}
	q := fmt.Sprintf("Item %s declares acceptance checks:\n  - %s\nHave they been done and passed? [y/N] ",
		run.ItemOf(task), strings.Join(checks, "\n  - "))
	ok, ch := decisions.Decide(run, "acceptance", q, task, nil, func(s string) { p.say("%s", s) })
	decisions.Note(run, "acceptance", task, nil, ok, ch, "", "")
	if ok {
		run.State["acceptance_recorded"] = true
		_ = run.Save()
		_ = run.Event("status", task, nil, "acceptance_recorded", ch)
		return true, 0
	}
	p.pause(run, "awaiting-acceptance", map[string]any{"task": task})
	_ = run.Event("status", task, nil, "acceptance_wait", strings.Join(checks, "; "))
	if ch == "tty" {
		p.say("acceptance not recorded; the status change waits. Run `./chalkeus forge resume` when the checks are done.")
		return false, ExitDeclined
	}
	p.say("waiting for the item's acceptance checks. %s", decisions.Hint(run, "acceptance"))
	p.githubPost(run)
	return false, ExitAwaitMerge
}
