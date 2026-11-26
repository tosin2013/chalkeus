package fixture

import (
	"context"
	"github.com/tosin2013/chalkeus/internal/scm"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func bare(t *testing.T, base, repo string) {
	t.Helper()
	d := filepath.Join(base, repo+".git")
	if out, err := exec.Command("git", "init", "--quiet", "--bare", d).CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
}

func TestResetAndFreshClone(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	t.Setenv("GIT_AUTHOR_NAME", "t")
	t.Setenv("GIT_AUTHOR_EMAIL", "t@e")
	t.Setenv("GIT_COMMITTER_NAME", "t")
	t.Setenv("GIT_COMMITTER_EMAIL", "t@e")
	remote := t.TempDir()
	bare(t, remote, "o/x-pilot-fixture")
	old := BaseURL
	BaseURL = "file://" + remote + "/"
	defer func() { BaseURL = old }()

	var out strings.Builder
	var closed []int
	base := filepath.Join(t.TempDir(), "fixture-base.txt")
	opts := ResetOptions{Repo: "o/x-pilot-fixture", Branch: "main", Template: "../../pilot/fixture", BaseFile: base,
		In: strings.NewReader("n\n"), Out: &out,
		ListPRs: func(context.Context) ([]PRRef, error) { return []PRRef{{Number: 7, HeadRef: "pilot/r/a"}}, nil },
		ClosePR: func(_ context.Context, p PRRef) error { closed = append(closed, p.Number); return nil }}
	if code, err := Reset(context.Background(), opts); code != 5 || err != nil || len(closed) != 0 {
		t.Fatalf("decline: %d %v %v", code, err, closed)
	}
	opts.Yes = true
	code, err := Reset(context.Background(), opts)
	if code != 0 || err != nil {
		t.Fatalf("%d %v\n%s", code, err, out.String())
	}
	if len(closed) != 1 || !strings.Contains(out.String(), "closed PR #7 (pilot/r/a)") || !strings.Contains(out.String(), "fixture reset: o/x-pilot-fixture@main = ") {
		t.Fatalf("output:\n%s", out.String())
	}
	raw, _ := os.ReadFile(base)
	dir, head, err := FreshClone(scm.Host{}, "o/x-pilot-fixture", "main", 1)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	if strings.TrimSpace(string(raw)) != head {
		t.Fatalf("base %q != head %q", raw, head)
	}
	if _, err := os.Stat(filepath.Join(dir, ".specify/memory/roadmap.md")); err != nil {
		t.Fatal("fixture content missing in the clone")
	}
}
