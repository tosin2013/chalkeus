package pipeline

import (
	"os"
	"os/exec"

	"github.com/tosin2013/chalkeus/internal/decisions"
	"github.com/tosin2013/chalkeus/internal/diagnose"
	"github.com/tosin2013/chalkeus/internal/evidence"
	"github.com/tosin2013/chalkeus/internal/watcher"
)

// diagnoseAttempt builds and records the diagnosis of a failed attempt (033 FR-001): it writes
// diagnosis.json into the attempt's evidence and the recommendation into the attempt record. It
// never acts on the recommendation (FR-007); a failure to diagnose is recorded, not fatal.
func (p *P) diagnoseAttempt(run *evidence.Run, task string, n int, att map[string]any) {
	dir, err := run.AttemptDir(task, n)
	if err != nil {
		return
	}
	in, err := diagnose.FromDir(dir, task, att, runAllowedPaths(run), runToolchain(run),
		runTestSeconds(run), runWorkerDeadline(run))
	if err != nil {
		_ = run.Event("attempt", task, n, "attempt_diagnosis_error", err.Error())
		return
	}
	in.Extensions = strList(run.State["spec_extensions"]) // US5 FR-019
	in.PrevDeadlineNoTrig = prevDeadlineNoTrigger(run, task, n)
	d := diagnose.Diagnose(in)
	if err := diagnose.Write(dir, d); err != nil {
		_ = run.Event("attempt", task, n, "attempt_diagnosis_error", err.Error())
		return
	}
	att["diagnosis"] = map[string]any{"recommend": d.Recommend, "triggers": d.Triggers}
	_ = run.Save()
	_ = run.Event("attempt", task, n, "attempt_diagnosed", d.Recommend)
}

// attemptDiagnosis loads attempt n's diagnosis, if it has one.
func attemptDiagnosis(run *evidence.Run, task string, n int) (diagnose.Diagnosis, bool) {
	dir, err := run.AttemptDir(task, n)
	if err != nil {
		return diagnose.Diagnosis{}, false
	}
	return diagnose.Read(dir)
}

// runAllowedPaths is the run's allowed paths from readiness (item runs); nil lets the attempt's
// meta.json decide.
func runAllowedPaths(run *evidence.Run) []string {
	r, _ := run.State["readiness"].(map[string]any)
	xs, ok := r["allowed_paths"].([]any)
	if !ok {
		return nil
	}
	out := []string{}
	for _, x := range xs {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func runToolchain(run *evidence.Run) string {
	if tc, ok := run.State["toolchain"].(map[string]any); ok {
		if s, ok := tc["name"].(string); ok {
			return s
		}
	}
	return ""
}

// runTestSeconds is one run of the project test command, from the run's baseline grade (033
// research R7); 0 when the grader image doesn't record it.
func runTestSeconds(run *evidence.Run) int {
	return asInt(run.State["baseline_test_seconds"])
}

func runWorkerDeadline(run *evidence.Run) int {
	mode, _ := run.State["mode"].(map[string]any)
	return asInt(mode["worker_deadline_s"])
}

// StoppedForPlanning is the final state of a run the owner ended to go back to planning (033).
const StoppedForPlanning = "stopped(back-to-planning)"

// stopForPlanning ends a run paused for a retry because the owner answered `replan` (033 FR-009):
// it records who and how, removes the grading namespace, writes the summary, closes the tracking
// issue and stops the watcher. Nothing else runs; a new run of the item or slice starts normally.
func (p *P) stopForPlanning(run *evidence.Run, task string, n int) int {
	a := decisions.TakeRecord(run, decisions.Replan, task, n)
	if a == nil {
		return ExitAwaitRetry
	}
	ch, _ := a["channel"].(string)
	if ch == "" {
		ch = "harness"
	}
	run.LastApproval = a
	decisions.Note(run, decisions.Replan, task, n, true, ch, "", "")
	by, _ := a["approver"].(string)
	run.State["stopped"] = map[string]any{"decision": "back-to-planning", "by": by, "channel": ch,
		"at": evidence.Now(), "task": task, "attempt": n - 1}
	delete(run.State, "pending")
	delete(run.State, "decision_note")
	p.dropGrading(run)
	_ = run.SetRunState(StoppedForPlanning, "BACK_TO_PLANNING")
	_, _ = run.WriteSummary()
	p.githubClose(run, "stopped: back to planning")
	watcher.Release(run.Dir, os.Getpid())
	who := ch
	if by != "" {
		who = "@" + by + " on " + ch
	}
	p.say("run %s stopped: back to planning (by %s); the item or slice can be run again after the planning change is merged.", run.ID, who)
	return ExitOK
}

// prevDeadlineNoTrigger reports whether attempt n-1 of task stopped at the deadline and its
// diagnosis pointed to nothing (033 FR-022).
func prevDeadlineNoTrigger(run *evidence.Run, task string, n int) bool {
	tasks, _ := run.State["tasks"].(map[string]any)
	t, _ := tasks[task].(map[string]any)
	for _, x := range anyList(t["attempts"]) {
		a, _ := x.(map[string]any)
		if !numEq(a["n"], n-1) {
			continue
		}
		if a["reason"] != "deadline" {
			return false
		}
		dg, _ := a["diagnosis"].(map[string]any)
		return len(anyList(dg["triggers"])) == 0 && len(strList(dg["triggers"])) == 0
	}
	return false
}

func strList(v any) []string {
	switch x := v.(type) {
	case []string:
		return x
	case []any:
		out := []string{}
		for _, e := range x {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

// recordExtensions stores the target's enabled Spec Kit extensions at the base revision (033
// FR-019), read from the prepared clone; missing or unreadable means none.
func recordExtensions(run *evidence.Run, clone, base string) {
	out, err := exec.Command("git", "-C", clone, "show", base+":.specify/extensions/.registry").Output()
	if err != nil {
		run.State["spec_extensions"] = []string{}
		return
	}
	run.State["spec_extensions"] = diagnose.EnabledExtensions(out)
}
