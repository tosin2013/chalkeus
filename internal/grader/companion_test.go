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
