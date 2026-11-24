package debt

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// Classes (research R4).
const (
	ClassSuperseded  = "superseded"
	ClassSafe        = "safe"
	ClassNeedsCode   = "needs code changes"
	ClassPending     = "pending"
	ClassNotTriaged  = "not triaged"
	groupNoneFailing = "no failing checks"
)

// Report is `forge debt --json` (data-model.md TriageReport).
type Report struct {
	Version              int               `json:"version"`
	Target               string            `json:"target"`
	Repo                 string            `json:"repo"`
	DefaultBranch        string            `json:"default_branch"`
	DefaultCommit        string            `json:"default_commit"`
	GeneratedAt          string            `json:"generated_at"`
	Credential           string            `json:"credential"`
	Permissions          map[string]string `json:"permissions,omitempty"`
	DefaultChecksFailing []string          `json:"default_checks_failing"`
	SharedChecks         map[string]string `json:"shared_checks"`
	Groups               []Group           `json:"groups"`
	NotTriaged           []Entry           `json:"not_triaged"`
	BaselineFlags        []Flag            `json:"baseline_flags"`
	Unreadable           []string          `json:"unreadable"`
	Counts               map[string]int    `json:"counts"`
}

// Group is the PRs that fail the same set of checks.
type Group struct {
	FailingChecks []string `json:"failing_checks"`
	Shared        []string `json:"shared"`
	PRs           []Entry  `json:"prs"`
}

// Entry is one bot PR.
type Entry struct {
	Number         int       `json:"number"`
	URL            string    `json:"url"`
	Author         string    `json:"author"`
	Ecosystem      string    `json:"ecosystem"`
	Directory      string    `json:"directory"`
	Packages       []Package `json:"packages"`
	Level          string    `json:"level"`
	Checks         Checks    `json:"checks"`
	Advisories     any       `json:"advisories"` // []string, or "unknown"
	Class          string    `json:"class"`
	Reason         string    `json:"reason"`
	DependencyItem bool      `json:"dependency_item"`
}

// Checks summarises a PR's head checks.
type Checks struct {
	Failing []string `json:"failing"`
	Pending []string `json:"pending"`
	Passing int      `json:"passing"`
}

// Flag is a baseline flag (research R7).
type Flag struct {
	Kind   string `json:"kind"`
	Where  string `json:"where"`
	Detail string `json:"detail"`
}

// WriteJSON writes the report as one JSON object.
func (r *Report) WriteJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// WriteText writes the owner's view (contracts/cli.md).
func (r *Report) WriteText(w io.Writer) {
	short := r.DefaultCommit
	if len(short) > 7 {
		short = short[:7]
	}
	fmt.Fprintf(w, "debt report for %s (target %s), %s @ %s, read with %s\n", r.Repo, r.Target, r.DefaultBranch, short, r.Credential)
	if len(r.DefaultChecksFailing) > 0 {
		fmt.Fprintf(w, "default branch: %d failing check%s: %s\n", len(r.DefaultChecksFailing), plural(len(r.DefaultChecksFailing)), strings.Join(r.DefaultChecksFailing, ", "))
	} else {
		fmt.Fprintf(w, "default branch: no failing checks\n")
	}
	if len(r.SharedChecks) > 0 {
		names := make([]string, 0, len(r.SharedChecks))
		for n := range r.SharedChecks {
			names = append(names, n)
		}
		sort.Strings(names)
		fmt.Fprintf(w, "shared failing checks (not counted against any one PR):\n")
		for _, n := range names {
			fmt.Fprintf(w, "  %s: %s\n", n, r.SharedChecks[n])
		}
	}
	n := 0
	for _, g := range r.Groups {
		n += len(g.PRs)
	}
	fmt.Fprintf(w, "\n%d Dependabot PR%s in %d group%s\n", n, plural(n), len(r.Groups), plural(len(r.Groups)))
	for _, g := range r.Groups {
		head := groupNoneFailing
		if len(g.FailingChecks) > 0 {
			head = "fails: " + strings.Join(g.FailingChecks, ", ")
			if len(g.Shared) > 0 {
				head += " [shared: " + strings.Join(g.Shared, ", ") + "]"
			}
		}
		fmt.Fprintf(w, "  %s   %d PR%s\n", head, len(g.PRs), plural(len(g.PRs)))
		for _, e := range g.PRs {
			fmt.Fprintf(w, "    %s\n", e.line())
		}
	}
	if len(r.NotTriaged) > 0 {
		var parts []string
		for _, e := range r.NotTriaged {
			parts = append(parts, fmt.Sprintf("#%d %s", e.Number, e.Reason))
		}
		fmt.Fprintf(w, "not triaged: %d (%s)\n", len(r.NotTriaged), strings.Join(parts, "; "))
	}
	if len(r.BaselineFlags) > 0 {
		fmt.Fprintf(w, "\nbaseline flags\n")
		for _, f := range r.BaselineFlags {
			fmt.Fprintf(w, "  %-20s %s: %s\n", f.Kind, f.Where, f.Detail)
		}
	} else {
		fmt.Fprintf(w, "\nbaseline flags: none found by the checks this report runs\n")
	}
	var counts []string
	for _, c := range []string{ClassSafe, ClassSuperseded, ClassNeedsCode, ClassPending, ClassNotTriaged} {
		counts = append(counts, fmt.Sprintf("%s %d", c, r.Counts[c]))
	}
	fmt.Fprintf(w, "\ncounts: %s\n", strings.Join(counts, " · "))
	if len(r.Unreadable) > 0 {
		fmt.Fprintf(w, "unreadable: %s\n", strings.Join(r.Unreadable, "; "))
	}
	fmt.Fprintf(w, "forge merges, closes and comments on nothing: act on the PRs yourself.\n")
}

func (e Entry) line() string {
	pkgs := "?"
	if len(e.Packages) > 0 {
		p := e.Packages[0]
		pkgs = fmt.Sprintf("%s %s → %s", p.Name, p.From, p.To)
		if len(e.Packages) > 1 {
			pkgs += fmt.Sprintf(" +%d more", len(e.Packages)-1)
		}
	}
	if e.Directory != "" && e.Directory != "/" {
		pkgs += " in " + e.Directory
	}
	s := fmt.Sprintf("#%-5d %-18s %s (%s)", e.Number, e.Class, pkgs, e.Level)
	if e.Reason != "" {
		s += ": " + e.Reason
	}
	if e.DependencyItem {
		s += " → suggest a dependency item"
	}
	return s
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// group puts entries into groups by their failing checks, largest first.
func group(entries []Entry, shared map[string]string) []Group {
	byKey := map[string]*Group{}
	var keys []string
	for _, e := range entries {
		f := append([]string{}, e.Checks.Failing...)
		sort.Strings(f)
		k := strings.Join(f, "\x00")
		g, ok := byKey[k]
		if !ok {
			g = &Group{FailingChecks: f, Shared: []string{}}
			for _, c := range f {
				if shared[c] != "" {
					g.Shared = append(g.Shared, c)
				}
			}
			byKey[k] = g
			keys = append(keys, k)
		}
		g.PRs = append(g.PRs, e)
	}
	out := make([]Group, 0, len(keys))
	for _, k := range keys {
		g := byKey[k]
		sort.Slice(g.PRs, func(i, j int) bool { return g.PRs[i].Number < g.PRs[j].Number })
		out = append(out, *g)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if len(out[i].PRs) != len(out[j].PRs) {
			return len(out[i].PRs) > len(out[j].PRs)
		}
		return strings.Join(out[i].FailingChecks, ",") < strings.Join(out[j].FailingChecks, ",")
	})
	return out
}
