package fakegh

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// gitEnv makes commits deterministic enough and independent of the host's git config.
var gitEnv = []string{"GIT_AUTHOR_NAME=fakegithub", "GIT_AUTHOR_EMAIL=fake@invalid", "GIT_COMMITTER_NAME=fakegithub",
	"GIT_COMMITTER_EMAIL=fake@invalid", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1"}

// git runs git in dir with stdin, returning trimmed stdout.
func git(dir string, stdin []byte, env []string, args ...string) (string, error) {
	c := exec.Command("git", args...)
	c.Dir = dir
	c.Env = append(append(os.Environ(), gitEnv...), env...)
	if stdin != nil {
		c.Stdin = bytes.NewReader(stdin)
	}
	var out, errb bytes.Buffer
	c.Stdout, c.Stderr = &out, &errb
	if err := c.Run(); err != nil {
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(errb.String()))
	}
	return strings.TrimSpace(out.String()), nil
}

// initRepo creates a bare repository whose main branch holds fromDir's files in one commit.
func initRepo(bare, fromDir string) error {
	if _, err := git("", nil, nil, "init", "-q", "--bare", "-b", "main", bare); err != nil {
		return err
	}
	if _, err := git(bare, nil, nil, "config", "http.receivepack", "true"); err != nil {
		return err
	}
	// GitHub serves any commit reachable from a branch or pull request by its id (029: readiness
	// fetches an approved bot commit that way).
	if _, err := git(bare, nil, nil, "config", "uploadpack.allowReachableSHA1InWant", "true"); err != nil {
		return err
	}
	if fromDir == "" {
		return nil
	}
	work, err := os.MkdirTemp("", "fakegh-seed-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(work) }()
	if _, err := git("", nil, nil, "init", "-q", "-b", "main", work); err != nil {
		return err
	}
	if err := copyTree(fromDir, work); err != nil {
		return err
	}
	for _, a := range [][]string{{"add", "-A"}, {"commit", "-q", "-m", "fixture"}, {"push", "-q", bare, "main"}} {
		if _, err := git(work, nil, nil, a...); err != nil {
			return err
		}
	}
	return nil
}

func copyTree(src, dst string) error {
	return filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		if rel == ".git" || strings.HasPrefix(rel, ".git"+string(filepath.Separator)) {
			return filepath.SkipDir
		}
		t := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(t, 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(t, b, info.Mode().Perm())
	})
}

// refSHA is a branch's commit ("" if absent).
func refSHA(bare, branch string) string {
	sha, err := git(bare, nil, nil, "rev-parse", "--verify", "-q", "refs/heads/"+branch)
	if err != nil {
		return ""
	}
	return sha
}

// writeTree builds a tree from base (may be "") plus entries {path, mode, sha|content}; an entry
// with a nil sha and no content deletes the path, as GitHub's API does.
func writeTree(bare, base string, entries []map[string]any) (string, error) {
	idx, err := os.CreateTemp("", "fakegh-index-")
	if err != nil {
		return "", err
	}
	_ = idx.Close()
	_ = os.Remove(idx.Name())
	defer func() { _ = os.Remove(idx.Name()) }()
	env := []string{"GIT_INDEX_FILE=" + idx.Name()}
	if base != "" {
		if _, err := git(bare, nil, env, "read-tree", base); err != nil {
			return "", err
		}
	} else if _, err := git(bare, nil, env, "read-tree", "--empty"); err != nil {
		return "", err
	}
	for _, e := range entries {
		path, _ := e["path"].(string)
		mode, _ := e["mode"].(string)
		if mode == "" {
			mode = "100644"
		}
		sha, _ := e["sha"].(string)
		if c, ok := e["content"].(string); ok {
			if sha, err = git(bare, []byte(c), nil, "hash-object", "-w", "--stdin"); err != nil {
				return "", err
			}
		}
		if sha == "" {
			if _, err := git(bare, nil, env, "update-index", "--force-remove", path); err != nil {
				return "", err
			}
			continue
		}
		if _, err := git(bare, nil, env, "update-index", "--add", "--cacheinfo", mode+","+sha+","+path); err != nil {
			return "", err
		}
	}
	return git(bare, nil, env, "write-tree")
}

// changedFiles lists the files a branch changes against base (merge-base diff), as PR files.
func changedFiles(bare, base, head string) ([]map[string]string, error) {
	out, err := git(bare, nil, nil, "diff", "--name-status", "refs/heads/"+base+"...refs/heads/"+head)
	if err != nil {
		return nil, err
	}
	status := map[string]string{"A": "added", "M": "modified", "D": "removed"}
	var files []map[string]string
	for _, l := range strings.Split(out, "\n") {
		f := strings.Fields(l)
		if len(f) < 2 {
			continue
		}
		st := status[f[0][:1]]
		if st == "" {
			st = "modified"
		}
		files = append(files, map[string]string{"filename": f[len(f)-1], "status": st})
	}
	return files, nil
}

// merge merges head into base with a merge commit by login, and returns its sha.
func merge(bare, base, head, login, message string) (string, error) {
	work, err := os.MkdirTemp("", "fakegh-merge-")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.RemoveAll(work) }()
	env := []string{"GIT_COMMITTER_NAME=" + login, "GIT_AUTHOR_NAME=" + login}
	steps := [][]string{{"clone", "-q", "-b", base, bare, work}}
	for _, a := range steps {
		if _, err := git("", nil, env, a...); err != nil {
			return "", err
		}
	}
	for _, a := range [][]string{{"fetch", "-q", "origin", head}, {"merge", "--no-ff", "-q", "-m", message, "FETCH_HEAD"}, {"push", "-q", "origin", base}} {
		if _, err := git(work, nil, env, a...); err != nil {
			return "", err
		}
	}
	return git(work, nil, nil, "rev-parse", "HEAD")
}

func decodeB64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(strings.ReplaceAll(s, "\n", ""))
}

// pullRef points refs/pull/<n>/head at a branch's commit, as GitHub keeps for every PR (the
// pilot fetches it to check a status-change PR). A missing branch leaves the ref as it was.
func pullRef(bare string, n int, branch string) {
	if sha := refSHA(bare, branch); sha != "" {
		_, _ = git(bare, nil, nil, "update-ref", fmt.Sprintf("refs/pull/%d/head", n), sha)
	}
}
