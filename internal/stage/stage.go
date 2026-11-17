// Package stage builds the tar a Job's receiver init container unpacks (014 R4): an item run's
// inputs (a repository snapshot, a change, hidden tests) can exceed a ConfigMap's 1 MiB.
package stage

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"
)

// Tar holds the entries to stage; Add and AddTar collect, Bytes writes them in sorted order.
type Tar struct {
	files map[string][]byte
}

// New starts an empty staging tar.
func New() *Tar { return &Tar{files: map[string][]byte{}} }

func clean(name string) (string, error) {
	n := path.Clean(strings.TrimPrefix(name, "/"))
	if n == "." || n == ".." || strings.HasPrefix(n, "../") {
		return "", fmt.Errorf("stage: bad path %q", name)
	}
	return n, nil
}

// AddBytes stages content under name.
func (t *Tar) AddBytes(name string, b []byte) error {
	n, err := clean(name)
	if err != nil {
		return err
	}
	t.files[n] = b
	return nil
}

// AddFile stages a host file under name.
func (t *Tar) AddFile(name, hostPath string) error {
	b, err := os.ReadFile(hostPath)
	if err != nil {
		return err
	}
	return t.AddBytes(name, b)
}

// AddTar stages every regular file of an existing tar under prefix (for example the companion
// overlay under grading/overlay/).
func (t *Tar) AddTar(prefix string, b []byte) error {
	tr := tar.NewReader(bytes.NewReader(b))
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if h.Typeflag != tar.TypeReg {
			continue
		}
		raw, err := io.ReadAll(tr)
		if err != nil {
			return err
		}
		if err := t.AddBytes(path.Join(prefix, h.Name), raw); err != nil {
			return err
		}
	}
}

// Names lists the staged paths, sorted.
func (t *Tar) Names() []string {
	out := make([]string, 0, len(t.files))
	for n := range t.files {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// Bytes writes the tar: regular files, mode 0644, fixed times, parent directories implied.
func (t *Tar) Bytes() ([]byte, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	dirs := map[string]bool{}
	for _, n := range t.Names() {
		for d := path.Dir(n); d != "." && !dirs[d]; d = path.Dir(d) {
			dirs[d] = true
		}
	}
	var ds []string
	for d := range dirs {
		ds = append(ds, d)
	}
	sort.Strings(ds)
	for _, d := range ds {
		if err := tw.WriteHeader(&tar.Header{Name: d + "/", Mode: 0o755, Typeflag: tar.TypeDir, Format: tar.FormatPAX}); err != nil {
			return nil, err
		}
	}
	for _, n := range t.Names() {
		b := t.files[n]
		if err := tw.WriteHeader(&tar.Header{Name: n, Mode: 0o644, Size: int64(len(b)), Typeflag: tar.TypeReg, Format: tar.FormatPAX}); err != nil {
			return nil, err
		}
		if _, err := tw.Write(b); err != nil {
			return nil, err
		}
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
