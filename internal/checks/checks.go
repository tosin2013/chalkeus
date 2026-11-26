// Package checks runs the live isolation and credential checks P01–P12 (with P06r for item
// runs whose target declares registries) and the gate
// fail-closed matrix GM01–GM04 (reference: the former Python pilot's checks.py;
// specs/002-m1-gated-pilot/contracts/evidence-record.md). Detail strings match the reference.
//
// Network probes run from a probe twin: a namespace rendered from the same worker template
// (same labels, policies, quota) holding a sleeping tools pod, because the worker namespace
// admits only one pod. Control probes prove that "blocked" comes from policy, not the network.
package checks

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/tosin2013/chalkeus/internal/evidence"
	"github.com/tosin2013/chalkeus/internal/gate"
	"github.com/tosin2013/chalkeus/internal/k8s"
	envnames "github.com/tosin2013/chalkeus/internal/names"
	"github.com/tosin2013/chalkeus/internal/paths"
	"github.com/tosin2013/chalkeus/internal/pyrepr"
)

const (
	file       = "checks/results.jsonl"
	standIn    = "http://target.pilot-grader-standin.svc.cluster.local:8080/"
	modelProxy = "http://proxy.pilot-egress-model.svc.cluster.local:8888"
)

var (
	scmRE = regexp.MustCompile(`(?i)TOKEN|GITHUB|^GH_|PASSWORD|GITLAB|BITBUCKET|GLPAT`) // 037: other hosts' names
	// The pilot's own non-credential env names (PILOT_TOKEN_CAP is a spend cap, not a token).
	pilotEnv = map[string]bool{"PILOT_MODEL": true, "PILOT_SPEND_CAP_USD": true, "PILOT_TOKEN_CAP": true,
		"HTTPS_PROXY": true, "https_proxy": true, "NO_PROXY": true}
	control = [2]string{"pilot-control", "control"}
)

// Deps are the checks' collaborators.
type Deps struct {
	L   paths.Layout
	C   *k8s.Client
	Out io.Writer
	Net *ItemNet // an item run's target proxies; nil when the target declares no registries
}

// ItemNet is an item run's target proxies (014 R6), for P06r.
type ItemNet struct {
	WorkerProxyNS, PrefetchProxyNS string
	Registries                     []string
}

func (d Deps) rec(run *evidence.Run, id, req string, ok bool, desc, unit, detail string) {
	r := "fail"
	if ok {
		r = "pass"
	}
	_, _ = run.RecordCheck(d.Out, file, id, req, r, desc, strings.TrimSpace(unit+" "+detail))
}

func (d Deps) curl(ns, pod, url, proxy string) string {
	args := []string{"curl", "-s", "-o", "/dev/null", "-w", "%{http_code}", "--max-time", "8"}
	if proxy != "" {
		args = append(args, "-x", proxy)
	}
	r := d.C.Exec(ns, pod, 30*time.Second, append(args, url)...)
	if c := strings.TrimSpace(r.Stdout); c != "" {
		return c
	}
	return "000"
}

func (d Deps) reach(ns, pod, url, proxy string) (bool, string) {
	c := d.curl(ns, pod, url, proxy)
	return c != "000", "http=" + c
}

func (d Deps) blocked(ns, pod, url, proxy string) (bool, string) {
	c := d.curl(ns, pod, url, proxy)
	return c == "000", "http=" + c
}

func m(v any) map[string]any { x, _ := v.(map[string]any); return x }
func l(v any) []any          { x, _ := v.([]any); return x }
func s(v any) string         { x, _ := v.(string); return x }

func (d Deps) waitPod(ns string, timeout time.Duration) map[string]any {
	return d.waitPodLabel(ns, "app=worker", timeout)
}

// waitPodLabel waits for the first pod matching a label selector, or returns nil.
func (d Deps) waitPodLabel(ns, selector string, timeout time.Duration) map[string]any {
	end := time.Now().Add(timeout)
	for time.Now().Before(end) {
		pods, _ := d.C.List(ns, "pods", selector)
		if len(pods) > 0 {
			return pods[0]
		}
		time.Sleep(2 * time.Second)
	}
	return nil
}

