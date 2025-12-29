package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// TestCLIValid exercises happy path for cli/run parsing a minimal registry.
func TestCLIValid(t *testing.T) {
	docPath := "test_registry_valid_doc.md"
	writeTestFile(t, docPath, "# Test\n- Status: Draft\n")
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
		"# empty documents\n",                    // missing documents section
		"documents:\n  id: missing-dash-entry\n", // field before item
		"documents:\n  - id missing-colon\n",     // malformed key
	}
	for i, c := range cases {
		tmp, err := os.CreateTemp(t.TempDir(), "bad-*.yaml")
		if err != nil {
			t.Fatalf("case %d temp: %v", i, err)
		}
		if _, err := tmp.WriteString(c); err != nil {
			t.Fatalf("case %d write: %v", i, err)
		}
		if err := tmp.Close(); err != nil {
			t.Fatalf("case %d close: %v", i, err)
		}
		if err := run(tmp.Name()); err == nil {
			t.Fatalf("case %d expected error", i)
		}
	}
}

// TestValidateDocumentDate ensures invalid date surfaces.
func TestValidateDocumentDate(t *testing.T) {
	content := "documents:\n  - id: RFC-2\n    type: RFC\n    title: Test\n    status: Draft\n    path: docs/rfc/rfc-0002.md\n    date: 2025-13-40\n"
	tmp, err := os.CreateTemp(t.TempDir(), "bad-date-*.yaml")
	if err != nil {
		t.Fatalf("temp: %v", err)
	}
	if _, err := tmp.WriteString(content); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := tmp.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if err := run(tmp.Name()); err == nil {
		t.Fatalf("expected invalid date error")
	}
}

// TestUnsupportedListField ensures unsupported list item field triggers error.
func TestUnsupportedListField(t *testing.T) {
	// Inject unknown list "unknowns:" with an item
	content := "documents:\n  - id: RFC-3\n    type: RFC\n    title: Test\n    status: Draft\n    path: docs/rfc/rfc-0003.md\n    unknowns:\n      - val\n"
	tmp, err := os.CreateTemp(t.TempDir(), "bad-list-*.yaml")
	if err != nil {
		t.Fatalf("temp: %v", err)
	}
	if _, err := tmp.WriteString(content); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := tmp.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if err := run(tmp.Name()); err == nil {
		t.Fatalf("expected unsupported list field error")
	}
}

func TestRunAllScalarFields(t *testing.T) {
	docPath := "test_registry_full_doc.md"
	writeTestFile(t, docPath, "# Test\n- Status: Draft\n")
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

// Keep original minimal test to ensure package counted.
func TestMainExec(_ *testing.T) {}
