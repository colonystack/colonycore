package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDomainImportForbiddenPredicate covers predicate behavior.
func TestDomainImportForbiddenPredicate(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"example.com/mod/pkg/domain", true},
		{"example.com/mod/pkg/domain@v1", true},
		{"example.com/mod/pkg/notdomain", false},
	}
	for _, c := range cases {
		if got := DomainImportForbidden(c.in); got != c.want {
			t.Fatalf("DomainImportForbidden(%q)=%v want %v", c.in, got, c.want)
		}
	}
}

func TestInternalImportForbiddenPredicate(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"example.com/mod/internal/x", true},
		{"example.com/mod/pkg/x", false},
	}
	for _, c := range cases {
		if got := InternalImportForbidden(c.in); got != c.want {
			t.Fatalf("InternalImportForbidden(%q)=%v want %v", c.in, got, c.want)
		}
	}
}

// TestAssertNoDirectImports exercises the success path by creating a tiny temp package with safe imports.
func TestAssertNoDirectImports(t *testing.T) {
	dir := t.TempDir()
	src := []byte("package tmp\nimport \"fmt\"\nfunc X(){fmt.Println(1)}")
	if err := os.WriteFile(filepath.Join(dir, "x.go"), src, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	AssertNoDirectImports(t, dir, func(string) bool { return false }, "none")
}

// TestAssertNoTransitiveDependency runs against a trivial module pattern (current repo) with a predicate that always returns false to exercise path.
func TestAssertNoTransitiveDependency(t *testing.T) {
	AssertNoTransitiveDependency(t, "./...", func(string) bool { return false }, "none")
}
