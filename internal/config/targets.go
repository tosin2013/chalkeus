package config

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Target profiles (014, specs/014-any-project/contracts/target-profile.md): one file per target
// repository beside pilot.env. The target repository itself declares nothing (owner decision
// 2026-11-17); cluster mode (011) holds the same fields in a cluster resource.

// FixtureTarget is the implicit target built from pilot.env alone.
const FixtureTarget = "fixture"

var (
	targetNameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,30}$`)
	hostRE       = regexp.MustCompile(`^[A-Za-z0-9.-]+$`)
	quantityRE   = regexp.MustCompile(`^[0-9]+(Mi|Gi)$`)
	digestRefRE  = regexp.MustCompile(`^[a-z0-9./:_-]+@sha256:[0-9a-f]{64}$`)
)

// Target is one configured target repository.
type Target struct {
	Name            string
	Repo            string
	Branch          string
	Toolchain       string
	ToolchainImage  string // operator-supplied toolchain image (digest-pinned), "" for the catalog's
	TestCommand     string
	InstallCommand  string   // installs the project\'s dependencies (prefetch and worker; 014 R7)
	ChangeType      string   // Conventional Commits type of the item's PR title (default feat)
	TestPaths       []string // the project's test files (globs): attempts must add tests and keep existing ones (G6, G7)
	RegistryHosts   []string
	CompanionRepo   string
	CompanionPath   string // folder inside the companion that holds items/ ("" = the root); one repo can serve many targets
	GithubIssues    string // on|off
	GithubApprovals string // on|off
	GraderMemory    string // optional override, e.g. 3Gi
	GraderDeadlineS int    // optional override, 0 = catalog default
	CacheSize       string // optional override
	MaxInputMB      int
	PublisherEnv    string // token mode: targets/<name>.publisher.env
	Fixture         bool
}

var companionPathRE = regexp.MustCompile(`^[A-Za-z0-9._-]+(/[A-Za-z0-9._-]+)*$`)

func companionPathOK(p string) bool {
	if !companionPathRE.MatchString(p) {
		return false
	}
	for _, part := range strings.Split(p, "/") {
		if part == "." || part == ".." {
			return false
		}
	}
	return true
}

// CompanionItemDir is where an item's hidden tests live inside the companion checkout
// (slash-separated): [<PILOT_COMPANION_PATH>/]items/<item>.
func (t Target) CompanionItemDir(item string) string {
	if t.CompanionPath == "" {
		return "items/" + item
	}
	return t.CompanionPath + "/items/" + item
}

var changeTypes = map[string]bool{"feat": true, "fix": true, "refactor": true, "perf": true, "test": true,
	"docs": true, "build": true, "ci": true, "style": true, "chore": true}

// TargetsDir is ConfigDir/targets.
func (c *Config) TargetsDir() string { return filepath.Join(c.ConfigDir, "targets") }

// Fixture is the implicit fixture target (today's single-repository pilot.env).
func (c *Config) Fixture() Target {
	return Target{Name: FixtureTarget, Repo: c.Repo(), Branch: c.Branch(), Toolchain: "python",
		GithubIssues: c.Values["PILOT_GITHUB_ISSUES"], GithubApprovals: c.Values["PILOT_GITHUB_APPROVALS"],
		MaxInputMB: 50, PublisherEnv: c.PublisherEnv, Fixture: true}
}

func targetErr(code, format string, a ...any) *Error {
	e := errf(2, format, a...)
	e.Msg = code + ": " + e.Msg
	return e
}

// ListTargets returns the configured target names, sorted ("fixture" is implicit and not listed).
func (c *Config) ListTargets() []string {
	m, _ := filepath.Glob(filepath.Join(c.TargetsDir(), "*.env"))
	var out []string
	for _, p := range m {
		n := strings.TrimSuffix(filepath.Base(p), ".env")
		if strings.HasSuffix(n, ".publisher") || !targetNameRE.MatchString(n) || n == FixtureTarget {
			continue
		}
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// LoadTarget reads and validates targets/<name>.env. "" or "fixture" is the fixture target.
// origin is this checkout's origin (the canary); it may be "".
func (c *Config) LoadTarget(name, origin string) (Target, error) {
	if name == "" || name == FixtureTarget {
		return c.Fixture(), nil
	}
	if !targetNameRE.MatchString(name) {
		return Target{}, targetErr("TARGET_INVALID", "target name %q must match %s", name, targetNameRE)
	}
	dir := c.TargetsDir()
	p := filepath.Join(dir, name+".env")
	if _, err := os.Stat(p); err != nil {
		return Target{}, targetErr("TARGET_UNKNOWN", "no target %q: create %s (see docs/targets.md)", name, p)
	}
	if st, err := os.Stat(dir); err == nil && st.Mode().Perm()&0o077 != 0 {
		return Target{}, targetErr("TARGET_MODE", "%s must be mode 0700 (is 0o%o)", dir, st.Mode().Perm())
	}
	if err := CheckMode(p); err != nil {
		return Target{}, targetErr("TARGET_MODE", "%s", err.Error())
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		return Target{}, err
	}
	t, err := parseTarget(name, ParseEnv(string(raw)), c)
	if err != nil {
		return Target{}, err
	}
	t.PublisherEnv = filepath.Join(dir, name+".publisher.env")
	if err := c.checkCanary(t, origin); err != nil {
		return Target{}, err
	}
	return t, nil
}

func parseTarget(name string, v map[string]string, c *Config) (Target, error) {
	invalid := func(key, why string) error {
		return targetErr("TARGET_INVALID", "targets/%s.env: %s %s", name, key, why)
	}
	t := Target{Name: name, Repo: v["PILOT_TARGET_REPO"], Branch: v["PILOT_TARGET_BRANCH"], Toolchain: v["PILOT_TOOLCHAIN"],
		ToolchainImage: v["PILOT_TOOLCHAIN_IMAGE"],
		TestCommand:    strings.TrimSpace(v["PILOT_TEST_COMMAND"]), CompanionRepo: v["PILOT_COMPANION_REPO"],
		CompanionPath:  strings.Trim(strings.TrimSpace(v["PILOT_COMPANION_PATH"]), "/"),
		InstallCommand: strings.TrimSpace(v["PILOT_INSTALL_COMMAND"]), ChangeType: v["PILOT_CHANGE_TYPE"],
		GithubIssues: v["PILOT_GITHUB_ISSUES"], GithubApprovals: v["PILOT_GITHUB_APPROVALS"],
		GraderMemory: v["PILOT_GRADER_MEMORY"], CacheSize: v["PILOT_CACHE_SIZE"], MaxInputMB: 50}
	if !repoRE.MatchString(t.Repo) {
		return Target{}, invalid("PILOT_TARGET_REPO", "must be owner/name")
	}
	if t.Branch == "" {
		t.Branch = "main"
	}
	if t.ToolchainImage != "" && !digestRefRE.MatchString(t.ToolchainImage) {
		return Target{}, invalid("PILOT_TOOLCHAIN_IMAGE", "must be pinned by digest: registry/name@sha256:<64 hex>")
	}
	if t.Toolchain == "" && t.ToolchainImage != "" {
		t.Toolchain = "operator-supplied"
	}
	if t.Toolchain == "" {
		return Target{}, invalid("PILOT_TOOLCHAIN", "is required (a name from pilot/images/toolchains.json, or set PILOT_TOOLCHAIN_IMAGE to your own image)")
	}
	if t.TestCommand == "" {
		return Target{}, invalid("PILOT_TEST_COMMAND", "is required")
	}
	if !repoRE.MatchString(t.CompanionRepo) {
		return Target{}, invalid("PILOT_COMPANION_REPO", "must be owner/name")
	}
	if t.CompanionPath != "" && !companionPathOK(t.CompanionPath) {
		return Target{}, invalid("PILOT_COMPANION_PATH", "must be a relative folder of letters, digits, '.', '_', '-' and '/', without '..'")
	}
	if strings.EqualFold(t.CompanionRepo, t.Repo) {
		return Target{}, invalid("PILOT_COMPANION_REPO", "must differ from PILOT_TARGET_REPO")
	}
	for _, h := range splitList(v["PILOT_REGISTRY_HOSTS"], false) {
		if !hostRE.MatchString(h) {
			return Target{}, invalid("PILOT_REGISTRY_HOSTS", "entries must be hostnames, got "+h)
		}
		t.RegistryHosts = append(t.RegistryHosts, h)
	}
	if t.InstallCommand != "" && len(t.RegistryHosts) == 0 {
		return Target{}, invalid("PILOT_INSTALL_COMMAND", "needs PILOT_REGISTRY_HOSTS: dependencies are downloaded from the registries you list")
	}
	for _, g := range splitList(v["PILOT_TEST_PATHS"], false) {
		if strings.HasPrefix(g, "/") || strings.Contains(g, "..") {
			return Target{}, invalid("PILOT_TEST_PATHS", "entries must be relative globs inside the repository, got "+g)
		}
		t.TestPaths = append(t.TestPaths, g)
	}
	if t.ChangeType == "" {
		t.ChangeType = "feat"
	}
	if !changeTypes[t.ChangeType] {
		return Target{}, invalid("PILOT_CHANGE_TYPE", "must be a Conventional Commits type: feat, fix, refactor, perf, test, docs, build, ci, style or chore")
	}
	if t.GithubIssues == "" {
		t.GithubIssues = "off" // a new target opts in (014 R14)
	}
	if t.GithubApprovals == "" && c != nil {
		t.GithubApprovals = c.Values["PILOT_GITHUB_APPROVALS"]
	}
	if t.GithubApprovals == "" {
		t.GithubApprovals = "off"
	}
	for k, x := range map[string]string{"PILOT_GITHUB_ISSUES": t.GithubIssues, "PILOT_GITHUB_APPROVALS": t.GithubApprovals} {
		if x != "on" && x != "off" {
			return Target{}, invalid(k, "must be on or off")
		}
	}
	for k, x := range map[string]string{"PILOT_GRADER_MEMORY": t.GraderMemory, "PILOT_CACHE_SIZE": t.CacheSize} {
		if x != "" && !quantityRE.MatchString(x) {
			return Target{}, invalid(k, "must be a quantity like 2Gi or 512Mi")
		}
	}
	for k, dst := range map[string]*int{"PILOT_GRADER_DEADLINE_S": &t.GraderDeadlineS, "PILOT_MAX_INPUT_MB": &t.MaxInputMB} {
		if x := v[k]; x != "" {
			n, err := strconv.Atoi(x)
			if err != nil || n <= 0 {
				return Target{}, invalid(k, "must be a positive integer")
			}
			*dst = n
		}
	}
	return t, nil
}

// checkCanary refuses a target that is this checkout's origin or another target's companion.
func (c *Config) checkCanary(t Target, origin string) error {
	if origin != "" && (strings.EqualFold(t.Repo, origin) || strings.EqualFold(t.CompanionRepo, origin)) {
		return targetErr("TARGET_IS_CANARY", "%s is this repository's origin (the write probe's canary)", origin)
	}
	for _, other := range c.ListTargets() {
		if other == t.Name {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(c.TargetsDir(), other+".env"))
		if err != nil {
			continue
		}
		v := ParseEnv(string(raw))
		if strings.EqualFold(v["PILOT_COMPANION_REPO"], t.Repo) {
			return targetErr("TARGET_IS_CANARY", "%s is target %q's companion repository; hidden tests must never be a build target", t.Repo, other)
		}
	}
	return nil
}
