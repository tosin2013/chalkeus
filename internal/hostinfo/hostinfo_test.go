package hostinfo

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

// 031: the probes, with the commands faked (no Docker or kubectl needed).

func TestProbes(t *testing.T) {
	oldCmd, oldOS, oldArch := Command, GOOS, GOARCH
	defer func() { Command, GOOS, GOARCH = oldCmd, oldOS, oldArch }()
	GOOS, GOARCH = "darwin", "arm64"
	Command = func(_ time.Duration, name string, args ...string) ([]byte, error) {
		switch name + " " + strings.Join(args, " ") {
		case "sw_vers -productVersion":
			return []byte("27.0.1\n"), nil
		case "docker info --format {{json .}}":
			return []byte(`{"OperatingSystem":"Docker Desktop","ServerVersion":"29.8.2","NCPU":14,"MemTotal":8318976000}`), nil
		case "docker version --format {{json .}}":
			return []byte(`{"Server":{"Platform":{"Name":"Docker Desktop 4.94.0 (241994)"}}}`), nil
		}
		if name == "kubectl" {
			return []byte("linux/arm64\n"), nil
		}
		if name == "docker" && args[0] == "buildx" {
			if args[len(args)-1] == "r/t@sha256:b" {
				return nil, errors.New("denied")
			}
			return os.ReadFile("../doctor/testdata/index-busybox.json")
		}
		return nil, errors.New("unexpected " + name)
	}
	if h := ProbeHost(); h != (Host{OS: "darwin", Arch: "arm64", OSVersion: "27.0.1"}) {
		t.Errorf("host: %+v", h)
	}
	rt := ProbeRuntime()
	if rt.Err != "" || !rt.DockerDesktop() || rt.CPUs != 14 || rt.PlatformName != "Docker Desktop 4.94.0 (241994)" {
		t.Errorf("runtime: %+v", rt)
	}
	if p, err := ProbeNodePlatform("kind-chalkeus"); err != nil || p != "linux/arm64" {
		t.Errorf("node: %q %v", p, err)
	}
	imgs := ProbeImages(map[string]string{"worker": "r/w@sha256:a", "tools": "r/t@sha256:b"})
	if !imgs["worker"].Index || imgs["tools"].Err == nil {
		t.Errorf("images: %+v", imgs)
	}
	// A stopped daemon: docker info fails.
	Command = func(time.Duration, string, ...string) ([]byte, error) { return nil, errors.New("exit status 1") }
	if rt := ProbeRuntime(); rt.Err == "" {
		t.Error("stopped daemon must set Err")
	}
}
