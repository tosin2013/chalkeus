package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/tosin2013/chalkeus/internal/scm"
)

// 037: the Git host of a target or of the fixture (contracts/config.md). An empty FORGE_SCM is
// GitHub, and every GitHub-only rule stays as it was.

// scmSyntax checks FORGE_SCM, FORGE_SCM_HOST and FORGE_SCM_CA_BUNDLE without touching files. It
// returns the kind and, on a problem, the offending key and why.
func scmSyntax(v map[string]string) (scm.Kind, string, string) {
	kind, err := scm.ParseKind(v["PILOT_SCM"])
	if err != nil {
		return "", "PILOT_SCM", err.Error()
	}
	host, ca := strings.TrimSpace(v["PILOT_SCM_HOST"]), strings.TrimSpace(v["PILOT_SCM_CA_BUNDLE"])
	if kind != scm.GitLab {
		if host != "" {
			return kind, "PILOT_SCM_HOST", "is for self-managed GitLab only (FORGE_SCM=gitlab)"
		}
		if ca != "" {
			return kind, "PILOT_SCM_CA_BUNDLE", "is for self-managed GitLab only (FORGE_SCM=gitlab)"
		}
	}
	if host != "" && !scm.ValidHostName(host) {
		return kind, "PILOT_SCM_HOST", "must be a bare host name such as gitlab.example.com (no scheme or path; a port as host:port)"
	}
	if ca != "" && (strings.ContainsAny(ca, `/\`) || ca == "." || ca == "..") {
		return kind, "PILOT_SCM_CA_BUNDLE", "must be a file name in the config folder, such as gitlab-ca.pem"
	}
	if kind != scm.GitHub {
		for _, k := range []string{"PILOT_GITHUB_ISSUES", "PILOT_GITHUB_APPROVALS"} {
			if v[k] == "on" {
				return kind, k, "is GitHub-only in this release; the loop on other hosts is roadmap 038"
			}
		}
		if v["PILOT_PUBLISHER_CREDENTIAL"] == "app" {
			return kind, "PILOT_PUBLISHER_CREDENTIAL", "=app is GitHub-only in this release; the loop on other hosts is roadmap 038"
		}
	}
	return kind, "", ""
}

// caBundlePath resolves and checks FORGE_SCM_CA_BUNDLE in the config folder: mode 0600 and a PEM
// certificate. "" when unset.
func caBundlePath(configDir, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", nil
	}
	p := filepath.Join(configDir, name)
	if _, err := os.Stat(p); err != nil {
		return "", errf(2, "FORGE_SCM_CA_BUNDLE: %s doesn't exist (put the PEM bundle in the config folder)", p)
	}
	if err := CheckMode(p); err != nil {
		return "", errf(2, "FORGE_SCM_CA_BUNDLE: %s", err.Error())
	}
	if _, err := scm.LoadCABundle(p); err != nil {
		return "", errf(2, "%s", err.Error())
	}
	return p, nil
}

// FixtureHost is the fixture's Git host from forge.env.
func (c *Config) FixtureHost() scm.Host {
	kind, _, _ := scmSyntax(c.Values)
	ca := ""
	if n := strings.TrimSpace(c.Values["PILOT_SCM_CA_BUNDLE"]); n != "" {
		ca = filepath.Join(c.ConfigDir, n)
	}
	return scm.NewHost(kind, strings.TrimSpace(c.Values["PILOT_SCM_HOST"]), ca)
}

// Host is the target's Git host.
func (t Target) Host() scm.Host {
	kind := t.SCM
	if kind == "" {
		kind = scm.GitHub
	}
	return scm.NewHost(kind, t.SCMHost, t.SCMCABundle)
}

// RepoGuardFor is RepoGuard for a fixture on any host: the path must be valid for the host and
// its last segment must end in -pilot-fixture.
func RepoGuardFor(kind scm.Kind, repo, origin string) error {
	if kind == "" || kind == scm.GitHub {
		return RepoGuard(repo, origin)
	}
	if err := scm.ValidRepo(kind, repo); err != nil {
		return errf(3, "FORGE_FIXTURE_REPO %s, got %q", err.Error(), repo)
	}
	if !strings.HasSuffix(scm.LastSegment(repo), "-pilot-fixture") {
		return errf(3, "repo guard: %s does not end in '-pilot-fixture'", repo)
	}
	return nil
}
