package readiness

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/tosin2013/chalkeus/internal/debt"
	"github.com/tosin2013/chalkeus/internal/images"
)

// 029: dependency items. An item whose spec.md has a `## Manifest change` section builds on an
// approved change to its dependency manifests (specs/029-brownfield-intake/contracts/item-spec.md,
// research R9). Readiness checks the change against the target before any cluster work.

// ManifestChange is the approved change: one commit's diff against its first parent, limited to
// the listed files.
type ManifestChange struct {
	Commit  string   `json:"commit"`
	Parent  string   `json:"parent,omitempty"`
	Files   []string `json:"files"`
	Subject string   `json:"subject,omitempty"`
	// Set when every rule passed (data-model.md ManifestChange): each file's blob at the parent
	// and at the commit, and the packages the change bumps, read from the diff.
	Changes  []FileChange   `json:"changes,omitempty"`
	Packages []debt.Package `json:"packages,omitempty"`
}

// FileChange is one listed file's blob before (at the parent) and after (at the commit); "" when
// the file doesn't exist there.
type FileChange struct {
	Path   string `json:"path"`
	Before string `json:"before"`
	After  string `json:"after"`
}

var (
	manifestHeading = regexp.MustCompile(`(?m)^##\s+Manifest change\s*$`)
	nextHeading     = regexp.MustCompile(`(?m)^##\s`)
	commitLine      = regexp.MustCompile(`(?im)^\s*[-*]\s*Commit:\s*(.*)$`)
	filesLine       = regexp.MustCompile(`(?im)^\s*[-*]\s*Files:\s*(.*)$`)
	backticked      = regexp.MustCompile("`([^`]+)`")
	hex40           = regexp.MustCompile(`^[0-9a-f]{40}$`)
)

// ParseManifestChange reads the `## Manifest change` section. present is false when the spec has
// none (an ordinary item). A section without exactly one 40-hex commit, or without a file, or with
// a glob or an absolute path, is an error (MANIFEST_CHANGE_UNREADABLE).
func ParseManifestChange(spec string) (mc *ManifestChange, present bool, err error) {
	loc := manifestHeading.FindStringIndex(spec)
	if loc == nil {
		return nil, false, nil
	}
	body := spec[loc[1]:]
	if n := nextHeading.FindStringIndex(body); n != nil {
		body = body[:n[0]]
	}
	var commits []string
	for _, m := range commitLine.FindAllStringSubmatch(body, -1) {
		for _, b := range backticked.FindAllStringSubmatch(m[1], -1) {
			commits = append(commits, strings.TrimSpace(b[1]))
		}
	}
	if len(commits) != 1 || !hex40.MatchString(commits[0]) {
		return nil, true, fmt.Errorf("the section needs exactly one `Commit:` with a full 40-character commit id in backticks (found %d)", len(commits))
	}
	var files []string
	seen := map[string]bool{}
	for _, m := range filesLine.FindAllStringSubmatch(body, -1) {
		for _, b := range backticked.FindAllStringSubmatch(m[1], -1) {
			f := strings.TrimPrefix(strings.TrimSpace(b[1]), "./")
			if f == "" || strings.HasPrefix(f, "/") || strings.ContainsAny(f, "*?[") || strings.Contains(f, "..") {
				return nil, true, fmt.Errorf("file %q must be a path relative to the repository root, without globs", b[1])
			}
			if !seen[f] {
				seen[f] = true
				files = append(files, f)
			}
		}
	}
	if len(files) == 0 {
		return nil, true, fmt.Errorf("the section needs a `Files:` line with at least one backticked path")
	}
	return &ManifestChange{Commit: commits[0], Files: files}, true, nil
}

// manifestRefusal is one failed manifest-change rule.
type manifestRefusal struct{ Code, Detail string }

