package ghloop

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/tosin2013/specfoundry/internal/evidence"
	"github.com/tosin2013/specfoundry/internal/fixture"
	"github.com/tosin2013/specfoundry/internal/paths"
	"github.com/tosin2013/specfoundry/internal/pyrepr"
)

// 004 GitHub loop: mirror the target roadmap as item issues, status labels and task checklists,
// and report drift (reference: ghsync.py; 004 research R1, R2, R8). The roadmap is the source
// of truth (constitution I): issues and labels are written from it and never read back as
// status.

// Statuses is the roadmap status vocabulary.
var Statuses = []string{"undecided", "needs-info", "planned", "specced", "in-progress", "implemented",
	"verified", "deferred", "abandoned"}

// LabelPrefix starts every status label.
const LabelPrefix = "status:"

var labelColors = map[string]string{"verified": "0e8a16", "implemented": "1d76db", "in-progress": "fbca04", "planned": "c5def5"}

var (
	itemMarker = regexp.MustCompile(`<!-- specfoundry:item=([^ ]+) -->`)
	taskLine   = regexp.MustCompile(`^- \[( |x|X)\] (.+)$`)
)

// Loop is the GitHub loop for one fixture repository.
type Loop struct {
	Repo, Branch string
	Approvers    []string
	R            Reader
	// W opens the publisher-credential writer (lazily: App mode mints a token first).
	W func() (Writer, error)
	// Snapshot returns spec status JSON for the default branch and each item's tasks text.
	Snapshot func() (map[string]any, map[string]*string, error)
	Ctx      context.Context
}

// Context is the loop's context (Background when unset).
func (l *Loop) Context() context.Context { return l.ctx() }

func (l *Loop) ctx() context.Context {
	if l.Ctx != nil {
		return l.Ctx
	}
	return context.Background()
}

// pyStr is str() of a JSON value.
func pyStr(v any) string {
	switch x := v.(type) {
	case nil:
		return "None"
	case string:
		return x
	}
	return fmt.Sprint(v)
}

// truthy is Python truthiness for JSON values.
func truthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case string:
		return x != ""
	case bool:
		return x
	case json.Number:
		return x.String() != "0"
	case []any:
		return len(x) > 0
	case map[string]any:
		return len(x) > 0
	}
	return true
}

// Checklist returns the task lines of a tasks.md, normalised to "- [x] …" / "- [ ] …".
func Checklist(tasks *string) []string {
	if tasks == nil || *tasks == "" {
		return nil
	}
	var out []string
	for _, line := range splitLines(*tasks) {
		m := taskLine.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		mark := " "
		if m[1] == "x" || m[1] == "X" {
			mark = "x"
		}
		out = append(out, "- ["+mark+"] "+m[2])
	}
	return out
}

// splitLines is str.splitlines() for \n and \r\n text.
func splitLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func tasksSource(item map[string]any) any {
	t, _ := item["tasks"].(map[string]any)
	return t["source"]
}

// IssueBody is the generated body of an item issue (marker first).
func IssueBody(item map[string]any, tasks *string) string {
	head := fmt.Sprintf("**%s — %s** · status `%s`", pyStr(item["id"]), pyStr(item["title"]), pyStr(item["status"]))
	if spec := item["spec_dir"]; truthy(spec) {
		head += fmt.Sprintf(" · spec `%s`", pyStr(spec))
	}
	desc := ""
	if truthy(item["description"]) {
		desc = pyStr(item["description"])
	}
	outcome := "-"
	if truthy(item["outcome"]) {
		outcome = pyStr(item["outcome"])
	}
	lines := []string{"<!-- specfoundry:item=" + pyStr(item["id"]) + " -->", head, "", desc, "", "**Outcome:** " + outcome}
	if tl := Checklist(tasks); len(tl) > 0 {
		src := "tasks.md"
		if s := tasksSource(item); truthy(s) {
			src = pyStr(s)
		}
		lines = append(append(lines, "", "### Tasks (from "+src+")"), tl...)
	}
	lines = append(lines, "", "_Generated from the roadmap by specfoundry. Edits here are overwritten by `pilot github sync`._")
	return strings.Join(lines, "\n") + "\n"
}

