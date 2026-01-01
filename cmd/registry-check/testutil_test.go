package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	code := runTests(m)
	os.Exit(code)
}

func writeTestFile(t *testing.T, name, content string) string {
	t.Helper()
	base := filepath.Base(name)
	ext := filepath.Ext(base)
	prefix := strings.TrimSuffix(base, ext)
	if prefix == "" {
		prefix = "test"
	}
	pattern := prefix + "-*"
	if ext != "" {
		pattern += ext
	}
	tmp, err := os.CreateTemp(".", pattern)
	if err != nil {
		t.Fatalf("create temp %s: %v", base, err)
	}
	if _, err := tmp.WriteString(content); err != nil {
		t.Fatalf("write %s: %v", tmp.Name(), err)
	}
	if err := tmp.Close(); err != nil {
		t.Fatalf("close %s: %v", tmp.Name(), err)
	}

	absPath, err := filepath.Abs(tmp.Name())
	if err != nil {
		t.Fatalf("abs %s: %v", tmp.Name(), err)
	}
	t.Cleanup(func() {
		_ = os.Remove(absPath)
	})

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("cwd: %v", err)
	}
	relPath, err := filepath.Rel(cwd, absPath)
	if err != nil {
		t.Fatalf("rel %s: %v", absPath, err)
	}
	if strings.Contains(relPath, "..") {
		t.Fatalf("temp file path escapes working dir: %s", absPath)
	}
	return relPath
}

func runTests(m *testing.M) int {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "getwd:", err)
		return 1
	}
	repoRoot, rootErr := findRepoRoot(cwd)
	if rootErr != nil {
		fmt.Fprintln(os.Stderr, "repo root:", rootErr)
		repoRoot = cwd
	}
	if err := os.Chdir(repoRoot); err != nil {
		fmt.Fprintln(os.Stderr, "chdir repo root:", err)
		return 1
	}
	code := m.Run()
	if err := os.Chdir(cwd); err != nil {
		fmt.Fprintln(os.Stderr, "chdir restore:", err)
	}
	if rootErr != nil && code == 0 {
		fmt.Fprintf(os.Stderr, "warning: repo root not found (%v); tests ran from cwd %s. Run tests from the repo root to enable repo-relative checks.\n", rootErr, cwd)
	}
	return code
}

func findRepoRoot(start string) (string, error) {
	if strings.TrimSpace(start) == "" {
		return "", fmt.Errorf("start path is empty")
	}
	resolved, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("abs %s: %w", start, err)
	}
	abs := resolved
	dir := abs
	for {
		if hasRepoMarker(dir) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("repo root not found from %s", abs)
		}
		dir = parent
	}
}

func hasRepoMarker(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		return true
	}
	return false
}
