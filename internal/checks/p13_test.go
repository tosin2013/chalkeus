package checks

import (
	"testing"

	"github.com/tosin2013/chalkeus/internal/scm"
)

func TestP13Decision(t *testing.T) {
	for _, tc := range []struct {
		host, other, direct string
		ok                  bool
	}{
		{"200", "000", "000", true},
		{"302", "000", "000", true},
		{"000", "000", "000", false}, // the run's host not reachable through the proxy
		{"200", "200", "000", false}, // another Git host reachable: the allowlist is too wide
		{"200", "000", "200", false}, // the publisher reaches the internet without the proxy
	} {
		if ok, d := P13Decision(tc.host, tc.other, tc.direct); ok != tc.ok {
			t.Errorf("%+v: %v (%s)", tc, ok, d)
		}
	}
}

// The publisher's allowlist holds only the run's host, and P13's other probe host is never in it.
func TestP13HostsPerKind(t *testing.T) {
	for _, h := range []scm.Host{scm.NewHost(scm.GitHub, "", ""), scm.NewHost(scm.GitLab, "", ""), scm.NewHost(scm.GitLab, "git.corp", ""), scm.NewHost(scm.Bitbucket, "", "")} {
		for _, e := range h.EgressHosts() {
			if e == h.OtherProbeHost() {
				t.Errorf("%s: the other probe host %s is in the allowlist", h.Name, e)
			}
		}
	}
}
