// Package images resolves the pilot's container images: digest-only references from
// pilot/images.lock (C-09), or a local build into the kind registry (`up --local-build`)
// (reference: pilot/specfoundry_pilot/images.py and pilot/images/build_args.py).
package images

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/tosin2013/specfoundry/internal/paths"
	"github.com/tosin2013/specfoundry/internal/pyrepr"
)

// Names are the three pilot images, in the reference's order.
var Names = []string{"worker", "tools", "egress"}

var lockKeys = map[string]string{"worker": "WORKER_IMAGE", "tools": "TOOLS_IMAGE", "egress": "EGRESS_IMAGE"}

var refRE = regexp.MustCompile(`^[a-z0-9./:-]+@sha256:[0-9a-f]{64}$`)

// RefError is an image reference problem.
type RefError struct{ Msg string }

func (e *RefError) Error() string { return e.Msg }

// ValidateRef accepts only name@sha256:<64 lowercase hex>: never a tag (also not tag@digest).
func ValidateRef(ref string) (string, error) {
	if ref == "" || !refRE.MatchString(ref) {
		return "", &RefError{"not a digest reference: " + pyrepr.Str(ref)}
	}
	name, _, _ := strings.Cut(ref, "@")
	last := name[strings.LastIndex(name, "/")+1:]
	if strings.Contains(last, ":") { // a tag on the last path component (a registry port is allowed earlier)
		return "", &RefError{"tagged reference not allowed: " + pyrepr.Str(ref)}
	}
	return ref, nil
}

// LoadPinned validates the three references in a lock map.
func LoadPinned(lock map[string]string) (map[string]string, error) {
	out := map[string]string{}
	for _, short := range Names {
		key := lockKeys[short]
		ref, err := ValidateRef(lock[key])
		if err != nil {
			return nil, &RefError{fmt.Sprintf("%s in pilot/images.lock: %s", key, err)}
		}
		out[short] = ref
	}
	return out, nil
}

// ReadLock reads pilot/images.lock.
func ReadLock(l paths.Layout) (map[string]string, error) {
	m, err := paths.ReadEnv(l.ImagesLock)
	if os.IsNotExist(err) {
		return nil, &RefError{"missing " + l.ImagesLock}
	}
	return m, err
}

// Load reads the images chosen at `pilot up` (pilot/.build/images.json).
func Load(l paths.Layout) (map[string]string, error) {
	raw, err := os.ReadFile(l.ImagesJSON)
	if err != nil {
		return nil, err
	}
	var m map[string]string
	return m, json.Unmarshal(raw, &m)
}

// Save writes images.json.
func Save(l paths.Layout, imgs map[string]string) error {
	raw, _ := json.MarshalIndent(imgs, "", "  ")
	if err := os.MkdirAll(l.Build, 0o755); err != nil {
		return err
	}
	return os.WriteFile(l.ImagesJSON, append(raw, '\n'), 0o644)
}

// BuildArgs mirrors pilot/images/build_args.py (the image workflow keeps using that script).
func BuildArgs(image string, pins map[string]string) (map[string]string, error) {
	split := func(ref string) (string, string) { r, t, _ := strings.Cut(ref, "@"); return r, t }
	switch image {
	case "worker":
		repo, tag := split(pins["OPENCODE_RELEASE"])
		return map[string]string{"DEBIAN_BASE": pins["DEBIAN_BASE"], "OPENCODE_REPO": repo, "OPENCODE_TAG": tag,
			"OPENCODE_ASSET": pins["OPENCODE_ASSET"], "OPENCODE_ASSET_SHA256": pins["OPENCODE_ASSET_SHA256"],
			"OPENCODE_SHA256": pins["OPENCODE_SHA256"]}, nil
	case "tools":
		repo, tag := split(pins["SPEC_RELEASE"])
		return map[string]string{"DEBIAN_BASE": pins["DEBIAN_BASE"], "SPEC_REPO": repo, "SPEC_TAG": tag,
			"SPEC_ASSET": pins["SPEC_ASSET"], "SPEC_ASSET_SHA256": pins["SPEC_ASSET_SHA256"],
			"SPEC_SHA256": pins["SPEC_SHA256"], "SPEC_CHECKSUMS": "spec_" + strings.TrimPrefix(tag, "v") + "_checksums.txt"}, nil
	case "egress":
		return map[string]string{"ALPINE_BASE": pins["ALPINE_BASE"]}, nil
	}
	return nil, fmt.Errorf("unknown image %s", pyrepr.Str(image))
}