// Desired is what one item's issue should look like.
type Desired struct {
	Item, Title, Body, Label, Status string
}

func isStatus(s string) bool {
	for _, x := range Statuses {
		if x == s {
			return true
		}
	}
	return false
}

// DesiredState lists the desired issue for every active item, in roadmap order (superseded items
// are history).
func DesiredState(doc map[string]any, texts map[string]*string) []Desired {
	var out []Desired
	items, _ := doc["items"].([]any)
	for _, x := range items {
		it, _ := x.(map[string]any)
		if a, ok := it["active"]; ok && !truthy(a) {
			continue
		}
		status, _ := it["status"].(string)
		if !isStatus(status) {
			status = "undecided"
		}
		id := pyStr(it["id"])
		out = append(out, Desired{Item: id, Title: id + " — " + pyStr(it["title"]), Body: IssueBody(it, texts[id]),
			Label: LabelPrefix + status, Status: status})
	}
	return out
}

// Indexed is an issue matched to its item marker.
type Indexed struct {
	Item  string
	Issue Issue
}

// ByItem indexes issues by their item marker, in issue-number order. It returns the first issue
// per item and the later duplicates.
func ByItem(issues []Issue) ([]Indexed, []Issue) {
	sorted := append([]Issue(nil), issues...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Number < sorted[j].Number })
	var idx []Indexed
	var dups []Issue
	seen := map[string]bool{}
	for _, i := range sorted {
		m := itemMarker.FindStringSubmatch(i.Body)
		if m == nil {
			continue
		}
		if seen[m[1]] {
			dups = append(dups, i)
			continue
		}
		seen[m[1]] = true
		idx = append(idx, Indexed{Item: m[1], Issue: i})
	}
	return idx, dups
}

func lookup(idx []Indexed, item string) (Issue, bool) {
	for _, x := range idx {
		if x.Item == item {
			return x.Issue, true
		}
	}
	return Issue{}, false
}

// Op is one planned write.
type Op struct {
	Kind   string // ensure-labels | create-issue | edit-issue
	Item   string
	Number int
	Labels []Label   // ensure-labels
	Title  string    // create-issue
	Body   string    // create-issue
	Set    []string  // create-issue labels
	Edit   IssueEdit // edit-issue
}

// PlanOps lists the writes that bring GitHub in line with desired: missing labels, missing
// issues, and title/body/status-label edits (other labels are kept). It never closes or
// deletes.
func PlanOps(desired []Desired, issues []Issue, labels map[string]bool) []Op {
	var ops []Op
	var missing []Label
	for _, s := range Statuses {
		if !labels[LabelPrefix+s] {
			c := labelColors[s]
			if c == "" {
				c = "ededed"
			}
			missing = append(missing, Label{Name: LabelPrefix + s, Color: c})
		}
	}
	if len(missing) > 0 {
		ops = append(ops, Op{Kind: "ensure-labels", Labels: missing})
	}
	idx, _ := ByItem(issues)
	for _, d := range desired {
		cur, ok := lookup(idx, d.Item)
		if !ok {
			ops = append(ops, Op{Kind: "create-issue", Item: d.Item, Title: d.Title, Body: d.Body, Set: []string{d.Label}})
			continue
		}
		var keep []string
		for _, l := range cur.Labels {
			if !strings.HasPrefix(l, LabelPrefix) {
				keep = append(keep, l)
			}
		}
		want := append(keep, d.Label)
		sort.Strings(want)
		have := append([]string(nil), cur.Labels...)
		sort.Strings(have)
		var e IssueEdit
		changed := false
		if strings.TrimSpace(cur.Body) != strings.TrimSpace(d.Body) {
			b := d.Body
			e.Body, changed = &b, true
		}
		if strings.Join(have, "\x00") != strings.Join(want, "\x00") || len(have) != len(want) {
			e.Labels, changed = &want, true
		}
		if cur.Title != d.Title {
			t := d.Title
			e.Title, changed = &t, true
		}
		if changed {
			ops = append(ops, Op{Kind: "edit-issue", Item: d.Item, Number: cur.Number, Edit: e})
		}
	}
	return ops
}

func num(v any) (int, bool) {
	switch x := v.(type) {
	case json.Number:
		i, err := x.Int64()
		return int(i), err == nil
	case float64:
		return int(x), true
	case int:
		return x, true
	}
	return 0, false
}

