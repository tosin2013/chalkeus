package stage

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestTarUnpacksWithSystemTar(t *testing.T) {
	inner := New()
	_ = inner.AddBytes("src/test/java/HiddenTest.java", []byte("class HiddenTest {}"))
	ib, _ := inner.Bytes()
	st := New()
	_ = st.AddBytes("in/meta.json", []byte(`{"a":1}`))
	if err := st.AddTar("grading/overlay", ib); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"../x", "/../y", ".."} {
		if err := st.AddBytes(bad, nil); err == nil {
			t.Errorf("accepted %q", bad)
		}
	}
	b, err := st.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	d := t.TempDir()
	cmd := exec.Command("tar", "-x", "-f", "-", "-C", d)
	cmd.Stdin = bytes.NewReader(b)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s", out)
	}
	got, _ := os.ReadFile(filepath.Join(d, "grading/overlay/src/test/java/HiddenTest.java"))
	if string(got) != "class HiddenTest {}" || strings.Join(st.Names(), ",") != "grading/overlay/src/test/java/HiddenTest.java,in/meta.json" {
		t.Fatalf("%q %v", got, st.Names())
	}
	again, _ := st.Bytes()
	if !bytes.Equal(again, b) {
		t.Error("staging tar must be reproducible")
	}
	tr := tar.NewReader(bytes.NewReader(b))
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if strings.Contains(h.Name, "..") {
			t.Fatal(h.Name)
		}
	}
}
