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
