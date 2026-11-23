package fakegh_test

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-github/v85/github"

	"github.com/tosin2013/chalkeus/internal/fakegh"
	"github.com/tosin2013/chalkeus/internal/ghloop"
	"github.com/tosin2013/chalkeus/internal/secret"
)

// world is a fake GitHub with a target and a companion, an owner, a publisher bot and a stranger.
func world(t *testing.T) (*fakegh.Server, string) {
	t.Helper()
	s := fakegh.New(t.TempDir())
	s.AddUser("owner-token", "owner")
	s.AddUser("bot-token", "pilot-bot")
	s.AddUser("stranger-token", "stranger")
	seed := t.TempDir()
	_ = os.MkdirAll(filepath.Join(seed, "specs"), 0o755)
	_ = os.WriteFile(filepath.Join(seed, "README.md"), []byte("stockroom\n"), 0o644)
	if err := s.AddRepo("me/target", seed, map[string]string{"owner": "admin", "pilot-bot": "write", "stranger": "read"}); err != nil {
		t.Fatal(err)
	}
	if err := s.AddRepo("me/companion", "", map[string]string{"owner": "admin"}); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(httpHandler(s))
	t.Cleanup(srv.Close)
	old := ghloop.API
	ghloop.API = srv.URL
	t.Cleanup(func() { ghloop.API = old })
	return s, srv.URL
}

func httpHandler(s *fakegh.Server) *fakegh.Mux { return fakegh.NewMux(s) }

func client(tok string) *github.Client { return ghloop.NewClient(secret.New(tok)) }

// The pilot's own token probe gives the answers it expects from GitHub.
func TestProbeAgainstTheFake(t *testing.T) {
	world(t)
	res := ghloop.Probe(ghloop.Credential{Token: secret.New("bot-token")}, "me/target", "tosin2013/specfoundry", true)
	if !res.OK || *res.Login != "pilot-bot" || !res.RepoPush || !*res.IssuesWrite || *res.CanaryPush {
		t.Fatalf("%+v %v", res, res.Problems)
	}
	res = ghloop.Probe(ghloop.Credential{Token: secret.New("stranger-token")}, "me/target", "", false)
	if res.OK || res.RepoPush {
		t.Fatalf("a read-only token can't push: %+v", res)
	}
}

