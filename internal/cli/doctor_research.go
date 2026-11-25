package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/google/go-github/v85/github"

	"github.com/tosin2013/chalkeus/internal/config"
	"github.com/tosin2013/chalkeus/internal/diagnose"
	"github.com/tosin2013/chalkeus/internal/doctor"
	"github.com/tosin2013/chalkeus/internal/ghloop"
	"github.com/tosin2013/chalkeus/internal/images"
	"github.com/tosin2013/chalkeus/internal/paths"
)

// doctorTaskWarnings adds 033's warnings for the slice's (or the whole item's) unchecked tasks:
// research wording (FR-012) and a runtime version the toolchain doesn't provide (FR-014). They are
// warnings only (FR-015). tasksPath is the file's repository path, for the line references.
func doctorTaskWarnings(r *doctor.Report, tasksText, tasksPath string, phases []string, toolchain, goVersion string) {
	var open []diagnose.Task
	for _, t := range diagnose.Tasks(tasksText) {
		if !t.Checked && diagnose.InPhases(t, phases) {
			open = append(open, t)
		}
	}
	var research []string
	for _, t := range open {
		if _, ok := diagnose.ResearchTask(t.Desc); ok {
			research = append(research, fmt.Sprintf("%s (%s:%d) %q", t.ID, path.Base(tasksPath), t.Line, clipText(t.Desc, 60)))
		}
	}
	if len(research) > 0 {
		r.Add("item", "research tasks", doctor.Warn, strings.Join(research, "; ")+
			"; their answer isn't known yet: resolve them in the plan's research first (docs/targets.md#research-tasks)")
	}
	for _, m := range diagnose.RuntimeMismatches(open, toolchain, goVersion) {
		r.Add("item", "runtime", doctor.Warn, fmt.Sprintf("%s (%s:%d) names %s; toolchain %s provides %s (change the runtime in your planning PR)",
			m.Task, path.Base(tasksPath), m.Line, m.Named, toolchain, m.Provides))
	}
}

// doctorAllowance adds the baseline's test-run time and allowance (033 FR-013).
func doctorAllowance(r *doctor.Report, testSeconds, deadlineS int) {
	if testSeconds <= 0 {
		r.Add("baseline", "test-run time", doctor.Info, "not recorded by this grader image")
		return
	}
	r.Add("baseline", "test-run time", doctor.Pass, fmt.Sprintf("one run of the test command took %d s", testSeconds))
	if deadlineS <= diagnose.SoftMarginS {
		return
	}
	n := (deadlineS - diagnose.SoftMarginS) / testSeconds
	if n < diagnose.AllowanceThreshold {
		r.Add("baseline", "test-run allowance", doctor.Warn, fmt.Sprintf("only %d full test runs fit in one attempt (deadline %d s); raise FORGE_WORKER_DEADLINE_S or make the tests faster (docs/targets.md#research-tasks)", n, deadlineS))
	}
}

// doctorTasks reads the item's tasks.md at the readiness commit and adds the task warnings.
// A read failure is a warning, never a failure.
func doctorTasks(ctx context.Context, r *doctor.Report, gh *github.Client, l paths.Layout, t config.Target, specDir, commit string, phases []string) {
	owner, name := ghloop.SplitRepo(t.Repo)
	p := strings.TrimSuffix(specDir, "/") + "/tasks.md"
	fc, _, _, err := gh.Repositories.GetContents(ctx, owner, name, p, &github.RepositoryContentGetOptions{Ref: commit})
	if err != nil || fc == nil {
		r.Add("item", "research tasks", doctor.Warn, fmt.Sprintf("could not read %s: %v", p, err))
		return
	}
	text, err := fc.GetContent()
	if err != nil {
		r.Add("item", "research tasks", doctor.Warn, fmt.Sprintf("could not read %s: %v", p, err))
		return
	}
	goVersion := ""
	if pins, err := paths.ReadEnv(l.Pins); err == nil {
		goVersion = pins["GO_VERSION"]
	}
	doctorTaskWarnings(r, text, p, phases, t.Toolchain, goVersion)
}

// doctorDeadline is the target's resolved worker deadline (030).
func doctorDeadline(l paths.Layout, t config.Target) int {
	cat, err := images.LoadCatalog(l)
	if err != nil {
		return 0
	}
	tc, ok := cat.For(t)
	if !ok {
		return 0
	}
	return images.Resolve(tc, t.Toolchain, t).Worker.DeadlineS
}

func clipText(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// toInt reads a JSON number as an int: float64, int, or json.Number (grade reports are decoded
// with UseNumber); 0 otherwise.
func toInt(v any) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	case json.Number:
		i, _ := x.Int64()
		return int(i)
	}
	return 0
}
