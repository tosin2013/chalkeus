package diagnose

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/tosin2013/chalkeus/internal/images"
)

// Signal kinds (data-model.md).
const (
	KindResearchTask    = "research_task"
	KindOutsidePaths    = "outside_paths"
	KindManifestOutside = "manifest_outside_paths"
	KindEgressRefused   = "egress_refused"
	KindInstallError    = "install_error"
	KindUnknownReason   = "unknown_reason"
	KindFailedChecks    = "failed_checks"
	KindAllowance       = "allowance"
)

// Triggers.
const (
	UnknownWork     = "unknown_work"
	EnvironmentNeed = "environment_need"
)

// Choices.
const (
	Retry          = "retry"
	RaiseBudget    = "raise_budget"
	BackToPlanning = "back_to_planning"
)

// Routes back to planning.
const (
	RoutePlanResearch = "plan_research"
	RoutePlanningPR   = "planning_pr"
)

// AllowanceThreshold is the fewest full test runs one attempt should afford (033 assumption).
const AllowanceThreshold = 5

// SoftMarginS mirrors images.SoftMarginS: the worker stops the agent this long before its deadline.
const SoftMarginS = images.SoftMarginS

// Signal is one fact from the evidence, with where it came from.
type Signal struct {
	Kind    string `json:"kind"`
	Detail  string `json:"detail"`
	Source  string `json:"source"`
	Trigger string `json:"trigger,omitempty"`
}

// Usage is what the attempt used, when kept.
type Usage struct {
	Tokens  any `json:"tokens,omitempty"`
	CostUSD any `json:"cost_usd,omitempty"`
}

// Diagnosis is one failed attempt's diagnosis (contracts/diagnosis.md).
type Diagnosis struct {
	Version   int      `json:"version"`
	Task      string   `json:"task"`
	Attempt   int      `json:"attempt"`
	Ended     string   `json:"ended"`
	DeadlineS int      `json:"deadline_s,omitempty"`
	Usage     Usage    `json:"usage"`
	Signals   []Signal `json:"signals"`
	Triggers  []string `json:"triggers"`
	Recommend string   `json:"recommend"`
	Routes    []string `json:"routes,omitempty"`
	Allowance int      `json:"allowance,omitempty"` // 0: unknown
	Text      string   `json:"text"`
}

// Input is everything a diagnosis reads. FromDir fills it from an attempt's evidence folder.
type Input struct {
	Task         string
	Attempt      int
	Reason       string // the attempt record's reason
	ReportError  string // report.json "error" (the worker's own)
	Tokens       any    // report or record spend
	CostUSD      any
	ChangedPaths []string // report or record changed_paths
	AllowedPaths []string // globs, fnmatch semantics as G2
	Toolchain    string
	TaskList     string   // the task list the worker saw
	TaskSource   string   // where TaskList came from, e.g. "prompt.md"
	SlicePhases  []string // the slice's phase headings; empty: every phase
	Denials      string   // proxy-denials.log
	FailedChecks []string // grade.json checks that didn't pass
	G4Detail     string   // grade.json G4 detail (install or download trouble)
	TestSeconds  int      // one project test run, from the baseline grade; 0 unknown
	DeadlineS    int      // the worker deadline; 0 unknown
}