// DriftReport lists the differences between the roadmap and GitHub (004 research R8). Empty
// means no drift. prs and numbers come from a run's evidence (may be nil).
func DriftReport(desired []Desired, issues []Issue, prs []map[string]any, numbers map[string]any) []pyrepr.OrderedMap {
	out := []pyrepr.OrderedMap{}
	idx, dups := ByItem(issues)
	want := map[string]bool{}
	for _, d := range desired {
		want[d.Item] = true
		cur, ok := lookup(idx, d.Item)
		if !ok {
			out = append(out, pyrepr.NewOrdered("kind", "missing_issue", "item", d.Item))
			continue
		}
		have := []any{}
		var hs []string
		for _, l := range cur.Labels {
			if strings.HasPrefix(l, LabelPrefix) {
				hs = append(hs, l)
			}
		}
		sort.Strings(hs)
		for _, l := range hs {
			have = append(have, l)
		}
		if len(hs) != 1 || hs[0] != d.Label {
			out = append(out, pyrepr.NewOrdered("kind", "label_mismatch", "item", d.Item, "issue", cur.Number, "want", d.Label, "have", have))
		}
		if d.Status == "verified" && cur.State == "open" {
			out = append(out, pyrepr.NewOrdered("kind", "open_but_verified", "item", d.Item, "issue", cur.Number))
		}
		if d.Status != "verified" && cur.State == "closed" {
			out = append(out, pyrepr.NewOrdered("kind", "closed_but_not_verified", "item", d.Item, "issue", cur.Number))
		}
	}
	for _, x := range idx {
		if !want[x.Item] {
			out = append(out, pyrepr.NewOrdered("kind", "orphan_issue", "item", x.Item, "issue", x.Issue.Number))
		}
	}
	for _, i := range dups {
		out = append(out, pyrepr.NewOrdered("kind", "duplicate_issue", "issue", i.Number))
	}
	for _, p := range prs {
		if k, ok := p["kind"]; ok && k != "task" {
			continue
		}
		task, _ := p["task"].(string)
		w, ok := num(numbers[paths.Tasks[task]])
		if !ok || w == 0 {
			continue
		}
		if c, ok := num(p["closes"]); !ok || c != w {
			out = append(out, pyrepr.NewOrdered("kind", "pr_without_closes", "pr", p["number"], "want", w))
		}
	}
	return out
}

// DefaultSnapshot clones the fixture default branch (depth 1) and runs spec status there.
func DefaultSnapshot(repo, branch string) func() (map[string]any, map[string]*string, error) {
	return func() (map[string]any, map[string]*string, error) {
		clone, _, err := fixture.FreshClone(repo, branch, 1)
		if err != nil {
			return nil, nil, loopErr("cannot clone %s: %v", repo, err)
		}
		defer func() { _ = os.RemoveAll(clone) }()
		bin := os.Getenv("SPEC_BIN")
		if bin == "" {
			bin = "spec"
		}
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		var so, se bytes.Buffer
		cmd := exec.CommandContext(ctx, bin, "status", "--json", "--root", clone)
		cmd.Stdout, cmd.Stderr = &so, &se
		err = cmd.Run()
		code := 0
		if err != nil {
			ee, ok := err.(*exec.ExitError)
			if !ok {
				return nil, nil, loopErr("spec status failed on %s: %v", repo, err)
			}
			code = ee.ExitCode()
		}
		if code != 0 && code != 1 {
			msg := strings.TrimSpace(se.String())
			if len(msg) > 200 {
				msg = msg[len(msg)-200:]
			}
			return nil, nil, loopErr("spec status failed on %s: %s", repo, msg)
		}
		var doc map[string]any
		if err := evidence.Decode(so.Bytes(), &doc); err != nil {
			return nil, nil, loopErr("spec status output unreadable on %s", repo)
		}
		texts := map[string]*string{}
		items, _ := doc["items"].([]any)
		for _, x := range items {
			it, _ := x.(map[string]any)
			id := pyStr(it["id"])
			texts[id] = nil
			if src := tasksSource(it); truthy(src) {
				if b, err := os.ReadFile(filepath.Join(clone, pyStr(src))); err == nil {
					s := string(b)
					texts[id] = &s
				}
			}
		}
		return doc, texts, nil
	}
}

