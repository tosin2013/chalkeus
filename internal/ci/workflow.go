// Package ci reviews a target's CI configuration and recent results (036 CI review): which
// commands CI runs, how they compare with the profile's test command, which checks grading never
// runs, and which runtime versions CI uses. The analysis is pure (workflow files and package
// scripts in, findings out); a Reader fetches the inputs from the target's Git host.
package ci

import (
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"

	"sigs.k8s.io/yaml"
)

// Workflow is one CI workflow file, reduced to what the review reads.
type Workflow struct {
	File string
	Name string
	// Gates is true when the workflow runs on pushes or pull requests (it checks changes);
	// release, schedule-only and manual workflows are listed but not compared.
	Gates bool
	Jobs  []Job
}

// Job is one workflow job.
type Job struct {
	ID, Name string
	Steps    []Step
	Matrix   map[string][]string // matrix values by key, as strings
}

// Step is one job step that gates the job: a `run` script or a `uses` action. Steps that run
// only after a failure, or that may fail without failing the job, are dropped.
type Step struct {
	Name string
	Run  string            // the script, when the step runs commands
	Uses string            // the action, when the step uses one (without its @ref)
	With map[string]string // the action's inputs
}

type rawWorkflow struct {
	Name string            `json:"name"`
	On   any               `json:"on"`
	Jobs map[string]rawJob `json:"jobs"`
}

type rawJob struct {
	Name     string         `json:"name"`
	Strategy map[string]any `json:"strategy"`
	Steps    []rawStep      `json:"steps"`
}

type rawStep struct {
	Name            string         `json:"name"`
	Run             string         `json:"run"`
	Uses            string         `json:"uses"`
	With            map[string]any `json:"with"`
	If              any            `json:"if"`
	ContinueOnError any            `json:"continue-on-error"`
}

// ParseWorkflow reads one workflow file. YAML's `on` key may parse as the boolean true, so both
// spellings are read.
func ParseWorkflow(file string, raw []byte) (Workflow, error) {
	js, err := yaml.YAMLToJSON(raw)
	if err != nil {
		return Workflow{}, fmt.Errorf("%s: %w", file, err)
	}
	var w rawWorkflow
	if err := json.Unmarshal(js, &w); err != nil {
		return Workflow{}, fmt.Errorf("%s: %w", file, err)
	}
	var all map[string]any
	_ = json.Unmarshal(js, &all)
	on := w.On
	if on == nil {
		on = all["true"] // `on:` read as a YAML 1.1 boolean
	}
	out := Workflow{File: file, Name: w.Name, Gates: gates(on)}
	if out.Name == "" {
		out.Name = path.Base(file)
	}
	ids := make([]string, 0, len(w.Jobs))
	for id := range w.Jobs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		j := w.Jobs[id]
		job := Job{ID: id, Name: j.Name, Matrix: matrix(j.Strategy)}
		if job.Name == "" {
			job.Name = id
		}
		for _, s := range j.Steps {
			if conditional(s.If) || truthy(s.ContinueOnError) {
				continue
			}
			st := Step{Name: s.Name, Run: strings.TrimSpace(s.Run), With: map[string]string{}}
			if s.Uses != "" {
				st.Uses, _, _ = strings.Cut(s.Uses, "@")
			}
			for k, v := range s.With {
				st.With[k] = fmt.Sprint(v)
			}
			if st.Name == "" {
				st.Name = firstNonEmpty(firstLine(st.Run), st.Uses)
			}
			job.Steps = append(job.Steps, st)
		}
		out.Jobs = append(out.Jobs, job)
	}
	return out, nil
}

func gates(on any) bool {
	has := func(k string) bool {
		return k == "push" || k == "pull_request" || k == "pull_request_target" || k == "merge_group"
	}
	switch v := on.(type) {
	case string:
		return has(v)
	case []any:
		for _, x := range v {
			if s, _ := x.(string); has(s) {
				return true
			}
		}
	case map[string]any:
		for k, cfg := range v {
			if !has(k) {
				continue
			}
			// `push: {tags: [...]}` with no branches is a release trigger, not a change check.
			if c, _ := cfg.(map[string]any); k == "push" && c != nil && c["tags"] != nil && c["branches"] == nil && c["paths"] == nil {
				continue
			}
			return true
		}
	}
	return false
}

// conditional is true for steps that only run after something failed (or always, as cleanup).
func conditional(v any) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	return strings.Contains(s, "failure()") || strings.Contains(s, "cancelled()")
}

func truthy(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case string:
		return x == "true"
	}
	return false
}

