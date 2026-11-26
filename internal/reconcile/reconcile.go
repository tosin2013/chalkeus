// Package reconcile runs after a human merge: check reality, propose `planned -> verified`, ask,
// apply (reference: the former Python pilot's reconcile.py). The status edit is pushed with the
// operator's own git credentials (a human action under constitution V), never with a worker,
// grader or publisher credential; or, with GitHub approvals, it becomes a status-change PR whose
// human merge is the approval (004 R6).
package reconcile

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/tosin2013/chalkeus/internal/scm"
	"github.com/tosin2013/chalkeus/internal/scm/hosts"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/tosin2013/chalkeus/internal/evidence"
	"github.com/tosin2013/chalkeus/internal/fixture"
	"github.com/tosin2013/chalkeus/internal/paths"
	"github.com/tosin2013/chalkeus/internal/slice"
)

// changeBundle is the attempt's change on the real base: the replayed bundle of an item run
// (014 R4), else the worker's own change bundle (the fixture's base is real).
func changeBundle(attemptDir string) string {
	if p := filepath.Join(attemptDir, "publish.bundle"); fileExists(p) {
		return p
	}
	return filepath.Join(attemptDir, "change.bundle")
}

func fileExists(p string) bool { _, err := os.Stat(p); return err == nil }

// itemOf is a task's roadmap item: the fixture's A/B, or an item run's label (the item id).
func itemOf(task string) string {
	if item, ok := paths.Tasks[task]; ok {
		return item
	}
	return task
}

// statusMsg is the status commit's message: the fixture's historical wording, or Conventional
// Commits for an item run (its task is the item id, not a fixture task).
func statusMsg(task, runID string) string {
	_, fixtureTask := paths.Tasks[task]
	if fixtureTask {
		return fmt.Sprintf("pilot: %s verified (run %s)", itemOf(task), runID)
	}
	return fmt.Sprintf("docs(roadmap): %s verified", task)
}

// RoadmapRel is the fixture roadmap path.
const RoadmapRel = ".specify/memory/roadmap.md"

// StatusEdit changes exactly one startable `[status: planned|specced|in-progress]` heading of
// item to verified (014 R12; the fixture's items are always planned).
func StatusEdit(roadmap, item string) (string, error) {
	re := statusRE(item)
	hits := re.FindAllStringIndex(roadmap, -1)
	if len(hits) != 1 {
		return "", fmt.Errorf("expected exactly one planned heading for %s, found %d", item, len(hits))
	}
	return re.ReplaceAllString(roadmap, "${1}verified${3}"), nil
}

// OldStatus is the status StatusEdit would replace ("" if there isn't exactly one).
func OldStatus(roadmap, item string) string {
	m := statusRE(item).FindAllStringSubmatch(roadmap, -1)
	if len(m) != 1 {
		return ""
	}
	return m[0][2]
}

func statusRE(item string) *regexp.Regexp {
	return regexp.MustCompile(`(?m)^(### ` + regexp.QuoteMeta(item) + ` — .*?\[status: )(planned|specced|in-progress)(\])[ \t]*$`)
}

// Reconciler holds the reality checks' data sources (injected so tests need no network).
type Reconciler struct {
	PRView       func(n int) (map[string]any, error) // state, mergedAt, mergeCommit
	TasksPercent func(item string) any               // the item's tasks percent from spec status, or nil
	// ItemStatus is the item's object from spec status on the merged default branch (026: the
	// per-phase counts a slice's merge is checked against), or nil.
	ItemStatus   func(item string) map[string]any
	DefaultFile  func(p string) []byte // nil if missing
	GradedFile   func(p string) []byte
	RoadmapText  func() string
	Push         func(text string) (string, error)
	Ask          func(question string) string
	StatusBundle func(text string) (base, bundle string, err error)
	Say          func(string)
	Where        string // "the fixture roadmap" (default) or "<owner/name>'s roadmap" for an item run
}

// NewRecord is the reconcile/A.json skeleton.
func NewRecord(task, item string, pr int) map[string]any {
	return map[string]any{"task": task, "item_id": item, "pr": pr, "pr_merged": false, "merge_commit": nil,
		"tasks_percent": nil, "tree_matches_graded": nil, "mismatched_paths": []any{}, "proposed": nil,
		"approved": false, "applied_commit": nil, "reason": nil, "at": evidence.Now()}
}

