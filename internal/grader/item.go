package grader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tosin2013/chalkeus/internal/evidence"
	"github.com/tosin2013/chalkeus/internal/frame"
	"github.com/tosin2013/chalkeus/internal/k8s"
	"github.com/tosin2013/chalkeus/internal/paths"
	"github.com/tosin2013/chalkeus/internal/runner"
	"github.com/tosin2013/chalkeus/internal/stage"
	"github.com/tosin2013/chalkeus/internal/toolchain"
)

// Item-run grading (014 R7, R8, contracts/grading.md): one grading namespace per run holds the
// baseline Job and each attempt's grader Job, one at a time. Inputs (the snapshot, the change,
// the hidden-test overlay) arrive through the Job's receiver init container.

// ItemGrading is a run's grading setup.
type ItemGrading struct {
	NS             string // pilot-g-<item>
	SnapshotBundle string
	Overlay        []byte // the companion overlay tar (grader.Overlay)
	// Earlier holds each earlier slice's overlay tar, in order (026 slice runs; staged as
	// grading/earlier/<n>, for G8). Empty for a whole-item run.
	Earlier     [][]byte
	DeadlineS   int
	CPU, Memory string
	InputLimit  string
	// Toolchain and dependencies (014 R5, R7): the toolchain image and its checked
	// toolchain.json (none: built into the tools image), and whether the run has a dependency cache
	// (a PVC filled by the prefetch Job through PrefetchProxyNS, mounted read-only into graders).
	ToolchainImage  string
	ToolchainSpec   *toolchain.Spec
	Cache           bool
	CacheSize       string
	PrefetchProxyNS string
	// HiddenOutput, when set (018 `author check` only), receives the hidden tests' output from
	// the grader's hidden_output.txt frame file. It is for the test author's terminal; it is never
	// written to evidence.
	HiddenOutput *[]byte
}

// Deps is where the cache PVC is mounted: read-write in the prefetch Job, read-only in graders,
// at the same path, so a Python venv's absolute paths stay valid.
const Deps = "/deps"

// cacheBlock is the grading namespace's cache: the PVC and the one egress rule, for the prefetch
// pod only, to the target's prefetch proxy. Graders keep the namespace's default deny.
func (g ItemGrading) cacheBlock() string {
	if !g.Cache {
		return ""
	}
	return `---
apiVersion: v1
kind: PersistentVolumeClaim
metadata: {name: deps, namespace: ` + g.NS + `}
spec:
  accessModes: [ReadWriteOnce]
  resources: {requests: {storage: ` + g.CacheSize + `}}
---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata: {name: allow-prefetch-egress, namespace: ` + g.NS + `}
spec:
  podSelector: {matchLabels: {pilot/role: prefetch}}
  policyTypes: [Egress]
  egress:
    - to:
        - namespaceSelector: {matchLabels: {kubernetes.io/metadata.name: kube-system}}
          podSelector: {matchLabels: {k8s-app: kube-dns}}
      ports: [{protocol: UDP, port: 53}, {protocol: TCP, port: 53}]
    - to:
        - namespaceSelector: {matchLabels: {kubernetes.io/metadata.name: ` + g.PrefetchProxyNS + `}}
          podSelector: {matchLabels: {app: proxy}}
      ports: [{protocol: TCP, port: 8888}]`
}

// podVars are a grading pod's toolchain and cache template variables.
func (g ItemGrading) podVars(readOnlyCache bool) map[string]string {
	b := toolchain.PodBlocks(g.ToolchainImage, g.ToolchainSpec, Deps, readOnlyCache) // read-only cache = an offline grader
	mounts, volumes := b.Mounts, b.Volumes
	if g.Cache && readOnlyCache {
		mounts = strings.TrimPrefix(mounts+"\n            - {name: deps, mountPath: "+Deps+", readOnly: true}", "\n")
		volumes = strings.TrimPrefix(volumes+"\n        - {name: deps, persistentVolumeClaim: {claimName: deps, readOnly: true}}", "\n")
	}
	return map[string]string{"TOOLCHAIN_INITS": b.Inits, "TOOLCHAIN_ENV": b.Env, "TOOLCHAIN_MOUNTS": mounts, "TOOLCHAIN_VOLUMES": volumes}
}

