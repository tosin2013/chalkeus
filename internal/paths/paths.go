// Package paths holds the pilot's well-known paths (reference: the former Python pilot's paths.py).
package paths

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const (
	Cluster       = "specfoundry-m1"
	Context       = "kind-" + Cluster
	RegistryName  = "specfoundry-registry"
	RegistryHost  = "localhost:5001"
	defaultFeat   = "specs/002-m1-gated-pilot"
	featureJSON   = ".specify/feature.json"
	evidenceEnvVr = "PILOT_EVIDENCE_FEATURE"
)

// Tasks maps the pilot's task labels to the fixture roadmap items behind them.
var Tasks = map[string]string{"A": "001", "B": "002"}

// SpecDirs maps task labels to the fixture spec directories.
var SpecDirs = map[string]string{"A": "specs/001-slugify", "B": "specs/002-unique-slug"}

// Root returns the repository root: $PILOT_ROOT, else the nearest ancestor of the working
// directory that contains pilot/pins.env.
func Root() (string, error) {
	if r := os.Getenv("PILOT_ROOT"); r != "" {
		return filepath.Abs(r)
	}
	d, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(d, "pilot", "pins.env")); err == nil {
			return d, nil
		}
		parent := filepath.Dir(d)
		if parent == d {
			return "", errors.New("not inside a specfoundry checkout (no pilot/pins.env above the working directory)")
		}
		d = parent
	}
}

// Layout is the set of paths derived from a root.
type Layout struct {
	Root, Pilot, Build, FixtureTemplate, Grading, Manifests, Images, Pins, ImagesLock string
	RunsDir, UpOK, ImagesJSON, ImageSource, FixtureBase, ModelHost                    string
}

// At returns the layout for a repository root.
func At(root string) Layout {
	p := filepath.Join(root, "pilot")
	b := filepath.Join(p, ".build")
	return Layout{
		Root: root, Pilot: p, Build: b,
		FixtureTemplate: filepath.Join(p, "fixture"), Grading: filepath.Join(p, "grading"),
		Manifests: filepath.Join(p, "manifests"), Images: filepath.Join(p, "images"),
		Pins: filepath.Join(p, "pins.env"), ImagesLock: filepath.Join(p, "images.lock"),
		RunsDir: filepath.Join(p, "runs"), UpOK: filepath.Join(b, "up.ok"),
		ImagesJSON: filepath.Join(b, "images.json"), ImageSource: filepath.Join(b, "image-source.txt"),
		FixtureBase: filepath.Join(b, "fixture-base.txt"), ModelHost: filepath.Join(b, "model-host.txt"),
	}
}

func activeFeature(root string) (string, bool) {
	raw, err := os.ReadFile(filepath.Join(root, featureJSON))
	if err != nil {
		return "", false
	}
	var fj struct {
		FeatureDirectory string `json:"feature_directory"`
	}
	if json.Unmarshal(raw, &fj) != nil || fj.FeatureDirectory == "" {
		return "", false
	}
	d := filepath.Join(root, fj.FeatureDirectory)
	if st, err := os.Stat(d); err == nil && st.IsDir() {
		return d, true
	}
	return "", false
}

// FeatureDir is the Spec Kit feature whose evidence folder receives runs:
// $PILOT_EVIDENCE_FEATURE, else the active feature, else specs/002-m1-gated-pilot.
func (l Layout) FeatureDir() string {
	if o := os.Getenv(evidenceEnvVr); o != "" {
		if filepath.IsAbs(o) { // as Python's ROOT / override with an absolute override
			return filepath.Clean(o)
		}
		return filepath.Join(l.Root, o)
	}
	if d, ok := activeFeature(l.Root); ok {
		return d
	}
	return filepath.Join(l.Root, defaultFeat)
}

// EvidenceRoot is where run evidence goes: <feature>/evidence for maintainers (override or an
// active feature), otherwise pilot/runs (operators; git-ignored).
func (l Layout) EvidenceRoot() string {
	if os.Getenv(evidenceEnvVr) != "" {
		return filepath.Join(l.FeatureDir(), "evidence")
	}
	if d, ok := activeFeature(l.Root); ok {
		return filepath.Join(d, "evidence")
	}
	return l.RunsDir
}

// ReadEnv reads KEY=VALUE lines, skipping blanks and comments (the reference _read_env).
func ReadEnv(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	out := map[string]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		k, v, _ := strings.Cut(line, "=")
		out[k] = v
	}
	return out, sc.Err()
}
