// Package cluster is `pilot up` / `pilot down`: kind + Calico + local registry + the pilot's
// long-lived infrastructure (reference: pilot/specfoundry_pilot/cluster.py and infra.py).
//
// kind and kubectl are the pinned binaries installed by test/isolation/tools.sh (versions from
// pilot/pins.env), so pins.env stays the single source of tool versions (016 research R4, as
// amended during implementation). Cluster objects go through client-go.
package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/tosin2013/specfoundry/internal/evidence"
	"github.com/tosin2013/specfoundry/internal/k8s"
	"github.com/tosin2013/specfoundry/internal/paths"
)

// Env carries what up/down need; Say prints a step line.
type Env struct {
	L   paths.Layout
	Out io.Writer
}

func (e Env) say(msg string) { fmt.Fprintf(e.Out, "== %s\n", msg) }

// Sh runs a program; check=false ignores a non-zero exit. stdin may be empty.
func Sh(timeout time.Duration, check bool, stdin string, env []string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	if env != nil {
		cmd.Env = append(os.Environ(), env...)
	}
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var so, se strings.Builder
	cmd.Stdout, cmd.Stderr = &so, &se
	err := cmd.Run()
	if err != nil && check {
		msg := strings.TrimSpace(se.String())
		if msg == "" {
			msg = strings.TrimSpace(so.String())
		}
		if len(msg) > 800 {
			msg = msg[:800]
		}
		return so.String(), fmt.Errorf("%s: %s", strings.Join(args[:min(3, len(args))], " "), msg)
	}
	return so.String(), nil
}

// Exists reports whether the kind cluster exists.
func Exists() bool {
	out, _ := Sh(60*time.Second, false, "", nil, "kind", "get", "clusters")
	for _, f := range strings.Fields(out) {
		if f == paths.Cluster {
			return true
		}
	}
	return false
}

func registryRunning() bool {
	out, _ := Sh(30*time.Second, false, "", nil, "docker", "inspect", "-f", "{{.State.Running}}", paths.RegistryName)
	return strings.TrimSpace(out) == "true"
}

// VerifyCalico refuses a CNI manifest whose sha256 differs from the pinned value (FR-007).
func VerifyCalico(path, expected string) error {
	if expected == "" {
		return fmt.Errorf("CALICO_SHA256 is not pinned in pilot/pins.env")
	}
	got, err := evidence.SHA256File(path)
	if err != nil {
		return err
	}
	if got != expected {
		return fmt.Errorf("Calico manifest hash mismatch: expected %s, got %s", expected, got)
	}
	return nil
}

func download(url, dst string) error {
	c := &http.Client{Timeout: 120 * time.Second}
	resp, err := c.Get(url)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: HTTP %d", url, resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, raw, 0o644)
}

// UpCluster installs the pinned tools, creates the kind cluster (and the local registry with
// --local-build), installs Calico from its pinned manifest and waits for the nodes.
func UpCluster(e Env, localBuild bool) error {
	pins, err := paths.ReadEnv(e.L.Pins)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(e.L.Build, 0o755); err != nil {
		return err
	}
	e.say("pinned tools")
	if _, err := Sh(600*time.Second, true, "", []string{"KIND_VERSION=" + pins["KIND_VERSION"], "KUBECTL_VERSION=" + pins["KUBECTL_VERSION"]},
		"bash", filepath.Join(e.L.Root, "test/isolation/tools.sh")); err != nil {
		return err
	}
	if localBuild {
		e.say("local registry (--local-build)")
		if !registryRunning() {
			_, _ = Sh(60*time.Second, false, "", nil, "docker", "rm", "-f", paths.RegistryName)
			if _, err := Sh(300*time.Second, true, "", nil, "docker", "run", "-d", "--restart=no", "--name", paths.RegistryName,
				"-p", "127.0.0.1:5001:5000", pins["REGISTRY_IMAGE"]); err != nil {
				return err
			}
		}
	}
	e.say("kind cluster " + paths.Cluster)
	if !Exists() {
		if _, err := Sh(900*time.Second, true, "", nil, "kind", "create", "cluster", "--config",
			filepath.Join(e.L.Pilot, "kind-config.yaml"), "--wait", "0s"); err != nil {
			return err
		}
	}
	if localBuild {
		node := paths.Cluster + "-control-plane"
		d := "/etc/containerd/certs.d/" + paths.RegistryHost
		if _, err := Sh(60*time.Second, true, "", nil, "docker", "exec", node, "mkdir", "-p", d); err != nil {
			return err
		}
		if _, err := Sh(60*time.Second, true, fmt.Sprintf("[host.\"http://%s:5000\"]\n", paths.RegistryName), nil,
			"docker", "exec", "-i", node, "cp", "/dev/stdin", d+"/hosts.toml"); err != nil {
			return err
		}
		_, _ = Sh(60*time.Second, false, "", nil, "docker", "network", "connect", "kind", paths.RegistryName)
	}
	e.say("calico " + pins["CALICO_VERSION"])
	calico := filepath.Join(e.L.Build, "calico.yaml")
	if err := download("https://raw.githubusercontent.com/projectcalico/calico/"+pins["CALICO_VERSION"]+"/manifests/calico.yaml", calico); err != nil {
		return err
	}
	if err := VerifyCalico(calico, pins["CALICO_SHA256"]); err != nil {
		return err
	}
	sum, _ := evidence.SHA256File(calico)
	if err := os.WriteFile(filepath.Join(e.L.Build, "calico.sha256"), []byte(sum+"\n"), 0o644); err != nil {
		return err
	}
	c, err := k8s.New(paths.Context)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(calico)
	if err != nil {
		return err
	}
	if err := c.Apply(string(raw)); err != nil {
		return err
	}
	if !c.DaemonSetReady("kube-system", "calico-node", 300*time.Second) {
		return fmt.Errorf("calico-node did not become ready")
	}
	if !c.RolloutStatus("kube-system", "calico-kube-controllers", 300*time.Second) {
		return fmt.Errorf("calico-kube-controllers did not become ready")
	}
	if !c.NodesReady(180 * time.Second) {
		return fmt.Errorf("nodes did not become Ready")
	}
	if !c.RolloutStatus("kube-system", "coredns", 180*time.Second) {
		return fmt.Errorf("coredns did not become ready")
	}
	return nil
}

