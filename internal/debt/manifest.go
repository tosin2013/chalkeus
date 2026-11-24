// Package debt is the brownfield triage report (029): open bot dependency PRs grouped by root
// cause and classified, plus baseline flags, read from GitHub without writing anything.
package debt

import (
	"bufio"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"path"
	"regexp"
	"sort"
	"strings"
)

// Unknown marks a value the report couldn't read; it never makes a PR superseded or safe.
const Unknown = "unknown"

// Package is one dependency a PR or a manifest change bumps (data-model.md).
type Package struct {
	Name                 string `json:"name"`
	From                 string `json:"from"`
	To                   string `json:"to"`
	DefaultBranchVersion string `json:"default_branch_version,omitempty"`
	Level                string `json:"level"`
}

// Resolved returns the version of pkg that a manifest file pins, or "" when this file doesn't
// pin one the report can read (research R5). file is the repository path (its base name picks
// the reader).
func Resolved(file string, content []byte, pkg string) string {
	return versions(file, content)[normName(file, pkg)]
}

// Changes lists the packages whose pinned version differs between two contents of the same
// manifest file, sorted by name (research R14). A package added or removed has "" on that side.
func Changes(file string, before, after []byte) []Package {
	a, b := versions(file, before), versions(file, after)
	names := map[string]bool{}
	for n := range a {
		names[n] = true
	}
	for n := range b {
		names[n] = true
	}
	var out []Package
	for n := range names {
		if a[n] != b[n] {
			out = append(out, Package{Name: n, From: a[n], To: b[n], Level: Level(a[n], b[n])})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// versions maps each package a manifest file pins to its version; nil for files it can't read.
func versions(file string, content []byte) map[string]string {
	if len(content) == 0 {
		return nil
	}
	base := path.Base(file)
	switch {
	case base == "package-lock.json":
		return npmLock(content)
	case strings.HasPrefix(base, "requirements") && strings.HasSuffix(base, ".txt"):
		return requirements(content)
	case base == "go.mod":
		return goMod(content)
	case base == "pom.xml":
		return pom(content)
	}
	return nil
}

// normName is a package name as versions keys it: PEP 503 names for requirements files.
func normName(file, pkg string) string {
	if b := path.Base(file); strings.HasPrefix(b, "requirements") && strings.HasSuffix(b, ".txt") {
		return pep503(pkg)
	}
	return pkg
}

// npmLock reads lockfile v2/v3 "packages" (top-level node_modules only), else v1 "dependencies".
func npmLock(content []byte) map[string]string {
	var lock struct {
		Packages map[string]struct {
			Version string `json:"version"`
		} `json:"packages"`
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}
	if json.Unmarshal(content, &lock) != nil {
		return nil
	}
	out := map[string]string{}
	for k, v := range lock.Packages {
		name, ok := strings.CutPrefix(k, "node_modules/")
		if !ok || strings.Contains(name, "/node_modules/") || v.Version == "" {
			continue
		}
		out[name] = v.Version
	}
	if len(lock.Packages) == 0 {
		for k, v := range lock.Dependencies {
			if v.Version != "" {
				out[k] = v.Version
			}
		}
	}
	return out
}

var reqPin = regexp.MustCompile(`^([A-Za-z0-9][A-Za-z0-9._-]*)(\[[^\]]*\])?\s*==\s*([^\s;#,]+)`)

// requirements reads exact pins only (name==version); ranges and URLs are unknown.
func requirements(content []byte) map[string]string {
	out := map[string]string{}
	sc := bufio.NewScanner(bytes.NewReader(content))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if m := reqPin.FindStringSubmatch(line); m != nil {
			out[pep503(m[1])] = m[3]
		}
	}
	return out
}

var pep503Sep = regexp.MustCompile(`[-_.]+`)

func pep503(name string) string { return pep503Sep.ReplaceAllString(strings.ToLower(name), "-") }

// goMod reads require lines, single or in a block.
func goMod(content []byte) map[string]string {
	out := map[string]string{}
	block := false
	sc := bufio.NewScanner(bytes.NewReader(content))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if i := strings.Index(line, "//"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		switch {
		case line == "require (":
			block = true
			continue
		case block && line == ")":
			block = false
			continue
		case strings.HasPrefix(line, "require "):
			line = strings.TrimSpace(strings.TrimPrefix(line, "require "))
		case !block:
			continue
		}
		if f := strings.Fields(line); len(f) == 2 {
			out[f[0]] = f[1]
		}
	}
	return out
}

// pom reads groupId:artifactId → version for dependencies (also in dependencyManagement) and
// plugins with a literal version or a ${property} from the same file's <properties>.
func pom(content []byte) map[string]string {
	type dep struct {
		GroupID    string `xml:"groupId"`
		ArtifactID string `xml:"artifactId"`
		Version    string `xml:"version"`
	}
	var p struct {
		Properties struct {
			Any []struct {
				XMLName xml.Name
				Value   string `xml:",chardata"`
			} `xml:",any"`
		} `xml:"properties"`
		Deps    []dep `xml:"dependencies>dependency"`
		Managed []dep `xml:"dependencyManagement>dependencies>dependency"`
		Plugins []dep `xml:"build>plugins>plugin"`
	}
	if xml.Unmarshal(content, &p) != nil {
		return nil
	}
	props := map[string]string{}
	for _, a := range p.Properties.Any {
		props[a.XMLName.Local] = strings.TrimSpace(a.Value)
	}
	out := map[string]string{}
	for _, d := range append(append(p.Deps, p.Managed...), p.Plugins...) {
		v := strings.TrimSpace(d.Version)
		if name, ok := strings.CutPrefix(v, "${"); ok && strings.HasSuffix(name, "}") {
			v = props[strings.TrimSuffix(name, "}")]
		}
		if v == "" || strings.ContainsAny(v, "$[(,") || d.ArtifactID == "" {
			continue
		}
		g := strings.TrimSpace(d.GroupID)
		if g == "" {
			g = "org.apache.maven.plugins" // a plugin's default groupId
		}
		out[g+":"+strings.TrimSpace(d.ArtifactID)] = v
	}
	return out
}
