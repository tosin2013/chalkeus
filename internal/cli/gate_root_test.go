package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// gate --target --item --root checks a copy of the checkout's committed HEAD; the checkout
// itself is left alone (readiness deletes its clone).
func TestLocalClone(t *testing.T) {
	root := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		c := exec.Command("git", append([]string{"-C", root, "-c", "user.name=t", "-c", "user.email=t@invalid"}, args...)...)
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	run("init", "-q")
	_ = os.WriteFile(filepath.Join(root, "committed.txt"), []byte("x"), 0o644)
	run("add", "-A")
	run("commit", "-q", "-m", "c")
	_ = os.WriteFile(filepath.Join(root, "uncommitted.txt"), []byte("y"), 0o644)
	d, commit, err := LocalClone(root)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(d)
	if len(commit) != 40 {
		t.Fatalf("commit %q", commit)
	}
	if _, err := os.Stat(filepath.Join(d, "committed.txt")); err != nil {
		t.Fatal("the committed file must be in the copy")
	}
	if _, err := os.Stat(filepath.Join(d, "uncommitted.txt")); err == nil {
		t.Fatal("only the committed HEAD is checked")
	}
	_ = os.RemoveAll(d)
	if _, err := os.Stat(filepath.Join(root, "committed.txt")); err != nil {
		t.Fatal("removing the copy must leave the checkout alone")
	}
	if _, _, err := LocalClone(t.TempDir()); err == nil {
		t.Fatal("a directory that isn't a git checkout must be an error")
	}
}
