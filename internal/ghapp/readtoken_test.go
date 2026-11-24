package ghapp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/tosin2013/chalkeus/internal/ghloop"
)

// readAPI fakes the installation and token endpoints; it records each token request's body.
type readAPI struct {
	granted  map[string]any // the installation's permissions
	returned map[string]any // nil: echo the requested permissions
	bodies   []map[string]any
}

func (f *readAPI) serve(t *testing.T) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/app/installations/456":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 456, "permissions": f.granted})
		case r.Method == "POST" && r.URL.Path == "/app/installations/456/access_tokens":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			f.bodies = append(f.bodies, body)
			perms := f.returned
			if perms == nil {
				perms = map[string]any{"metadata": "read"}
				for k, v := range body["permissions"].(map[string]any) {
					perms[k] = v
				}
			}
			w.WriteHeader(201)
			_ = json.NewEncoder(w).Encode(map[string]any{"token": "ghs_READONLYREADONLY1234", "permissions": perms})
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)
	old := ghloop.API
	ghloop.API = srv.URL
	t.Cleanup(func() { ghloop.API = old })
}

func TestReadTokenAsksOnlyForReadAndWritesNothing(t *testing.T) {
	dir := t.TempDir()
	key, _ := writeKey(t, dir)
	f := &readAPI{granted: map[string]any{"contents": "write", "pull_requests": "write", "issues": "write", "checks": "read", "metadata": "read"}}
	f.serve(t)
	tok, perms, err := ReadToken(key, "123", "456", repo, time.Now())
	if err != nil || tok.Reveal() != "ghs_READONLYREADONLY1234" {
		t.Fatal(err)
	}
	if perms["checks"] != "granted" || perms["statuses"] != "missing" || perms["vulnerability_alerts"] != "missing" {
		t.Fatal(perms)
	}
	req := f.bodies[0]["permissions"].(map[string]any)
	if len(req) != 3 || req["contents"] != "read" || req["pull_requests"] != "read" || req["checks"] != "read" {
		t.Fatalf("requested %v", req)
	}
	if r := f.bodies[0]["repositories"].([]any); len(r) != 1 || r[0] != "x-pilot-fixture" {
		t.Fatal(r)
	}
	if ents, _ := os.ReadDir(dir); len(ents) != 1 { // only the key
		t.Fatalf("files written: %v", ents)
	}
}

func TestReadTokenRefuses(t *testing.T) {
	dir := t.TempDir()
	key, _ := writeKey(t, dir)
	for name, f := range map[string]*readAPI{
		"required permission missing": {granted: map[string]any{"contents": "read"}},
		"token came back with write":  {granted: map[string]any{"contents": "write", "pull_requests": "write"}, returned: map[string]any{"contents": "write", "pull_requests": "read"}},
		"token carries an extra":      {granted: map[string]any{"contents": "write", "pull_requests": "write"}, returned: map[string]any{"contents": "read", "pull_requests": "read", "issues": "write"}},
	} {
		f.serve(t)
		if _, _, err := ReadToken(key, "123", "456", repo, time.Now()); err == nil {
			t.Errorf("%s: no error", name)
		} else if strings.Contains(err.Error(), "ghs_") {
			t.Errorf("%s: token in error", name)
		}
	}
}
