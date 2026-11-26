package fixture

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Snapshot and replay (014 R4): a real repository reaches the pods as one commit whose tree is
// the base revision's tree, so the bundle carries no history. The worker commits on top of it;
// before publishing, the pilot replays those commits onto the real base revision. The trees are
// identical, so the graded tree is exactly what is published.

// snapshotEnv fixes author, committer and dates, so a snapshot of the same tree is the same
// commit every time.
var snapshotEnv = []string{
	"GIT_AUTHOR_NAME=pilot-snapshot", "GIT_AUTHOR_EMAIL=pilot-snapshot@invalid", "GIT_AUTHOR_DATE=2000-01-01T00:00:00Z",
	"GIT_COMMITTER_NAME=pilot-snapshot", "GIT_COMMITTER_EMAIL=pilot-snapshot@invalid", "GIT_COMMITTER_DATE=2000-01-01T00:00:00Z",
}

func gitEnv(dir string, env []string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %s", strings.Join(args[:min(2, len(args))], " "), strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// Snapshot creates, in clone, a parentless commit with base's tree, points branch
// pilot-snapshot at it, and writes a bundle of that branch to out. It returns the snapshot
// commit, which is the base revision the worker and grader see.
func Snapshot(clone, base, out string) (string, error) {
	tree, err := gitEnv(clone, nil, "rev-parse", base+"^{tree}")
	if err != nil {
		return "", err
	}
	snap, err := gitEnv(clone, snapshotEnv, "commit-tree", tree, "-m", "pilot snapshot of "+base)
	if err != nil {
		return "", err
	}
	if _, err := gitEnv(clone, nil, "update-ref", "refs/heads/pilot-snapshot", snap); err != nil {
		return "", err
	}
	// HEAD names the snapshot, so a clone of the bundle checks it out without a warning. The
	// clone's worktree already holds base's tree, which is the snapshot's.
	if _, err := gitEnv(clone, nil, "symbolic-ref", "HEAD", "refs/heads/pilot-snapshot"); err != nil {
		return "", err
	}
	if _, err := gitEnv(clone, nil, "bundle", "create", "-q", out, "HEAD", "pilot-snapshot"); err != nil {
		return "", err
	}
	return snap, nil
}

// Manifest is a dependency item's approved manifest change (029, research R10): the listed
// files as they are at Commit, which is on the target (a branch or a pull request).
type Manifest struct {
	Commit string
	Files  []string
}

// fetchCommit makes commit available in clone, fetching it from origin when it's missing.
func fetchCommit(clone, commit string) error {
	if _, err := gitEnv(clone, nil, "cat-file", "-e", commit+"^{commit}"); err == nil {
		return nil
	}
	if _, err := gitEnv(clone, nil, "fetch", "-q", "origin", commit); err != nil {
		return fmt.Errorf("manifest change %s: %v", short(commit), err)
	}
	return nil
}

func short(c string) string {
	if len(c) > 7 {
		return c[:7]
	}
	return c
}

// manifestTree is base's tree with m's files replaced by their blobs at m.Commit (a file absent
// there is removed). It uses a temporary index, so the clone's index and worktree are untouched.
func manifestTree(clone, base string, m *Manifest) (string, error) {
	idx, err := os.CreateTemp("", "pilot-manifest-index-")
	if err != nil {
		return "", err
	}
	_ = idx.Close()
	_ = os.Remove(idx.Name()) // git creates it
	defer func() { _ = os.Remove(idx.Name()) }()
	env := []string{"GIT_INDEX_FILE=" + idx.Name()}
	if _, err := gitEnv(clone, env, "read-tree", base); err != nil {
		return "", err
	}
	for _, f := range m.Files {
		entry, err := gitEnv(clone, nil, "ls-tree", m.Commit, "--", f)
		if err != nil {
			return "", err
		}
		if entry == "" {
			if _, err := gitEnv(clone, env, "update-index", "--force-remove", "--", f); err != nil {
				return "", err
			}
			continue
		}
		meta, _, _ := strings.Cut(entry, "\t") // "<mode> blob <id>\t<path>"
		fields := strings.Fields(meta)
		if len(fields) != 3 || fields[1] != "blob" {
			return "", fmt.Errorf("manifest change %s: %s is not a file there", short(m.Commit), f)
		}
		if _, err := gitEnv(clone, env, "update-index", "--add", "--cacheinfo", fields[0]+","+fields[2]+","+f); err != nil {
			return "", err
		}
	}
	return gitEnv(clone, env, "write-tree")
}

// SnapshotWith is Snapshot for a dependency item: the snapshot's tree is base's tree with the
// approved manifest change applied, so the worker, the baseline and the prefetch all start from
// it. m nil is Snapshot.
func SnapshotWith(clone, base, out string, m *Manifest) (string, error) {
	if m == nil {
		return Snapshot(clone, base, out)
	}
	if err := fetchCommit(clone, m.Commit); err != nil {
		return "", err
	}
	tree, err := manifestTree(clone, base, m)
	if err != nil {
		return "", err
	}
	snap, err := gitEnv(clone, snapshotEnv, "commit-tree", tree, "-m", "pilot snapshot of "+base+" with manifest change "+m.Commit)
	if err != nil {
		return "", err
	}
	if _, err := gitEnv(clone, nil, "update-ref", "refs/heads/pilot-snapshot", snap); err != nil {
		return "", err
	}
	if _, err := gitEnv(clone, nil, "symbolic-ref", "HEAD", "refs/heads/pilot-snapshot"); err != nil {
		return "", err
	}
	if _, err := gitEnv(clone, nil, "bundle", "create", "-q", out, "HEAD", "pilot-snapshot"); err != nil {
		return "", err
	}
	return snap, nil
}

// manifestCommit commits the approved change on base: the snapshot's tree, the approved commit's
// author and subject, and a body naming the commit (research R10). The owner reviews it apart
// from the worker's fixes.
func manifestCommit(clone, base, snapshot string, m *Manifest, trailers []string) (string, error) {
	tree, err := manifestTree(clone, base, m)
	if err != nil {
		return "", err
	}
	if snapTree := mustTree(clone, snapshot); tree != snapTree {
		return "", fmt.Errorf("snapshot %s is not %s with manifest change %s", short(snapshot), short(base), short(m.Commit))
	}
	meta, err := gitEnv(clone, nil, "show", "-s", "--format=%an%x00%ae%x00%aI%x00%cn%x00%ce%x00%cI%x00%s", m.Commit)
	if err != nil {
		return "", err
	}
	f := strings.SplitN(meta, "\x00", 7)
	if len(f) != 7 {
		return "", fmt.Errorf("unreadable commit %s", m.Commit)
	}
	env := []string{"GIT_AUTHOR_NAME=" + f[0], "GIT_AUTHOR_EMAIL=" + f[1], "GIT_AUTHOR_DATE=" + f[2],
		"GIT_COMMITTER_NAME=" + f[3], "GIT_COMMITTER_EMAIL=" + f[4], "GIT_COMMITTER_DATE=" + f[5]}
	msg := f[6] + "\n\nThe approved manifest change: " + strings.Join(m.Files, ", ") + " as in " + m.Commit +
		", applied to " + short(base) + ".\n"
	if len(trailers) > 0 {
		msg += "\n" + strings.Join(trailers, "\n") + "\n"
	}
	return gitEnv(clone, env, "commit-tree", tree, "-p", base, "-m", msg)
}

// ReplayMismatch is REPLAY_MISMATCH: the replayed change's tree differs from the graded tree.
type ReplayMismatch struct{ Graded, Replayed string }

func (e *ReplayMismatch) Error() string {
	return fmt.Sprintf("REPLAY_MISMATCH: replayed tree %s != graded tree %s", e.Replayed, e.Graded)
}

// Replay fetches the worker's change bundle (snapshot..HEAD) into clone and recreates each of its
// commits on top of base, keeping authors, dates and messages. Each recreated commit gets the
// original commit's tree, which is valid because the snapshot's tree equals base's tree. It
// writes a base..replayed bundle to out and returns the replayed head.
//
// subject, when set, becomes the first line of the last commit's message, with the worker's
// message and trailers kept below it: GitHub uses a one-commit PR's commit message, not its
// title, for a squash merge, so the commit must carry the PR's Conventional Commits title too.
func Replay(clone, base, snapshot, changeBundle, out string, trailers ...string) (string, error) {
	return ReplayAs(clone, base, snapshot, changeBundle, out, "", trailers...)
}

// ReplayAs is Replay with the last commit's subject replaced (see Replay).
func ReplayAs(clone, base, snapshot, changeBundle, out, subject string, trailers ...string) (string, error) {
	return ReplayWith(clone, base, snapshot, changeBundle, out, subject, nil, trailers...)
}

// ReplayWith is ReplayAs for a dependency item (m not nil, 029): it first commits the approved
// manifest change on base (checking its tree is the snapshot's), then replays the worker's commits
// on top, so the publish bundle holds the approved change byte for byte, then the fixes.
func ReplayWith(clone, base, snapshot, changeBundle, out, subject string, m *Manifest, trailers ...string) (string, error) {
	if _, err := gitEnv(clone, nil, "fetch", "-q", changeBundle, "HEAD:refs/pilot/change"); err != nil {
		return "", err
	}
	parent := base
	if m != nil {
		if err := fetchCommit(clone, m.Commit); err != nil {
			return "", err
		}
		mc, err := manifestCommit(clone, base, snapshot, m, trailers)
		if err != nil {
			return "", err
		}
		parent = mc
	} else if t1, _ := gitEnv(clone, nil, "rev-parse", snapshot+"^{tree}"); t1 != mustTree(clone, base) {
		return "", fmt.Errorf("snapshot %s is not a snapshot of %s", snapshot, base)
	}
	revs, err := gitEnv(clone, nil, "rev-list", "--reverse", "--first-parent", snapshot+"..refs/pilot/change")
	if err != nil {
		return "", err
	}
	if revs == "" {
		return "", fmt.Errorf("the change has no commits on top of the snapshot")
	}
	commits := strings.Fields(revs)
	for i, c := range commits {
		meta, err := gitEnv(clone, nil, "show", "-s", "--format=%an%x00%ae%x00%aI%x00%cn%x00%ce%x00%cI%x00%B", c)
		if err != nil {
			return "", err
		}
		f := strings.SplitN(meta, "\x00", 7)
		if len(f) != 7 {
			return "", fmt.Errorf("unreadable commit %s", c)
		}
		env := []string{"GIT_AUTHOR_NAME=" + f[0], "GIT_AUTHOR_EMAIL=" + f[1], "GIT_AUTHOR_DATE=" + f[2],
			"GIT_COMMITTER_NAME=" + f[3], "GIT_COMMITTER_EMAIL=" + f[4], "GIT_COMMITTER_DATE=" + f[5]}
		msg := f[6]
		if i == len(commits)-1 && subject != "" {
			msg = subject + "\n\n" + strings.TrimSpace(msg) + "\n"
			if len(trailers) > 0 {
				msg += "\n" + strings.Join(trailers, "\n") + "\n"
			}
		}
		next, err := gitEnv(clone, env, "commit-tree", mustTree(clone, c), "-p", parent, "-m", msg)
		if err != nil {
			return "", err
		}
		parent = next
	}
	if graded, replayed := mustTree(clone, "refs/pilot/change"), mustTree(clone, parent); graded != replayed {
		return "", &ReplayMismatch{Graded: graded, Replayed: replayed}
	}
	if _, err := gitEnv(clone, nil, "update-ref", "refs/heads/pilot-replayed", parent); err != nil {
		return "", err
	}
	// The publisher and reconcile fetch the bundle's HEAD, as for a worker's change bundle.
	if _, err := gitEnv(clone, nil, "symbolic-ref", "HEAD", "refs/heads/pilot-replayed"); err != nil {
		return "", err
	}
	if _, err := gitEnv(clone, nil, "bundle", "create", "-q", out, base+"..HEAD"); err != nil {
		return "", err
	}
	return parent, nil
}

func mustTree(dir, rev string) string {
	t, _ := gitEnv(dir, nil, "rev-parse", rev+"^{tree}")
	return t
}
