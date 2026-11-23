package fakegh_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-github/v85/github"

	"github.com/tosin2013/chalkeus/internal/fakegh"
	"github.com/tosin2013/chalkeus/internal/ghloop"
)

func post(t *testing.T, base, path string, body map[string]any) map[string]any {
	t.Helper()
	b, _ := json.Marshal(body)
	resp, err := http.Post(base+path, "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if resp.StatusCode != 200 {
		t.Fatalf("%s: %d %v", path, resp.StatusCode, out)
	}
	return out
}

func gitRun(t *testing.T, dir string, env []string, args ...string) error {
	t.Helper()
	c := exec.Command("git", args...)
	c.Dir = dir
	c.Env = append(os.Environ(), append([]string{"GIT_TERMINAL_PROMPT=0", "GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@invalid", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@invalid"}, env...)...)
	out, err := c.CombinedOutput()
	if err != nil {
		t.Logf("git %v: %s", args, out)
	}
	return err
}

// Seed through /_test/, clone over HTTP, push the way the publisher does (GIT_ASKPASS with
// x-access-token), open a PR over REST, merge it as the owner, and clone again.
func TestGitOverHTTPAndTheOwnerAPI(t *testing.T) {
	srv := httptest.NewServer(fakegh.NewMux(fakegh.New(t.TempDir())))
	defer srv.Close()
	old := ghloop.API
	ghloop.API = srv.URL
	defer func() { ghloop.API = old }()
	seed := t.TempDir()
	_ = os.WriteFile(filepath.Join(seed, "README.md"), []byte("hello\n"), 0o644)
	post(t, srv.URL, "/_test/users", map[string]any{"token": "bot-token", "login": "pilot-bot"})
	post(t, srv.URL, "/_test/users", map[string]any{"token": "owner-token", "login": "owner"})
	post(t, srv.URL, "/_test/repos", map[string]any{"repo": "me/target", "from": seed, "perms": map[string]string{"pilot-bot": "write", "owner": "admin"}})

	url := srv.URL + "/git/me/target.git"
	work := filepath.Join(t.TempDir(), "w")
	if err := gitRun(t, "", nil, "clone", "-q", url, work); err != nil {
		t.Fatal("anonymous clone")
	}
	_ = os.WriteFile(filepath.Join(work, "feature.txt"), []byte("new\n"), 0o644)
	if err := gitRun(t, work, nil, "add", "-A"); err != nil {
		t.Fatal(err)
	}
	_ = gitRun(t, work, nil, "commit", "-q", "-m", "feature")
	if err := gitRun(t, work, nil, "push", "-q", "origin", "HEAD:refs/heads/pilot/r1/010"); err == nil {
		t.Fatal("a push without credentials must be refused")
	}
	askpass := filepath.Join(t.TempDir(), "askpass.sh")
	_ = os.WriteFile(askpass, []byte("#!/bin/sh\ncase \"$1\" in Username*) echo x-access-token;; *) printf %s bot-token;; esac\n"), 0o700)
	if err := gitRun(t, work, []string{"GIT_ASKPASS=" + askpass}, "push", "-q", "origin", "HEAD:refs/heads/pilot/r1/010"); err != nil {
		t.Fatal("the publisher's push")
	}

	ctx := context.Background()
	gh := ghloop.NewClient(secretOf("bot-token"))
	pr, _, err := gh.PullRequests.Create(ctx, "me", "target", &github.NewPullRequest{Title: github.Ptr("feat(010)"),
		Head: github.Ptr("pilot/r1/010"), Base: github.Ptr("main"), Draft: github.Ptr(true)})
	if err != nil {
		t.Fatal(err)
	}
	// GitHub's pull/<n>/head ref, which the pilot fetches for a status-change PR.
	if err := gitRun(t, work, nil, "fetch", "-q", "origin", fmt.Sprintf("pull/%d/head", pr.GetNumber())); err != nil {
		t.Fatal("pull/<n>/head is fetchable")
	}
	post(t, srv.URL, "/_test/comment", map[string]any{"repo": "me/target", "issue": pr.GetNumber(), "login": "owner", "body": "/approve mode"})
	post(t, srv.URL, "/_test/merge", map[string]any{"repo": "me/target", "pr": pr.GetNumber(), "login": "owner"})

	again := filepath.Join(t.TempDir(), "again")
	if err := gitRun(t, "", nil, "clone", "-q", url, again); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(filepath.Join(again, "feature.txt")); err != nil || string(b) != "new\n" {
		t.Fatalf("the merged file is on main: %q %v", b, err)
	}
	resp, _ := http.Get(srv.URL + "/_test/state")
	var st map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&st)
	_ = resp.Body.Close()
	s, _ := json.Marshal(st)
	if !strings.Contains(string(s), `"merged":true`) || !strings.Contains(string(s), "owner: /approve mode") {
		t.Fatalf("%s", s)
	}
}

// The stand-in model answers doctor's one-token check only with a bearer key.
func TestStandInModel(t *testing.T) {
	srv := httptest.NewServer(fakegh.NewMux(fakegh.New(t.TempDir())))
	defer srv.Close()
	for _, c := range []struct {
		auth string
		want int
	}{{"Bearer k", 200}, {"", 401}} {
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/model/v1/chat/completions", strings.NewReader(`{}`))
		if c.auth != "" {
			req.Header.Set("Authorization", c.auth)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != c.want {
			t.Errorf("auth %q: %d, want %d", c.auth, resp.StatusCode, c.want)
		}
	}
}
