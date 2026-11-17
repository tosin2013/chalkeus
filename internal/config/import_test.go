package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Port of the former Python pilot's tests/test_import_opencode.py.

const importHost = "llm.internal.example"

func hostConfig() map[string]any {
	return map[string]any{
		"$schema": "https://opencode.ai/config.json",
		"providers": map[string]any{
			"corp": map[string]any{"name": "Corp", "package": "@opencode/ai/providers/openai-compatible",
				"settings": map[string]any{"baseURL": "https://" + importHost + "/v1",
					"apiKey": "{file:~/.config/opencode/corp_api_key}"},
				"models": map[string]any{"m1": map[string]any{"name": "M1", "cost": map[string]any{"input": 1, "output": 2}}}},
			"other": map[string]any{"settings": map[string]any{"baseURL": "https://other.example/v1", "apiKey": "literal-in-other"},
				"models": map[string]any{"x": map[string]any{}}},
		},
		"mcp":   map[string]any{"some-server": map[string]any{"type": "remote"}},
		"model": "corp/m1",
	}
}

type importEnv struct{ src, cfg, root string }

func setupImport(t *testing.T) importEnv {
	t.Helper()
	d := t.TempDir()
	e := importEnv{filepath.Join(d, "host-opencode"), filepath.Join(d, "specfoundry"), filepath.Join(d, "repo")}
	for _, p := range []string{e.src, e.root} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	e.writeSrc(t, hostConfig())
	_ = os.WriteFile(filepath.Join(e.src, "corp_api_key"), []byte("secret-value\n"), 0o600)
	_ = os.WriteFile(filepath.Join(e.src, "unrelated_api_key"), []byte("other\n"), 0o600)
	t.Setenv("SPECFOUNDRY_CONFIG", e.cfg)
	return e
}

