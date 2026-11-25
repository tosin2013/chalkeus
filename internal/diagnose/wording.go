package diagnose

import (
	"regexp"
	"strings"
)

// ResearchPhrases start a task that is research rather than a known change (033 research R3):
// its answer, and so its cost, isn't known when the task is written. Doctor and the diagnosis
// use this one list (FR-016); docs/targets.md lists the same phrases (docs test).
var ResearchPhrases = []string{
	"locate", "find out", "find the", "find which", "investigate", "identify the cause",
	"identify which", "figure out", "determine why", "determine which", "diagnose", "research",
	"explore", "spike",
}

// taskLine is one task-list line: checkbox, task ID, then the description (with markers).
var taskLine = regexp.MustCompile(`^\s*[-*] \[([ xX])\] (T\d+[A-Za-z]?)\b\s*(.*)$`)

// marker is a leading "[P]", "[US1]" or similar tag before a task's description.
var marker = regexp.MustCompile(`^\[[^\]]*\]\s*`)

// Task is one task-list line.
type Task struct {
	ID      string
	Checked bool
	Desc    string // without the leading markers
	Line    int    // 1-based line in the text it was read from
	Phase   string // the "## " heading it sits under, without the hashes
}

// Tasks parses the task lines of a task list, with the phase heading each one sits under.
func Tasks(text string) []Task {
	var out []Task
	phase := ""
	for i, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		if strings.HasPrefix(line, "## ") {
			phase = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			continue
		}
		m := taskLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		desc := strings.TrimSpace(m[3])
		for marker.MatchString(desc) {
			desc = marker.ReplaceAllString(desc, "")
		}
		out = append(out, Task{ID: m[2], Checked: m[1] != " ", Desc: desc, Line: i + 1, Phase: phase})
	}
	return out
}

// ResearchTask reports the research phrase a task description starts with, if any. desc may still
// carry its markers; matching is case-insensitive and needs a word boundary after the phrase.
func ResearchTask(desc string) (string, bool) {
	d := strings.TrimSpace(desc)
	for marker.MatchString(d) {
		d = marker.ReplaceAllString(d, "")
	}
	low := strings.ToLower(d)
	for _, p := range ResearchPhrases {
		if !strings.HasPrefix(low, p) {
			continue
		}
		rest := low[len(p):]
		if rest == "" || !isLetter(rest[0]) {
			return p, true
		}
	}
	return "", false
}

func isLetter(b byte) bool { return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' }

// InPhases reports whether a task's phase is one of phases (all phases when phases is empty).
func InPhases(t Task, phases []string) bool {
	if len(phases) == 0 {
		return true
	}
	for _, p := range phases {
		if strings.TrimSpace(p) == t.Phase {
			return true
		}
	}
	return false
}
