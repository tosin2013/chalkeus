package cli

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tosin2013/chalkeus/internal/config"
	"github.com/tosin2013/chalkeus/internal/debt"
	"github.com/tosin2013/chalkeus/internal/ghloop"
)

func TestWriteReport(t *testing.T) {
	rep := &debt.Report{Version: 1, Repo: "o/r", Target: "t", DefaultBranch: "main", DefaultCommit: "abcdef0123", Credential: "operator-read",
		Counts: map[string]int{}}
	var b bytes.Buffer
	if err := writeReport(&b, rep, false); err != nil || !strings.Contains(b.String(), "debt report for o/r (target t), main @ abcdef0, read with operator-read") ||
		!strings.Contains(b.String(), "forge merges, closes and comments on nothing") {
		t.Fatalf("%v\n%s", err, b.String())
	}
	b.Reset()
	if err := writeReport(&b, rep, true); err != nil || !json.Valid(b.Bytes()) {
		t.Fatal(err)
	}
}

// Token mode: the operator's read client; the publisher's token file is never read.
func TestReportClientOperator(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GH_TOKEN", "operator-read-token")
	pub := filepath.Join(dir, "publisher.env")
	cfg := &config.Config{Values: map[string]string{"PILOT_PUBLISHER_CREDENTIAL": "token"}, ConfigDir: dir, PublisherEnv: pub}
	_, cred, perms, err := reportClient(cfg, config.Target{Name: "t", Repo: "o/r", PublisherEnv: pub})
	if err != nil || cred != "operator-read" || perms != nil {
		t.Fatal(cred, err)
	}
}

// App mode: a read-only token is minted in memory; nothing is written to the config folder.
func TestReportClientApp(t *testing.T) {
	dir := t.TempDir()
	k, _ := rsa.GenerateKey(rand.Reader, 2048)
	key := filepath.Join(dir, "github-app.pem")
	if err := os.WriteFile(key, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}), 0o600); err != nil {
		t.Fatal(err)
	}
	var asked map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/app/installations/9":
			_ = json.NewEncoder(w).Encode(map[string]any{"permissions": map[string]any{"contents": "write", "pull_requests": "write", "issues": "write", "checks": "read"}})
		case "/app/installations/9/access_tokens":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			asked = body["permissions"].(map[string]any)
			w.WriteHeader(201)
			_ = json.NewEncoder(w).Encode(map[string]any{"token": "ghs_R", "permissions": asked})
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	old := ghloop.API
	ghloop.API = srv.URL
	defer func() { ghloop.API = old }()
	cfg := &config.Config{Values: map[string]string{"PILOT_PUBLISHER_CREDENTIAL": "app", "PILOT_APP_ID": "1", "PILOT_APP_INSTALLATION_ID": "9"}, ConfigDir: dir}
	_, cred, perms, err := reportClient(cfg, config.Target{Name: "t", Repo: "o/r"})
	if err != nil || cred != "app-read" || perms["checks"] != "granted" || perms["statuses"] != "missing" {
		t.Fatal(cred, perms, err)
	}
	for p, v := range asked {
		if v != "read" {
			t.Errorf("asked for %s: %v", p, v)
		}
	}
	if ents, _ := os.ReadDir(dir); len(ents) != 1 {
		t.Fatalf("files in the config folder: %v", ents)
	}
}
