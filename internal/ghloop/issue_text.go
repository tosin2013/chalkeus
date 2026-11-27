package ghloop

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/tosin2013/chalkeus/internal/decisions"
	"github.com/tosin2013/chalkeus/internal/evidence"
)

// The tracking issue is read by people who have never heard of Chalkeus (anyone browsing the
// target repository's issues), so its title and body lead with what is being built and whether a
// visitor needs to do anything; the run id and the tooling details come last. The machine parts
// (the hidden run marker, the decision markers, the exact `/approve` commands) are unchanged.

// TrackingLabel marks tracking issues so maintainers can filter them out of triage.
const TrackingLabel = "chalkeus"

// DocsURL is where a visitor learns what Chalkeus is.
const DocsURL = "https://github.com/tosin2013/chalkeus#readme"

var (
	phasePrefix = regexp.MustCompile(`^(?i:phase\s+\d+\s*:\s*)?(?i:user\s+story\s+\d+\s*[-–—:]\s*)?`)
	phaseSuffix = regexp.MustCompile(`\s*\((?i:priority)[^)]*\).*$`)
)

// storyName is a tasks.md phase heading without its numbering, priority and markers:
// "Phase 2: User Story 2 - Flagship skill (Priority: P2) 🎯 MVP" → "Flagship skill".
func storyName(heading string) string {
	s := phaseSuffix.ReplaceAllString(strings.TrimSpace(heading), "")
	return strings.TrimSpace(phasePrefix.ReplaceAllString(s, ""))
}

// issueFacts is what the run knows about the work, for the issue text. Every field may be empty.
type issueFacts struct {
	item, title, specDir, slice, story, branch, mode string
	sliceIndex, sliceOf                              int
}

func facts(run *evidence.Run) issueFacts {
	f := issueFacts{}
	f.item, _ = run.State["item"].(string)
	f.slice, _ = run.State["slice"].(string)
	r, _ := run.State["readiness"].(map[string]any)
	f.title, _ = r["title"].(string)
	f.specDir, _ = r["spec_dir"].(string)
	if s, _ := r["slice"].(map[string]any); s != nil {
		if ph, _ := s["phases"].([]any); len(ph) > 0 {
			h, _ := ph[0].(string)
			f.story = storyName(h)
		}
		i, _ := num(s["index"])
		f.sliceIndex = i + 1
		f.sliceOf, _ = num(s["of"])
	}
	if t, _ := run.State["target"].(map[string]any); t != nil {
		f.branch, _ = t["branch"].(string)
	}
	if m, _ := run.State["mode"].(map[string]any); m != nil {
		f.mode, _ = m["name"].(string)
	}
	return f
}

// issueTitle names the work, not the run: "Chalkeus: <story> (<item> · <slice>)".
func issueTitle(run *evidence.Run) string {
	f := facts(run)
	what := f.story
	if what == "" {
		what = f.title
	}
	var ids []string
	for _, s := range []string{f.item, f.slice} {
		if s != "" {
			ids = append(ids, s)
		}
	}
	switch {
	case what != "" && len(ids) > 0:
		return fmt.Sprintf("Chalkeus: %s (%s)", what, strings.Join(ids, " · "))
	case len(ids) > 0:
		return fmt.Sprintf("Chalkeus: item %s (run %s)", strings.Join(ids, " · "), run.ID)
	}
	return "Chalkeus run " + run.ID
}

// prNumber is the run's latest task pull request, or 0.
func prNumber(run *evidence.Run) int {
	ts, _ := run.State["tasks"].(map[string]any)
	keys := make([]string, 0, len(ts))
	for k := range ts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	best := 0
	for _, k := range keys {
		t, _ := ts[k].(map[string]any)
		pr, _ := t["pr"].(map[string]any)
		if n, ok := num(pr["number"]); ok && n > best {
			best = n
		}
	}
	return best
}

// openStatus is the Status row while the run is open.
func openStatus(run *evidence.Run, p *decisions.Pending) string {
	if p == nil {
		return "🟡 in progress"
	}
	switch p.Step {
	case "mode":
		return "⏸ waiting for a maintainer to approve the start"
	case "retry":
		return fmt.Sprintf("⏸ attempt %v didn't pass; waiting for a maintainer to approve another attempt or a replan", prevAttempt(p))
	case "merge":
		if n := prNumber(run); n > 0 {
			return fmt.Sprintf("🟢 draft pull request #%d is ready for review", n)
		}
		return "🟢 a draft pull request is ready for review"
	}
	return "⏸ waiting for a maintainer's decision"
}

