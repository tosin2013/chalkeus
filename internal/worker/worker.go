// Package worker runs one worker attempt: an isolated namespace, inputs, the Job, and frame
// collection (reference: the former Python pilot's worker.py; FR-005–FR-009c).
package worker

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/tosin2013/specfoundry/internal/config"
	"github.com/tosin2013/specfoundry/internal/evidence"
	"github.com/tosin2013/specfoundry/internal/frame"
	"github.com/tosin2013/specfoundry/internal/k8s"
	"github.com/tosin2013/specfoundry/internal/paths"
	"github.com/tosin2013/specfoundry/internal/prompt"
	"github.com/tosin2013/specfoundry/internal/runner"
	"github.com/tosin2013/specfoundry/internal/stage"
	"github.com/tosin2013/specfoundry/internal/toolchain"
)

// Deadline is the worker Job's active deadline in seconds.
const Deadline = 1200

// Deps are the worker's collaborators.
type Deps struct {
	L      paths.Layout
	C      *k8s.Client
	Runner runner.Runner
	// Checks runs the live P01–P08 checks while the worker runs (nil to skip).
	Checks func(task string, n int, ns string)
	// Item is set for an item run (014): its inputs come from the target, not the fixture.
	Item *Item
}

// Item is what an item run's worker needs (014 R4, R10).
type Item struct {
	SnapshotBundle string   // host path of the snapshot bundle (one commit, the base tree)
	SnapshotCommit string   // the snapshot commit: the base revision the worker and grader see
	SpecDir        string   // the item's spec folder in the target
	AllowedPaths   []string // from the spec's "## Allowed paths" (+ its tasks.md)
	Toolchain      string
	TestCommand    string
	CommandTimeout int    // seconds per test command in the grader
	InputLimit     string // emptyDir size for staged inputs, e.g. "64Mi"
	// Egress and toolchain (014 R5, R6): the worker's one proxy, the toolchain image and its checked
	// toolchain.json (nil: built into the images), and the setup script ("" when nothing to install).
	ProxyNS, ProxyURL string
	ToolchainImage    string
	ToolchainSpec     *toolchain.Spec
	SetupScript       string
	HomeLimit         string
}

// WorkerDeps is where an item run's worker installs dependencies (its home volume).
const WorkerDeps = "/home/worker/deps"

// AllowedPaths are the paths a worker may change for a task.
func AllowedPaths(task string) []string {
	return []string{"src/slugkit/*", "tests/*", paths.SpecDirs[task] + "/tasks.md"}
}

// NS is the worker namespace of an attempt.
func NS(task string, n int) string { return fmt.Sprintf("pilot-w-%s-%d", strings.ToLower(task), n) }

// InputsDir is where an attempt's inputs (base bundle, prompt, meta) are written.
func InputsDir(l paths.Layout, runID, task string, n int) string {
	return filepath.Join(l.Build, "inputs", runID, fmt.Sprintf("%s-%d", task, n))
}

// BaseBundle is the attempt's base bundle.
func BaseBundle(l paths.Layout, runID, task string, n int) string {
	return filepath.Join(InputsDir(l, runID, task, n), "base.bundle")
}

