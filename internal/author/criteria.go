package author

import (
	"fmt"
	"regexp"
	"strings"
)

// Criterion is one acceptance criterion of an item's spec: US<n>-AS<m> (acceptance scenario m
// of user story n) or FR-<nnn> (a functional requirement).
type Criterion struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

var (
	storyRE    = regexp.MustCompile(`^###\s+User Story\s+(\d+)\b`)
	h2RE       = regexp.MustCompile(`^##\s`)
	scenarioRE = regexp.MustCompile(`^\s*(\d+)\.\s+(.*)$`)
	frRE       = regexp.MustCompile(`^\s*[-*]\s+\*\*(FR-\d+[a-z]?)\*\*:?\s*(.*)$`)
	boldRE     = regexp.MustCompile(`\*\*([^*]+)\*\*`)
)

// Criteria lists a spec's acceptance criteria in order.
func Criteria(spec string) []Criterion {
	var out []Criterion
	story, inScenarios := "", false
	for _, line := range strings.Split(spec, "\n") {
		if m := storyRE.FindStringSubmatch(line); m != nil {
			story, inScenarios = m[1], false
			continue
		}
		if h2RE.MatchString(line) || strings.HasPrefix(line, "### ") {
			story, inScenarios = "", false
		}
		if story != "" && strings.Contains(line, "**Acceptance Scenarios**") {
			inScenarios = true
			continue
		}
		if inScenarios {
			if m := scenarioRE.FindStringSubmatch(line); m != nil {
				out = append(out, Criterion{ID: fmt.Sprintf("US%s-AS%s", story, m[1]), Title: title(m[2])})
			}
		}
		if m := frRE.FindStringSubmatch(line); m != nil {
			out = append(out, Criterion{ID: m[1], Title: title(m[2])})
		}
	}
	return out
}

func title(s string) string {
	s = strings.TrimSpace(boldRE.ReplaceAllString(s, "$1"))
	if len(s) > 120 {
		s = s[:117] + "..."
	}
	return s
}
