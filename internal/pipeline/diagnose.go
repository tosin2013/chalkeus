package pipeline

import (
	"github.com/tosin2013/chalkeus/internal/diagnose"
	"github.com/tosin2013/chalkeus/internal/evidence"
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
