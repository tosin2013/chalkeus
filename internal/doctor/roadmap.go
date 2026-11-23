package doctor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tosin2013/chalkeus/internal/readiness"
)

// Info is a roadmap row: information, never a failure.
const Info = "INFO"

// startable statuses (readiness and the gate accept these; 014 R12).
var startable = map[string]bool{"planned": true, "specced": true, "in-progress": true}

type ssItem struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Status  string `json:"status"`
	Active  bool   `json:"active"`
	SpecDir string `json:"spec_dir"`
	Tasks   *struct {
		Checked *int `json:"checked"`
		Total   *int `json:"total"`
	} `json:"tasks"`
	Dependencies []struct {
		ID        string `json:"id"`
		Satisfied bool   `json:"satisfied"`
	} `json:"dependencies"`
}

// Roadmap summarises a target's roadmap for the pilot (014 doctor): for each active item, its
// status and tasks and what the pilot needs next (done, ready to run, or what's missing), and
// spec status's own findings (errors and warnings). statusJSON is `spec status --json` of targetDir;
// companionItemDir returns an item's hidden-test folder in the companion checkout ("" if the
// companion couldn't be read). Rows are INFO, PASS (ready) or WARN (a spec status finding); never
// FAIL.
func (r *Report) Roadmap(statusJSON, targetDir string, companionItemDir func(item string) string) {
	var doc struct {
		Items       []ssItem `json:"items"`
		Diagnostics []struct {
			Code     string `json:"code"`
			Severity string `json:"severity"`
			Message  string `json:"message"`
			ItemID   string `json:"item_id"`
		} `json:"diagnostics"`
	}
	if err := json.Unmarshal([]byte(statusJSON), &doc); err != nil || len(doc.Items) == 0 {
		r.Add("roadmap", "spec status", Warn, "the target's roadmap could not be read")
		return
	}
	var specDirs []string
	verified, active := 0, 0
	for _, it := range doc.Items {
		if it.SpecDir != "" {
			specDirs = append(specDirs, strings.TrimSuffix(strings.TrimPrefix(it.SpecDir, "./"), "/"))
		}
		if it.Active {
			active++
			if it.Status == "verified" {
				verified++
			}
		}
	}
	pct := 0
	if active > 0 {
		pct = verified * 100 / active
	}
	r.Add("roadmap", "progress", Info, fmt.Sprintf("%d of %d active items verified (%d%%)", verified, active, pct))
	// spec status's own findings come first: they are its rules, not the pilot's (dependency
	// warnings are already in each item's "needs").
	for _, d := range doc.Diagnostics {
		if (d.Severity == "error" || d.Severity == "warning") && d.Code != "SS_DEPENDENCY_BLOCKED" {
			where := d.ItemID
			if where == "" {
				where = "roadmap"
			}
			r.Add("roadmap", "spec status "+where, Warn, d.Code+": "+clip(d.Message, 160))
		}
	}
	for _, it := range doc.Items {
		if !it.Active {
			continue
		}
		label := fmt.Sprintf("%s %s [%s]%s", it.ID, clip(it.Title, 48), it.Status, tasks(it))
		switch { // status-vs-tasks mismatches are spec status's rule (SS_STATUS_AHEAD_OF_TASKS), shown above
		case it.Status == "verified":
			r.Add("roadmap", label, Info, "done")
		case it.Status == "implemented":
			r.Add("roadmap", label, Info, "implemented; verifying it is the owner's call (not a pilot run)")
		case !startable[it.Status]:
			r.Add("roadmap", label, Info, "not startable while "+it.Status)
		default:
			if missing := r.needs(it, targetDir, specDirs, companionItemDir); len(missing) > 0 {
				r.Add("roadmap", label, Info, "needs: "+strings.Join(missing, "; "))
			} else {
				r.Add("roadmap", label, Pass, "ready to run (check it with --item "+it.ID+")")
			}
		}
	}
}

// needs lists what keeps a startable item from running, in the order a human fixes them.
func (r *Report) needs(it ssItem, targetDir string, specDirs []string, companionItemDir func(string) string) []string {
	var out []string
	var unmet []string
	for _, d := range it.Dependencies {
		if !d.Satisfied {
			unmet = append(unmet, d.ID)
		}
	}
	if len(unmet) > 0 {
		out = append(out, "dependencies verified first ("+strings.Join(unmet, ", ")+")")
	}
	dir := strings.TrimSuffix(strings.TrimPrefix(it.SpecDir, "./"), "/")
	if dir == "" {
		return append(out, "a spec (Spec dir on the roadmap), tasks, allowed paths and hidden tests")
	}
	spec, err := os.ReadFile(filepath.Join(targetDir, dir, "spec.md"))
	if err != nil {
		out = append(out, "spec.md")
	}
	if _, err := os.Stat(filepath.Join(targetDir, dir, "tasks.md")); err != nil {
		out = append(out, "tasks.md")
	}
	if err == nil {
		if strings.Contains(string(spec), "[NEEDS CLARIFICATION") {
			out = append(out, "clarifications resolved")
		}
		var others []string
		for _, d := range specDirs {
			if d != dir {
				others = append(others, d)
			}
		}
		if _, perr := readiness.AllowedPaths(string(spec), dir, others); perr != nil {
			out = append(out, "a valid ## Allowed paths section")
		}
	}
	switch hd := companionItemDir(it.ID); {
	case hd == "":
		out = append(out, "a readable companion repository")
	case countTests(hd) == 0:
		out = append(out, "hidden tests (write them, or use the test author)")
	}
	return out
}

func countTests(dir string) int {
	n := 0
	_ = filepath.WalkDir(dir, func(_ string, e os.DirEntry, err error) error {
		if err == nil && e.Type().IsRegular() && e.Name() != readiness.MapFile {
			n++
		}
		return nil
	})
	return n
}

func counts(it ssItem) (int, int) {
	if it.Tasks == nil || it.Tasks.Checked == nil || it.Tasks.Total == nil {
		return 0, 0
	}
	return *it.Tasks.Checked, *it.Tasks.Total
}

func tasks(it ssItem) string {
	c, t := counts(it)
	if t == 0 {
		return ""
	}
	return fmt.Sprintf(" tasks %d/%d", c, t)
}

func clip(s string, n int) string {
	if len(s) > n {
		return s[:n-1] + "…"
	}
	return s
}