func strsAny(xs []string) []any {
	out := make([]any, len(xs))
	for i, x := range xs {
		out[i] = x
	}
	return out
}

// RunWorkerChecks runs P01–P08 against a running worker attempt.
func (d Deps) RunWorkerChecks(run *evidence.Run, imgs map[string]string, task string, n int, ns string) {
	unit := fmt.Sprintf("%s-%d", task, n)
	twin := ns + "-probe"
	sa := fmt.Sprintf("system:serviceaccount:%s:worker", ns)
	pod := d.waitPod(ns, 120*time.Second)
	podName := ""
	if pod != nil {
		podName = s(m(pod["metadata"])["name"])
		d.C.WaitPodReady(ns, podName, 120*time.Second)
	}
	spec := m(pod["spec"])
	containers := l(spec["containers"])
	// P01 one pod, one container
	pods, _ := d.C.List(ns, "pods", "")
	d.rec(run, "P01", "FR-005", len(pods) == 1 && len(containers) == 1, "worker namespace has exactly 1 pod with 1 container",
		unit, fmt.Sprintf("pods=%d containers=%d", len(pods), len(containers)))
	// P02 no service-account token
	saObj, _ := d.C.Get(ns, "sa", "worker")
	tok := "None"
	if pod != nil {
		r := d.C.Exec(ns, podName, 20*time.Second, "sh", "-c", "test ! -e /var/run/secrets/kubernetes.io/serviceaccount/token && echo absent")
		tok = strings.TrimSpace(r.Stdout)
		if tok == "" {
			if r.Code != 0 {
				tok = "pod-finished"
			} else {
				tok = "present"
			}
		}
	}
	ok := saObj["automountServiceAccountToken"] == false && pod != nil && spec["automountServiceAccountToken"] == false &&
		(tok == "absent" || tok == "pod-finished")
	d.rec(run, "P02", "FR-006", ok, "SA and pod automount off; no token file", unit, "token="+tok)
	// P03 policies, quota, deadline
	np, _ := d.C.Get(ns, "netpol", "default-deny-all")
	rq, _ := d.C.Get(ns, "resourcequota", "worker-quota")
	q := m(m(rq["spec"])["hard"])
	if q == nil {
		q = map[string]any{}
	}
	job, _ := d.C.Get(ns, "job", "worker")
	dl := m(job["spec"])["activeDeadlineSeconds"]
	types := map[string]bool{}
	for _, t := range l(m(np["spec"])["policyTypes"]) {
		types[s(t)] = true
	}
	ok = len(types) == 2 && types["Ingress"] && types["Egress"] && q["pods"] == "1" && q["count/secrets"] == "1" && dl != nil && dl != int64(0) && dl != float64(0)
	d.rec(run, "P03", "FR-006", ok, "default-deny netpol, quota pods=1 secrets=1, Job deadline", unit,
		fmt.Sprintf("quota=%s deadline=%s", pyrepr.Dict(q), pyrepr.Value(dl)))
	// P04 only model-key Secret; no SCM-like names in secret keys or env
	secrets, _ := d.C.List(ns, "secrets", "")
	var names, keys, bad []string
	for _, sec := range secrets {
		names = append(names, s(m(sec["metadata"])["name"]))
		data := m(sec["data"])
		ks := make([]string, 0, len(data))
		for k := range data {
			ks = append(ks, k)
		}
		sort.Strings(ks)
		keys = append(keys, ks...)
	}
	var envs []string
	for _, c := range containers {
		for _, e := range l(m(c)["env"]) {
			envs = append(envs, s(m(e)["name"]))
		}
	}
	for _, x := range append(append([]string{}, keys...), envs...) {
		if scmRE.MatchString(x) && !pilotEnv[x] {
			bad = append(bad, x)
		}
	}
	d.rec(run, "P04", "FR-007", len(names) == 1 && names[0] == "model-key" && len(bad) == 0 && pod != nil,
		"only Secret is model-key; no SCM-like secret keys or env names", unit,
		fmt.Sprintf("secrets=%s keys=%s bad=%s", pyrepr.Value(strsAny(names)), pyrepr.Value(strsAny(keys)), pyrepr.Value(strsAny(bad))))
	// P05 no grading material
	cms, _ := d.C.List(ns, "configmaps", "")
	var cmNames []string
	hasGrading := false
	for _, cm := range cms {
		n := s(m(cm["metadata"])["name"])
		cmNames = append(cmNames, n)
		hasGrading = hasGrading || n == "grading"
	}
	gradingMount := false
	for _, c := range containers {
		for _, vm := range l(m(c)["volumeMounts"]) {
			gradingMount = gradingMount || strings.HasPrefix(s(m(vm)["mountPath"]), "/grading")
		}
	}
	d.rec(run, "P05", "FR-007", !hasGrading && !gradingMount, "no grading material in the worker namespace", unit,
		"configmaps="+pyrepr.Value(strsAny(cmNames)))
	if run.Kind() == evidence.KindItem {
		var cmText strings.Builder
		for _, cm := range cms {
			for k, v := range m(cm["data"]) {
				cmText.WriteString(k + "\n" + s(v) + "\n")
			}
		}
		adir, _ := run.AttemptDir(task, n)
		promptText, _ := os.ReadFile(filepath.Join(adir, "prompt.md"))
		ok, detail := HiddenTestsAbsent(run.Dir, cmText.String(), string(promptText), []string{"base.bundle", "meta.json", "prompt.md"})
		d.rec(run, "P11", "014 FR-008", ok, "no hidden-test file reaches the worker (ConfigMaps, staged inputs, prompt)", unit, detail)
	}
	// P06/P07 network, from the probe twin
	func() {
		defer d.C.DeleteNS(twin, false)
		d.C.DeleteNS(twin, true)
		ready := false
		if y, err := k8s.Render(filepath.Join(d.L.Manifests, "worker-ns.yaml.tmpl"), map[string]string{"NS": twin, "RUN_ID": run.ID, "QUOTA_CPU": "2", "QUOTA_MEMORY": "2Gi"}); err == nil && d.C.Apply(y) == nil {
			if y, err := k8s.Render(filepath.Join(d.L.Manifests, "worker-probe.yaml.tmpl"), map[string]string{"NS": twin, "TOOLS_IMAGE": imgs["tools"]}); err == nil && d.C.Apply(y) == nil {
				ready = d.C.WaitPodReady(twin, "probe", 120*time.Second)
			}
		}
		mh, _ := os.ReadFile(d.L.ModelHost)
		host := envnames.Env("PROBE_MODEL_URL")
		if host == "" {
			host = "https://" + strings.TrimSpace(string(mh)) + "/"
		}
		host = strings.TrimRight(host, "/") + "/"
		mOK, mD := false, "probe not ready"
		gOK, gD, dOK, dD := false, "", false, ""
		if ready {
			mOK, mD = d.reach(twin, "probe", host, modelProxy)
			gOK, gD = d.blocked(twin, "probe", "https://github.com/", modelProxy)
			dOK, dD = d.blocked(twin, "probe", "https://1.1.1.1/", "")
		}
		cOK, cD := d.reach(control[0], control[1], "https://github.com/", "")
		d.rec(run, "P06", "FR-009b", mOK && gOK && dOK && cOK,
			"model host via proxy reachable; github.com via proxy and direct internet blocked; control reaches github.com",
			unit, fmt.Sprintf("model=%s github_via_proxy=%s direct=%s control=%s", mD, gD, dD, cD))
		sOK, sD := false, ""
		if ready {
			sOK, sD = d.blocked(twin, "probe", standIn, "")
		}
		scOK, scD := d.reach(control[0], control[1], standIn, "")
		d.rec(run, "P07", "FR-010", sOK && scOK, "worker-policy pod cannot reach the grader stand-in; control can",
			unit, fmt.Sprintf("twin=%s control=%s", sD, scD))
	}()
	if d.Net != nil && len(d.Net.Registries) > 0 {
		d.registryCheck(run, imgs, ns, unit)
	}
	// P08 RBAC
	gns := fmt.Sprintf("pilot-g-%s-%d", strings.ToLower(task), n)
	if run.Kind() == evidence.KindItem { // one grading namespace per item run (014 R7)
		gns = "pilot-g-" + strings.ToLower(task)
	}
	ans := map[string]any{}
	for _, r := range []string{"pods", "configmaps", "pods/exec"} {
		ans[r] = d.C.CanI("create", r, gns, sa)
	}
	ans["patch-configmaps"] = d.C.CanI("patch", "configmaps", gns, sa)
	allNo := true
	for _, v := range ans {
		allNo = allNo && v == "no"
	}
	d.rec(run, "P08", "FR-010", allNo, "worker SA cannot create/patch/exec in its grader namespace", unit, pyrepr.SortedJSON(ans))
}

