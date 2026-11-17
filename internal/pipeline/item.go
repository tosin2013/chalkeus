package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tosin2013/specfoundry/internal/cluster"
	"github.com/tosin2013/specfoundry/internal/evidence"
	"github.com/tosin2013/specfoundry/internal/fixture"
	"github.com/tosin2013/specfoundry/internal/ghloop"
	"github.com/tosin2013/specfoundry/internal/grader"
	"github.com/tosin2013/specfoundry/internal/images"
	"github.com/tosin2013/specfoundry/internal/paths"
	"github.com/tosin2013/specfoundry/internal/readiness"
	"github.com/tosin2013/specfoundry/internal/toolchain"
	"github.com/tosin2013/specfoundry/internal/verbose"
	"github.com/tosin2013/specfoundry/internal/worker"
)

// An item run (014, specs/014-any-project): one prepared item of a target repository, under
// the fixture's guarantees. readiness → mode approval → prepare (snapshot, hidden-test overlay,
// grading namespace) → baseline → attempts → replay + draft PR → human merge → status change
// `verified` → audit. Fixture-only steps (gating B, the A/1 induction) never apply.

// itemDir holds an item run's host-side inputs: the target clone, the snapshot bundle and the
// overlay tar. They are never evidence; the overlay holds the hidden tests.
func (p *P) itemDir(run *evidence.Run) string {
	return filepath.Join(p.L.Build, "inputs", run.ID, "item")
}

// RunItem starts an item run of p.Target.
func (p *P) RunItem(item string) int {
	imgs, err := images.Load(p.L)
	if err != nil {
		p.say("pilot: %v", err)
		return ExitError
	}
	p.Imgs = imgs
	run, err := evidence.NewRun(p.L.EvidenceRoot())
	if err != nil {
		p.say("pilot: %v", err)
		return ExitError
	}
	p.say("== run %s (target %s, item %s)", run.ID, p.Target.Name, item)
	run.SetItemRun(map[string]any{"name": p.Target.Name, "repo": p.Target.Repo, "branch": p.Target.Branch,
		"toolchain": p.Target.Toolchain}, item)
	_ = run.Save()
	if hosts := p.Cfg.ModelHosts(); len(hosts) > 0 { // for the P06 probe; never in evidence
		_ = os.WriteFile(p.L.ModelHost, []byte(hosts[0]+"\n"), 0o644)
	}
	_ = evidence.WriteConfigJSON(run, p.Cfg.Public(), p.Cfg.ConfigDir, append(p.Cfg.ModelFiles(), p.Target.PublisherEnv))
	pins, _ := paths.ReadEnv(p.L.Pins)
	_ = evidence.WriteVersions(run, pins, p.L.Build, paths.Context, paths.Cluster, imgs)
	_ = run.Event("run", nil, nil, "run_started", p.Target.Repo)

	// Readiness: the gate plus the item rules (contracts/readiness.md); any doubt refuses.
	seq, _ := run.NextGateSeq()
	res, err := readiness.Check(p.readinessDeps(), p.Target, item, seq, filepath.Join(run.Dir, "gate"))
	if err != nil {
		return p.fail(run, "READINESS_ERROR")
	}
	t := run.Task(item)
	t["gate_seqs"] = append(anyList(t["gate_seqs"]), seq)
	_ = run.Event("gate", item, nil, "gate_"+res.Decision, fmt.Sprintf("seq=%d %s", seq, strings.Join(res.Reasons, ",")))
	p.say("  readiness %s: %s %v", item, res.Decision, res.Reasons)
	if res.Decision != "allow" {
		_ = run.SetRunState("failed", "NOT_READY:"+res.Reasons[0])
		_, _ = run.WriteSummary()
		p.say("run %s refused: item %s is not prepared (%s). See docs/targets.md.", run.ID, item, strings.Join(res.Reasons, ", "))
		return ExitRefuse
	}
	t["base_revision"] = res.Commit
	run.State["readiness"] = map[string]any{"spec_dir": res.SpecDir, "allowed_paths": toAnyList(res.AllowedPaths),
		"title": res.Title, "change_type": p.Target.ChangeType, "tests_may_change": toAnyList(res.TestsMayChange),
		"companion_commit": res.CompanionCommit, "hidden_files": res.HiddenFiles, "input_bytes": res.InputBytes}
	_ = run.Save()

	if p.issuesOn() { // 004: mirror the target's roadmap only when the profile opted in
		l, err := p.loop()
		if err == nil {
			_, err = l.Sync(run)
		}
		if err != nil {
			p.say("github sync failed: %v", err)
			return p.fail(run, "GITHUB_SYNC")
		}
	}
	p.checks().GateMatrix(run, p.Status) // the gate's own fail-closed matrix (AU05), as for the fixture
	rows, _ := evidence.ReadJSONL(filepath.Join(run.Dir, "checks", "results.jsonl"))
	for _, r := range rows {
		if id, _ := r["id"].(string); strings.HasPrefix(id, "GM") && r["result"] != "pass" {
			return p.fail(run, "GATE_MATRIX")
		}
	}
	ok, ch := p.confirmMode(run)
	if !ok {
		return p.modeNotConfirmed(run, ch)
	}
	_ = run.Save()
	return p.startItem(run)
}

