package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindRepoRootFound(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example\n"), 0o600); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	sub := filepath.Join(root, "nested", "dir")
	if err := os.MkdirAll(sub, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	got, err := findRepoRoot(sub)
	if err != nil {
		t.Fatalf("expected repo root, got %v", err)
	}
	expected, err := filepath.Abs(root)
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	expected, err = filepath.EvalSymlinks(expected)
	if err != nil {
		t.Fatalf("eval symlinks: %v", err)
	}
	actual, err := filepath.Abs(got)
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	actual, err = filepath.EvalSymlinks(actual)
	if err != nil {
		t.Fatalf("eval symlinks: %v", err)
	}
	if actual != expected {
		t.Fatalf("expected %s, got %s", expected, actual)
	}
}

func TestFindRepoRootNotFound(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "nested", "dir")
	if err := os.MkdirAll(sub, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	got, err := findRepoRoot(sub)
	if err == nil {
		t.Fatalf("expected error for missing repo root")
	}
	if got != "" {
		t.Fatalf("expected empty repo root on error, got %q", got)
	}
}
