//go:build integration

// Package integration runs the pilot end to end on kind against the fake GitHub, with the stub
// harness in place of the model (025). Run it with:
//
//	go test -tags integration -timeout 45m ./test/integration/
//
// It never touches the operator's own setup: the pilot runs from a temporary copy of pilot/
// (PILOT_ROOT) with a temporary config folder (SPECFOUNDRY_CONFIG), and it refuses to start if
// a pilot cluster already exists, because it creates and deletes its own.
package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/tosin2013/specfoundry/internal/evidence"
	"github.com/tosin2013/specfoundry/internal/fakegh"
	"github.com/tosin2013/specfoundry/internal/paths"
)

const (
	fixtureRepo = "owner/specfoundry-pilot-fixture"
	ownerLogin  = "owner"
	botLogin    = "pilot-bot"
	ownerToken  = "owner-token"
	botToken    = "bot-token"
)

// world is the shared setup: one fake GitHub and one `pilot up` for every scenario.
type world struct {
	repo    string // this checkout
	root    string // the pilot's temporary root (a copy of pilot/)
	config  string // the temporary config folder
	bin     string // the pilot binary
	out     string // test/integration/out: evidence and logs kept for CI artifacts
	fake    *fakegh.Server
	api     string // the fake's address as kind's pods see it
	local   string // the fake's address from this machine
	baseEnv []string
}

var w *world

func TestMain(m *testing.M) {
	var err error
	w, err = setUp()
	if err != nil {
		fmt.Fprintln(os.Stderr, "integration setup:", err)
		if w != nil {
			w.tearDown(true)
		}
		os.Exit(1)
	}
	code := m.Run()
	w.tearDown(code != 0)
	os.Exit(code)
}

func setUp() (*world, error) {
	for _, tool := range []string{"docker", "git", "spec", "go"} { // kind and kubectl: `pilot up` installs its pinned ones
		if _, err := exec.LookPath(tool); err != nil {
			return nil, fmt.Errorf("%s is not installed", tool)
		}
	}
	home, _ := os.UserHomeDir()
	kind := filepath.Join(home, ".local", "bin", "kind")
	if _, err := os.Stat(kind); err != nil {
		kind = "kind"
	}
	if out, _ := exec.Command(kind, "get", "clusters").Output(); strings.Contains(string(out), paths.Cluster) {
		return nil, fmt.Errorf("a %s cluster already exists; the integration tests create and delete their own, so run `pilot/pilot down` first", paths.Cluster)
	}
	repo, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		return nil, err
	}
	tmp, err := os.MkdirTemp("", "specfoundry-it-")
	if err != nil {
		return nil, err
	}
	x := &world{repo: repo, root: filepath.Join(tmp, "root"), config: filepath.Join(tmp, "config"),
		bin: filepath.Join(tmp, "bin", "pilot"), out: filepath.Join(repo, "test", "integration", "out")}
	_ = os.RemoveAll(x.out)
	if err := os.MkdirAll(x.out, 0o755); err != nil {
		return x, err
	}
	say("building the pilot")
	if out, err := run(repo, nil, "go", "build", "-o", x.bin, "./cmd/pilot"); err != nil {
		return x, fmt.Errorf("go build: %v\n%s", err, out)
	}
	if err := copyPilot(filepath.Join(repo, "pilot"), filepath.Join(x.root, "pilot")); err != nil {
		return x, err
	}
	// `pilot up` installs kind and kubectl with this script, from its root.
	if err := copyPilot(filepath.Join(repo, "test", "isolation"), filepath.Join(x.root, "test", "isolation")); err != nil {
		return x, err
	}
	gw, err := kindGateway()
	if err != nil {
		return x, err
	}
	if err := x.startFake(gw, filepath.Join(tmp, "fake")); err != nil {
		return x, err
	}
	if err := x.writeConfig(); err != nil {
		return x, err
	}
	// notes: automatic drafting (T011), at most one author Job per draft.
	if err := x.addTarget("notes", "PILOT_AUTHOR_AUTO=on\nPILOT_AUTHOR_AUTO_ATTEMPTS=1\n"); err != nil {
		return x, err
	}
	// tally: step mode (T012), with GitHub issues and approvals and automatic drafting.
	if err := x.addTarget("tally", "PILOT_AUTHOR_AUTO=on\nPILOT_AUTHOR_AUTO_ATTEMPTS=1\nPILOT_GITHUB_ISSUES=on\nPILOT_GITHUB_APPROVALS=on\n"); err != nil {
		return x, err
	}
	// gaps: step mode's debrief gaps (T013): a non-approver's label, a failing doctor.
	if err := x.addTarget("gaps", "PILOT_GITHUB_ISSUES=on\nPILOT_GITHUB_APPROVALS=on\n"); err != nil {
		return x, err
	}
	x.baseEnv = []string{"PILOT_ROOT=" + x.root, "SPECFOUNDRY_CONFIG=" + x.config, "PILOT_TEST_MODE=1",
		"PILOT_TEST_GITHUB_API=" + x.api, "PILOT_TEST_GIT_BASE=" + x.api + "/git/", "GH_TOKEN=" + ownerToken, "PILOT_NO_TTY=1",
		// The pilot commits the status change with the operator's own git identity; a CI runner
		// has none.
		"GIT_AUTHOR_NAME=" + ownerLogin, "GIT_AUTHOR_EMAIL=owner@example.invalid",
		"GIT_COMMITTER_NAME=" + ownerLogin, "GIT_COMMITTER_EMAIL=owner@example.invalid"}
	say("pilot up --local-build (several minutes)")
	if out, code := x.pilot("up", "--local-build"); code != 0 {
		return x, fmt.Errorf("pilot up exited %d:\n%s", code, tail(out, 60))
	}
	return x, nil
}