func git(dir string, args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %s", strings.Join(args[:min(2, len(args))], " "), strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// MakeInputs writes the worker inputs (base bundle in pilot/.build; prompt and meta to both
// the inputs dir and the evidence) and returns the inputs dir and the induction record.
func MakeInputs(d Deps, run *evidence.Run, task string, n int, clone, baseRev string) (string, map[string]any, error) {
	adir, err := run.AttemptDir(task, n)
	if err != nil {
		return "", nil, err
	}
	idir := InputsDir(d.L, run.ID, task, n)
	if err := os.MkdirAll(idir, 0o755); err != nil {
		return "", nil, err
	}
	head, _ := git(clone, "rev-parse", "HEAD")
	if head != baseRev {
		return "", nil, fmt.Errorf("clone HEAD %s != base %s", head, baseRev)
	}
	if _, err := git(clone, "bundle", "create", "-q", filepath.Join(idir, "base.bundle"), "HEAD"); err != nil {
		return "", nil, err
	}
	sd := filepath.Join(clone, paths.SpecDirs[task])
	rd := func(n string) (string, error) { b, err := os.ReadFile(filepath.Join(sd, n)); return string(b), err }
	spec, err := rd("spec.md")
	if err != nil {
		return "", nil, err
	}
	tasks, err := rd("tasks.md")
	if err != nil {
		return "", nil, err
	}
	plan, err := rd("plan.md")
	if err != nil {
		return "", nil, err
	}
	induction := ""
	if task == "A" && n == 1 {
		induction = prompt.Induction
	}
	text, rec, err := prompt.Build(spec, tasks, plan, induction, paths.SpecDirs[task]+"/tasks.md")
	if err != nil {
		return "", nil, err
	}
	var indMeta, record map[string]any
	if rec != nil {
		indMeta = map[string]any{"name": rec.Name, "heading": rec.RemovedHeading, "spec_path": paths.SpecDirs[task] + "/spec.md"}
		record = map[string]any{"name": rec.Name, "removed_heading": rec.RemovedHeading, "removed_sha256": rec.RemovedSHA256,
			"removed_bytes": rec.RemovedBytes, "workspace_spec_trimmed": paths.SpecDirs[task] + "/spec.md"}
	}
	allowed := make([]any, 0, 3)
	for _, p := range AllowedPaths(task) {
		allowed = append(allowed, p)
	}
	meta := map[string]any{"run_id": run.ID, "task": task, "attempt": n, "item_id": paths.Tasks[task], "base_revision": baseRev,
		"allowed_paths": allowed, "induction": nilIfEmpty(indMeta)}
	for _, dir := range []string{idir, adir} {
		if err := os.WriteFile(filepath.Join(dir, "prompt.md"), []byte(text), 0o644); err != nil {
			return "", nil, err
		}
		if err := evidence.WriteJSON(filepath.Join(dir, "meta.json"), meta); err != nil {
			return "", nil, err
		}
	}
	if record != nil {
		if err := evidence.WriteJSON(filepath.Join(adir, "induction.json"), record); err != nil {
			return "", nil, err
		}
	}
	return idir, record, nil
}

func nilIfEmpty(m map[string]any) any {
	if m == nil {
		return nil
	}
	return m
}

// ProxyDenials returns the model egress proxy's refusal lines since a time.
func ProxyDenials(c *k8s.Client, since string) string {
	var t *time.Time
	if p, err := time.Parse(evidence.TimeFormat, since); err == nil {
		t = &p
	}
	var keep []string
	for _, l := range strings.Split(c.DeploymentLogs("pilot-egress-model", "proxy", t), "\n") {
		if strings.Contains(strings.ToLower(l), "refused") {
			keep = append(keep, l)
		}
	}
	return strings.Join(keep, "\n") + "\n"
}

// RunAttempt runs one attempt and returns its record (stored in run.json tasks.<T>.attempts).
func RunAttempt(d Deps, run *evidence.Run, cfg *config.Config, imgs map[string]string, task string, n int, clone, baseRev string) map[string]any {
	ns := NS(task, n)
	adir, _ := run.AttemptDir(task, n)
	att := map[string]any{"n": n, "namespace": ns, "induction": nil, "image": imgs["worker"], "base_revision": baseRev,
		"state": "created", "reason": nil}
	update := func(state, reason string) {
		att["state"] = state
		att["reason"] = reason
	}
	var idir string
	var record map[string]any
	var err error
	if d.Item != nil {
		idir, err = MakeItemInputs(d, run, task, n, clone)
		att["base_revision"] = d.Item.SnapshotCommit
	} else {
		idir, record, err = MakeInputs(d, run, task, n, clone, baseRev)
	}
	if err != nil {
		update("failed", "harness_error")
		_ = run.Event("attempt", task, n, "attempt_error", trim(err))
		_ = run.Event("attempt", task, n, "attempt_failed", "harness_error")
		return att
	}
	if record != nil {
		att["induction"] = record["name"]
	}
	_ = run.Event("attempt", task, n, "attempt_created", fmt.Sprintf("ns=%s induction=%s", ns, pyNone(att["induction"])))
	d.C.DeleteNS(ns, true)
	d.C.DeleteNS(ns+"-probe", true)
	var handle *runner.Handle
	func() {
		defer func() {
			if r := recover(); r != nil {
				update("failed", "harness_error")
				_ = run.Event("attempt", task, n, "attempt_error", fmt.Sprintf("panic: %v", r))
			}
		}()
		nsTmpl, nsVars := "worker-ns.yaml.tmpl", map[string]string{"NS": ns, "RUN_ID": run.ID}
		if d.Item != nil {
			nsTmpl, nsVars["PROXY_NS"] = "worker-ns-item.yaml.tmpl", d.Item.ProxyNS
		}
		nsYAML, err := k8s.Render(filepath.Join(d.L.Manifests, nsTmpl), nsVars)
		if err != nil {
			update("failed", "harness_error")
			_ = run.Event("attempt", task, n, "attempt_error", trim(err))
			return
		}
		tmpl, vars := "worker-job.yaml.tmpl", map[string]string{
			"NS": ns, "DEADLINE": fmt.Sprint(Deadline), "WORKER_IMAGE": imgs["worker"], "PILOT_MODEL": cfg.Get("PILOT_MODEL"),
			"SPEND_CAP": cfg.Get("PILOT_SPEND_CAP_USD"), "TOKEN_CAP": cfg.Get("PILOT_TOKEN_CAP")}
		if d.Item != nil {
			it := d.Item
			tmpl, vars["TOOLS_IMAGE"], vars["INPUT_LIMIT"] = "worker-job-item.yaml.tmpl", imgs["tools"], it.InputLimit
			b := toolchain.PodBlocks(it.ToolchainImage, it.ToolchainSpec, WorkerDeps)
			inits := b.Inits
			if it.SetupScript != "" {
				inits = strings.TrimPrefix(inits+"\n"+toolchain.SetupInit(imgs["tools"], it.ProxyURL, it.ToolchainSpec, WorkerDeps), "\n")
			}
			vars["PROXY_URL"], vars["HOME_LIMIT"] = it.ProxyURL, it.HomeLimit
			vars["TOOLCHAIN_INITS"], vars["TOOLCHAIN_ENV"], vars["TOOLCHAIN_MOUNTS"], vars["TOOLCHAIN_VOLUMES"] = inits, b.Env, b.Mounts, b.Volumes
		}
		jobYAML, err := k8s.Render(filepath.Join(d.L.Manifests, tmpl), vars)
		if err != nil {
			update("failed", "harness_error")
			_ = run.Event("attempt", task, n, "attempt_error", trim(err))
			return
		}
		_ = os.WriteFile(filepath.Join(adir, "manifests.yaml"), []byte(nsYAML+"\n---\n"+jobYAML), 0o644) // no Secret data
		started := evidence.Now()
		steps := []func() error{
			func() error { return d.C.Apply(nsYAML) },
			func() error { return d.C.CreateSecretFromFiles(ns, "model-key", cfg.ModelFiles()) },
			func() error {
				if d.Item != nil { // staged after submit instead (014 R4)
					return nil
				}
				return d.C.CreateConfigMap(ns, "inputs", map[string]string{"base.bundle": filepath.Join(idir, "base.bundle"),
					"prompt.md": filepath.Join(idir, "prompt.md"), "meta.json": filepath.Join(idir, "meta.json")}, nil)
			},
		}
		for _, s := range steps {
			if err := s(); err != nil {
				update("failed", "harness_error")
				_ = run.Event("attempt", task, n, "attempt_error", trim(err))
				return
			}
		}
		h, err := d.Runner.Submit("worker", ns, "worker", jobYAML, true)
		if err != nil {
			update("failed", "harness_error")
			_ = run.Event("attempt", task, n, "attempt_error", trim(err))
			return
		}
		handle = &h
		if d.Item != nil {
			if err := stageInputs(d, idir, ns); err != nil {
				update("failed", "harness_error")
				_ = run.Event("attempt", task, n, "attempt_error", trim(err))
				return
			}
		}
		att["state"] = "running"
		_ = run.Event("attempt", task, n, "attempt_running", imgs["worker"])
		if d.Checks != nil {
			d.Checks(task, n, ns)
		}
		outcome := d.Runner.Wait(h, (Deadline+90)*time.Second)
		st := d.Runner.Status(h)
		st["outcome"] = outcome
		_ = evidence.WriteJSON(filepath.Join(adir, "job.json"), st)
		log := d.Runner.Collect(h)
		_ = os.WriteFile(filepath.Join(adir, "proxy-denials.log"), []byte(ProxyDenials(d.C, started)), 0o644)
		files, ferr := frame.Decode(log)
		if files != nil {
			_ = os.WriteFile(filepath.Join(adir, "report.json"), files["report.json"], 0o644)
			if b, ok := files["change.bundle"]; ok {
				_ = os.WriteFile(filepath.Join(adir, "change.bundle"), b, 0o644)
			}
			var report map[string]any
			_ = evidence.Decode(files["report.json"], &report)
			att["spend"] = report["spend"]
			harness, _ := report["harness"].(map[string]any)
			att["harness_exit"] = harness["exit"]
			if cp, ok := report["changed_paths"].([]any); ok && cp != nil {
				att["changed_paths"] = cp
			} else {
				att["changed_paths"] = []any{}
			}
		}
		att["state"] = "collected"
		spend, _ := att["spend"].(map[string]any)
		switch {
		case outcome == "DeadlineExceeded" || outcome == "Timeout":
			update("failed", "deadline")
		case ferr != "":
			update("failed", ferr)
		case spend != nil && spend["stopped_by_cap"] == true:
			update("failed", "spend_cap")
		}
	}()
	if handle != nil {
		d.Runner.Delete(*handle)
	} else {
		d.C.DeleteNS(ns, true)
	}
	d.C.DeleteNS(ns+"-probe", false)
	reason, _ := att["reason"].(string)
	_ = run.Event("attempt", task, n, "attempt_"+fmt.Sprint(att["state"]), reason)
	return att
}

// MakeItemInputs writes an item attempt's prompt and meta (014 R10) to the inputs dir and the
// evidence; the snapshot bundle stays where the run made it. clone is the target at its real base,
// whose tree is the snapshot's.
func MakeItemInputs(d Deps, run *evidence.Run, task string, n int, clone string) (string, error) {
	it := d.Item
	adir, err := run.AttemptDir(task, n)
	if err != nil {
		return "", err
	}
	idir := InputsDir(d.L, run.ID, task, n)
	if err := os.MkdirAll(idir, 0o755); err != nil {
		return "", err
	}
	rd := func(name string) (string, error) {
		b, err := os.ReadFile(filepath.Join(clone, it.SpecDir, name))
		return string(b), err
	}
	spec, err := rd("spec.md")
	if err != nil {
		return "", err
	}
	tasks, err := rd("tasks.md")
	if err != nil {
		return "", err
	}
	plan, _ := rd("plan.md") // optional for an item
	text, err := prompt.BuildItem(prompt.Item{Spec: spec, Plan: plan, Tasks: tasks, TasksPath: it.SpecDir + "/tasks.md",
		AllowedPaths: it.AllowedPaths, Toolchain: it.Toolchain, TestCommand: it.TestCommand})
	if err != nil {
		return "", err
	}
	allowed := make([]any, 0, len(it.AllowedPaths))
	for _, p := range it.AllowedPaths {
		allowed = append(allowed, p)
	}
	meta := map[string]any{"run_id": run.ID, "task": task, "attempt": n, "item_id": task, "base_revision": it.SnapshotCommit,
		"allowed_paths": allowed, "induction": nil, "mode": "attempt", "test_command": it.TestCommand, "hidden_mode": "overlay",
		"toolchain": it.Toolchain, "spec_sha256": prompt.SpecSHA256(spec), "command_timeout_s": it.CommandTimeout}
	for _, dir := range []string{idir, adir} {
		if err := os.WriteFile(filepath.Join(dir, "prompt.md"), []byte(text), 0o644); err != nil {
			return "", err
		}
		if err := evidence.WriteJSON(filepath.Join(dir, "meta.json"), meta); err != nil {
			return "", err
		}
	}
	return idir, nil
}

// stageInputs streams the attempt's inputs to the worker Job's receiver (014 R4).
func stageInputs(d Deps, idir, ns string) error {
	t := stage.New()
	for name, p := range map[string]string{"prompt.md": filepath.Join(idir, "prompt.md"), "meta.json": filepath.Join(idir, "meta.json"),
		"base.bundle": d.Item.SnapshotBundle} {
		if err := t.AddFile(name, p); err != nil {
			return err
		}
	}
	if d.Item.SetupScript != "" {
		if err := t.AddBytes("setup.sh", []byte(d.Item.SetupScript)); err != nil {
			return err
		}
	}
	b, err := t.Bytes()
	if err != nil {
		return err
	}
	return d.C.StageInputs(ns, "worker", "receiver", "/stage", b, 5*time.Minute)
}

func pyNone(v any) string {
	if v == nil {
		return "None"
	}
	return fmt.Sprint(v)
}

func trim(err error) string {
	s := fmt.Sprintf("%T: %v", err, err)
	if i := strings.LastIndex(s, "."); i > 0 && strings.HasPrefix(s, "*") {
		s = s[i+1:]
	}
	if len(s) > 300 {
		s = s[:300]
	}
	return s
}
