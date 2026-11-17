package grader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tosin2013/specfoundry/internal/evidence"
	"github.com/tosin2013/specfoundry/internal/frame"
	"github.com/tosin2013/specfoundry/internal/k8s"
	"github.com/tosin2013/specfoundry/internal/paths"
	"github.com/tosin2013/specfoundry/internal/runner"
	"github.com/tosin2013/specfoundry/internal/stage"
)

// Item-run grading (014 R7, R8, contracts/grading.md): one grading namespace per run holds the
// baseline Job and each attempt's grader Job, one at a time. Inputs (the snapshot, the change,
// the hidden-test overlay) arrive through the Job's receiver init container.

// ItemGrading is a run's grading setup.
type ItemGrading struct {
	NS             string // pilot-g-<item>
	SnapshotBundle string
	Overlay        []byte // the companion overlay tar (grader.Overlay)
	DeadlineS      int
	CPU, Memory    string
	InputLimit     string
}

// NS is the run's grading namespace for item (the fixture uses pilot-g-<task>-<n>).
func NS(item string) string { return "pilot-g-" + strings.ToLower(item) }

// EnsureNamespace creates the run's grading namespace (no egress, no secrets).
func EnsureNamespace(l paths.Layout, c *k8s.Client, g ItemGrading, runID string) error {
	y, err := k8s.Render(filepath.Join(l.Manifests, "grading-ns.yaml.tmpl"), map[string]string{
		"NS": g.NS, "RUN_ID": runID, "QUOTA_CPU": g.CPU, "QUOTA_MEMORY": g.Memory})
	if err != nil {
		return err
	}
	return c.Apply(y)
}

// BaselineMeta is the baseline's meta.json: the attempt meta's fields with mode=baseline,
// attempt 0 and no change.
func BaselineMeta(attemptMeta map[string]any) map[string]any {
	m := map[string]any{}
	for k, v := range attemptMeta {
		m[k] = v
	}
	m["mode"], m["attempt"] = "baseline", 0
	return m
}

// GradeItem runs one grading Job of an item run: the baseline (n = 0) or attempt n. Evidence
// goes to outDir (run/baseline, or the attempt dir). Any grader trouble is a fail, never a pass.
func GradeItem(l paths.Layout, c *k8s.Client, rn runner.Runner, run *evidence.Run, imgs map[string]string,
	task string, n int, g ItemGrading, metaPath, outDir string) map[string]any {
	baseline := n == 0
	stageName, job := "grade", fmt.Sprintf("grader-%d", n)
	if baseline {
		stageName, job = "baseline", "baseline"
	}
	if !baseline {
		_ = run.Event("grade", task, n, "grade_started", g.NS)
	}
	var report map[string]any
	fail := func(f string, err error) {
		report = failReport(run.ID, task, n, imgs, f)
		if err != nil {
			_ = run.Event(stageName, task, n, stageName+"_error", fmt.Sprint(err))
		}
	}
	var handle *runner.Handle
	func() {
		t := stage.New()
		if err := t.AddFile("in/base.bundle", g.SnapshotBundle); err != nil {
			fail("grader_error", err)
			return
		}
		if err := t.AddFile("in/meta.json", metaPath); err != nil {
			fail("grader_error", err)
			return
		}
		if cb := filepath.Join(outDir, "change.bundle"); !baseline && fileExists(cb) {
			if err := t.AddFile("in/change.bundle", cb); err != nil {
				fail("grader_error", err)
				return
			}
		}
		if err := t.AddTar("grading/overlay", g.Overlay); err != nil {
			fail("grader_error", err)
			return
		}
		tarb, err := t.Bytes()
		if err != nil {
			fail("grader_error", err)
			return
		}
		y, err := k8s.Render(filepath.Join(l.Manifests, "grader-job-item.yaml.tmpl"), map[string]string{
			"NS": g.NS, "JOB": job, "DEADLINE": fmt.Sprint(g.DeadlineS), "TOOLS_IMAGE": imgs["tools"],
			"GRADER_CPU": g.CPU, "GRADER_MEMORY": g.Memory, "INPUT_LIMIT": g.InputLimit})
		if err != nil {
			fail("grader_error", err)
			return
		}
		_ = c.DeleteJob(g.NS, job, true)
		h, err := rn.Submit("grader", g.NS, job, y, false)
		if err != nil {
			fail("grader_error", err)
			return
		}
		handle = &h
		if err := c.StageInputs(g.NS, job, "receiver", "/stage", tarb, 5*time.Minute); err != nil {
			fail("grader_error", err)
			return
		}
		outcome := rn.Wait(h, time.Duration(g.DeadlineS+60)*time.Second)
		if outcome == "DeadlineExceeded" || outcome == "Timeout" {
			fail("grader_timeout", nil)
			return
		}
		files, ferr := frame.Decode(rn.Collect(h), "grade.json")
		if ferr != "" {
			f := "bad_frame"
			if ferr == frame.ErrNoResult {
				f = "no_result"
			}
			fail(f, nil)
			return
		}
		if err := evidence.Decode(files["grade.json"], &report); err != nil {
			fail("bad_frame", nil)
			return
		}
		att, ok := num(report["attempt"])
		if r := report["result"]; (r != "pass" && r != "fail") || !ok || att != n || report["task"] != task ||
			(baseline && report["mode"] != "baseline") {
			fail("bad_frame", nil)
		}
	}()
	if handle != nil {
		rn.Delete(*handle)
	}
	_ = os.MkdirAll(outDir, 0o755)
	_ = evidence.WriteJSON(filepath.Join(outDir, "grade.json"), report)
	var failed []string
	if checks, ok := report["checks"].([]any); ok {
		for _, x := range checks {
			if ch, _ := x.(map[string]any); ch["result"] == "fail" {
				failed = append(failed, fmt.Sprint(ch["id"]))
			}
		}
	}
	detail := strings.Join(failed, ",")
	if f, ok := report["failure"].(string); ok && f != "" {
		detail = f
	}
	_ = run.Event(stageName, task, n, stageName+"_"+fmt.Sprint(report["result"]), detail)
	return report
}

// BaselineReason maps a failed baseline report to the run's failure reason (contracts/readiness.md).
func BaselineReason(report map[string]any) string {
	checks, _ := report["checks"].([]any)
	by := map[string]string{}
	for _, x := range checks {
		ch, _ := x.(map[string]any)
		by[fmt.Sprint(ch["id"])] = fmt.Sprint(ch["result"])
	}
	switch {
	case by["G4_visible_tests"] == "fail":
		return "BASELINE_SUITE_FAILS"
	case by["G4_visible_tests"] == "pass" && by["G5_hidden_tests"] == "fail":
		return "BASELINE_HIDDEN_PASSES"
	}
	return "BASELINE_GRADER_ERROR"
}