// Diagnose builds the diagnosis. Same input, same output.
func Diagnose(in Input) Diagnosis {
	d := Diagnosis{Version: 1, Task: in.Task, Attempt: in.Attempt, Ended: in.Reason,
		Usage: Usage{Tokens: in.Tokens, CostUSD: in.CostUSD}, Signals: []Signal{}, Triggers: []string{}}
	if in.Reason == "deadline" {
		d.DeadlineS = in.DeadlineS
	}

	// Research tasks: unchecked tasks of this slice whose wording is research.
	var research []string
	for _, t := range Tasks(in.TaskList) {
		if t.Checked || !InPhases(t, in.SlicePhases) {
			continue
		}
		if _, ok := ResearchTask(t.Desc); ok {
			research = append(research, t.ID)
			d.Signals = append(d.Signals, Signal{Kind: KindResearchTask, Trigger: UnknownWork,
				Detail: fmt.Sprintf("task %s reads as research: %q", t.ID, clip(t.Desc, 70)),
				Source: fmt.Sprintf("%s task list, line %d", or(in.TaskSource, "prompt.md"), t.Line)})
		}
	}

	// Files changed outside the allowed paths; manifests and lock files are an environment need.
	var manifests []string
	for _, p := range sorted(in.ChangedPaths) {
		if allowed(p, in.AllowedPaths) {
			continue
		}
		if images.IsManifest(in.Toolchain, p) {
			manifests = append(manifests, p)
			d.Signals = append(d.Signals, Signal{Kind: KindManifestOutside, Trigger: EnvironmentNeed,
				Detail: p + " changed, outside the allowed paths (dependency manifest or lock file)",
				Source: "report.json changed_paths"})
			continue
		}
		d.Signals = append(d.Signals, Signal{Kind: KindOutsidePaths,
			Detail: p + " changed, outside the allowed paths", Source: "report.json changed_paths"})
	}

	// Install or download trouble.
	install := in.ReportError == "clone_failed" || in.ReportError == "bundle_failed" ||
		networkError.MatchString(in.G4Detail)
	if install {
		what := "install or download failed"
		if in.ReportError != "" && in.ReportError != in.Reason {
			what += " (" + in.ReportError + ")"
		}
		d.Signals = append(d.Signals, Signal{Kind: KindInstallError, Trigger: EnvironmentNeed,
			Detail: what, Source: "report.json, grade.json G4"})
	}

	// Egress refusals other than the probes; an environment need only with install trouble or a
	// changed manifest (research R5).
	egressTrigger := ""
	if install || len(manifests) > 0 {
		egressTrigger = EnvironmentNeed
	}
	for _, r := range refusals(in.Denials) {
		d.Signals = append(d.Signals, Signal{Kind: KindEgressRefused, Trigger: egressTrigger,
			Detail: fmt.Sprintf("%s refused %d× by the %s proxy", r.host, r.n, r.proxy),
			Source: "proxy-denials.log"})
	}

	// A failure reason nobody has catalogued.
	for _, r := range []string{in.Reason, in.ReportError} {
		if r != "" && !Known(r) {
			d.Signals = append(d.Signals, Signal{Kind: KindUnknownReason, Trigger: UnknownWork,
				Detail: fmt.Sprintf("failure %q isn't a known cause (docs/troubleshooting.md)", r),
				Source: "attempt record, report.json"})
		}
	}

	if len(in.FailedChecks) > 0 {
		d.Signals = append(d.Signals, Signal{Kind: KindFailedChecks,
			Detail: "grading failed: " + strings.Join(in.FailedChecks, ", "), Source: "grade.json"})
	}

	if in.TestSeconds > 0 && in.DeadlineS > SoftMarginS {
		d.Allowance = (in.DeadlineS - SoftMarginS) / in.TestSeconds
		d.Signals = append(d.Signals, Signal{Kind: KindAllowance,
			Detail: fmt.Sprintf("%d full test runs fit in one attempt (one run %d s, deadline %d s)",
				d.Allowance, in.TestSeconds, in.DeadlineS),
			Source: "baseline grade"})
	}

	seen := map[string]bool{}
	for _, s := range d.Signals {
		if s.Trigger != "" && !seen[s.Trigger] {
			seen[s.Trigger] = true
			d.Triggers = append(d.Triggers, s.Trigger)
		}
	}
	sort.Strings(d.Triggers)

	switch {
	case len(d.Triggers) > 0:
		d.Recommend = BackToPlanning
		if seen[UnknownWork] {
			d.Routes = append(d.Routes, RoutePlanResearch)
		}
		if seen[EnvironmentNeed] {
			d.Routes = append(d.Routes, RoutePlanningPR)
		}
	case (in.Reason == "deadline" || in.Reason == "spend_cap") && d.Allowance > 0 && d.Allowance < AllowanceThreshold:
		d.Recommend = RaiseBudget
	default:
		d.Recommend = Retry
	}
	d.Text = render(d, research, manifests)
	return d
}

// networkError matches install or download failures in a test command's output.
var networkError = regexp.MustCompile(`(?i)ENOTFOUND|EAI_AGAIN|ECONNREFUSED|ETIMEDOUT|getaddrinfo|could not resolve host|network is unreachable|failed to fetch|unable to access 'http`)

type refusal struct {
	host, proxy string
	n           int
}

var refusedLine = regexp.MustCompile(`Proxying refused on filtered domain "([^"]+)"`)

