package config

import (
	"github.com/tosin2013/specfoundry/internal/names"
	"net/url"
	"strings"
)

// TestEndpoints are the integration tests' settings (025, contracts/test-mode.md): the GitHub
// API and git base addresses of a fake GitHub. They exist only for tests.
type TestEndpoints struct {
	On      bool   // FORGE_TEST_MODE=1 (old: PILOT_TEST_MODE): every run is marked as a test run
	API     string // replaces https://api.github.com ("" keeps it)
	GitBase string // replaces https://github.com/ ("" keeps it)
}

// TestMode reads the test settings from the environment. Either address without
// FORGE_TEST_MODE=1 is refused, so a real run can never be pointed elsewhere by accident.
func TestMode() (TestEndpoints, error) {
	t := TestEndpoints{On: names.Env("TEST_MODE") == "1",
		API: strings.TrimSpace(names.Env("TEST_GITHUB_API")), GitBase: strings.TrimSpace(names.Env("TEST_GIT_BASE"))}
	if (t.API != "" || t.GitBase != "") && !t.On {
		return TestEndpoints{}, cerr("test endpoints need FORGE_TEST_MODE=1 (they are for the integration tests only)")
	}
	for name, v := range map[string]string{"PILOT_TEST_GITHUB_API": t.API, "PILOT_TEST_GIT_BASE": t.GitBase} {
		if v == "" {
			continue
		}
		u, err := url.Parse(v)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return TestEndpoints{}, cerr("%s must be an http(s) URL", name)
		}
	}
	if t.GitBase != "" && !strings.HasSuffix(t.GitBase, "/") {
		t.GitBase += "/"
	}
	return t, nil
}

// Record is what a test run's config.json records ("" values left out).
func (t TestEndpoints) Record() map[string]any {
	if !t.On {
		return nil
	}
	m := map[string]any{"test_mode": true}
	if t.API != "" {
		m["test_github_api"] = t.API
	}
	if t.GitBase != "" {
		m["test_git_base"] = t.GitBase
	}
	return m
}
