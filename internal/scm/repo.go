package scm

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	githubRepoRE    = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
	gitlabSegRE     = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]*$`)
	bitbucketRepoRE = regexp.MustCompile(`^[a-z0-9_.-]+/[a-z0-9_.-]+$`)
	hostNameRE      = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9-]*[A-Za-z0-9])?(\.[A-Za-z0-9]([A-Za-z0-9-]*[A-Za-z0-9])?)*(:[0-9]{1,5})?$`)
)

// ValidRepo checks a repository path in the host's form (data-model.md "Repo path") and returns
// the form's description for messages.
func ValidRepo(kind Kind, repo string) error {
	switch kind {
	case GitLab:
		segs := strings.Split(repo, "/")
		if len(segs) < 2 || len(segs) > 20 || strings.HasSuffix(repo, ".git") {
			return fmt.Errorf("must be group/project or group/subgroup/…/project")
		}
		for _, s := range segs {
			if !gitlabSegRE.MatchString(s) {
				return fmt.Errorf("must be group/project or group/subgroup/…/project")
			}
		}
		return nil
	case Bitbucket:
		if !bitbucketRepoRE.MatchString(repo) {
			return fmt.Errorf("must be workspace/repo_slug (lowercase)")
		}
		return nil
	}
	if !githubRepoRE.MatchString(repo) {
		return fmt.Errorf("must be owner/name")
	}
	return nil
}

// ValidHostName checks FORGE_SCM_HOST: a bare host name, optionally with a port.
func ValidHostName(s string) bool { return hostNameRE.MatchString(s) }

// LastSegment is a repository path's last segment (the fixture's -pilot-fixture guard).
func LastSegment(repo string) string { return repo[strings.LastIndex(repo, "/")+1:] }

// RepoURL is the HTTPS clone URL of repo on h.
func RepoURL(h Host, repo string) string {
	base := h.GitBase
	if base == "" {
		base = NewHost(h.KindOrGitHub(), h.Name, "").GitBase
	}
	return strings.TrimRight(base, "/") + "/" + repo + ".git"
}

// ChangeRequestURLPrefix is the part of a change request's web URL the audit's AU03 checks.
func ChangeRequestURLPrefix(h Host, repo string) string {
	switch h.KindOrGitHub() {
	case GitLab:
		return hostOnlyName(h) + "/" + repo + "/-/merge_requests/"
	case Bitbucket:
		return BitbucketHost + "/" + repo + "/pull-requests/"
	}
	return "github.com/" + repo + "/pull/"
}

func hostOnlyName(h Host) string {
	if h.Name == "" {
		return GitLabDotCom
	}
	return h.Name
}