// CheckManifestChange evaluates the change against dir, a clone of the target at its base (HEAD),
// whose origin is the target. It fetches the commit and its parent (depth 2). It returns every
// failed rule, in contract order; none means the change can be applied to this base.
func CheckManifestChange(dir, toolchain, branch string, mc *ManifestChange, sliced bool) []manifestRefusal {
	var out []manifestRefusal
	short := mc.Commit[:7]
	if sliced {
		out = append(out, manifestRefusal{"MANIFEST_CHANGE_SLICED", "a dependency item is built whole: run it without --slice"})
	}
	var notManifest []string
	for _, f := range mc.Files {
		if !images.IsManifest(toolchain, f) {
			notManifest = append(notManifest, f)
		}
	}
	if len(notManifest) > 0 {
		out = append(out, manifestRefusal{"MANIFEST_CHANGE_NOT_MANIFEST", fmt.Sprintf("%s %s not a dependency manifest of toolchain %s (manifests: %s); list only manifest files",
			strings.Join(notManifest, ", "), isAre(len(notManifest)), toolchain, strings.Join(images.Manifests(toolchain), ", "))})
	}
	if _, err := git(dir, "fetch", "--quiet", "--depth", "2", "origin", mc.Commit); err != nil {
		if _, err2 := git(dir, "cat-file", "-e", mc.Commit+"^{commit}"); err2 != nil {
			return append(out, manifestRefusal{"MANIFEST_CHANGE_UNRESOLVED", fmt.Sprintf("commit %s can't be fetched from the target (it must be on a branch or a pull request of the repository): %v", short, err)})
		}
	}
	parent, err := git(dir, "rev-parse", mc.Commit+"^1")
	if err != nil {
		return append(out, manifestRefusal{"MANIFEST_CHANGE_UNRESOLVED", fmt.Sprintf("commit %s has no readable first parent", short)})
	}
	mc.Parent = parent
	mc.Subject, _ = git(dir, "show", "-s", "--format=%s", mc.Commit)
	diff, err := git(dir, "diff", "--name-only", "--no-renames", parent, mc.Commit)
	if err != nil {
		return append(out, manifestRefusal{"MANIFEST_CHANGE_UNRESOLVED", fmt.Sprintf("the diff of commit %s can't be read: %v", short, err)})
	}
	listed := map[string]bool{}
	for _, f := range mc.Files {
		listed[f] = true
	}
	var unlisted []string
	for _, f := range strings.Fields(diff) {
		if !listed[f] {
			unlisted = append(unlisted, f)
		}
	}
	if len(unlisted) > 0 {
		out = append(out, manifestRefusal{"MANIFEST_CHANGE_NOT_MANIFEST", fmt.Sprintf("commit %s also changes %s, which %s not listed: the approved change must touch only the listed manifest files",
			short, strings.Join(unlisted, ", "), isAre(len(unlisted)))})
	}
	var stale, same []string
	for _, f := range mc.Files {
		atBase, atParent, atCommit := blob(dir, "HEAD", f), blob(dir, parent, f), blob(dir, mc.Commit, f)
		if atBase != atParent {
			stale = append(stale, f)
		}
		if atParent == atCommit {
			same = append(same, f)
		}
	}
	if len(stale) > 0 {
		out = append(out, manifestRefusal{"MANIFEST_CHANGE_STALE", fmt.Sprintf("%s changed on %s since %s's parent: rebase the bot PR (or commit the change again on the current %s) and update the commit in spec.md",
			strings.Join(stale, ", "), branch, short, branch)})
	}
	if len(same) == len(mc.Files) {
		out = append(out, manifestRefusal{"MANIFEST_CHANGE_EMPTY", fmt.Sprintf("commit %s doesn't change %s: name the commit that bumps the dependencies", short, strings.Join(mc.Files, ", "))})
	}
	if len(out) == 0 {
		for _, f := range mc.Files {
			before, after := blob(dir, parent, f), blob(dir, mc.Commit, f)
			mc.Changes = append(mc.Changes, FileChange{Path: f, Before: before, After: after})
			mc.Packages = append(mc.Packages, debt.Changes(f, show(dir, parent, f), show(dir, mc.Commit, f))...)
		}
	}
	return out
}

// show is path's content at rev, or nil when it doesn't exist there.
func show(dir, rev, path string) []byte {
	cmd := exec.Command("git", "show", rev+":"+path)
	cmd.Dir = dir
	b, err := cmd.Output()
	if err != nil {
		return nil
	}
	return b
}

// blob is path's blob id at rev, or "" when the file doesn't exist there.
func blob(dir, rev, path string) string {
	b, err := git(dir, "rev-parse", "--verify", "--quiet", rev+":"+path)
	if err != nil {
		return ""
	}
	return b
}

func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %s", args[0], strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func isAre(n int) string {
	if n == 1 {
		return "is"
	}
	return "are"
}