// A branch made through the git data API (as the test author's submit does), a PR, its files,
// listing by head, and the PR found by the pilot's own helpers.
func TestGitDataAndPulls(t *testing.T) {
	world(t)
	ctx := context.Background()
	gh := client("bot-token")
	ref, _, err := gh.Git.GetRef(ctx, "me", "target", "refs/heads/main")
	if err != nil {
		t.Fatal(err)
	}
	base, _, err := gh.Git.GetCommit(ctx, "me", "target", ref.GetObject().GetSHA())
	if err != nil {
		t.Fatal(err)
	}
	blob, _, err := gh.Git.CreateBlob(ctx, "me", "target", github.Blob{Content: github.Ptr("hello"), Encoding: github.Ptr("utf-8")})
	if err != nil {
		t.Fatal(err)
	}
	tree, _, err := gh.Git.CreateTree(ctx, "me", "target", base.GetTree().GetSHA(),
		[]*github.TreeEntry{{Path: github.Ptr("specs/010-x/spec.md"), Mode: github.Ptr("100644"), Type: github.Ptr("blob"), SHA: blob.SHA}})
	if err != nil {
		t.Fatal(err)
	}
	commit, _, err := gh.Git.CreateCommit(ctx, "me", "target", github.Commit{Message: github.Ptr("plan 010"), Tree: tree,
		Parents: []*github.Commit{{SHA: base.SHA}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := gh.Git.CreateRef(ctx, "me", "target", github.CreateRef{Ref: "refs/heads/plan-010", SHA: commit.GetSHA()}); err != nil {
		t.Fatal(err)
	}
	pr, _, err := gh.PullRequests.Create(ctx, "me", "target", &github.NewPullRequest{Title: github.Ptr("docs(010): plan"),
		Head: github.Ptr("plan-010"), Base: github.Ptr("main"), Draft: github.Ptr(true)})
	if err != nil || pr.GetNumber() != 1 || !pr.GetDraft() || pr.GetHead().GetSHA() != commit.GetSHA() {
		t.Fatalf("%+v %v", pr, err)
	}
	files, _, err := gh.PullRequests.ListFiles(ctx, "me", "target", 1, nil)
	if err != nil || len(files) != 1 || files[0].GetFilename() != "specs/010-x/spec.md" || files[0].GetStatus() != "added" {
		t.Fatalf("%+v %v", files, err)
	}
	open, _, err := gh.PullRequests.List(ctx, "me", "target", &github.PullRequestListOptions{Head: "me:plan-010", State: "open"})
	if err != nil || len(open) != 1 {
		t.Fatalf("%v %v", open, err)
	}
	planning, err := ghloop.PlanningPRs(ctx, gh, "me/target")
	if err != nil || len(planning) != 1 || !strings.Contains(planning[0], "/me/target/pull/1") {
		t.Fatalf("%v %v", planning, err)
	}
	// A read-only user can read but not write; an unknown repository is a 404.
	if _, _, err := client("stranger-token").PullRequests.Create(ctx, "me", "target", &github.NewPullRequest{Title: github.Ptr("x"),
		Head: github.Ptr("plan-010"), Base: github.Ptr("main")}); err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("read-only: %v", err)
	}
	if _, _, err := gh.Repositories.Get(ctx, "me", "nope"); err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("unknown repo: %v", err)
	}
}

// Issues, labels with their events, and comments: what sync, approvals and step mode read.
func TestIssuesLabelsEventsComments(t *testing.T) {
	s, _ := world(t)
	ctx := context.Background()
	bot := client("bot-token")
	iss, _, err := bot.Issues.Create(ctx, "me", "target", &github.IssueRequest{Title: github.Ptr("010 — Update stock"),
		Body: github.Ptr("<!-- specfoundry:item 010 -->"), Labels: &[]string{"status:specced"}})
	if err != nil || iss.GetNumber() != 1 {
		t.Fatalf("%v %v", iss, err)
	}
	if err := s.Label("me/target", 1, "owner", "pilot/go", false); err != nil {
		t.Fatal(err)
	}
	p, err := ghloop.ProposedBy(ctx, bot, "me/target", 1, "pilot/go")
	if err != nil || !p.Labelled || p.AddedBy != "owner" {
		t.Fatalf("%+v %v", p, err)
	}
	if _, _, err := bot.Issues.CreateComment(ctx, "me", "target", 1, &github.IssueComment{Body: github.Ptr("needs tests")}); err != nil {
		t.Fatal(err)
	}
	cs, _, err := bot.Issues.ListComments(ctx, "me", "target", 1, nil)
	if err != nil || len(cs) != 1 || cs[0].GetUser().GetLogin() != "pilot-bot" {
		t.Fatalf("%v %v", cs, err)
	}
	lab, _, err := bot.Issues.GetLabel(ctx, "me", "target", "status:specced")
	if err != nil || lab.GetName() != "status:specced" {
		t.Fatalf("%v %v", lab, err)
	}
	all, _, err := bot.Issues.ListByRepo(ctx, "me", "target", &github.IssueListByRepoOptions{State: "all"})
	if err != nil || len(all) != 1 || len(all[0].Labels) != 2 {
		t.Fatalf("%v %v", all, err)
	}
}

// The owner's merge: a real merge commit on the base, the PR closed and marked merged.
func TestMerge(t *testing.T) {
	s, _ := world(t)
	ctx := context.Background()
	gh := client("bot-token")
	ref, _, _ := gh.Git.GetRef(ctx, "me", "target", "refs/heads/main")
	base, _, _ := gh.Git.GetCommit(ctx, "me", "target", ref.GetObject().GetSHA())
	tree, _, _ := gh.Git.CreateTree(ctx, "me", "target", base.GetTree().GetSHA(),
		[]*github.TreeEntry{{Path: github.Ptr("a.txt"), Mode: github.Ptr("100644"), Type: github.Ptr("blob"), Content: github.Ptr("a")}})
	commit, _, _ := gh.Git.CreateCommit(ctx, "me", "target", github.Commit{Message: github.Ptr("a"), Tree: tree, Parents: []*github.Commit{{SHA: base.SHA}}}, nil)
	_, _, _ = gh.Git.CreateRef(ctx, "me", "target", github.CreateRef{Ref: "refs/heads/feat", SHA: commit.GetSHA()})
	_, _, _ = gh.PullRequests.Create(ctx, "me", "target", &github.NewPullRequest{Title: github.Ptr("feat"), Head: github.Ptr("feat"), Base: github.Ptr("main")})
	sha, err := s.Merge("me/target", 1, "owner")
	if err != nil {
		t.Fatal(err)
	}
	pr, _, _ := gh.PullRequests.Get(ctx, "me", "target", 1)
	if !pr.GetMerged() || pr.GetMergedBy().GetLogin() != "owner" || pr.GetState() != "closed" || pr.GetMergeCommitSHA() != sha {
		t.Fatalf("%+v", pr)
	}
	tr, _, err := gh.Git.GetTree(ctx, "me", "target", sha, false)
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, e := range tr.Entries {
		paths = append(paths, e.GetPath())
	}
	if !strings.Contains(strings.Join(paths, ","), "a.txt") {
		t.Fatalf("the merge brings the branch's files to main: %v", paths)
	}
	if _, err := s.Merge("me/target", 1, "owner"); err == nil {
		t.Fatal("a merged PR can't be merged again")
	}
}

func secretOf(s string) secret.Value { return secret.New(s) }