func (p *P) readinessDeps() readiness.Deps {
	return readiness.Deps{Src: p.Status,
		Clone: func(repo, branch string) (string, string, error) { return fixture.FreshClone(repo, branch, 1) },
		OpenPRHeads: func(repo string) ([]string, error) {
			gh, err := ghloop.OperatorClient()
			if err != nil {
				return nil, err
			}
			prs, err := ghloop.OpenPilotPRs(context.Background(), gh, repo)
			heads := make([]string, 0, len(prs))
			for _, pr := range prs {
				heads = append(heads, pr.HeadRef)
			}
			return heads, err
		}}
}

// modeExtra is printed with the mode for an item run: target, item, toolchain and egress. The
// model host is counted, never named (it is kept out of output and evidence).
func (p *P) modeExtra(run *evidence.Run) string {
	if run.Kind() != evidence.KindItem {
		return ""
	}
	regs := strings.Join(p.Target.RegistryHosts, ", ")
	if regs == "" {
		regs = "none"
	}
	tc := p.Target.Toolchain
	if p.Target.ToolchainImage != "" {
		tc += " (operator-supplied " + p.Target.ToolchainImage + ")"
	}
	item, _ := run.State["item"].(string)
	return fmt.Sprintf("  target %s (%s@%s), item %s, toolchain %s\n  egress: model host(s) %d (from pilot.env); registries: %s",
		p.Target.Name, p.Target.Repo, p.Target.Branch, item, tc, len(p.Cfg.ModelHosts()), regs)
}

// modeExtraGitHub is modeExtra's text for the GitHub decision post: target, item, toolchain and
// registries, without the model host count (contracts/cli.md).
func (p *P) modeExtraGitHub(run *evidence.Run) string {
	regs := strings.Join(p.Target.RegistryHosts, ", ")
	if regs == "" {
		regs = "none"
	}
	tc := p.Target.Toolchain
	if p.Target.ToolchainImage != "" {
		tc += " (operator-supplied " + p.Target.ToolchainImage + ")"
	}
	item, _ := run.State["item"].(string)
	return fmt.Sprintf("target %s (%s@%s), item %s, toolchain %s, registries: %s", p.Target.Name, p.Target.Repo, p.Target.Branch, item, tc, regs)
}

// startItem prepares the item's inputs, runs the baseline, then the first attempt.
func (p *P) startItem(run *evidence.Run) int {
	item, _ := run.State["item"].(string)
	prepStart := time.Now()
	err := p.prepareItem(run)
	verbose.V(1, "prepare %s: snapshot, hidden-test overlay and grading namespace in %s", item, time.Since(prepStart).Round(time.Millisecond))
	if err != nil {
		_ = run.Event("prepare", item, nil, "prepare_failed", clip(err.Error(), 300))
		p.say("could not prepare item %s: %v", item, err)
		reason := "PREPARE"
		for _, code := range []string{"TOOLCHAIN_CONTRACT", "TOOLCHAIN_UNAVAILABLE", "PREFETCH_FAILED"} {
			if strings.Contains(err.Error(), code) {
				reason = code
			}
		}
		return p.fail(run, reason)
	}
	mp, err := p.writeBaselineMeta(run)
	if err != nil {
		return p.fail(run, "PREPARE")
	}
	bdir := filepath.Dir(mp)
	start := time.Now()
	g := grader.GradeItem(p.L, p.C, p.Runner, run, p.Imgs, item, 0, p.itemGrading(run), mp, bdir)
	verbose.V(1, "baseline %s: %v after %s", item, g["result"], time.Since(start).Round(time.Second))
	if g["result"] != "pass" {
		p.say("baseline for item %s failed: on the unchanged code the project's tests must pass and the hidden tests must fail", item)
		return p.fail(run, grader.BaselineReason(g))
	}
	p.say("  baseline %s: pass (the project's tests pass and the hidden tests fail on the base)", item)
	return p.runTask(run, item, 1)
}

