package pipeline

import (
	"os"
	"path/filepath"

	"github.com/tosin2013/specfoundry/internal/evidence"
	"github.com/tosin2013/specfoundry/internal/grader"
	"github.com/tosin2013/specfoundry/internal/images"
	"github.com/tosin2013/specfoundry/internal/readiness"
)

// AuthorBaseline is `pilot author check` (018 R2): the same preparation and baseline as an item
// run (snapshot, grading namespace, toolchain probe, target proxies, prefetch, baseline grader),
// with the test author's draft as the hidden tests. dir is the check's own evidence folder and id
// its name; it is not a run and never reaches a worker. It returns the baseline report (hidden
// output withheld) and, separately, the hidden tests' output for the author's terminal only.
func (p *P) AuthorBaseline(dir, id, item string, res readiness.Result, overlay []byte) (map[string]any, []byte, error) {
	return p.ItemBaseline(dir, id, item, res, overlay, true)
}

// ItemBaseline runs an item's baseline outside a run: for the test author (showHidden: its own
// tests' output comes back) or for `pilot doctor --baseline` (never: doctor runs in the session
// that drives runs, which must not see hidden tests).
func (p *P) ItemBaseline(dir, id, item string, res readiness.Result, overlay []byte, showHidden bool) (map[string]any, []byte, error) {
	imgs, err := images.Load(p.L)
	if err != nil {
		return nil, nil, err
	}
	p.Imgs = imgs
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, nil, err
	}
	run := &evidence.Run{Dir: dir, ID: id, State: map[string]any{"run_id": id}}
	run.SetItemRun(map[string]any{"name": p.Target.Name, "repo": p.Target.Repo, "branch": p.Target.Branch,
		"toolchain": p.Target.Toolchain}, item)
	if showHidden {
		run.State["author_check"] = true
	}
	run.Task(item)["base_revision"] = res.Commit
	run.State["readiness"] = map[string]any{"spec_dir": res.SpecDir, "allowed_paths": toAnyList(res.AllowedPaths),
		"title": res.Title, "change_type": p.Target.ChangeType, "tests_may_change": toAnyList(res.TestsMayChange)}
	_ = run.Save()
	in := p.itemDir(run)
	defer func() { _ = os.RemoveAll(filepath.Dir(in)) }() // the overlay holds the draft's tests
	if err := os.MkdirAll(in, 0o700); err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(filepath.Join(in, "overlay.tar"), overlay, 0o600); err != nil {
		return nil, nil, err
	}
	defer p.dropGrading(run)
	if err := p.prepareItem(run); err != nil {
		return nil, nil, err
	}
	mp, err := p.writeBaselineMeta(run)
	if err != nil {
		return nil, nil, err
	}
	var hidden []byte
	g := p.itemGrading(run)
	if showHidden {
		g.HiddenOutput = &hidden
	}
	report := grader.GradeItem(p.L, p.C, p.Runner, run, p.Imgs, item, 0, g, mp, filepath.Dir(mp))
	return report, hidden, nil
}
