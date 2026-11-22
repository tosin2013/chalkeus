package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

const slicesDoc = `{"items":[{"id":"006","tasks":{"checked":5,"total":10,"phases":[
 {"heading":"Phase 1: Setup","line":3,"story":null,"checked":2,"total":2},
 {"heading":"Phase 2: User Story 1 - a","line":8,"story":1,"checked":3,"total":3},
 {"heading":"Phase 3: User Story 2 - b","line":14,"story":2,"checked":0,"total":3},
 {"heading":"Phase 4: Polish","line":20,"story":null,"checked":0,"total":2}]}},
 {"id":"007","tasks":{"checked":0,"total":1}}]}`

func TestPrintSlices(t *testing.T) {
	var b bytes.Buffer
	if err := printSlices(&b, "006", slicesDoc, "spec 0.4.0", "abcdef1234567890", false); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	for _, want := range []string{"item 006 — 2 slice(s) (from spec 0.4.0, abcdef123456)", "US1  merged    5/5   Phase 1: Setup · Phase 2: User Story 1 - a",
		"US2  untouched 0/5   Phase 3: User Story 2 - b · Phase 4: Polish", "current: US2"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in\n%s", want, out)
		}
	}
	b.Reset()
	if err := printSlices(&b, "006", slicesDoc, "spec 0.4.0", "abc", true); err != nil {
		t.Fatal(err)
	}
	var rows []sliceRow
	if err := json.Unmarshal(b.Bytes(), &rows); err != nil || len(rows) != 2 || rows[1].ID != "US2" || rows[1].State != "untouched" {
		t.Fatalf("%v %+v", err, rows)
	}
	err := printSlices(&b, "007", slicesDoc, "spec 0.3.0", "abc", false)
	if err == nil || !strings.Contains(err.Error(), "SLICE_UNEVALUABLE") {
		t.Fatalf("want SLICE_UNEVALUABLE, got %v", err)
	}
}
