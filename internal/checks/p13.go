package checks

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/tosin2013/chalkeus/internal/evidence"
	"github.com/tosin2013/chalkeus/internal/k8s"
	"github.com/tosin2013/chalkeus/internal/scm"
)

const scmProxy = "http://proxy.pilot-egress-scm.svc.cluster.local:8888"

// P13Decision is P13's rule (037 contracts/checks.md): through the SCM proxy the run's host
// answers and the other probe host is refused, and the run's host is unreachable without the
// proxy. Codes are curl's http_code ("000" = no HTTP answer, as a proxy refusal or a blocked
// connection gives).
func P13Decision(hostCode, otherCode, directCode string) (bool, string) {
	ok := hostCode != "000" && otherCode == "000" && directCode == "000"
	return ok, fmt.Sprintf("host_via_proxy=http=%s other_via_proxy=http=%s direct=http=%s", hostCode, otherCode, directCode)
}

// RunPublisherEgressCheck records P13 for h from a credential-less pod in pilot-publish.
func (d Deps) RunPublisherEgressCheck(run *evidence.Run, toolsImage string, h scm.Host) {
	const ns, pod = "pilot-publish", "p13-probe"
	host := h.EgressHosts()[0]
	if h.IsGitHub() {
		host = "github.com"
	}
	other := h.OtherProbeHost()
	_ = d.C.DeletePod(ns, pod)
	ready := false
	if y, err := k8s.Render(filepath.Join(d.L.Manifests, "publisher-probe.yaml.tmpl"), map[string]string{"TOOLS_IMAGE": toolsImage}); err == nil && d.C.Apply(y) == nil {
		ready = d.C.WaitPodReady(ns, pod, 120*time.Second)
	}
	hc, oc, dc := "000", "000", "000"
	if ready {
		hc = d.curl(ns, pod, "https://"+host+"/", scmProxy)
		oc = d.curl(ns, pod, "https://"+other+"/", scmProxy)
		dc = d.curl(ns, pod, "https://"+host+"/", "")
	}
	ok, detail := P13Decision(hc, oc, dc)
	if !ready {
		ok, detail = false, "probe pod not ready; "+detail
	}
	d.rec(run, "P13", "037 FR-011", ok, fmt.Sprintf("publisher reaches %s through its proxy only; %s is refused", host, other),
		"cluster", strings.TrimSpace(detail))
	_ = d.C.DeletePod(ns, pod)
}
