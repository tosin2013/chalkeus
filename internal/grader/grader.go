// Package grader grades one attempt in a fresh grader namespace
// (reference: the former Python pilot's grader.py; FR-010–FR-012).
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
	"github.com/tosin2013/specfoundry/internal/worker"
)

// Timeout is the grader's budget in seconds.
const Timeout = 300

var checkIDs = []string{"G1_bundle_applies", "G2_allowed_paths", "G3_tasks_checked", "G4_visible_tests", "G5_hidden_tests"}

func failReport(runID, task string, n int, imgs map[string]string, failure string) map[string]any {
	checks := make([]any, 0, len(checkIDs))
	for _, c := range checkIDs {
		checks = append(checks, map[string]any{"id": c, "result": "skipped", "detail": failure})
	}
	return map[string]any{"version": 1, "run_id": runID, "task": task, "attempt": n, "result": "fail", "failure": failure,
		"grader_image": imgs["tools"], "checks": checks}
}

func num(v any) (int, bool) {
	switch x := v.(type) {
	case float64:
		return int(x), x == float64(int(x))
	case int:
		return x, true
	}
	if s, ok := v.(interface{ Int64() (int64, error) }); ok {
		i, err := s.Int64()
		return int(i), err == nil
	}
	return 0, false
}

// Grade runs the grader Job; any grader trouble is a fail, never a pass.
func Grade(l paths.Layout, c *k8s.Client, rn runner.Runner, run *evidence.Run, imgs map[string]string, task string, n int) map[string]any {
	ns := fmt.Sprintf("pilot-g-%s-%d", strings.ToLower(task), n)
	adir, _ := run.AttemptDir(task, n)
	item := paths.Tasks[task]
	_ = run.Event("grade", task, n, "grade_started", ns)
	c.DeleteNS(ns, true)
	var report map[string]any
	var handle *runner.Handle
	fail := func(f string, err error) {
		report = failReport(run.ID, task, n, imgs, f)
		if err != nil {
			_ = run.Event("grade", task, n, "grade_error", fmt.Sprint(err))
		}
	}
	func() {
		y, err := k8s.Render(filepath.Join(l.Manifests, "grader.yaml.tmpl"), map[string]string{"NS": ns, "RUN_ID": run.ID})
		if err == nil {
			err = c.Apply(y)
		}
		if err != nil {
			fail("grader_error", err)
			return
		}
		sub := map[string]string{"base.bundle": worker.BaseBundle(l, run.ID, task, n), "meta.json": filepath.Join(adir, "meta.json")}
		if cb := filepath.Join(adir, "change.bundle"); fileExists(cb) {
			sub["change.bundle"] = cb
		}
		if err := c.CreateConfigMap(ns, "submission", sub, nil); err != nil {
			fail("grader_error", err)
			return
		}
		hidden := "test_hidden_" + item + ".py"
		if err := c.CreateConfigMap(ns, "grading", map[string]string{hidden: filepath.Join(l.Grading, hidden)}, nil); err != nil {
			fail("grader_error", err)
			return
		}
		job, err := k8s.Render(filepath.Join(l.Manifests, "grader-job.yaml.tmpl"), map[string]string{"NS": ns, "TOOLS_IMAGE": imgs["tools"]})
		if err != nil {
			fail("grader_error", err)
			return
		}
		h, err := rn.Submit("grader", ns, "grader", job, true)
		if err != nil {
			fail("grader_error", err)
			return
		}
		handle = &h
		outcome := rn.Wait(h, (Timeout+60)*time.Second)
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
		if r := report["result"]; (r != "pass" && r != "fail") || !ok || att != n || report["task"] != task {
			fail("bad_frame", nil)
		}
	}()
	if handle != nil {
		rn.Delete(*handle)
	} else {
		c.DeleteNS(ns, true)
	}
	_ = evidence.WriteJSON(filepath.Join(adir, "grade.json"), report)
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
	_ = run.Event("grade", task, n, "grade_"+fmt.Sprint(report["result"]), detail)
	return report
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
