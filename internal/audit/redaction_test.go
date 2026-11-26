package audit

import (
	"os"
	"path/filepath"
	"testing"
)

// Token prefixes must start a word; real-looking tokens still match.
func TestPatternsWordStart(t *testing.T) {
	no := []string{
		"- Risk assessments: `risk-assessment-YYYY-MM-DD-topic.md`",
		"desk-organisation-guide-for-2026-and-beyond",
		"laughs_aaaaaaaaaaaaaaaaaaaaaaaa",
		`{"doc": "see risk-assessment-2026-11-25-topic.md"}`,
	}
	yes := []string{
		"OPENAI_API_KEY=sk-abcdefghijklmnop1234",
		"token: ghp_abcdefghijklmnopqrstuvwxyz",
		"github_pat_abcdefghij_klmnop",
		"Authorization: Bearer abcdefghijklmnopqrstu",
		"-----BEGIN RSA PRIVATE KEY-----",
		"sk-abcdefghijklmnop1234",
		"GITLAB_TOKEN=glpat-abcdefghijklmnopqrst12", // 037: GitLab personal, project and group tokens
		// JSON evidence: a token after an escaped newline, tab or carriage return.
		`{"detail": "line one\nghp_abcdefghijklmnopqrstuvwxyz"}`,
		`{"out": "x\tsk-abcdefghijklmnop1234"}`,
		`{"out": "x\rgithub_pat_abcdefghij_klmnop"}`,
	}
	for _, s := range no {
		if Patterns.MatchString(s) {
			t.Errorf("false positive: %q", s)
		}
	}
	for _, s := range yes {
		if !Patterns.MatchString(s) {
			t.Errorf("missed: %q", s)
		}
	}
}

// reconcile's .graded-<task> clone of the target isn't evidence.
func TestRedactionScanSkipsGradedClone(t *testing.T) {
	d := t.TempDir()
	_ = os.MkdirAll(filepath.Join(d, ".graded-001", "docs"), 0o755)
	_ = os.WriteFile(filepath.Join(d, ".graded-001", "docs", "x.md"), []byte("ghp_abcdefghijklmnopqrstuvwxyz\n"), 0o644)
	_ = os.WriteFile(filepath.Join(d, "run.json"), []byte("{}\n"), 0o644)
	hits, err := RedactionScan(d, nil, nil)
	if err != nil || len(hits) != 0 {
		t.Fatalf("%v %v", hits, err)
	}
	_ = os.WriteFile(filepath.Join(d, "leak.txt"), []byte("ghp_abcdefghijklmnopqrstuvwxyz\n"), 0o644)
	if hits, _ := RedactionScan(d, nil, nil); len(hits) != 1 || hits[0] != "leak.txt" {
		t.Fatalf("a real leak must still be found: %v", hits)
	}
}