// refusals groups proxy-denials.log by host and proxy, without the probes' own refusals:
// github.com through the model proxy (P06) and example.invalid through the target proxy (P06r).
func refusals(log string) []refusal {
	count := map[[2]string]int{}
	for _, line := range strings.Split(log, "\n") {
		m := refusedLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		proxy := "model"
		if strings.HasPrefix(strings.TrimSpace(line), "[") {
			proxy = "target"
		}
		host := m[1]
		if proxy == "model" && host == "github.com" || proxy == "target" && host == "example.invalid" {
			continue
		}
		count[[2]string{host, proxy}]++
	}
	var out []refusal
	for k, n := range count {
		out = append(out, refusal{host: k[0], proxy: k[1], n: n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].host != out[j].host {
			return out[i].host < out[j].host
		}
		return out[i].proxy < out[j].proxy
	})
	return out
}

// allowed matches a path against allowed-path globs with fnmatch semantics (as the grader's G2:
// "*" also matches "/").
func allowed(p string, globs []string) bool {
	for _, g := range globs {
		if globRegexp(g).MatchString(p) {
			return true
		}
	}
	return false
}

func globRegexp(g string) *regexp.Regexp {
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(g); i++ {
		switch c := g[i]; c {
		case '*':
			b.WriteString(".*")
		case '?':
			b.WriteString(".")
		case '[':
			j := strings.IndexByte(g[i:], ']')
			if j < 0 {
				b.WriteString(`\[`)
				continue
			}
			class := g[i+1 : i+j]
			if strings.HasPrefix(class, "!") {
				class = "^" + class[1:]
			}
			b.WriteString("[" + class + "]")
			i += j
		default:
			b.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	b.WriteString("$")
	return regexp.MustCompile(b.String())
}

func render(d Diagnosis, research, manifests []string) string {
	var b strings.Builder
	head := fmt.Sprintf("  diagnosis of attempt %d (%s): %s", d.Attempt, d.Task, d.Ended)
	if d.Ended == "deadline" && d.DeadlineS > 0 {
		head = fmt.Sprintf("  diagnosis of attempt %d (%s): stopped at the %d s deadline", d.Attempt, d.Task, d.DeadlineS)
	}
	if d.Usage.Tokens != nil {
		head += fmt.Sprintf(", %v tokens, $%v", d.Usage.Tokens, or(fmt.Sprint(d.Usage.CostUSD), "0"))
	}
	b.WriteString(head + "\n")
	if len(d.Signals) > 0 {
		b.WriteString("  signals:\n")
		for _, s := range d.Signals {
			fmt.Fprintf(&b, "    - %s (%s)\n", s.Detail, s.Source)
		}
	}
	points := "nothing outside the attempt"
	if len(d.Triggers) > 0 {
		var w []string
		for _, t := range d.Triggers {
			w = append(w, map[string]string{UnknownWork: "unknown work", EnvironmentNeed: "an environment need"}[t])
		}
		points = strings.Join(w, ", ")
	}
	fmt.Fprintf(&b, "  points to: %s\n", points)
	fmt.Fprintf(&b, "  recommended: %s\n", map[string]string{Retry: "retry", RaiseBudget: "raise the budget", BackToPlanning: "back to planning"}[d.Recommend])
	for _, r := range d.Routes {
		switch r {
		case RoutePlanResearch:
			if len(research) > 0 {
				fmt.Fprintf(&b, "    back to planning: resolve %s in the plan's research (/speckit-plan), then /speckit-tasks\n", strings.Join(research, ", "))
			} else {
				b.WriteString("    back to planning: find the cause before retrying (the plan's research, /speckit-plan), then /speckit-tasks\n")
			}
		case RoutePlanningPR:
			if len(manifests) > 0 {
				fmt.Fprintf(&b, "    back to planning: change %s in your planning PR, not in an attempt\n", strings.Join(manifests, ", "))
			} else {
				b.WriteString("    back to planning: give the project what it needs (hosts, dependencies, toolchain) in your planning PR or the target profile\n")
			}
		}
	}
	if d.Recommend == RaiseBudget {
		b.WriteString("    raise the budget: FORGE_WORKER_DEADLINE_S in the target profile, then a new run\n")
	}
	b.WriteString("  your choice: approve retry, or end the run with `replan` (you decide; forge does neither)\n")
	return b.String()
}

// FromDir reads an attempt's evidence folder (or a replay copy of one) into an Input. record is
// the attempt record from run.json; allowedPaths and toolchain fall back to meta.json.
func FromDir(dir string, task string, record map[string]any, allowedPaths []string, toolchain string, testSeconds, deadlineS int) (Input, error) {
	in := Input{Task: task, TestSeconds: testSeconds, DeadlineS: deadlineS, Toolchain: toolchain, AllowedPaths: allowedPaths}
	in.Attempt = toInt(record["n"])
	in.Reason, _ = record["reason"].(string)
	in.ChangedPaths = strs(record["changed_paths"])
	if sp, ok := record["spend"].(map[string]any); ok {
		in.Tokens, in.CostUSD = sp["tokens"], sp["cost_usd"]
	}
	var report map[string]any
	if readJSON(filepath.Join(dir, "report.json"), &report) == nil {
		in.ReportError, _ = report["error"].(string)
		if in.ChangedPaths == nil {
			in.ChangedPaths = strs(report["changed_paths"])
		}
		if sp, ok := report["spend"].(map[string]any); ok && in.Tokens == nil {
			in.Tokens, in.CostUSD = sp["tokens"], sp["cost_usd"]
		}
	}
	var meta map[string]any
	if readJSON(filepath.Join(dir, "meta.json"), &meta) == nil {
		if in.AllowedPaths == nil {
			in.AllowedPaths = strs(meta["allowed_paths"])
		}
		if in.Toolchain == "" {
			in.Toolchain, _ = meta["toolchain"].(string)
		}
		if sl, ok := meta["slice"].(map[string]any); ok {
			in.SlicePhases = strs(sl["phases"])
		}
	}
	if b, err := os.ReadFile(filepath.Join(dir, "prompt.md")); err == nil {
		in.TaskList, in.TaskSource = TaskListSection(string(b)), "prompt.md"
	} else if b, err := os.ReadFile(filepath.Join(dir, "tasklist.md")); err == nil {
		in.TaskList, in.TaskSource = string(b), "prompt.md"
	}
	if b, err := os.ReadFile(filepath.Join(dir, "proxy-denials.log")); err == nil {
		in.Denials = string(b)
	}
	var grade map[string]any
	if readJSON(filepath.Join(dir, "grade.json"), &grade) == nil {
		checks, _ := grade["checks"].([]any)
		for _, x := range checks {
			c, _ := x.(map[string]any)
			id, _ := c["id"].(string)
			if c["result"] != "pass" && id != "" {
				in.FailedChecks = append(in.FailedChecks, id)
			}
			if strings.HasPrefix(id, "G4") {
				in.G4Detail = fmt.Sprint(c["detail"])
			}
		}
	}
	return in, nil
}

// TaskListSection returns the task list a worker prompt carries: from "# Task list" to the next
// "# This run" heading or the end.
func TaskListSection(prompt string) string {
	var b strings.Builder
	on := false
	for _, line := range strings.Split(prompt, "\n") {
		if strings.HasPrefix(line, "# Task list") {
			on = true
			continue
		}
		if on && strings.HasPrefix(line, "# This run") {
			break
		}
		if on {
			b.WriteString(line + "\n")
		}
	}
	return b.String()
}

// Write stores the diagnosis as diagnosis.json in dir.
func Write(dir string, d Diagnosis) error {
	b, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "diagnosis.json"), append(b, '\n'), 0o644)
}

// Read loads dir/diagnosis.json; ok is false when there is none.
func Read(dir string) (Diagnosis, bool) {
	var d Diagnosis
	if readJSON(filepath.Join(dir, "diagnosis.json"), &d) != nil {
		return d, false
	}
	return d, true
}

func readJSON(p string, v any) error {
	b, err := os.ReadFile(p)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

func strs(v any) []string {
	xs, ok := v.([]any)
	if !ok {
		if s, ok := v.([]string); ok {
			return s
		}
		return nil
	}
	out := []string{}
	for _, x := range xs {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func toInt(v any) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	case int64:
		return int(x)
	case json.Number:
		n, _ := x.Int64()
		return int(n)
	}
	return 0
}

func sorted(xs []string) []string {
	out := append([]string(nil), xs...)
	sort.Strings(out)
	return out
}

func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

func or(s, def string) string {
	if s == "" || s == "<nil>" {
		return def
	}
	return s
}