// closedStatus is the Status row once the run has ended (outcome as CloseTracking gets it).
func closedStatus(run *evidence.Run, outcome string) string {
	switch {
	case outcome == "succeeded":
		if n := prNumber(run); n > 0 {
			return fmt.Sprintf("✅ finished: pull request #%d merged", n)
		}
		return "✅ finished"
	case strings.HasPrefix(outcome, "stopped"):
		return "⏹ ended: back to planning (a maintainer chose to rework the plan before building again)"
	}
	return "❌ ended without a result: " + outcome
}

func prevAttempt(p *decisions.Pending) any {
	if n, ok := num(p.N); ok && n > 1 {
		return n - 1
	}
	return "the last"
}

// issueBody is the tracking issue's body for the given Status row.
func (l *Loop) issueBody(run *evidence.Run, status string) string {
	f := facts(run)
	var b strings.Builder
	fmt.Fprintf(&b, "<!-- chalkeus:run=%s -->\n", run.ID)
	b.WriteString("**An AI coding agent is building one planned change, and a maintainer approves every step.**\n\n")
	b.WriteString("| | |\n|---|---|\n")
	if f.title != "" || f.item != "" {
		change := f.title
		if change == "" {
			change = "item " + f.item
		} else if f.item != "" {
			change = f.item + ": " + change
		}
		if f.specDir != "" {
			change = fmt.Sprintf("[%s](%s)", change, l.blob(f.branch, f.specDir+"/spec.md"))
		}
		fmt.Fprintf(&b, "| **Change** | %s |\n", change)
	}
	if f.slice != "" {
		part := f.slice
		if f.story != "" {
			part += ": " + f.story
		}
		if f.sliceOf > 1 {
			part += fmt.Sprintf(" (part %d of %d)", f.sliceIndex, f.sliceOf)
		}
		if f.specDir != "" {
			part += fmt.Sprintf(" · [tasks](%s)", l.blob(f.branch, f.specDir+"/tasks.md"))
		}
		fmt.Fprintf(&b, "| **This part** | %s |\n", part)
	}
	fmt.Fprintf(&b, "| **Status** | %s |\n", status)
	b.WriteString("| **Result** | a draft pull request for human review, linked here when it is ready |\n\n")
	var at []string
	for _, a := range l.Approvers {
		at = append(at, "@"+a)
	}
	approvers := "a maintainer"
	if len(at) > 0 {
		approvers = strings.Join(at, ", ")
	}
	fmt.Fprintf(&b, "**Just visiting?** You don't need to do anything. Each step is approved by %s replying "+
		"with the command shown in the posts below.\n\n", approvers)
	b.WriteString("_Chalkeus never merges; merging is always a maintainer's decision._\n\n")
	b.WriteString("<details><summary>How this works</summary>\n\n")
	b.WriteString("The agent works in an isolated sandbox without access to secrets. Its change is then graded " +
		"against acceptance tests it can't see, plus the project's own tests and rules. When an attempt fails, " +
		"Chalkeus posts a diagnosis here and the maintainer chooses: try again, or go back to planning. " +
		"More: [Chalkeus](" + DocsURL + ").\n</details>\n\n")
	foot := "run `" + run.ID + "`"
	if f.mode != "" {
		foot += " · " + f.mode + " mode"
	}
	fmt.Fprintf(&b, "<sub>%s</sub>\n", foot)
	return b.String()
}

func (l *Loop) blob(branch, path string) string {
	if branch == "" {
		branch = l.Branch
	}
	if branch == "" {
		branch = "main"
	}
	return fmt.Sprintf("https://github.com/%s/blob/%s/%s", l.Repo, branch, path)
}

// decisionLead is the plain-language first line of a decision post.
func decisionLead(p *decisions.Pending) string {
	switch p.Step {
	case "mode":
		return "**Ready to start. Approve the run?**"
	case "retry":
		return fmt.Sprintf("**Attempt %v didn't pass. Try again?**", prevAttempt(p))
	case "acceptance":
		return "**The last part is merged. Confirm the item's acceptance checks?**"
	case "reconcile":
		return "**Record the item's new status on the roadmap?**"
	}
	return "**A decision is needed.**"
}

// updateStatus rewrites the tracking issue's Status row. Best effort: the issue text is for
// people, and a failed edit never affects the run.
func (l *Loop) updateStatus(run *evidence.Run, w Writer, issue int, status string) {
	body := l.issueBody(run, status)
	_ = w.EditIssue(l.ctx(), l.Repo, issue, IssueEdit{Body: &body})
}
