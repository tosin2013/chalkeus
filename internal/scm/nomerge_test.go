package scm

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 037 R12: the GitLab and Bitbucket implementations are plain HTTP, so the guard looks at their
// strings: no merge, approve or auto-merge endpoint may appear in non-test code. (The go-github
// method check in internal/ghloop/nomerge_test.go already walks internal/scm/github.)
func TestNoMergeOrApprovePathsInHostPackages(t *testing.T) {
	banned := []string{`/merge"`, `/merge?`, `/merge/`, `/approve`, `/approvals`, `/unapprove`,
		"merge_when_pipeline_succeeds", "auto_merge", "/accept"}
	fset := token.NewFileSet()
	scanned := 0
	for _, dir := range []string{"gitlab", "bitbucket"} {
		files, _ := filepath.Glob(filepath.Join(dir, "*.go"))
		for _, path := range files {
			if strings.HasSuffix(path, "_test.go") {
				continue
			}
			f, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			scanned++
			ast.Inspect(f, func(n ast.Node) bool {
				if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING {
					for _, b := range banned {
						if strings.Contains(lit.Value, b) {
							t.Errorf("%s: %s contains %q (forge never merges or approves)", fset.Position(lit.Pos()), lit.Value, b)
						}
					}
				}
				return true
			})
		}
	}
	if scanned == 0 {
		t.Fatal("scanned no host package files")
	}
}

// CloseChangeRequest exists for reset-fixture only (contracts/host-interface.md): no other
// production code calls it.
func TestCloseChangeRequestOnlyFromResetFixture(t *testing.T) {
	root := filepath.Join("..", "..")
	allowed := map[string]bool{"internal/cli/fixture.go": true}
	fset := token.NewFileSet()
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if info.IsDir() {
			switch info.Name() {
			case ".git", "node_modules", ".build", "testdata":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") || strings.HasPrefix(rel, "internal/scm/") || allowed[rel] {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if s, ok := n.(*ast.SelectorExpr); ok && s.Sel.Name == "CloseChangeRequest" {
				t.Errorf("%s: calls CloseChangeRequest; only reset-fixture may", fset.Position(s.Pos()))
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
