package audit

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// 014 T011: an item run never passes the fixture-shaped checks by default, and records without
// a kind (every committed record) keep today's checks — golden_test.go covers those unchanged.
func TestItemRunUsesItemVariants(t *testing.T) {
	root, _ := filepath.Abs("../..")
	src := filepath.Join(root, "specs", "016-go-pilot", "evidence", "20261117T001332Z")
	d := filepath.Join(t.TempDir(), "r")
	if err := exec.Command("cp", "-r", src, d).Run(); err != nil {
		t.Fatal(err)
	}
	fixtureRows, err := Dir(d, nil, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(d, "run.json"))
	raw = append([]byte(`{"kind": "item", `), raw[1:]...)
	if err := os.WriteFile(filepath.Join(d, "run.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	itemRows, err := Dir(d, nil, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	for i, r := range itemRows {
		switch r.ID {
		case "AU01", "AU02", "AU03", "AU11":
			if r.Result != "fail" {
				t.Errorf("%s passed for an item run before its variant exists", r.ID)
			}
		default:
			if r != fixtureRows[i] {
				t.Errorf("%s changed: %+v vs %+v", r.ID, r, fixtureRows[i])
			}
		}
	}
}
