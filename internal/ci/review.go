package ci

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Finding is one line of the CI review. Status is one of the doctor statuses (pass, warn, info).
type Finding struct {
	Check  string
	Status string
	Detail string
}

// Statuses, the same strings as internal/doctor's.
const (
	Pass = "PASS"
	Warn = "WARN"
	Info = "INFO"
)

// JobRun is the latest result of one CI job on the default branch.
type JobRun struct {
	Workflow, Job string
	Conclusion    string // success, failure, cancelled, skipped, …
	FailedStep    string // the first failed step's name, for a failed job
	ErrorLine     string // the most telling line of the failed step's log, when found
	URL           string
	When          string // RFC 3339
}

// Input is what the review reads.
type Input struct {
	Branch      string
	Workflows   []Workflow
	Scripts     map[string]string // package.json scripts, when the project has them
	TestCommand string            // the profile's FORGE_TEST_COMMAND
	Toolchain   string            // the target's toolchain id (node24, go, jdk21-maven)
	GoVersion   string            // the go toolchain's pinned version, for `go`
	Runs        []JobRun
	RunsErr     error // the runs couldn't be read (no Actions access, …)
}

// Review produces the CI review's findings, in report order.
func Review(in Input) []Finding {
	var out []Finding
	var gating, other []string
	jobs := 0
	for _, w := range in.Workflows {
		if w.Gates {
			gating = append(gating, w.File)
			jobs += len(w.Jobs)
		} else {
			other = append(other, w.File)
		}
	}
	d := fmt.Sprintf("%d workflow(s) check changes (%s; %d job(s))", len(gating), strings.Join(base(gating), ", "), jobs)
	if len(other) > 0 {
		d += fmt.Sprintf("; %d other(s) not compared (release, schedule or manual: %s)", len(other), strings.Join(base(other), ", "))
	}
	out = append(out, Finding{"workflows", Info, d})

	out = append(out, branchHealth(in)...)
	ci := Commands(in.Workflows, in.Scripts)
	prof := ProfileCommands(in.TestCommand, in.Scripts)
	out = append(out, testCommand(ci, prof))
	if f, ok := ciOnly(ci, prof); ok {
		out = append(out, f)
	}
	out = append(out, runtime(in)...)
	return out
}

func base(files []string) []string {
	out := make([]string, len(files))
	for i, f := range files {
		out[i] = f[strings.LastIndex(f, "/")+1:]
	}
	return out
}

func branchHealth(in Input) []Finding {
	if in.RunsErr != nil {
		return []Finding{{"default branch", Info, "couldn't read CI results: " + in.RunsErr.Error() + " (the checks below come from the workflow files)"}}
	}
	if len(in.Runs) == 0 {
		return []Finding{{"default branch", Info, "no completed CI run on " + in.Branch + " yet"}}
	}
	var red []Finding
	green := 0
	for _, r := range in.Runs {
		switch r.Conclusion {
		case "success", "skipped", "neutral":
			green++
		case "failure", "timed_out", "startup_failure":
			d := fmt.Sprintf("%s › %s failed on %s", r.Workflow, r.Job, in.Branch)
			if r.When != "" {
				d += " (" + r.When[:min(10, len(r.When))] + ")"
			}
			cmd := ""
			if r.FailedStep != "" {
				d += ", step " + quote(r.FailedStep)
				if cmd = stepCommand(in.Workflows, r.Workflow, r.Job, r.FailedStep); cmd != "" {
					d += " (`" + clip(cmd, 90) + "`)"
				}
			}
			if r.ErrorLine != "" {
				d += ": " + strings.TrimRight(clip(spaces.ReplaceAllString(r.ErrorLine, " "), 160), ".")
			}
			status := Warn
			switch kind := Classify(Resolve(cmd, in.Scripts)); kind {
			case "audit":
				d += ". Fix the default branch first: every PR inherits this red check (a dependency bump fits a dependency item, docs/targets.md#a-dependency-upgrade-as-an-item)"
			case "test", "typecheck", "lint", "build", "install":
				d += ". Fix the default branch first: a forge baseline fails on it too, and every PR inherits the red check"
			default:
				status = Info
				d += ". Not a check grading relates to (deployment, credentials or automation), but every PR shows it red"
			}
			if r.URL != "" {
				d += " " + r.URL
			}
			red = append(red, Finding{"default branch", status, d})
		}
	}
	if len(red) > 0 {
		sort.SliceStable(red, func(i, k int) bool { return red[i].Status == Warn && red[k].Status != Warn })
		return red
	}
	return []Finding{{"default branch", Pass, fmt.Sprintf("the latest CI runs on %s are green (%d job(s))", in.Branch, green)}}
}

func stepCommand(wfs []Workflow, wf, job, step string) string {
	for _, w := range wfs {
		if w.Name != wf {
			continue
		}
		for _, j := range w.Jobs {
			if j.Name != job && !strings.HasPrefix(job, j.Name+" (") { // matrix jobs: "Test (24.x)"
				continue
			}
			for _, s := range j.Steps {
				if s.Name == step {
					return firstLine(s.Run)
				}
			}
		}
	}
	return ""
}

