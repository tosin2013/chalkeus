package cli

import (
	"context"
	"sort"

	"github.com/tosin2013/chalkeus/internal/ci"
	"github.com/tosin2013/chalkeus/internal/config"
	"github.com/tosin2013/chalkeus/internal/doctor"
	"github.com/tosin2013/chalkeus/internal/ghloop"
	"github.com/tosin2013/chalkeus/internal/paths"
)

// doctorCI adds the 036 CI review: whenever the target has CI configuration, doctor reads it
// (read-only) and compares it with the profile. Every problem is a warning or information,
// never a failure. GitHub Actions only for now; other hosts get one information line.
func doctorCI(ctx context.Context, r *doctor.Report, l paths.Layout, t config.Target) {
	if !t.Host().IsGitHub() {
		r.Add("ci", "CI review", doctor.Info, "not available for "+t.Host().Name+" yet (GitHub Actions only)")
		return
	}
	c, err := ghloop.OperatorClient()
	if err != nil {
		r.Add("ci", "CI review", doctor.Info, "skipped: "+err.Error())
		return
	}
	goVersion := ""
	if pins, err := paths.ReadEnv(l.Pins); err == nil {
		goVersion = pins["GO_VERSION"]
	}
	for _, f := range reviewCI(ctx, ci.GitHub{C: c}, t, goVersion) {
		r.Add("ci", f.Check, f.Status, f.Detail)
	}
}

// reviewCI reads the target's CI through rd and returns the findings (testable with a fake).
func reviewCI(ctx context.Context, rd ci.Reader, t config.Target, goVersion string) []ci.Finding {
	files, err := rd.Workflows(ctx, t.Repo, t.Branch)
	if err != nil {
		return []ci.Finding{{Check: "CI review", Status: ci.Info, Detail: "couldn't read the CI configuration: " + err.Error()}}
	}
	if len(files) == 0 {
		return []ci.Finding{{Check: "CI review", Status: ci.Info, Detail: "no CI found (.github/workflows): grading is the only check on changes"}}
	}
	names := make([]string, 0, len(files))
	for p := range files {
		names = append(names, p)
	}
	sort.Strings(names)
	in := ci.Input{Branch: t.Branch, TestCommand: t.TestCommand, Toolchain: t.Toolchain, GoVersion: goVersion}
	var bad []ci.Finding
	for _, p := range names {
		w, err := ci.ParseWorkflow(p, files[p])
		if err != nil {
			bad = append(bad, ci.Finding{Check: "workflows", Status: ci.Info, Detail: "couldn't parse " + err.Error()})
			continue
		}
		in.Workflows = append(in.Workflows, w)
	}
	if raw, err := rd.File(ctx, t.Repo, t.Branch, "package.json"); err == nil {
		in.Scripts = ci.Scripts(raw)
	}
	in.Runs, in.RunsErr = rd.LatestRuns(ctx, t.Repo, t.Branch)
	return append(bad, ci.Review(in)...)
}
