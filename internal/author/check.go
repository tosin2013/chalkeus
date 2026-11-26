package author

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tosin2013/chalkeus/internal/evidence"
	"github.com/tosin2013/chalkeus/internal/grader"
	"github.com/tosin2013/chalkeus/internal/readiness"
)

// Baseline runs the item's baseline with overlay as the hidden tests (pipeline.AuthorBaseline).
type Baseline func(dir, id, item string, res readiness.Result, overlay []byte) (map[string]any, []byte, error)

// Check is `./chalkeus forge author check` (018 T010): validate the draft and its map, then prove on the
// cluster that the project's tests pass and the drafted tests fail on the unchanged base. The
// drafted tests' output is printed for the author and never stored.
func Check(ctx context.Context, e Env, item string, baseline Baseline) (*Draft, error) {
	d, err := LoadDraft(Root(e.EvidenceRoot), "", e.Target.Name, item)
	if err != nil {
		return nil, err
	}
	draftDir := filepath.Join(d.Workspace, "draft")
	files, hash, err := DraftFiles(draftDir)
	if err != nil {
		return d, d.fail("", err)
	}
	spec, err := os.ReadFile(filepath.Join(d.Workspace, "spec.md"))
	if err != nil {
		return d, err
	}
	if clash := PathsInPlanning(d.Workspace, files); len(clash) > 0 {
		return d, d.fail(hash, refuse(CodeDraftPathInSpec, "%s named in the item's spec, plan or tasks; those are the build worker's own files. Rename the hidden test file(s), for example with an AcceptanceTests suffix",
			strings.Join(clash, ", ")))
	}
	mapping, err := CheckMap(draftDir, SliceCriteria(Criteria(string(spec)), d.Slice), d.SpecSHA256, files) // F12: a slice draft maps only its own criteria, as start writes them
	if err != nil {
		return d, d.fail(hash, err)
	}
	tarb, _, err := grader.Overlay(d.Workspace, d.Companion, "draft", "draft", item)
	if err != nil {
		return d, err
	}
	res := readiness.Result{Commit: d.BaseRevision, SpecDir: str(d.Readiness["spec_dir"]), Title: d.Title,
		AllowedPaths: strs(d.Readiness["allowed_paths"]), TestsMayChange: strs(d.Readiness["tests_may_change"])}
	if si := readiness.SliceInfoOf(d.Readiness["slice"]); si != nil { // 026: the slice baseline
		res.Slice, res.Earlier, res.Overlay = si, strs(d.Readiness["earlier"]), strs(d.Readiness["overlay"])
	}
	n := len(d.Checks) + 1
	id := fmt.Sprintf("%s-check%d", d.ID, n)
	fmt.Fprintf(e.Out, "checking draft %s (%d file(s)) against the unchanged base on the cluster; this takes a few minutes...\n", d.ID, len(files))
	report, hidden, err := baseline(filepath.Join(d.Dir(), fmt.Sprintf("check%d", n)), id, item, res, tarb)
	if err != nil {
		_ = d.Event("check_error", clip(err.Error()))
		return d, err
	}
	code := CodeDraftOK
	if report["result"] != "pass" {
		code = map[string]string{"BASELINE_SUITE_FAILS": CodeDraftSuiteFails, "BASELINE_HIDDEN_PASSES": CodeDraftHiddenPasses}[grader.BaselineReason(report)]
		if code == "" {
			code = CodeDraftGraderError
		}
	}
	if len(hidden) > 0 {
		fmt.Fprintf(e.Out, "----- your drafted tests' output on the unchanged base (not stored) -----\n%s\n-----\n", hidden)
	}
	d.lastOutput = string(hidden)
	d.Files, d.Mapping = files, mapping
	d.Checks = append(d.Checks, CheckRecord{At: evidence.Now(), DraftHash: hash, Result: code, GradePath: filepath.Join(fmt.Sprintf("check%d", n), "baseline", "grade.json")})
	d.State = "checked(fail)"
	if code == CodeDraftOK {
		d.State = "checked(pass)"
	}
	_ = d.Save()
	_ = d.Event("checked", fmt.Sprintf("%s files=%d uncovered=%v", code, len(files), mapping["uncovered"]))
	if code != CodeDraftOK {
		return d, &Error{Code: code, Detail: map[string]string{
			CodeDraftSuiteFails:   "with your tests added, the project's own tests fail on the unchanged base; your tests may break the build or need an undeclared dependency",
			CodeDraftHiddenPasses: "your tests already pass on the unchanged base, so they can't tell a correct change from no change; they must fail until the spec is implemented",
			CodeDraftGraderError:  "the grader could not finish; see the check's grade.json and try again",
		}[code]}
	}
	fmt.Fprintf(e.Out, "DRAFT_OK: the project's tests pass and your tests fail on the base. Covered %v of %v criteria; uncovered: %v\nnext: ./chalkeus forge author submit --target %s --item %s\n",
		mapping["covered"], mapping["criteria"], mapping["uncovered"], d.Target, item)
	return d, nil
}

// fail records a refused check and returns err.
func (d *Draft) fail(hash string, err error) error {
	code := "error"
	if e, ok := err.(*Error); ok {
		code = e.Code
	}
	d.Checks = append(d.Checks, CheckRecord{At: evidence.Now(), DraftHash: hash, Result: code})
	d.State = "checked(fail)"
	_ = d.Save()
	_ = d.Event("check_refused", code)
	return err
}

func str(v any) string { s, _ := v.(string); return s }

func strs(v any) []string {
	var out []string
	switch x := v.(type) {
	case []string:
		return x
	case []any:
		for _, e := range x {
			out = append(out, fmt.Sprint(e))
		}
	}
	return out
}

func clip(s string) string {
	if len(s) > 300 {
		return s[:300]
	}
	return s
}

// PathsInPlanning returns the drafted files whose path appears in the workspace's spec.md,
// plan.md or tasks.md (P11's rule, applied before a draft can reach a pull request).
func PathsInPlanning(workspace string, files []File) []string {
	var text strings.Builder
	for _, f := range []string{"spec.md", "plan.md", "tasks.md"} {
		b, _ := os.ReadFile(filepath.Join(workspace, f))
		text.Write(b)
		text.WriteByte('\n')
	}
	var clash []string
	for _, f := range files {
		if f.Path != MapFile && strings.Contains(text.String(), f.Path) {
			clash = append(clash, f.Path)
		}
	}
	return clash
}
