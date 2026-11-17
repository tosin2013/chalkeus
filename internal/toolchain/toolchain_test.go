package toolchain

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShippedPythonToolchainParses(t *testing.T) {
	root, _ := filepath.Abs("../..")
	raw, err := os.ReadFile(filepath.Join(root, "pilot/images/toolchain-python3.13/toolchain.json"))
	if err != nil {
		t.Fatal(err)
	}
	s, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	env := s.PodEnv("/deps", "/usr/bin:/bin")
	got := map[string]string{}
	for _, kv := range env {
		got[kv[0]] = kv[1]
	}
	if got["PILOT_DEPS"] != "/deps" || got["VIRTUAL_ENV"] != "/deps/venv" ||
		got["PATH"] != "/deps/venv/bin:/opt/toolchain/python/bin:/usr/bin:/bin" || s.Proxy != "env" {
		t.Fatalf("%v", got)
	}
	script := s.SetupScript("pip install -r requirements.txt")
	if !strings.Contains(script, "python3 -m venv") || !strings.HasSuffix(script, "pip install -r requirements.txt\n") {
		t.Fatal(script)
	}
	y := EnvYAML(env[:2], "  ")
	if !strings.HasPrefix(y, `  - {name: PILOT_DEPS, value: "/deps"}`) {
		t.Fatal(y)
	}
}

func TestContractRules(t *testing.T) {
	ok := `{"name":"x","version_command":["x","-v"],"path":["/opt/toolchain/bin"],"env":{"X_HOME":"/opt/toolchain"},"proxy":"env"}`
	if _, err := Parse([]byte(ok)); err != nil {
		t.Fatal(err)
	}
	for name, bad := range map[string]string{
		"not json":      `{`,
		"no name":       `{"version_command":["x"],"proxy":"env"}`,
		"no version":    `{"name":"x","proxy":"env"}`,
		"path outside":  `{"name":"x","version_command":["x"],"path":["/usr/local/bin"],"proxy":"env"}`,
		"sets proxy":    `{"name":"x","version_command":["x"],"env":{"HTTPS_PROXY":"h"},"proxy":"env"}`,
		"sets PATH":     `{"name":"x","version_command":["x"],"env":{"PATH":"/x"},"proxy":"env"}`,
		"secret-like":   `{"name":"x","version_command":["x"],"env":{"NPM_TOKEN":"t"},"proxy":"env"}`,
		"bad var name":  `{"name":"x","version_command":["x"],"env":{"A-B":"t"},"proxy":"env"}`,
		"unknown proxy": `{"name":"x","version_command":["x"],"proxy":"socks"}`,
	} {
		if _, err := Parse([]byte(bad)); err == nil || !strings.HasPrefix(err.Error(), "TOOLCHAIN_CONTRACT:") {
			t.Errorf("%s: %v", name, err)
		}
	}
	s, _ := Parse([]byte(`{"name":"x","version_command":["x"],"setup_command":["sh","-c","echo 'hi'"],"proxy":"env"}`))
	if !strings.Contains(s.SetupScript(""), `'sh' '-c' 'echo '"'"'hi'"'"''`) {
		t.Fatal(s.SetupScript(""))
	}
}
