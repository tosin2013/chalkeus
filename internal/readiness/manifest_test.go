package readiness

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseManifestChange(t *testing.T) {
	const c = "3f9c2d0e8b7a6c5d4e3f2a1b0c9d8e7f6a5b4c3d"
	mc, present, err := ParseManifestChange("# 010\n\n## Manifest change\n\n- Commit: `" + c + "`\n- Files: `package.json`, `./package-lock.json`\n- From Dependabot PR #183\n\n## Allowed paths\n\n- `src/*`\n")
	if err != nil || !present || mc.Commit != c || strings.Join(mc.Files, ",") != "package.json,package-lock.json" {
		t.Fatalf("%+v %v %v", mc, present, err)
	}
	if _, present, err := ParseManifestChange("# 010\n\n## Allowed paths\n\n- `src/*`\n"); present || err != nil {
		t.Fatalf("an ordinary item: %v %v", present, err)
	}
	for name, body := range map[string]string{
		"short commit":  "- Commit: `3f9c2d0`\n- Files: `package.json`\n",
		"two commits":   "- Commit: `" + c + "`, `" + c + "`\n- Files: `package.json`\n",
		"no files":      "- Commit: `" + c + "`\n",
		"glob":          "- Commit: `" + c + "`\n- Files: `requirements*.txt`\n",
		"absolute path": "- Commit: `" + c + "`\n- Files: `/package.json`\n",
		"outside":       "- Commit: `" + c + "`\n- Files: `../package.json`\n",
		"files later":   "- Commit: `" + c + "`\n\n## Notes\n\n- Files: `package.json`\n",
	} {
		if _, present, err := ParseManifestChange("## Manifest change\n\n" + body); !present || err == nil {
			t.Errorf("%s: want UNREADABLE, got present=%v err=%v", name, present, err)
		}
	}
}

// manifestWorld is a target repository with a base on main and an approved bump commit on a
// Dependabot branch; dir is a depth-1 clone of main, as readiness uses.
type manifestWorld struct {
	origin, dir, bump string
}

func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@invalid"}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %s", args, out)
	}
	return strings.TrimSpace(string(out))
}

