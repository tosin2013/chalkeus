package images

import "path"

// manifests are each catalog toolchain's manifest files (029 research R8): the files that
// declare dependencies and pin their resolved versions, matched against a path's base name, so
// a manifest in a subdirectory counts too. Kept here rather than in toolchains.json, which is an
// image-workflow input: this table changes no image.
var manifests = map[string][]string{
	"go":          {"go.mod", "go.sum"},
	"jdk21-maven": {"pom.xml"},
	"node24":      {"package.json", "package-lock.json"},
	"python":      {"requirements*.txt", "pyproject.toml"},
	"python3.13":  {"requirements*.txt", "pyproject.toml"},
}

// Manifests returns a toolchain's manifest file patterns (nil for an unknown toolchain, such as
// an operator-supplied one).
func Manifests(toolchain string) []string { return manifests[toolchain] }

// IsManifest reports whether p (a repository path) is a manifest file of toolchain.
func IsManifest(toolchain, p string) bool {
	base := path.Base(p)
	for _, g := range manifests[toolchain] {
		if ok, _ := path.Match(g, base); ok {
			return true
		}
	}
	return false
}
