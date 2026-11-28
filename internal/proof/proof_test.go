package proof

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTreeSHA256(t *testing.T) {
	d := t.TempDir()
	if h, err := TreeSHA256(filepath.Join(d, "missing")); err != nil || h != "" {
		t.Fatalf("missing dir: %q %v", h, err)
	}
	_ = os.MkdirAll(filepath.Join(d, "tests"), 0o755)
	_ = os.WriteFile(filepath.Join(d, "tests", "a.test.ts"), []byte("a"), 0o644)
	h1, _ := TreeSHA256(d)
	_ = os.WriteFile(filepath.Join(d, MapFile), []byte("{}"), 0o644)
	if h2, _ := TreeSHA256(d); h2 != h1 || h1 == "" {
		t.Fatal("the coverage map is not a test")
	}
	_ = os.WriteFile(filepath.Join(d, "tests", "a.test.ts"), []byte("b"), 0o644)
	if h3, _ := TreeSHA256(d); h3 == h1 {
		t.Fatal("a changed test changes the hash")
	}
	_ = os.WriteFile(filepath.Join(d, "tests", "a.test.ts"), []byte("a"), 0o644)
	_ = os.Rename(filepath.Join(d, "tests", "a.test.ts"), filepath.Join(d, "tests", "b.test.ts"))
	if h4, _ := TreeSHA256(d); h4 == h1 {
		t.Fatal("a renamed test changes the hash")
	}
}

func TestParseAndJudge(t *testing.T) {
	good := &Proof{Version: 1, Item: "004", Reference: "patch", TestsTreeSHA256: "t1", Result: Proven}
	p, err := Parse(good.Marshal())
	if err != nil {
		t.Fatal(err)
	}
	if s := Judge(p, nil, "t1"); !s.OK || s.Warn {
		t.Fatalf("%+v", s)
	}
	if s := Judge(p, nil, "t2"); s.OK || s.Problem != "stale: the hidden tests changed since the proof" {
		t.Fatalf("%+v", s)
	}
	if s := Judge(nil, nil, "t1"); s.OK || s.Problem != "missing" {
		t.Fatalf("%+v", s)
	}
	failed := *good
	failed.Result, failed.Code = Failed, "DRAFT_FAILS_ON_REFERENCE"
	if s := Judge(&failed, nil, "t1"); s.OK || s.Problem != "failed (DRAFT_FAILS_ON_REFERENCE)" {
		t.Fatalf("%+v", s)
	}
	none, err := Parse([]byte(`{"version":1,"item":"004","reference":"none","result":"none","decided_by":"owner"}`))
	if err != nil {
		t.Fatal(err)
	}
	if s := Judge(none, nil, "anything"); !s.OK || !s.Warn {
		t.Fatalf("%+v", s)
	}
	for _, bad := range []string{`{`, `{"version":2,"result":"proven","tests_tree_sha256":"x"}`, `{"version":1,"result":"maybe"}`,
		`{"version":1,"result":"none","reference":"patch"}`, `{"version":1,"result":"proven"}`} {
		_, err := Parse([]byte(bad))
		if err == nil {
			t.Errorf("accepted %s", bad)
		}
		if s := Judge(nil, err, "t"); s.OK {
			t.Errorf("an invalid proof is never ok: %s", bad)
		}
	}
}
