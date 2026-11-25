package doctor

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
)

// 031: every row and status of specs/031-macos-multiarch/contracts/doctor.md.

func TestHostRow(t *testing.T) {
	for _, c := range []struct {
		h      Host
		status string
		want   string
	}{
		{Host{OS: "darwin", Arch: "arm64", OSVersion: "27.0.1"}, Pass, "darwin/arm64 (macOS 27.0.1), validated"},
		{Host{OS: "linux", Arch: "amd64", OSVersion: ""}, Pass, "linux/amd64, validated"},
		{Host{OS: "darwin", Arch: "amd64", OSVersion: "15.4"}, Pass, "supported by design, not maintainer-validated"},
		{Host{OS: "linux", Arch: "arm64", OSVersion: ""}, Warn, "untested"},
		{Host{OS: "windows", Arch: "amd64", OSVersion: ""}, Fail, "unsupported host windows/amd64; supported: " + Supported},
		{Host{OS: "linux", Arch: "riscv64", OSVersion: ""}, Fail, "unsupported host linux/riscv64"},
	} {
		st, det := HostRow(c.h)
		if st != c.status || !strings.Contains(det, c.want) {
			t.Errorf("%v: %s %q, want %s containing %q", c.h, st, det, c.status, c.want)
		}
	}
}

func TestRuntimeRow(t *testing.T) {
	mac, linux := Host{OS: "darwin", Arch: "arm64"}, Host{OS: "linux", Arch: "amd64"}
	dd := Runtime{OperatingSystem: "Docker Desktop", PlatformName: "Docker Desktop 4.94.0 (241994)", ServerVersion: "29.8.2", CPUs: 14, MemBytes: 8318976000}
	if st, det := RuntimeRow(mac, dd); st != Pass || det != "Docker Desktop 4.94.0 (241994) (Engine 29.8.2), 14 CPUs, 7.7 GiB" {
		t.Errorf("docker desktop: %s %q", st, det)
	}
	orb := Runtime{OperatingSystem: "OrbStack", ServerVersion: "28.0.0", CPUs: 8, MemBytes: 8 << 30}
	if st, det := RuntimeRow(mac, orb); st != Warn || !strings.Contains(det, "only Docker Desktop is validated on macOS") {
		t.Errorf("other runtime on macOS: %s %q", st, det)
	}
	engine := Runtime{OperatingSystem: "Ubuntu 24.04.1 LTS", PlatformName: "Docker Engine - Community", ServerVersion: "28.1.1", CPUs: 4, MemBytes: 16 << 30}
	if st, _ := RuntimeRow(linux, engine); st != Pass {
		t.Errorf("docker engine on linux: %s", st)
	}
	down := Runtime{Err: "docker info: exit status 1"}
	if st, det := RuntimeRow(mac, down); st != Fail || det != "Docker isn't running: start Docker Desktop" {
		t.Errorf("stopped on macOS: %s %q", st, det)
	}
	if st, det := RuntimeRow(linux, down); st != Fail || det != "Docker isn't running: start the Docker daemon" {
		t.Errorf("stopped on linux: %s %q", st, det)
	}
}

func TestNodePlatformRow(t *testing.T) {
	for p, want := range map[string]string{"linux/amd64": Pass, "linux/arm64": Pass, "linux/s390x": Fail, "": Fail} {
		if st, _ := NodePlatformRow(p); st != want {
			t.Errorf("%q: %s, want %s", p, st, want)
		}
	}
}

