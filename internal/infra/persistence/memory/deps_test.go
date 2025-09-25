package memory

import (
	"go/build"
	"strings"
	"testing"
)

func TestImportsAreDomainOrStdlib(t *testing.T) {
	pkg, err := build.Default.ImportDir(".", 0)
	if err != nil {
		t.Fatalf("import dir: %v", err)
	}
	for _, imp := range pkg.Imports {
		if strings.HasPrefix(imp, "colonycore/") && imp != "colonycore/pkg/domain" {
			t.Fatalf("unexpected dependency: %s", imp)
		}
	}
}
