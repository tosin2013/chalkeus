package author

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/tosin2013/specfoundry/internal/config"
	"github.com/tosin2013/specfoundry/internal/fixture"
	"github.com/tosin2013/specfoundry/internal/readiness"
)

// Env is what the author commands need from the CLI.
type Env struct {
	Target       config.Target
	EvidenceRoot string // draft evidence goes to <EvidenceRoot>/drafts
	BuildDir     string // workspaces go to <BuildDir>/author (git-ignored)
	// Ready is readiness.Check with SkipHidden (the hidden-test rule is what the author fills),
	// for the whole item or (026) one slice of it.
	Ready    func(t config.Target, item, slice string) (readiness.Result, error)
	Identity func(ctx context.Context, t config.Target) (*Identity, error)
	// OpenAuthorPRs lists the companion's open PRs whose head starts with prefix.
	OpenAuthorPRs func(ctx context.Context, id *Identity, repo, prefix string) ([]string, error)
	Out           io.Writer
	// Slice is the slice the draft is for (026; "" = the whole item): its tests go to
	// items/<item>/<slice>/ and its criteria are that story's.
	Slice string
}

func branchPrefix(item string) string { return "author/" + item + "-" }

// companionDir is where the draft's tests go: the item's folder, or (026) the slice's.
func (e Env) companionDir(item string) string {
	if e.Slice != "" {
		return e.Target.CompanionSliceDir(item, e.Slice)
	}
	return e.Target.CompanionItemDir(item)
}

// Start is `pilot author start` (018 T007): checks, then the workspace the skill works in.
func Start(ctx context.Context, e Env, item string, replace bool) (*Draft, error) {
	id, err := e.Identity(ctx, e.Target)
	if err != nil {
		return nil, err
	}
	res, err := e.Ready(e.Target, item, e.Slice)
	if err != nil {
		return nil, err
	}
	if res.Decision != "allow" {
		detail := strings.Join(res.Reasons, ", ")
		for _, r := range res.Rules { // the failing rule's own detail, e.g. the snapshot size
			if r.Result == "fail" && r.Detail != "" {
				detail = r.Detail
				break
			}
		}
		return nil, refuse(res.Reasons[0], "item %s is not ready to draft tests for: %s", item, detail)
	}
	if res.HiddenFiles > 0 && !replace {
		return nil, refuse(CodeCompanionNotEmpty, "%s already has %d file(s) under %s/; pass --replace to propose replacing them (still only a pull request)",
			e.Target.CompanionRepo, res.HiddenFiles, e.companionDir(item))
	}
	if open, err := e.OpenAuthorPRs(ctx, id, e.Target.CompanionRepo, branchPrefix(item)); err != nil {
		return nil, err
	} else if len(open) > 0 {
		return nil, refuse(CodeDraftPROpen, "a draft pull request is already open on %s (%s); review it, or close it first", e.Target.CompanionRepo, strings.Join(open, ", "))
	}
	d, err := NewDraft(Root(e.EvidenceRoot))
	if err != nil {
		return nil, err
	}
	d.Target, d.Repo, d.Item, d.Title = e.Target.Name, e.Target.Repo, item, res.Title
	d.SpecSHA256, d.BaseRevision = res.SpecSHA256, res.Commit
	d.Level, d.Identity, d.CanWriteTarget = id.Level, id.Login, id.CanWriteTarget
	d.Companion, d.CompanionDir = e.Target.CompanionRepo, e.companionDir(item)
	d.Slice = e.Slice
	d.Workspace = filepath.Join(e.BuildDir, "author", d.ID)
	d.Readiness = map[string]any{"spec_dir": res.SpecDir, "allowed_paths": res.AllowedPaths, "tests_may_change": res.TestsMayChange,
		"title": res.Title, "replace": replace, "existing_files": res.HiddenFiles}
	if res.Slice != nil { // 026: author check's baseline is the slice baseline (earlier slices' tests pass)
		d.Readiness["slice"], d.Readiness["earlier"], d.Readiness["overlay"] = res.Slice, res.Earlier, res.Overlay
	}
	if err := makeWorkspace(d, e.Target, res); err != nil {
		return nil, err
	}
	if err := d.Save(); err != nil {
		return nil, err
	}
	_ = d.Event("started", fmt.Sprintf("level=%s identity=%s can_write_target=%t", d.Level, d.Identity, d.CanWriteTarget))
	w := e.Out
	fmt.Fprintf(w, "draft %s for %s item %s (%s)\n", d.ID, d.Target, item, d.Title)
	fmt.Fprintf(w, "  workspace: %s\n", d.Workspace)
	fmt.Fprintf(w, "  level=%s identity=%s\n", d.Level, d.Identity)
	if d.CanWriteTarget {
		fmt.Fprintf(w, "  note: %s can also write the target %s (minimal level). Protect its main branch so changes arrive only by reviewed PR.\n", d.Identity, d.Repo)
	}
	fmt.Fprintf(w, "  write tests under %s/draft/ (laid out like the project) and %s/draft/%s,\n  then run: pilot/pilot author check --target %s --item %s\n",
		d.Workspace, d.Workspace, MapFile, d.Target, item)
	return d, nil
}

// makeWorkspace writes spec.md, plan.md, tasks.md, criteria.json, a read-only base/ checkout
// and an empty draft/.
func makeWorkspace(d *Draft, t config.Target, res readiness.Result) error {
	ws := d.Workspace
	if err := os.MkdirAll(filepath.Join(ws, "draft"), 0o755); err != nil {
		return err
	}
	clone, _, err := fixture.FreshClone(t.Repo, t.Branch, 0)
	if err != nil {
		return err
	}
	if _, err := fixture.Git(clone, "checkout", "--quiet", res.Commit); err != nil {
		_ = os.RemoveAll(clone)
		return err
	}
	_ = os.RemoveAll(filepath.Join(clone, ".git"))
	base := filepath.Join(ws, "base")
	if err := os.Rename(clone, base); err != nil {
		return err
	}
	for _, f := range []string{"spec.md", "plan.md", "tasks.md"} {
		if raw, err := os.ReadFile(filepath.Join(base, res.SpecDir, f)); err == nil {
			if err := os.WriteFile(filepath.Join(ws, f), raw, 0o444); err != nil {
				return err
			}
		}
	}
	spec, _ := os.ReadFile(filepath.Join(base, res.SpecDir, "spec.md"))
	crit, _ := json.MarshalIndent(map[string]any{"item": d.Item, "spec_sha256": d.SpecSHA256, "criteria": SliceCriteria(Criteria(string(spec)), d.Slice)}, "", "  ")
	if err := os.WriteFile(filepath.Join(ws, "criteria.json"), append(crit, '\n'), 0o444); err != nil {
		return err
	}
	return readOnly(base)
}

func readOnly(dir string) error {
	return filepath.WalkDir(dir, func(p string, e fs.DirEntry, err error) error {
		if err != nil || e.IsDir() || e.Type()&fs.ModeSymlink != 0 { // directories stay writable, so the workspace can be removed
			return err
		}
		info, err := e.Info()
		if err != nil {
			return err
		}
		return os.Chmod(p, info.Mode().Perm()&^0o222)
	})
}