// registryCheck records P06r (014 R6) from a second probe twin rendered like the item's worker
// namespace, whose one proxy is the target's worker proxy.
func (d Deps) registryCheck(run *evidence.Run, imgs map[string]string, ns, unit string) {
	twin := ns + "-probe-r"
	defer d.C.DeleteNS(twin, false)
	d.C.DeleteNS(twin, true)
	ready := false
	if y, err := k8s.Render(filepath.Join(d.L.Manifests, "worker-ns-item.yaml.tmpl"),
		map[string]string{"NS": twin, "RUN_ID": run.ID, "PROXY_NS": d.Net.WorkerProxyNS, "QUOTA_CPU": "2", "QUOTA_MEMORY": "2Gi"}); err == nil && d.C.Apply(y) == nil {
		if y, err := k8s.Render(filepath.Join(d.L.Manifests, "worker-probe.yaml.tmpl"), map[string]string{"NS": twin, "TOOLS_IMAGE": imgs["tools"]}); err == nil && d.C.Apply(y) == nil {
			ready = d.C.WaitPodReady(twin, "probe", 120*time.Second)
		}
	}
	proxy := "http://proxy." + d.Net.WorkerProxyNS + ".svc.cluster.local:8888"
	reg, bad := "000", "000"
	if ready {
		reg = d.curl(twin, "probe", "https://"+d.Net.Registries[0]+"/", proxy)
		bad = d.curl(twin, "probe", "https://"+undeclaredHost+"/", proxy)
	}
	cm, _ := d.C.Get(d.Net.PrefetchProxyNS, "configmap", "proxy-config")
	ok, detail := RegistryProxy(ready, reg, bad, s(m(cm["data"])["allowlist"]), d.Net.Registries)
	d.rec(run, "P06r", "014 R6", ok,
		"a declared registry is reachable through the target's proxy; "+undeclaredHost+" is refused; the prefetch proxy allows only the registries",
		unit, detail)
}