// kindGateway is the IPv4 gateway of Docker's `kind` network (created here if missing, as kind
// would), the address kind's pods use to reach this machine.
func kindGateway() (string, error) {
	if _, err := run("", nil, "docker", "network", "inspect", "kind"); err != nil {
		if out, err := run("", nil, "docker", "network", "create", "kind"); err != nil {
			return "", fmt.Errorf("docker network create kind: %v\n%s", err, out)
		}
	}
	out, err := run("", nil, "docker", "network", "inspect", "kind", "-f", "{{range .IPAM.Config}}{{.Gateway}} {{end}}")
	if err != nil {
		return "", fmt.Errorf("docker network inspect kind: %v\n%s", err, out)
	}
	for _, g := range strings.Fields(out) {
		if ip := net.ParseIP(g); ip != nil && ip.To4() != nil {
			return g, nil
		}
	}
	return "", fmt.Errorf("the kind network has no IPv4 gateway (%q)", strings.TrimSpace(out))
}

// startFake serves the fake GitHub on every interface, seeded with the owner, the publisher bot
// and the fixture repository from pilot/fixture, and records the fixture's base revision.
func (x *world) startFake(gw, dir string) error {
	x.fake = fakegh.New(dir)
	// Port 443: the pilot's egress proxies only let traffic out to port 443, and the tests keep
	// that rule as it is. Binding it needs root or net.ipv4.ip_unprivileged_port_start<=443 (the
	// workflow sets it).
	ln, err := net.Listen("tcp", "0.0.0.0:443")
	if err != nil {
		return fmt.Errorf("the fake GitHub must listen on port 443 (the egress proxies allow only 443): %v; "+
			"run `sudo sysctl -w net.ipv4.ip_unprivileged_port_start=443` first", err)
	}
	x.api, x.local = fmt.Sprintf("http://%s:443", gw), "http://127.0.0.1:443"
	go func() { _ = http.Serve(ln, fakegh.NewMux(x.fake)) }()
	x.fake.AddUser(ownerToken, ownerLogin)
	x.fake.AddUser(botToken, botLogin)
	if err := x.fake.AddRepo(fixtureRepo, filepath.Join(x.root, "pilot", "fixture"),
		map[string]string{ownerLogin: "admin", botLogin: "write"}); err != nil {
		return err
	}
	out, err := run("", nil, "git", "ls-remote", x.local+"/git/"+fixtureRepo+".git", "refs/heads/main")
	if err != nil || len(strings.Fields(out)) == 0 {
		return fmt.Errorf("the fake's fixture has no main: %v %s", err, out)
	}
	build := filepath.Join(x.root, "pilot", ".build")
	if err := os.MkdirAll(build, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(build, "fixture-base.txt"), []byte(strings.Fields(out)[0]+"\n"), 0o644)
}

// writeConfig writes a throwaway config: the fixture on the fake, GitHub approvals by the owner,
// the bot's token for the publisher, and a stub model whose key is a dummy (the stub harness never
// calls a model). The model host is a public site only so the P06 probe has something to reach.
func (x *world) writeConfig() error {
	files := map[string]string{
		"pilot.env": "PILOT_FIXTURE_REPO=" + fixtureRepo + "\nPILOT_MODEL=stub/stub\nPILOT_MODEL_EGRESS_HOSTS=example.com\n" +
			"PILOT_GITHUB_ISSUES=on\nPILOT_GITHUB_APPROVALS=on\nPILOT_APPROVERS=" + ownerLogin + "\nPILOT_PUBLISHER_CREDENTIAL=token\n",
		"publisher.env": "PILOT_SCM_TOKEN=" + botToken + "\n",
		// doctor's model check calls providers.<p>.settings.baseURL: the fake's stand-in model.
		"opencode/opencode.json": `{"model":"stub/stub","providers":{"stub":{"settings":{"baseURL":"` + x.local +
			`/model/v1","apiKey":"{file:/run/secrets/model/stub_api_key}"}}}}` + "\n",
		"opencode/stub_api_key": "not-a-real-key\n",
	}
	for _, d := range []string{x.config, filepath.Join(x.config, "opencode")} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return err
		}
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(x.config, name), []byte(body), 0o600); err != nil {
			return err
		}
	}
	return nil
}