// prepareItem makes the run's snapshot, overlay and grading namespace (idempotent on resume).
func (p *P) prepareItem(run *evidence.Run) error {
	item, _ := run.State["item"].(string)
	t := run.Task(item)
	base, _ := t["base_revision"].(string)
	d := p.itemDir(run)
	if err := os.MkdirAll(d, 0o700); err != nil {
		return err
	}
	clone := filepath.Join(d, "target")
	bundle := filepath.Join(d, "snapshot.bundle")
	if _, err := os.Stat(bundle); err != nil {
		_ = os.RemoveAll(clone)
		tmp, _, err := fixture.FreshClone(p.repo(), p.branch(), 0)
		if err != nil {
			return err
		}
		if err := os.Rename(tmp, clone); err != nil {
			return err
		}
		snap, err := fixture.Snapshot(clone, base, bundle)
		if err != nil {
			return err
		}
		st, _ := os.Stat(bundle)
		run.State["snapshot"] = map[string]any{"base_revision": base, "snapshot_commit": snap, "bytes": st.Size()}
		_ = run.Save()
	}
	overlay := filepath.Join(d, "overlay.tar")
	if _, err := os.Stat(overlay); err != nil {
		cdir, commit, err := fixture.FreshClone(p.Target.CompanionRepo, "", 1)
		if err != nil {
			return fmt.Errorf("companion repository %s: %v", p.Target.CompanionRepo, err)
		}
		tarb, c, err := grader.Overlay(cdir, p.Target.CompanionRepo, commit, p.Target.CompanionItemDir(item), item)
		_ = os.RemoveAll(cdir)
		if err != nil {
			return err
		}
		if c.FileCount == 0 {
			return fmt.Errorf("%s has no files under %s/", p.Target.CompanionRepo, p.Target.CompanionItemDir(item))
		}
		if err := os.WriteFile(overlay, tarb, 0o600); err != nil {
			return err
		}
		if err := grader.WriteCompanion(run.Dir, c); err != nil {
			return err
		}
		run.State["companion"] = map[string]any{"repo": c.Repo, "commit": c.Commit, "dir": c.Dir, "file_count": c.FileCount, "files_sha256": c.FilesSHA256}
		_ = run.Save()
	}
	if err := grader.EnsureNamespace(p.L, p.C, p.itemGrading(run), run.ID); err != nil {
		return err
	}
	if err := p.probeToolchain(run); err != nil {
		return err
	}
	if len(p.Target.RegistryHosts) > 0 { // the worker's and the prefetch's proxies (014 R6)
		if _, err := cluster.EnsureTargetProxies(p.L, p.C, p.Target.Name, p.Cfg.ModelHosts(), p.Target.RegistryHosts, p.Imgs["egress"]); err != nil {
			return err
		}
	}
	return p.prefetch(run)
}

// toolchainImage is the run's toolchain image: the target's own, or the catalog's ("" when the
// toolchain is built into the images).
func (p *P) toolchainImage() (string, string, error) {
	if p.Target.ToolchainImage != "" {
		return p.Target.ToolchainImage, "operator-supplied", nil
	}
	cat, err := images.LoadCatalog(p.L)
	if err != nil {
		return "", "", err
	}
	lock, _ := images.ReadLock(p.L)
	ref, err := images.ToolchainImage(cat, lock, p.Target.Toolchain)
	return ref, "catalog", err
}

// probeToolchain checks the toolchain image against the contract once per run and records it.
func (p *P) probeToolchain(run *evidence.Run) error {
	if _, done := run.State["toolchain"]; done {
		return nil
	}
	image, source, err := p.toolchainImage()
	if err != nil || image == "" {
		return err
	}
	pr, err := grader.ProbeToolchain(p.L, p.C, p.Runner, grader.NS(itemOf(run)), p.Imgs["tools"], image)
	if err != nil {
		_ = run.Event("prepare", itemOf(run), nil, "toolchain_refused", clip(err.Error(), 300))
		return err
	}
	rec := map[string]any{"source": source, "image": image, "name": pr.Spec.Name, "version": pr.Version, "spec": pr.Raw,
		"attestation": "not checked"}
	run.State["toolchain"] = rec
	_ = run.Save()
	_ = evidence.WriteJSON(filepath.Join(run.Dir, "toolchain.json"), rec)
	_ = run.Event("prepare", itemOf(run), nil, "toolchain_ok", fmt.Sprintf("%s %s (%s)", pr.Spec.Name, pr.Version, source))
	return nil
}

