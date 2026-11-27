package ci

import (
	"errors"
	"os"
	"strings"
	"testing"
)

var documcpScripts = map[string]string{
	"test":      "jest",
	"test:ci":   "jest --coverage --ci --watchAll=false",
	"typecheck": "tsc --noEmit",
	"lint":      "eslint . --ext .ts",
	"build":     "tsc",
}

func documcpCI(t *testing.T) Workflow {
	t.Helper()
	raw, err := os.ReadFile("testdata/ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	w, err := ParseWorkflow(".github/workflows/ci.yml", raw)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

const releaseYML = `name: Release
on:
  push:
    tags: ["v*.*.*"]
  workflow_dispatch:
jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - run: npm run lint && npm test -- --forceExit
`

func TestParseWorkflow(t *testing.T) {
	w := documcpCI(t)
	if !w.Gates || w.Name != "CI" || len(w.Jobs) != 3 {
		t.Fatalf("%+v", w)
	}
	// jobs are sorted by id: commitlint, security, test
	test := w.Jobs[2]
	if test.Name != "Test Suite" || test.Matrix["node-version"][0] != "24.x" {
		t.Fatalf("%+v", test)
	}
	for _, s := range test.Steps {
		if strings.Contains(s.Run, "--json") {
			t.Errorf("a step that runs only after a failure was kept: %q", s.Run)
		}
	}
	rel, err := ParseWorkflow("release.yml", []byte(releaseYML))
	if err != nil || rel.Gates {
		t.Fatalf("a tag-only push is a release trigger, not a change check: %+v %v", rel, err)
	}
}

func TestResolveAndClassify(t *testing.T) {
	for in, want := range map[string][2]string{
		"npm test":                     {"jest", "test"},
		"npm test -- --coverage":       {"jest --coverage", "test"},
		"npm run test:ci":              {"jest --coverage --ci --watchAll=false", "test"},
		"npm run typecheck":            {"tsc --noEmit", "typecheck"},
		"npm run lint":                 {"eslint . --ext .ts", "lint"},
		"npm run build":                {"tsc", "build"},
		"npm ci":                       {"npm ci", "install"},
		"npm audit --audit-level=high": {"npm audit --audit-level=high", "audit"},
		"go test -race ./...":          {"go test -race ./...", "test"},
		"golangci-lint run":            {"golangci-lint run", "lint"},
		"npx commitlint --from x":      {"npx commitlint --from x", "commitlint"},
	} {
		r := Resolve(in, documcpScripts)
		if r != want[0] || Classify(r) != want[1] {
			t.Errorf("%q → %q (%q), want %q (%q)", in, r, Classify(r), want[0], want[1])
		}
	}
}

func TestSplitDropsShellNoise(t *testing.T) {
	got := Split("cd app && npm ci\n# comment\n\"jest\"\nFOO=1\necho hi; npm test || echo failed")
	if strings.Join(got, "|") != "npm ci|npm test" {
		t.Fatalf("%q", got)
	}
}

func find(fs []Finding, check string) []Finding {
	var out []Finding
	for _, f := range fs {
		if f.Check == check {
			out = append(out, f)
		}
	}
	return out
}

func TestReviewDocumcp(t *testing.T) {
	rel, _ := ParseWorkflow(".github/workflows/release.yml", []byte(releaseYML))
	in := Input{Branch: "main", Workflows: []Workflow{documcpCI(t), rel}, Scripts: documcpScripts,
		TestCommand: "npm ci --offline --ignore-scripts && npm run test:ci", Toolchain: "node24",
		Runs: []JobRun{
			{Workflow: "CI", Job: "Test Suite (24.x)", Conclusion: "success"},
			{Workflow: "CI", Job: "Security Audit", Conclusion: "failure", FailedStep: "Check for high severity vulnerabilities (production)",
				ErrorLine: "zod  <4.6.6 (severity: high)", When: "2026-11-27T20:00:00Z"},
		}}
	fs := Review(in)

	wf := find(fs, "workflows")[0]
	if !strings.Contains(wf.Detail, "1 workflow(s) check changes (ci.yml; 3 job(s))") || !strings.Contains(wf.Detail, "release.yml") {
		t.Errorf("workflows: %s", wf.Detail)
	}
	br := find(fs, "default branch")
	if len(br) != 1 || br[0].Status != Warn ||
		!strings.Contains(br[0].Detail, "CI › Security Audit failed on main (2026-11-27)") ||
		!strings.Contains(br[0].Detail, "`npm audit --audit-level=high --production`") ||
		!strings.Contains(br[0].Detail, "zod <4.6.6 (severity: high)") ||
		!strings.Contains(br[0].Detail, "dependency item") {
		t.Errorf("default branch: %+v", br)
	}
	tc := find(fs, "test command")[0]
	if tc.Status != Pass || !strings.Contains(tc.Detail, "`npm test -- --coverage` → `jest --coverage` (CI › Test Suite)") {
		t.Errorf("test command: %+v", tc)
	}
	only := find(fs, "checks grading doesn't run")
	if len(only) != 1 || only[0].Status != Warn {
		t.Fatalf("ci-only: %+v", only)
	}
	for _, want := range []string{"typecheck: `npm run typecheck` → `tsc --noEmit` (CI › Test Suite)",
		"lint: `npm run lint` → `eslint . --ext .ts`", "audit: `npm audit --audit-level=high --production` (CI › Security Audit)",
		"commitlint: `npx commitlint --from HEAD~1 --to HEAD --verbose`"} {
		if !strings.Contains(only[0].Detail, want) {
			t.Errorf("ci-only lacks %q: %s", want, only[0].Detail)
		}
	}
	if strings.Contains(only[0].Detail, "--forceExit") {
		t.Error("the release workflow (tags only) was compared")
	}
	rt := find(fs, "runtime")[0]
	if rt.Status != Pass || !strings.Contains(rt.Detail, "node 24.x") {
		t.Errorf("runtime: %+v", rt)
	}
}

func TestTestCommandMismatch(t *testing.T) {
	in := Input{Branch: "main", Workflows: []Workflow{documcpCI(t)}, Scripts: documcpScripts,
		TestCommand: "npx jest --ci --forceExit", Toolchain: "node22"}
	fs := Review(in)
	tc := find(fs, "test command")[0]
	if tc.Status != Warn || !strings.Contains(tc.Detail, "CI uses `--coverage`") ||
		!strings.Contains(tc.Detail, "grading can pass while CI fails") || !strings.Contains(tc.Detail, "forge uses `--forceExit`") {
		t.Errorf("%+v", tc)
	}
	rt := find(fs, "runtime")[0]
	if rt.Status != Warn || !strings.Contains(rt.Detail, "the toolchain node22 provides node 22") {
		t.Errorf("%+v", rt)
	}
}

func TestBranchHealthCases(t *testing.T) {
	w := documcpCI(t)
	fs := Review(Input{Branch: "main", Workflows: []Workflow{w}, RunsErr: errors.New("403 Resource not accessible")})
	if b := find(fs, "default branch")[0]; b.Status != Info || !strings.Contains(b.Detail, "403") {
		t.Errorf("%+v", b)
	}
	fs = Review(Input{Branch: "main", Workflows: []Workflow{w}, Runs: []JobRun{{Workflow: "CI", Job: "Test Suite", Conclusion: "success"}}})
	if b := find(fs, "default branch")[0]; b.Status != Pass {
		t.Errorf("%+v", b)
	}
	deploy, _ := ParseWorkflow("deploy.yml", []byte("name: Deploy\non: [push]\njobs:\n  deploy:\n    steps:\n      - name: Pull env\n        run: vercel pull --yes\n"))
	fs = Review(Input{Branch: "main", Workflows: []Workflow{deploy}, Runs: []JobRun{{Workflow: "Deploy", Job: "deploy", Conclusion: "failure", FailedStep: "Pull env", ErrorLine: "Error: token not valid."}}})
	if b := find(fs, "default branch")[0]; b.Status != Info || !strings.Contains(b.Detail, "Not a check grading relates to") || strings.Contains(b.Detail, "valid..") {
		t.Errorf("a deployment failure is information: %+v", b)
	}
}

func TestErrorLine(t *testing.T) {
	audit := `2026-11-27T20:01:02.0000000Z ##[group]Run npm audit --audit-level=high --production
2026-11-27T20:01:03.0000000Z # npm audit report
2026-11-27T20:01:03.0000000Z glob  9.0.0 - 14.0.1
2026-11-27T20:01:03.0000000Z Severity: moderate
2026-11-27T20:01:03.0000000Z zod  <4.6.6
2026-11-27T20:01:03.0000000Z Severity: high
2026-11-27T20:01:04.0000000Z ##[error]Process completed with exit code 1.`
	if got := ErrorLine(StepSection(audit, "")); got != "zod  <4.6.6 (severity: high)" {
		t.Errorf("audit: %q", got)
	}
	for log, want := range map[string]string{
		"x\nFAIL tests/a.test.ts\n##[error]Process completed with exit code 1.":                  "FAIL tests/a.test.ts",
		"src/a.ts(3,1): error TS2345: bad\n##[error]Process completed with exit code 2.":         "src/a.ts(3,1): error TS2345: bad",
		"building\nError: The token is not valid.\n##[error]Process completed with exit code 1.": "Error: The token is not valid.",
		"nothing here": "",
	} {
		if got := ErrorLine(log); got != want {
			t.Errorf("%q → %q, want %q", log, got, want)
		}
	}
}

func TestStepSectionTakesTheFailingGroup(t *testing.T) {
	log := "##[group]Run npm audit || echo ok\nSeverity: high\n##[endgroup]\n##[group]Run npm test\nFAIL x\n##[error]Process completed with exit code 1.\nPost job"
	if got := ErrorLine(StepSection(log, "")); got != "FAIL x" {
		t.Errorf("%q", got)
	}
}
