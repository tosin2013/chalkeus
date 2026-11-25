package diagnose

import (
	"fmt"
	"regexp"
	"strings"
)

// runtimeRE finds a runtime version named in task text, per toolchain family (033 FR-014).
var runtimeRE = map[string]*regexp.Regexp{
	"node":   regexp.MustCompile(`(?i)\bnode(?:\.js)?\s+v?(\d{1,2})\b`),
	"python": regexp.MustCompile(`(?i)\bpython\s+(3\.\d{1,2})\b`),
	"java":   regexp.MustCompile(`(?i)\b(?:jdk|java)\s*(\d{1,2})\b`),
	"go":     regexp.MustCompile(`(?i)\bgo\s+(1\.\d{1,2})\b`),
}

// ToolchainRuntime is the runtime family and version a catalog toolchain provides: node24 → node
// 24, python3.13 → python 3.13, jdk21-maven → java 21, go → go <goVersion's major.minor> (from
// pins.env GO_VERSION, e.g. go1.27.4). ok is false when the version isn't known (e.g. "python").
func ToolchainRuntime(toolchain, goVersion string) (family, version string, ok bool) {
	switch {
	case strings.HasPrefix(toolchain, "node") && len(toolchain) > 4:
		return "node", strings.TrimPrefix(toolchain, "node"), true
	case strings.HasPrefix(toolchain, "python3."):
		return "python", strings.TrimPrefix(toolchain, "python"), true
	case strings.HasPrefix(toolchain, "jdk"):
		v := strings.TrimPrefix(toolchain, "jdk")
		if i := strings.IndexByte(v, '-'); i > 0 {
			v = v[:i]
		}
		return "java", v, v != ""
	case toolchain == "go":
		m := regexp.MustCompile(`^go(1\.\d+)`).FindStringSubmatch(goVersion)
		if m == nil {
			return "go", "", false
		}
		return "go", m[1], true
	}
	return "", "", false
}

// RuntimeMismatch is a runtime version a task names that the toolchain doesn't provide.
type RuntimeMismatch struct {
	Task, Named, Provides string
	Line                  int
}

// RuntimeMismatches lists the tasks that name a version of the toolchain's runtime other than
// the one it provides. Other runtimes' versions are ignored.
func RuntimeMismatches(tasks []Task, toolchain, goVersion string) []RuntimeMismatch {
	family, version, ok := ToolchainRuntime(toolchain, goVersion)
	if !ok {
		return nil
	}
	var out []RuntimeMismatch
	for _, t := range tasks {
		for _, m := range runtimeRE[family].FindAllStringSubmatch(t.Desc, -1) {
			if m[1] != version {
				out = append(out, RuntimeMismatch{Task: t.ID, Line: t.Line,
					Named: fmt.Sprintf("%s %s", family, m[1]), Provides: fmt.Sprintf("%s %s", family, version)})
			}
		}
	}
	return out
}
