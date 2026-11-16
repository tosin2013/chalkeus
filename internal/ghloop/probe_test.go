package ghloop

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tosin2013/specfoundry/internal/secret"
)

// Port of the former Python pilot's tests/test_check_token.py against a fake GitHub API.

const repo, canary = "o/x-pilot-fixture", "o/specfoundry"

type fakeAPI struct {
	user, repoWrite, canaryWrite, pulls, labels int
	sawUser                                     bool
}

func (f *fakeAPI) server(t *testing.T) func() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			t.Error("no bearer token")
		}
		switch {
		case r.URL.Path == "/user":
			f.sawUser = true
			w.Header().Set("GitHub-Authentication-Token-Expiration", "2026-11-20 00:00:00 UTC")
			w.WriteHeader(f.user)
			_ = json.NewEncoder(w).Encode(map[string]string{"login": "bot"})
		case r.Method == "POST" && r.URL.Path == "/repos/"+repo+"/git/blobs":
			w.WriteHeader(f.repoWrite)
		case r.Method == "POST" && r.URL.Path == "/repos/"+canary+"/git/blobs":
			w.WriteHeader(f.canaryWrite)
		case r.Method == "POST" && r.URL.Path == "/repos/"+repo+"/labels":
			var b map[string]string
			_ = json.NewDecoder(r.Body).Decode(&b)
			if b["color"] != "not-a-color" {
				t.Error("issues probe must use an invalid colour")
			}
			w.WriteHeader(f.labels)
		case strings.HasPrefix(r.URL.Path, "/repos/"+repo+"/pulls"):
			w.WriteHeader(f.pulls)
		case r.URL.Path == "/repos/"+repo:
			w.WriteHeader(200)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	old := API
	API = srv.URL
	return func() { API = old; srv.Close() }
}

func good() *fakeAPI {
	return &fakeAPI{user: 200, repoWrite: 201, canaryWrite: 404, pulls: 200, labels: 422}
}

func TestProbe(t *testing.T) {
	c := Credential{Token: secret.New("t")}
	cases := map[string]struct {
		api     *fakeAPI
		issues  bool
		ok      bool
		problem string
	}{
		"ok":              {good(), false, true, ""},
		"rejected":        {&fakeAPI{user: 401, repoWrite: 201, canaryWrite: 404, pulls: 200, labels: 422}, false, false, "token rejected by GitHub (HTTP 401)"},
		"no write":        {&fakeAPI{user: 200, repoWrite: 403, canaryWrite: 404, pulls: 200, labels: 422}, false, false, "needs Contents: read and write"},
		"canary writable": {&fakeAPI{user: 200, repoWrite: 201, canaryWrite: 201, pulls: 200, labels: 422}, false, false, "scope it to the fixture repo only"},
		"no pulls":        {&fakeAPI{user: 200, repoWrite: 201, canaryWrite: 404, pulls: 403, labels: 422}, false, false, "needs Pull requests"},
		"issues ok":       {good(), true, true, ""},
		"issues missing":  {&fakeAPI{user: 200, repoWrite: 201, canaryWrite: 404, pulls: 200, labels: 403}, true, false, "Issues: read and write"},
		"issues unasked":  {&fakeAPI{user: 200, repoWrite: 201, canaryWrite: 404, pulls: 200, labels: 403}, false, true, ""},
	}
	for name, cs := range cases {
		t.Run(name, func(t *testing.T) {
			defer cs.api.server(t)()
			r := Probe(c, repo, canary, cs.issues)
			if r.OK != cs.ok || (cs.problem != "" && !strings.Contains(strings.Join(r.Problems, ";"), cs.problem)) {
				t.Fatalf("%+v", r)
			}
		})
	}
}

func TestAppTokenSkipsUser(t *testing.T) {
	api := good()
	defer api.server(t)()
	r := Probe(Credential{Token: secret.New("t"), Kind: "app", Login: "pilot[bot]"}, repo, canary, true)
	if !r.OK || api.sawUser || *r.Login != "pilot[bot]" {
		t.Fatalf("%+v sawUser=%v", r, api.sawUser)
	}
}

func TestReportFormat(t *testing.T) {
	var b strings.Builder
	l := "bot"
	Report(&b, ProbeResult{OK: true, Login: &l, Repo: repo, RepoPush: true, Problems: []string{}})
	want := "publisher token: OK\n  login: bot\n  repo: o/x-pilot-fixture\n  repo_push: True\n  canary: None\n  canary_push: None\n  expires: None\n"
	if b.String() != want {
		t.Fatalf("%q", b.String())
	}
}
