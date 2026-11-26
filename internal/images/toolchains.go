package images

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/tosin2013/chalkeus/internal/config"
	"github.com/tosin2013/chalkeus/internal/names"
	"github.com/tosin2013/chalkeus/internal/paths"
)

// Toolchains (014 R5, contracts/toolchain-image.md). The worker and tools images stay ours, one
// each. A toolchain image only carries /opt/toolchain (with its own toolchain.json), copied into
// the worker, prefetch and grader pods by an init container. It comes from the catalog
// (pilot/images/toolchains.json, built by this repository's workflow) or is supplied by the
// operator in the target profile (PILOT_TOOLCHAIN_IMAGE, digest-pinned; owner decision
// 2026-11-17).

// RequestForm is where a missing toolchain image is asked for (014 FR-012).
const RequestForm = "https://github.com/tosin2013/chalkeus/issues/new?template=toolchain-image.yml"

// Toolchain is one catalog entry.
type Toolchain struct {
	ImageKey *string `json:"image_key"`
	// Proxy is how the package manager reaches the registry proxy: "env" (HTTPS_PROXY: npm, pip,
	// Go, cargo, …) or "maven-settings" (a generated settings.xml). An operator-supplied image
	// declares it in its toolchain.json.
	Proxy  string `json:"proxy"`
	Worker *struct {
		Home string            `json:"home"`
		Tmp  string            `json:"tmp"`
		Env  map[string]string `json:"env"`
		// 030: the worker pod's limits and attempt deadline; omitted = DefaultWorker.
		CPU       string `json:"cpu,omitempty"`
		Memory    string `json:"memory,omitempty"`
		DeadlineS int    `json:"deadline_s,omitempty"`
	} `json:"worker"`
	Grader *struct {
		CPU             string `json:"cpu"`
		Memory          string `json:"memory"`
		DeadlineS       int    `json:"deadline_s"`
		CommandTimeoutS int    `json:"command_timeout_s"`
	} `json:"grader"`
	Cache *struct {
		Size string `json:"size"`
	} `json:"cache"`
	// OperatorSupplied is set for a target's own image (not from the catalog).
	OperatorSupplied bool `json:"-"`
}

// operatorDefaults are the resources an operator-supplied toolchain gets; the profile may raise
// them, as for catalog toolchains.
const operatorDefaults = `{"image_key": null, "proxy": "env",
 "worker": {"home": "1Gi", "tmp": "512Mi", "env": {}},
 "grader": {"cpu": "2", "memory": "2Gi", "deadline_s": 900, "command_timeout_s": 600},
 "cache": {"size": "2Gi"}}`

// For returns the toolchain a target uses: its own image's defaults, or the catalog entry.
func (c Catalog) For(t config.Target) (Toolchain, bool) {
	if t.ToolchainImage != "" {
		var tc Toolchain
		_ = json.Unmarshal([]byte(operatorDefaults), &tc)
		tc.OperatorSupplied = true
		return tc, true
	}
	tc, ok := c[t.Toolchain]
	return tc, ok
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
	if t.ToolchainImage != "" {
		if _, ok := c[t.Toolchain]; ok {
			return &config.Error{Code: 2, Msg: fmt.Sprintf("TARGET_INVALID: targets/%s.env: FORGE_TOOLCHAIN %q is a Chalkeus toolchain; "+
				"with FORGE_TOOLCHAIN_IMAGE use your own name for it, or drop FORGE_TOOLCHAIN_IMAGE", t.Name, t.Toolchain)}
		}
		if _, err := ValidateRef(t.ToolchainImage); err != nil {
			return &config.Error{Code: 2, Msg: fmt.Sprintf("TARGET_INVALID: targets/%s.env: FORGE_TOOLCHAIN_IMAGE %v", t.Name, err)}
		}
	} else if _, err := ToolchainImage(c, lock, t.Toolchain); err != nil {
		return err
	}
	tc, _ := c.For(t)
	low := func(key, why string) error {
		return &config.Error{Code: 2, Msg: fmt.Sprintf("TARGET_INVALID: targets/%s.env: %s %s", t.Name, names.Display(key), why)}
	}
	if tc.Grader != nil {
		if t.GraderMemory != "" && quantityMi(t.GraderMemory) < quantityMi(tc.Grader.Memory) {
			return low("PILOT_GRADER_MEMORY", "may only raise the toolchain default "+tc.Grader.Memory)
		}
		if t.GraderDeadlineS != 0 && t.GraderDeadlineS < tc.Grader.DeadlineS {
			return low("PILOT_GRADER_DEADLINE_S", "may only raise the toolchain default "+strconv.Itoa(tc.Grader.DeadlineS))
		}
	}
	wd := WorkerDefaults(tc)
	if t.WorkerCPU != "" && cpuMilli(t.WorkerCPU) < cpuMilli(wd.CPU) {
		return low("PILOT_WORKER_CPU", "may only raise the toolchain default "+wd.CPU)
	}
	if t.WorkerMemory != "" && quantityMi(t.WorkerMemory) < quantityMi(wd.Memory) {
		return low("PILOT_WORKER_MEMORY", "may only raise the toolchain default "+wd.Memory)
	}
	if t.WorkerDeadlineS != 0 && t.WorkerDeadlineS < wd.DeadlineS {
		return low("PILOT_WORKER_DEADLINE_S", "may only raise the toolchain default "+strconv.Itoa(wd.DeadlineS))
	}
	if t.AuthorDeadlineS != 0 && t.AuthorDeadlineS < AuthorDeadlineS {
		return low("PILOT_AUTHOR_DEADLINE_S", "may only raise the default "+strconv.Itoa(AuthorDeadlineS))
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

// AuthorDeadlineS is the test author Job's default active deadline in seconds (the worker's
// default); a target may raise it with FORGE_AUTHOR_DEADLINE_S (030 T021).
const AuthorDeadlineS = 1200

// AuthorDeadline is a target's author Job deadline: the default, raised by its profile.
func AuthorDeadline(t config.Target) int {
	if t.AuthorDeadlineS > AuthorDeadlineS {
		return t.AuthorDeadlineS
	}
	return AuthorDeadlineS
}
