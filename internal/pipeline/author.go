package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tosin2013/specfoundry/internal/evidence"
	"github.com/tosin2013/specfoundry/internal/fixture"
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
	if res.Slice != nil { // 026 `doctor --baseline --slice`: the slice baseline (G8 passes, G5 fails)
		run.SetSlice(res.Slice.ID)
		r := run.State["readiness"].(map[string]any)
		r["slice"], r["earlier"] = res.Slice, toAnyList(res.Earlier)
	}
	_ = run.Save()
	in := p.itemDir(run)
	defer func() { _ = os.RemoveAll(filepath.Dir(in)) }() // the overlay holds the draft's tests
	if err := os.MkdirAll(in, 0o700); err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(filepath.Join(in, "overlay.tar"), overlay, 0o600); err != nil {
		return nil, nil, err
	}
	if res.Slice != nil && len(res.Earlier) > 0 {
		if err := p.writeEarlierOverlays(in, res.Earlier); err != nil {
			return nil, nil, err
		}
	}
	// Wait for the grading namespace to be gone before returning, so a check run right after
	// this one doesn't meet a namespace that is still terminating (found by the first test
	// author session).
	defer func() {
		if p.C != nil {
			p.C.DeleteNS(grader.NS(item), true)
		}
	}()
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

// writeEarlierOverlays writes each earlier slice's companion folder as earlier-<n>.tar in dir, for
// the grader's G8 (026). They never reach a worker.
func (p *P) writeEarlierOverlays(dir string, folders []string) error {
	cdir, commit, err := fixture.FreshClone(p.Target.CompanionRepo, "", 1)
	if err != nil {
		return fmt.Errorf("companion repository %s: %v", p.Target.CompanionRepo, err)
	}
	defer func() { _ = os.RemoveAll(cdir) }()
	for i, f := range folders {
		tarb, c, err := grader.Overlay(cdir, p.Target.CompanionRepo, commit, f, itemOfFolder(f))
		if err == nil && c.FileCount == 0 {
			err = fmt.Errorf("%s has no files under %s/ (an earlier slice)", p.Target.CompanionRepo, f)
		}
		if err == nil {
			err = os.WriteFile(filepath.Join(dir, fmt.Sprintf("earlier-%d.tar", i)), tarb, 0o600)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// itemOfFolder is the item id in a companion folder path ([<path>/]items/<item>/<slice>).
func itemOfFolder(f string) string {
	parts := strings.Split(f, "/")
	for i, x := range parts {
		if x == "items" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}
