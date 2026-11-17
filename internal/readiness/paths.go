// Package readiness decides whether a target repository's roadmap item is prepared to run
// (014 US2, specs/014-any-project/contracts/readiness.md). It wraps the fail-closed gate and adds
// the item rules: spec and tasks exist, no open questions, declared allowed paths, hidden tests
// in the companion repository, input size, no open PR for the item. Any doubt refuses.
package readiness

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	allowedHeadingRE = regexp.MustCompile(`^##\s+Allowed paths\s*$`)
	headingRE        = regexp.MustCompile(`^#{1,2}\s`)
	listItemRE       = regexp.MustCompile("^\\s*[-*]\\s+.*?`([^`]+)`")
)

// Refusal is one readiness refusal: a reason code and a human detail.
type Refusal struct {
	Code   string
	Detail string
}

func (r *Refusal) Error() string { return r.Code + ": " + r.Detail }

// protected are paths no item may change: CI, Spec Kit's own files, and the roadmap.
var protected = []string{".github/workflows/ci.yml", ".github/x", ".specify/memory/roadmap.md", ".specify/x"}

// AllowedPaths parses the "## Allowed paths" section of an item's spec.md (contracts/item-spec.md):
// each list item's first backticked span is one glob. The item's own tasks.md is always added.
// A glob that could match a protected path or another item's spec folder is refused.
func AllowedPaths(specText, specDir string, otherSpecDirs []string) ([]string, error) {
	var globs []string
	in := false
	for _, line := range strings.Split(specText, "\n") {
		switch {
		case allowedHeadingRE.MatchString(line):
			in = true
		case in && headingRE.MatchString(line):
			in = false
		case in:
			if m := listItemRE.FindStringSubmatch(line); m != nil {
				globs = append(globs, strings.TrimSpace(m[1]))
			}
		}
	}
	if len(globs) == 0 {
		return nil, &Refusal{"ITEM_NO_ALLOWED_PATHS", specDir + "/spec.md has no \"## Allowed paths\" list of backticked globs"}
	}
	probes := append([]string{}, protected...)
	for _, d := range otherSpecDirs {
		probes = append(probes, strings.TrimSuffix(d, "/")+"/spec.md")
	}
	for _, g := range globs {
		if g == "" || strings.HasPrefix(g, "/") || strings.Contains(g, "..") {
			return nil, &Refusal{"ITEM_PATHS_TOO_WIDE", fmt.Sprintf("allowed path %q must be a relative glob inside the repository", g)}
		}
		for _, p := range probes {
			if Match(g, p) {
				return nil, &Refusal{"ITEM_PATHS_TOO_WIDE", fmt.Sprintf("allowed path %q would let the item change %s", g, p)}
			}
		}
	}
	return append(globs, strings.TrimSuffix(specDir, "/")+"/tasks.md"), nil
}

// Match is the grader's G2 rule: Python fnmatch, where "*" also matches "/".
func Match(glob, name string) bool {
	// path.Match's "*" stops at "/"; fnmatch's doesn't, so translate the glob to a regexp.
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(glob); i++ {
		switch c := glob[i]; c {
		case '*':
			b.WriteString(".*")
		case '?':
			b.WriteString(".")
		case '[':
			j := strings.IndexByte(glob[i:], ']')
			if j < 0 {
				b.WriteString(`\[`)
				continue
			}
			cls := glob[i+1 : i+j]
			if strings.HasPrefix(cls, "!") {
				cls = "^" + cls[1:]
			}
			b.WriteString("[" + strings.ReplaceAll(cls, `\`, `\\`) + "]")
			i += j
		default:
			b.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	b.WriteString("$")
	re, err := regexp.Compile(b.String())
	return err == nil && re.MatchString(name)
}
