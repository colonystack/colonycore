// Package sqlite provides dependency boundary tests for infra persistence (in-memory + sqlite).
package sqlite

import (
	"go/build"
	"strings"
	"testing"
)

var allowedDomainImports = map[string]struct{}{
	"colonycore/pkg/domain":             {},
	"colonycore/pkg/domain/entitymodel": {},
}

func TestImportsAreDomainOrStdlib(t *testing.T) {
	pkg, err := build.Default.ImportDir(".", 0)
	if err != nil {
		t.Fatalf("import dir: %v", err)
	}
	for _, imp := range pkg.Imports {
		if !strings.HasPrefix(imp, "colonycore/") {
			continue
		}
		if _, ok := allowedDomainImports[imp]; ok {
			continue
		}
		t.Fatalf("unexpected dependency: %s", imp)
	}
}
