package config

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func probeCfg(t *testing.T, base string) *Config {
	t.Helper()
	d := t.TempDir()
	writeJSON(t, filepath.Join(d, "opencode.json"), map[string]any{"model": "gw/m", "providers": map[string]any{
		"gw": map[string]any{"settings": map[string]any{"baseURL": base + "/v1", "apiKey": "{file:~/.config/opencode/gw_api_key}"}}}})
	if err := os.WriteFile(filepath.Join(d, "gw_api_key"), []byte("sk-PROBE\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return &Config{ModelDir: d}
}

func TestProbeModel(t *testing.T) {
	var auth string
	blocked := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		if r.URL.Path != "/v1/chat/completions" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		if blocked {
			w.WriteHeader(401)
			_, _ = w.Write([]byte(`{"error":{"message":"Authentication Error, Key is blocked. key=sk-PROBE"}}`))
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["model"] != "m" || body["max_tokens"] != float64(1) {
			w.WriteHeader(400)
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"OK"}}]}`))
	}))
	defer srv.Close()
	c := probeCfg(t, srv.URL)
	p, err := c.ProbeModel(context.Background())
	if err != nil || p.Status != 200 || auth != "Bearer sk-PROBE" || p.Provider != "gw" {
		t.Fatalf("%+v %v %q", p, err, auth)
	}
	blocked = true
	p, err = c.ProbeModel(context.Background())
	if err != nil || p.Status != 401 || !strings.Contains(p.Message, "Key is blocked") || strings.Contains(p.Message, "sk-PROBE") {
		t.Fatalf("%+v %v", p, err)
	}
	if _, err := (&Config{ModelDir: t.TempDir()}).ProbeModel(context.Background()); err == nil {
		t.Fatal("no model config must be an error")
	}
}
