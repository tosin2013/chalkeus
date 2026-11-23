package author

import (
	"archive/tar"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tosin2013/chalkeus/internal/frame"
	"github.com/tosin2013/chalkeus/internal/runner"
)

type fakeCluster struct {
	applied, deleted []string
	staged           []byte
	secretFiles      []string
}

func (f *fakeCluster) Apply(y string) error { f.applied = append(f.applied, y); return nil }
func (f *fakeCluster) CreateSecretFromFiles(ns, name string, files []string) error {
	if name != "model-key" {
		panic("only the model key")
	}
	f.secretFiles = files
	return nil
}
func (f *fakeCluster) StageInputs(ns, job, container, dir string, b []byte, _ time.Duration) error {
	f.staged = b
	return nil
}
func (f *fakeCluster) DeleteNS(ns string, wait bool) { f.deleted = append(f.deleted, ns) }

type fakeRunner struct {
	log   string
	block chan struct{}
}

func (r *fakeRunner) Submit(kind, ns, name, manifest string, own bool) (runner.Handle, error) {
	return runner.Handle{Kind: kind, Namespace: ns, Name: name, OwnNamespace: own}, nil
}
func (r *fakeRunner) Wait(h runner.Handle, _ time.Duration) string {
	if r.block != nil {
		<-r.block
	}
	return "Complete"
}
func (r *fakeRunner) Collect(h runner.Handle) string        { return r.log }
func (r *fakeRunner) Status(h runner.Handle) map[string]any { return map[string]any{} }
func (r *fakeRunner) Delete(h runner.Handle)                {}

func tarOf(t *testing.T, entries map[string]string, extra ...*tar.Header) []byte {
	t.Helper()
	var b bytes.Buffer
	tw := tar.NewWriter(&b)
	for n, c := range entries {
		_ = tw.WriteHeader(&tar.Header{Name: n, Mode: 0o644, Size: int64(len(c)), Typeflag: tar.TypeReg})
		_, _ = tw.Write([]byte(c))
	}
	for _, h := range extra {
		_ = tw.WriteHeader(h)
	}
	_ = tw.Close()
	return b.Bytes()
}

func jobEnv(t *testing.T, c *fakeCluster, r *fakeRunner) (JobEnv, *Draft) {
	root, _ := filepath.Abs("../..")
	ws := t.TempDir()
	for p, c := range map[string]string{"spec.md": "# spec", "criteria.json": "{}", "base/pom.xml": "<project/>", "draft/old.txt": "old"} {
		_ = os.MkdirAll(filepath.Dir(filepath.Join(ws, p)), 0o755)
		_ = os.WriteFile(filepath.Join(ws, p), []byte(c), 0o644)
	}
	_ = os.Symlink("/etc/passwd", filepath.Join(ws, "base", "link"))
	return JobEnv{Manifests: filepath.Join(root, "pilot", "manifests"), Cluster: c, Runner: r,
			Images: map[string]string{"worker": "w@sha256:1", "tools": "t@sha256:2"}, Model: "p/m", SpendCap: "1", TokenCap: "2000000",
			ModelFiles: []string{"/x/opencode.json"}, Deadline: 1200},
		&Draft{ID: "D1", Target: "stockroom", Item: "009", Workspace: ws}
}

func TestRunJobReplacesTheDraft(t *testing.T) {
	c, r := &fakeCluster{}, &fakeRunner{}
	r.log = "noise\n" + frame.Encode(map[string][]byte{
		"report.json": []byte(`{"harness":{"exit":0},"spend":{"tokens":10},"error":null}`),
		"draft.tar":   tarOf(t, map[string]string{"draft/src/test/T.java": "class T {}", "draft/.acceptance-map.json": "{}"}),
	})
	je, d := jobEnv(t, c, r)
	rec := RunJob(context.Background(), je, d, 1, "prompt")
	if rec.Error != "" || rec.DraftFiles != 2 || rec.Outcome != "Complete" || rec.Namespace != "pilot-a-stockroom-009" {
		t.Fatalf("%+v", rec)
	}
	if _, err := os.Stat(filepath.Join(d.Workspace, "draft", "old.txt")); err == nil {
		t.Fatal("the Job's draft replaces the old one")
	}
	if b, _ := os.ReadFile(filepath.Join(d.Workspace, "draft", "src", "test", "T.java")); string(b) != "class T {}" {
		t.Fatal("draft file")
	}
	if len(c.applied) != 1 || !strings.Contains(c.applied[0], "chalkeus.io/role: author") {
		t.Fatal("the author namespace is applied")
	}
	if len(c.deleted) < 2 || c.deleted[len(c.deleted)-1] != "pilot-a-stockroom-009" {
		t.Fatal("the namespace is deleted afterwards")
	}
	// The staged workspace has no links.
	if bytes.Contains(c.staged, []byte("etc/passwd")) {
		t.Fatal("links are not staged")
	}
}

func TestRunJobRefusesAnEscapingDraft(t *testing.T) {
	for name, tb := range map[string][]byte{
		"outside": tarOf(t, map[string]string{"base/pom.xml": "x"}),
		"dotdot":  tarOf(t, map[string]string{"draft/../base/pom.xml": "x"}),
		"link":    tarOf(t, nil, &tar.Header{Name: "draft/l", Typeflag: tar.TypeSymlink, Linkname: "/etc/passwd"}),
	} {
		c, r := &fakeCluster{}, &fakeRunner{}
		r.log = frame.Encode(map[string][]byte{"report.json": []byte(`{"error":null}`), "draft.tar": tb})
		je, d := jobEnv(t, c, r)
		rec := RunJob(context.Background(), je, d, 1, "p")
		if !strings.HasPrefix(rec.Error, "bad_draft") {
			t.Fatalf("%s: %+v", name, rec)
		}
		if b, _ := os.ReadFile(filepath.Join(d.Workspace, "draft", "old.txt")); string(b) != "old" {
			t.Fatalf("%s: a refused draft leaves the old one", name)
		}
		if b, _ := os.ReadFile(filepath.Join(d.Workspace, "base", "pom.xml")); string(b) != "<project/>" {
			t.Fatalf("%s: base/ untouched", name)
		}
	}
}

func TestRunJobNoResultAndCancel(t *testing.T) {
	c, r := &fakeCluster{}, &fakeRunner{log: "no frame here"}
	je, d := jobEnv(t, c, r)
	if rec := RunJob(context.Background(), je, d, 1, "p"); !strings.HasPrefix(rec.Error, "no_result") {
		t.Fatalf("%+v", rec)
	}
	r.block = make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rec := RunJob(ctx, je, d, 2, "p")
	close(r.block)
	if rec.Error != "cancelled" || rec.Outcome != "cancelled" {
		t.Fatalf("%+v", rec)
	}
	if c.deleted[len(c.deleted)-1] != "pilot-a-stockroom-009" {
		t.Fatal("a cancelled Job's namespace is deleted")
	}
}

func TestRunJobNeedsImages(t *testing.T) {
	c, r := &fakeCluster{}, &fakeRunner{}
	je, d := jobEnv(t, c, r)
	je.Images = map[string]string{}
	if rec := RunJob(context.Background(), je, d, 1, "p"); !strings.Contains(rec.Error, "./chalkeus forge up") || len(c.applied) != 0 {
		t.Fatalf("%+v", rec)
	}
}