// addTarget seeds a test target and its companion on the fake from test/integration/fixtures/
// (<name> and <name>-acceptance) and writes its profile and publisher token (the bot's). The target
// uses the built-in python toolchain, so no toolchain image or package registry is needed.
func (x *world) addTarget(name, extra string) error {
	repo, companion := "owner/"+name, "owner/"+name+"-acceptance"
	perms := map[string]string{ownerLogin: "admin", botLogin: "write"}
	fx := filepath.Join(x.repo, "test", "integration", "fixtures")
	if err := x.fake.AddRepo(repo, filepath.Join(fx, name), perms); err != nil {
		return err
	}
	if err := x.fake.AddRepo(companion, filepath.Join(fx, name+"-acceptance"), map[string]string{ownerLogin: "admin"}); err != nil {
		return err
	}
	dir := filepath.Join(x.config, "targets")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	profile := "PILOT_TARGET_REPO=" + repo + "\nPILOT_TARGET_BRANCH=main\nPILOT_TOOLCHAIN=python\n" +
		"PILOT_TEST_COMMAND=python3 -m pytest -q tests/\nPILOT_COMPANION_REPO=" + companion + "\n" + extra
	if err := os.WriteFile(filepath.Join(dir, name+".env"), []byte(profile), 0o600); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, name+".publisher.env"), []byte("PILOT_SCM_TOKEN="+botToken+"\n"), 0o600)
}

// copyPilot copies pilot/ without its build and run output.
func copyPilot(src, dst string) error {
	return filepath.WalkDir(src, func(p string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		if e.IsDir() && (rel == ".build" || rel == "runs") {
			return filepath.SkipDir
		}
		t := filepath.Join(dst, rel)
		if e.IsDir() {
			return os.MkdirAll(t, 0o755)
		}
		if !e.Type().IsRegular() {
			return nil
		}
		info, err := e.Info()
		if err != nil {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(t, b, info.Mode().Perm())
	})
}

// tearDown keeps the evidence and logs (always in out/, which CI uploads on failure) and
// deletes the cluster.
func (x *world) tearDown(failed bool) {
	if x.root != "" {
		_ = copyTree(filepath.Join(x.root, "pilot", "runs"), filepath.Join(x.out, "runs"))
	}
	if failed {
		_, _ = run("", nil, "kind", "export", "logs", filepath.Join(x.out, "kind"), "--name", paths.Cluster)
	}
	if x.bin != "" && x.baseEnv != nil {
		say("pilot down")
		if out, code := x.pilot("down"); code != 0 {
			fmt.Fprintf(os.Stderr, "pilot down exited %d:\n%s\n", code, tail(out, 30))
		}
	}
}

// pilot runs a pilot command from the temporary root and returns its output and exit code.
func (x *world) pilot(args ...string) (string, int) {
	c := exec.Command(x.bin, args...)
	c.Dir = x.root
	c.Env = append(cleanEnv(), x.baseEnv...)
	var b bytes.Buffer
	c.Stdout, c.Stderr = &b, &b
	err := c.Run()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		code = -1
		b.WriteString("\n" + err.Error())
	}
	return b.String(), code
}

// cleanEnv is this process's environment without GitHub credentials or pilot settings, so the
// operator's own never reach a test run.
func cleanEnv() []string {
	var out []string
	for _, kv := range os.Environ() {
		k, _, _ := strings.Cut(kv, "=")
		if k == "GH_TOKEN" || k == "GITHUB_TOKEN" || strings.HasPrefix(k, "PILOT_") || k == "SPECFOUNDRY_CONFIG" {
			continue
		}
		out = append(out, kv)
	}
	return out
}

// status is `pilot status --run id` as JSON.
func (x *world) status(t *testing.T, id string) map[string]any {
	t.Helper()
	out, code := x.pilot("status", "--run", id)
	var st map[string]any
	if code != 0 || json.Unmarshal([]byte(out), &st) != nil {
		t.Fatalf("pilot status exited %d: %s", code, tail(out, 20))
	}
	return st
}

// runDir is a run's evidence folder in the temporary root.
func (x *world) runDir(id string) string { return filepath.Join(x.root, "pilot", "runs", id) }

var runIDRE = regexp.MustCompile(`== run (\S+)`)

func runID(t *testing.T, out string) string {
	t.Helper()
	m := runIDRE.FindStringSubmatch(out)
	if m == nil {
		t.Fatalf("no run id in:\n%s", tail(out, 40))
	}
	return m[1]
}

func run(dir string, env []string, name string, args ...string) (string, error) {
	c := exec.Command(name, args...)
	c.Dir = dir
	c.Env = append(os.Environ(), env...)
	b, err := c.CombinedOutput()
	return string(b), err
}

func copyTree(src, dst string) error {
	if _, err := os.Stat(src); err != nil {
		return nil
	}
	return copyPilot(src, dst)
}

func tail(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

func say(s string) { fmt.Fprintf(os.Stderr, "[integration %s] %s\n", time.Now().Format("15:04:05"), s) }

// readRows reads a JSONL file (nil, nil when it doesn't exist yet).
func readRows(p string) ([]map[string]any, error) {
	if _, err := os.Stat(p); err != nil {
		return nil, nil
	}
	rows, err := evidence.ReadJSONL(p)
	return rows, err
}

func filepathJoin(parts ...string) string { return filepath.Join(parts...) }
