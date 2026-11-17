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

// ReplayMismatch is REPLAY_MISMATCH: the replayed change's tree differs from the graded tree.
type ReplayMismatch struct{ Graded, Replayed string }

func (e *ReplayMismatch) Error() string {
	return fmt.Sprintf("REPLAY_MISMATCH: replayed tree %s != graded tree %s", e.Replayed, e.Graded)
}

// Replay fetches the worker's change bundle (snapshot..HEAD) into clone and recreates each of its
// commits on top of base, keeping authors, dates and messages. Each recreated commit gets the
// original commit's tree, which is valid because the snapshot's tree equals base's tree. It
// writes a base..replayed bundle to out and returns the replayed head.
func Replay(clone, base, snapshot, changeBundle, out string) (string, error) {
	if _, err := gitEnv(clone, nil, "fetch", "-q", changeBundle, "HEAD:refs/pilot/change"); err != nil {
		return "", err
	}
	if t1, _ := gitEnv(clone, nil, "rev-parse", snapshot+"^{tree}"); t1 != mustTree(clone, base) {
		return "", fmt.Errorf("snapshot %s is not a snapshot of %s", snapshot, base)
	}
	revs, err := gitEnv(clone, nil, "rev-list", "--reverse", "--first-parent", snapshot+"..refs/pilot/change")
	if err != nil {
		return "", err
	}
	if revs == "" {
		return "", fmt.Errorf("the change has no commits on top of the snapshot")
	}
	parent := base
	for _, c := range strings.Fields(revs) {
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
		next, err := gitEnv(clone, env, "commit-tree", mustTree(clone, c), "-p", parent, "-m", f[6])
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
