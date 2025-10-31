// Package testutil provides reusable testing helpers for enforcing architectural
// and API boundary invariants across the repository.
package testutil

import (
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// AssertNoTransitiveDependency shells out to `go list -deps` with the provided pattern
// (e.g. ./... or .) and fails the test if any dependency path satisfies the forbidden predicate.
// The reason string is appended to the failure for clarity.
func AssertNoTransitiveDependency(t testing.TB, pattern string, forbidden func(path string) bool, reason string) {
	t.Helper()
	viols, out, err := transitiveDependencyViolations(pattern, forbidden)
	if err != nil {
		t.Fatalf("go list failed: %v\n%s", err, string(out))
	}
	failIfTransitiveViolations(t, reason, viols)
}

// AssertNoDirectImports scans all non-test .go files in dir (typically "." from within the package)
// and fails if any import path satisfies the forbidden predicate. It does not follow build tags.
func AssertNoDirectImports(t testing.TB, dir string, forbidden func(importPath string) bool, reason string) {
	t.Helper()
	viols, err := directImportViolations(dir, forbidden)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	failIfDirectViolations(t, reason, viols)
}

// DomainImportForbidden returns a predicate matching any import path that points to the domain package.
func DomainImportForbidden(path string) bool {
	return strings.HasSuffix(path, "/pkg/domain") || strings.Contains(path, "/pkg/domain@")
}

// InternalImportForbidden returns a predicate matching any import path containing /internal/.
func InternalImportForbidden(path string) bool {
	return strings.Contains(path, "/internal/")
}

var goListDeps = func(pattern string) ([]byte, error) {
	cmd := exec.Command("go", "list", "-deps", pattern)
	return cmd.CombinedOutput()
}

func transitiveDependencyViolations(pattern string, forbidden func(path string) bool) ([]string, []byte, error) {
	out, err := goListDeps(pattern)
	if err != nil {
		return nil, out, err
	}
	var viols []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if forbidden(line) {
			viols = append(viols, line)
		}
	}
	return viols, out, nil
}

func directImportViolations(dir string, forbidden func(importPath string) bool) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	var viols []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(dir, name)
		fileAst, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil, err
		}
		for _, imp := range fileAst.Imports {
			ip := strings.Trim(imp.Path.Value, "\"")
			if forbidden(ip) {
				viols = append(viols, ip+" (in "+name+")")
			}
		}
	}
	return viols, nil
}

type fatalLogger interface {
	Fatalf(format string, args ...any)
}

func failIfTransitiveViolations(t fatalLogger, reason string, viols []string) {
	if len(viols) > 0 {
		t.Fatalf("forbidden transitive dependency detected (%s):\n%s", reason, strings.Join(viols, "\n"))
	}
}

func failIfDirectViolations(t fatalLogger, reason string, viols []string) {
	if len(viols) > 0 {
		t.Fatalf("forbidden direct imports detected (%s):\n%s", reason, strings.Join(viols, "\n"))
	}
}
