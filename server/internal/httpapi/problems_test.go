package httpapi

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Every problem type the server sends is listed in api/openapi.yaml, so
// clients can map it (ADR-0012). The test finds every problemKind literal in
// this package's source, so a new kind fails here until the contract's
// Problem schema documents its slug.
func TestEveryProblemKindIsDocumentedInTheContract(t *testing.T) {
	slugs := problemKindSlugsInSource(t)
	if len(slugs) == 0 {
		t.Fatal("found no problemKind literals; has the type been renamed?")
	}
	problemSchema := problemSchemaOf(t, filepath.Join("..", "..", "..", "api", "openapi.yaml"))

	for _, slug := range slugs {
		if entry := "- `" + slug + "`:"; !strings.Contains(problemSchema, entry) {
			t.Errorf("problem slug %q is not in the Problem schema's slug list in api/openapi.yaml (want a line %q)",
				slug, entry)
		}
	}
}

// problemKindSlugsInSource returns the slug of every problemKind{…} literal
// in the package's non-test files.
func problemKindSlugsInSource(t *testing.T) []string {
	t.Helper()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("list package files: %v", err)
	}
	var slugs []string
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
			lit, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			if typ, ok := lit.Type.(*ast.Ident); !ok || typ.Name != "problemKind" || len(lit.Elts) == 0 {
				return true
			}
			first, ok := lit.Elts[0].(*ast.BasicLit)
			if !ok || first.Kind != token.STRING {
				t.Errorf("%s: problemKind literal must start with a string slug", fset.Position(lit.Pos()))
				return true
			}
			slug, err := strconv.Unquote(first.Value)
			if err != nil {
				t.Fatalf("%s: unquote slug: %v", fset.Position(first.Pos()), err)
			}
			slugs = append(slugs, slug)
			return true
		})
	}
	return slugs
}

// problemSchemaOf returns the text of components.schemas.Problem in the
// contract: from its key line to the next schema at the same indentation.
func problemSchemaOf(t *testing.T, specPath string) string {
	t.Helper()
	spec, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("read contract: %v", err)
	}
	const key = "\n    Problem:\n"
	text := string(spec)
	start := strings.Index(text, key)
	if start < 0 {
		t.Fatalf("no components.schemas.Problem in %s", specPath)
	}
	rest := text[start+len(key):]
	for i, line := range strings.SplitAfter(rest, "\n") {
		if i > 0 && strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "     ") {
			return rest[:strings.Index(rest, line)]
		}
	}
	return rest
}
