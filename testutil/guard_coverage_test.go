package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

const testForbiddenImport = "some/forbidden/package"

// TestAssertNoDirectImportsSuccess tests the success path with allowed imports
func TestAssertNoDirectImportsSuccess(t *testing.T) {
	dir := t.TempDir()

	// Create a file with allowed imports
	src := []byte(`package tmp
import "fmt"
import "os"
func X() { fmt.Println("test") }`)
	if err := os.WriteFile(filepath.Join(dir, "test.go"), src, 0o600); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	// This should succeed because we're not forbidding these imports
	AssertNoDirectImports(t, dir, func(importPath string) bool {
		return importPath == testForbiddenImport
	}, "should allow fmt and os")
}

// TestAssertNoDirectImportsWithTestFiles tests that _test.go files are ignored
func TestAssertNoDirectImportsWithTestFiles(t *testing.T) {
	dir := t.TempDir()

	// Create a regular file with safe imports
	src1 := []byte(`package tmp
import "fmt"
func X() { fmt.Println("test") }`)
	if err := os.WriteFile(filepath.Join(dir, "main.go"), src1, 0o600); err != nil {
		t.Fatalf("write main file: %v", err)
	}

	// Create a test file with forbidden imports (should be ignored)
	src2 := []byte(`package tmp
import "testing"
import "` + testForbiddenImport + `"
func TestX(t *testing.T) {}`)
	if err := os.WriteFile(filepath.Join(dir, "main_test.go"), src2, 0o600); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	// This should not fail because test files are ignored
	AssertNoDirectImports(t, dir, func(importPath string) bool {
		return importPath == testForbiddenImport
	}, "should ignore test files")
}

// TestAssertNoDirectImportsWithDirectories tests that directories are ignored
func TestAssertNoDirectImportsWithDirectories(t *testing.T) {
	dir := t.TempDir()

	// Create a subdirectory
	subdir := filepath.Join(dir, "subdir")
	if err := os.Mkdir(subdir, 0o750); err != nil {
		t.Fatalf("create subdir: %v", err)
	}

	// Create a file in the subdirectory with forbidden imports
	src := []byte(`package subpkg
import "forbidden/package"
func X() {}`)
	if err := os.WriteFile(filepath.Join(subdir, "sub.go"), src, 0o600); err != nil {
		t.Fatalf("write subdir file: %v", err)
	}

	// Create a safe file in the main directory
	safeSrc := []byte(`package tmp
import "fmt"
func Y() { fmt.Println("safe") }`)
	if err := os.WriteFile(filepath.Join(dir, "safe.go"), safeSrc, 0o600); err != nil {
		t.Fatalf("write safe file: %v", err)
	}

	// This should not fail because subdirectories are not scanned
	AssertNoDirectImports(t, dir, func(importPath string) bool {
		return importPath == "forbidden/package"
	}, "should ignore subdirectories")
}

// TestAssertNoDirectImportsWithNonGoFiles tests that non-.go files are ignored
func TestAssertNoDirectImportsWithNonGoFiles(t *testing.T) {
	dir := t.TempDir()

	// Create a non-Go file
	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("some text"), 0o600); err != nil {
		t.Fatalf("write txt file: %v", err)
	}

	// Create a Go file with safe imports
	src := []byte(`package tmp
import "fmt"
func X() { fmt.Println("test") }`)
	if err := os.WriteFile(filepath.Join(dir, "main.go"), src, 0o600); err != nil {
		t.Fatalf("write go file: %v", err)
	}

	// This should not fail
	AssertNoDirectImports(t, dir, func(_ string) bool {
		return false // forbid nothing
	}, "should ignore non-go files")
}

// TestAssertNoDirectImportsWithEmptyDirectory tests behavior with empty directory
func TestAssertNoDirectImportsWithEmptyDirectory(t *testing.T) {
	dir := t.TempDir()

	// Empty directory should not cause any issues
	AssertNoDirectImports(t, dir, func(string) bool { return true }, "should handle empty directory")
}

// TestAssertNoTransitiveDependencySuccess tests the success path for transitive dependencies
func TestAssertNoTransitiveDependencySuccess(t *testing.T) {
	// Test with current package - should not have forbidden dependencies
	AssertNoTransitiveDependency(t, ".", func(path string) bool {
		// Forbid a package that shouldn't be in our dependencies
		return path == "github.com/some/nonexistent/package"
	}, "should not depend on nonexistent package")
}

// TestAssertNoTransitiveDependencyWithAllowedDeps tests with different forbidden predicates
func TestAssertNoTransitiveDependencyWithAllowedDeps(t *testing.T) {
	// Test that it passes when we forbid packages we don't use
	AssertNoTransitiveDependency(t, ".", func(path string) bool {
		return path == "github.com/some/package/we/dont/use"
	}, "should allow dependencies we actually have")
}

// TestDomainImportForbiddenEdgeCases tests additional edge cases for the domain import predicate
func TestDomainImportForbiddenEdgeCases(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		// Edge cases not covered in original test
		{"colonycore/pkg/domain", true},              // Our actual domain path
		{"example.com/pkg/domain/subpackage", false}, // Subdirectory of domain
		{"example.com/pkg/domainutil", false},        // Similar but not exact
		{"domain/pkg/something", false},              // Domain at start but wrong position
		{"", false},                                  // Empty string
		{"something/pkg/domain@v1.2.3", true},        // Version suffix
	}

	for _, c := range cases {
		if got := DomainImportForbidden(c.in); got != c.want {
			t.Errorf("DomainImportForbidden(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

// TestInternalImportForbiddenEdgeCases tests additional edge cases for the internal import predicate
func TestInternalImportForbiddenEdgeCases(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		// Edge cases not covered in original test
		{"internal", false},                           // Just "internal" without slash
		{"example.com/internal", false},               // Internal at end without slash
		{"example.com/some/internal/deep/path", true}, // Deep internal path
		{"", false},                  // Empty string
		{"notinternal", false},       // Contains "internal" but not as segment
		{"some/internal/path", true}, // Internal in middle
	}

	for _, c := range cases {
		if got := InternalImportForbidden(c.in); got != c.want {
			t.Errorf("InternalImportForbidden(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

// TestAssertNoDirectImportsWithMultipleImports tests handling of files with multiple imports
func TestAssertNoDirectImportsWithMultipleImports(t *testing.T) {
	dir := t.TempDir()

	// Create a file with multiple allowed imports
	src := []byte(`package tmp
import "fmt"
import "os"
import "io"
func X() {
	fmt.Println("test")
}`)
	if err := os.WriteFile(filepath.Join(dir, "multi.go"), src, 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	// Should pass when none of the imports are forbidden
	AssertNoDirectImports(t, dir, func(importPath string) bool {
		return importPath == testForbiddenImport
	}, "should allow standard library imports")
}

// TestAssertNoDirectImportsWithQuotedImports tests handling of quoted imports
func TestAssertNoDirectImportsWithQuotedImports(t *testing.T) {
	dir := t.TempDir()

	// Create a file with various import styles
	src := []byte(`package tmp
import "fmt"
import (
	"os"
	alias "context"
	. "io"
)
func X() {}`)
	if err := os.WriteFile(filepath.Join(dir, "quotes.go"), src, 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	// Should pass when we don't forbid any of these imports
	AssertNoDirectImports(t, dir, func(importPath string) bool {
		return importPath == testForbiddenImport
	}, "should handle various import styles")
}