func (e importEnv) writeSrc(t *testing.T, doc map[string]any) {
	raw, _ := json.Marshal(doc)
	if err := os.WriteFile(filepath.Join(e.src, "opencode.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func (e importEnv) run(model, repo string, force bool) (string, error) {
	var out strings.Builder
	err := RunImport(e.src, model, repo, e.root, force, &out)
	return out.String(), err
}

func mode(t *testing.T, p string) os.FileMode {
	st, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	return st.Mode().Perm()
}

func TestImportsOnlyTheModelProvider(t *testing.T) {
	e := setupImport(t)
	out, err := e.run("", "me/me-pilot-fixture", false)
	if err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(e.cfg, "opencode")
	var doc map[string]any
	raw, _ := os.ReadFile(filepath.Join(dest, "opencode.json"))
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	provs := doc["providers"].(map[string]any)
	if len(provs) != 1 || provs["corp"] == nil {
		t.Fatalf("providers: %v", provs)
	}
	if _, ok := doc["mcp"]; ok {
		t.Error("mcp must be left out")
	}
	cost := provs["corp"].(map[string]any)["models"].(map[string]any)["m1"].(map[string]any)["cost"].(map[string]any)
	if cost["input"] != 1.0 || cost["output"] != 2.0 {
		t.Errorf("cost: %v", cost)
	}
	ents, _ := os.ReadDir(dest)
	var names []string
	for _, x := range ents {
		names = append(names, x.Name())
		if m := mode(t, filepath.Join(dest, x.Name())); m != 0o600 {
			t.Errorf("%s mode %o", x.Name(), m)
		}
	}
	if strings.Join(names, ",") != "corp_api_key,opencode.json" {
		t.Errorf("files: %v", names)
	}
	if m := mode(t, dest); m != 0o700 {
		t.Errorf("dir mode %o", m)
	}
	if !strings.Contains(out, "left out other providers: other") || !strings.Contains(out, "mcp") {
		t.Errorf("output: %s", out)
	}
}

func TestWritesPilotEnvWithoutPrintingHost(t *testing.T) {
	e := setupImport(t)
	out, err := e.run("", "me/me-pilot-fixture", false)
	if err != nil {
		t.Fatal(err)
	}
	pe := filepath.Join(e.cfg, "pilot.env")
	raw, _ := os.ReadFile(pe)
	env := ParseEnv(string(raw))
	if env["PILOT_MODEL"] != "corp/m1" || env["PILOT_MODEL_EGRESS_HOSTS"] != importHost || env["PILOT_FIXTURE_REPO"] != "me/me-pilot-fixture" {
		t.Errorf("pilot.env: %v", env)
	}
	if m := mode(t, pe); m != 0o600 {
		t.Errorf("pilot.env mode %o", m)
	}
	if strings.Contains(out, importHost) || strings.Contains(out, "secret-value") {
		t.Error("output leaks the host or the key")
	}
	if _, err := Load(e.root, false, true); err != nil {
		t.Errorf("result is not a valid config: %v", err)
	}
}

func TestExistingPilotEnvIsComparedNotChanged(t *testing.T) {
	e := setupImport(t)
	_ = os.MkdirAll(e.cfg, 0o700)
	pe := filepath.Join(e.cfg, "pilot.env")
	before := "PILOT_FIXTURE_REPO=me/me-pilot-fixture\nPILOT_MODEL=corp/m1\nPILOT_MODEL_EGRESS_HOSTS=wrong.example\n"
	_ = os.WriteFile(pe, []byte(before), 0o600)
	out, err := e.run("", "", false)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(pe)
	if string(after) != before {
		t.Error("pilot.env changed")
	}
	if !strings.Contains(out, "doesn't include your provider's host") || strings.Contains(out, importHost) {
		t.Errorf("output: %s", out)
	}
}

func TestRefusesLiteralAndEnvKeys(t *testing.T) {
	for _, c := range [][2]string{{"sk-literal", "literal secret"}, {"{env:CORP_KEY}", "{env:...}"}} {
		e := setupImport(t)
		doc := hostConfig()
		doc["providers"].(map[string]any)["corp"].(map[string]any)["settings"].(map[string]any)["apiKey"] = c[0]
		e.writeSrc(t, doc)
		_, err := e.run("", "", false)
		var ce *Error
		if !errors.As(err, &ce) || !strings.Contains(err.Error(), c[1]) || strings.Contains(err.Error(), "sk-literal") {
			t.Errorf("%s: %v", c[0], err)
		}
		if exists(filepath.Join(e.cfg, "opencode")) {
			t.Error("nothing may be copied on refusal")
		}
	}
}

func TestRefusesKeyOutsideWorkerDir(t *testing.T) {
	e := setupImport(t)
	doc := hostConfig()
	doc["providers"].(map[string]any)["corp"].(map[string]any)["settings"].(map[string]any)["apiKey"] = "{file:~/keys/corp.txt}"
	e.writeSrc(t, doc)
	if _, err := e.run("", "", false); err == nil || !strings.Contains(err.Error(), "_api_key") {
		t.Errorf("got %v", err)
	}
}

func TestRefusesToOverwriteWithoutForce(t *testing.T) {
	e := setupImport(t)
	if _, err := e.run("", "me/me-pilot-fixture", false); err != nil {
		t.Fatal(err)
	}
	if _, err := e.run("", "", false); err == nil {
		t.Error("second import without --force must fail")
	}
	if _, err := e.run("", "", true); err != nil {
		t.Errorf("--force: %v", err)
	}
}

func TestModelMustExist(t *testing.T) {
	e := setupImport(t)
	if _, err := e.run("corp/nope", "", false); err == nil || !strings.Contains(err.Error(), "nope") {
		t.Errorf("got %v", err)
	}
}

func TestImportFixtureRepoGuard(t *testing.T) {
	e := setupImport(t)
	_, err := e.run("", "me/important-repo", false)
	var ce *Error
	if !errors.As(err, &ce) || ce.ExitCode() != 3 {
		t.Errorf("got %v", err)
	}
}
