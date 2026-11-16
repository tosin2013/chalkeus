// Package frame decodes the stdout result frame printed by worker, grader and publisher pods
// (reference: the former Python pilot's frame.py; specs/002-m1-gated-pilot/contracts/worker-io.md):
//
//	PILOT-RESULT-BEGIN v1
//	<base64 of a tar of named files>
//	PILOT-RESULT-END sha256=<hex of the decoded tar>
package frame

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"io"
	"sort"
	"strings"
)

const (
	Begin     = "PILOT-RESULT-BEGIN v1"
	EndPrefix = "PILOT-RESULT-END sha256="
	MaxBytes  = 512 * 1024
)

// Errors returned by Decode, as in the reference.
const (
	ErrNoResult = "no_result"
	ErrBadFrame = "bad_frame"
	ErrOversize = "oversize"
)

// Decode returns the files in the frame, or an error code (no_result, bad_frame, oversize).
// With no required names, "report.json" is required, as in the reference.
func Decode(logText string, required ...string) (map[string][]byte, string) {
	if len(required) == 0 {
		required = []string{"report.json"}
	}
	lines := strings.Split(strings.ReplaceAll(logText, "\r\n", "\n"), "\n")
	var begins []int
	for i, l := range lines {
		if strings.TrimSpace(l) == Begin {
			begins = append(begins, i)
		}
	}
	if len(begins) == 0 {
		return nil, ErrNoResult
	}
	if len(begins) > 1 {
		return nil, ErrBadFrame
	}
	start := begins[0]
	var ends []int
	for i := start + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], EndPrefix) {
			ends = append(ends, i)
		}
	}
	if len(ends) != 1 {
		return nil, ErrBadFrame
	}
	end := ends[0]
	want := strings.TrimSpace(lines[end][len(EndPrefix):])
	var body strings.Builder
	for _, l := range lines[start+1 : end] {
		body.WriteString(strings.TrimSpace(l))
	}
	if body.Len() > (MaxBytes*4)/3+16 {
		return nil, ErrOversize
	}
	raw, err := base64.StdEncoding.DecodeString(body.String())
	if err != nil {
		return nil, ErrBadFrame
	}
	if len(raw) > MaxBytes {
		return nil, ErrOversize
	}
	sum := sha256.Sum256(raw)
	if hex.EncodeToString(sum[:]) != want {
		return nil, ErrBadFrame
	}
	files := map[string][]byte{}
	tr := tar.NewReader(bytes.NewReader(raw))
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, ErrBadFrame
		}
		if h.Typeflag == tar.TypeXHeader || h.Typeflag == tar.TypeXGlobalHeader {
			continue
		}
		if h.Typeflag != tar.TypeReg || strings.Contains(h.Name, "/") || strings.HasPrefix(h.Name, ".") {
			return nil, ErrBadFrame
		}
		if _, dup := files[h.Name]; dup {
			return nil, ErrBadFrame
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			return nil, ErrBadFrame
		}
		files[h.Name] = data
	}
	for _, r := range required {
		if _, ok := files[r]; !ok {
			return nil, ErrBadFrame
		}
	}
	return files, ""
}

// Encode builds a frame (used by tests; the pods use the Python frame.py in their images).
func Encode(files map[string][]byte) string {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		_ = tw.WriteHeader(&tar.Header{Name: n, Mode: 0o644, Size: int64(len(files[n])), Typeflag: tar.TypeReg})
		_, _ = tw.Write(files[n])
	}
	_ = tw.Close()
	raw := buf.Bytes()
	b64 := base64.StdEncoding.EncodeToString(raw)
	var wrapped strings.Builder
	for i := 0; i < len(b64); i += 76 {
		j := min(i+76, len(b64))
		wrapped.WriteString(b64[i:j] + "\n")
	}
	sum := sha256.Sum256(raw)
	return Begin + "\n" + wrapped.String() + EndPrefix + hex.EncodeToString(sum[:]) + "\n"
}
