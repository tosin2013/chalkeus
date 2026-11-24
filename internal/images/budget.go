package images

import (
	"strconv"
	"strings"

	"github.com/tosin2013/chalkeus/internal/config"
)

// 030 (specs/030-resource-budgets, data-model.md): a run's resource budget per pod role, from the
// toolchain catalog raised by the target profile. It is resolved once when a run starts, stored in
// run.json and read back on resume, so a later profile change never alters a run in flight.

// SoftMarginS is how long before the worker Job's deadline the worker's watchdog stops the agent,
// so the change and usage can still be collected (research R1). Not a profile key.
const SoftMarginS = 120

// DefaultWorker is the worker's budget when the catalog doesn't set one (today's fixed values).
var DefaultWorker = Role{CPU: "2", Memory: "2Gi", DeadlineS: 1200}

// Role is one pod role's limits and time.
type Role struct {
	CPU             string `json:"cpu"`
	Memory          string `json:"memory"`
	DeadlineS       int    `json:"deadline_s"`
	CommandTimeoutS int    `json:"command_timeout_s,omitempty"`
	SoftMarginS     int    `json:"soft_margin_s,omitempty"`
}

// SoftDeadlineS is the worker's soft deadline (seconds after the harness starts), or 0 when the
// deadline is too short to keep a margin.
func (r Role) SoftDeadlineS() int {
	if r.DeadlineS-r.SoftMarginS < 60 || r.SoftMarginS == 0 {
		return 0
	}
	return r.DeadlineS - r.SoftMarginS
}

// Budget is a run's resolved budget.
type Budget struct {
	Worker Role     `json:"worker"`
	Raised []string `json:"raised"`
	Source string   `json:"source"`
}

// WorkerDefaults is the worker's catalog budget for a toolchain.
func WorkerDefaults(tc Toolchain) Role {
	r := DefaultWorker
	if w := tc.Worker; w != nil {
		if w.CPU != "" {
			r.CPU = w.CPU
		}
		if w.Memory != "" {
			r.Memory = w.Memory
		}
		if w.DeadlineS > 0 {
			r.DeadlineS = w.DeadlineS
		}
	}
	return r
}

// Resolve returns a target's budget: the catalog's, raised by the profile. CheckTarget has
// already refused values below the floors.
func Resolve(tc Toolchain, toolchain string, t config.Target) Budget {
	w := WorkerDefaults(tc)
	b := Budget{Raised: []string{}, Source: "catalog:" + toolchain}
	if tc.OperatorSupplied {
		b.Source = "operator:" + toolchain
	}
	if t.WorkerCPU != "" && t.WorkerCPU != w.CPU {
		w.CPU = t.WorkerCPU
		b.Raised = append(b.Raised, "FORGE_WORKER_CPU")
	}
	if t.WorkerMemory != "" && t.WorkerMemory != w.Memory {
		w.Memory = t.WorkerMemory
		b.Raised = append(b.Raised, "FORGE_WORKER_MEMORY")
	}
	if t.WorkerDeadlineS > 0 && t.WorkerDeadlineS != w.DeadlineS {
		w.DeadlineS = t.WorkerDeadlineS
		b.Raised = append(b.Raised, "FORGE_WORKER_DEADLINE_S")
	}
	w.SoftMarginS = SoftMarginS
	b.Worker = w
	return b
}

// Fixture is the budget of a fixture run (no target profile): the defaults.
func Fixture() Budget {
	w := DefaultWorker
	w.SoftMarginS = SoftMarginS
	return Budget{Worker: w, Raised: []string{}, Source: "default"}
}

// cpuMilli converts a CPU quantity ("2", "1.5", "3500m") to millicores (0 if malformed).
func cpuMilli(q string) int {
	if strings.HasSuffix(q, "m") {
		n, _ := strconv.Atoi(strings.TrimSuffix(q, "m"))
		return n
	}
	f, err := strconv.ParseFloat(q, 64)
	if err != nil {
		return 0
	}
	return int(f * 1000)
}
