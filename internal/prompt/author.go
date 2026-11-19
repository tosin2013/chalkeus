package prompt

import (
	"fmt"
	"strings"
)

// Author is what the automatic test author's prompt needs (024 R4). The workspace itself
// (spec.md, plan.md, tasks.md, criteria.json, base/) is in the Job's working directory.
type Author struct {
	Item       string
	SpecSHA256 string
	Toolchain  string
	TestPaths  []string // the project's test files, where the drafts go (laid out like the project)
	// Feedback is set on a retry: the failing check's code and the drafted tests' own output on
	// the unchanged base (the author's own tests, never the project's secrets).
	FeedbackCode   string
	FeedbackOutput string
}

// AuthorRules are the test-author skill's rules (.claude/skills/test-author/SKILL.md, steps
// 3-4), restated for an unattended Job that has no pilot commands. A test keeps them in step.
var AuthorRules = []string{
	"fail on today's code and pass once the spec is implemented",
	"test behaviour the spec states, through the project's public interfaces, not its internals",
	"use only the project's test framework and dependencies it already declares (the grader is offline)",
	"not depend on anything outside the project (network, clock, other files of yours)",
}

// BuildAuthor is the automatic test author's prompt.
func BuildAuthor(a Author) (string, error) {
	if a.Item == "" || a.SpecSHA256 == "" {
		return "", fmt.Errorf("author prompt needs the item and its spec hash")
	}
	var b strings.Builder
	fmt.Fprintf(&b, "You are the test author for item %s. You draft its hidden acceptance tests: they decide whether a\n", a.Item)
	b.WriteString("later change does what the item's spec asks. No human is watching; work until you are done, then stop.\n\n")
	b.WriteString("The current directory holds `spec.md`, `plan.md`, `tasks.md`, `criteria.json` and the project as it is\n")
	b.WriteString("today under `base/` (read-only). Write only under `draft/`.\n\nSteps:\n")
	b.WriteString("1. Read `spec.md`, `plan.md`, `tasks.md`, `criteria.json` and the project under `base/`.\n")
	b.WriteString("2. Write tests under `draft/`, laid out like the project")
	if len(a.TestPaths) > 0 {
		fmt.Fprintf(&b, " (its tests live in %s)", quoteList(a.TestPaths))
	}
	b.WriteString(". The tests must:\n")
	for _, r := range AuthorRules {
		fmt.Fprintf(&b, "   - %s;\n", r)
	}
	fmt.Fprintf(&b, "3. Write `draft/.acceptance-map.json`:\n   `{\"version\": 1, \"spec_sha256\": \"%s\", \"criteria\": {\"<id>\": [\"<test file>::<test name>\"]}, \"uncovered\": [\"<id>\"]}`\n", a.SpecSHA256)
	b.WriteString("   using the criterion ids from `criteria.json`. Cover every criterion you can; list the rest under\n")
	b.WriteString("   `uncovered`, honestly.\n\n")
	fmt.Fprintf(&b, "You cannot run the tests here (the project uses the %s toolchain, and this sandbox reaches only the\n", a.Toolchain)
	b.WriteString("model). After you stop, the pilot runs them on the unchanged project: the project's own tests must still\n")
	b.WriteString("pass, and yours must fail. Write code that compiles against `base/` as it is, plus the classes the spec\n")
	b.WriteString("says the item adds.\n\nNever write outside `draft/`, and never put test contents anywhere but the draft files.\n")
	if a.FeedbackCode != "" {
		fmt.Fprintf(&b, "\n# Your previous draft failed its check: %s\n\n", a.FeedbackCode)
		switch a.FeedbackCode {
		case "DRAFT_HIDDEN_PASSES":
			b.WriteString("Your tests already pass on today's code, so they can't tell a correct change from no change. Make each\none test behaviour the spec adds.\n")
		case "DRAFT_SUITE_FAILS":
			b.WriteString("With your tests added, the project's own tests fail on today's code: your tests may break the build\n(for example a compile error) or need an undeclared dependency. Fix that.\n")
		}
		if o := strings.TrimSpace(a.FeedbackOutput); o != "" {
			fmt.Fprintf(&b, "\nYour tests' output on today's code:\n\n```text\n%s\n```\n", clipTail(o, 6000))
		}
		b.WriteString("\nYour previous draft is still under `draft/`; change it there.\n")
	}
	return b.String(), nil
}

func clipTail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "…" + s[len(s)-n:]
}
