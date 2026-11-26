package config

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tosin2013/chalkeus/internal/scm"
)

const gitlabTarget = `FORGE_SCM=gitlab
FORGE_TARGET_REPO=group/sub/greeting
FORGE_TOOLCHAIN=jdk21-maven
FORGE_TEST_COMMAND=mvn -B test
FORGE_COMPANION_REPO=group/greeting-acceptance
`

func writePEM(t *testing.T, p string) {
	t.Helper()
	k, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "test CA"}, NotBefore: time.Now(),
		NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &k.PublicKey, k)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestTargetHosts(t *testing.T) {
	c := targetCfg(t)
	writeTarget(t, c, "gl", gitlabTarget)
	got, err := c.LoadTarget("gl", "")
	if err != nil {
		t.Fatal(err)
	}
	if got.GithubApprovals != "off" || got.GithubIssues != "off" {
		t.Fatalf("a GitLab target must not inherit forge.env's GitHub approvals: %+v", got)
	}
	if got.SCM != scm.GitLab || got.Host().Name != "gitlab.com" || got.Host().APIBase != "https://gitlab.com/api/v4" {
		t.Fatalf("%+v %+v", got, got.Host())
	}

	writePEM(t, filepath.Join(c.ConfigDir, "corp-ca.pem"))
	writeTarget(t, c, "sm", gitlabTarget+"FORGE_SCM_HOST=git.corp:8443\nFORGE_SCM_CA_BUNDLE=corp-ca.pem\n")
	sm, err := c.LoadTarget("sm", "")
	if err != nil {
		t.Fatal(err)
	}
	if h := sm.Host(); h.Name != "git.corp:8443" || h.CABundle != filepath.Join(c.ConfigDir, "corp-ca.pem") {
		t.Fatalf("%+v", h)
	}

	writeTarget(t, c, "bb", strings.NewReplacer("gitlab", "bitbucket", "group/sub/greeting", "ws/greeting", "group/greeting-acceptance", "ws/greeting-acceptance").Replace(gitlabTarget))
	bb, err := c.LoadTarget("bb", "")
	if err != nil || bb.Host().Kind != scm.Bitbucket || bb.Host().APIBase != "https://api.bitbucket.org/2.0" {
		t.Fatalf("%v %+v", err, bb)
	}

	writeTarget(t, c, "gh", goodTarget)
	gh, _ := c.LoadTarget("gh", "")
	if gh.SCM != "" || !gh.Host().IsGitHub() {
		t.Fatalf("an existing profile must load as GitHub with SCM unset: %+v", gh)
	}
}

func TestTargetHostRefusals(t *testing.T) {
	c := targetCfg(t)
	writePEM(t, filepath.Join(c.ConfigDir, "ok.pem"))
	if err := os.WriteFile(filepath.Join(c.ConfigDir, "open.pem"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(c.ConfigDir, "notpem.pem"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	gh := strings.Replace(gitlabTarget, "FORGE_SCM=gitlab\n", "", 1)
	for name, tc := range map[string]struct{ body, want string }{
		"unknown kind":        {strings.Replace(gitlabTarget, "=gitlab", "=gitea", 1), "FORGE_SCM must be github, gitlab or bitbucket"},
		"host on github":      {goodTarget + "FORGE_SCM_HOST=git.corp\n", "FORGE_SCM_HOST is for self-managed GitLab only"},
		"ca on bitbucket":     {strings.Replace(gitlabTarget, "=gitlab", "=bitbucket", 1) + "FORGE_SCM_CA_BUNDLE=ok.pem\n", "FORGE_SCM_CA_BUNDLE is for self-managed GitLab only"},
		"host with scheme":    {gitlabTarget + "FORGE_SCM_HOST=https://git.corp\n", "FORGE_SCM_HOST must be a bare host name"},
		"ca outside folder":   {gitlabTarget + "FORGE_SCM_CA_BUNDLE=../ca.pem\n", "must be a file name in the config folder"},
		"ca missing":          {gitlabTarget + "FORGE_SCM_CA_BUNDLE=none.pem\n", "doesn't exist"},
		"ca mode":             {gitlabTarget + "FORGE_SCM_CA_BUNDLE=open.pem\n", "0600"},
		"ca not pem":          {gitlabTarget + "FORGE_SCM_CA_BUNDLE=notpem.pem\n", "holds no PEM certificate"},
		"github repo form":    {gh, "FORGE_TARGET_REPO must be owner/name"},
		"bitbucket uppercase": {strings.NewReplacer("gitlab", "bitbucket", "group/sub/greeting", "WS/greeting").Replace(gitlabTarget), "FORGE_TARGET_REPO must be workspace/repo_slug"},
		"issues on gitlab":    {gitlabTarget + "FORGE_GITHUB_ISSUES=on\n", "FORGE_GITHUB_ISSUES is GitHub-only in this release; the loop on other hosts is roadmap 038"},
		"approvals on gitlab": {gitlabTarget + "FORGE_GITHUB_APPROVALS=on\n", "FORGE_GITHUB_APPROVALS is GitHub-only in this release"},
	} {
		writeTarget(t, c, "t", tc.body)
		_, err := c.LoadTarget("t", "")
		if err == nil || !strings.Contains(err.Error(), tc.want) || !strings.HasPrefix(err.Error(), "TARGET_INVALID:") {
			t.Errorf("%s: want %q, got %v", name, tc.want, err)
		}
	}
}

func TestForgeEnvHosts(t *testing.T) {
	base := map[string]string{"PILOT_FIXTURE_REPO": "group/sub/x-pilot-fixture", "PILOT_MODEL": "p/m", "PILOT_MODEL_EGRESS_HOSTS": "h", "PILOT_SCM": "gitlab"}
	v, err := ValidateValues(base)
	if err != nil {
		t.Fatal(err)
	}
	c := &Config{Values: v, ConfigDir: t.TempDir()}
	if h := c.FixtureHost(); h.Kind != scm.GitLab || h.Name != "gitlab.com" {
		t.Fatalf("%+v", h)
	}
	if f := c.Fixture(); f.SCM != scm.GitLab || f.Host().Kind != scm.GitLab {
		t.Fatalf("%+v", f)
	}
	if err := RepoGuardFor(scm.GitLab, "group/sub/x-pilot-fixture", ""); err != nil {
		t.Fatal(err)
	}
	if err := RepoGuardFor(scm.GitLab, "group/sub/x", ""); err == nil {
		t.Fatal("a GitLab fixture without -pilot-fixture passed the guard")
	}
	for k, val := range map[string]string{"PILOT_SCM": "gitea", "PILOT_GITHUB_APPROVALS": "on", "PILOT_PUBLISHER_CREDENTIAL": "app"} {
		bad := map[string]string{}
		for kk, vv := range base {
			bad[kk] = vv
		}
		bad[k] = val
		bad["PILOT_APPROVERS"] = "1"
		if k == "PILOT_PUBLISHER_CREDENTIAL" {
			bad["PILOT_APP_ID"], bad["PILOT_APP_INSTALLATION_ID"] = "1", "2"
		}
		if _, err := ValidateValues(bad); err == nil || strings.Contains(err.Error(), "PILOT_") {
			t.Errorf("%s=%s: %v", k, val, err)
		}
	}
	gh, _ := ValidateValues(map[string]string{"PILOT_FIXTURE_REPO": "o/x-pilot-fixture", "PILOT_MODEL": "p/m", "PILOT_MODEL_EGRESS_HOSTS": "h"})
	if f := (&Config{Values: gh}).Fixture(); f.SCM != "" {
		t.Fatalf("a GitHub fixture keeps SCM unset: %+v", f)
	}
}
