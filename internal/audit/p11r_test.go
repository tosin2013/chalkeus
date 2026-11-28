package audit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tosin2013/chalkeus/internal/evidence"
)

func TestAU04NeedsP11rWhereRecorded(t *testing.T) {
	d := t.TempDir()
	_ = os.MkdirAll(filepath.Join(d, "checks"), 0o755)
	_ = os.WriteFile(filepath.Join(d, "run.json"), []byte(`{"security_level":"minimal"}`), 0o644)
	write := func(rows string) {
		_ = os.WriteFile(filepath.Join(d, "checks", "results.jsonl"), []byte(rows), 0o644)
	}
	base := evidence.CheckRow{ID: "AU04", Result: "pass", Detail: "rows=1"}
	write(`{"id":"P11","result":"pass"}` + "\n")
	if r := itemAU04(d, base); r.Result != "pass" || strings.Contains(r.Detail, "p11r") {
		t.Fatalf("before 041: %+v", r)
	}
	write(`{"id":"P11","result":"pass"}` + "\n" + `{"id":"P11r","result":"pass"}` + "\n")
	if r := itemAU04(d, base); r.Result != "pass" || !strings.Contains(r.Detail, "p11r_rows=1") {
		t.Fatalf("%+v", r)
	}
	write(`{"id":"P11","result":"pass"}` + "\n" + `{"id":"P11r","result":"fail"}` + "\n")
	if r := itemAU04(d, base); r.Result != "fail" {
		t.Fatalf("a failed P11r fails AU04: %+v", r)
	}
}