// prefetch fills the run's dependency cache once, from the base snapshot and the hidden tests.
func (p *P) prefetch(run *evidence.Run) error {
	g := p.itemGrading(run)
	if !g.Cache {
		return nil
	}
	if c, _ := run.State["cache"].(map[string]any); c["prefetch"] == "pass" {
		return nil
	}
	mp, err := p.writeBaselineMeta(run)
	if err != nil {
		return err
	}
	proxy := "http://proxy." + cluster.PrefetchProxyNS(p.Target.Name) + ".svc.cluster.local:8888"
	err = grader.Prefetch(p.L, p.C, p.Runner, run, p.Imgs, g, proxy, p.setupScript(run), mp)
	res := "pass"
	if err != nil {
		res = "fail"
	}
	run.State["cache"] = map[string]any{"pvc": "deps", "size": g.CacheSize, "prefetch": res}
	_ = run.Save()
	p.checks().RunGradingChecks(run, g.NS, g.PrefetchProxyNS)
	_ = run.Event("prepare", itemOf(run), nil, "prefetch_"+res, "")
	return err
}

func itemOf(run *evidence.Run) string { item, _ := run.State["item"].(string); return item }

// toolchainSpec is the run's checked toolchain.json (nil when built into the images).
func toolchainSpec(run *evidence.Run) *toolchain.Spec {
	tc, _ := run.State["toolchain"].(map[string]any)
	raw, _ := tc["spec"].(string)
	if raw == "" {
		return nil
	}
	s, err := toolchain.Parse([]byte(raw))
	if err != nil {
		return nil
	}
	return &s
}

// setupScript is what the worker's and the prefetch's setup step run ("" if nothing to do).
func (p *P) setupScript(run *evidence.Run) string {
	spec := toolchainSpec(run)
	if p.Target.InstallCommand == "" && (spec == nil || len(spec.SetupCommand) == 0) {
		return ""
	}
	if spec == nil {
		spec = &toolchain.Spec{}
	}
	return spec.SetupScript(p.Target.InstallCommand)
}

func (p *P) writeBaselineMeta(run *evidence.Run) (string, error) {
	bdir := filepath.Join(run.Dir, "baseline")
	if err := os.MkdirAll(bdir, 0o755); err != nil {
		return "", err
	}
	mp := filepath.Join(bdir, "meta.json")
	return mp, evidence.WriteJSON(mp, p.baselineMeta(run))
}

// replay puts an item's passing change onto the real base before publishing (014 R4).
func (p *P) replay(run *evidence.Run, task string, n int) error {
	adir, _ := run.AttemptDir(task, n)
	out := filepath.Join(adir, "publish.bundle")
	if _, err := os.Stat(out); err == nil {
		return nil
	}
	snap, _ := run.State["snapshot"].(map[string]any)
	base, _ := snap["base_revision"].(string)
	commit, _ := snap["snapshot_commit"].(string)
	clone := filepath.Join(p.itemDir(run), "target")
	if _, err := os.Stat(filepath.Join(clone, ".git")); err != nil {
		return fmt.Errorf("the run's target clone is gone (%s); start a new run", clone)
	}
	head, err := fixture.Replay(clone, base, commit, filepath.Join(adir, "change.bundle"), out)
	if err != nil {
		return err
	}
	_ = run.Event("publish", task, n, "change_replayed", head)
	return nil
}

// dropGrading deletes an item run's grading namespace (no-op for the fixture or without a cluster).
func (p *P) dropGrading(run *evidence.Run) {
	if run.Kind() != evidence.KindItem || p.C == nil {
		return
	}
	item, _ := run.State["item"].(string)
	p.C.DeleteNS(grader.NS(item), false)
}

func (p *P) catalogEntry() images.Toolchain {
	cat, _ := images.LoadCatalog(p.L)
	tc, _ := cat.For(p.Target)
	return tc
}

