package main

import (
	"os"
	"strings"
	"testing"
)

func TestValidatePath(t *testing.T) {
	if _, err := validatePath(""); err == nil {
		t.Fatalf("expected error for empty path")
	}
	if _, err := validatePath("/abs/path"); err == nil {
		t.Fatalf("expected error for absolute path")
	}
	if _, err := validatePath("../traverse"); err == nil {
		t.Fatalf("expected error for traversal path")
	}
	if p, err := validatePath("docs/rfc/registry.yaml"); err != nil || p == "" {
		t.Fatalf("expected clean path, got %q err %v", p, err)
	}
}

func TestAssignScalarUnsupported(t *testing.T) {
	d := &Document{}
	if err := assignScalar(d, "unknown_field", "val"); err == nil {
		t.Fatalf("expected error for unsupported scalar field")
	}
}

func TestAppendListUnsupported(t *testing.T) {
	d := &Document{}
	if err := appendList(d, "unknown_list", "val"); err == nil {
		t.Fatalf("expected error for unsupported list field")
	}
}

func TestValidateDate(t *testing.T) {
	if err := validateDate("2024-13-01"); err == nil { // invalid month
		t.Fatalf("expected invalid date error")
	}
	if err := validateDate("2025-10-04"); err != nil { // valid
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseRegistryErrorsAdditional(t *testing.T) {
	// malformed content (missing documents: header)
	f, err := os.CreateTemp(t.TempDir(), "badreg-*.yaml")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	if _, err := f.WriteString("notdocuments: \n"); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		t.Fatalf("seek: %v", err)
	}
	if _, err := parseRegistry(f); err == nil {
		t.Fatalf("expected header error")
	}
	_ = f.Close()

	// unsupported scalar field inside a document
	f2, err := os.CreateTemp(t.TempDir(), "badreg2-*.yaml")
	if err != nil {
		t.Fatalf("temp file2: %v", err)
	}
	content := strings.Join([]string{
		"documents:",
		"  - id: DOC-1",
		"    unknown: value",
	}, "\n")
	if _, err := f2.WriteString(content); err != nil {
		t.Fatalf("write2: %v", err)
	}
	if _, err := f2.Seek(0, 0); err != nil {
		t.Fatalf("seek2: %v", err)
	}
	if _, err := parseRegistry(f2); err == nil {
		t.Fatalf("expected unsupported scalar error")
	}
	_ = f2.Close()
}

// TestRegistryHelperFunctions exercises individual helper functions to lift coverage.
func TestRegistryHelperFunctions(t *testing.T) {
	if n := countLeadingSpaces("   abc"); n != 3 {
		t.Fatalf("expected 3, got %d", n)
	}
	if n := countLeadingSpaces("noindent"); n != 0 {
		t.Fatalf("expected 0, got %d", n)
	}

	k, v, err := splitKeyValue("key: value")
	if err != nil || k != "key" || v != "value" {
		t.Fatalf("splitKeyValue unexpected: %s %s %v", k, v, err)
	}
	if _, _, err := splitKeyValue("novalue"); err == nil {
		t.Fatalf("expected error for missing colon")
	}

	testValue := "TestValue"
	if got := normalizeScalar("\"" + testValue + "\""); got != testValue {
		t.Fatalf("expected %s, got %q", testValue, got)
	}
	if got := normalizeScalar("'" + testValue + "'"); got != testValue {
		t.Fatalf("expected %s, got %q", testValue, got)
	}
	if got := normalizeScalar("'Don''t'"); got != "Don't" {
		t.Fatalf("expected Don't, got %q", got)
	}

	var d Document
	if err := assignScalar(&d, "id", "DOC-1"); err != nil || d.ID != "DOC-1" {
		t.Fatalf("assignScalar id failed: %+v %v", d, err)
	}
	if err := assignScalar(&d, "unsupported_field", "x"); err == nil {
		t.Fatalf("expected unsupported field error")
	}

	resetList(&d, "authors")
	if err := appendList(&d, "authors", "Alice"); err != nil || len(d.Authors) != 1 {
		t.Fatalf("append authors failed: %+v %v", d.Authors, err)
	}
	if err := appendList(&d, "stakeholders", "Stake"); err != nil || len(d.Stakeholders) != 1 {
		t.Fatalf("append stakeholders failed")
	}
	if err := appendList(&d, "reviewers", "Rev"); err != nil || len(d.Reviewers) != 1 {
		t.Fatalf("append reviewers failed")
	}
	if err := appendList(&d, "owners", "Own"); err != nil || len(d.Owners) != 1 {
		t.Fatalf("append owners failed")
	}
	if err := appendList(&d, "deciders", "Dec"); err != nil || len(d.Deciders) != 1 {
		t.Fatalf("append deciders failed")
	}
	if err := appendList(&d, "linked_annexes", "A1"); err != nil || len(d.LinkedAnnexes) != 1 {
		t.Fatalf("append annexes failed")
	}
	if err := appendList(&d, "linked_adrs", "ADR1"); err != nil || len(d.LinkedADRs) != 1 {
		t.Fatalf("append adrs failed")
	}
	if err := appendList(&d, "linked_rfcs", "RFC1"); err != nil || len(d.LinkedRFCs) != 1 {
		t.Fatalf("append rfcs failed")
	}
	if err := appendList(&d, "unknown_field", "val"); err == nil {
		t.Fatalf("expected error for unknown list field")
	}

	// validateDocument error cases
	if err := validateDocument(Document{}); err == nil {
		t.Fatalf("expected error for missing fields")
	}
	badType := Document{ID: "X", Type: "ZZ", Title: "T", Status: statusMap[statusDraftKey], Path: "p"}
	if err := validateDocument(badType); err == nil {
		t.Fatalf("expected invalid type error")
	}
	badStatus := Document{ID: "X", Type: "RFC", Title: "T", Status: "Bogus", Path: "p"}
	if err := validateDocument(badStatus); err == nil {
		t.Fatalf("expected invalid status error")
	}
	// valid minimal document
	good := Document{ID: "X", Type: "RFC", Title: "T", Status: statusMap[statusDraftKey], Path: "p"}
	if err := validateDocument(good); err != nil {
		t.Fatalf("unexpected validate error: %v", err)
	}

	if err := validateDate("2025-02-30"); err == nil {
		t.Fatalf("expected invalid date")
	}
	if err := validateDate("2025-02-28"); err != nil {
		t.Fatalf("expected valid date, got %v", err)
	}
}
