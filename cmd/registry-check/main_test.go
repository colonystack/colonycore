package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// TestCLIValid exercises happy path for cli/run parsing a minimal registry.
func TestCLIValid(t *testing.T) {
	docPath := writeTestFile(t, "test_registry_valid_doc.md", "# Test\n- Status: Draft\n")
	content := "documents:\n  - id: RFC-1\n    type: RFC\n    title: Test\n    status: Draft\n    path: " + docPath + "\n"
	// create relative file inside current working directory (test executes in module root)
	rel := "test_registry_valid.yaml"
	if err := os.WriteFile(rel, []byte(content), 0o600); err != nil {
		t.Fatalf("write rel: %v", err)
	}
	defer func() { _ = os.Remove(rel) }()
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	code := cli([]string{"-registry", rel}, out, errOut)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "passed") {
		t.Fatalf("expected success output: %s", out.String())
	}
}

// TestCLIInvalidPath covers validatePath error branches.
func TestCLIInvalidPath(t *testing.T) {
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	code := cli([]string{"-registry", "../outside.yaml"}, out, errOut)
	if code == 0 {
		t.Fatalf("expected non-zero exit for traversal path")
	}
}

func TestCLIFlagParseError(t *testing.T) {
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	code := cli([]string{"-unknown"}, out, errOut)
	if code != 2 {
		t.Fatalf("expected flag parse error exit code 2, got %d", code)
	}
	if !strings.Contains(errOut.String(), "flag provided but not defined") {
		t.Fatalf("expected flag parse error message, got %s", errOut.String())
	}
}

// TestParseRegistryErrors covers structural parse error conditions.
func TestParseRegistryErrors(t *testing.T) {
	cases := []string{
		"# empty documents\n",                          // missing documents section
		"docs:\n  - id: RFC-1\n",                       // unexpected root key
		"documents:\n  id: missing-dash-entry\n",       // field before item
		"documents:\n  - id missing-colon\n",           // malformed key
		"documents:\n  - id: RFC-1\n      - stray\n",   // list item without list field
		"documents:\n  - id: RFC-1\n   weird: value\n", // unsupported structure
	}
	for i, c := range cases {
		path := writeTestFile(t, "test_registry_errors.yaml", c)
		if err := run(path); err == nil {
			t.Fatalf("case %d expected error", i)
		}
	}
}

// TestValidateDocumentDate ensures invalid date surfaces.
func TestValidateDocumentDate(t *testing.T) {
	content := "documents:\n  - id: RFC-2\n    type: RFC\n    title: Test\n    status: Draft\n    path: docs/rfc/rfc-0002.md\n    date: 2025-13-40\n"
	path := writeTestFile(t, "bad-date.yaml", content)
	if err := run(path); err == nil {
		t.Fatalf("expected invalid date error")
	}
}

// TestUnsupportedListField ensures unsupported list item field triggers error.
func TestUnsupportedListField(t *testing.T) {
	// Inject unknown list "unknowns:" with an item
	content := "documents:\n  - id: RFC-3\n    type: RFC\n    title: Test\n    status: Draft\n    path: docs/rfc/rfc-0003.md\n    unknowns:\n      - val\n"
	path := writeTestFile(t, "bad-list.yaml", content)
	if err := run(path); err == nil {
		t.Fatalf("expected unsupported list field error")
	}
}

func TestRunAllScalarFields(t *testing.T) {
	docPath := writeTestFile(t, "test_registry_full_doc.md", "# Test\n- Status: Draft\n")
	content := strings.Join([]string{
		"documents:",
		"  - id: RFC-4",
		"    type: RFC",
		"    title: Example",
		"    status: Draft",
		"    created: 2025-01-01",
		"    date: 2025-01-01",
		"    last_updated: 2025-01-02",
		"    quorum: simple",
		"    target_release: v1",
		"    path: " + docPath,
		"    authors:",
		"      - Alice",
		"    stakeholders:",
		"      - Bob",
		"    reviewers:",
		"      - Carol",
		"    owners:",
		"      - Ops",
		"    deciders:",
		"      - Board",
		"    linked_annexes:",
		"      - ANNEX-1",
		"    linked_adrs:",
		"      - ADR-1",
		"    linked_rfcs:",
		"      - RFC-2",
	}, "\n") + "\n"
	rel := "test_registry_full.yaml"
	if err := os.WriteFile(rel, []byte(content), 0o600); err != nil {
		t.Fatalf("write rel: %v", err)
	}
	defer func() { _ = os.Remove(rel) }()
	if err := run(rel); err != nil {
		t.Fatalf("expected run success with full scalar coverage, got %v", err)
	}
}

func TestValidateDocumentErrors(t *testing.T) {
	base := Document{
		ID:     "RFC-100",
		Type:   "RFC",
		Title:  "Title",
		Status: "Draft",
		Path:   "docs/rfc/rfc-0100.md",
	}
	cases := []struct {
		name string
		doc  Document
		want string
	}{
		{"missing-id", Document{Type: base.Type, Title: base.Title, Status: base.Status, Path: base.Path}, "missing id"},
		{"invalid-type", Document{ID: base.ID, Type: "NOPE", Title: base.Title, Status: base.Status, Path: base.Path}, "invalid type"},
		{"invalid-status", Document{ID: base.ID, Type: base.Type, Title: base.Title, Status: "Weird", Path: base.Path}, "invalid status"},
		{"missing-path", Document{ID: base.ID, Type: base.Type, Title: base.Title, Status: base.Status}, "missing path"},
		{"bad-created", Document{ID: base.ID, Type: base.Type, Title: base.Title, Status: base.Status, Path: base.Path, Created: "2025-14-01"}, "created: invalid date"},
		{"bad-date", Document{ID: base.ID, Type: base.Type, Title: base.Title, Status: base.Status, Path: base.Path, Date: "2025-14-02"}, "date: invalid date"},
		{"bad-last-updated", Document{ID: base.ID, Type: base.Type, Title: base.Title, Status: base.Status, Path: base.Path, LastUpdated: "2025-14-03"}, "last_updated: invalid date"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateDocument(tc.doc); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q error, got %v", tc.want, err)
			}
		})
	}
}

// Keep original minimal test to ensure package counted.
func TestMainExec(_ *testing.T) {}
