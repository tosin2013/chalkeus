// Package fixture handles the throwaway fixture repo (reference: pilot/specfoundry_pilot/fixture.py).
//
// Host-side git uses the operator's own `gh` login through a one-off credential helper
// (`-c credential.helper=!gh auth git-credential`); no git config is changed and the publisher
// token is never used here.
package fixture

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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

// URL is the HTTPS clone URL of owner/name.
func URL(repo string) string { return "https://github.com/" + repo + ".git" }

// FreshClone clones the branch into a new temp dir (depth 0 = full history). It returns the
// directory and HEAD's sha; the caller removes the directory.
func FreshClone(repo, branch string, depth int) (string, string, error) {
	d, err := os.MkdirTemp("", "pilot-fixture-")
	if err != nil {
		return "", "", err
	}
	args := []string{"clone", "--quiet", "--branch", branch}
	if depth > 0 {
		args = append(args, "--depth", fmt.Sprint(depth))
	}
	if _, err := Git("", append(args, URL(repo), d)...); err != nil {
		os.RemoveAll(d)
		return "", "", err
	}
	head, err := Git(d, "rev-parse", "HEAD")
	if err != nil {
		os.RemoveAll(d)
		return "", "", err
	}
	return d, strings.TrimSpace(head), nil
}
