package evidence

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/tosin2013/chalkeus/internal/hostinfo"
)

// 031: versions.txt's host and platform lines (specs/031-macos-multiarch/data-model.md).

func TestHostLines(t *testing.T) {
	got := HostLines(hostinfo.Host{OS: "darwin", Arch: "arm64", OSVersion: "27.0.1"},
		hostinfo.Runtime{OperatingSystem: "Docker Desktop", PlatformName: "Docker Desktop 4.94.0 (241994)", ServerVersion: "29.8.2", CPUs: 14, MemBytes: 8318976000},
		"linux/arm64", []string{"egress", "tools", "worker"})
	want := []string{
		"host_os: darwin 27.0.1",
		"host_arch: arm64",
		"runtime: Docker Desktop 4.94.0 (241994) (Engine 29.8.2), 14 CPUs, 7.7 GiB",
		"node_platform: linux/arm64",
		"image_egress_platform: linux/arm64",
		"image_tools_platform: linux/arm64",
		"image_worker_platform: linux/arm64",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if got := HostLines(hostinfo.Host{OS: "linux", Arch: "amd64"}, hostinfo.Runtime{Err: "docker info: exit status 1"}, "unknown", nil); got[0] != "host_os: linux" ||
		got[2] != "runtime: unreachable (docker info: exit status 1)" {
		t.Fatalf("linux, no runtime: %v", got)
	}
}

func TestWriteVersionsPicksTheNodesChecksums(t *testing.T) {
	old := hostinfo.Command
	defer func() { hostinfo.Command = old }()
	pins := map[string]string{"OPENCODE_VERSION": "2.2.7", "OPENCODE_SHA256_AMD64": "amd-oc", "OPENCODE_SHA256_ARM64": "arm-oc",
		"SPEC_SHA256_AMD64": "amd-spec", "SPEC_SHA256_ARM64": "arm-spec", "CALICO_VERSION": "v3.29.3", "REGISTRY_IMAGE": "registry:2@sha256:x"}
	for _, c := range []struct{ node, oc, spec string }{{"linux/arm64", "arm-oc", "arm-spec"}, {"linux/amd64", "amd-oc", "amd-spec"}} {
		hostinfo.Command = func(_ time.Duration, name string, args ...string) ([]byte, error) {
			if name == "kubectl" {
				return []byte(c.node + "\n"), nil
			}
			return nil, errors.New("not faked")
		}
		dir := t.TempDir()
		r := &Run{ID: "20990101T000000Z", Dir: dir, State: map[string]any{}}
		if err := WriteVersions(r, pins, t.TempDir(), "kind-x", "x", map[string]string{"worker": "w@sha256:a"}); err != nil {
			t.Fatal(err)
		}
		raw, _ := os.ReadFile(filepath.Join(dir, "versions.txt"))
		out := string(raw)
		for _, want := range []string{
			"opencode: 2.2.7 sha256=" + c.oc,
			"sha256=" + c.spec,
			"image_worker: w@sha256:a", // existing lines are unchanged
			"node_platform: " + c.node,
			"image_worker_platform: " + c.node,
			"host_arch: ",
		} {
			if !strings.Contains(out, want) {
				t.Errorf("%s: missing %q in\n%s", c.node, want, out)
			}
		}
		if strings.Index(out, "host: ") > strings.Index(out, "host_os: ") {
			t.Errorf("%s: the new lines must follow host:\n%s", c.node, out)
		}
	}
}
