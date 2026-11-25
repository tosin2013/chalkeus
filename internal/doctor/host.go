package doctor

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/tosin2013/chalkeus/internal/cnitoken"
	"github.com/tosin2013/chalkeus/internal/hostinfo"
)

// The host types and probes live in internal/hostinfo, so evidence can use them too.
type (
	Host      = hostinfo.Host
	Runtime   = hostinfo.Runtime
	ImageInfo = hostinfo.ImageInfo
)

var (
	IndexPlatforms    = hostinfo.IndexPlatforms
	ProbeHost         = hostinfo.ProbeHost
	ProbeRuntime      = hostinfo.ProbeRuntime
	ProbeNodePlatform = hostinfo.ProbeNodePlatform
	ProbeImages       = hostinfo.ProbeImages
)

// 031: the host area (specs/031-macos-multiarch/contracts/doctor.md). The functions here are
// pure: probe.go gathers the inputs, so every row and status is unit-tested without Docker.

// Supported lists the hosts forge recognises, for refusal messages.
const Supported = "darwin/arm64, darwin/amd64, linux/amd64 (linux/arm64 untested)"

// HostRow is the "host" row (R9 table).
func HostRow(h Host) (string, string) {
	name := h.Platform()
	if h.OS == "darwin" && h.OSVersion != "" {
		name += " (macOS " + h.OSVersion + ")"
	}
	switch h.Platform() {
	case "darwin/arm64", "linux/amd64":
		return Pass, name + ", validated"
	case "darwin/amd64":
		return Pass, name + ", supported by design, not maintainer-validated"
	case "linux/arm64":
		return Warn, name + ": untested; the images run, but no maintainer evidence exists"
	}
	return Fail, "unsupported host " + h.Platform() + "; supported: " + Supported
}

// RuntimeRow is the "container runtime" row.
func RuntimeRow(h Host, rt Runtime) (string, string) {
	if rt.Err != "" {
		if h.OS == "darwin" {
			return Fail, "Docker isn't running: start Docker Desktop"
		}
		return Fail, "Docker isn't running: start the Docker daemon"
	}
	if h.OS == "darwin" && !rt.DockerDesktop() {
		return Warn, rt.Describe() + ": only Docker Desktop is validated on macOS; the isolation suite and audit still decide"
	}
	return Pass, rt.Describe()
}

// SpecRow is the "spec-status" row: the gate runs the host's spec-status, so it should be the
// pinned release (pilot/pins.env SPEC_VERSION). version is the first line of `spec --version`
// ("spec 0.4.1"), or "" when spec isn't installed.
func SpecRow(version, pinned string) (string, string) {
	if version == "" {
		return Fail, "spec-status isn't installed: install release v" + pinned + " (docs/operator-guide.md, step 4)"
	}
	got := strings.TrimSpace(strings.TrimPrefix(version, "spec "))
	if f := strings.Fields(got); len(f) > 0 {
		got = f[0]
	}
	if got == strings.TrimPrefix(pinned, "v") {
		return Pass, "spec " + got + ", the pinned release"
	}
	return Warn, "spec " + got + " on this host, pinned " + pinned + ": the gate runs the host's copy; install the pinned release"
}

// NetworkTokenWarn is how much time left on Calico's CNI token earns a warning (031 T034): about
// one attempt with its grading.
const NetworkTokenWarn = 2 * time.Hour

// NetworkTokenRow is the "network token" row (after `up`): the time left on Calico's CNI token.
// An expired or short token isn't a failure: each attempt restarts calico-node first when the
// token won't last it. err is a failure to read the token's expiry.
func NetworkTokenRow(left time.Duration, err error) (string, string) {
	const fix = "each attempt restarts calico-node first; to refresh it now: kubectl --context kind-chalkeus-m1 -n kube-system rollout restart ds/calico-node (docs/troubleshooting.md#calico-token-expired)"
	switch {
	case err != nil:
		return Warn, "couldn't read Calico's CNI token: " + err.Error()
	case left <= 0:
		return Warn, "Calico's CNI token has expired, so new pods can't start; " + fix
	case left < NetworkTokenWarn:
		return Warn, "Calico's CNI token expires in " + cnitoken.Left(left) + "; " + fix
	}
	return Pass, "Calico's CNI token expires in " + cnitoken.Left(left)
}

// NodePlatformRow is the "node platform" row (after `up`).
func NodePlatformRow(p string) (string, string) {
	if p == "linux/amd64" || p == "linux/arm64" {
		return Pass, p
	}
	return Fail, fmt.Sprintf("node platform %q isn't linux/amd64 or linux/arm64", p)
}

// ImagePlatformsRow is the "image platforms" row: every pinned image must have a build for the
// cluster node's platform.
func ImagePlatformsRow(source, node string, images map[string]ImageInfo) (string, string) {
	if source == "local-build" {
		return Warn, "local build: images built for this host (`./chalkeus forge up --local-build`)"
	}
	names := make([]string, 0, len(images))
	for n := range images {
		names = append(names, n)
	}
	sort.Strings(names)
	var missing, single, unread []string
	for _, n := range names {
		info := images[n]
		switch {
		case info.Err != nil:
			unread = append(unread, n+": "+info.Err.Error())
		case !info.Index:
			single = append(single, n)
		case !contains(info.Platforms, node):
			missing = append(missing, n)
		}
	}
	if len(missing) > 0 {
		return Fail, fmt.Sprintf("%s: no build for %s (pilot/images.lock may predate 031); pull main and run `./chalkeus forge up`", strings.Join(missing, ", "), node)
	}
	if len(single) > 0 && node != "linux/amd64" {
		return Fail, fmt.Sprintf("%s: single-platform image, no build for %s (pilot/images.lock predates 031); pull main and run `./chalkeus forge up`", strings.Join(single, ", "), node)
	}
	if len(unread) > 0 {
		return Warn, "couldn't check image platforms: " + strings.Join(unread, "; ")
	}
	if len(single) > 0 {
		return Warn, fmt.Sprintf("%d single-platform image(s) (pilot/images.lock predates 031); they run on linux/amd64 only", len(single))
	}
	return Pass, fmt.Sprintf("all %d images have %s", len(names), node)
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

// HostRows adds the "host" area's first two rows (both are also `up`'s early refusals).
func (r *Report) HostRows(h Host, rt Runtime) {
	st, det := HostRow(h)
	r.Add("host", "host", st, det)
	st, det = RuntimeRow(h, rt)
	r.Add("host", "container runtime", st, det)
}

// LockImages maps pilot/images.lock's *_IMAGE entries to image names (WORKER_IMAGE → worker,
// TOOLCHAIN_PYTHON313_IMAGE → toolchain_python313); the registry line is not an image.
func LockImages(lock map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range lock {
		if strings.HasSuffix(k, "_IMAGE") && v != "" {
			out[strings.ToLower(strings.TrimSuffix(k, "_IMAGE"))] = v
		}
	}
	return out
}
