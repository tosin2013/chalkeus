package images

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tosin2013/chalkeus/internal/paths"
)

// Ported from the former Python pilot's test_pins.py, plus BuildArgs against pilot/images/build_args.py.

var d = strings.Repeat("a", 64)

func TestValidateRef(t *testing.T) {
	for _, ok := range []string{"ghcr.io/tosin2013/specfoundry-pilot-worker@sha256:" + d, "localhost:5001/pilot-worker@sha256:" + d} {
		if _, err := ValidateRef(ok); err != nil {
			t.Errorf("%s: %v", ok, err)
		}
	}
	for _, bad := range []string{"ghcr.io/x/y:latest", "ghcr.io/x/y:sha-abc", "", "ghcr.io/x/y@sha256:" + strings.Repeat("A", 64),
		"ghcr.io/x/y@sha256:" + strings.Repeat("a", 63), "ghcr.io/x/y:tag@sha256:" + d} {
		if _, err := ValidateRef(bad); err == nil {
			t.Errorf("accepted %q", bad)
		}
	}
}

func TestLoadPinned(t *testing.T) {
	good := map[string]string{"WORKER_IMAGE": "ghcr.io/o/w@sha256:" + d, "TOOLS_IMAGE": "ghcr.io/o/t@sha256:" + d, "EGRESS_IMAGE": "ghcr.io/o/e@sha256:" + d}
	m, err := LoadPinned(good)
	if err != nil || len(m) != 3 || m["worker"] != good["WORKER_IMAGE"] {
		t.Fatalf("%v %v", m, err)
	}
	for _, key := range []string{"WORKER_IMAGE", "TOOLS_IMAGE", "EGRESS_IMAGE"} {
		bad := map[string]string{}
		for k, v := range good {
			bad[k] = v
		}
		bad[key] = "ghcr.io/o/x:latest"
		if _, err := LoadPinned(bad); err == nil || !strings.Contains(err.Error(), key) {
			t.Errorf("%s: %v", key, err)
		}
	}
	empty := map[string]string{"WORKER_IMAGE": "", "TOOLS_IMAGE": good["TOOLS_IMAGE"], "EGRESS_IMAGE": good["EGRESS_IMAGE"]}
	if _, err := LoadPinned(empty); err == nil {
		t.Error("empty placeholder accepted")
	}
}

func TestReadLockMissing(t *testing.T) {
	l := paths.At(t.TempDir())
	if _, err := ReadLock(l); err == nil {
		t.Fatal("missing lock must be an error")
	}
	_ = os.MkdirAll(filepath.Dir(l.ImagesLock), 0o755)
	_ = os.WriteFile(l.ImagesLock, []byte("WORKER_IMAGE=ghcr.io/o/w@sha256:"+d+"\n"), 0o644)
	if m, err := ReadLock(l); err != nil || m["WORKER_IMAGE"] != "ghcr.io/o/w@sha256:"+d {
		t.Fatalf("%v %v", m, err)
	}
}

func TestBuildArgsMatchPythonScript(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not installed")
	}
	pins, err := paths.ReadEnv("../../pilot/pins.env")
	if err != nil {
		t.Fatal(err)
	}
	for _, img := range Names {
		out, err := exec.Command("python3", "../../pilot/images/build_args.py", img).Output()
		if err != nil {
			t.Fatalf("%s: %v", img, err)
		}
		want := map[string]string{}
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			k, v, _ := strings.Cut(line, "=")
			want[k] = v
		}
		got, _ := BuildArgs(img, pins)
		if len(got) != len(want) {
			t.Errorf("%s: %v vs %v", img, got, want)
		}
		for k, v := range want {
			if got[k] != v {
				t.Errorf("%s %s: %q != %q", img, k, got[k], v)
			}
		}
	}
}

func TestBuildArgsMissingPin(t *testing.T) {
	pins, err := paths.ReadEnv("../../pilot/pins.env")
	if err != nil {
		t.Fatal(err)
	}
	for _, img := range Names {
		args, err := BuildArgs(img, pins)
		if err != nil {
			t.Fatalf("%s: %v", img, err)
		}
		for k := range args {
			if _, ok := pins[k]; !ok {
				continue // derived: repo, tag, checksum file name
			}
			cut := map[string]string{}
			for pk, pv := range pins {
				if pk != k {
					cut[pk] = pv
				}
			}
			if _, err := BuildArgs(img, cut); err == nil || err.Error() != "pins.env: missing "+k {
				t.Errorf("%s without %s: %v", img, k, err)
			}
		}
	}
	// 031: the old single-arch keys are gone, so nothing can read a stale amd64-only value.
	for _, k := range []string{"OPENCODE_ASSET", "OPENCODE_ASSET_SHA256", "OPENCODE_SHA256", "SPEC_ASSET", "SPEC_ASSET_SHA256", "SPEC_SHA256"} {
		if _, ok := pins[k]; ok {
			t.Errorf("pins.env still has %s", k)
		}
	}
}

func TestHashDirStable(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a"), []byte("x"), 0o644)
	h1, _ := hashDir(dir, map[string]string{"K": "v"})
	h2, _ := hashDir(dir, map[string]string{"K": "v"})
	h3, _ := hashDir(dir, map[string]string{"K": "w"})
	if h1 != h2 || h1 == h3 || len(h1) != 12 {
		t.Fatal(h1, h2, h3)
	}
}

// The hash the former Python pilot computed for the same tree (frozen when it was removed, 016 US5).
func TestHashDirIsStable(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "sub"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM x\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "sub", "f.py"), []byte("print(1)\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "sub-b"), []byte("z"), 0o644) // "sub/f.py" vs "sub-b": component order
	if got, _ := hashDir(dir, map[string]string{"B": "2", "A": "1"}); got != "a0ed6c99954d" {
		t.Fatalf("hash %s", got)
	}
}
