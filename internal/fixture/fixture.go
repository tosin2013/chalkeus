// Package fixture handles the throwaway fixture repo (reference: the former Python pilot's fixture.py).
//
// Host-side git uses the operator's own `gh` login through a one-off credential helper
// (`-c credential.helper=!gh auth git-credential`); no git config is changed and the publisher
// token is never used here.
package fixture

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var ghCred = []string{"-c", "credential.helper=", "-c", "credential.helper=!gh auth git-credential"}

// Git runs git with the one-off credential helper. On failure the error carries git's message.
func Git(dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append(append([]string{}, ghCred...), args...)...)
	cmd.Dir = dir
	var stdout, stderr strings.Builder
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		if len(msg) > 500 {
			msg = msg[:500]
		}
		head := args
		if len(head) > 2 {
			head = head[:2]
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(head, " "), msg)
	}
	return stdout.String(), nil
}

// BaseURL is where fixture repos live (tests point it at local bare repositories).
var BaseURL = "https://github.com/"

// URL is the clone URL of owner/name.
func URL(repo string) string { return BaseURL + repo + ".git" }

// FreshClone clones the branch into a new temp dir (depth 0 = full history). It returns the
// directory and HEAD's sha; the caller removes the directory.
func FreshClone(repo, branch string, depth int) (string, string, error) {
	d, err := os.MkdirTemp("", "pilot-fixture-")
	if err != nil {
		return "", "", err
	}
	args := []string{"clone", "--quiet"}
	if branch != "" { // "" = the remote's default branch (a companion repository, 014)
		args = append(args, "--branch", branch)
	}
	if depth > 0 {
		args = append(args, "--depth", fmt.Sprint(depth))
	}
	if _, err := Git("", append(args, URL(repo), d)...); err != nil {
		_ = os.RemoveAll(d)
		return "", "", err
	}
	head, err := Git(d, "rev-parse", "HEAD")
	if err != nil {
		_ = os.RemoveAll(d)
		return "", "", err
	}
	return d, strings.TrimSpace(head), nil
}

// Reset force-resets the fixture repo to pilot/fixture/, closes open pilot/* PRs, deletes stale
// pilot/* branches and records the new base (the reference fixture.reset). It returns the exit
// code: 0 done, 5 declined.
func Reset(ctx context.Context, opts ResetOptions) (int, error) {
	out := opts.Out
	fmt.Fprintf(out, "This force-resets %s@%s to pilot/fixture/ and closes open pilot/* PRs.\n", opts.Repo, opts.Branch)
	if !opts.Yes {
		fmt.Fprint(out, "Type y to continue: ")
		line, _ := bufio.NewReader(opts.In).ReadString('\n')
		if strings.ToLower(strings.TrimSpace(line)) != "y" {
			fmt.Fprintln(out, "declined")
			return 5, nil
		}
	}
	prs, err := opts.ListPRs(ctx)
	if err != nil {
		return 2, err
	}
	for _, pr := range prs {
		if err := opts.ClosePR(ctx, pr); err != nil {
			return 2, err
		}
		fmt.Fprintf(out, "closed PR #%d (%s)\n", pr.Number, pr.HeadRef)
	}
	t, err := os.MkdirTemp("", "pilot-reset-")
	if err != nil {
		return 2, err
	}
	defer func() { _ = os.RemoveAll(t) }()
	if err := copyTree(opts.Template, t); err != nil {
		return 2, err
	}
	for _, args := range [][]string{
		{"init", "--quiet", "-b", opts.Branch}, {"add", "-A"},
		{"commit", "--quiet", "-m", "pilot: reset fixture"}, // the operator's own git identity, as in the reference
		{"push", "--quiet", "--force", URL(opts.Repo), "HEAD:refs/heads/" + opts.Branch},
	} {
		if _, err := Git(t, args...); err != nil {
			return 2, err
		}
	}
	base, err := Git(t, "rev-parse", "HEAD")
	if err != nil {
		return 2, err
	}
	base = strings.TrimSpace(base)
	if heads, err := Git("", "ls-remote", "--heads", URL(opts.Repo), "refs/heads/pilot/*"); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(heads), "\n") {
			if f := strings.Fields(line); len(f) == 2 {
				_, _ = Git("", "push", "--quiet", URL(opts.Repo), ":"+f[1])
			}
		}
	}
	if err := os.MkdirAll(filepath.Dir(opts.BaseFile), 0o755); err != nil {
		return 2, err
	}
	if err := os.WriteFile(opts.BaseFile, []byte(base+"\n"), 0o644); err != nil {
		return 2, err
	}
	fmt.Fprintf(out, "fixture reset: %s@%s = %s\n", opts.Repo, opts.Branch, base)
	return 0, nil
}

// ResetOptions are Reset's inputs; ListPRs and ClosePR are injected so tests need no network.
type ResetOptions struct {
	Repo, Branch, Template, BaseFile string
	Yes                              bool
	In                               io.Reader
	Out                              io.Writer
	ListPRs                          func(context.Context) ([]PRRef, error)
	ClosePR                          func(context.Context, PRRef) error
}

// PRRef identifies an open pilot PR.
type PRRef struct {
	Number  int
	HeadRef string
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, raw, info.Mode().Perm())
	})
}
