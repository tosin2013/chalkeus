package scmtest

import (
	"context"
	"sort"
	"strings"
	"testing"

	"github.com/tosin2013/chalkeus/internal/scm"
)

// Merge kinds a Fake can perform.
const (
	MergeCommit = "merge"
	Squash      = "squash"
	FastForward = "ff"
)

// Fake is a host test double the suite drives as the repository owner would (contracts/
// host-interface.md, research R13). Implementations live next to each client's tests.
type Fake interface {
	// CreateRepo makes repo with branch main holding files.
	CreateRepo(t *testing.T, repo string, files map[string]string)
	// PushBranch commits files on a new branch off main and returns the commit.
	PushBranch(t *testing.T, repo, branch string, files map[string]string) string
	// OpenRequest opens a request from head into main as author.
	OpenRequest(t *testing.T, repo, head string, draft bool, author scm.Account) int
	// MergeAsOwner merges a request the given way as by (the owner, never forge) and returns
	// the commit that landed on main.
	MergeAsOwner(t *testing.T, repo string, id int, how string, by scm.Account) string
	// Decline closes a request without merging.
	Decline(t *testing.T, repo string, id int)
	// SetPermission gives a an effective permission on repo.
	SetPermission(t *testing.T, repo string, a scm.Account, p scm.Permission)
	// Protect protects branch (on Bitbucket: with a restrict_merges restriction).
	Protect(t *testing.T, repo, branch string)
	// MergeKinds lists the merge kinds the host supports.
	MergeKinds() []string
	// Accounts are two accounts the fake knows: an owner (the client's own credential) and another.
	Accounts() (owner, other scm.Account)
}

