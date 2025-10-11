package frog

import (
	"strings"
	"testing"

	"colonycore/testutil"
)

// TestAPIBoundaryGuards enforces that the frog plugin does not directly or transitively
// depend on forbidden internal or domain packages.
func TestAPIBoundaryGuards(t *testing.T) {
	// Direct imports guard.
	testutil.AssertNoDirectImports(t, ".", func(ip string) bool {
		return testutil.InternalImportForbidden(ip) || testutil.DomainImportForbidden(ip)
	}, "no direct imports of internal or domain packages")

	// Transitive dependency guard (replaces old dependency_guard_test).
	testutil.AssertNoTransitiveDependency(t, "./...", func(p string) bool {
		// We ignore standard library paths (no dot or domain style) implicitly since domain/internal substrings won't match.
		return testutil.DomainImportForbidden(p)
	}, "transitive dependency on domain disallowed")

	// Future extension example: assert no import path leaks a private segment name.
	_ = strings.Contains // silence potential unused import if predicates change.
}
