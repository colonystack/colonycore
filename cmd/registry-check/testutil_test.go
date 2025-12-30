package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
