package ghloop

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// T046 / 016 R6: the pilot never merges. No production source in the module calls a go-github
// merge or review-approve method, nor builds a merge request by hand.
func TestNoMergeOrApproveCalls(t *testing.T) {
	banned := map[string]bool{"Merge": true, "MergeUpstream": true, "CreateReview": true, "SubmitReview": true,
		"UpdateBranch": true, "EnableAutoMerge": true}
	root := filepath.Join("..", "..")
	fset := token.NewFileSet()
	var scanned int
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			switch info.Name() {
			case ".git", "node_modules", ".build", "testdata":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		scanned++
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.SelectorExpr:
				if banned[x.Sel.Name] {
					t.Errorf("%s: calls %s (the pilot never merges or approves PRs)", fset.Position(x.Pos()), x.Sel.Name)
				}
			case *ast.BasicLit:
				if x.Kind == token.STRING && (strings.Contains(x.Value, "/merge\"") || strings.Contains(x.Value, "/merge?") || strings.Contains(x.Value, "/reviews\"")) {
					t.Errorf("%s: hand-built merge or review URL %s", fset.Position(x.Pos()), x.Value)
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if scanned < 20 {
		t.Fatalf("scanned only %d files", scanned)
	}
}
