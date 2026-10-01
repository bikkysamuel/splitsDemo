package httpapi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Every problem type the server sends is listed in api/openapi.yaml, so
// clients can map it (ADR-0012). A new problemKind fails here until the
// contract's Problem schema documents its slug.
func TestEveryProblemKindIsDocumentedInTheContract(t *testing.T) {
	spec, err := os.ReadFile(filepath.Join("..", "..", "..", "api", "openapi.yaml"))
	if err != nil {
		t.Fatalf("read contract: %v", err)
	}

	for _, kind := range problemKinds {
		entry := "- `" + kind.slug + "`:"
		if !strings.Contains(string(spec), entry) {
			t.Errorf("problem slug %q is not in the Problem schema's slug list in api/openapi.yaml (want a line %q)",
				kind.slug, entry)
		}
	}
}
