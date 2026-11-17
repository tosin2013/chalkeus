package images

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/tosin2013/specfoundry/internal/config"
	"github.com/tosin2013/specfoundry/internal/paths"
)

// The toolchain catalog (014 R5, pilot/images/toolchains.json). The worker and tools images
// stay one each; a toolchain with an image is copied into pods by an init container.

// RequestForm is where a missing toolchain image is asked for (014 FR-012).
const RequestForm = "https://github.com/tosin2013/specfoundry/issues/new?template=toolchain-image.yml"

// Toolchain is one catalog entry.
type Toolchain struct {
	ImageKey *string           `json:"image_key"`
	Env      map[string]string `json:"env"`
	Proxy    string            `json:"proxy"`
	Worker   *struct {
		Home string            `json:"home"`
		Tmp  string            `json:"tmp"`
		Env  map[string]string `json:"env"`
	} `json:"worker"`
	Grader *struct {
		CPU             string `json:"cpu"`
		Memory          string `json:"memory"`
		DeadlineS       int    `json:"deadline_s"`
		CommandTimeoutS int    `json:"command_timeout_s"`
	} `json:"grader"`
	Cache *struct {
		Size          string `json:"size"`
		LocalRepoFlag string `json:"local_repo_flag"`
	} `json:"cache"`
}

// Catalog maps toolchain names to entries.
type Catalog map[string]Toolchain

// CatalogPath is pilot/images/toolchains.json.
func CatalogPath(l paths.Layout) string { return filepath.Join(l.Images, "toolchains.json") }

// LoadCatalog reads the toolchain catalog.
func LoadCatalog(l paths.Layout) (Catalog, error) {
	raw, err := os.ReadFile(CatalogPath(l))
	if err != nil {
		return nil, err
	}
	var c Catalog
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("%s: %v", CatalogPath(l), err)
	}
	return c, nil
}

// Names lists the catalog's toolchains, sorted.
func (c Catalog) Names() []string {
	out := make([]string, 0, len(c))
	for n := range c {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// ImageKeys are the lock keys of toolchain images, sorted (for the image workflow and checks).
func (c Catalog) ImageKeys() []string {
	var out []string
	for _, t := range c {
		if t.ImageKey != nil {
			out = append(out, *t.ImageKey)
		}
	}
	sort.Strings(out)
	return out
}

// Unavailable is TOOLCHAIN_UNAVAILABLE (exit 2): unknown toolchain, or no locked image.
func Unavailable(name, why string) error {
	return &config.Error{Code: 2, Msg: fmt.Sprintf("TOOLCHAIN_UNAVAILABLE: toolchain %q %s; request one: %s", name, why, RequestForm)}
}

// ToolchainImage returns the toolchain's pinned image ("" for a toolchain built into the
// worker and tools images), or TOOLCHAIN_UNAVAILABLE.
func ToolchainImage(c Catalog, lock map[string]string, name string) (string, error) {
	t, ok := c[name]
	if !ok {
		return "", Unavailable(name, "is not in pilot/images/toolchains.json (known: "+strings.Join(c.Names(), ", ")+")")
	}
	if t.ImageKey == nil {
		return "", nil
	}
	ref, err := ValidateRef(lock[*t.ImageKey])
	if err != nil {
		return "", Unavailable(name, "has no image digest in pilot/images.lock ("+*t.ImageKey+")")
	}
	return ref, nil
}

// CheckTarget checks a target's toolchain is available and that its overrides only raise the
// catalog defaults (014 data-model: overrides "may only raise the catalog default").
func CheckTarget(c Catalog, lock map[string]string, t config.Target) error {
	if _, err := ToolchainImage(c, lock, t.Toolchain); err != nil {
		return err
	}
	tc := c[t.Toolchain]
	low := func(key, why string) error {
		return &config.Error{Code: 2, Msg: fmt.Sprintf("TARGET_INVALID: targets/%s.env: %s %s", t.Name, key, why)}
	}
	if tc.Grader != nil {
		if t.GraderMemory != "" && quantityMi(t.GraderMemory) < quantityMi(tc.Grader.Memory) {
			return low("PILOT_GRADER_MEMORY", "may only raise the toolchain default "+tc.Grader.Memory)
		}
		if t.GraderDeadlineS != 0 && t.GraderDeadlineS < tc.Grader.DeadlineS {
			return low("PILOT_GRADER_DEADLINE_S", "may only raise the toolchain default "+strconv.Itoa(tc.Grader.DeadlineS))
		}
	}
	if tc.Cache != nil && t.CacheSize != "" && quantityMi(t.CacheSize) < quantityMi(tc.Cache.Size) {
		return low("PILOT_CACHE_SIZE", "may only raise the toolchain default "+tc.Cache.Size)
	}
	return nil
}

// quantityMi converts "2Gi"/"512Mi" to MiB (0 if malformed; config validates the shape).
func quantityMi(q string) int {
	switch {
	case strings.HasSuffix(q, "Gi"):
		n, _ := strconv.Atoi(strings.TrimSuffix(q, "Gi"))
		return n * 1024
	case strings.HasSuffix(q, "Mi"):
		n, _ := strconv.Atoi(strings.TrimSuffix(q, "Mi"))
		return n
	}
	return 0
}
