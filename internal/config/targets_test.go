package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func targetCfg(t *testing.T) *Config {
	t.Helper()
	d := t.TempDir()
	v, err := ValidateValues(map[string]string{"PILOT_FIXTURE_REPO": "o/x-pilot-fixture", "PILOT_MODEL": "p/m",
		"PILOT_MODEL_EGRESS_HOSTS": "h", "PILOT_GITHUB_APPROVALS": "on", "PILOT_APPROVERS": "alice"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(d, "targets"), 0o700); err != nil {
		t.Fatal(err)
	}
	return &Config{Values: v, ConfigDir: d, PublisherEnv: filepath.Join(d, "publisher.env")}
}

const goodTarget = `PILOT_TARGET_REPO=me/greeting-service
PILOT_TOOLCHAIN=jdk21-maven
PILOT_TEST_COMMAND=mvn -B test
PILOT_REGISTRY_HOSTS=repo.maven.apache.org, repo1.example.org
PILOT_COMPANION_REPO=me/greeting-service-acceptance
`

func writeTarget(t *testing.T, c *Config, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(c.TargetsDir(), name+".env"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func wantCode(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil || !strings.HasPrefix(err.Error(), code+":") {
		t.Fatalf("want %s, got %v", code, err)
	}
	if e, ok := err.(*Error); !ok || e.Code != 2 {
		t.Fatalf("want exit 2, got %#v", err)
	}
}

func TestLoadTargetGood(t *testing.T) {
	c := targetCfg(t)
	writeTarget(t, c, "greeting", goodTarget)
	got, err := c.LoadTarget("greeting", "tosin2013/specfoundry")
	if err != nil {
		t.Fatal(err)
	}
	want := Target{Name: "greeting", Repo: "me/greeting-service", Branch: "main", Toolchain: "jdk21-maven",
		TestCommand: "mvn -B test", RegistryHosts: []string{"repo.maven.apache.org", "repo1.example.org"},
		CompanionRepo: "me/greeting-service-acceptance", GithubIssues: "off", GithubApprovals: "on", MaxInputMB: 50, ChangeType: "feat",
		PublisherEnv: filepath.Join(c.TargetsDir(), "greeting.publisher.env"), SecurityLevel: LevelMinimal,
		AuthorEnv: filepath.Join(c.TargetsDir(), "greeting.author.env"), AuthorAuto: "off", AuthorIntervalS: 120, AuthorAttempts: 2}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("\n got %+v\nwant %+v", got, want)
	}
	if names := c.ListTargets(); !reflect.DeepEqual(names, []string{"greeting"}) {
		t.Fatal(names)
	}
}

func TestFixtureTargetIsTodaysConfig(t *testing.T) {
	c := targetCfg(t)
	for _, name := range []string{"", FixtureTarget} {
		got, err := c.LoadTarget(name, "")
		if err != nil {
			t.Fatal(err)
		}
		if !got.Fixture || got.Repo != c.Repo() || got.Branch != c.Branch() || got.Toolchain != "python" || got.PublisherEnv != c.PublisherEnv {
			t.Fatalf("%+v", got)
		}
	}
}

func TestLoadTargetRefusals(t *testing.T) {
	c := targetCfg(t)
	_, err := c.LoadTarget("nope", "")
	wantCode(t, err, "TARGET_UNKNOWN")
	_, err = c.LoadTarget("Bad_Name", "")
	wantCode(t, err, "TARGET_INVALID")

	for key, body := range map[string]string{
		"PILOT_TARGET_REPO":          strings.Replace(goodTarget, "me/greeting-service\n", "not-a-repo\n", 1),
		"PILOT_TOOLCHAIN":            strings.Replace(goodTarget, "PILOT_TOOLCHAIN=jdk21-maven\n", "", 1),
		"PILOT_TEST_COMMAND":         strings.Replace(goodTarget, "mvn -B test", "  ", 1),
		"PILOT_COMPANION_REPO":       strings.Replace(goodTarget, "me/greeting-service-acceptance", "me/greeting-service", 1),
		"PILOT_REGISTRY_HOSTS":       strings.Replace(goodTarget, "repo1.example.org", "https://x.org/y", 1),
		"PILOT_GITHUB_ISSUES":        goodTarget + "PILOT_GITHUB_ISSUES=yes\n",
		"PILOT_GRADER_MEMORY":        goodTarget + "PILOT_GRADER_MEMORY=lots\n",
		"PILOT_GRADER_DEADLINE_S":    goodTarget + "PILOT_GRADER_DEADLINE_S=-1\n",
		"PILOT_MAX_INPUT_MB":         goodTarget + "PILOT_MAX_INPUT_MB=x\n",
		"PILOT_AUTHOR_AUTO":          goodTarget + "PILOT_AUTHOR_AUTO=yes\n",
		"PILOT_AUTHOR_INTERVAL_S":    goodTarget + "PILOT_AUTHOR_INTERVAL_S=5\n",
		"PILOT_AUTHOR_AUTO_ATTEMPTS": goodTarget + "PILOT_AUTHOR_AUTO_ATTEMPTS=4\n",
	} {
		writeTarget(t, c, "t", body)
		_, err := c.LoadTarget("t", "")
		wantCode(t, err, "TARGET_INVALID")
		if !strings.Contains(err.Error(), key) {
			t.Errorf("%s: message doesn't name the key: %v", key, err)
		}
	}
}

func TestLoadTargetModes(t *testing.T) {
	c := targetCfg(t)
	writeTarget(t, c, "greeting", goodTarget)
	_ = os.Chmod(filepath.Join(c.TargetsDir(), "greeting.env"), 0o644)
	_, err := c.LoadTarget("greeting", "")
	wantCode(t, err, "TARGET_MODE")
	_ = os.Chmod(filepath.Join(c.TargetsDir(), "greeting.env"), 0o600)
	_ = os.Chmod(c.TargetsDir(), 0o755)
	_, err = c.LoadTarget("greeting", "")
	wantCode(t, err, "TARGET_MODE")
}

func TestLoadTargetCanary(t *testing.T) {
	c := targetCfg(t)
	writeTarget(t, c, "greeting", goodTarget)
	_, err := c.LoadTarget("greeting", "me/greeting-service")
	wantCode(t, err, "TARGET_IS_CANARY")
	// another target's companion repository can never be built
	writeTarget(t, c, "evil", strings.Replace(strings.Replace(goodTarget, "me/greeting-service-acceptance", "me/other-acceptance", 1),
		"PILOT_TARGET_REPO=me/greeting-service", "PILOT_TARGET_REPO=me/greeting-service-acceptance", 1))
	_, err = c.LoadTarget("evil", "")
	wantCode(t, err, "TARGET_IS_CANARY")
}

func TestOverridesAndDefaults(t *testing.T) {
	c := targetCfg(t)
	c.Values["PILOT_GITHUB_APPROVALS"] = "off"
	writeTarget(t, c, "g", goodTarget+"PILOT_TARGET_BRANCH=dev\nPILOT_GRADER_MEMORY=3Gi\nPILOT_GRADER_DEADLINE_S=1200\nPILOT_CACHE_SIZE=4Gi\nPILOT_MAX_INPUT_MB=80\n")
	got, err := c.LoadTarget("g", "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Branch != "dev" || got.GraderMemory != "3Gi" || got.GraderDeadlineS != 1200 || got.CacheSize != "4Gi" ||
		got.MaxInputMB != 80 || got.GithubIssues != "off" || got.GithubApprovals != "off" {
		t.Fatalf("%+v", got)
	}
}

func TestOwnToolchainImageKey(t *testing.T) {
	c := targetCfg(t)
	own := "quay.io/acme/toolchain-rust@sha256:" + strings.Repeat("c", 64)
	writeTarget(t, c, "r", strings.Replace(goodTarget, "PILOT_TOOLCHAIN=jdk21-maven\n", "PILOT_TOOLCHAIN_IMAGE="+own+"\n", 1))
	got, err := c.LoadTarget("r", "")
	if err != nil || got.ToolchainImage != own || got.Toolchain != "operator-supplied" {
		t.Fatalf("%+v %v", got, err)
	}
	writeTarget(t, c, "r", strings.Replace(goodTarget, "PILOT_TOOLCHAIN=jdk21-maven\n", "PILOT_TOOLCHAIN_IMAGE=quay.io/acme/rust:latest\n", 1))
	_, err = c.LoadTarget("r", "")
	wantCode(t, err, "TARGET_INVALID")
}

func TestInstallCommandNeedsRegistries(t *testing.T) {
	c := targetCfg(t)
	writeTarget(t, c, "p", goodTarget+"PILOT_INSTALL_COMMAND=pip install -r requirements.txt\n")
	got, err := c.LoadTarget("p", "")
	if err != nil || got.InstallCommand != "pip install -r requirements.txt" {
		t.Fatalf("%+v %v", got, err)
	}
	writeTarget(t, c, "p", strings.Replace(goodTarget, "PILOT_REGISTRY_HOSTS=repo.maven.apache.org, repo1.example.org\n", "", 1)+"PILOT_INSTALL_COMMAND=pip install x\n")
	_, err = c.LoadTarget("p", "")
	wantCode(t, err, "TARGET_INVALID")
}

func TestChangeType(t *testing.T) {
	c := targetCfg(t)
	writeTarget(t, c, "p", goodTarget+"PILOT_CHANGE_TYPE=fix\n")
	if got, err := c.LoadTarget("p", ""); err != nil || got.ChangeType != "fix" {
		t.Fatalf("%+v %v", got, err)
	}
	writeTarget(t, c, "p", goodTarget+"PILOT_CHANGE_TYPE=feature\n")
	_, err := c.LoadTarget("p", "")
	wantCode(t, err, "TARGET_INVALID")
}

func TestTestPaths(t *testing.T) {
	c := targetCfg(t)
	writeTarget(t, c, "p", goodTarget+"PILOT_TEST_PATHS=tests/*, src/test/*\n")
	if got, err := c.LoadTarget("p", ""); err != nil || strings.Join(got.TestPaths, "|") != "tests/*|src/test/*" {
		t.Fatalf("%+v %v", got, err)
	}
	writeTarget(t, c, "p", goodTarget+"PILOT_TEST_PATHS=../x\n")
	_, err := c.LoadTarget("p", "")
	wantCode(t, err, "TARGET_INVALID")
}

func TestCompanionPath(t *testing.T) {
	c := targetCfg(t)
	writeTarget(t, c, "p", goodTarget+"PILOT_COMPANION_PATH=/cli-schema-mcp/\n")
	got, err := c.LoadTarget("p", "")
	if err != nil || got.CompanionPath != "cli-schema-mcp" || got.CompanionItemDir("006") != "cli-schema-mcp/items/006" {
		t.Fatalf("%+v %v", got, err)
	}
	if (Target{}).CompanionItemDir("006") != "items/006" {
		t.Fatal("no path: items/ at the root, as before")
	}
	for _, bad := range []string{"../x", "a/../b", "a b", "a/./b", "a\\b"} {
		writeTarget(t, c, "p", goodTarget+"PILOT_COMPANION_PATH="+bad+"\n")
		_, err := c.LoadTarget("p", "")
		wantCode(t, err, "TARGET_INVALID")
	}
}

func TestSecurityLevel(t *testing.T) {
	c := targetCfg(t)
	writeTarget(t, c, "p", goodTarget)
	if got, err := c.LoadTarget("p", ""); err != nil || got.SecurityLevel != LevelMinimal {
		t.Fatalf("default must be minimal: %+v %v", got, err)
	}
	writeTarget(t, c, "p", goodTarget+"PILOT_SECURITY_LEVEL=recommended\n")
	got, err := c.LoadTarget("p", "")
	if err != nil || got.SecurityLevel != LevelRecommended {
		t.Fatalf("%+v %v", got, err)
	}
	if _, err := got.AuthorToken(); err == nil {
		t.Fatal("no author.env: the token must be refused")
	}
	if err := os.WriteFile(got.AuthorEnv, []byte("PILOT_AUTHOR_TOKEN=tok\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if tok, err := got.AuthorToken(); err != nil || tok != "tok" {
		t.Fatalf("%q %v", tok, err)
	}
	_ = os.Chmod(got.AuthorEnv, 0o644)
	if _, err := got.AuthorToken(); err == nil {
		t.Fatal("a world-readable author.env must be refused")
	}
	writeTarget(t, c, "p", goodTarget+"PILOT_SECURITY_LEVEL=max\n")
	_, err = c.LoadTarget("p", "")
	wantCode(t, err, "TARGET_INVALID")
}

// 024: auto-draft is off by default; the watcher's interval and attempts have defaults.
func TestTargetAuthorAuto(t *testing.T) {
	c := targetCfg(t)
	writeTarget(t, c, "g", goodTarget)
	got, err := c.LoadTarget("g", "")
	if err != nil || got.AuthorAuto != "off" || got.AuthorIntervalS != 120 || got.AuthorAttempts != 2 {
		t.Fatalf("%+v %v", got, err)
	}
	writeTarget(t, c, "g", goodTarget+"PILOT_AUTHOR_AUTO=on\nPILOT_AUTHOR_INTERVAL_S=60\nPILOT_AUTHOR_AUTO_ATTEMPTS=1\n")
	got, err = c.LoadTarget("g", "")
	if err != nil || got.AuthorAuto != "on" || got.AuthorIntervalS != 60 || got.AuthorAttempts != 1 {
		t.Fatalf("%+v %v", got, err)
	}
}
