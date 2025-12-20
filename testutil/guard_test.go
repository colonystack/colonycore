package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func TestDirectImportViolations(t *testing.T) {
	dir := t.TempDir()
	src := []byte("package tmp\nimport \"example.com/forbidden\"\n")
	if err := os.WriteFile(filepath.Join(dir, "x.go"), src, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	viols, err := directImportViolations(dir, func(path string) bool { return strings.Contains(path, "forbidden") })
	if err != nil {
		t.Fatalf("directImportViolations error: %v", err)
	}
	if len(viols) != 1 || !strings.Contains(viols[0], "forbidden") {
		t.Fatalf("expected single violation mentioning forbidden import, got %+v", viols)
	}
}

// TestAssertNoTransitiveDependency runs against a trivial module pattern (current repo) with a predicate that always returns false to exercise path.
func TestAssertNoTransitiveDependency(t *testing.T) {
	AssertNoTransitiveDependency(t, "./...", func(string) bool { return false }, "none")
}

func TestTransitiveDependencyViolations(t *testing.T) {
	original := goListDeps
	defer func() { goListDeps = original }()

	goListDeps = func(string) ([]byte, error) {
		return []byte("example.com/forbidden\n"), nil
	}
	viols, out, err := transitiveDependencyViolations("./...", func(path string) bool { return strings.Contains(path, "forbidden") })
	if err != nil {
		t.Fatalf("transitiveDependencyViolations err: %v", err)
	}
	if string(out) == "" {
		t.Fatalf("expected go list output to be returned")
	}
	if len(viols) != 1 || viols[0] != "example.com/forbidden" {
		t.Fatalf("unexpected violations: %+v", viols)
	}
}

func TestTransitiveDependencyViolationsCommandError(t *testing.T) {
	original := goListDeps
	defer func() { goListDeps = original }()

	goListDeps = func(string) ([]byte, error) {
		return []byte("boom"), fmt.Errorf("failed")
	}
	viols, out, err := transitiveDependencyViolations("./...", func(string) bool { return false })
	if err == nil {
		t.Fatalf("expected error from transitiveDependencyViolations")
	}
	if string(out) != "boom" {
		t.Fatalf("expected command output 'boom', got %q", string(out))
	}
	if len(viols) != 0 {
		t.Fatalf("expected no violations when command fails, got %+v", viols)
	}
}

func TestFailIfDirectViolations(t *testing.T) {
	logger := &fakeLogger{}
	failIfDirectViolations(logger, "reason", []string{"violation"})
	if !logger.called {
		t.Fatalf("expected logger to be called")
	}
	if !strings.Contains(logger.msg, "reason") || !strings.Contains(logger.msg, "violation") {
		t.Fatalf("expected message to contain reason and violation, got %q", logger.msg)
	}

	logger = &fakeLogger{}
	failIfDirectViolations(logger, "reason", nil)
	if logger.called {
		t.Fatalf("expected logger not to be called when there are no violations")
	}
}

func TestFailIfTransitiveViolations(t *testing.T) {
	logger := &fakeLogger{}
	failIfTransitiveViolations(logger, "reason", []string{"pkg"})
	if !logger.called {
		t.Fatalf("expected logger to be called for violations")
	}
	if !strings.Contains(logger.msg, "pkg") || !strings.Contains(logger.msg, "reason") {
		t.Fatalf("expected message to contain violation and reason, got %q", logger.msg)
	}

	logger = &fakeLogger{}
	failIfTransitiveViolations(logger, "reason", nil)
	if logger.called {
		t.Fatalf("expected logger not to be called when violations slice empty")
	}
}

type fakeLogger struct {
	called bool
	msg    string
}

func (f *fakeLogger) Fatalf(format string, args ...any) {
	f.called = true
	f.msg = fmt.Sprintf(format, args...)
}
