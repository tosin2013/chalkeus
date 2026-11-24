// Package hostinfo reads the machine forge runs on: its platform, its container runtime, the
// cluster node's platform and the platforms of the pinned images (031). doctor turns these into
// rows and refusals; evidence records them in versions.txt. Probes are read-only.
package hostinfo

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"
)

// Host is the machine forge runs on: the binary's own GOOS and GOARCH (R9), plus the macOS
// version when known.
type Host struct {
	OS        string // runtime.GOOS: darwin, linux, ...
	Arch      string // runtime.GOARCH: arm64, amd64, ...
	OSVersion string // macOS product version (sw_vers), "" elsewhere
}

// Platform is GOOS/GOARCH.
func (h Host) Platform() string { return h.OS + "/" + h.Arch }

// Runtime is the container runtime as `docker info` and `docker version` report it.
type Runtime struct {
	Err             string // non-empty when the daemon can't be reached
	OperatingSystem string // docker info .OperatingSystem: "Docker Desktop", "Ubuntu 24.04.1 LTS", "OrbStack", ...
	PlatformName    string // docker version .Server.Platform.Name: "Docker Desktop 4.94.0 (241994)", "Docker Engine - Community"
	ServerVersion   string // docker info .ServerVersion
	CPUs            int    // docker info .NCPU
	MemBytes        int64  // docker info .MemTotal
}

// DockerDesktop reports whether the runtime is Docker Desktop.
func (rt Runtime) DockerDesktop() bool {
	return strings.Contains(rt.OperatingSystem, "Docker Desktop") || strings.HasPrefix(rt.PlatformName, "Docker Desktop")
}

// Describe is the runtime's one-line summary used by doctor and versions.txt.
func (rt Runtime) Describe() string {
	name := rt.PlatformName
	if name == "" {
		name = rt.OperatingSystem
	}
	if name == "" {
		name = "Docker"
	}
	return fmt.Sprintf("%s (Engine %s), %d CPUs, %.1f GiB", name, rt.ServerVersion, rt.CPUs, float64(rt.MemBytes)/(1<<30))
}

// IndexPlatforms reads `docker buildx imagetools inspect --raw` output. For an image index it
// returns the image platforms, leaving out attestation manifests (unknown/unknown). For a
// single-platform manifest (a lock that predates 031) it returns index=false.
func IndexPlatforms(raw []byte) (platforms []string, index bool, err error) {
	var doc struct {
		Manifests []struct {
			Platform *struct {
				OS           string `json:"os"`
				Architecture string `json:"architecture"`
			} `json:"platform"`
		} `json:"manifests"`
		Config json.RawMessage `json:"config"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, false, fmt.Errorf("unreadable manifest: %w", err)
	}
	if doc.Manifests == nil {
		if doc.Config == nil {
			return nil, false, fmt.Errorf("neither an image index nor an image manifest")
		}
		return nil, false, nil
	}
	seen := map[string]bool{}
	for _, m := range doc.Manifests {
		if m.Platform == nil || m.Platform.OS == "unknown" {
			continue
		}
		seen[m.Platform.OS+"/"+m.Platform.Architecture] = true
	}
	for p := range seen {
		platforms = append(platforms, p)
	}
	sort.Strings(platforms)
	return platforms, true, nil
}

// ImageInfo is one locked image's platform check input.
type ImageInfo struct {
	Platforms []string
	Index     bool
	Err       error // the registry couldn't be read
}

// Command runs a probe command and returns its stdout (replaced in tests). Probes are read-only.
var Command = func(timeout time.Duration, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return exec.CommandContext(ctx, name, args...).Output()
}

// GOOS and GOARCH are this binary's platform (replaced in tests).
var GOOS, GOARCH = runtime.GOOS, runtime.GOARCH

// ProbeHost reads the host's platform and, on macOS, its version.
func ProbeHost() Host {
	h := Host{OS: GOOS, Arch: GOARCH}
	if h.OS == "darwin" {
		if out, err := Command(10*time.Second, "sw_vers", "-productVersion"); err == nil {
			h.OSVersion = strings.TrimSpace(string(out))
		}
	}
	return h
}

// ProbeRuntime reads `docker info` and `docker version`. An unreachable daemon sets Err.
func ProbeRuntime() Runtime {
	var rt Runtime
	out, err := Command(30*time.Second, "docker", "info", "--format", "{{json .}}")
	if err != nil {
		rt.Err = "docker info: " + err.Error()
		return rt
	}
	var info struct {
		OperatingSystem string `json:"OperatingSystem"`
		ServerVersion   string `json:"ServerVersion"`
		NCPU            int    `json:"NCPU"`
		MemTotal        int64  `json:"MemTotal"`
	}
	if err := json.Unmarshal(out, &info); err != nil || info.ServerVersion == "" {
		rt.Err = "docker info: no server information (is the daemon running?)"
		return rt
	}
	rt.OperatingSystem, rt.ServerVersion, rt.CPUs, rt.MemBytes = info.OperatingSystem, info.ServerVersion, info.NCPU, info.MemTotal
	if out, err := Command(30*time.Second, "docker", "version", "--format", "{{json .}}"); err == nil {
		var v struct {
			Server *struct {
				Platform struct {
					Name string `json:"Name"`
				} `json:"Platform"`
			} `json:"Server"`
		}
		if json.Unmarshal(out, &v) == nil && v.Server != nil {
			rt.PlatformName = v.Server.Platform.Name
		}
	}
	return rt
}

// ProbeNodePlatform reads the cluster node's os/arch through kubectl.
func ProbeNodePlatform(kubeContext string) (string, error) {
	out, err := Command(30*time.Second, "kubectl", "--context", kubeContext, "get", "nodes", "-o",
		"jsonpath={range .items[*]}{.status.nodeInfo.operatingSystem}/{.status.nodeInfo.architecture}{\"\\n\"}{end}")
	if err != nil {
		return "", err
	}
	seen := map[string]bool{}
	for _, l := range strings.Fields(string(out)) {
		seen[l] = true
	}
	ps := make([]string, 0, len(seen))
	for p := range seen {
		ps = append(ps, p)
	}
	sort.Strings(ps)
	return strings.Join(ps, ","), nil // a single-node kind cluster gives one value
}

// ProbeImages reads each locked image's platforms from its registry (`imagetools inspect --raw`).
// lock maps names (worker, tools, ...) to digest references.
func ProbeImages(lock map[string]string) map[string]ImageInfo {
	out := map[string]ImageInfo{}
	for name, ref := range lock {
		raw, err := Command(60*time.Second, "docker", "buildx", "imagetools", "inspect", "--raw", ref)
		if err != nil {
			out[name] = ImageInfo{Err: err}
			continue
		}
		ps, index, err := IndexPlatforms(raw)
		out[name] = ImageInfo{Platforms: ps, Index: index, Err: err}
	}
	return out
}
