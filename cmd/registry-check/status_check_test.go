package main

import (
	"strings"
	"testing"
)

func TestReadDocumentStatusInline(t *testing.T) {
	docPath := "test_doc_status_inline.md"
	writeTestFile(t, docPath, "# Doc\n- Status: Draft\n")
	status, err := readDocumentStatus(docPath)
	if err != nil {
		t.Fatalf("read status: %v", err)
	}
	if status != "Draft" {
		t.Fatalf("expected Draft, got %q", status)
	}
}

func TestReadDocumentStatusHeader(t *testing.T) {
	docPath := "test_doc_status_header.md"
	writeTestFile(t, docPath, "# Doc\n## Status\nAccepted (baseline)\n")
	status, err := readDocumentStatus(docPath)
	if err != nil {
		t.Fatalf("read status: %v", err)
	}
	if status != "Accepted" {
		t.Fatalf("expected Accepted, got %q", status)
	}
}

func TestRunStatusMismatch(t *testing.T) {
	docPath := "test_doc_status_mismatch.md"
	writeTestFile(t, docPath, "# Doc\n- Status: Accepted\n")
	regPath := "test_registry_status_mismatch.yaml"
	writeTestFile(t, regPath, "documents:\n  - id: ADR-1\n    type: ADR\n    title: Test\n    status: Draft\n    path: "+docPath+"\n")
	if err := run(regPath); err == nil || !strings.Contains(err.Error(), "status mismatch") {
		t.Fatalf("expected status mismatch error, got %v", err)
	}
}
