package ghloop

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/tosin2013/specfoundry/internal/secret"
)

// Cases ported from the former Python pilot's tests/test_ghops.py: the writes against a fake GitHub API; the token
// never appears in a result or error; there is no merge operation.

type writeAPI struct {
	labels map[string]bool
	calls  []string
	status int // forced status for every call (0: normal)
}

func (f *writeAPI) serve(t *testing.T) GHWriter {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.calls = append(f.calls, r.Method+" "+r.URL.EscapedPath())
		if f.status != 0 {
			w.WriteHeader(f.status)
			return
		}
		p := r.URL.Path
		reply := func(code int, v any) { w.WriteHeader(code); _ = json.NewEncoder(w).Encode(v) }
		switch {
		case p == "/user":
			reply(200, map[string]any{"login": "pub-bot"})
		case r.Method == "GET" && strings.Contains(p, "/labels/"):
			if f.labels[p[strings.LastIndex(p, "/")+1:]] {
				reply(200, map[string]any{})
			} else {
				reply(404, map[string]any{"message": "Not Found"})
			}
		case r.Method == "POST" && strings.HasSuffix(p, "/labels"):
			reply(201, map[string]any{})
		case r.Method == "POST" && strings.HasSuffix(p, "/issues"):
			reply(201, map[string]any{"number": 7, "html_url": "u"})
		case r.Method == "PATCH":
			reply(200, map[string]any{"number": 7})
		case r.Method == "POST" && strings.HasSuffix(p, "/comments"):
			reply(201, map[string]any{"id": 99, "html_url": "c"})
		default:
			reply(500, nil)
		}
	}))
	t.Cleanup(srv.Close)
	old := API
	API = srv.URL
	t.Cleanup(func() { API = old })
	tok := secret.New("SECRET-VALUE-123")
	return GHWriter{C: NewClient(tok), Cred: Credential{Token: tok}}
}

func TestWriterWhoami(t *testing.T) {
	w := (&writeAPI{}).serve(t)
	if l, err := w.Whoami(context.Background()); err != nil || l != "pub-bot" {
		t.Fatal(l, err)
	}
	app := GHWriter{Cred: Credential{Kind: "app", Login: "bot[bot]"}}
	if l, _ := app.Whoami(context.Background()); l != "bot[bot]" {
		t.Fatal(l)
	}
}

func TestEnsureLabelsOnlyCreatesMissing(t *testing.T) {
	f := &writeAPI{labels: map[string]bool{"status:planned": true}}
	w := f.serve(t)
	got, err := w.EnsureLabels(context.Background(), "o/r", []Label{{Name: "status:planned"}, {Name: "status:verified"}})
	if err != nil || !reflect.DeepEqual(got, []string{"status:verified"}) {
		t.Fatal(got, err)
	}
}

func TestCreateCommentClose(t *testing.T) {
	w := (&writeAPI{}).serve(t)
	ctx := context.Background()
	if n, err := w.CreateIssue(ctx, "o/r", "t", "b", nil); err != nil || n != 7 {
		t.Fatal(n, err)
	}
	if id, err := w.Comment(ctx, "o/r", 7, "x"); err != nil || id != 99 {
		t.Fatal(id, err)
	}
	if err := w.CloseIssue(ctx, "o/r", 7); err != nil {
		t.Fatal(err)
	}
}

func TestWriteErrorCarriesNoToken(t *testing.T) {
	w := (&writeAPI{status: 403}).serve(t)
	_, err := w.CreateIssue(context.Background(), "o/r", "t", "b", nil)
	if err == nil || err.Error() != "ghops create-issue: create issue failed (HTTP 403)" {
		t.Fatal(err)
	}
	if strings.Contains(err.Error(), "SECRET-VALUE") {
		t.Fatal("token in error")
	}
}
