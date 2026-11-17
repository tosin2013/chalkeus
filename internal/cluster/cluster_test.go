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
