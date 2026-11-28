package publisher

import (
	"testing"

	"github.com/tosin2013/chalkeus/internal/scm"
)

func TestSCMBlock(t *testing.T) {
	if scmBlock(scm.Host{}) != nil || scmBlock(scm.NewHost(scm.GitHub, "", "")) != nil {
		t.Fatal("GitHub's pr.json carries no scm block")
	}
	b := scmBlock(scm.NewHost(scm.Bitbucket, "", ""))
	if b["kind"] != "bitbucket" || b["api"] != "https://api.bitbucket.org/2.0" || b["git_base"] != "https://bitbucket.org/" {
		t.Fatalf("%v", b)
	}
}
