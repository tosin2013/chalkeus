package cluster

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/tosin2013/specfoundry/internal/k8s"
	"github.com/tosin2013/specfoundry/internal/paths"
	"github.com/tosin2013/specfoundry/internal/pyrepr"
)

var hostRE = regexp.MustCompile(`^[A-Za-z0-9.-]+$`)

// pySpecial are the characters Python's re.escape escapes (3.7+).
const pySpecial = "()[]{}?*+-|^$\\.&~# \t\n\r\v\f"

// pyEscape is Python's re.escape, so the tinyproxy allowlist matches the reference byte for byte.
func pyEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		if strings.ContainsRune(pySpecial, r) {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// Allowlist renders tinyproxy filter lines for the egress hosts.
func Allowlist(hosts []string) (string, error) {
	var lines []string
	for _, h := range hosts {
		if !hostRE.MatchString(h) {
			return "", fmt.Errorf("bad egress host %s", pyrepr.Str(h))
		}
		lines = append(lines, "^"+pyEscape(h)+"$")
	}
	return strings.Join(lines, "\n") + "\n", nil
}

func indent(text string, n int) string {
	pad := strings.Repeat(" ", n)
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	for i, l := range lines {
		if l != "" {
			lines[i] = pad + l
		}
	}
	return strings.Join(lines, "\n")
}

// RenderEgress renders an egress proxy (tinyproxy) for one client role and host allowlist.
func RenderEgress(l paths.Layout, ns, role string, hosts []string, image string) (string, error) {
	conf, err := os.ReadFile(filepath.Join(l.Images, "egress", "tinyproxy.conf.tmpl"))
	if err != nil {
		return "", err
	}
	al, err := Allowlist(hosts)
	if err != nil {
		return "", err
	}
	return k8s.Render(filepath.Join(l.Manifests, "egress.yaml.tmpl"), map[string]string{
		"NS": ns, "CLIENT_ROLE": role, "IMAGE": image, "CONF": indent(string(conf), 4), "ALLOWLIST": indent(al, 4)})
}

// FailOnPull fails closed with the image reference when a pull failed (no local-build fallback).
func FailOnPull(c *k8s.Client, ns, what string) error {
	if errs := c.PullErrors(ns); len(errs) > 0 {
		return fmt.Errorf("%s: image pull failed: %s. Check pilot/images.lock and registry access; no local-build fallback is attempted.", //nolint:staticcheck // verbatim reference message
			what, strings.Join(errs, "; "))
	}
	return fmt.Errorf("%s", what)
}

// ApplyAll applies the long-lived infrastructure: egress proxies, the publisher namespace and
// its SCM secret (from credentialEnv: the token or App token env file), and the check helpers.
func ApplyAll(e Env, c *k8s.Client, modelHosts []string, imgs map[string]string, credentialEnv string) error {
	e.say("egress proxies")
	for _, p := range []struct {
		ns, role string
		hosts    []string
	}{{"pilot-egress-model", "worker", modelHosts}, {"pilot-egress-scm", "publisher", []string{"github.com", "api.github.com"}}} {
		y, err := RenderEgress(e.L, p.ns, p.role, p.hosts, imgs["egress"])
		if err != nil {
			return err
		}
		if err := c.Apply(y); err != nil {
			return err
		}
	}
	for _, ns := range []string{"pilot-egress-model", "pilot-egress-scm"} {
		if !c.RolloutStatus(ns, "proxy", 180*time.Second) {
			return FailOnPull(c, ns, "egress proxy in "+ns+" did not become ready")
		}
	}
	e.say("publisher namespace")
	nsYAML, err := os.ReadFile(filepath.Join(e.L.Manifests, "publisher-ns.yaml.tmpl"))
	if err != nil {
		return err
	}
	if err := c.Apply(string(nsYAML)); err != nil {
		return err
	}
	// Replace, not delete+create: the namespace's quota allows one Secret, and a fresh
	// namespace's quota may not be computed yet (the create waits for it).
	if err := c.ReplaceSecretFromEnvFile("pilot-publish", "scm-token", credentialEnv); err != nil {
		return err
	}
	e.say("check helpers")
	infra, err := k8s.Render(filepath.Join(e.L.Manifests, "infra.yaml.tmpl"), map[string]string{"TOOLS_IMAGE": imgs["tools"]})
	if err != nil {
		return err
	}
	if err := c.Apply(infra); err != nil {
		return err
	}
	for _, p := range [][2]string{{"pilot-control", "control"}, {"pilot-grader-standin", "target"}} {
		if !c.WaitPodReady(p[0], p[1], 180*time.Second) {
			return FailOnPull(c, p[0], p[0]+"/"+p[1]+" not ready")
		}
	}
	return nil
}