// undeclaredHost is the host P06r expects every proxy to refuse.
const undeclaredHost = "example.invalid"

// RegistryProxy is P06r's rule: the probe reached the first declared registry through the
// worker's target proxy (any HTTP status), the undeclared host was refused (no status), and the
// prefetch proxy's allowlist is exactly the declared registries. The detail names no host, so
// an unexpected allowlist entry (such as a model host) is only counted.
func RegistryProxy(ready bool, registryCode, undeclaredCode, prefetchAllowlist string, registries []string) (bool, string) {
	want := map[string]bool{}
	for _, h := range registries {
		want[h] = true
	}
	got, extra := map[string]bool{}, 0
	for _, line := range strings.Split(prefetchAllowlist, "\n") {
		h := strings.ReplaceAll(strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(line), "^"), "$"), `\`, "")
		if h == "" {
			continue
		}
		got[h] = true
		if !want[h] {
			extra++
		}
	}
	missing := 0
	for h := range want {
		if !got[h] {
			missing++
		}
	}
	ok := ready && registryCode != "000" && undeclaredCode == "000" && extra == 0 && missing == 0
	return ok, fmt.Sprintf("probe_ready=%t registry=http=%s undeclared=http=%s prefetch_allowlist=%d missing=%d unexpected=%d",
		ready, registryCode, undeclaredCode, len(got), missing, extra)
}

// HiddenTestsAbsent is P11's rule (014 FR-008): no hidden-test path from companion.json appears in
// the worker's ConfigMaps or prompt, and the staged inputs are exactly the expected three.
func HiddenTestsAbsent(runDir, configMaps, prompt string, staged []string) (bool, string) {
	var c struct {
		Files []struct {
			Path string `json:"path"`
		} `json:"files"`
	}
	raw, err := os.ReadFile(filepath.Join(runDir, "companion.json"))
	if err != nil || json.Unmarshal(raw, &c) != nil || len(c.Files) == 0 {
		return false, "companion.json missing or empty"
	}
	var leaks []string
	for _, f := range c.Files {
		if strings.Contains(configMaps, f.Path) || strings.Contains(prompt, f.Path) {
			leaks = append(leaks, f.Path)
		}
	}
	want := map[string]bool{"base.bundle": true, "meta.json": true, "prompt.md": true}
	for _, n := range staged {
		if !want[n] {
			leaks = append(leaks, "staged:"+n)
		}
	}
	return len(leaks) == 0, fmt.Sprintf("hidden_files=%d leaks=%s", len(c.Files), pyrepr.Value(strsAny(leaks)))
}

// GradingPolicies is P12's rule (014 R7) over a grading namespace's NetworkPolicies: a default deny
// for every pod, and at most one allow rule, which selects only the prefetch pod and reaches only
// DNS and the target's prefetch proxy. So graders have no egress at all.
func GradingPolicies(policies []map[string]any, prefetchProxyNS string) (bool, string) {
	deny, extra := false, []string{}
	for _, np := range policies {
		name := s(m(np["metadata"])["name"])
		spec := m(np["spec"])
		sel := m(spec["podSelector"])
		labels := m(sel["matchLabels"])
		switch {
		case name == "default-deny-all" && len(labels) == 0 && len(l(spec["egress"])) == 0 && len(l(spec["ingress"])) == 0:
			deny = true
		case name == "allow-prefetch-egress" && len(labels) == 1 && s(labels["pilot/role"]) == "prefetch":
			for _, rule := range l(spec["egress"]) {
				for _, to := range l(m(rule)["to"]) {
					ns := s(m(m(m(to)["namespaceSelector"])["matchLabels"])["kubernetes.io/metadata.name"])
					if ns != "kube-system" && ns != prefetchProxyNS {
						extra = append(extra, name+"->"+ns)
					}
					if m(to)["ipBlock"] != nil {
						extra = append(extra, name+"->ipBlock")
					}
				}
			}
		default:
			extra = append(extra, name)
		}
	}
	return deny && len(extra) == 0, fmt.Sprintf("default_deny=%t unexpected=%s", deny, pyrepr.Value(strsAny(extra)))
}

// RunGradingChecks records P12 for an item run's grading namespace.
func (d Deps) RunGradingChecks(run *evidence.Run, ns, prefetchProxyNS string) {
	nps, _ := d.C.List(ns, "netpol", "")
	ok, detail := GradingPolicies(nps, prefetchProxyNS)
	d.rec(run, "P12", "014 R7", ok, "graders have no egress; only the prefetch pod reaches DNS and its registries-only proxy", ns, detail)
}

// RunClusterChecks runs P09 and P10.
func (d Deps) RunClusterChecks(run *evidence.Run) {
	// P09: only pilot-publish/scm-token holds an SCM-like credential.
	secrets, _ := d.C.List("", "secrets", "")
	set := map[string]bool{}
	for _, sec := range secrets {
		md := m(sec["metadata"])
		ns := s(md["namespace"])
		if !strings.HasPrefix(ns, "pilot-") {
			continue
		}
		for k := range m(sec["data"]) {
			if scmRE.MatchString(k) {
				set[ns+"/"+s(md["name"])] = true
			}
		}
	}
	holders := make([]string, 0, len(set))
	for h := range set {
		holders = append(holders, h)
	}
	sort.Strings(holders)
	d.rec(run, "P09", "FR-013", len(holders) == 1 && holders[0] == "pilot-publish/scm-token", "only pilot-publish holds an SCM credential",
		"cluster", "holders="+pyrepr.Value(strsAny(holders)))
	// P10: every pilot pod is digest-pinned, non-root, no privilege escalation.
	pods, _ := d.C.List("", "pods", "")
	var bad []string
	count := 0
	for _, p := range pods {
		md := m(p["metadata"])
		ns := s(md["namespace"])
		if !strings.HasPrefix(ns, "pilot-") {
			continue
		}
		count++
		spec := m(p["spec"])
		psc := m(spec["securityContext"])
		for _, c := range l(spec["containers"]) {
			cm := m(c)
			sc := m(cm["securityContext"])
			nonroot, has := sc["runAsNonRoot"]
			if !has {
				nonroot = psc["runAsNonRoot"]
			}
			if !strings.Contains(s(cm["image"]), "@sha256:") || nonroot != true || sc["allowPrivilegeEscalation"] != false {
				bad = append(bad, fmt.Sprintf("%s/%s/%s", ns, s(md["name"]), s(cm["name"])))
			}
		}
	}
	d.rec(run, "P10", "S6", len(bad) == 0, "pilot pods digest-pinned, non-root, no privilege escalation", "cluster",
		fmt.Sprintf("pods=%d bad=%s", count, pyrepr.Value(strsAny(bad))))
}

// GateMatrix runs GM01 (the unit matrix, in-process) and the live fail-closed variants GM02–GM04.
func (d Deps) GateMatrix(run *evidence.Run, src gate.StatusSource) {
	ok, detail := gate.RunMatrix()
	d.rec(run, "GM01", "FR-002", ok, "gate unit matrix (all refusal cases refuse)", "gate", detail)
	t, err := os.MkdirTemp("", "pilot-gm-")
	if err != nil {
		return
	}
	defer func() { _ = os.RemoveAll(t) }()
	dec, _ := gate.Evaluate(src, "A", "001", t, 0, "", "variant:no-roadmap", nil)
	d.rec(run, "GM02", "FR-002", dec.Decision == "refuse", "missing roadmap refuses", "gate", strings.Join(dec.Reasons, ","))
	fx := filepath.Join(t, "fx")
	if err := copyDir(d.L.FixtureTemplate, fx); err != nil {
		return
	}
	dec, _ = gate.Evaluate(gate.ExecStatus{Bin: "/bin/false"}, "A", "001", fx, 0, "", "variant:tool-error", nil)
	d.rec(run, "GM03", "FR-002", dec.Decision == "refuse", "status tool error refuses", "gate", strings.Join(dec.Reasons, ","))
	rm := filepath.Join(fx, ".specify/memory/roadmap.md")
	raw, _ := os.ReadFile(rm)
	text := string(raw)
	if i := strings.Index(text, "### 002"); i >= 0 {
		_ = os.WriteFile(rm, []byte(text[:i]+"### 002 — Unique sl"), 0o644)
	}
	dec, _ = gate.Evaluate(src, "B", "002", fx, 0, "", "variant:truncated", nil)
	d.rec(run, "GM04", "FR-002", dec.Decision == "refuse", "truncated roadmap refuses B", "gate", strings.Join(dec.Reasons, ","))
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		t := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(t, 0o755)
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(t, raw, info.Mode().Perm())
	})
}
