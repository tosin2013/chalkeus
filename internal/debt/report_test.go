package debt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v85/github"
)

// recorded serves a testdata/<name>/responses.json snapshot and records every request's method.
type recorded struct {
	Repo          string `json:"repo"`
	DefaultCommit string `json:"default_commit"`
	Responses     map[string]struct {
		Status int             `json:"status"`
		Body   json.RawMessage `json:"body"`
	} `json:"responses"`
	methods []string
	misses  []string
}

func key(r *http.Request) string {
	q := r.URL.Query()
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		parts = append(parts, k+"="+q.Get(k))
	}
	p, _ := url.PathUnescape(r.URL.Path)
	p = strings.TrimPrefix(p, "/api/v3")
	if len(parts) > 0 {
		p += "?" + strings.Join(parts, "&")
	}
	return r.Method + " " + p
}

func serveRecorded(t *testing.T, dir string) (*recorded, Reader) {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + dir + "/responses.json")
	if err != nil {
		t.Fatal(err)
	}
	rec := &recorded{}
	if err := json.Unmarshal(raw, rec); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.methods = append(rec.methods, r.Method)
		resp, ok := rec.Responses[key(r)]
		if !ok {
			if !strings.Contains(r.URL.Path, "/contents/") { // a file not recorded doesn't exist
				rec.misses = append(rec.misses, key(r))
			}
			w.WriteHeader(404)
			return
		}
		w.WriteHeader(resp.Status)
		if resp.Status != 200 {
			_, _ = w.Write([]byte(`{"message":"recorded status"}`))
			return
		}
		if strings.HasPrefix(string(resp.Body), `"`) { // a raw file
			var s string
			_ = json.Unmarshal(resp.Body, &s)
			_, _ = w.Write([]byte(s))
			return
		}
		_, _ = w.Write(resp.Body)
	}))
	t.Cleanup(srv.Close)
	c := github.NewClient(nil)
	u, _ := url.Parse(srv.URL + "/")
	c.BaseURL = u
	return rec, GitHub{C: c}
}

func buildDocumcp(t *testing.T) (*Report, *recorded) {
	t.Helper()
	rec, r := serveRecorded(t, "documcp-20261124")
	rep, err := Build(context.Background(), r, Options{Target: "documcp", Repo: rec.Repo, Credential: "operator-read",
		Now: time.Date(2026, 11, 24, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	return rep, rec
}

// SC-001 and SC-002 on the recorded documcp snapshot (22 Dependabot PRs; npm audit red on main).
func TestDocumcpSnapshot(t *testing.T) {
	rep, rec := buildDocumcp(t)
	for _, m := range rec.methods {
		if m != "GET" {
			t.Fatalf("the report made a %s request", m)
		}
	}
	if len(rec.misses) > 0 {
		t.Fatalf("requests not in the recording: %v", rec.misses)
	}
	var buf bytes.Buffer
	rep.WriteText(&buf)
	t.Log("\n" + buf.String())
	class := map[int]string{}
	for _, g := range rep.Groups {
		for _, e := range g.PRs {
			class[e.Number] = e.Class
		}
	}
	if len(class) != 22 || len(rep.NotTriaged) != 0 {
		t.Fatalf("%d PRs triaged, %d not", len(class), len(rep.NotTriaged))
	}
	// SC-001: the shared root cause is one group, and every PR in it fails only shared checks.
	big := rep.Groups[0]
	if len(big.PRs) != 19 || len(big.Shared) != len(big.FailingChecks) || !contains(big.Shared, "Security Audit") {
		t.Fatalf("largest group: %d PRs, failing %v, shared %v", len(big.PRs), big.FailingChecks, big.Shared)
	}
	var sup []int
	for n, c := range class {
		if c == ClassSuperseded {
			sup = append(sup, n)
		}
	}
	sort.Ints(sup)
	if fmt.Sprint(sup) != "[173 176 177 181 188 190 198 199 200 202]" {
		t.Errorf("superseded %v", sup)
	}
	for n, want := range map[int]string{180: ClassNeedsCode, 182: ClassNeedsCode, 195: ClassNeedsCode, 172: ClassSafe, 185: ClassSafe} {
		if class[n] != want {
			t.Errorf("#%d is %s, want %s", n, class[n], want)
		}
	}
	if rep.SharedChecks["auto-merge"] == "" || rep.SharedChecks["Test Suite (20.x)"] != "" {
		t.Errorf("shared checks %v", rep.SharedChecks)
	}
	if rep.Advisories() != Unknown {
		t.Errorf("advisories should be unknown without access, got %v", rep.Advisories())
	}
	kinds := map[string]int{}
	for _, f := range rep.BaselineFlags {
		kinds[f.Kind]++
	}
	if kinds[FlagRedDefault] != 3 || kinds[FlagForceExit] != 1 || kinds[FlagInstallScripts] != 1 {
		t.Errorf("flags %v", kinds)
	}
	var js bytes.Buffer
	if err := rep.WriteJSON(&js); err != nil || !json.Valid(js.Bytes()) {
		t.Fatal(err)
	}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// Advisories is "unknown" when the report couldn't read them.
func (r *Report) Advisories() any {
	for _, u := range r.Unreadable {
		if strings.HasPrefix(u, "advisories") {
			return Unknown
		}
	}
	return "read"
}
