package images

import (
	"strings"
	"testing"

	"github.com/tosin2013/chalkeus/internal/config"
	"github.com/tosin2013/chalkeus/internal/names"
)

// 030 (contracts/profile-keys.md): the worker's budget is the catalog's, raised by the profile.
func TestResolveWorkerBudget(t *testing.T) {
	c := catalog(t)
	tc := c["node24"]
	b := Resolve(tc, "node24", config.Target{Toolchain: "node24"})
	if b.Worker.CPU != "2" || b.Worker.Memory != "2Gi" || b.Worker.DeadlineS != 1200 || len(b.Raised) != 0 || b.Source != "catalog:node24" {
		t.Fatalf("defaults: %+v", b)
	}
	if b.Worker.SoftDeadlineS() != 1200-SoftMarginS {
		t.Fatalf("soft deadline %d", b.Worker.SoftDeadlineS())
	}
	b = Resolve(tc, "node24", config.Target{Toolchain: "node24", WorkerCPU: "3", WorkerMemory: "4Gi", WorkerDeadlineS: 2400})
	if b.Worker.CPU != "3" || b.Worker.Memory != "4Gi" || b.Worker.DeadlineS != 2400 || b.Worker.SoftDeadlineS() != 2280 {
		t.Fatalf("raised: %+v", b)
	}
	if strings.Join(b.Raised, ",") != "FORGE_WORKER_CPU,FORGE_WORKER_MEMORY,FORGE_WORKER_DEADLINE_S" {
		t.Fatalf("raised keys: %v", b.Raised)
	}
	// A value equal to the default isn't a raise.
	if b = Resolve(tc, "node24", config.Target{WorkerDeadlineS: 1200}); len(b.Raised) != 0 {
		t.Fatalf("%v", b.Raised)
	}
	if f := Fixture(); f.Worker.DeadlineS != 1200 || f.Worker.SoftDeadlineS() != 1080 {
		t.Fatalf("fixture: %+v", f)
	}
}

func TestCheckTargetWorkerFloors(t *testing.T) {
	c := catalog(t)
	lock := map[string]string{"TOOLCHAIN_NODE24_IMAGE": "r/x@sha256:" + strings.Repeat("b", 64)}
	if err := CheckTarget(c, lock, config.Target{Name: "g", Toolchain: "node24", WorkerCPU: "2500m", WorkerMemory: "3Gi", WorkerDeadlineS: 2400}); err != nil {
		t.Fatal(err)
	}
	for key, tg := range map[string]config.Target{
		"PILOT_WORKER_CPU":        {Name: "g", Toolchain: "node24", WorkerCPU: "1500m"},
		"PILOT_WORKER_MEMORY":     {Name: "g", Toolchain: "node24", WorkerMemory: "1Gi"},
		"PILOT_WORKER_DEADLINE_S": {Name: "g", Toolchain: "node24", WorkerDeadlineS: 600},
		"PILOT_AUTHOR_DEADLINE_S": {Name: "g", Toolchain: "node24", AuthorDeadlineS: 600},
	} {
		err := CheckTarget(c, lock, tg)
		if err == nil || !strings.Contains(err.Error(), names.Display(key)) || !strings.Contains(err.Error(), "may only raise") {
			t.Errorf("%s: %v", key, err)
		}
	}
}

func TestCPUMilli(t *testing.T) {
	for q, want := range map[string]int{"2": 2000, "1.5": 1500, "3500m": 3500, "x": 0} {
		if got := cpuMilli(q); got != want {
			t.Errorf("%s: %d", q, got)
		}
	}
}

// 030 T021: a target may raise the test author's deadline, never lower it.
func TestAuthorDeadline(t *testing.T) {
	if got := AuthorDeadline(config.Target{}); got != 1200 {
		t.Fatalf("default %d", got)
	}
	if got := AuthorDeadline(config.Target{AuthorDeadlineS: 2400}); got != 2400 {
		t.Fatalf("raised %d", got)
	}
}
