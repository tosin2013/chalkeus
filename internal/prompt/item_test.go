package prompt

import (
	"strings"
	"testing"
)

func TestBuildItem(t *testing.T) {
	it := Item{Spec: "# Greeting\n\n## Allowed paths\n- `src/main/java/*`\n", Tasks: "- [ ] T001 do\n",
		TasksPath: "specs/002-x/tasks.md", AllowedPaths: []string{"src/main/java/*", "specs/002-x/tasks.md"},
		Toolchain: "jdk17-maven", TestCommand: "mvn -B test"}
	got, err := BuildItem(it)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"  - `src/main/java/*`\n  - `specs/002-x/tasks.md`\n",
		"The project uses the jdk17-maven toolchain. Check your work with: `mvn -B test`",
		"\n# Specification\n\n# Greeting\n",
		"\n# Task list (`specs/002-x/tasks.md`)\n\n- [ ] T001 do\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in\n%s", want, got)
		}
	}
	for _, not := range []string{"Python standard library", "src/slugkit", "# Plan", Heading} {
		if strings.Contains(got, not) {
			t.Errorf("item prompt contains %q", not)
		}
	}
	it.Plan = "use a map"
	if got, _ = BuildItem(it); !strings.Contains(got, "\n# Plan\n\nuse a map\n") {
		t.Error(got)
	}
	if !strings.Contains(got, strings.TrimSpace(it.Spec)) || SpecSHA256(it.Spec) != SpecSHA256(it.Spec+"\n\n") {
		t.Error("spec hash must ignore surrounding whitespace, and the prompt must carry the spec unaltered")
	}
	for _, bad := range []Item{{}, {Spec: "s", Tasks: "t", TasksPath: "p"}} {
		if _, err := BuildItem(bad); err == nil {
			t.Errorf("accepted %+v", bad)
		}
	}
}
