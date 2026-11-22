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
	"github.com/tosin2013/specfoundry/internal/publisher"
	"github.com/tosin2013/specfoundry/internal/readiness"
	"github.com/tosin2013/specfoundry/internal/slice"
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

// RunItem starts an item run of p.Target: the whole item, or one slice of it (026; sliceID "" is
// the whole item, exactly as before).
func (p *P) RunItem(item, sliceID string) int {
	sliceID = slice.Normalize(sliceID)
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
	if sliceID != "" {
		p.say("== run %s (target %s, item %s, slice %s)", run.ID, p.Target.Name, item, sliceID)
	} else {
		p.say("== run %s (target %s, item %s)", run.ID, p.Target.Name, item)
	}
	run.SetItemRun(map[string]any{"name": p.Target.Name, "repo": p.Target.Repo, "branch": p.Target.Branch,
		"toolchain": p.Target.Toolchain}, item)
	run.SetSlice(sliceID)
	_ = run.Save()
	if hosts := p.Cfg.ModelHosts(); len(hosts) > 0 { // for the P06 probe; never in evidence
		_ = os.WriteFile(p.L.ModelHost, []byte(hosts[0]+"\n"), 0o644)
	}
	_ = evidence.WriteConfigJSON(run, p.Cfg.Public(), p.Cfg.ConfigDir, append(p.Cfg.ModelFiles(), p.Target.PublisherEnv))
	pins, _ := paths.ReadEnv(p.L.Pins)
	_ = evidence.WriteVersions(run, pins, p.L.Build, paths.Context, paths.Cluster, imgs)
	_ = run.Event("run", nil, nil, "run_started", p.Target.Repo)
	if by := StartedBy(os.Getenv("PILOT_STARTED_BY")); by != "" { // 021 FR-007: started by step mode
		_ = run.Event("run", nil, nil, "run_started_by", by)
	}

	// Readiness: the gate plus the item rules (contracts/readiness.md); any doubt refuses.
	seq, _ := run.NextGateSeq()
	rd := p.readinessDeps()
	rd.Slice = sliceID
	if sliceID != "" {
		rd.PriorSlices = PriorSlices(p.L.EvidenceRoot(), p.Target.Name, item, run.ID)
	}
	res, err := readiness.Check(rd, p.Target, item, seq, filepath.Join(run.Dir, "gate"))
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
		"companion_commit": res.CompanionCommit, "hidden_files": res.HiddenFiles, "input_bytes": res.InputBytes,
		"spec_sha256": res.SpecSHA256, "mapping": res.Mapping}
	r := run.State["readiness"].(map[string]any)
	if len(res.Overlay) > 0 { // 026: the companion folders to overlay (absent in older runs: the item folder)
		r["overlay"] = toAnyList(res.Overlay)
	}
	if len(res.Acceptance) > 0 {
		r["acceptance"] = toAnyList(res.Acceptance)
	}
	if res.Slice != nil {
		r["slice"] = res.Slice
		r["earlier"] = toAnyList(res.Earlier)
		run.State["slices_last"] = res.SlicesLast
	}
	run.State["security_level"] = p.Target.SecurityLevel // roadmap C-14; reported, never a failure
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
	return fmt.Sprintf("  target %s (%s@%s), item %s, toolchain %s\n  egress: model host(s) %d (from pilot.env); registries: %s\n  security level: %s; hidden tests: %s",
		p.Target.Name, p.Target.Repo, p.Target.Branch, item, tc, len(p.Cfg.ModelHosts()), regs, p.Target.SecurityLevel, coverage(run))
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
	return fmt.Sprintf("target %s (%s@%s), item %s, toolchain %s, registries: %s, security level: %s, hidden tests: %s",
		p.Target.Name, p.Target.Repo, p.Target.Branch, item, tc, regs, p.Target.SecurityLevel, coverage(run))
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
		dirs, earlier := overlayFolders(run, p.Target.CompanionItemDir(item))
		tarb, c, err := grader.OverlayDirs(cdir, p.Target.CompanionRepo, commit, dirs, item)
		if err != nil {
			_ = os.RemoveAll(cdir)
			return err
		}
		if c.FileCount == 0 {
			_ = os.RemoveAll(cdir)
			return fmt.Errorf("%s has no files under %s/", p.Target.CompanionRepo, strings.Join(dirs, ", "))
		}
		// 026: each earlier slice's hidden tests, for the grader's G8 (never a worker input, P11).
		for i, e := range earlier {
			etar, ec, err := grader.Overlay(cdir, p.Target.CompanionRepo, commit, e, item)
			if err == nil && ec.FileCount == 0 {
				err = fmt.Errorf("%s has no files under %s/ (an earlier slice)", p.Target.CompanionRepo, e)
			}
			if err == nil {
				err = os.WriteFile(filepath.Join(d, fmt.Sprintf("earlier-%d.tar", i)), etar, 0o600)
			}
			if err != nil {
				_ = os.RemoveAll(cdir)
				return err
			}
		}
		_ = os.RemoveAll(cdir)
		if err := os.WriteFile(overlay, tarb, 0o600); err != nil {
			return err
		}
		if err := grader.WriteCompanion(run.Dir, c); err != nil {
			return err
		}
		comp := map[string]any{"repo": c.Repo, "commit": c.Commit, "dir": c.Dir, "file_count": c.FileCount, "files_sha256": c.FilesSHA256}
		if len(earlier) > 0 || len(dirs) > 1 {
			comp["folders"] = toAnyList(append(append([]string{}, earlier...), dirs...))
		}
		run.State["companion"] = comp
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
	meta := p.baselineMeta(run)
	if run.State["author_check"] == true { // 018: the author sees its own tests' output
		meta["author_check"] = true
	}
	return mp, evidence.WriteJSON(mp, meta)
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
	// The PR's Conventional Commits title goes on the commit too: GitHub squash-merges a one-commit
	// PR with the commit's message, which becomes the target's changelog entry.
	head, err := fixture.ReplayAs(clone, base, commit, filepath.Join(adir, "change.bundle"), out,
		publisher.ItemTitle(run, task), "Specfoundry-Run: "+run.ID)
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
	sliceID, phases, earlierPhases := sliceOf(run)
	return &worker.Item{Slice: sliceID, SlicePhases: phases, EarlierPhases: earlierPhases, TestPaths: p.Target.TestPaths, TestsMayChange: mayChange, SnapshotBundle: filepath.Join(p.itemDir(run), "snapshot.bundle"), SnapshotCommit: commit,
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
	var earlier [][]byte
	for i := 0; ; i++ { // 026: earlier slices' overlays, written by prepareItem
		e, err := os.ReadFile(filepath.Join(p.itemDir(run), fmt.Sprintf("earlier-%d.tar", i)))
		if err != nil {
			break
		}
		earlier = append(earlier, e)
	}
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
		Overlay: ov, Earlier: earlier, DeadlineS: deadline, CPU: cpu, Memory: mem, InputLimit: p.inputLimit(),
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
	m := map[string]any{"run_id": run.ID, "task": item, "item_id": item, "base_revision": it.SnapshotCommit,
		"allowed_paths": toAnyList(it.AllowedPaths), "induction": nil, "test_command": it.TestCommand, "hidden_mode": "overlay",
		"toolchain": it.Toolchain, "command_timeout_s": it.CommandTimeout}
	if sm := it.SliceMeta(); sm != nil { // 026: the slice baseline (G8 passes, G5 fails)
		m["slice"] = sm
	}
	return grader.BaselineMeta(m)
}

