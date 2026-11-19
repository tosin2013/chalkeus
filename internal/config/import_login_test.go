package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeJSON(t *testing.T, p string, v any) {
	t.Helper()
	raw, _ := json.Marshal(v)
	if err := os.WriteFile(p, raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoginSourceFromCatalog(t *testing.T) {
	d := t.TempDir()
	auth, cache, cfg := filepath.Join(d, "auth.json"), filepath.Join(d, "cache"), filepath.Join(d, "cfg")
	_ = os.MkdirAll(cache, 0o700)
	_ = os.MkdirAll(cfg, 0o700)
	writeJSON(t, auth, map[string]any{"gw": map[string]any{"type": "api", "key": "sk-SECRET"}})
	writeJSON(t, filepath.Join(cache, "models.json"), map[string]any{"gw": map[string]any{"api": "https://api.gw.example/v1"}})
	src, cleanup, err := LoginSource(auth, cfg, cache, "gw/fast-1", "")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	p, err := PlanImport(src, "")
	if err != nil {
		t.Fatalf("the generated config must pass the normal import checks: %v", err)
	}
	if p.Model != "gw/fast-1" || p.Host != "api.gw.example" || strings.Join(p.Keys, ",") != "gw_api_key" {
		t.Fatalf("%+v", p)
	}
	raw, _ := os.ReadFile(filepath.Join(src, "opencode.json"))
	if strings.Contains(string(raw), "sk-SECRET") {
		t.Fatal("the key must be in the key file only")
	}
	if st, _ := os.Stat(filepath.Join(src, "gw_api_key")); st.Mode().Perm() != 0o600 {
		t.Fatal("key file must be 0600")
	}
}

func TestLoginSourceKeepsTheUsersProviderBlock(t *testing.T) {
	d := t.TempDir()
	auth, cfg := filepath.Join(d, "auth.json"), filepath.Join(d, "cfg")
	_ = os.MkdirAll(cfg, 0o700)
	writeJSON(t, auth, map[string]any{"corp": map[string]any{"type": "api", "key": "k"}})
	writeJSON(t, filepath.Join(cfg, "opencode.json"), map[string]any{"providers": map[string]any{"corp": map[string]any{
		"settings": map[string]any{"baseURL": "https://llm.corp.example/v1", "apiKey": "literal-old"},
		"models":   map[string]any{"m1": map[string]any{"name": "M1"}}}}})
	src, cleanup, err := LoginSource(auth, cfg, filepath.Join(d, "nocache"), "corp/m1", "")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	p, err := PlanImport(src, "")
	if err != nil || p.Host != "llm.corp.example" {
		t.Fatalf("%+v %v", p, err)
	}
	raw, _ := os.ReadFile(filepath.Join(src, "opencode.json"))
	if strings.Contains(string(raw), "literal-old") || !strings.Contains(string(raw), `"name": "M1"`) {
		t.Fatal(string(raw))
	}
}

func TestLoginSourceRefusals(t *testing.T) {
	d := t.TempDir()
	auth := filepath.Join(d, "auth.json")
	writeJSON(t, auth, map[string]any{
		"sub":    map[string]any{"type": "oauth", "access": "a", "refresh": "r"},
		"nohost": map[string]any{"type": "api", "key": "k"},
	})
	for name, c := range map[string]struct{ model, base, want string }{
		"no model":      {"", "", "--model"},
		"not logged in": {"other/m", "", "not logged in"},
		"oauth":         {"sub/m", "", "API keys only"},
		"no endpoint":   {"nohost/m", "", "--base-url"},
		"http endpoint": {"nohost/m", "http://plain.example/v1", "--base-url"},
	} {
		_, cleanup, err := LoginSource(auth, d, d, c.model, c.base)
		cleanup()
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v", name, err)
		}
		if err != nil && strings.Contains(err.Error(), "\"k\"") {
			t.Errorf("%s: an error must never carry the key", name)
		}
	}
	if _, cleanup, err := LoginSource(filepath.Join(d, "missing.json"), d, d, "x/y", ""); err == nil {
		t.Error("a missing auth file must refuse")
	} else {
		cleanup()
	}
	src, cleanup, err := LoginSource(auth, d, d, "nohost/m", "https://api.nohost.example/v1")
	defer cleanup()
	if err != nil {
		t.Fatal(err)
	}
	if p, err := PlanImport(src, ""); err != nil || p.Host != "api.nohost.example" {
		t.Fatalf("--base-url: %+v %v", p, err)
	}
}

// OpenCode 2.2 keeps logins in its database; `opencode auth export` prints them as a list.
func TestLoginSourceFromAuthExport(t *testing.T) {
	d := t.TempDir()
	auth, cfg := filepath.Join(d, "export.json"), filepath.Join(d, "cfg")
	_ = os.MkdirAll(cfg, 0o700)
	writeJSON(t, auth, []any{
		map[string]any{"id": "1", "integrationID": "opencode-go", "label": "old", "active": false, "value": map[string]any{"type": "key", "key": "sk-OLD"}},
		map[string]any{"id": "2", "integrationID": "opencode-go", "label": "new", "active": true, "value": map[string]any{"type": "key", "key": "sk-NEW"}},
	})
	src, cleanup, err := LoginSource(auth, cfg, filepath.Join(d, "nocache"), "opencode-go/deepseek-v4-pro", "https://gateway.example/go/openai/v1")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	p, err := PlanImport(src, "")
	if err != nil || p.Host != "gateway.example" || p.Model != "opencode-go/deepseek-v4-pro" {
		t.Fatalf("%+v %v", p, err)
	}
	key, _ := os.ReadFile(filepath.Join(src, "opencode-go_api_key"))
	if strings.TrimSpace(string(key)) != "sk-NEW" {
		t.Fatal("the active login's key must be used")
	}
	// Only inactive logins for the provider: not logged in.
	writeJSON(t, auth, []any{map[string]any{"integrationID": "opencode-go", "active": false, "value": map[string]any{"type": "key", "key": "k"}}})
	if _, c, err := LoginSource(auth, cfg, filepath.Join(d, "nocache"), "opencode-go/m", "https://gateway.example/v1"); err == nil || !strings.Contains(err.Error(), "not logged in") {
		c()
		t.Fatalf("got %v", err)
	}
	// An OAuth entry in the export is still refused.
	writeJSON(t, auth, []any{map[string]any{"integrationID": "sub", "active": true, "value": map[string]any{"type": "oauth", "access": "a"}}})
	if _, c, err := LoginSource(auth, cfg, filepath.Join(d, "nocache"), "sub/m", "https://gateway.example/v1"); err == nil || !strings.Contains(err.Error(), "API keys only") {
		c()
		t.Fatalf("got %v", err)
	}
}