func matrix(strategy map[string]any) map[string][]string {
	m, _ := strategy["matrix"].(map[string]any)
	out := map[string][]string{}
	for k, v := range m {
		if vs, ok := v.([]any); ok {
			for _, x := range vs {
				if _, isMap := x.(map[string]any); !isMap {
					out[k] = append(out[k], fmt.Sprint(x))
				}
			}
		}
	}
	return out
}

func firstLine(s string) string {
	l, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return strings.TrimSpace(l)
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

// Command is one shell command from a step (or from the profile's test command), with what it
// resolves to through the project's package scripts.
type Command struct {
	Text     string // as written, e.g. `npm run lint`
	Resolved string // after package scripts, e.g. `eslint src/**/*.ts`; equal to Text otherwise
	Kind     string // test, typecheck, lint, audit, build, commitlint, install, or ""
	Where    string // "workflow › job" for CI commands, "profile" for forge's
}

var (
	segSplit    = regexp.MustCompile(`\s*(?:&&|\|\||;)\s*`)
	commandWord = regexp.MustCompile(`^[A-Za-z.~/][A-Za-z0-9._/~+-]*$`)
)

// Split cuts a script into commands: one per line, and per `&&`, `||` or `;`. Comments, `cd`,
// `echo` and environment exports are dropped.
func Split(script string) []string {
	var out []string
	for _, line := range strings.Split(strings.ReplaceAll(script, "\\\n", " "), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		for _, seg := range segSplit.Split(line, -1) {
			seg = strings.TrimSpace(seg)
			f := strings.Fields(seg)
			if len(f) == 0 || !commandWord.MatchString(f[0]) {
				continue // a quoted string, a variable assignment or shell syntax, not a command
			}
			switch f[0] {
			case "cd", "echo", "export", "set", "rm", "mkdir", "cp", "mv", "true", "false", "exit", "if", "then", "fi", "else":
				continue
			}
			out = append(out, seg)
		}
	}
	return out
}

var (
	npmRun    = regexp.MustCompile(`^(?:npm|pnpm|yarn)\s+(?:run(?:-script)?\s+)?([A-Za-z0-9:_.-]+)(.*)$`)
	npmBuilt  = map[string]bool{"ci": true, "install": true, "i": true, "audit": true, "exec": true, "x": true, "publish": true, "pack": true, "ls": true, "add": true}
	testShort = map[string]bool{"test": true, "t": true}
)

// Resolve follows `npm run <script>` (and `npm test`, pnpm, yarn) through the package scripts,
// a few levels deep, keeping extra arguments (`npm test -- --coverage`).
func Resolve(cmd string, scripts map[string]string) string {
	for i := 0; i < 4; i++ {
		m := npmRun.FindStringSubmatch(strings.TrimSpace(cmd))
		if m == nil || npmBuilt[m[1]] {
			return cmd
		}
		name, rest := m[1], strings.TrimSpace(m[2])
		if testShort[name] {
			name = "test"
		}
		body, ok := scripts[name]
		if !ok {
			return cmd
		}
		rest = strings.TrimSpace(strings.TrimPrefix(rest, "--"))
		if rest != "" {
			body += " " + rest
		}
		if len(Split(body)) > 1 { // a script of several commands: keep it whole
			return body
		}
		cmd = body
	}
	return cmd
}

var kinds = []struct {
	kind string
	re   *regexp.Regexp
}{
	{"install", regexp.MustCompile(`^(?:npm (?:ci|install|i)\b|pnpm install|yarn(?: install)?$|pip install|go mod download|bundle install|poetry install)`)},
	{"audit", regexp.MustCompile(`\b(?:npm audit|pnpm audit|yarn audit|govulncheck|pip-audit|osv-scanner|safety check|bundle audit|snyk test)\b`)},
	{"commitlint", regexp.MustCompile(`\bcommitlint\b`)},
	{"typecheck", regexp.MustCompile(`\btsc\b[^|]*--noEmit|\bmypy\b|\bpyright\b|\bgo vet\b|\bvue-tsc\b`)},
	{"lint", regexp.MustCompile(`\b(?:eslint|golangci-lint|ruff|flake8|pylint|prettier\s+(?:--check|-c)|stylelint|markdownlint|rubocop|staticcheck|biome (?:check|lint))\b`)},
	{"test", regexp.MustCompile(`\b(?:jest|vitest|mocha|ava|tap|go test|pytest|tox|mvn\b.*\b(?:test|verify)|gradle\b.*\btest|cargo test|rspec)\b`)},
	{"build", regexp.MustCompile(`\b(?:tsc\b|go build|mvn\b.*\bpackage|gradle\b.*\bbuild|cargo build|webpack|vite build|rollup|esbuild|next build)`)},
}

// Classify names what a resolved command checks; "" when it is none of the known kinds.
func Classify(resolved string) string {
	for _, k := range kinds {
		if k.re.MatchString(resolved) {
			return k.kind
		}
	}
	return ""
}

var actionKinds = map[string]string{
	"github/codeql-action/analyze":        "code scanning (CodeQL)",
	"actions/dependency-review-action":    "dependency review",
	"golangci/golangci-lint-action":       "lint",
	"wagoid/commitlint-github-action":     "commitlint",
	"returntocorp/semgrep-action":         "code scanning (Semgrep)",
	"aquasecurity/trivy-action":           "audit",
	"snyk/actions/node":                   "audit",
	"pypa/gh-action-pip-audit":            "audit",
	"golang/govulncheck-action":           "audit",
	"super-linter/super-linter":           "lint",
	"github/super-linter":                 "lint",
	"DavidAnson/markdownlint-cli2-action": "lint",
	"codecov/codecov-action":              "", // uploads, checks nothing
	"actions/upload-artifact":             "",
	"actions/checkout":                    "",
	"actions/cache":                       "",
}

// Commands lists the gating workflows' commands, resolved and classified.
func Commands(wfs []Workflow, scripts map[string]string) []Command {
	var out []Command
	for _, w := range wfs {
		if !w.Gates {
			continue
		}
		for _, j := range w.Jobs {
			where := w.Name + " › " + j.Name
			for _, s := range j.Steps {
				if s.Uses != "" {
					if k := actionKinds[s.Uses]; k != "" {
						out = append(out, Command{Text: "uses: " + s.Uses, Resolved: "uses: " + s.Uses, Kind: k, Where: where})
					}
					continue
				}
				for _, c := range Split(s.Run) {
					r := Resolve(c, scripts)
					out = append(out, Command{Text: c, Resolved: r, Kind: Classify(r), Where: where})
				}
			}
		}
	}
	return out
}

// ProfileCommands is the profile's test command, resolved and classified the same way.
func ProfileCommands(testCommand string, scripts map[string]string) []Command {
	var out []Command
	for _, c := range Split(testCommand) {
		r := Resolve(c, scripts)
		k := Classify(r)
		if k == "" && npmRun.MatchString(c) {
			k = Classify(c)
		}
		out = append(out, Command{Text: c, Resolved: r, Kind: k, Where: "profile"})
	}
	return out
}

// Flag is a test-runner option that changes what a test run proves.
type Flag struct{ Name, Why string }

var flags = []struct {
	name string
	re   *regexp.Regexp
	why  string
}{
	{"coverage", regexp.MustCompile(`--coverage\b|--cov(?:=|\b)|-cover(?:profile)?\b|--cov-fail-under`), "checks coverage (and any thresholds the project sets)"},
	{"--forceExit", regexp.MustCompile(`--forceExit\b`), "ends the run even when tests leave handles open, hiding those leaks"},
	{"--runInBand", regexp.MustCompile(`--runInBand\b|(?:^|\s)-i(?:\s|$)`), "runs tests in one process"},
	{"--maxWorkers", regexp.MustCompile(`--maxWorkers[= ]\S+|(?:^|\s)-w\s*\d+`), "sets the number of parallel workers"},
	{"-race", regexp.MustCompile(`(?:^|\s)-race\b`), "runs Go's race detector"},
	{"--detectOpenHandles", regexp.MustCompile(`--detectOpenHandles\b`), "reports leaked handles"},
}

// Flags lists the notable flags in a resolved test command.
func Flags(resolved string) map[string]string {
	out := map[string]string{}
	for _, f := range flags {
		if m := f.re.FindString(resolved); m != "" {
			out[f.name] = strings.TrimSpace(m)
		}
	}
	return out
}

// FlagWhy explains a flag for the report.
func FlagWhy(name string) string {
	for _, f := range flags {
		if f.name == name {
			return f.why
		}
	}
	return ""
}

// Scripts reads package.json's scripts; empty when absent or unreadable.
func Scripts(packageJSON []byte) map[string]string {
	var p struct {
		Scripts map[string]string `json:"scripts"`
	}
	if json.Unmarshal(packageJSON, &p) != nil || p.Scripts == nil {
		return map[string]string{}
	}
	return p.Scripts
}
