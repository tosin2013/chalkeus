package debt

import (
	"regexp"
	"strings"
)

// DependabotLogin is the REST login of Dependabot's pull requests.
const DependabotLogin = "dependabot[bot]"

// ecosystems maps Dependabot's branch ecosystem to the catalog toolchain whose manifests it
// changes (research R3); anything else isn't triaged.
var ecosystems = map[string]string{
	"npm_and_yarn": "node24",
	"pip":          "python3.13",
	"go_modules":   "go",
	"maven":        "jdk21-maven",
}

// Bump is what a Dependabot PR changes, parsed from its title, body and branch.
type Bump struct {
	Ecosystem string
	Toolchain string // "" when no catalog toolchain covers the ecosystem
	Directory string // "/" by default
	Packages  []Package
}

var (
	// "chore(deps): bump zod from 3.25.76 to 4.1.12 in /docs" (any Conventional Commits prefix).
	titleOne = regexp.MustCompile(`(?i)^(?:[a-z]+(?:\([^)]*\))?!?:\s*)?bump (\S+) from (\S+) to (\S+?)(?: in (/\S*))?$`)
	// "chore(deps): bump esbuild and tsx", "bump ws in /docs", "bump the npm group across …".
	titleMany = regexp.MustCompile(`(?i)^(?:[a-z]+(?:\([^)]*\))?!?:\s*)?bump .+?(?: in (/\S*))?$`)
	// Body lines: "Updates `esbuild` from 0.25.9 to 0.28.1".
	bodyUpdates = regexp.MustCompile("(?m)^Updates `([^`]+)` from (\\S+) to (\\S+?)\\.?$")
	// Body: "Bumps [ws](https://…) from 8.18.3 to 8.19.0."
	bodyBumps = regexp.MustCompile(`Bumps \[([^\]]+)\]\([^)]*\) from (\S+) to (\S+?)\.?(?:\s|$)`)
	// Grouped-update table rows: "| [zod](https://…) | `3.25.76` | `4.1.12` |".
	bodyTable = regexp.MustCompile("(?m)^\\|\\s*\\[([^\\]]+)\\]\\([^)]*\\)\\s*\\|\\s*`([^`]+)`\\s*\\|\\s*`([^`]+)`\\s*\\|")
	bodyDir   = regexp.MustCompile(`in the (/\S*) directory`)
)

// IsDependabot reports whether a PR was opened by Dependabot.
func IsDependabot(p PR) bool { return p.Author == DependabotLogin }

// ParseBump reads a Dependabot PR. ok is false when no package with versions could be read.
func ParseBump(p PR) (Bump, bool) {
	b := Bump{Directory: "/"}
	if parts := strings.Split(p.HeadRef, "/"); len(parts) >= 3 && parts[0] == "dependabot" {
		b.Ecosystem = parts[1]
		b.Toolchain = ecosystems[b.Ecosystem]
	}
	title := strings.TrimSpace(p.Title)
	seen := map[string]bool{}
	add := func(name, from, to string) {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		b.Packages = append(b.Packages, Package{Name: name, From: from, To: to, Level: Level(from, to)})
	}
	if m := titleOne.FindStringSubmatch(title); m != nil {
		add(m[1], m[2], m[3])
		if m[4] != "" {
			b.Directory = m[4]
		}
	} else if m := titleMany.FindStringSubmatch(title); m != nil && m[1] != "" {
		b.Directory = m[1]
	}
	for _, re := range []*regexp.Regexp{bodyUpdates, bodyTable, bodyBumps} {
		for _, m := range re.FindAllStringSubmatch(p.Body, -1) {
			add(m[1], m[2], m[3])
		}
	}
	if b.Directory == "/" {
		if m := bodyDir.FindStringSubmatch(p.Body); m != nil {
			b.Directory = m[1]
		}
	}
	return b, len(b.Packages) > 0
}

// level is the highest level of a bump's packages.
func (b Bump) level() string {
	l := ""
	for _, p := range b.Packages {
		l = maxLevel(l, p.Level)
	}
	if l == "" {
		return LevelUnknown
	}
	return l
}