// Check runs the reality checks (merged, tasks 100%, merged tree matches the graded attempt).
// It returns (0, rec, newRoadmap) when a proposal can be made, else (code, rec, "").
func (r Reconciler) Check(task, item string, pr int, changed []string) (int, map[string]any, string) {
	rec := NewRecord(task, item, pr)
	view, err := r.PRView(pr)
	if err != nil {
		view = map[string]any{}
	}
	merged := view["state"] == "MERGED" && view["mergedAt"] != nil && view["mergedAt"] != ""
	rec["pr_merged"] = merged
	if mc, ok := view["mergeCommit"].(map[string]any); ok {
		rec["merge_commit"] = mc["oid"]
	}
	if !merged {
		rec["reason"] = "NOT_MERGED"
		return 4, rec, ""
	}
	pct := r.TasksPercent(item)
	rec["tasks_percent"] = pct
	if p, ok := asInt(pct); !ok || p != 100 {
		rec["reason"] = "TASKS_INCOMPLETE"
		return 1, rec, ""
	}
	mism := []any{}
	for _, p := range changed {
		if string(r.DefaultFile(p)) != string(r.GradedFile(p)) || (r.DefaultFile(p) == nil) != (r.GradedFile(p) == nil) {
			mism = append(mism, p)
		}
	}
	rec["tree_matches_graded"] = len(mism) == 0
	rec["mismatched_paths"] = mism
	if len(mism) > 0 {
		rec["reason"] = "TREE_MISMATCH"
		return 1, rec, ""
	}
	old := r.RoadmapText()
	edited, err := StatusEdit(old, item)
	if err != nil {
		rec["reason"] = "STATUS_EDIT"
		return 1, rec, ""
	}
	var diff []string
	ol, nl := strings.Split(old, "\n"), strings.Split(edited, "\n")
	for i := range min(len(ol), len(nl)) {
		if ol[i] != nl[i] {
			diff = append(diff, "- "+ol[i]+"\n+ "+nl[i])
		}
	}
	rec["proposed"] = "planned→verified"
	if from := OldStatus(old, item); from != "planned" {
		rec["proposed"] = from + "→verified" // an item run starting from specced or in-progress (014)
	}
	r.Say(fmt.Sprintf("Reconcile proposal for task %s (item %s), PR #%d merged (%v), tasks 100%%, graded tree matches:\n%s",
		task, item, pr, rec["merge_commit"], strings.Join(diff, "\n")))
	return 0, rec, edited
}

// CheckSlice is the merge check for a slice that isn't the item's last (026 R7): the PR is merged,
// every phase of the slice is fully ticked on the default branch, and the merged tree matches the
// graded attempt. It never proposes a status and never edits the roadmap (FR-017). Codes are
// Check's: 0 ok (record "slice_merged"), 1 a check failed, 4 not merged.
func (r Reconciler) CheckSlice(task, item, sliceID string, phases []string, pr int, changed []string) (int, map[string]any) {
	rec := NewRecord(task, item, pr)
	rec["slice"] = sliceID
	view, err := r.PRView(pr)
	if err != nil {
		view = map[string]any{}
	}
	merged := view["state"] == "MERGED" && view["mergedAt"] != nil && view["mergedAt"] != ""
	rec["pr_merged"] = merged
	if mc, ok := view["mergeCommit"].(map[string]any); ok {
		rec["merge_commit"] = mc["oid"]
	}
	if !merged {
		rec["reason"] = "NOT_MERGED"
		return 4, rec
	}
	var st map[string]any
	if r.ItemStatus != nil {
		st = r.ItemStatus(item)
	}
	got, ok := slice.Parse(st)
	if !ok {
		rec["reason"] = "SLICE_UNEVALUABLE"
		return 1, rec
	}
	counts := map[string][2]int{}
	for _, p := range got {
		counts[p.Heading] = [2]int{p.Checked, p.Total}
	}
	var open []any
	for _, h := range phases {
		c, found := counts[h]
		if !found || c[0] != c[1] {
			open = append(open, h)
		}
	}
	rec["slice_phases_open"] = open
	if len(open) > 0 {
		rec["reason"] = "SLICE_TASKS_INCOMPLETE"
		return 1, rec
	}
	var mism []any
	for _, p := range changed {
		if string(r.DefaultFile(p)) != string(r.GradedFile(p)) || (r.DefaultFile(p) == nil) != (r.GradedFile(p) == nil) {
			mism = append(mism, p)
		}
	}
	rec["tree_matches_graded"] = len(mism) == 0
	rec["mismatched_paths"] = mism
	if len(mism) > 0 {
		rec["reason"] = "TREE_MISMATCH"
		return 1, rec
	}
	rec["proposed"] = "slice_merged"
	if r.Say != nil {
		r.Say(fmt.Sprintf("Slice %s of item %s merged (PR #%d, %v): its tasks are ticked and the graded tree matches. No status change until the item's last slice.",
			sliceID, item, pr, rec["merge_commit"]))
	}
	return 0, rec
}

