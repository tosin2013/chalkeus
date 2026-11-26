package scm

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tosin2013/chalkeus/internal/secret"
)

func TestOperatorCredentialOrder(t *testing.T) {
	for _, k := range []string{"GH_TOKEN", "GITHUB_TOKEN", "GITLAB_TOKEN", "GITLAB_ACCESS_TOKEN", "BITBUCKET_TOKEN"} {
		t.Setenv(k, "")
	}
	old := commandOutput
	t.Cleanup(func() { commandOutput = old })
	var ran []string
	commandOutput = func(name string, args ...string) ([]byte, error) {
		ran = append(ran, name+" "+strings.Join(args, " "))
		return []byte("from-cli\n"), nil
	}
	gl := NewHost(GitLab, "git.corp", "")
	if v, err := OperatorCredential(gl); err != nil || v.Reveal() != "from-cli" || ran[0] != "glab config get token --host git.corp" {
		t.Fatalf("%v %q %v", err, v.Reveal(), ran)
	}
	t.Setenv("GITLAB_ACCESS_TOKEN", "second")
	if v, _ := OperatorCredential(gl); v.Reveal() != "second" {
		t.Error("GITLAB_ACCESS_TOKEN not used")
	}
	t.Setenv("GITLAB_TOKEN", "first")
	if v, _ := OperatorCredential(gl); v.Reveal() != "first" {
		t.Error("GITLAB_TOKEN not first")
	}
	bb := NewHost(Bitbucket, "", "")
	_, err := OperatorCredential(bb)
	if !errors.Is(err, ErrNoCredential) || !strings.Contains(err.Error(), "BITBUCKET_TOKEN") {
		t.Fatalf("bitbucket without a token: %v", err)
	}
	t.Setenv("BITBUCKET_TOKEN", "bb")
	if v, _ := OperatorCredential(bb); v.Reveal() != "bb" {
		t.Error("BITBUCKET_TOKEN")
	}
	t.Setenv("GH_TOKEN", "gh")
	if v, _ := OperatorCredential(Host{}); v.Reveal() != "gh" {
		t.Error("GH_TOKEN")
	}
}

func TestGitEnv(t *testing.T) {
	if GitEnv(Host{}, secret.New("x")) != nil {
		t.Error("GitHub keeps the gh helper")
	}
	env := strings.Join(GitEnv(NewHost(GitLab, "", "/cfg/ca.pem"), secret.New("tok")), "\n")
	for _, want := range []string{"GIT_CONFIG_COUNT=3", "GIT_CONFIG_KEY_1=credential.https://gitlab.com.helper", "GIT_CONFIG_VALUE_0=\n", "username=oauth2", "http.sslCAInfo", "/cfg/ca.pem", GitTokenEnv + "=tok"} {
		if !strings.Contains(env+"\n", want) {
			t.Errorf("missing %q in\n%s", want, env)
		}
	}
	if GitUser(NewHost(Bitbucket, "", ""), true) != "x-token-auth" || GitUser(NewHost(Bitbucket, "", ""), false) != "x-bitbucket-api-token-auth" {
		t.Error("bitbucket git users")
	}
}

// The inline helper really answers git: a credential fill returns the username and the token.
func TestGitEnvHelperAnswersGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	cmd := exec.Command("git", "credential", "fill")
	cmd.Dir = t.TempDir()
	cmd.Env = append(append(os.Environ(), "HOME="+filepath.Join(cmd.Dir, "home")), GitEnv(NewHost(Bitbucket, "", ""), secret.New("s3cr3t"))...)
	cmd.Stdin = strings.NewReader("protocol=https\nhost=bitbucket.org\n\n")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "username=x-bitbucket-api-token-auth") || !strings.Contains(string(out), "password=s3cr3t") {
		t.Fatalf("%s", out)
	}
}

// The helper is scoped to its host: git asking for another host's credential gets nothing from it.
func TestGitEnvHelperIsScopedToItsHost(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	cmd := exec.Command("git", "credential", "fill")
	cmd.Dir = t.TempDir()
	cmd.Env = append(append(os.Environ(), "HOME="+filepath.Join(cmd.Dir, "home"), "GIT_ASKPASS=", "SSH_ASKPASS="), GitEnv(NewHost(GitLab, "", ""), secret.New("s3cr3t"))...)
	cmd.Stdin = strings.NewReader("protocol=https\nhost=evil.example\n\n")
	out, _ := cmd.CombinedOutput() // fails: no helper answers and prompting is off
	if strings.Contains(string(out), "s3cr3t") {
		t.Fatalf("the token was offered to another host: %s", out)
	}
}
