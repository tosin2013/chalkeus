// Package prompt builds the worker task prompt and the declared induction
// (reference: the former Python pilot's prompt.py).
package prompt

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	Induction = "withheld-acceptance-section"
	Heading   = "## Acceptance details"
)

// Preamble is the fixed instruction block (verbatim from the reference).
const Preamble = `You are a coding agent working in a git repository (the current directory).
Implement the specification below completely, then mark every task in the task list as
done by changing ` + "`- [ ]`" + ` to ` + "`- [X]`" + ` in that tasks.md file.

Rules:
- Change only files under ` + "`src/slugkit/`, `tests/`" + `, and the task list file named below.
- Use the Python standard library only. Do not install packages.
- Do not run git commands; the harness commits your work.
- When finished, stop.
`

// span returns the byte offsets of the acceptance section.
func span(spec string) (int, int, error) {
	start := strings.Index(spec, Heading)
	if start < 0 {
		return 0, 0, fmt.Errorf("induction requested but '%s' not found", Heading)
	}
	rest := spec[start+len(Heading):]
	nxt := strings.Index(rest, "\n## ")
	if nxt < 0 {
		return start, len(spec), nil
	}
	return start, start + len(Heading) + nxt + 1, nil
}

// SectionSHA256 is the SHA-256 of the acceptance section.
func SectionSHA256(text string) (string, error) {
	s, e, err := span(text)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256([]byte(text[s:e]))
	return hex.EncodeToString(h[:]), nil
}

// Record describes an applied induction (induction.json).
type Record struct {
	Name           string `json:"name"`
	RemovedHeading string `json:"removed_heading"`
	RemovedSHA256  string `json:"removed_sha256"`
	RemovedBytes   int    `json:"removed_bytes"`
}

// Build returns the prompt text and, when induction is requested, its record.
func Build(spec, tasks, plan, induction, tasksPath string) (string, *Record, error) {
	if tasksPath == "" {
		tasksPath = "tasks.md"
	}
	var rec *Record
	if induction != "" {
		if induction != Induction {
			return "", nil, fmt.Errorf("unknown induction '%s'", induction)
		}
		s, e, err := span(spec)
		if err != nil {
			return "", nil, err
		}
		removed := spec[s:e]
		spec = spec[:s] + spec[e:]
		h := sha256.Sum256([]byte(removed))
		rec = &Record{Name: Induction, RemovedHeading: Heading, RemovedSHA256: hex.EncodeToString(h[:]), RemovedBytes: len(removed)}
	}
	text := fmt.Sprintf("%s\n# Specification\n\n%s\n\n# Plan\n\n%s\n\n# Task list (`%s`)\n\n%s\n",
		Preamble, strings.TrimSpace(spec), strings.TrimSpace(plan), tasksPath, strings.TrimSpace(tasks))
	return text, rec, nil
}
