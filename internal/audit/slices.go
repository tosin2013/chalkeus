package audit

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tosin2013/chalkeus/internal/config"
	"github.com/tosin2013/chalkeus/internal/evidence"
)

// 026 slice runs (specs/026-slice-runs/contracts/audit.md). Whole-item runs never reach this
// file, so committed evidence audits unchanged (FR-022).

func prsOf(d string) []map[string]any {
	var prs []map[string]any
	readJSON(filepath.Join(d, "prs.json"), &prs)
	return prs
}

func statusPRs(d string) int {
	n := 0
	for _, p := range prsOf(d) {
		if p["kind"] == "status" {
			n++
		}
	}
	return n
}

// sliceAU03: exactly 1 draft task PR whose head is the slice's (<item>-us<n>), and status PRs:
// none unless this is the item's last slice, then at most one; all by the publisher identity, on
// the target repository.
func sliceAU03(d string, st map[string]any, item, sliceID string) evidence.CheckRow {
	target, _ := st["target"].(map[string]any)
	repo, _ := target["repo"].(string)
	last, _ := st["slices_last"].(bool)
	want := "/" + strings.ToLower(item+"-"+sliceID)
	nTask, nStatus, ok := 0, 0, repo != ""
	for _, p := range prsOf(d) {
		head, _ := p["head"].(string)
		switch p["kind"] {
		case "task", nil:
			nTask++
			if !strings.HasSuffix(head, want) {
				ok = false
			}
		case "status":
			nStatus++
		default:
			ok = false
		}
		if !prOnHost(p, st, repo) {
			ok = false
		}
	}
	maxStatus := 0
	if last {
		maxStatus = 1
	}
	ok = ok && nTask == 1 && nStatus <= maxStatus
	return row("AU03", ok, "exactly 1 draft task PR for the slice (plus a status PR only after the last slice) on the target, by the publisher identity (026)",
		fmt.Sprintf("repo=%s slice=%s last=%t task_prs=%d status_prs=%d", repo, sliceID, last, nTask, nStatus))
}

// sliceAU13 (026 "slice order"): readiness found every earlier slice merged on the default branch
// (and recorded the revision); a slice before the last proposed no status; the last slice made at
// most one proposal across the local slice runs of the item. Earlier slices built elsewhere are
// reported, never a failure: the default branch is the authority (research R3).
func sliceAU13(d string, st map[string]any, item, sliceID string) evidence.CheckRow {
	last, _ := st["slices_last"].(bool)
	target, _ := st["target"].(map[string]any)
	tname, _ := target["name"].(string)
	ok := true
	why := []string{}

	// 1. The readiness record: allow, the slice rule passed, earlier slices and the revision.
	var rd struct {
		Decision string `json:"decision"`
		Rules    []struct {
			ID, Result string
		} `json:"rules"`
		Slice struct {
			ID              string   `json:"id"`
			Of              int      `json:"of"`
			Earlier         []string `json:"earlier"`
			DefaultRevision string   `json:"default_revision"`
		} `json:"slice"`
	}
	recs, _ := filepath.Glob(filepath.Join(d, "gate", "*-"+item+".readiness.json"))
	sort.Strings(recs)
	if len(recs) == 0 || !readJSON(recs[len(recs)-1], &rd) {
		ok = false
		why = append(why, "no readiness record")
	}
	sliceRule := false
	for _, r := range rd.Rules {
		if r.ID == "slice" && r.Result == "pass" {
			sliceRule = true
		}
	}
	if rd.Decision != "allow" || !sliceRule || rd.Slice.ID != sliceID || rd.Slice.DefaultRevision == "" {
		ok = false
		why = append(why, "readiness did not record the slice as allowed with a default-branch revision")
	}
	rev := rd.Slice.DefaultRevision
	if len(rev) > 7 {
		rev = rev[:7]
	}

	// 2. Earlier slices: local evidence if any, else "evidence elsewhere".
	local := localSliceRuns(filepath.Dir(d), filepath.Base(d), tname, item)
	var earlier []string
	for _, e := range rd.Slice.Earlier {
		if id, found := local[e]; found {
			earlier = append(earlier, fmt.Sprintf("%s merged@%s (run %s)", e, rev, id))
		} else {
			earlier = append(earlier, fmt.Sprintf("%s merged@%s, evidence elsewhere", e, rev))
		}
	}

	// 3. Status proposals.
	mine := statusPRs(d)
	others := 0
	for _, dir := range localSliceDirs(filepath.Dir(d), filepath.Base(d), tname, item) {
		others += statusPRs(dir)
	}
	switch {
	case !last && mine > 0:
		ok = false
		why = append(why, "a slice before the last opened a status PR")
	case last && mine+others > 1:
		ok = false
		why = append(why, "more than one status proposal for the item across its slice runs")
	}
	detail := fmt.Sprintf("item=%s slice=%s of %d earlier=%s status_proposals=%d", item, sliceID, rd.Slice.Of, listOrNone(earlier), mine)
	if last {
		detail += fmt.Sprintf(" (other local slice runs: %d)", others)
	}
	if len(why) > 0 {
		detail += " problems=" + strings.Join(why, "; ")
	}
	return row("AU13", ok, "slices built in order; no status change before the item's last slice (026)", detail)
}

