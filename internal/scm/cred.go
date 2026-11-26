package scm

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/tosin2013/chalkeus/internal/secret"
)

// commandOutput runs a credential CLI (gh, glab); replaced in tests.
var commandOutput = func(name string, args ...string) ([]byte, error) {
	if _, err := exec.LookPath(name); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, name, args...).Output()
}

// OperatorCredential is the human operator's own token for h (research R8). It is only for reads
// and git on the operator's machine: never written to disk by forge, never put in a Secret,
// never passed to a pod.
func OperatorCredential(h Host) (secret.Value, error) {
	var envs []string
	var cli []string
	var how string
	switch h.KindOrGitHub() {
	case GitLab:
		envs = []string{"GITLAB_TOKEN", "GITLAB_ACCESS_TOKEN"}
		cli = []string{"glab", "config", "get", "token", "--host", hostOnly(h.Name)}
		how = "set GITLAB_TOKEN to your own GitLab token for " + h.Name + " (scope api), or log in with glab"
	case Bitbucket:
		envs = []string{"BITBUCKET_TOKEN"}
		how = "set BITBUCKET_TOKEN to your own Atlassian API token for Bitbucket (repository and pull request read; repository admin for doctor's approver check)"
	default:
		envs = []string{"GH_TOKEN", "GITHUB_TOKEN"}
		cli = []string{"gh", "auth", "token"}
		how = "set GH_TOKEN, or log in with gh auth login"
	}
	for _, k := range envs {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return secret.New(v), nil
		}
	}
	if len(cli) > 0 {
		if out, err := commandOutput(cli[0], cli[1:]...); err == nil {
			if v := strings.TrimSpace(string(out)); v != "" {
				return secret.New(v), nil
			}
		}
	}
	return secret.Value{}, &NoCredentialError{Host: h, How: how}
}

// NoCredentialError says how to provide the operator's credential for a host.
type NoCredentialError struct {
	Host Host
	How  string
}

func (e *NoCredentialError) Error() string {
	return "no credential of yours for " + e.Host.Name + ": " + e.How
}

// ErrNoCredential matches a *NoCredentialError with errors.Is.
var ErrNoCredential = errors.New("no operator credential")

func (e *NoCredentialError) Is(target error) bool { return target == ErrNoCredential }

// GitUser is the HTTPS username git sends with a token on each host (research R4, R8).
func GitUser(h Host, publisher bool) string {
	switch h.KindOrGitHub() {
	case GitLab:
		return "oauth2"
	case Bitbucket:
		if publisher {
			return "x-token-auth" // repository access tokens
		}
		return "x-bitbucket-api-token-auth" // the operator's Atlassian API token
	}
	return "x-access-token"
}

// GitTokenEnv is the variable GitEnv's credential helper reads the token from.
const GitTokenEnv = "CHALKEUS_GIT_TOKEN"

// GitEnv is the environment host-side git needs for h with the operator's token: GitHub returns
// nil (git keeps using the gh credential helper, unchanged). GitLab and Bitbucket get an inline
// credential helper through git's GIT_CONFIG_COUNT variables (git 2.31+) that answers with the
// username and the token from GitTokenEnv, so the token is never on a command line; plus
// http.sslCAInfo when a CA bundle is set.
func GitEnv(h Host, token secret.Value) []string {
	if h.IsGitHub() {
		return nil
	}
	pairs := [][2]string{
		{"credential.helper", ""}, // drop any helper the operator configured
		{"credential.helper", `!f() { test "$1" = get || exit 0; echo username=` + GitUser(h, false) + `; echo "password=$` + GitTokenEnv + `"; }; f`},
	}
	if h.CABundle != "" {
		pairs = append(pairs, [2]string{"http.sslCAInfo", h.CABundle})
	}
	env := []string{"GIT_TERMINAL_PROMPT=0", GitTokenEnv + "=" + token.Reveal(), "GIT_CONFIG_COUNT=" + strconv.Itoa(len(pairs))}
	for i, p := range pairs {
		env = append(env, "GIT_CONFIG_KEY_"+strconv.Itoa(i)+"="+p[0], "GIT_CONFIG_VALUE_"+strconv.Itoa(i)+"="+p[1])
	}
	return env
}