// Run checks, asks, and pushes the edit on yes. Codes: 0 applied, 1 check failed, 4 not merged,
// 5 declined.
func (r Reconciler) Run(task, item string, pr int, changed []string) (int, map[string]any) {
	code, rec, edited := r.Check(task, item, pr, changed)
	if code != 0 {
		return code, rec
	}
	q := fmt.Sprintf("Apply '%s: planned → verified' to the fixture roadmap? [y/N] ", item) // the guide quotes this
	if r.Where != "" {                                                                      // an item run (014): its own status and repository
		q = fmt.Sprintf("Apply '%s: %s → verified' to %s? [y/N] ", item, OldStatus(r.RoadmapText(), item), r.Where)
	}
	ans := strings.ToLower(strings.TrimSpace(r.Ask(q)))
	if ans != "y" {
		rec["reason"] = "DECLINED"
		return 5, rec
	}
	rec["approved"] = true
	commit, err := r.Push(edited)
	if err != nil {
		rec["reason"] = "PUSH_FAILED"
		return 1, rec
	}
	rec["applied_commit"] = commit
	return 0, rec
}

// CheckStatusPR: is a merged status-change PR exactly the proposed edit, merged by an allowed
// human? (004 R6; shared vectors in specs/004-github-loop/contracts/vectors/status_pr.json)
func CheckStatusPR(state, mergedBy, baseText, headText string, changed []string, item string, approvers []string, publisher string) (bool, string) {
	if state != "MERGED" {
		return false, "NOT_MERGED"
	}
	want, err := StatusEdit(baseText, item)
	if err != nil || len(changed) != 1 || changed[0] != RoadmapRel || headText != want {
		return false, "STATUS_PR_EDITED"
	}
	if publisher != "" && strings.EqualFold(mergedBy, publisher) {
		return false, "STATUS_PR_MERGED_BY_PUBLISHER"
	}
	for _, a := range approvers {
		if strings.EqualFold(a, mergedBy) {
			return true, ""
		}
	}
	return false, "STATUS_PR_NOT_APPROVER"
}

func asInt(v any) (int, bool) {
	switch x := v.(type) {
	case float64:
		return int(x), x == float64(int(x))
	case int:
		return x, true
	case json.Number:
		i, err := x.Int64()
		return int(i), err == nil
	}
	return 0, false
}

