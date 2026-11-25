package audit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tosin2013/chalkeus/internal/evidence"
)

// 033 FR-011: on a run the owner ended for planning, AU02 and AU03 don't apply.
func TestStoppedForPlanningAU02AU03NA(t *testing.T) {
	d := t.TempDir()
	run := `{"kind": "item", "item": "001", "state": "stopped(back-to-planning)", "target": {"repo": "o/r"}}`
	if err := os.WriteFile(filepath.Join(d, "run.json"), []byte(run), 0o644); err != nil {
		t.Fatal(err)
	}
	rows := itemVariants(d, []evidence.CheckRow{{ID: "AU02"}, {ID: "AU03"}, {ID: "AU05"}})
	for _, r := range rows[:2] {
		if r.Result != NA || r.Detail != "run ended by the owner for planning" {
			t.Errorf("%s: %s %q", r.ID, r.Result, r.Detail)
		}
	}
	if rows[2].ID != "AU05" || rows[2].Result == NA {
		t.Errorf("other checks run as before: %+v", rows[2])
	}
}