// Run runs the contract suite against a fresh client and fake per case.
func Run(t *testing.T, newClient func(t *testing.T) (scm.Client, Fake)) {
	ctx := context.Background()
	t.Run("OpenLookupIncludesDrafts", func(t *testing.T) {
		c, f := newClient(t)
		owner, _ := f.Accounts()
		f.CreateRepo(t, "ws/repo", map[string]string{"README.md": "hi\n"})
		f.PushBranch(t, "ws/repo", "pilot/001-a1", map[string]string{"a.txt": "a\n"})
		f.PushBranch(t, "ws/repo", "other/x", map[string]string{"b.txt": "b\n"})
		d := f.OpenRequest(t, "ws/repo", "pilot/001-a1", true, owner)
		f.OpenRequest(t, "ws/repo", "other/x", false, owner)
		got, err := c.OpenChangeRequests(ctx, "ws/repo", "pilot/")
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].ID != d || !got[0].Draft || got[0].HeadRef != "pilot/001-a1" || got[0].State != scm.StateOpen {
			t.Fatalf("%+v", got)
		}
	})
	t.Run("StatesAndLandedCommit", func(t *testing.T) {
		for _, how := range f0(newClient, t).MergeKinds() {
			t.Run(how, func(t *testing.T) {
				c, f := newClient(t)
				owner, other := f.Accounts()
				f.CreateRepo(t, "ws/repo", map[string]string{"README.md": "hi\n"})
				f.PushBranch(t, "ws/repo", "pilot/x", map[string]string{"a.txt": "a\n"})
				id := f.OpenRequest(t, "ws/repo", "pilot/x", true, owner)
				landed := f.MergeAsOwner(t, "ws/repo", id, how, other)
				cr, err := c.ChangeRequest(ctx, "ws/repo", id)
				if err != nil {
					t.Fatal(err)
				}
				if !cr.Merged() || cr.LandedCommit != landed || !cr.MergedBy.Same(other) || !cr.Author.Same(owner) {
					t.Fatalf("landed %s: %+v", landed, cr)
				}
				files, err := c.ChangedFiles(ctx, "ws/repo", cr)
				if err != nil || len(files) != 1 || files[0] != "a.txt" {
					t.Fatalf("changed files %v %v", files, err)
				}
			})
		}
	})
	t.Run("DeclinedIsClosed", func(t *testing.T) {
		c, f := newClient(t)
		owner, _ := f.Accounts()
		f.CreateRepo(t, "ws/repo", map[string]string{"README.md": "hi\n"})
		f.PushBranch(t, "ws/repo", "pilot/x", map[string]string{"a.txt": "a\n"})
		id := f.OpenRequest(t, "ws/repo", "pilot/x", false, owner)
		f.Decline(t, "ws/repo", id)
		cr, err := c.ChangeRequest(ctx, "ws/repo", id)
		if err != nil || cr.State != scm.StateClosed || cr.Merged() || cr.MergedBy.ID != "" {
			t.Fatalf("%+v %v", cr, err)
		}
	})
	t.Run("FileAt", func(t *testing.T) {
		c, f := newClient(t)
		f.CreateRepo(t, "ws/repo", map[string]string{"docs/roadmap.md": "v1\n"})
		sha := f.PushBranch(t, "ws/repo", "pilot/x", map[string]string{"docs/roadmap.md": "v2\n"})
		got, err := c.FileAt(ctx, "ws/repo", sha, "docs/roadmap.md")
		if err != nil || string(got) != "v2\n" {
			t.Fatalf("%q %v", got, err)
		}
		if _, err := c.FileAt(ctx, "ws/repo", sha, "missing.md"); !scm.IsCode(err, scm.ErrNotFound) {
			t.Fatalf("missing file: %v", err)
		}
	})
	t.Run("PermissionAndProtection", func(t *testing.T) {
		c, f := newClient(t)
		_, other := f.Accounts()
		f.CreateRepo(t, "ws/repo", map[string]string{"README.md": "hi\n"})
		f.SetPermission(t, "ws/repo", other, scm.PermWrite)
		p, err := c.Permission(ctx, "ws/repo", other)
		if err != nil || !p.CanWrite() {
			t.Fatalf("%v %v", p, err)
		}
		pr, err := c.BranchProtection(ctx, "ws/repo", "main")
		if err != nil || pr.Protected {
			t.Fatalf("unprotected: %+v %v", pr, err)
		}
		f.Protect(t, "ws/repo", "main")
		if pr, err := c.BranchProtection(ctx, "ws/repo", "main"); err != nil || !pr.Protected {
			t.Fatalf("protected: %+v %v", pr, err)
		}
		info, err := c.Repository(ctx, "ws/repo")
		if err != nil || info.DefaultBranch != "main" {
			t.Fatalf("%+v %v", info, err)
		}
	})
	t.Run("CommitFilesAndOpenRequest", func(t *testing.T) {
		c, f := newClient(t)
		f.CreateRepo(t, "ws/repo", map[string]string{"README.md": "hi\n"})
		sha, err := c.CommitFiles(ctx, "ws/repo", "main", "author/006-x", []scm.File{
			{Path: "tests/a_test.py", Content: []byte("a\n")}, {Path: "items/006/map.json", Content: []byte("{}\n")}}, "test: hidden tests for 006")
		if err != nil || sha == "" {
			t.Fatalf("%q %v", sha, err)
		}
		cr, err := c.OpenRequest(ctx, "ws/repo", "author/006-x", "main", "test: 006", "body", true)
		if err != nil || !cr.Draft || cr.HeadRef != "author/006-x" || cr.State != scm.StateOpen {
			t.Fatalf("%+v %v", cr, err)
		}
		files, err := c.ChangedFiles(ctx, "ws/repo", cr)
		sort.Strings(files)
		if err != nil || strings.Join(files, ",") != "items/006/map.json,tests/a_test.py" {
			t.Fatalf("%v %v", files, err)
		}
		if got, _ := c.FileAt(ctx, "ws/repo", sha, "tests/a_test.py"); string(got) != "a\n" {
			t.Fatalf("committed content %q", got)
		}
		sha2, err := c.CommitFiles(ctx, "ws/repo", "main", "author/006-y", []scm.File{
			{Path: "tests/b_test.py", Content: []byte("b\n")}, {Path: "README.md", Delete: true}}, "test: replace")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := c.FileAt(ctx, "ws/repo", sha2, "README.md"); !scm.IsCode(err, scm.ErrNotFound) {
			t.Fatalf("deleted file still there: %v", err)
		}
	})
	t.Run("CloseChangeRequest", func(t *testing.T) {
		c, f := newClient(t)
		owner, _ := f.Accounts()
		f.CreateRepo(t, "ws/repo", map[string]string{"README.md": "hi\n"})
		f.PushBranch(t, "ws/repo", "pilot/x", map[string]string{"a.txt": "a\n"})
		id := f.OpenRequest(t, "ws/repo", "pilot/x", true, owner)
		if err := c.CloseChangeRequest(ctx, "ws/repo", id, "reset"); err != nil {
			t.Fatal(err)
		}
		if cr, _ := c.ChangeRequest(ctx, "ws/repo", id); cr.State != scm.StateClosed {
			t.Fatalf("%+v", cr)
		}
		if open, _ := c.OpenChangeRequests(ctx, "ws/repo", "pilot/"); len(open) != 0 {
			t.Fatalf("still open: %+v", open)
		}
	})
	t.Run("NotFound", func(t *testing.T) {
		c, _ := newClient(t)
		if _, err := c.ChangeRequest(ctx, "ws/none", 1); !scm.IsCode(err, scm.ErrNotFound) {
			t.Fatalf("%v", err)
		}
	})
}

// f0 reads the merge kinds from a throwaway fake.
func f0(newClient func(t *testing.T) (scm.Client, Fake), t *testing.T) Fake {
	_, f := newClient(t)
	return f
}