// FramePy is where the image build finds frame.py (both images COPY it).
func FramePy(l paths.Layout) string {
	if p := filepath.Join(l.Images, "common", "frame.py"); exists(p) {
		return p
	}
	return filepath.Join(l.Pilot, "specfoundry_pilot", "frame.py")
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		t := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(t, 0o755)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(t, raw, info.Mode().Perm())
	})
}

// contextDir prepares the build context for one image, as the reference _context does.
func contextDir(l paths.Layout, name string) (string, error) {
	ctx := filepath.Join(l.Build, "ctx-"+name)
	_ = os.RemoveAll(ctx)
	if err := copyDir(filepath.Join(l.Images, name), ctx); err != nil {
		return "", err
	}
	raw, err := os.ReadFile(FramePy(l))
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(ctx, "frame.py"), raw, 0o644); err != nil {
		return "", err
	}
	_ = os.Remove(filepath.Join(ctx, "build_args.py"))
	return ctx, nil
}

// hashDir is the content tag: build args plus every file's name and bytes, in path order.
func hashDir(ctx string, args map[string]string) (string, error) {
	h := sha256.New()
	keys := make([]string, 0, len(args))
	for k := range args {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	om := pyrepr.OrderedMap{Keys: keys, Values: map[string]any{}}
	for _, k := range keys {
		om.Values[k] = args[k]
	}
	h.Write([]byte(pyrepr.JSONDumps(om))) // json.dumps(args, sort_keys=True), byte for byte
	var files []string
	_ = filepath.WalkDir(ctx, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			files = append(files, p)
		}
		return nil
	})
	// Python's sorted(Path.rglob()) compares path components, not whole strings
	// ("a/x" sorts before "a-b"); match it.
	sort.Slice(files, func(i, j int) bool {
		a, b := strings.Split(files[i], string(filepath.Separator)), strings.Split(files[j], string(filepath.Separator))
		for k := 0; k < len(a) && k < len(b); k++ {
			if a[k] != b[k] {
				return a[k] < b[k]
			}
		}
		return len(a) < len(b)
	})
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			return "", err
		}
		h.Write([]byte(filepath.Base(f)))
		h.Write(raw)
	}
	return hex.EncodeToString(h.Sum(nil))[:12], nil
}

func sh(args ...string) (string, error) {
	out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
	if err != nil {
		s := strings.TrimSpace(string(out))
		if len(s) > 1500 {
			s = s[len(s)-1500:]
		}
		return "", fmt.Errorf("%s: %s", strings.Join(args[:min(3, len(args))], " "), s)
	}
	return string(out), nil
}

// Build builds one image into the local registry and returns its digest reference.
func Build(l paths.Layout, name string, args map[string]string, say func(string)) (string, error) {
	ctx, err := contextDir(l, name)
	if err != nil {
		return "", err
	}
	tag, err := hashDir(ctx, args)
	if err != nil {
		return "", err
	}
	ref := fmt.Sprintf("%s/pilot-%s:%s", paths.RegistryHost, name, tag)
	cmd := []string{"docker", "build", "-q", "-t", ref}
	keys := make([]string, 0, len(args))
	for k := range args {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		cmd = append(cmd, "--build-arg", k+"="+args[k])
	}
	say(fmt.Sprintf("== image pilot-%s:%s", name, tag))
	if _, err := sh(append(cmd, ctx)...); err != nil {
		return "", err
	}
	if _, err := sh("docker", "push", "-q", ref); err != nil {
		return "", err
	}
	out, err := sh("docker", "inspect", "--format", "{{range .RepoDigests}}{{println .}}{{end}}", ref)
	if err != nil {
		return "", err
	}
	prefix := fmt.Sprintf("%s/pilot-%s@sha256:", paths.RegistryHost, name)
	for _, line := range strings.Fields(out) {
		if strings.HasPrefix(line, prefix) {
			return line, nil
		}
	}
	return "", fmt.Errorf("no registry digest for %s", ref)
}

// BuildAll builds the three images (`pilot up --local-build`) and saves images.json.
func BuildAll(l paths.Layout, say func(string)) (map[string]string, error) {
	pins, err := paths.ReadEnv(l.Pins)
	if err != nil {
		return nil, err
	}
	imgs := map[string]string{}
	for _, n := range Names {
		args, err := BuildArgs(n, pins)
		if err != nil {
			return nil, err
		}
		if imgs[n], err = Build(l, n, args, say); err != nil {
			return nil, err
		}
	}
	return imgs, Save(l, imgs)
}
