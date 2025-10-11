package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestValidatePathErrors covers empty and absolute path validation failures.
func TestValidatePathErrors(t *testing.T) {
	if _, err := validatePath(""); err == nil || !strings.Contains(err.Error(), "empty path") {
		t.Fatalf("expected empty path error, got %v", err)
	}
	abs := filepath.Join(os.TempDir(), "abs-registry.yaml")
	if _, err := validatePath(abs); err == nil || !strings.Contains(err.Error(), "absolute paths not allowed") {
		t.Fatalf("expected absolute path error, got %v", err)
	}
}

// TestRunEmptyDocuments ensures the empty documents list triggers error.
func TestRunEmptyDocuments(t *testing.T) {
	content := "documents:\n" // no list entries
	tmp, err := os.CreateTemp(t.TempDir(), "empty-docs-*.yaml")
	if err != nil {
		t.Fatalf("temp: %v", err)
	}
	if _, err := tmp.WriteString(content); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := tmp.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	// copy to relative path (run requires non-absolute)
	rel := "empty-docs.yaml"
	data, err := os.ReadFile(tmp.Name())
	if err != nil {
		t.Fatalf("read temp: %v", err)
	}
	if err := os.WriteFile(rel, data, 0o600); err != nil {
		t.Fatalf("write rel: %v", err)
	}
	defer func() { _ = os.Remove(rel) }()
	if err := run(rel); err == nil || !strings.Contains(err.Error(), "documents entry is empty") {
		t.Fatalf("expected empty documents error, got %v", err)
	}
}

// TestParseRegistryDeepErrors exercises additional structural parse branches not yet covered.
func TestParseRegistryDeepErrors(t *testing.T) {
	cases := []struct{ name, content, wantSubstr string }{
		{"top-level-token", "bad:\n  - something\n", "expected 'documents:'"},
		{"list-item-without-field", "documents:\n  - id: RFC-9\n    type: RFC\n    title: T\n    status: Draft\n    path: docs/rfc/rfc-0009.md\n      - stray\n", "list item without active list field"},
		{"unsupported-indent", "documents:\n  - id: RFC-10\n    type: RFC\n    title: T\n    status: Draft\n    path: docs/rfc/rfc-0010.md\n   badindent: value\n", "unsupported structure"},
	}
	for _, tc := range cases {
		tmp, err := os.CreateTemp(t.TempDir(), tc.name+"-*.yaml")
		if err != nil {
			t.Fatalf("%s temp: %v", tc.name, err)
		}
		if _, err := io.WriteString(tmp, tc.content); err != nil {
			t.Fatalf("%s write: %v", tc.name, err)
		}
		if err := tmp.Close(); err != nil {
			t.Fatalf("%s close: %v", tc.name, err)
		}
		// make relative copy
		rel := tc.name + "-case.yaml"
		b, err := os.ReadFile(tmp.Name())
		if err != nil {
			t.Fatalf("%s read: %v", tc.name, err)
		}
		if err := os.WriteFile(rel, b, 0o600); err != nil {
			t.Fatalf("rel write: %v", err)
		}
		err = run(rel)
		_ = os.Remove(rel)
		if err == nil || !strings.Contains(err.Error(), tc.wantSubstr) {
			t.Fatalf("%s: expected error containing %q got %v", tc.name, tc.wantSubstr, err)
		}
	}
}

// TestAssignScalarAndValidateDocumentBranches increases coverage of assignScalar, resetList, and validateDocument.
func TestAssignScalarAndValidateDocumentBranches(t *testing.T) {
	// Build a registry with a document missing various required fields incrementally.
	// Start with just id to trigger missing type.
	doc := Document{ID: "X"}
	if err := validateDocument(doc); err == nil || !strings.Contains(err.Error(), "missing type") {
		t.Fatalf("expected missing type error: %v", err)
	}
	doc.Type = "RFC"
	if err := validateDocument(doc); err == nil || !strings.Contains(err.Error(), "missing title") {
		t.Fatalf("expected missing title: %v", err)
	}
	doc.Title = "Title"
	if err := validateDocument(doc); err == nil || !strings.Contains(err.Error(), "missing status") {
		t.Fatalf("expected missing status: %v", err)
	}
	doc.Status = "Draft"
	if err := validateDocument(doc); err == nil || !strings.Contains(err.Error(), "missing path") {
		t.Fatalf("expected missing path: %v", err)
	}
	doc.Path = "docs/rfc/x.md"
	if err := validateDocument(doc); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}

	// Exercise resetList with unknown field (should be silent) then supported list with no items.
	resetList(&doc, "unknown_list")
	resetList(&doc, "authors")
	if len(doc.Authors) != 0 {
		t.Fatalf("expected authors cleared")
	}
	// assignScalar unsupported already covered; ensure additional scalar fields hit.
	if err := assignScalar(&doc, "created", "2024-01-02"); err != nil {
		t.Fatalf("assign created: %v", err)
	}
	if err := assignScalar(&doc, "last_updated", "2024-02-03"); err != nil {
		t.Fatalf("assign last_updated: %v", err)
	}
	if err := validateDocument(doc); err != nil {
		t.Fatalf("validate after dates: %v", err)
	}
}
