package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The automatic author follows the same rules as the test-author skill (024 R4).
func TestAuthorRulesMatchTheSkill(t *testing.T) {
	root, _ := filepath.Abs("../..")
	raw, err := os.ReadFile(filepath.Join(root, ".claude", "skills", "test-author", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	skill := strings.Join(strings.Fields(strings.ReplaceAll(string(raw), "**", "")), " ")
	for _, r := range AuthorRules {
		if !strings.Contains(skill, r) {
			t.Errorf("rule not in the skill (keep them in step): %q", r)
		}
	}
}

func TestBuildAuthor(t *testing.T) {
	p, err := BuildAuthor(Author{Item: "009", SpecSHA256: "abc", Toolchain: "jdk21-maven", TestPaths: []string{"src/test/*"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range append([]string{"item 009", "`draft/.acceptance-map.json`", `"spec_sha256": "abc"`, "`src/test/*`", "jdk21-maven", "uncovered"}, AuthorRules...) {
		if !strings.Contains(p, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(p, "failed its check") || strings.Contains(p, "./chalkeus forge") {
		t.Error("no feedback and no pilot commands on a first attempt")
	}
	r, _ := BuildAuthor(Author{Item: "009", SpecSHA256: "abc", FeedbackCode: "DRAFT_HIDDEN_PASSES", FeedbackOutput: strings.Repeat("x", 9000) + "Tests run: 3, Failures: 0"})
	if !strings.Contains(r, "failed its check: DRAFT_HIDDEN_PASSES") || !strings.Contains(r, "Failures: 0") || len(r) > 9000 {
		t.Errorf("retry feedback: %d bytes", len(r))
	}
	if _, err := BuildAuthor(Author{}); err == nil {
		t.Error("needs item and spec hash")
	}
}
