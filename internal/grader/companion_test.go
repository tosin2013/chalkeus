package grader

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const secretTest = "class AcceptanceTest { void hiddenExpectation() {} }\n"

func TestOverlayAndRecord(t *testing.T) {
	comp := t.TempDir()
	for p, body := range map[string]string{
		"items/002/src/test/java/AcceptanceTest.java": secretTest,
		"items/002/src/test/resources/data.json":      "{}",
		"items/003/other.java":                        "not this item",
	} {
		_ = os.MkdirAll(filepath.Join(comp, filepath.Dir(p)), 0o755)
		_ = os.WriteFile(filepath.Join(comp, p), []byte(body), 0o644)
	}
	tarb, c, err := Overlay(comp, "me/g-acceptance", "abc123", "items/002", "002")
	if err != nil {
		t.Fatal(err)
	}
	if c.FileCount != 2 || c.Files[0].Path != "src/test/java/AcceptanceTest.java" || len(c.FilesSHA256) != 64 {
		t.Fatalf("%+v", c)
	}
	var names []string
	tr := tar.NewReader(bytes.NewReader(tarb))
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, h.Name)
	}
	if strings.Join(names, ",") != "src/test/java/AcceptanceTest.java,src/test/resources/data.json" {
		t.Fatal(names)
	}
	again, c2, _ := Overlay(comp, "me/g-acceptance", "abc123", "items/002", "002")
	if !bytes.Equal(again, tarb) || c2.FilesSHA256 != c.FilesSHA256 {
		t.Error("the overlay must be reproducible")
	}
	run := t.TempDir()
	if err := WriteCompanion(run, c); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(run, "companion.json"))
	if strings.Contains(string(raw), "hiddenExpectation") || strings.Contains(string(raw), "not this item") {
		t.Fatal("companion.json carries hidden test contents")
	}
}

func TestOverlaySharedCompanion(t *testing.T) {
	comp := t.TempDir()
	for p, body := range map[string]string{"svc/items/002/tests/test_a.py": "a", "other/items/002/tests/test_b.py": "b"} {
		_ = os.MkdirAll(filepath.Join(comp, filepath.Dir(p)), 0o755)
		_ = os.WriteFile(filepath.Join(comp, p), []byte(body), 0o644)
	}
	_, c, err := Overlay(comp, "me/acceptance", "abc", "svc/items/002", "002")
	if err != nil || c.FileCount != 1 || c.Files[0].Path != "tests/test_a.py" || c.Dir != "svc/items/002" {
		t.Fatalf("%+v %v", c, err)
	}
}

func TestOverlaySkipsTheMap(t *testing.T) {
	comp := t.TempDir()
	for p, body := range map[string]string{"items/002/tests/test_a.py": "a", "items/002/" + MapFile: "{}"} {
		_ = os.MkdirAll(filepath.Join(comp, filepath.Dir(p)), 0o755)
		_ = os.WriteFile(filepath.Join(comp, p), []byte(body), 0o644)
	}
	_, c, err := Overlay(comp, "me/acc", "abc", "items/002", "002")
	if err != nil || c.FileCount != 1 || c.Files[0].Path != "tests/test_a.py" {
		t.Fatalf("%+v %v", c, err)
	}
}

func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for p, body := range files {
		_ = os.MkdirAll(filepath.Join(root, filepath.Dir(p)), 0o755)
		if err := os.WriteFile(filepath.Join(root, p), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// 026 T009: one folder gives exactly the overlay it gave before OverlayDirs existed (the hash was
// taken from the pre-026 code), and several folders overlay in order, the later winning.
func TestOverlayDirs(t *testing.T) {
	comp := t.TempDir()
	writeTree(t, comp, map[string]string{"items/006/a/T1.java": "one", "items/006/b/T2.py": "two\n", "items/006/.acceptance-map.json": "{}"})
	_, one, err := Overlay(comp, "r", "c", "items/006", "006")
	if err != nil || one.FilesSHA256 != "93a620b5ed1f5221d974bee9bebe61661a67d48de7d2c19498c8929376d4693c" || one.Dirs != nil {
		t.Fatalf("single-folder overlay changed: %v %+v", err, one)
	}
	run := t.TempDir()
	_ = WriteCompanion(run, one)
	if raw, _ := os.ReadFile(filepath.Join(run, "companion.json")); strings.Contains(string(raw), `"dirs"`) {
		t.Fatal("a single-folder companion.json must not gain a dirs key")
	}

	writeTree(t, comp, map[string]string{
		"items/007/US1/tests/test_a.py":      "us1 a",
		"items/007/US1/tests/test_shared.py": "us1 shared",
		"items/007/US2/tests/test_b.py":      "us2 b",
		"items/007/US2/tests/test_shared.py": "us2 shared",
	})
	tarb, c, err := OverlayDirs(comp, "r", "c", []string{"items/007/US1", "items/007/US2"}, "007")
	if err != nil {
		t.Fatal(err)
	}
	if c.FileCount != 3 || strings.Join(c.Dirs, ",") != "items/007/US1,items/007/US2" {
		t.Fatalf("%+v", c)
	}
	got := map[string]string{}
	tr := tar.NewReader(bytes.NewReader(tarb))
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		b, _ := io.ReadAll(tr)
		got[h.Name] = string(b)
	}
	if got["tests/test_shared.py"] != "us2 shared" || got["tests/test_a.py"] != "us1 a" || got["tests/test_b.py"] != "us2 b" {
		t.Fatalf("later folder must win: %v", got)
	}
}
