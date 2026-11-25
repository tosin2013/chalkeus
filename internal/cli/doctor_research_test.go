package cli

import (
	"strings"
	"testing"

	"github.com/tosin2013/chalkeus/internal/doctor"
)

const doctorTasksMD = `# Tasks
## Phase 1: User Story 1 - Node 24 (Priority: P1)
- [ ] T001 [P] [US1] Set ` + "`.nvmrc`" + ` to Node 24
- [ ] T003 [US1] Locate the suites that leave handles open
- [x] T004 [US1] Investigate the done one
- [ ] T005 [US1] Keep Node 22 working
## Phase 2: User Story 2 - Skills (Priority: P2)
- [ ] T010 [US2] Investigate the skill format
`

// 033 FR-012, FR-014, FR-015.
func TestDoctorTaskWarnings(t *testing.T) {
	var r doctor.Report
	doctorTaskWarnings(&r, doctorTasksMD, "specs/001-x/tasks.md", []string{"Phase 1: User Story 1 - Node 24 (Priority: P1)"}, "node24", "")
	var lines []string
	for _, row := range r.Rows {
		if row.Status != doctor.Warn {
			t.Errorf("only warnings: %+v", row)
		}
		lines = append(lines, row.Check+": "+row.Detail)
	}
	all := strings.Join(lines, "\n")
	if !strings.Contains(all, `T003 (tasks.md:4) "Locate the suites that leave handles open"`) {
		t.Errorf("research warning missing:\n%s", all)
	}
	if strings.Contains(all, "T004") || strings.Contains(all, "T010") {
		t.Errorf("checked tasks and other slices' tasks must not warn:\n%s", all)
	}
	if !strings.Contains(all, "T005 (tasks.md:6) names node 22; toolchain node24 provides node 24") {
		t.Errorf("runtime warning missing:\n%s", all)
	}
	if r.Failed() {
		t.Error("warnings never fail doctor")
	}
}

func TestDoctorAllowance(t *testing.T) {
	var r doctor.Report
	doctorAllowance(&r, 0, 2400)
	if len(r.Rows) != 1 || r.Rows[0].Status != doctor.Info {
		t.Fatalf("%+v", r.Rows)
	}
	r = doctor.Report{}
	doctorAllowance(&r, 65, 2400)
	if len(r.Rows) != 1 || !strings.Contains(r.Rows[0].Detail, "took 65 s") {
		t.Fatalf("ample allowance: %+v", r.Rows)
	}
	r = doctor.Report{}
	doctorAllowance(&r, 300, 1200)
	if len(r.Rows) != 2 || r.Rows[1].Status != doctor.Warn || !strings.Contains(r.Rows[1].Detail, "only 3 full test runs") {
		t.Fatalf("low allowance: %+v", r.Rows)
	}
	if r.Failed() {
		t.Error("warnings never fail doctor")
	}
}
