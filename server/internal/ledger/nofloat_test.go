package ledger_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// Money is never floating point (ADR-0002, ADR-0006). This scans ledger's
// non-test source for float and complex types and float helpers, so the
// rule holds without a reviewer having to spot it.
func TestLedgerUsesNoFloatingPoint(t *testing.T) {
	banned := map[string]bool{"float32": true, "float64": true, "complex64": true, "complex128": true}
	bannedSelectors := map[string]bool{"big.Float": true, "strconv.ParseFloat": true, "strconv.FormatFloat": true}

	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.Ident:
				if banned[n.Name] {
					t.Errorf("%s: %s is not allowed in ledger", fset.Position(n.Pos()), n.Name)
				}
			case *ast.SelectorExpr:
				if pkg, ok := n.X.(*ast.Ident); ok && bannedSelectors[pkg.Name+"."+n.Sel.Name] {
					t.Errorf("%s: %s.%s is not allowed in ledger", fset.Position(n.Pos()), pkg.Name, n.Sel.Name)
				}
			}
			return true
		})
	}
}
