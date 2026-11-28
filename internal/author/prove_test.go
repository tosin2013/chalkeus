package author

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tosin2013/chalkeus/internal/config"
	"github.com/tosin2013/chalkeus/internal/fixture"
	"github.com/tosin2013/chalkeus/internal/proof"
	"github.com/tosin2013/chalkeus/internal/readiness"
)

// proveWorld: the companion has merged hidden tests for 006 and, unless noRef, a reference.
func proveWorld(t *testing.T, report map[string]any, noRef bool) (*world, *int) {
	w := newWorld(t, config.LevelMinimal)
	files := map[string]string{"proj/items/006/tests/test_x.py": "def test_x():\n    assert False, '" + marker + "'\n",
		"proj/items/006/" + MapFile: "{}"}
	if !noRef {
		files["proj/references/006.patch"] = "diff --git a/app.py b/app.py\n" + refMarker + "\n"
	}
	gitRepo(t, fixture.BaseURL, "me/acc.git", files)
	calls := 0
	w.env.Reference = func(dir, id, item string, res readiness.Result, overlay, ref []byte) (map[string]any, []byte, error) {
		calls++
		if !bytes.Contains(ref, []byte(refMarker)) || !bytes.Contains(overlay, []byte(marker)) || bytes.Contains(overlay, []byte(refMarker)) {
			t.Fatal("prove staged the wrong inputs")
		}
		if res.Commit != w.res.Commit {
			t.Fatalf("prove checks on readiness's base: %s", res.Commit)
		}
		_ = os.MkdirAll(filepath.Join(dir, "reference"), 0o755)
		return report, []byte("output " + marker), nil
	}
	return w, &calls
}

func TestProveOpensAPRWithTheProof(t *testing.T) {
	w, calls := proveWorld(t, refReport("", "proven"), false)
	p, err := Prove(context.Background(), w.env, "006")
	if err != nil || *calls != 1 {
		t.Fatalf("%v calls=%d", err, *calls)
	}
	if len(w.fake.tree) != 1 || w.fake.tree[0]["path"] != "proj/proofs/006.json" {
		t.Fatalf("the PR adds only the proof: %v", w.fake.tree)
	}
	got, err := proof.Parse([]byte(w.fake.blobs[0]))
	if err != nil || got.Result != proof.Proven || got.TestsTreeSHA256 != p.TestsTreeSHA256 || got.ReferencePath != "proj/references/006.patch" {
		t.Fatalf("%v %+v", err, got)
	}
	// The tree hash is the one readiness recomputes from the companion folder.
	tmp := t.TempDir()
	write(t, tmp, "tests/test_x.py", "def test_x():\n    assert False, '"+marker+"'\n")
	if want, _ := proof.TreeSHA256(tmp); got.TestsTreeSHA256 != want {
		t.Fatal("tree hash differs from readiness's")
	}
	if !strings.HasPrefix(w.fake.prTitle, "test(006): proof of the hidden tests") || !strings.Contains(w.fake.prBody, "HIDDEN_TESTS_UNVERIFIED") {
		t.Fatalf("%s\n%s", w.fake.prTitle, w.fake.prBody)
	}
	if !strings.Contains(w.out.String(), marker) {
		t.Fatal("the owner sees the hidden tests' output in the terminal")
	}
	for _, s := range []string{w.fake.prBody, w.fake.blobs[0]} {
		if strings.Contains(s, marker) || strings.Contains(s, refMarker) {
			t.Fatal("test or reference contents leaked")
		}
	}
	// The evidence keeps verdicts only.
	_ = filepath.Walk(filepath.Join(w.env.EvidenceRoot, "proves"), func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			raw, _ := os.ReadFile(p)
			if bytes.Contains(raw, []byte(marker)) || bytes.Contains(raw, []byte(refMarker)) {
				t.Errorf("%s carries contents", p)
			}
		}
		return nil
	})
}

func TestProveFailureOpensNoPR(t *testing.T) {
	w, _ := proveWorld(t, refReport("DRAFT_FAILS_ON_REFERENCE", "fails_on_reference"), false)
	if _, err := Prove(context.Background(), w.env, "006"); code(err) != CodeDraftFailsOnReference || !strings.Contains(err.Error(), "no pull request opened") {
		t.Fatalf("%v", err)
	}
	if len(w.fake.tree) != 0 {
		t.Fatal("a failed proof opens no PR")
	}
}

func TestProveNeedsAReference(t *testing.T) {
	w, calls := proveWorld(t, nil, true)
	if _, err := Prove(context.Background(), w.env, "006"); code(err) != CodeNoReference || *calls != 0 ||
		!strings.Contains(err.Error(), "proj/references/006.patch") || !strings.Contains(err.Error(), "reference: none") {
		t.Fatalf("%v", err)
	}
}