// NS is the run's grading namespace for item (the fixture uses pilot-g-<task>-<n>).
func NS(item string) string { return "pilot-g-" + strings.ToLower(item) }

// EnsureNamespace creates the run's grading namespace (no egress, no secrets).
func EnsureNamespace(l paths.Layout, c *k8s.Client, g ItemGrading, runID string) error {
	y, err := k8s.Render(filepath.Join(l.Manifests, "grading-ns.yaml.tmpl"), map[string]string{
		"NS": g.NS, "RUN_ID": runID, "QUOTA_CPU": g.CPU, "QUOTA_MEMORY": g.Memory, "CACHE_BLOCK": g.cacheBlock()})
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
		for i, e := range g.Earlier { // 026: earlier slices' hidden tests, for G8 (grader only, P11)
			if err := t.AddTar(fmt.Sprintf("grading/earlier/%d", i), e); err != nil {
				fail("grader_error", err)
				return
			}
		}
		tarb, err := t.Bytes()
		if err != nil {
			fail("grader_error", err)
			return
		}
		vars := g.podVars(true)
		for k, v := range map[string]string{"NS": g.NS, "JOB": job, "DEADLINE": fmt.Sprint(g.DeadlineS), "TOOLS_IMAGE": imgs["tools"],
			"GRADER_CPU": g.CPU, "GRADER_MEMORY": g.Memory, "INPUT_LIMIT": g.InputLimit} {
			vars[k] = v
		}
		y, err := k8s.Render(filepath.Join(l.Manifests, "grader-job-item.yaml.tmpl"), vars)
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
		if g.HiddenOutput != nil {
			*g.HiddenOutput = files["hidden_output.txt"]
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
	case by["G8_earlier_slices_kept"] == "fail": // 026: a merged slice's tests fail on the default branch
		return "EARLIER_SLICE_BROKEN_ON_BASE"
	case by["G4_visible_tests"] == "pass" && by["G5_hidden_tests"] == "fail":
		return "BASELINE_HIDDEN_PASSES"
	}
	return "BASELINE_GRADER_ERROR"
}

// Prefetch fills the cache once per run (014 R7): the base snapshot plus the hidden tests, never
// agent code; setup.sh is the toolchain's setup and the target's install command. Its outcome
// goes to run/prefetch.json; a failure is PREFETCH_FAILED.
func Prefetch(l paths.Layout, c *k8s.Client, rn runner.Runner, run *evidence.Run, imgs map[string]string,
	g ItemGrading, proxyURL, setupScript, metaPath string) error {
	start := time.Now()
	rec := map[string]any{"result": "fail", "seconds": 0, "at": evidence.Now()}
	defer func() {
		rec["seconds"] = int(time.Since(start).Seconds())
		_ = evidence.WriteJSON(filepath.Join(run.Dir, "prefetch.json"), rec)
	}()
	t := stage.New()
	if err := t.AddFile("in/base.bundle", g.SnapshotBundle); err != nil {
		return err
	}
	if err := t.AddFile("in/meta.json", metaPath); err != nil {
		return err
	}
	if err := t.AddBytes("in/setup.sh", []byte(setupScript)); err != nil {
		return err
	}
	if g.ToolchainSpec != nil && g.ToolchainSpec.Proxy == "maven-settings" { // T046: the registries-only proxy
		if err := t.AddBytes("in/settings.xml", []byte(toolchain.MavenSettings(proxyURL))); err != nil {
			return err
		}
	}
	if err := t.AddTar("grading/overlay", g.Overlay); err != nil {
		return err
	}
	tarb, err := t.Bytes()
	if err != nil {
		return err
	}
	vars := g.podVars(false)
	vars["TOOLCHAIN_ENV"] = toolchain.EnvYAML(nil, "")
	if g.ToolchainSpec != nil {
		vars["TOOLCHAIN_ENV"] = toolchain.EnvYAML(g.ToolchainSpec.RoleEnv(Deps, toolchain.BasePath, false), "            ")
	}
	for k, v := range map[string]string{"NS": g.NS, "DEADLINE": fmt.Sprint(g.DeadlineS), "TOOLS_IMAGE": imgs["tools"],
		"GRADER_CPU": g.CPU, "GRADER_MEMORY": g.Memory, "INPUT_LIMIT": g.InputLimit, "PROXY_URL": proxyURL} {
		vars[k] = v
	}
	y, err := k8s.Render(filepath.Join(l.Manifests, "prefetch-job.yaml.tmpl"), vars)
	if err != nil {
		return err
	}
	_ = c.DeleteJob(g.NS, "prefetch", true)
	h, err := rn.Submit("prefetch", g.NS, "prefetch", y, false)
	if err != nil {
		return err
	}
	defer rn.Delete(h)
	if err := c.StageInputs(g.NS, "prefetch", "receiver", "/stage", tarb, 5*time.Minute); err != nil {
		return err
	}
	outcome := rn.Wait(h, time.Duration(g.DeadlineS+60)*time.Second)
	rec["outcome"] = outcome
	logs := rn.Collect(h)
	if len(logs) > 2000 {
		logs = logs[len(logs)-2000:]
	}
	rec["log_tail"] = logs
	if outcome != "Complete" {
		return fmt.Errorf("PREFETCH_FAILED: the install command did not complete (%s); see prefetch.json", outcome)
	}
	rec["result"] = "pass"
	return nil
}