func TestIndexPlatforms(t *testing.T) {
	read := func(name string) []byte {
		raw, err := os.ReadFile("testdata/" + name)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	// A multi-platform official image (recorded 2026-11-24).
	ps, index, err := IndexPlatforms(read("index-busybox.json"))
	if err != nil || !index || !contains(ps, "linux/amd64") || !contains(ps, "linux/arm64") || contains(ps, "unknown/unknown") {
		t.Fatalf("busybox: %v %v %v", ps, index, err)
	}
	// Today's published worker: an index (provenance makes it one) with linux/amd64 only.
	ps, index, err = IndexPlatforms(read("index-pre031-worker.json"))
	if err != nil || !index || !reflect.DeepEqual(ps, []string{"linux/amd64"}) {
		t.Fatalf("pre-031 worker: %v %v %v", ps, index, err)
	}
	// A plain image manifest (no attestations).
	if ps, index, err := IndexPlatforms([]byte(`{"schemaVersion":2,"config":{"digest":"sha256:x"},"layers":[]}`)); err != nil || index || ps != nil {
		t.Fatalf("manifest: %v %v %v", ps, index, err)
	}
	if _, _, err := IndexPlatforms([]byte(`{"schemaVersion":2}`)); err == nil {
		t.Fatal("neither index nor manifest must be an error")
	}
	if _, _, err := IndexPlatforms([]byte(`nope`)); err == nil {
		t.Fatal("garbage must be an error")
	}
}

func TestImagePlatformsRow(t *testing.T) {
	both := ImageInfo{Platforms: []string{"linux/amd64", "linux/arm64"}, Index: true}
	amdOnly := ImageInfo{Platforms: []string{"linux/amd64"}, Index: true}
	single := ImageInfo{}
	unread := ImageInfo{Err: errors.New("denied")}

	if st, det := ImagePlatformsRow("pinned", "linux/arm64", map[string]ImageInfo{"worker": both, "tools": both}); st != Pass || det != "all 2 images have linux/arm64" {
		t.Errorf("all good: %s %q", st, det)
	}
	if st, det := ImagePlatformsRow("pinned", "linux/arm64", map[string]ImageInfo{"worker": both, "tools": amdOnly}); st != Fail ||
		!strings.HasPrefix(det, "tools: no build for linux/arm64 (pilot/images.lock may predate 031)") {
		t.Errorf("missing arm64: %s %q", st, det)
	}
	if st, _ := ImagePlatformsRow("pinned", "linux/amd64", map[string]ImageInfo{"worker": amdOnly}); st != Pass {
		t.Errorf("pre-031 lock on amd64 must pass: %s", st)
	}
	if st, _ := ImagePlatformsRow("pinned", "linux/arm64", map[string]ImageInfo{"worker": single}); st != Fail {
		t.Errorf("single manifest on arm64: %s", st)
	}
	if st, _ := ImagePlatformsRow("pinned", "linux/amd64", map[string]ImageInfo{"worker": single}); st != Warn {
		t.Errorf("single manifest on amd64: %s", st)
	}
	if st, det := ImagePlatformsRow("pinned", "linux/arm64", map[string]ImageInfo{"worker": both, "tools": unread}); st != Warn ||
		!strings.Contains(det, "couldn't check image platforms: tools: denied") {
		t.Errorf("unreadable: %s %q", st, det)
	}
	// A missing build outranks an unreadable registry.
	if st, _ := ImagePlatformsRow("pinned", "linux/arm64", map[string]ImageInfo{"worker": amdOnly, "tools": unread}); st != Fail {
		t.Errorf("missing + unreadable: %s", st)
	}
	if st, det := ImagePlatformsRow("local-build", "linux/arm64", nil); st != Warn || !strings.Contains(det, "local build") {
		t.Errorf("local build: %s %q", st, det)
	}
}

func TestHostRowsAndLockImages(t *testing.T) {
	var r Report
	r.HostRows(Host{OS: "darwin", Arch: "arm64", OSVersion: "27.0.1"}, Runtime{Err: "docker info: exit status 1"})
	if len(r.Rows) != 2 || r.Rows[0].Area != "host" || r.Rows[0].Status != Pass || r.Rows[1].Check != "container runtime" || !r.Failed() {
		t.Fatalf("rows: %+v", r.Rows)
	}
	got := LockImages(map[string]string{
		"IMAGE_REGISTRY":            "ghcr.io/tosin2013",
		"WORKER_IMAGE":              "ghcr.io/x/chalkeus-worker@sha256:" + strings.Repeat("a", 64),
		"TOOLCHAIN_PYTHON313_IMAGE": "ghcr.io/x/chalkeus-toolchain-python3.13@sha256:" + strings.Repeat("b", 64),
		"EMPTY_IMAGE":               "",
	})
	if len(got) != 2 || got["worker"] == "" || got["toolchain_python313"] == "" {
		t.Fatalf("lock images: %v", got)
	}
}

func TestSpecRow(t *testing.T) {
	for _, c := range []struct{ version, pinned, st, has string }{
		{"spec 0.4.1", "0.4.1", Pass, "pinned release"},
		{"spec 0.4.1", "v0.4.1", Pass, "pinned release"},
		{"spec 0.3.0", "0.4.1", Warn, "spec 0.3.0 on this host, pinned 0.4.1"},
		{"", "0.4.1", Fail, "isn't installed"},
	} {
		st, det := SpecRow(c.version, c.pinned)
		if st != c.st || !strings.Contains(det, c.has) {
			t.Errorf("SpecRow(%q, %q) = %s %q", c.version, c.pinned, st, det)
		}
	}
}
