package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestGolden: the Go audit reproduces the reference audit on every committed evidence record
// (specs/016-go-pilot/contracts/parity.md, FR-005, SC-001).
func TestGolden(t *testing.T) {
	root, _ := filepath.Abs("../..")
	raw, err := os.ReadFile(filepath.Join(root, "specs/016-go-pilot/golden/audit-expected.json"))
	if err != nil {
		t.Fatal(err)
	}
	var g struct {
		Records []struct {
			Path  string `json:"path"`
			Final bool   `json:"final"`
			Rows  []struct {
				ID, Result, Description, Detail string
			} `json:"rows"`
		} `json:"records"`
	}
	if err := json.Unmarshal(raw, &g); err != nil {
		t.Fatal(err)
	}
	if len(g.Records) < 8 {
		t.Fatalf("expected at least 8 golden records, got %d", len(g.Records))
	}
	total := 0
	for _, rec := range g.Records {
		got, err := Dir(filepath.Join(root, rec.Path), nil, nil, rec.Final)
		if err != nil {
			t.Fatalf("%s: %v", rec.Path, err)
		}
		if len(got) != len(rec.Rows) {
			t.Errorf("%s: %d rows, want %d", rec.Path, len(got), len(rec.Rows))
			continue
		}
		for i, want := range rec.Rows {
			r := got[i]
			if r.ID != want.ID || r.Result != want.Result || r.Description != want.Description || r.Detail != want.Detail {
				t.Errorf("%s %s:\n got  %s %s %q\n want %s %s %q", rec.Path, want.ID, r.ID, r.Result, r.Detail, want.ID, want.Result, want.Detail)
			}
			total++
		}
	}
	t.Logf("golden: %d rows on %d records", total, len(g.Records))
}