// UpOK is the marker written when `pilot up` completes.
type UpOK struct {
	At          string            `json:"at"`
	Images      map[string]string `json:"images"`
	ImageSource string            `json:"image_source"`
}

// WriteUpOK writes pilot/.build/up.ok.
func WriteUpOK(l paths.Layout, imgs map[string]string, source string) error {
	raw, _ := json.Marshal(UpOK{At: evidence.Now(), Images: imgs, ImageSource: source})
	return os.WriteFile(l.UpOK, append(raw, '\n'), 0o644)
}

// TeardownRows checks T01–T03: cluster gone, no node containers, registry removed.
func TeardownRows() []evidence.CheckRow {
	pf := func(ok bool) string {
		if ok {
			return "pass"
		}
		return "fail"
	}
	var rows []evidence.CheckRow
	gone := !Exists()
	detail := "present"
	if gone {
		detail = "absent"
	}
	rows = append(rows, evidence.NewCheckRow("T01", "N3", pf(gone), "kind cluster deleted", detail))
	n, _ := Sh(60*time.Second, false, "", nil, "docker", "ps", "-a", "--filter", "label=io.x-k8s.kind.cluster="+paths.Cluster, "-q")
	nodes := len(strings.Fields(n))
	rows = append(rows, evidence.NewCheckRow("T02", "N3", pf(nodes == 0), "no kind node containers remain", fmt.Sprintf("containers=%d", nodes)))
	r, _ := Sh(60*time.Second, false, "", nil, "docker", "ps", "-a", "--filter", "name=^"+paths.RegistryName+"$", "-q")
	reg := len(strings.Fields(r))
	rows = append(rows, evidence.NewCheckRow("T03", "N3", pf(reg == 0), "local registry container removed", fmt.Sprintf("containers=%d", reg)))
	return rows
}

// Down deletes the cluster and registry, then records the teardown checks in the latest run
// (or pilot/.build/teardown.jsonl). It returns 0 when every check passes, else 1.
func Down(e Env) (int, error) {
	e.say("teardown")
	_, _ = Sh(300*time.Second, false, "", nil, "kind", "delete", "cluster", "--name", paths.Cluster)
	_, _ = Sh(60*time.Second, false, "", nil, "docker", "rm", "-f", paths.RegistryName)
	_ = os.Remove(e.L.UpOK)
	rows := TeardownRows()
	run, err := evidence.LoadRun(e.L.EvidenceRoot(), "")
	if err != nil {
		return 2, err
	}
	target := filepath.Join(e.L.Build, "teardown.jsonl")
	if run != nil {
		target = filepath.Join(run.Dir, "teardown.jsonl")
	}
	_ = os.Remove(target) // one teardown record per run: the latest
	pass := true
	for _, row := range rows {
		if err := evidence.AppendJSONL(target, row); err != nil {
			return 2, err
		}
		mark := "PASS"
		if row.Result != "pass" {
			mark, pass = "FAIL", false
		}
		fmt.Fprintf(e.Out, "  %s %s %s (%s)\n", mark, row.ID, row.Description, row.Detail)
	}
	if run != nil && run.State["state"] == "succeeded" {
		if err := run.Event("teardown", nil, nil, "torn_down", filepath.Base(target)); err != nil {
			return 2, err
		}
	}
	if pass {
		return 0, nil
	}
	return 1, nil
}
