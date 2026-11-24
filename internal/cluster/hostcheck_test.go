package cluster

import (
	"bytes"
	"strings"
	"testing"

	"github.com/tosin2013/chalkeus/internal/hostinfo"
	"github.com/tosin2013/chalkeus/internal/paths"
)

// 031: `up` refuses an unsupported host or a stopped runtime before installing anything.

func TestHostCheck(t *testing.T) {
	var out bytes.Buffer
	e := Env{Out: &out}
	mac := hostinfo.Host{OS: "darwin", Arch: "arm64", OSVersion: "27.0.1"}
	dd := hostinfo.Runtime{OperatingSystem: "Docker Desktop", ServerVersion: "29.8.2", CPUs: 8, MemBytes: 8 << 30}
	if err := HostCheck(e, mac, dd); err != nil || out.Len() != 0 {
		t.Fatalf("supported: %v %q", err, out.String())
	}
	if err := HostCheck(e, mac, hostinfo.Runtime{Err: "docker info: exit status 1"}); err == nil || err.Error() != "Docker isn't running: start Docker Desktop" {
		t.Fatalf("stopped: %v", err)
	}
	if err := HostCheck(e, hostinfo.Host{OS: "windows", Arch: "amd64"}, dd); err == nil || !strings.HasPrefix(err.Error(), "unsupported host windows/amd64") {
		t.Fatalf("unsupported: %v", err)
	}
	out.Reset()
	if err := HostCheck(e, hostinfo.Host{OS: "linux", Arch: "arm64"}, hostinfo.Runtime{OperatingSystem: "Ubuntu", ServerVersion: "28"}); err != nil ||
		!strings.Contains(out.String(), "warning: linux/arm64: untested") {
		t.Fatalf("warning continues: %v %q", err, out.String())
	}
}

func TestUpClusterRefusesBeforeInstalling(t *testing.T) {
	old := ProbeHostAndRuntime
	defer func() { ProbeHostAndRuntime = old }()
	ProbeHostAndRuntime = func() (hostinfo.Host, hostinfo.Runtime) {
		return hostinfo.Host{OS: "darwin", Arch: "arm64"}, hostinfo.Runtime{Err: "docker info: exit status 1"}
	}
	var out bytes.Buffer
	// An empty layout: anything past the host check (pins, tools, kind) would fail differently.
	err := UpCluster(Env{L: paths.At(t.TempDir()), Out: &out}, false)
	if err == nil || err.Error() != "Docker isn't running: start Docker Desktop" {
		t.Fatalf("got %v", err)
	}
	if strings.Contains(out.String(), "pinned tools") {
		t.Fatalf("installed tools after a refusal: %q", out.String())
	}
}