func toAnyList(xs []string) []any {
	out := make([]any, len(xs))
	for i, x := range xs {
		out[i] = x
	}
	return out
}

// coverage describes the hidden tests' coverage map (018) for the mode approval: uncovered
// criteria, a map written for an older spec, or no map (hand-written tests). Never a refusal.
func coverage(run *evidence.Run) string {
	r, _ := run.State["readiness"].(map[string]any)
	n := asInt(r["hidden_files"]) // int when fresh, json.Number after a resume
	m, _ := r["mapping"].(map[string]any)
	switch {
	case m == nil || m["present"] != true:
		return fmt.Sprintf("%d file(s), no coverage map", n)
	case m["valid"] != true:
		return fmt.Sprintf("%d file(s), coverage map unreadable", n)
	}
	unc := anyList(m["uncovered"])
	s := fmt.Sprintf("%d file(s), every criterion covered", n)
	if len(unc) > 0 {
		s = fmt.Sprintf("%d file(s), uncovered criteria: %v", n, unc)
	}
	if m["stale"] == true {
		s += " (map written for an older version of the spec)"
	}
	return s
}

// StartedBy is who started a run on the operator's behalf, from PILOT_STARTED_BY: only "step"
// (021's manager loop) is recorded; anything else is ignored.
func StartedBy(v string) string {
	if v == "step" {
		return v
	}
	return ""
}

