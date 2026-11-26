package audit

import (
	"path/filepath"
	"testing"
)

// 037: a recorded P13 (the publisher reached only its Git host) is part of AU04; a run without
// one (before 037) still passes.
func TestAU04IncludesP13WhenRecorded(t *testing.T) {
	d := build(t, filepath.Join(t.TempDir(), "20261126T000000Z"))
	p := filepath.Join(d, "checks", "results.jsonl")
	rows := readJSONL(t, p)
	rows = append(rows, ev{"id": "P13", "result": "pass", "detail": "host_via_proxy=http=200 other_via_proxy=http=000 direct=http=000"})
	writeJSONL(t, p, rows)
	if res := results(t, d); res["AU04"] != "pass" {
		t.Fatalf("P13 passing: %v", res)
	}
	failCheck(t, d, "P13", "")
	if res := results(t, d); res["AU04"] != "fail" {
		t.Fatalf("P13 failing must fail AU04: %v", res)
	}
}