// workerItem is the worker's view of an item run.
func (p *P) workerItem(run *evidence.Run) *worker.Item {
	r, _ := run.State["readiness"].(map[string]any)
	specDir, _ := r["spec_dir"].(string)
	var allowed []string
	for _, x := range anyList(r["allowed_paths"]) {
		allowed = append(allowed, fmt.Sprint(x))
	}
	snap, _ := run.State["snapshot"].(map[string]any)
	commit, _ := snap["snapshot_commit"].(string)
	proxyNS := "pilot-egress-model"
	if len(p.Target.RegistryHosts) > 0 {
		proxyNS = cluster.WorkerProxyNS(p.Target.Name)
	}
	tc, _ := run.State["toolchain"].(map[string]any)
	image, _ := tc["image"].(string)
	home := "256Mi"
	if w := p.catalogEntry().Worker; w != nil && w.Home != "" {
		home = w.Home
	}
	var mayChange []string
	for _, x := range anyList(r["tests_may_change"]) {
		mayChange = append(mayChange, fmt.Sprint(x))
	}
	return &worker.Item{TestPaths: p.Target.TestPaths, TestsMayChange: mayChange, SnapshotBundle: filepath.Join(p.itemDir(run), "snapshot.bundle"), SnapshotCommit: commit,
		SpecDir: specDir, AllowedPaths: allowed, Toolchain: p.Target.Toolchain, TestCommand: p.Target.TestCommand,
		CommandTimeout: p.commandTimeout(), InputLimit: p.inputLimit(),
		ProxyNS: proxyNS, ProxyURL: "http://proxy." + proxyNS + ".svc.cluster.local:8888",
		ToolchainImage: image, ToolchainSpec: toolchainSpec(run), SetupScript: p.setupScript(run), HomeLimit: home}
}

// itemGrading is the run's grading setup (catalog defaults, raised by the profile).
func (p *P) itemGrading(run *evidence.Run) grader.ItemGrading {
	item, _ := run.State["item"].(string)
	cpu, mem, deadline := "1", "1Gi", 600
	if g := p.catalogEntry().Grader; g != nil {
		cpu, mem, deadline = g.CPU, g.Memory, g.DeadlineS
	}
	if p.Target.GraderMemory != "" {
		mem = p.Target.GraderMemory
	}
	if p.Target.GraderDeadlineS > 0 {
		deadline = p.Target.GraderDeadlineS
	}
	ov, _ := os.ReadFile(filepath.Join(p.itemDir(run), "overlay.tar"))
	tc, _ := run.State["toolchain"].(map[string]any)
	image, _ := tc["image"].(string)
	cache, size := len(p.Target.RegistryHosts) > 0 && p.Target.InstallCommand != "", "2Gi"
	if c := p.catalogEntry().Cache; c != nil && c.Size != "" {
		size = c.Size
	}
	if p.Target.CacheSize != "" {
		size = p.Target.CacheSize
	}
	return grader.ItemGrading{NS: grader.NS(item), SnapshotBundle: filepath.Join(p.itemDir(run), "snapshot.bundle"),
		Overlay: ov, DeadlineS: deadline, CPU: cpu, Memory: mem, InputLimit: p.inputLimit(),
		ToolchainImage: image, ToolchainSpec: toolchainSpec(run), Cache: cache, CacheSize: size,
		PrefetchProxyNS: cluster.PrefetchProxyNS(p.Target.Name)}
}

func (p *P) commandTimeout() int {
	if g := p.catalogEntry().Grader; g != nil && g.CommandTimeoutS > 0 {
		return g.CommandTimeoutS
	}
	return 300
}

// inputLimit sizes the staged-input emptyDir: twice the snapshot limit, for the change too.
func (p *P) inputLimit() string {
	mb := p.Target.MaxInputMB
	if mb <= 0 {
		mb = 50
	}
	return fmt.Sprintf("%dMi", 2*mb+16)
}

// baselineMeta is the baseline grading's meta.json (mode baseline, attempt 0).
func (p *P) baselineMeta(run *evidence.Run) map[string]any {
	it := p.workerItem(run)
	item, _ := run.State["item"].(string)
	return grader.BaselineMeta(map[string]any{"run_id": run.ID, "task": item, "item_id": item, "base_revision": it.SnapshotCommit,
		"allowed_paths": toAnyList(it.AllowedPaths), "induction": nil, "test_command": it.TestCommand, "hidden_mode": "overlay",
		"toolchain": it.Toolchain, "command_timeout_s": it.CommandTimeout})
}

func toAnyList(xs []string) []any {
	out := make([]any, len(xs))
	for i, x := range xs {
		out[i] = x
	}
	return out
}
