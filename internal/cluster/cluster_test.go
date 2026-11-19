package cluster

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tosin2013/specfoundry/internal/paths"
)

func TestAllowlistLikePythonReEscape(t *testing.T) {
	got, err := Allowlist([]string{"api.example-x.com", "github.com"})
	if err != nil || got != "^api\\.example\\-x\\.com$\n^github\\.com$\n" {
		t.Fatalf("%q %v", got, err)
	}
	if _, err := Allowlist([]string{"evil.com/x"}); err == nil {
		t.Error("bad host accepted")
	}
}

func TestVerifyCalico(t *testing.T) {
	f := filepath.Join(t.TempDir(), "calico.yaml")
	_ = os.WriteFile(f, []byte("x"), 0o644)
	if err := VerifyCalico(f, ""); err == nil {
		t.Error("unpinned must fail")
	}
	if err := VerifyCalico(f, strings.Repeat("0", 64)); err == nil || !strings.Contains(err.Error(), "Calico manifest hash mismatch") {
		t.Errorf("%v", err)
	}
	if err := VerifyCalico(f, "2d711642b726b04401627ca9fbac32f5c8530fb1903cc4db02258717921a4881"); err != nil {
		t.Error(err)
	}
}

// testdata/egress-golden.yaml is what the former Python pilot rendered for the same input
// (frozen when it was removed, 016 US5).
func TestRenderEgressGolden(t *testing.T) {
	root, _ := filepath.Abs("../..")
	got, err := RenderEgress(paths.At(root), "pilot-egress-model", "worker", []string{"api.example.com"}, "img@sha256:x")
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("testdata/egress-golden.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Fatalf("rendered egress differs from the golden copy:\n%s\n---\n%s", got, want)
	}
}

// 014 R6: a target's worker proxy allows the model host and its registries; the prefetch proxy
// allows the registries only, for grading-namespace clients.
func TestTargetProxyRenders(t *testing.T) {
	root, _ := filepath.Abs("../..")
	l := paths.At(root)
	w, err := RenderEgress(l, WorkerProxyNS("pysample"), "worker", []string{"model.example", "pypi.org", "files.pythonhosted.org"}, "e@sha256:0")
	if err != nil {
		t.Fatal(err)
	}
	r, err := RenderEgress(l, PrefetchProxyNS("pysample"), "grader", []string{"pypi.org", "files.pythonhosted.org"}, "e@sha256:0")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"name: pilot-egress-w-pysample", `^model\.example$`, `^pypi\.org$`, "specfoundry.io/role: worker"} {
		if !strings.Contains(w, want) {
			t.Errorf("worker proxy lacks %q", want)
		}
	}
	if strings.Contains(r, "model.example") || strings.Contains(r, `model\.example`) || !strings.Contains(r, `^files\.pythonhosted\.org$`) || !strings.Contains(r, "specfoundry.io/role: grader") {
		t.Errorf("prefetch proxy:\n%s", r)
	}
}

func TestAllowlistChanged(t *testing.T) {
	al, _ := Allowlist([]string{"old.example"})
	if AllowlistChanged(al, []string{"old.example"}) || AllowlistChanged("  "+al+"\n", []string{"old.example"}) {
		t.Fatal("same hosts must not count as a change")
	}
	if !AllowlistChanged(al, []string{"new.example"}) || !AllowlistChanged("", []string{"new.example"}) {
		t.Fatal("a different host list is a change")
	}
}

// 024 R6: the author proxy serves only role=author namespaces and allows only the model hosts.
func TestAuthorProxyRenders(t *testing.T) {
	root, _ := filepath.Abs("../..")
	y, err := RenderEgress(paths.At(root), AuthorProxyNS, "author", []string{"model.example"}, "e@sha256:0")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"name: pilot-egress-a", "specfoundry.io/role: author", `^model\.example$`} {
		if !strings.Contains(y, want) {
			t.Fatalf("missing %q in\n%s", want, y)
		}
	}
	for _, not := range []string{"github", "role: worker", "maven"} {
		if strings.Contains(y, not) {
			t.Fatalf("unexpected %q in the author proxy", not)
		}
	}
}
