package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Port of the former Python pilot's tests/test_config.py.

const pilotEnv = `PILOT_FIXTURE_REPO=owner/foo-pilot-fixture
PILOT_FIXTURE_BRANCH=main
PILOT_MODEL=provider/model
PILOT_MODEL_EGRESS_HOSTS=api.example.com
`

func code(err error) int {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return -1
}

func TestRepoGuard(t *testing.T) {
	if err := RepoGuard("owner/foo-pilot-fixture", "tosin2013/specfoundry"); err != nil {
		t.Fatal(err)
	}
	for _, c := range [][2]string{{"owner/specfoundry", ""}, {"owner/specfoundry-test", ""},
		{"tosin2013/x-pilot-fixture", "tosin2013/x-pilot-fixture"}, {"not a repo", ""}} {
		if err := RepoGuard(c[0], c[1]); code(err) != 3 {
			t.Errorf("%v: %v", c, err)
		}
	}
}

type env struct {
	t *testing.T
	d string
}

func setup(t *testing.T) env {
	d := t.TempDir()
	t.Setenv("SPECFOUNDRY_CONFIG", d)
	e := env{t, d}
	e.write("pilot.env", pilotEnv, 0o600)
	if err := os.Mkdir(filepath.Join(d, "opencode"), 0o700); err != nil {
		t.Fatal(err)
	}
	e.write("opencode/opencode.json", `{"providers": {"p": {"settings": {"baseURL": "https://x", "apiKey": "{file:~/.config/opencode/p_api_key}"}}}}`, 0o600)
	e.write("opencode/p_api_key", "abc", 0o600)
	e.write("opencode/opencode.json.bak.x", `{"apiKey": "sk-literal"}`, 0o600)
	e.write("publisher.env", "PILOT_SCM_TOKEN=xyz\n", 0o600)
	return e
}

func (e env) write(name, text string, mode os.FileMode) {
	p := filepath.Join(e.d, name)
	if err := os.WriteFile(p, []byte(text), mode); err != nil {
		e.t.Fatal(err)
	}
	_ = os.Chmod(p, mode)
}

func TestLoadOK(t *testing.T) {
	setup(t)
	c, err := Load(".", true, false)
	if err != nil {
		t.Fatal(err)
	}
	if c.RetryLimit() != 2 || strings.Join(c.ModelHosts(), ",") != "api.example.com" || c.Get("PILOT_SPEND_CAP_USD") != "1.00" {
		t.Fatalf("%+v", c.Values)
	}
	var names []string
	for _, f := range c.ModelFiles() {
		names = append(names, filepath.Base(f))
	}
	if strings.Join(names, ",") != "opencode.json,p_api_key" {
		t.Fatalf("model files %v", names)
	}
}

func TestMissingFilesSayWhatToDo(t *testing.T) {
	e := setup(t)
	_ = os.Remove(filepath.Join(e.d, "pilot.env"))
	if _, err := Load(".", false, false); err == nil || !strings.Contains(err.Error(), "section 5") {
		t.Fatalf("%v", err)
	}
	e.write("pilot.env", pilotEnv, 0o600)
	_ = os.Remove(filepath.Join(e.d, "publisher.env"))
	if _, err := Load(".", true, false); err == nil || !strings.Contains(err.Error(), "./chalkeus forge publisher-token") {
		t.Fatalf("%v", err)
	}
}

func TestRejections(t *testing.T) {
	cases := map[string]func(e env){
		"key mode 0644": func(e env) { e.write("opencode/p_api_key", "abc", 0o644) },
		"dir mode 0755": func(e env) { _ = os.Chmod(filepath.Join(e.d, "opencode"), 0o755) },
		"literal key": func(e env) {
			e.write("opencode/opencode.json", `{"providers": {"p": {"settings": {"apiKey": "sk-nope"}}}}`, 0o600)
		},
		"scm-like file":   func(e env) { e.write("opencode/github_api_key", "x", 0o600) },
		"missing key":     func(e env) { _ = os.Remove(filepath.Join(e.d, "opencode/p_api_key")) },
		"retry limit 3":   func(e env) { e.write("pilot.env", pilotEnv+"RETRY_LIMIT=3\n", 0o600) },
		"not on/off":      func(e env) { e.write("pilot.env", pilotEnv+"PILOT_GITHUB_ISSUES=yes\n", 0o600) },
		"no approvers":    func(e env) { e.write("pilot.env", pilotEnv+"PILOT_GITHUB_APPROVALS=on\n", 0o600) },
		"bad credential":  func(e env) { e.write("pilot.env", pilotEnv+"PILOT_PUBLISHER_CREDENTIAL=pat\n", 0o600) },
		"app without ids": func(e env) { e.write("pilot.env", pilotEnv+"PILOT_PUBLISHER_CREDENTIAL=app\n", 0o600) },
	}
	for name, mut := range cases {
		t.Run(name, func(t *testing.T) {
			e := setup(t)
			mut(e)
			_, err := Load(".", true, false)
			if err == nil {
				t.Fatal("expected an error")
			}
			if strings.Contains(err.Error(), "sk-nope") {
				t.Fatal("secret in error message")
			}
		})
	}
}

func TestGuardOnLoad(t *testing.T) {
	e := setup(t)
	e.write("pilot.env", strings.Replace(pilotEnv, "foo-pilot-fixture", "specfoundry", 1), 0o600)
	if _, err := Load(".", true, true); code(err) != 3 {
		t.Fatalf("%v", err)
	}
}

func TestGitHubLoopKeys(t *testing.T) {
	e := setup(t)
	e.write("pilot.env", pilotEnv+"PILOT_GITHUB_APPROVALS=on\nPILOT_APPROVERS=alice, @bob\n", 0o600)
	c, err := Load(".", false, false)
	if err != nil {
		t.Fatal(err)
	}
	if !c.GithubIssues() || strings.Join(c.Approvers(), ",") != "alice,bob" {
		t.Fatalf("issues=%v approvers=%v", c.GithubIssues(), c.Approvers())
	}
	if cred, err := c.Credential(); err != nil || cred != "token" {
		t.Fatalf("%s %v", cred, err)
	}
	e.write("github-app.pem", "x", 0o600)
	if _, err := c.Credential(); err == nil {
		t.Fatal("must refuse to guess when token and key both exist")
	}
	e.write("pilot.env", pilotEnv+"PILOT_PUBLISHER_CREDENTIAL=app\nPILOT_APP_ID=1\nPILOT_APP_INSTALLATION_ID=2\n", 0o600)
	c, _ = Load(".", false, false)
	if cred, _ := c.Credential(); cred != "app" {
		t.Fatal(cred)
	}
}

func TestPublicHashesHosts(t *testing.T) {
	c := &Config{Values: map[string]string{"PILOT_MODEL_EGRESS_HOSTS": "api.example.com"}}
	if v := c.Public()["PILOT_MODEL_EGRESS_HOSTS"]; !strings.HasPrefix(v, "sha256:") || strings.Contains(v, "example") {
		t.Fatal(v)
	}
}