func listOrNone(xs []string) string {
	if len(xs) == 0 {
		return "[]"
	}
	return "[" + strings.Join(xs, ", ") + "]"
}

// localSliceDirs lists the other run folders under root that are slice runs of item on target.
func localSliceDirs(root, self, target, item string) []string {
	var out []string
	entries, _ := os.ReadDir(root)
	for _, e := range entries {
		if !e.IsDir() || e.Name() == self {
			continue
		}
		var st map[string]any
		if !readJSON(filepath.Join(root, e.Name(), "run.json"), &st) {
			continue
		}
		t, _ := st["target"].(map[string]any)
		if st["kind"] != "item" || st["item"] != item || t["name"] != target || st["slice"] == nil {
			continue
		}
		out = append(out, filepath.Join(root, e.Name()))
	}
	sort.Strings(out)
	return out
}

// localSliceRuns maps each slice id to its newest local run id that succeeded.
func localSliceRuns(root, self, target, item string) map[string]string {
	out := map[string]string{}
	for _, dir := range localSliceDirs(root, self, target, item) {
		var st map[string]any
		readJSON(filepath.Join(dir, "run.json"), &st)
		id, _ := st["slice"].(string)
		if state, _ := st["state"].(string); state == "succeeded" && filepath.Base(dir) > out[id] {
			out[id] = filepath.Base(dir)
		}
	}
	return out
}

// ItemView is `./chalkeus forge audit --target <t> --item NNN` (026, contracts/audit.md): one line per local
// run of the item (whole-item and slice runs) with its slice, state and audit result, then the
// newest run's rows. It writes nothing. It reports whether the newest run's audit passes.
func ItemView(root, target, item string, cfg *config.Config, out io.Writer) (bool, error) {
	keyFiles, envFiles, cleanup := secretFiles(cfg)
	defer cleanup()
	entries, err := os.ReadDir(root)
	if err != nil {
		return false, err
	}
	var dirs []string
	for _, e := range entries {
		var st map[string]any
		if !e.IsDir() || !readJSON(filepath.Join(root, e.Name(), "run.json"), &st) {
			continue
		}
		t, _ := st["target"].(map[string]any)
		if st["kind"] == "item" && st["item"] == item && t["name"] == target {
			dirs = append(dirs, filepath.Join(root, e.Name()))
		}
	}
	sort.Strings(dirs)
	if len(dirs) == 0 {
		_, _ = fmt.Fprintf(out, "no local runs of item %s on target %s\n", item, target)
		return false, nil
	}
	_, _ = fmt.Fprintf(out, "item %s on target %s: %d local run(s)\n", item, target, len(dirs))
	var newest []evidence.CheckRow
	for _, d := range dirs {
		var st map[string]any
		readJSON(filepath.Join(d, "run.json"), &st)
		sl, _ := st["slice"].(string)
		if sl == "" {
			sl = "whole"
		}
		rows, err := Dir(d, keyFiles, envFiles, false)
		res := "unreadable"
		if err == nil {
			pass := 0
			for _, r := range rows {
				if r.Result == "pass" {
					pass++
				}
			}
			res = fmt.Sprintf("audit %d/%d", pass, len(rows))
			newest = rows
		}
		_, _ = fmt.Fprintf(out, "  %s  %-5s  %-28v  %s\n", filepath.Base(d), sl, st["state"], res)
	}
	_, _ = fmt.Fprintf(out, "newest run %s:\n", filepath.Base(dirs[len(dirs)-1]))
	ok := len(newest) > 0
	for _, r := range newest {
		mark := "FAIL"
		if r.Result == "pass" {
			mark = "PASS"
		} else {
			ok = false
		}
		_, _ = fmt.Fprintf(out, "  %s %s %s\n", mark, r.ID, r.Description)
	}
	return ok, nil
}
