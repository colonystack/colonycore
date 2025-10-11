package blob

import (
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

// TestOnlyBlobPackageImportsInfra ensures that only the top-level blob
// package wraps the infra-backed implementations. Other packages must depend
// on the blob.Store interface instead of importing infra packages directly.
func TestOnlyBlobPackageImportsInfra(t *testing.T) {
	infraPrefix := "colonycore/internal/infra/blob"
	allowedPrefix := "colonycore/internal/blob"

	cfg := &packages.Config{Mode: packages.NeedName | packages.NeedImports, Tests: true}
	pkgs, err := packages.Load(cfg, "colonycore/...")
	if err != nil {
		t.Fatalf("load packages: %v", err)
	}

	seen := make(map[string]struct{})

	for _, pkg := range pkgs {
		if strings.HasPrefix(pkg.PkgPath, allowedPrefix) {
			continue
		}
		if strings.HasPrefix(pkg.PkgPath, infraPrefix) {
			continue
		}
		for importPath := range pkg.Imports {
			if isInfraImport(importPath, infraPrefix) {
				pos := filepath.Join(pkg.PkgPath, "...")
				seen[pos+": "+importPath] = struct{}{}
			}
		}
	}

	if len(seen) > 0 {
		violations := make([]string, 0, len(seen))
		for v := range seen {
			violations = append(violations, v)
		}
		sort.Strings(violations)
		for _, v := range violations {
			t.Errorf("forbidden import of infra blob package: %s", v)
		}
		t.Fatalf("found %d forbidden imports of infra blob packages", len(violations))
	}
}

func isInfraImport(importPath, prefix string) bool {
	return importPath == prefix || strings.HasPrefix(importPath, prefix+"/")
}