// ProbeResult is what the toolchain probe found.
type ProbeResult struct {
	Spec    toolchain.Spec
	Raw     string
	Version string
}

// ProbeToolchain runs the toolchain contract probe (contracts/toolchain-image.md) in the run's
// grading namespace: no network, no secrets. Any failed check is TOOLCHAIN_CONTRACT.
func ProbeToolchain(l paths.Layout, c *k8s.Client, rn runner.Runner, ns, toolsImage, image string) (ProbeResult, error) {
	var pr ProbeResult
	blocks := toolchain.PodBlocks(image, &toolchain.Spec{}, Deps, true)
	y, err := k8s.Render(filepath.Join(l.Manifests, "toolchain-probe-job.yaml.tmpl"), map[string]string{
		"NS": ns, "TOOLS_IMAGE": toolsImage, "TOOLCHAIN_INITS": blocks.Inits})
	if err != nil {
		return pr, err
	}
	_ = c.DeleteJob(ns, "toolchain-probe", true)
	h, err := rn.Submit("probe", ns, "toolchain-probe", y, false)
	if err != nil {
		return pr, err
	}
	defer rn.Delete(h)
	outcome := rn.Wait(h, 240*time.Second)
	return parseProbe(rn.Collect(h), outcome)
}

func parseProbe(logs, outcome string) (ProbeResult, error) {
	var pr ProbeResult
	var out struct {
		ToolchainJSON *string  `json:"toolchain_json"`
		Version       string   `json:"version"`
		VersionRC     *int     `json:"version_rc"`
		Setuid        []string `json:"setuid"`
		Error         string   `json:"error"`
	}
	found := false
	for _, line := range strings.Split(logs, "\n") {
		if rest, ok := strings.CutPrefix(line, "PILOT_PROBE "); ok {
			found = evidence.Decode([]byte(rest), &out) == nil
		}
	}
	switch {
	case !found:
		return pr, fmt.Errorf("TOOLCHAIN_CONTRACT: the probe produced no result (%s); the image must copy /opt/toolchain with cp", outcome)
	case out.ToolchainJSON == nil:
		return pr, fmt.Errorf("TOOLCHAIN_CONTRACT: no /opt/toolchain/toolchain.json")
	}
	spec, err := toolchain.Parse([]byte(*out.ToolchainJSON))
	if err != nil {
		return pr, err
	}
	switch {
	case len(out.Setuid) > 0:
		return pr, fmt.Errorf("TOOLCHAIN_CONTRACT: setuid or setgid files: %s", strings.Join(out.Setuid, ", "))
	case out.VersionRC == nil || *out.VersionRC != 0:
		return pr, fmt.Errorf("TOOLCHAIN_CONTRACT: version_command failed: %s %s", out.Version, out.Error)
	}
	return ProbeResult{Spec: spec, Raw: *out.ToolchainJSON, Version: out.Version}, nil
}
