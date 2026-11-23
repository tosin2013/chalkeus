package checks

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/tosin2013/chalkeus/internal/k8s"
	envnames "github.com/tosin2013/chalkeus/internal/names"
)

// AuthorRow is one isolation check of an automatic draft's author namespace (024 R6).
type AuthorRow struct {
	ID, Result, Detail string
}

const authorProxy = "http://proxy.pilot-egress-a.svc.cluster.local:8888"

func row(id string, ok bool, detail string) AuthorRow {
	r := "fail"
	if ok {
		r = "pass"
	}
	return AuthorRow{ID: id, Result: r, Detail: detail}
}

// AuthorPodRule is A01: one pod with one container, and no service-account token (SA and pod).
func AuthorPodRule(pods int, containers int, saAutomount, podAutomount any) (bool, string) {
	return pods == 1 && containers == 1 && saAutomount == false && podAutomount == false,
		fmt.Sprintf("pods=%d containers=%d sa_automount=%v pod_automount=%v", pods, containers, saAutomount, podAutomount)
}

// AuthorSecretsRule is A02: the namespace's only Secret is model-key, and no container has an
// SCM-like environment variable.
func AuthorSecretsRule(secretNames, envNames []string) (bool, string) {
	var bad []string
	for _, e := range envNames {
		if scmRE.MatchString(e) && !pilotEnv[e] {
			bad = append(bad, e)
		}
	}
	sort.Strings(secretNames)
	return strings.Join(secretNames, ",") == "model-key" && len(bad) == 0,
		fmt.Sprintf("secrets=%v scm_like_env=%v", secretNames, bad)
}

// AuthorProxyIngressRule is A05: the author proxy accepts clients only from role=author
// namespaces, so no worker, grader or publisher can use it.
func AuthorProxyIngressRule(policy map[string]any) (bool, string) {
	var roles []string
	for _, in := range l(m(policy["spec"])["ingress"]) {
		for _, from := range l(m(in)["from"]) {
			sel := m(m(m(from)["namespaceSelector"])["matchLabels"])
			if len(sel) != 1 || s(sel["chalkeus.io/role"]) == "" || m(from)["ipBlock"] != nil {
				return false, "ingress has a selector other than chalkeus.io/role"
			}
			roles = append(roles, s(sel["chalkeus.io/role"]))
		}
	}
	return strings.Join(roles, ",") == "author", fmt.Sprintf("ingress_roles=%v", roles)
}

// RunAuthorChecks runs A01-A05 against a running author namespace: A01, A02 on the namespace
// itself, A03, A04 from a probe twin with the same policies, A05 on the author proxy.
func (d Deps) RunAuthorChecks(ns, draftID string, imgs map[string]string) []AuthorRow {
	var rows []AuthorRow
	pod := d.waitPodLabel(ns, "app=author", 120*time.Second) // the author Job's pod (author-job.yaml.tmpl)
	spec := m(pod["spec"])
	pods, _ := d.C.List(ns, "pods", "")
	sa, _ := d.C.Get(ns, "sa", "author")
	ok, det := AuthorPodRule(len(pods), len(l(spec["containers"])), sa["automountServiceAccountToken"], spec["automountServiceAccountToken"])
	rows = append(rows, row("A01", ok, det))

	secrets, _ := d.C.List(ns, "secrets", "")
	var names, envs []string
	for _, sec := range secrets {
		names = append(names, s(m(sec["metadata"])["name"]))
	}
	for _, c := range append(l(spec["containers"]), l(spec["initContainers"])...) {
		for _, e := range l(m(c)["env"]) {
			envs = append(envs, s(m(e)["name"]))
		}
	}
	ok, det = AuthorSecretsRule(names, envs)
	if pod == nil { // no pod read means no env checked: never pass by default
		ok, det = false, "no author pod found; "+det
	}
	rows = append(rows, row("A02", ok, det))

	twin := ns + "-probe"
	func() {
		defer d.C.DeleteNS(twin, false)
		d.C.DeleteNS(twin, true)
		ready := false
		if y, err := k8s.Render(filepath.Join(d.L.Manifests, "author-ns.yaml.tmpl"), map[string]string{"NS": twin, "DRAFT_ID": draftID}); err == nil && d.C.Apply(y) == nil {
			if y, err := k8s.Render(filepath.Join(d.L.Manifests, "author-probe.yaml.tmpl"), map[string]string{"NS": twin, "TOOLS_IMAGE": imgs["tools"]}); err == nil && d.C.Apply(y) == nil {
				ready = d.C.WaitPodReady(twin, "probe", 120*time.Second)
			}
		}
		mh, _ := os.ReadFile(d.L.ModelHost)
		host := envnames.Env("PROBE_MODEL_URL")
		if host == "" {
			host = "https://" + strings.TrimSpace(string(mh)) + "/"
		}
		mOK, mD, gOK, gD, dOK, dD := false, "probe not ready", false, "", false, ""
		sOK, sD, wOK, wD := false, "", false, ""
		if ready {
			mOK, mD = d.reach(twin, "probe", host, authorProxy)
			gOK, gD = d.blocked(twin, "probe", "https://github.com/", authorProxy)
			dOK, dD = d.blocked(twin, "probe", "https://1.1.1.1/", "")
			sOK, sD = d.blocked(twin, "probe", standIn, "")
			wOK, wD = d.blocked(twin, "probe", "https://github.com/", modelProxy)
		}
		rows = append(rows, row("A03", mOK && gOK && dOK, fmt.Sprintf("model=%s github_via_proxy=%s direct=%s", mD, gD, dD)))
		rows = append(rows, row("A04", sOK && wOK, fmt.Sprintf("grader_standin=%s worker_proxy=%s", sD, wD)))
	}()

	np, _ := d.C.Get("pilot-egress-a", "netpol", "proxy")
	ok, det = AuthorProxyIngressRule(np)
	rows = append(rows, row("A05", ok, det))
	return rows
}