// SyncResult is the outcome of Sync.
type SyncResult struct {
	Created, Changed int
	Issues           pyrepr.OrderedMap // item -> issue number, existing first, then created
}

// Sync creates missing item issues and labels and updates labels, titles and bodies. It is
// idempotent and never deletes, closes or reopens. With run, it records github/issues.json and
// a github_sync event.
func (l *Loop) Sync(run *evidence.Run) (SyncResult, error) {
	doc, texts, err := l.Snapshot()
	if err != nil {
		return SyncResult{}, err
	}
	desired := DesiredState(doc, texts)
	issues, err := l.R.Issues(l.ctx(), l.Repo)
	if err != nil {
		return SyncResult{}, err
	}
	labels, err := l.R.Labels(l.ctx(), l.Repo)
	if err != nil {
		return SyncResult{}, err
	}
	ops := PlanOps(desired, issues, labels)
	res := SyncResult{Issues: pyrepr.OrderedMap{Values: map[string]any{}}}
	idx, _ := ByItem(issues)
	for _, x := range idx {
		res.Issues.Keys = append(res.Issues.Keys, x.Item)
		res.Issues.Values[x.Item] = x.Issue.Number
	}
	var w Writer
	for _, op := range ops {
		if w == nil {
			if w, err = l.W(); err != nil {
				return SyncResult{}, err
			}
		}
		switch op.Kind {
		case "ensure-labels":
			_, err = w.EnsureLabels(l.ctx(), l.Repo, op.Labels)
		case "create-issue":
			var n int
			n, err = w.CreateIssue(l.ctx(), l.Repo, op.Title, op.Body, op.Set)
			if err == nil {
				res.Created++
				if _, ok := res.Issues.Values[op.Item]; !ok {
					res.Issues.Keys = append(res.Issues.Keys, op.Item)
				}
				res.Issues.Values[op.Item] = n
			}
		case "edit-issue":
			err = w.EditIssue(l.ctx(), l.Repo, op.Number, op.Edit)
			if err == nil {
				res.Changed++
			}
		}
		if err != nil {
			return SyncResult{}, err
		}
	}
	if run != nil {
		if err := evidence.WriteJSON(filepath.Join(run.Dir, "github", "issues.json"),
			map[string]any{"repo": l.Repo, "issues": res.Issues.Values, "at": evidence.Now()}); err != nil {
			return res, err
		}
		_ = run.Event("github", nil, nil, "github_sync", fmt.Sprintf("created=%d changed=%d", res.Created, res.Changed))
	}
	return res, nil
}

func readIssues(run *evidence.Run) map[string]any {
	raw, err := os.ReadFile(filepath.Join(run.Dir, "github", "issues.json"))
	if err != nil {
		return nil
	}
	var doc map[string]any
	if evidence.Decode(raw, &doc) != nil {
		return nil
	}
	m, _ := doc["issues"].(map[string]any)
	return m
}

// Drift reports the differences; with run, also against its PRs, written to github/drift.json.
func (l *Loop) Drift(run *evidence.Run) ([]pyrepr.OrderedMap, error) {
	doc, texts, err := l.Snapshot()
	if err != nil {
		return nil, err
	}
	issues, err := l.R.Issues(l.ctx(), l.Repo)
	if err != nil {
		return nil, err
	}
	var prs []map[string]any
	var numbers map[string]any
	if run != nil {
		if raw, err := os.ReadFile(filepath.Join(run.Dir, "prs.json")); err == nil {
			_ = evidence.Decode(raw, &prs)
		}
		numbers = readIssues(run)
	}
	rep := DriftReport(DesiredState(doc, texts), issues, prs, numbers)
	if run != nil {
		if err := evidence.WriteJSON(filepath.Join(run.Dir, "github", "drift.json"), map[string]any{"drift": rep, "at": evidence.Now()}); err != nil {
			return rep, err
		}
	}
	return rep, nil
}

// IssueFor is the item issue number recorded for task in the run, or 0.
func IssueFor(run *evidence.Run, task string) int {
	n, _ := num(readIssues(run)[paths.Tasks[task]])
	return n
}
