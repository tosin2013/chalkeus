package cli

import (
	"strings"
	"testing"

	"github.com/tosin2013/chalkeus/internal/scm"
)

func TestGitHubOnly(t *testing.T) {
	if gitHubOnly("forge debt", scm.Host{}) != nil || gitHubOnly("forge debt", scm.NewHost(scm.GitHub, "", "")) != nil {
		t.Fatal("GitHub must pass")
	}
	err := gitHubOnly("forge debt", scm.NewHost(scm.Bitbucket, "", ""))
	if err == nil || !strings.Contains(err.Error(), "forge debt is GitHub-only in this release") || !strings.Contains(err.Error(), "bitbucket.org") {
		t.Fatalf("%v", err)
	}
	if e, ok := err.(*ExitErr); !ok || e.Code != ExitError {
		t.Fatalf("exit code: %#v", err)
	}
}
