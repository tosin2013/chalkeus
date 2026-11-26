// Package prompt builds the worker task prompt and the declared induction
// (reference: the former Python pilot's prompt.py).
package prompt

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	Induction = "withheld-acceptance-section"
	Heading   = "## Acceptance details"
)

// Preamble is the fixed instruction block (verbatim from the reference).
const Preamble = `You are a coding agent working in a git repository (the current directory).
Implement the specification below completely, then mark every task in the task list as
done by changing ` + "`- [ ]`" + ` to ` + "`- [X]`" + ` in that tasks.md file.

Rules:
- Change only files under ` + "`src/slugkit/`, `tests/`" + `, and the task list file named below.
- Use the Python standard library only. Do not install packages.
- Do not run git commands; the harness commits your work.
- When finished, stop.
`

// span returns the byte offsets of the acceptance section.
func span(spec string) (int, int, error) {
	start := strings.Index(spec, Heading)
	if start < 0 {
		return 0, 0, fmt.Errorf("induction requested but '%s' not found", Heading)
	}
	rest := spec[start+len(Heading):]
	nxt := strings.Index(rest, "\n## ")
	if nxt < 0 {
		return start, len(spec), nil
	}
	return start, start + len(Heading) + nxt + 1, nil
}

// SectionSHA256 is the SHA-256 of the acceptance section.
func SectionSHA256(text string) (string, error) {
	s, e, err := span(text)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256([]byte(text[s:e]))
	return hex.EncodeToString(h[:]), nil
}

// Record describes an applied induction (induction.json).
type Record struct {
	Name           string `json:"name"`
	RemovedHeading string `json:"removed_heading"`
	RemovedSHA256  string `json:"removed_sha256"`
	RemovedBytes   int    `json:"removed_bytes"`
}

// Build returns the prompt text and, when induction is requested, its record.
func Build(spec, tasks, plan, induction, tasksPath string) (string, *Record, error) {
	if tasksPath == "" {
		tasksPath = "tasks.md"
	}
	var rec *Record
	if induction != "" {
		if induction != Induction {
			return "", nil, fmt.Errorf("unknown induction '%s'", induction)
		}
		s, e, err := span(spec)
		if err != nil {
			return "", nil, err
		}
		removed := spec[s:e]
		spec = spec[:s] + spec[e:]
		h := sha256.Sum256([]byte(removed))
		rec = &Record{Name: Induction, RemovedHeading: Heading, RemovedSHA256: hex.EncodeToString(h[:]), RemovedBytes: len(removed)}
	}
	text := fmt.Sprintf("%s\n# Specification\n\n%s\n\n# Plan\n\n%s\n\n# Task list (`%s`)\n\n%s\n",
		Preamble, strings.TrimSpace(spec), strings.TrimSpace(plan), tasksPath, strings.TrimSpace(tasks))
	return text, rec, nil
}

// Item is what an item run's prompt is built from (014 R10).
type Item struct {
	Spec, Plan, Tasks string // plan may be empty: it is optional for item runs
	TasksPath         string // <spec dir>/tasks.md
	AllowedPaths      []string
	Toolchain         string
	TestCommand       string
	TestPaths         []string // the project's test files; when set, the change must add tests
	TestsMayChange    []string // existing tests the spec allows to change
	// 026: a slice run builds only these phases of the task list (Slice is the id, e.g. "US2").
	Slice       string
	SlicePhases []string
	// 029: a dependency item's approved manifest files, fixed: the worker fixes the code instead.
	FixedPaths []string
}

// BuildItem is the prompt for one item of a target repository. It has no induction and no
// fixture-specific rule; the allowed paths are the item's own list. The fixture keeps Build.
func BuildItem(it Item) (string, error) {
	if strings.TrimSpace(it.Spec) == "" || strings.TrimSpace(it.Tasks) == "" || it.TasksPath == "" {
		return "", fmt.Errorf("item prompt needs a spec, a task list and its path")
	}
	if len(it.AllowedPaths) == 0 {
		return "", fmt.Errorf("item prompt needs the allowed paths")
	}
	var b strings.Builder
	b.WriteString("You are a coding agent working in a git repository (the current directory).\n")
	if it.Slice != "" {
		fmt.Fprintf(&b, "Build only slice %s of the specification below: the tasks under the phases listed after\n", it.Slice)
		b.WriteString("the task list. Then mark exactly those tasks as done by changing `- [ ]` to `- [X]` in that\n")
		b.WriteString("tasks.md file. Leave every other task as it is.\n\nRules:\n")
	} else {
		b.WriteString("Implement the specification below completely, then mark every task in the task list as\n")
		b.WriteString("done by changing `- [ ]` to `- [X]` in that tasks.md file.\n\nRules:\n")
	}
	b.WriteString("- Change only files matching these paths (`*` also matches `/`):\n")
	for _, p := range it.AllowedPaths {
		fmt.Fprintf(&b, "  - `%s`\n", p)
	}
	if len(it.FixedPaths) > 0 {
		fmt.Fprintf(&b, "- The dependency files %s already hold an approved dependency update, and they are fixed:\n", quoteList(it.FixedPaths))
		b.WriteString("  don't edit, delete or rename them. Fix the code so it works with the updated dependencies.\n")
	}
	fmt.Fprintf(&b, "- The project uses the %s toolchain. Check your work with: `%s`\n", it.Toolchain, it.TestCommand)
	if len(it.TestPaths) > 0 {
		fmt.Fprintf(&b, "- Add or extend tests for your change in the project's test files (%s). Keep every existing\n", quoteList(it.TestPaths))
		b.WriteString("  test passing unchanged: the grader re-runs the original versions of any test file you edit or delete")
		if len(it.TestsMayChange) > 0 {
			fmt.Fprintf(&b, ",\n  except %s, which the specification allows to change", quoteList(it.TestsMayChange))
		}
		b.WriteString(".\n")
	}
	b.WriteString("- Only the project's existing dependencies are available to the grader; do not add new ones.\n")
	b.WriteString("- Do not run git commands; the harness commits your work.\n- When finished, stop.\n")
	fmt.Fprintf(&b, "\n# Specification\n\n%s\n", strings.TrimSpace(it.Spec))
	if p := strings.TrimSpace(it.Plan); p != "" {
		fmt.Fprintf(&b, "\n# Plan\n\n%s\n", p)
	}
	fmt.Fprintf(&b, "\n# Task list (`%s`)\n\n%s\n", it.TasksPath, strings.TrimSpace(it.Tasks))
	if it.Slice != "" {
		fmt.Fprintf(&b, "\n# This run: slice %s\n\nBuild only slice %s: do and tick only the tasks under these phases of the task list.\n", it.Slice, it.Slice)
		b.WriteString("Earlier slices are already merged; later slices are built in later runs, so leave their\n")
		b.WriteString("tasks unticked and don't start their work.\n\n")
		for _, ph := range it.SlicePhases {
			fmt.Fprintf(&b, "- %s\n", ph)
		}
	}
	return b.String(), nil
}

// SpecSHA256 is the SHA-256 of a spec's text, recorded so the audit can check that the worker
// was given the spec unaltered (014 AU11 item variant).
func SpecSHA256(spec string) string {
	h := sha256.Sum256([]byte(strings.TrimSpace(spec)))
	return hex.EncodeToString(h[:])
}

func quoteList(xs []string) string {
	q := make([]string, len(xs))
	for i, x := range xs {
		q[i] = "`" + x + "`"
	}
	return strings.Join(q, ", ")
}