func gitIn(dir string, args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %s", strings.Join(args[:min(2, len(args))], " "), strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// ForFixture builds a Reconciler bound to a fresh full clone of the fixture default branch and
// a graded tree (base + the passing attempt's bundle). The cleanup func removes both.
func ForFixture(h scm.Host, repo, branch string, runDir, runID, task, attemptDir, baseRev string, ask func(string) string, say func(string)) (Reconciler, func(), error) {
	clone, _, err := fixture.FreshClone(h, repo, branch, 0)
	if err != nil {
		return Reconciler{}, func() {}, err
	}
	graded := filepath.Join(runDir, ".graded-"+task)
	cleanup := func() { _ = os.RemoveAll(clone); _ = os.RemoveAll(graded) }
	_ = os.RemoveAll(graded)
	steps := [][]string{
		{"clone", "--quiet", clone, graded},
	}
	if out, err := exec.Command("git", steps[0]...).CombinedOutput(); err != nil {
		cleanup()
		return Reconciler{}, func() {}, fmt.Errorf("git clone: %s", out)
	}
	for _, a := range [][]string{{"checkout", "--quiet", baseRev},
		{"fetch", "--quiet", changeBundle(attemptDir), "HEAD:refs/pilot/graded"},
		{"checkout", "--quiet", "refs/pilot/graded"}} {
		if _, err := gitIn(graded, a...); err != nil {
			cleanup()
			return Reconciler{}, func() {}, err
		}
	}
	read := func(root string) func(string) []byte {
		return func(p string) []byte {
			b, err := os.ReadFile(filepath.Join(root, p))
			if err != nil {
				return nil
			}
			return b
		}
	}
	r := Reconciler{
		PRView: func(n int) (map[string]any, error) {
			sc, err := hosts.Operator(h)
			if err != nil {
				return nil, err
			}
			cr, err := sc.ChangeRequest(context.Background(), repo, n)
			if err != nil {
				return nil, err
			}
			return PRViewOf(cr), nil
		},
		ItemStatus: func(item string) map[string]any {
			out, _ := exec.Command("spec", "status", "--json", "--root", clone).Output()
			var doc map[string]any
			if evidence.Decode(out, &doc) != nil {
				return nil
			}
			items, _ := doc["items"].([]any)
			for _, x := range items {
				if it, _ := x.(map[string]any); it["id"] == item {
					return it
				}
			}
			return nil
		},
		TasksPercent: func(item string) any {
			out, _ := exec.Command("spec", "status", "--json", "--root", clone).Output()
			var doc map[string]any
			if evidence.Decode(out, &doc) != nil {
				return nil
			}
			items, _ := doc["items"].([]any)
			for _, x := range items {
				it, _ := x.(map[string]any)
				if it["id"] == item {
					tasks, _ := it["tasks"].(map[string]any)
					return tasks["percent"]
				}
			}
			return nil
		},
		DefaultFile: read(clone), GradedFile: read(graded),
		RoadmapText: func() string { b, _ := os.ReadFile(filepath.Join(clone, RoadmapRel)); return string(b) },
		Push: func(text string) (string, error) {
			if err := os.WriteFile(filepath.Join(clone, RoadmapRel), []byte(text), 0o644); err != nil {
				return "", err
			}
			if _, err := fixture.Git(h, clone, "add", RoadmapRel); err != nil {
				return "", err
			}
			if _, err := fixture.Git(h, clone, "commit", "--quiet", "-m", statusMsg(task, runID)); err != nil {
				return "", err
			}
			if _, err := fixture.Git(h, clone, "push", "--quiet", fixture.URL(h, repo), "HEAD:refs/heads/"+branch); err != nil {
				return "", err
			}
			return gitIn(clone, "rev-parse", "HEAD")
		},
		StatusBundle: func(text string) (string, string, error) {
			base, err := gitIn(clone, "rev-parse", "HEAD")
			if err != nil {
				return "", "", err
			}
			if err := os.WriteFile(filepath.Join(clone, RoadmapRel), []byte(text), 0o644); err != nil {
				return "", "", err
			}
			if _, err := fixture.Git(h, clone, "add", RoadmapRel); err != nil {
				return "", "", err
			}
			if _, err := fixture.Git(h, clone, "commit", "--quiet", "-m", statusMsg(task, runID)); err != nil {
				return "", "", err
			}
			out := filepath.Join(runDir, ".status-"+task+".bundle")
			if _, err := fixture.Git(h, clone, "bundle", "create", "--quiet", out, base+"..HEAD"); err != nil {
				return "", "", err
			}
			return base, out, nil
		},
		Ask: ask, Say: say,
	}
	return r, cleanup, nil
}

// PRViewOf is the view Check reads, from a change request on any host (037): the landed commit
// is the merge commit, the squash commit or the fast-forwarded head (research R5).
func PRViewOf(cr scm.ChangeRequest) map[string]any {
	view := map[string]any{"state": strings.ToUpper(cr.State), "mergedAt": nil, "mergeCommit": nil}
	if cr.Merged() {
		view["state"] = "MERGED"
		view["mergedAt"] = "merged" // Check needs a non-empty value; the host's merge time isn't recorded
		view["mergeCommit"] = map[string]any{"oid": cr.LandedCommit}
	}
	return view
}
