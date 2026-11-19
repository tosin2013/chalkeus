package checks

import (
	"strings"
	"testing"

	"github.com/tosin2013/specfoundry/internal/cluster"
)

func TestRegistryProxy(t *testing.T) {
	regs := []string{"repo.maven.apache.org"}
	allow, err := cluster.Allowlist(regs)
	if err != nil {
		t.Fatal(err)
	}
	withModel, _ := cluster.Allowlist([]string{"repo.maven.apache.org", "model.example.test"})
	cases := []struct {
		name              string
		ready             bool
		reg, bad, allow   string
		want              bool
		detailContains    string
		detailMustNotHave string
	}{
		{"pass", true, "200", "000", allow, true, "registry=http=200 undeclared=http=000 prefetch_allowlist=1 missing=0 unexpected=0", ""},
		{"registry status counts as reached", true, "404", "000", allow, true, "registry=http=404", ""},
		{"probe not ready", false, "000", "000", allow, false, "probe_ready=false", ""},
		{"registry unreachable", true, "000", "000", allow, false, "registry=http=000", ""},
		{"undeclared host answered", true, "200", "200", allow, false, "undeclared=http=200", ""},
		{"prefetch allows the model host", true, "200", "000", withModel, false, "unexpected=1", "model.example.test"},
		{"prefetch missing a registry", true, "200", "000", "", false, "missing=1", ""},
	}
	for _, c := range cases {
		ok, detail := RegistryProxy(c.ready, c.reg, c.bad, c.allow, regs)
		if ok != c.want || !strings.Contains(detail, c.detailContains) {
			t.Errorf("%s: got %t %q", c.name, ok, detail)
		}
		if c.detailMustNotHave != "" && strings.Contains(detail, c.detailMustNotHave) {
			t.Errorf("%s: detail names a host: %q", c.name, detail)
		}
	}
}
