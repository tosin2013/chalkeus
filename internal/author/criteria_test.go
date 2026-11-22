package author

import (
	"strings"
	"testing"
)

// 026 T041: a slice draft covers its own story's scenarios, with the FRs as context.
func TestSliceCriteria(t *testing.T) {
	all := []Criterion{{ID: "US1-AS1"}, {ID: "US1-AS2"}, {ID: "US2-AS1"}, {ID: "US12-AS1"}, {ID: "FR-001"}}
	var ids []string
	for _, c := range SliceCriteria(all, "US2") {
		ids = append(ids, c.ID)
	}
	if strings.Join(ids, ",") != "US2-AS1,FR-001" {
		t.Fatalf("US2: %v", ids)
	}
	ids = nil
	for _, c := range SliceCriteria(all, "US1") {
		ids = append(ids, c.ID)
	}
	if strings.Join(ids, ",") != "US1-AS1,US1-AS2,FR-001" {
		t.Fatalf("US1 must not pick up US12: %v", ids)
	}
	if len(SliceCriteria(all, "")) != 5 || len(SliceCriteria(all, "ALL")) != 5 {
		t.Fatal("a whole item keeps every criterion")
	}
}
