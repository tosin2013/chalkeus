package ghloop

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tosin2013/chalkeus/internal/secret"
)

func labelServer(t *testing.T, labels []string, events []map[string]any) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/repos/me/proj/issues/41":
			var ls []any
			for _, l := range labels {
				ls = append(ls, map[string]any{"name": l})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"number": 41, "labels": ls})
		case "/repos/me/proj/issues/41/events":
			page := r.URL.Query().Get("page")
			if page == "" || page == "1" { // two pages: the second holds the newest event
				w.Header().Set("Link", fmt.Sprintf(`<%s/repos/me/proj/issues/41/events?page=2>; rel="next"`, "http://"+r.Host))
				_ = json.NewEncoder(w).Encode(events[:len(events)-1])
				return
			}
			_ = json.NewEncoder(w).Encode(events[len(events)-1:])
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	old := API
	API = srv.URL
	t.Cleanup(func() { API = old })
}

func ev(kind, label, login string) map[string]any {
	return map[string]any{"event": kind, "label": map[string]any{"name": label}, "actor": map[string]any{"login": login}}
}

func TestProposedBy(t *testing.T) {
	ctx := context.Background()
	labelServer(t, []string{"status:specced", "pilot/go"}, []map[string]any{
		ev("labeled", "status:specced", "bot"), ev("labeled", "pilot/go", "alice"), ev("unlabeled", "pilot/go", "alice"),
		ev("labeled", "pilot/go", "mallory")})
	p, err := ProposedBy(ctx, NewClient(secret.New("t")), "me/proj", 41, "pilot/go")
	if err != nil || !p.Labelled || p.AddedBy != "mallory" {
		t.Fatalf("the last labeled event wins, across pages: %+v %v", p, err)
	}
	if IsApprover(p.AddedBy, []string{"alice", "@bob"}) || !IsApprover("Bob", []string{"alice", "@bob"}) || IsApprover("", []string{""}) {
		t.Fatal("approver matching")
	}
	labelServer(t, []string{"status:specced"}, []map[string]any{ev("labeled", "pilot/go", "alice"), ev("unlabeled", "pilot/go", "alice")})
	if p, _ := ProposedBy(ctx, NewClient(secret.New("t")), "me/proj", 41, "pilot/go"); p.Labelled || p.AddedBy != "" {
		t.Fatalf("a removed label proposes nothing: %+v", p)
	}
}

func TestPlanningPRs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/repos/me/proj/pulls":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"number": 46, "title": "docs(010): plan", "html_url": "https://x/pull/46", "head": map[string]any{"ref": "plan-010"}},
				{"number": 47, "title": "fix typo", "html_url": "https://x/pull/47", "head": map[string]any{"ref": "typo"}},
				{"number": 48, "title": "feat(010)", "html_url": "https://x/pull/48", "head": map[string]any{"ref": "pilot/R1/010"}}})
		case "/repos/me/proj/pulls/46/files":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"filename": "specs/010-x/spec.md"}})
		case "/repos/me/proj/pulls/47/files":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"filename": "README.md"}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	old := API
	API = srv.URL
	defer func() { API = old }()
	got, err := PlanningPRs(context.Background(), NewClient(secret.New("t")), "me/proj")
	if err != nil || len(got) != 1 || got[0] != "https://x/pull/46 (docs(010): plan)" {
		t.Fatalf("%v %v", got, err)
	}
}
