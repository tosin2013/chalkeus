package frame

import (
	"archive/tar"
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"os"
	"strings"
	"testing"
)

// Port of pilot/tests/test_frame.py, plus a frame produced by the Python frame.py the images use.

var report = []byte(`{"version": 1}`)

func TestRoundtrip(t *testing.T) {
	text := "noise before\n" + Encode(map[string][]byte{"report.json": report, "change.bundle": []byte("\x00bin")}) + "after\n"
	files, err := Decode(text)
	if err != "" || string(files["report.json"]) != string(report) || string(files["change.bundle"]) != "\x00bin" {
		t.Fatalf("%v %q", err, files)
	}
}

func TestPythonFrame(t *testing.T) {
	raw, _ := os.ReadFile("testdata/python-frame.txt")
	files, err := Decode(string(raw))
	if err != "" || len(files["change.bundle"]) != 768 || string(files["report.json"]) != `{"version": 1}` {
		t.Fatalf("python frame: %v", err)
	}
}

func expect(t *testing.T, text, want string, required ...string) {
	t.Helper()
	if _, err := Decode(text, required...); err != want {
		t.Errorf("got %q want %q", err, want)
	}
}

func TestErrors(t *testing.T) {
	f := Encode(map[string][]byte{"report.json": report})
	expect(t, "just logs\n", ErrNoResult)
	expect(t, "", ErrNoResult)
	expect(t, f+f, ErrBadFrame)
	expect(t, strings.Replace(f, Begin, "PILOT-RESULT-BEGIN v2", 1), ErrNoResult)
	lines := strings.Split(strings.TrimSuffix(f, "\n"), "\n")
	lines[len(lines)-1] = EndPrefix + strings.Repeat("0", 64)
	expect(t, strings.Join(lines, "\n")+"\n", ErrBadFrame)
	big := make([]byte, MaxBytes+10)
	_, _ = rand.Read(big)
	expect(t, Encode(map[string][]byte{"report.json": report, "big": big}), ErrOversize)
	expect(t, Encode(map[string][]byte{"change.bundle": []byte("x")}), ErrBadFrame)
	if _, err := Decode(Encode(map[string][]byte{"grade.json": []byte("{}")}), "grade.json"); err != "" {
		t.Error("required override")
	}
	expect(t, f+"PILOT-RESULT-END sha256=deadbeef\n", ErrBadFrame)
	if _, err := Decode(f + "trailing log line\n"); err != "" {
		t.Error("trailing text must be ignored")
	}
}

func TestPathTraversalRejected(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, n := range []string{"report.json", "../evil"} {
		_ = tw.WriteHeader(&tar.Header{Name: n, Size: 1, Mode: 0o644, Typeflag: tar.TypeReg})
		_, _ = tw.Write([]byte("x"))
	}
	_ = tw.Close()
	raw := buf.Bytes()
	sum := sha256.Sum256(raw)
	text := Begin + "\n" + base64.StdEncoding.EncodeToString(raw) + "\n" + EndPrefix + hex.EncodeToString(sum[:]) + "\n"
	expect(t, text, ErrBadFrame)
}