// overlayFolders is the run's companion folders: the ones to overlay (readiness's "overlay", or
// the item folder for runs from before 026) and each earlier slice's (slice runs only).
func overlayFolders(run *evidence.Run, itemDir string) (dirs, earlier []string) {
	r, _ := run.State["readiness"].(map[string]any)
	for _, x := range anyList(r["overlay"]) {
		dirs = append(dirs, fmt.Sprint(x))
	}
	if len(dirs) == 0 {
		dirs = []string{itemDir}
	}
	for _, x := range anyList(r["earlier"]) {
		earlier = append(earlier, fmt.Sprint(x))
	}
	return dirs, earlier
}

// sliceOf is a slice run's slice id, its phases and each earlier slice's phases, from the readiness
// record ("" for a whole-item run). It reads both a fresh *readiness.SliceInfo and the map a
// resumed run.json gives.
func sliceOf(run *evidence.Run) (string, []string, [][]string) {
	r, _ := run.State["readiness"].(map[string]any)
	switch s := r["slice"].(type) {
	case *readiness.SliceInfo:
		return s.ID, s.Phases, s.EarlierPhases
	case map[string]any:
		id, _ := s["id"].(string)
		var phases []string
		for _, x := range anyList(s["phases"]) {
			phases = append(phases, fmt.Sprint(x))
		}
		var earlier [][]string
		for _, e := range anyList(s["earlier_phases"]) {
			var ph []string
			for _, x := range anyList(e) {
				ph = append(ph, fmt.Sprint(x))
			}
			earlier = append(earlier, ph)
		}
		return id, phases, earlier
	}
	return "", nil, nil
}

// SlicesLast reports whether a run builds the item's last slice, or is a whole-item run (both end
// with the status proposal, 026 FR-017).
func SlicesLast(run *evidence.Run) bool {
	if run.Slice() == "" {
		return true
	}
	last, _ := run.State["slices_last"].(bool)
	return last
}

// PriorSlices is the slicing local slice runs of item on target recorded, for SLICE_MAP_CHANGED:
// for each slice id, the phases and slice count of its newest succeeded run (skipping exceptRun).
func PriorSlices(root, target, item, exceptRun string) map[string]readiness.PriorSlice {
	out := map[string]readiness.PriorSlice{}
	newest := map[string]string{}
	entries, _ := os.ReadDir(root)
	for _, e := range entries {
		if !e.IsDir() || e.Name() == exceptRun {
			continue
		}
		r, err := evidence.OpenRun(filepath.Join(root, e.Name()))
		if err != nil || r.Kind() != evidence.KindItem || itemOf(r) != item || r.Slice() == "" {
			continue
		}
		if t, _ := r.State["target"].(map[string]any); t["name"] != target {
			continue
		}
		if st, _ := r.State["state"].(string); st != "succeeded" {
			continue
		}
		id, phases, _ := sliceOf(r)
		rd, _ := r.State["readiness"].(map[string]any)
		sm, _ := rd["slice"].(map[string]any)
		if id == "" || e.Name() < newest[id] {
			continue
		}
		newest[id] = e.Name()
		out[id] = readiness.PriorSlice{Phases: phases, Of: asInt(sm["of"])}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
