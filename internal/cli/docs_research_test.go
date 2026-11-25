package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/tosin2013/chalkeus/internal/diagnose"
)

// 033 FR-016, FR-017: every catalogued failure reason has a troubleshooting entry, and the
// research phrases docs/targets.md lists are exactly the ones doctor and the diagnosis use.
func TestDocsResearchBeforeBuild(t *testing.T) {
	root := filepath.Join("..", "..")
	ts, err := os.ReadFile(filepath.Join(root, "docs", "troubleshooting.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range diagnose.KnownReasons {
		if !strings.Contains(string(ts), "`"+r+"`") {
			t.Errorf("docs/troubleshooting.md lacks the failure reason `%s` (internal/diagnose/catalogue.go)", r)
		}
	}
	tg, err := os.ReadFile(filepath.Join(root, "docs", "targets.md"))
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile("(?s)A task that starts with one of\\s+these phrases reads as research: (.*?)\\. Do the").FindSubmatch(tg)
	if m == nil {
		t.Fatal("docs/targets.md#research-tasks: the research phrase list is missing")
	}
	var listed []string
	for _, x := range regexp.MustCompile("`([^`]+)`").FindAllSubmatch(m[1], -1) {
		listed = append(listed, string(x[1]))
	}
	if strings.Join(listed, "|") != strings.Join(diagnose.ResearchPhrases, "|") {
		t.Errorf("docs/targets.md research phrases %v differ from diagnose.ResearchPhrases %v", listed, diagnose.ResearchPhrases)
	}
}
