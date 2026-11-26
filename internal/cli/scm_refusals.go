package cli

import "github.com/tosin2013/chalkeus/internal/scm"

// gitHubOnly refuses a GitHub-only command for a repository on another host (037 FR-015,
// contracts/config.md): the GitHub loop, step mode, the GitHub App and the debt report.
func gitHubOnly(command string, h scm.Host) error {
	if h.IsGitHub() {
		return nil
	}
	return Exit(ExitError, "%s is GitHub-only in this release (this repository is on %s); the loop on other hosts is roadmap 038", command, h.Name)
}