func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for p, body := range files {
		_ = os.MkdirAll(filepath.Join(dir, filepath.Dir(p)), 0o755)
		if err := os.WriteFile(filepath.Join(dir, p), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// newManifestWorld commits base, then bump (on dependabot/npm), then, when mainAfter is set,
// another commit on main.
func newManifestWorld(t *testing.T, bump, mainAfter map[string]string) manifestWorld {
	t.Helper()
	o := t.TempDir()
	gitIn(t, o, "init", "-q", "-b", "main")
	writeFiles(t, o, map[string]string{"package.json": `{"dependencies":{"zod":"3.25.0"}}`, "package-lock.json": lock("3.25.0"), "src/index.js": "x\n"})
	gitIn(t, o, "add", "-A")
	gitIn(t, o, "commit", "-qm", "base")
	gitIn(t, o, "checkout", "-q", "-b", "dependabot/npm")
	writeFiles(t, o, bump)
	gitIn(t, o, "add", "-A")
	gitIn(t, o, "commit", "-qm", "chore(deps): bump zod from 3.25.0 to 4.1.0")
	sha := gitIn(t, o, "rev-parse", "HEAD")
	gitIn(t, o, "checkout", "-q", "main")
	if mainAfter != nil {
		writeFiles(t, o, mainAfter)
		gitIn(t, o, "add", "-A")
		gitIn(t, o, "commit", "-qm", "later on main")
	}
	gitIn(t, o, "config", "uploadpack.allowAnySHA1InWant", "true")
	d := filepath.Join(t.TempDir(), "clone")
	if out, err := exec.Command("git", "clone", "-q", "--depth", "1", "file://"+o, d).CombinedOutput(); err != nil {
		t.Fatalf("%s", out)
	}
	return manifestWorld{origin: o, dir: d, bump: sha}
}

func lock(v string) string {
	return `{"lockfileVersion":3,"packages":{"":{"name":"t"},"node_modules/zod":{"version":"` + v + `"}}}`
}

var zod4 = map[string]string{"package.json": `{"dependencies":{"zod":"4.1.0"}}`, "package-lock.json": lock("4.1.0")}

func mcodes(rs []manifestRefusal) string {
	var c []string
	for _, r := range rs {
		c = append(c, r.Code)
	}
	return strings.Join(c, ",")
}

func TestCheckManifestChange(t *testing.T) {
	files := []string{"package.json", "package-lock.json"}
	t.Run("ready", func(t *testing.T) {
		w := newManifestWorld(t, zod4, nil)
		mc := &ManifestChange{Commit: w.bump, Files: files}
		if rs := CheckManifestChange(w.dir, "node24", "main", mc, false); len(rs) != 0 {
			t.Fatalf("%v", rs)
		}
		if mc.Parent == "" || !strings.HasPrefix(mc.Subject, "chore(deps): bump zod") {
			t.Fatalf("parent and subject: %+v", mc)
		}
		if len(mc.Changes) != 2 || mc.Changes[0].Before == mc.Changes[0].After || mc.Changes[0].After == "" {
			t.Fatalf("file changes: %+v", mc.Changes)
		}
		if len(mc.Packages) == 0 || mc.Packages[0].Name != "zod" || mc.Packages[0].From != "3.25.0" || mc.Packages[0].To != "4.1.0" {
			t.Fatalf("packages: %+v", mc.Packages)
		}
	})
	t.Run("unresolved", func(t *testing.T) {
		w := newManifestWorld(t, zod4, nil)
		mc := &ManifestChange{Commit: strings.Repeat("ab", 20), Files: files}
		if got := mcodes(CheckManifestChange(w.dir, "node24", "main", mc, false)); got != "MANIFEST_CHANGE_UNRESOLVED" {
			t.Fatal(got)
		}
	})
	t.Run("not a manifest of the toolchain", func(t *testing.T) {
		w := newManifestWorld(t, zod4, nil)
		mc := &ManifestChange{Commit: w.bump, Files: append(files, "src/index.js")}
		rs := CheckManifestChange(w.dir, "node24", "main", mc, false)
		if got := mcodes(rs); got != "MANIFEST_CHANGE_NOT_MANIFEST" || !strings.Contains(rs[0].Detail, "src/index.js") {
			t.Fatalf("%s %v", got, rs)
		}
	})
	t.Run("the commit touches an unlisted file", func(t *testing.T) {
		w := newManifestWorld(t, map[string]string{"package.json": zod4["package.json"], "package-lock.json": zod4["package-lock.json"], "src/index.js": "y\n"}, nil)
		rs := CheckManifestChange(w.dir, "node24", "main", &ManifestChange{Commit: w.bump, Files: files}, false)
		if got := mcodes(rs); got != "MANIFEST_CHANGE_NOT_MANIFEST" || !strings.Contains(rs[0].Detail, "also changes src/index.js") {
			t.Fatalf("%s %v", got, rs)
		}
	})
	t.Run("stale", func(t *testing.T) {
		w := newManifestWorld(t, zod4, map[string]string{"package-lock.json": lock("3.25.1")})
		rs := CheckManifestChange(w.dir, "node24", "main", &ManifestChange{Commit: w.bump, Files: files}, false)
		if got := mcodes(rs); got != "MANIFEST_CHANGE_STALE" || !strings.HasPrefix(rs[0].Detail, "package-lock.json changed on main since") {
			t.Fatalf("%s %v", got, rs)
		}
	})
	t.Run("main moved elsewhere is fine", func(t *testing.T) {
		w := newManifestWorld(t, zod4, map[string]string{"src/index.js": "z\n"})
		if rs := CheckManifestChange(w.dir, "node24", "main", &ManifestChange{Commit: w.bump, Files: files}, false); len(rs) != 0 {
			t.Fatalf("%v", rs)
		}
	})
	t.Run("empty", func(t *testing.T) {
		w := newManifestWorld(t, map[string]string{"README.md": "x\n"}, nil)
		rs := CheckManifestChange(w.dir, "node24", "main", &ManifestChange{Commit: w.bump, Files: files}, false)
		if got := mcodes(rs); got != "MANIFEST_CHANGE_NOT_MANIFEST,MANIFEST_CHANGE_EMPTY" {
			t.Fatal(got)
		}
	})
	t.Run("sliced", func(t *testing.T) {
		w := newManifestWorld(t, zod4, nil)
		if got := mcodes(CheckManifestChange(w.dir, "node24", "main", &ManifestChange{Commit: w.bump, Files: files}, true)); got != "MANIFEST_CHANGE_SLICED" {
			t.Fatal(got)
		}
	})
}
