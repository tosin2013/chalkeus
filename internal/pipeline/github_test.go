package pipeline

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/tosin2013/specfoundry/internal/config"
	"github.com/tosin2013/specfoundry/internal/ghloop"
)

// A minimal in-memory GitHub for the pipeline's 004 hooks.
type miniGH struct {
	comments map[int][]ghloop.Comment
	next     int
	cid      int64
	fail     bool
}

func (m *miniGH) Issues(context.Context, string) ([]ghloop.Issue, error) { return nil, m.err() }
func (m *miniGH) Labels(context.Context, string) (map[string]bool, error) {
	return map[string]bool{}, m.err()
}
func (m *miniGH) Comments(_ context.Context, _ string, n int) ([]ghloop.Comment, error) {
	return m.comments[n], m.err()
}
func (m *miniGH) Permission(context.Context, string, string) string { return "write" }
func (m *miniGH) PR(context.Context, string, int) (ghloop.PRView, error) {
	return ghloop.PRView{}, m.err()
}
func (m *miniGH) err() error {
	if m.fail {
		return errors.New("GitHub unreachable")
	}
	return nil
}
func (m *miniGH) Whoami(context.Context) (string, error) { return "pub-bot", nil }
func (m *miniGH) EnsureLabels(context.Context, string, []ghloop.Label) ([]string, error) {
	return nil, nil
}
func (m *miniGH) CreateIssue(context.Context, string, string, string, []string) (int, error) {
	m.next++
	return m.next, nil
}
func (m *miniGH) EditIssue(context.Context, string, int, ghloop.IssueEdit) error { return nil }
func (m *miniGH) Comment(_ context.Context, _ string, n int, body string) (int64, error) {
	m.cid++
	m.comments[n] = append(m.comments[n], ghloop.Comment{ID: m.cid, Body: body, Author: "pub-bot", CreatedAt: "t", UpdatedAt: "t"})
	return m.cid, nil
}
func (m *miniGH) CloseIssue(context.Context, string, int) error { return nil }

func withGitHub(t *testing.T, p *P) *miniGH {
	t.Helper()
	v, _ := config.ValidateValues(map[string]string{"PILOT_FIXTURE_REPO": "o/x-pilot-fixture", "PILOT_MODEL": "p/m",
		"PILOT_MODEL_EGRESS_HOSTS": "h", "PILOT_GITHUB_APPROVALS": "on", "PILOT_APPROVERS": "alice"})
	p.Cfg = &config.Config{Values: v}
	m := &miniGH{comments: map[int][]ghloop.Comment{}}
	p.GH = &ghloop.Loop{Repo: "o/x-pilot-fixture", Approvers: []string{"alice"}, R: m,
		W: func() (ghloop.Writer, error) { return m, nil },
		Snapshot: func() (map[string]any, map[string]*string, error) { return map[string]any{}, nil, m.err() }}
	return m
}

func TestPauseWithGitHubApprovalsPostsTheDecision(t *testing.T) {
	p, r, out := newP(t)
	m := withGitHub(t, p)
	p.modeNotConfirmed(r, "none")
	if !strings.Contains(out.String(), "  or approve on GitHub: https://github.com/o/x-pilot-fixture/issues/1 (reply `/approve mode`; allowed: alice)") {
		t.Fatalf("%q", out.String())
	}
	if len(m.comments[1]) != 1 || !strings.Contains(m.comments[1][0].Body, "step=mode task=- n=- -->") {
		t.Fatal(m.comments)
	}
	// a valid /approve then becomes the pending approval on resume
	m.comments[1] = append(m.comments[1], ghloop.Comment{ID: 50, Body: "/approve mode", Author: "alice", CreatedAt: "t2", UpdatedAt: "t2"})
	out.Reset()
	p.collectGitHub(r)
	if !strings.Contains(out.String(), "approved by @alice on GitHub: start the run in this mode") {
		t.Fatalf("%q", out.String())
	}
}

func TestGitHubPostFailureIsReportedNotFatal(t *testing.T) {
	p, r, out := newP(t)
	withGitHub(t, p)
	p.GH.W = func() (ghloop.Writer, error) { return nil, errors.New("no credential") }
	if code := p.modeNotConfirmed(r, "none"); code != ExitDeclined {
		t.Fatal(code)
	}
	if !strings.Contains(out.String(), "  (could not post the decision on GitHub: no credential)") {
		t.Fatalf("%q", out.String())
	}
}

func TestGitHubRefreshFailureIsRecorded(t *testing.T) {
	p, r, out := newP(t)
	m := withGitHub(t, p)
	m.fail = true
	p.githubRefresh(r, true)
	if !strings.Contains(out.String(), "warning: github sync failed: GitHub unreachable") {
		t.Fatalf("%q", out.String())
	}
	ev, _ := r.Events()
	if ev[len(ev)-1]["event"] != "github_sync_failed" {
		t.Fatal(ev)
	}
}