func quote(s string) string { return "“" + s + "”" }

func clip(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n]) + "…"
}

// describe renders commands as `text` (→ `resolved`), each once, with where CI runs it.
func describe(cs []Command, withWhere bool) string {
	var keys []string
	where := map[string][]string{}
	for _, c := range cs {
		k := "`" + clip(c.Text, 70) + "`"
		if c.Resolved != c.Text {
			k += " → `" + clip(c.Resolved, 70) + "`"
		}
		if _, ok := where[k]; !ok {
			keys = append(keys, k)
		}
		if !contains(where[k], c.Where) {
			where[k] = append(where[k], c.Where)
		}
	}
	var parts []string
	for _, k := range keys {
		if withWhere {
			ws := where[k]
			if len(ws) > 2 {
				ws = append(ws[:2:2], fmt.Sprintf("%d more", len(where[k])-2))
			}
			k += " (" + strings.Join(ws, ", ") + ")"
		}
		parts = append(parts, k)
	}
	return strings.Join(parts, "; ")
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

var spaces = regexp.MustCompile(`\s+`)

var warnFlags = []string{"coverage", "--forceExit", "-race", "--detectOpenHandles"}

func testCommand(ci, prof []Command) Finding {
	var ciTests, profTests []Command
	for _, c := range ci {
		if c.Kind == "test" {
			ciTests = append(ciTests, c)
		}
	}
	for _, c := range prof {
		if c.Kind == "test" {
			profTests = append(profTests, c)
		}
	}
	switch {
	case len(ciTests) == 0:
		return Finding{"test command", Info, "CI runs no test command the review recognises; forge grades with " + describe(prof, false)}
	case len(profTests) == 0:
		return Finding{"test command", Warn, "the profile's test command (" + describe(prof, false) + ") isn't a test run the review recognises; CI tests with " + describe(ciTests, true)}
	}
	ciFlags, profFlags := map[string]string{}, map[string]string{}
	for _, c := range ciTests {
		for k, v := range Flags(c.Resolved) {
			ciFlags[k] = v
		}
	}
	for _, c := range profTests {
		for k, v := range Flags(c.Resolved) {
			profFlags[k] = v
		}
	}
	var diffs []string
	for _, f := range warnFlags {
		_, inCI := ciFlags[f]
		_, inProf := profFlags[f]
		switch {
		case inCI && !inProf:
			diffs = append(diffs, fmt.Sprintf("CI uses %s (%s) and forge doesn't: grading can pass while CI fails", "`"+ciFlags[f]+"`", FlagWhy(f)))
		case inProf && !inCI:
			diffs = append(diffs, fmt.Sprintf("forge uses %s (%s) and CI doesn't", "`"+profFlags[f]+"`", FlagWhy(f)))
		}
	}
	d := "CI: " + describe(ciTests, true) + ". forge: " + describe(profTests, false)
	if len(diffs) == 0 {
		return Finding{"test command", Pass, d + ". They agree on coverage and exit behaviour"}
	}
	return Finding{"test command", Warn, d + ". " + strings.Join(diffs, "; ") + ". Set FORGE_TEST_COMMAND to what CI runs, or change CI (docs/strategies.md#3-grade-the-way-ci-does)"}
}

func ciOnly(ci, prof []Command) (Finding, bool) {
	have := map[string]bool{"test": true, "install": true, "": true}
	for _, c := range prof {
		have[c.Kind] = true
	}
	by := map[string][]Command{}
	var order []string
	for _, c := range ci {
		if have[c.Kind] {
			continue
		}
		if _, ok := by[c.Kind]; !ok {
			order = append(order, c.Kind)
		}
		by[c.Kind] = append(by[c.Kind], c)
	}
	if len(order) == 0 {
		return Finding{}, false
	}
	var parts []string
	for _, k := range order {
		parts = append(parts, k+": "+describe(by[k], true))
	}
	return Finding{"checks grading doesn't run", Warn, strings.Join(parts, " · ") +
		". A slice can pass grading and still turn these red: add the ones a change can break to FORGE_TEST_COMMAND, or accept the gap"}, true
}

var majorRE = regexp.MustCompile(`^v?(\d+)(?:\.(\d+))?`)

type runtimeUse struct{ lang, version, where string }

func runtime(in Input) []Finding {
	want := map[string]string{} // language → the toolchain's version (major, or major.minor for go)
	switch {
	case strings.HasPrefix(in.Toolchain, "node"):
		want["node"] = strings.TrimPrefix(in.Toolchain, "node")
	case in.Toolchain == "go":
		if m := majorRE.FindStringSubmatch(in.GoVersion); m != nil && m[2] != "" {
			want["go"] = m[1] + "." + m[2]
		}
	case strings.HasPrefix(in.Toolchain, "jdk"):
		want["java"] = strings.TrimPrefix(strings.SplitN(in.Toolchain, "-", 2)[0], "jdk")
	}
	var uses []runtimeUse
	for _, w := range in.Workflows {
		if !w.Gates {
			continue
		}
		for _, j := range w.Jobs {
			for _, s := range j.Steps {
				lang, key := "", ""
				switch s.Uses {
				case "actions/setup-node":
					lang, key = "node", "node-version"
				case "actions/setup-go":
					lang, key = "go", "go-version"
				case "actions/setup-java":
					lang, key = "java", "java-version"
				case "actions/setup-python":
					lang, key = "python", "python-version"
				default:
					continue
				}
				v := s.With[key]
				if v == "" {
					if f := s.With[key+"-file"]; f != "" {
						uses = append(uses, runtimeUse{lang, "from " + f, w.Name + " › " + j.Name})
					}
					continue
				}
				for _, x := range expand(v, j.Matrix) {
					uses = append(uses, runtimeUse{lang, x, w.Name + " › " + j.Name})
				}
			}
		}
	}
	if len(uses) == 0 {
		return nil
	}
	var bad, good []string
	for _, u := range uses {
		w, ok := want[u.lang]
		if !ok || strings.HasPrefix(u.version, "from ") {
			continue
		}
		got := ""
		if m := majorRE.FindStringSubmatch(u.version); m != nil {
			got = m[1]
			if u.lang == "go" && m[2] != "" {
				got += "." + m[2]
			}
		}
		if got == "" {
			continue // lts/*, stable, …: can't compare
		}
		s := u.lang + " " + u.version
		if got == w {
			good = append(good, s)
		} else {
			bad = append(bad, s)
		}
	}
	sort.Strings(bad)
	switch {
	case len(bad) > 0:
		return []Finding{{"runtime", Warn, fmt.Sprintf("CI uses %s; the toolchain %s provides %s. Align CI or the toolchain in a planning PR", strings.Join(dedupe(bad), ", "), in.Toolchain, langVersions(want))}}
	case len(good) > 0:
		return []Finding{{"runtime", Pass, fmt.Sprintf("CI uses %s (%d job step(s)), as the toolchain %s does", strings.Join(dedupe(good), ", "), len(good), in.Toolchain)}}
	}
	var all []string
	for _, u := range uses {
		all = append(all, fmt.Sprintf("%s %s (%s)", u.lang, u.version, u.where))
	}
	return []Finding{{"runtime", Info, "CI uses " + strings.Join(dedupe(all), ", ") + "; not compared with the toolchain " + in.Toolchain}}
}

var matrixRef = regexp.MustCompile(`^\$\{\{\s*matrix\.([A-Za-z0-9_-]+)\s*\}\}$`)

func expand(v string, m map[string][]string) []string {
	if mm := matrixRef.FindStringSubmatch(strings.TrimSpace(v)); mm != nil {
		if vs := m[mm[1]]; len(vs) > 0 {
			return vs
		}
	}
	return []string{v}
}

func dedupe(ss []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func langVersions(m map[string]string) string {
	var out []string
	for k, v := range m {
		out = append(out, k+" "+v)
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}

// ErrorLine picks the most telling line of a failed step's log: an advisory's package and
// severity, a failing test file, a compiler error, else the last line before the runner's
// "Process completed with exit code" marker.
func ErrorLine(log string) string {
	lines := strings.Split(strings.ReplaceAll(log, "\r", ""), "\n")
	clean := make([]string, len(lines))
	for i, l := range lines {
		clean[i] = strings.TrimSpace(tsRE.ReplaceAllString(l, ""))
	}
	for i, l := range clean { // npm audit: "<pkg>  <range>" then "Severity: high"
		if sevRE.MatchString(l) {
			for j := i - 1; j >= 0 && j >= i-3; j-- {
				if clean[j] != "" {
					return clean[j] + " (" + strings.ToLower(l) + ")"
				}
			}
			return l
		}
	}
	for _, re := range telling {
		for _, l := range clean {
			if re.MatchString(l) {
				return l
			}
		}
	}
	for i, l := range clean {
		if strings.Contains(l, "##[error]Process completed with exit code") {
			for j := i - 1; j >= 0; j-- {
				if c := clean[j]; c != "" && !strings.HasPrefix(c, "##[") {
					return c
				}
			}
		}
	}
	for _, l := range clean {
		if strings.HasPrefix(l, "##[error]") {
			return strings.TrimPrefix(l, "##[error]")
		}
	}
	return ""
}

var (
	tsRE    = regexp.MustCompile(`^\d{4}-\d\d-\d\dT[\d:.]+Z\s?`)
	sevRE   = regexp.MustCompile(`(?i)^Severity:\s*(high|critical)\b`)
	telling = []*regexp.Regexp{
		regexp.MustCompile(`^FAIL\s+\S`),                 // jest, go test
		regexp.MustCompile(`\berror TS\d+:`),             // tsc
		regexp.MustCompile(`^--- FAIL:`),                 // go test
		regexp.MustCompile(`^\S+:\d+:\d+:?\s+(?:error)`), // eslint/compilers, file:line:col error
		regexp.MustCompile(`(?i)^Jest: "global" coverage threshold`),
		regexp.MustCompile(`^npm (?:ERR!|error)\s+\S`),
	}
)
