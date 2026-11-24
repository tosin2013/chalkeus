package pipeline

import (
	"strings"
	"testing"

	"github.com/tosin2013/chalkeus/internal/config"
	"github.com/tosin2013/chalkeus/internal/paths"
)

// 030 FR-006: the budget is resolved once, stored in run.json, and kept on resume even if the
// profile loses its raises; the mode shows the raised worker deadline.
func TestRunBudgetStoredAndKept(t *testing.T) {
	p, r, _ := newP(t)
	p.L = paths.At(repoRoot(t))
	p.Target = config.Target{Name: "documcp", Repo: "me/documcp", Toolchain: "node24", WorkerDeadlineS: 2400, WorkerMemory: "4Gi"}
	b := p.runBudget(r)
	if b.Worker.DeadlineS != 2400 || b.Worker.Memory != "4Gi" || b.Worker.SoftDeadlineS() != 2280 {
		t.Fatalf("%+v", b)
	}
	if got := modeJSON(p.ModeOf()); !strings.Contains(got, `"worker_deadline_s": 2400`) {
		t.Fatal(got)
	}
	if l := p.workerLimits(r); l.DeadlineS != 2400 || l.SoftDeadlineS != 2280 || l.Memory != "4Gi" || l.CPU != "2" {
		t.Fatalf("%+v", l)
	}
	ev, _ := r.Events()
	if ev[len(ev)-1]["event"] != "budget_resolved" {
		t.Fatalf("%v", ev[len(ev)-1])
	}

	// Resume in a new process, with the raises removed from the profile.
	q, _, _ := newP(t)
	q.L, q.Target = p.L, config.Target{Name: "documcp", Repo: "me/documcp", Toolchain: "node24"}
	if b := q.runBudget(r); b.Worker.DeadlineS != 2400 || b.Worker.Memory != "4Gi" {
		t.Fatalf("resume must keep the run's budget: %+v", b)
	}
}

func TestFixtureBudgetIsDefault(t *testing.T) {
	p, r, _ := newP(t)
	if b := p.runBudget(r); b.Worker.DeadlineS != 1200 || b.Source != "default" {
		t.Fatalf("%+v", b)
	}
}

// 030 FR-017: a retry after a deadline stop says what the stopped attempt used and changed.
func TestDeadlineNote(t *testing.T) {
	_, r, _ := newP(t)
	r.State["mode"] = map[string]any{"worker_deadline_s": 2400}
	r.State["tasks"] = map[string]any{"001": map[string]any{"attempts": []any{
		map[string]any{"n": 1.0, "reason": "deadline", "spend": map[string]any{"tokens": 812345.0, "cost_usd": 0.42},
			"changed_paths": []any{"package.json", ".nvmrc"}},
		map[string]any{"n": 2.0, "reason": "deadline"},
		map[string]any{"n": 3.0, "reason": "grade_failed"},
	}}}
	if got := deadlineNote(r, "001", 1); !strings.Contains(got, "stopped at the 2400 s deadline: 812345 tokens, $0.42; changed 2 file(s)") {
		t.Fatal(got)
	}
	if got := deadlineNote(r, "001", 2); !strings.Contains(got, "weren't collected") {
		t.Fatal(got)
	}
	if got := deadlineNote(r, "001", 3); got != "" {
		t.Fatal(got)
	}
}
