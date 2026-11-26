package fixture

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func sh(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=a", "GIT_AUTHOR_EMAIL=a@invalid", "GIT_COMMITTER_NAME=a", "GIT_COMMITTER_EMAIL=a@invalid")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %s", args, out)
	}
	return strings.TrimSpace(string(out))
}

func write(t *testing.T, p, body string) {
	t.Helper()
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A target with history: the snapshot carries none of it, and the replay lands on the real base.
func TestSnapshotThenReplay(t *testing.T) {
	repo := t.TempDir()
	sh(t, repo, "init", "-q", "-b", "main")
	write(t, filepath.Join(repo, "a.txt"), "one\n")
	sh(t, repo, "add", "-A")
	sh(t, repo, "commit", "-qm", "first")
	write(t, filepath.Join(repo, "src/b.txt"), "two\n")
	sh(t, repo, "add", "-A")
	sh(t, repo, "commit", "-qm", "second")
	base := sh(t, repo, "rev-parse", "HEAD")

	bundle := filepath.Join(t.TempDir(), "base.bundle")
	snap, err := Snapshot(repo, base, bundle)
	if err != nil {
		t.Fatal(err)
	}
	again, _ := Snapshot(repo, base, filepath.Join(t.TempDir(), "again.bundle"))
	if snap == base || again != snap {
		t.Fatalf("snapshot %s base %s again %s", snap, base, again)
	}

	// The worker: clone the snapshot bundle, check out the snapshot, commit twice.
	w := filepath.Join(t.TempDir(), "w")
	sh(t, "", "clone", "-q", bundle, w)
	sh(t, w, "checkout", "-q", snap)
	if n := sh(t, w, "rev-list", "--count", "HEAD"); n != "1" {
		t.Fatalf("snapshot carries history: %s commits", n)
	}
	write(t, filepath.Join(w, "src/b.txt"), "two, changed\n")
	sh(t, w, "commit", "-qam", "pilot: change 1")
	write(t, filepath.Join(w, "src/c.txt"), "new\n")
	sh(t, w, "add", "-A")
	sh(t, w, "commit", "-qm", "pilot: change 2")
	graded := sh(t, w, "rev-parse", "HEAD^{tree}")
	change := filepath.Join(t.TempDir(), "change.bundle")
	sh(t, w, "bundle", "create", "-q", change, snap+"..HEAD")

	out := filepath.Join(t.TempDir(), "replayed.bundle")
	head, err := Replay(repo, base, snap, change, out)
	if err != nil {
		t.Fatal(err)
	}
	if got := sh(t, repo, "rev-parse", head+"^{tree}"); got != graded {
		t.Fatalf("replayed tree %s != graded %s", got, graded)
	}
	if p := sh(t, repo, "rev-parse", head+"~2"); p != base {
		t.Fatalf("replay is not on the real base: %s", p)
	}
	if msg := sh(t, repo, "log", "-1", "--format=%s", head); msg != "pilot: change 2" {
		t.Fatal(msg)
	}
	// An item run's commit carries the PR's Conventional Commits title (squash merges use it).
	head2, err := ReplayAs(repo, base, snap, change, filepath.Join(t.TempDir(), "r2.bundle"), "fix(006): bundled flags", "Chalkeus-Run: R1")
	if err != nil {
		t.Fatal(err)
	}
	if msg := sh(t, repo, "log", "-1", "--format=%B", head2); !strings.HasPrefix(msg, "fix(006): bundled flags\n\npilot: change 2") || !strings.Contains(msg, "Chalkeus-Run: R1") {
		t.Fatalf("%q", msg)
	}
	if m := sh(t, repo, "log", "-1", "--format=%s", head2+"~1"); m != "pilot: change 1" {
		t.Fatalf("only the last commit is retitled: %q", m)
	}
	if got := sh(t, repo, "rev-parse", head2+"^{tree}"); got != graded {
		t.Fatal("the tree must not change")
	}
	// The publisher's view: a fresh clone at base accepts the base..replayed bundle.
	p := filepath.Join(t.TempDir(), "p")
	sh(t, "", "clone", "-q", repo, p)
	sh(t, p, "bundle", "verify", out)
	sh(t, p, "fetch", "-q", out, "HEAD:refs/pilot/published")
	if got := sh(t, p, "rev-parse", "refs/pilot/published"); got != head {
		t.Fatalf("the bundle's HEAD is %s, want %s", got, head)
	}
}

func TestReplayRefusesForeignSnapshotAndEmptyChange(t *testing.T) {
	repo := t.TempDir()
	sh(t, repo, "init", "-q", "-b", "main")
	write(t, filepath.Join(repo, "a.txt"), "one\n")
	sh(t, repo, "add", "-A")
	sh(t, repo, "commit", "-qm", "first")
	base := sh(t, repo, "rev-parse", "HEAD")
	snap, err := Snapshot(repo, base, filepath.Join(t.TempDir(), "b.bundle"))
	if err != nil {
		t.Fatal(err)
	}
	// an empty change: the worker committed nothing
	b := filepath.Join(t.TempDir(), "w.bundle")
	sh(t, repo, "bundle", "create", "-q", b, "pilot-snapshot")
	w := filepath.Join(t.TempDir(), "w")
	sh(t, "", "clone", "-q", b, w)
	sh(t, w, "checkout", "-q", snap)
	empty := filepath.Join(t.TempDir(), "empty.bundle")
	sh(t, w, "bundle", "create", "-q", empty, "HEAD")
	if _, err := Replay(repo, base, snap, empty, filepath.Join(t.TempDir(), "o")); err == nil || !strings.Contains(err.Error(), "no commits") {
		t.Fatalf("empty change: %v", err)
	}
	// a snapshot of another tree is refused
	write(t, filepath.Join(repo, "a.txt"), "two\n")
	sh(t, repo, "commit", "-qam", "second")
	if _, err := Replay(repo, sh(t, repo, "rev-parse", "HEAD"), snap, empty, filepath.Join(t.TempDir(), "o")); err == nil || !strings.Contains(err.Error(), "not a snapshot") {
		t.Fatalf("foreign snapshot: %v", err)
	}
}

// 029: a dependency item's snapshot carries the approved manifest change, and the replay puts
// that change on the base as its own commit, then the worker's fix.
func TestSnapshotAndReplayWithManifest(t *testing.T) {
	origin := t.TempDir()
	sh(t, origin, "init", "-q", "-b", "main")
	write(t, filepath.Join(origin, "package.json"), `{"zod":"3.25.0"}`+"\n")
	write(t, filepath.Join(origin, "package-lock.json"), `{"zod":"3.25.0"}`+"\n")
	write(t, filepath.Join(origin, "src/index.js"), "old\n")
	sh(t, origin, "add", "-A")
	sh(t, origin, "commit", "-qm", "base")
	base := sh(t, origin, "rev-parse", "HEAD")
	sh(t, origin, "checkout", "-q", "-b", "dependabot/npm")
	write(t, filepath.Join(origin, "package.json"), `{"zod":"4.1.0"}`+"\n")
	write(t, filepath.Join(origin, "package-lock.json"), `{"zod":"4.1.0"}`+"\n")
	sh(t, origin, "add", "-A")
	cmd := exec.Command("git", "commit", "-qm", "chore(deps): bump zod from 3.25.0 to 4.1.0")
	cmd.Dir = origin
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=dependabot[bot]", "GIT_AUTHOR_EMAIL=bot@invalid", "GIT_COMMITTER_NAME=GitHub", "GIT_COMMITTER_EMAIL=gh@invalid")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s", out)
	}
	bump := sh(t, origin, "rev-parse", "HEAD")
	sh(t, origin, "checkout", "-q", "main")
	clone := filepath.Join(t.TempDir(), "clone") // the run's clone of main: it lacks the bump
	sh(t, "", "clone", "-q", origin, clone)
	m := &Manifest{Commit: bump, Files: []string{"package.json", "package-lock.json"}}

	bundle := filepath.Join(t.TempDir(), "base.bundle")
	snap, err := SnapshotWith(clone, base, bundle, m)
	if err != nil {
		t.Fatal(err)
	}
	if plain, _ := Snapshot(clone, base, filepath.Join(t.TempDir(), "plain.bundle")); plain == snap {
		t.Fatal("the snapshot must differ from a plain one")
	}
	_, _ = SnapshotWith(clone, base, bundle, m) // Snapshot above moved the ref; make it ours again
	w := filepath.Join(t.TempDir(), "w")
	sh(t, "", "clone", "-q", bundle, w)
	sh(t, w, "checkout", "-q", snap)
	if got, _ := os.ReadFile(filepath.Join(w, "package.json")); string(got) != `{"zod":"4.1.0"}`+"\n" {
		t.Fatalf("the worker must see the approved manifest: %s", got)
	}
	write(t, filepath.Join(w, "src/index.js"), "fixed for zod 4\n")
	sh(t, w, "commit", "-qam", "fix: zod 4")
	change := filepath.Join(t.TempDir(), "change.bundle")
	sh(t, w, "bundle", "create", "-q", change, snap+"..HEAD")

	out := filepath.Join(t.TempDir(), "publish.bundle")
	head, err := ReplayWith(clone, base, snap, change, out, "fix(010): zod 4", m, "Chalkeus-Run: R1")
	if err != nil {
		t.Fatal(err)
	}
	revs := strings.Fields(sh(t, clone, "rev-list", "--reverse", base+".."+head))
	if len(revs) != 2 {
		t.Fatalf("want the manifest commit then the fix, got %d commits", len(revs))
	}
	for _, f := range m.Files {
		if a, b := sh(t, clone, "rev-parse", revs[0]+":"+f), sh(t, clone, "rev-parse", bump+":"+f); a != b {
			t.Errorf("%s in the manifest commit differs from the approved commit's", f)
		}
	}
	if got := sh(t, clone, "show", "-s", "--format=%an|%s", revs[0]); got != "dependabot[bot]|chore(deps): bump zod from 3.25.0 to 4.1.0" {
		t.Errorf("manifest commit author and subject: %q", got)
	}
	body := sh(t, clone, "show", "-s", "--format=%B", revs[0])
	if !strings.Contains(body, bump) || !strings.Contains(body, "Chalkeus-Run: R1") {
		t.Errorf("manifest commit body: %q", body)
	}
	if got := sh(t, clone, "show", "-s", "--format=%s", revs[1]); got != "fix(010): zod 4" {
		t.Errorf("fix subject: %q", got)
	}
	if mustTree(clone, head) != mustTree(clone, "refs/pilot/change") {
		t.Error("the published tree must be the graded tree")
	}
	if p := sh(t, clone, "rev-parse", revs[0]+"^"); p != base {
		t.Errorf("the manifest commit's parent is %s, want the base", p)
	}

	// A plain snapshot can't be replayed as a dependency item.
	plain, _ := Snapshot(clone, base, filepath.Join(t.TempDir(), "p.bundle"))
	if _, err := ReplayWith(clone, base, plain, change, filepath.Join(t.TempDir(), "x.bundle"), "", m); err == nil || !strings.Contains(err.Error(), "with manifest change") {
		t.Fatalf("mismatched snapshot: %v", err)
	}
}
