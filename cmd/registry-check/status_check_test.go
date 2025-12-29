package main

import (
	"strings"
	"testing"
)

func TestReadDocumentStatusInline(t *testing.T) {
	docPath := writeTestFile(t, "test_doc_status_inline.md", "# Doc\n- Status: Draft\n")
	status, err := readDocumentStatus(docPath)
	if err != nil {
		t.Fatalf("read status: %v", err)
	}
	if status != "Draft" {
		t.Fatalf("expected Draft, got %q", status)
	}
}

func TestReadDocumentStatusHeader(t *testing.T) {
	docPath := writeTestFile(t, "test_doc_status_header.md", "# Doc\n## Status\nAccepted (baseline)\n")
	status, err := readDocumentStatus(docPath)
	if err != nil {
		t.Fatalf("read status: %v", err)
	}
	if status != "Accepted" {
		t.Fatalf("expected Accepted, got %q", status)
	}
}

func TestRunStatusMismatch(t *testing.T) {
	docPath := writeTestFile(t, "test_doc_status_mismatch.md", "# Doc\n- Status: Accepted\n")
	regPath := writeTestFile(t, "test_registry_status_mismatch.yaml", "documents:\n  - id: ADR-1\n    type: ADR\n    title: Test\n    status: Draft\n    path: "+docPath+"\n")
	if err := run(regPath); err == nil || !strings.Contains(err.Error(), "status mismatch") {
		t.Fatalf("expected status mismatch error, got %v", err)
	}
}

func TestReadDocumentStatusHeaderWithoutValue(t *testing.T) {
	docPath := writeTestFile(t, "test_doc_status_missing.md", "# Doc\n## Status\n")
	if _, err := readDocumentStatus(docPath); err == nil || !strings.Contains(err.Error(), "status header without value") {
		t.Fatalf("expected header without value error, got %v", err)
	}
}

func TestReadDocumentStatusNotFound(t *testing.T) {
	docPath := writeTestFile(t, "test_doc_status_not_found.md", "# Doc\nContent only\n")
	if _, err := readDocumentStatus(docPath); err == nil || !strings.Contains(err.Error(), "status not found") {
		t.Fatalf("expected status not found error, got %v", err)
	}
}

func TestParseInlineStatusInvalid(t *testing.T) {
	_, ok, err := parseInlineStatus("- Status: Unknown")
	if !ok {
		t.Fatalf("expected inline status match")
	}
	if err == nil {
		t.Fatalf("expected inline status error")
	}
}

func TestCanonicalizeStatusErrors(t *testing.T) {
	if _, err := canonicalizeStatus(" "); err == nil {
		t.Fatalf("expected missing status error")
	}
	if _, err := canonicalizeStatus("Unknown"); err == nil {
		t.Fatalf("expected invalid status error")
	}
}

func TestValidateDocumentStatusReadError(t *testing.T) {
	doc := Document{
		ID:     "RFC-404",
		Status: "Draft",
		Path:   "missing-status-doc.md",
	}
	if err := validateDocumentStatus(doc); err == nil || !strings.Contains(err.Error(), "status check for RFC-404") {
		t.Fatalf("expected status check error, got %v", err)
	}
}
