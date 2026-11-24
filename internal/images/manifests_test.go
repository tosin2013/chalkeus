package images

import "testing"

func TestManifestsCoverTheCatalog(t *testing.T) {
	c := catalog(t)
	for _, n := range c.Names() {
		if len(Manifests(n)) == 0 {
			t.Errorf("toolchain %s has no manifest files", n)
		}
	}
}

func TestIsManifest(t *testing.T) {
	for _, c := range []struct {
		tc, p string
		want  bool
	}{
		{"node24", "package-lock.json", true},
		{"node24", "packages/api/package.json", true},
		{"node24", "src/package.ts", false},
		{"python3.13", "requirements-dev.txt", true},
		{"python3.13", "pyproject.toml", true},
		{"python3.13", "setup.py", false},
		{"go", "tools/go.sum", true},
		{"jdk21-maven", "module/pom.xml", true},
		{"jdk21-maven", "build.gradle", false},
		{"own-image", "package.json", false},
	} {
		if got := IsManifest(c.tc, c.p); got != c.want {
			t.Errorf("IsManifest(%s, %s) = %v", c.tc, c.p, got)
		}
	}
}
