package main

import (
	"bytes"
	"os"
	"path/filepath"
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

func TestCLIObservabilityOptIn(t *testing.T) {
	docPath := writeTestFile(t, "test_registry_observability_doc.md", "# Test\n- Status: Draft\n")
	content := "documents:\n  - id: RFC-9\n    type: RFC\n    title: Observability\n    status: Draft\n    path: " + docPath + "\n"
	rel := "test_registry_observability.yaml"
	if err := os.WriteFile(rel, []byte(content), 0o600); err != nil {
		t.Fatalf("write rel: %v", err)
	}
	defer func() { _ = os.Remove(rel) }()

	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	code := cli([]string{"-registry", rel}, out, errOut)
	if code != 0 {
		t.Fatalf("expected exit 0 without observability flag, got %d stderr=%s", code, errOut.String())
	}
	if strings.Contains(errOut.String(), "\"schema_version\"") {
		t.Fatalf("expected no structured events without opt-in, got %s", errOut.String())
	}

	out.Reset()
	errOut.Reset()
	code = cli([]string{"-registry", rel, "-observability-json"}, out, errOut)
	if code != 0 {
		t.Fatalf("expected exit 0 with observability flag, got %d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(errOut.String(), "\"schema_version\":\"colonycore.observability.v1\"") {
		t.Fatalf("expected structured events with opt-in, got %s", errOut.String())
	}
}

func TestCLIFixCanonicalizesRegistry(t *testing.T) {
	docPath := writeTestFile(t, "test_registry_fix_doc.md", "# Test\n- Status: Draft\n")
	registryPath := writeTestFile(t, "test_registry_fix.yaml", strings.Join([]string{
		"documents:",
		"  - id: rfc-7",
		"    type: rfc",
		"    title: Fix Me",
		"    status: draft (working copy)",
		"    quorum: 1 / 2",
		"    linked_annexes:",
		"      - annex-2",
		"    linked_adrs:",
		"      - adr-3",
		"    linked_rfcs:",
		"      - rfc-4",
		"    path: .\\" + filepath.Base(docPath),
	}, "\n")+"\n")

	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	code := cli([]string{"-registry", registryPath, "-fix"}, out, errOut)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "Applied 8 registry fix(es).") {
		t.Fatalf("expected fix count output, got %s", out.String())
	}
	if !strings.Contains(out.String(), "Registry validation passed.") {
		t.Fatalf("expected validation success output, got %s", out.String())
	}

	fixed, err := os.ReadFile(registryPath) // #nosec G304 -- registryPath is created by writeTestFile within the repo root
	if err != nil {
		t.Fatalf("read fixed registry: %v", err)
	}
	want := strings.Join([]string{
		"documents:",
		"  - id: RFC-7",
		"    type: RFC",
		"    title: Fix Me",
		"    status: Draft",
		"    quorum: 1/2",
		"    linked_annexes:",
		"      - Annex-2",
		"    linked_adrs:",
		"      - ADR-3",
		"    linked_rfcs:",
		"      - RFC-4",
		"    path: " + filepath.Base(docPath),
		"",
	}, "\n")
	if string(fixed) != want {
		t.Fatalf("unexpected fixed registry:\n%s", string(fixed))
	}
}

func TestNormalizeDocumentForFix(t *testing.T) {
	doc := Document{
		ID:            " rfc-42 ",
		Type:          "annex",
		Status:        "accepted (recorded)",
		Quorum:        " Majority ",
		Path:          " ./docs\\rfc\\registry.yaml ",
		LinkedAnnexes: []string{" annex-1 "},
		LinkedADRs:    []string{" adr-2 "},
		LinkedRFCs:    []string{" rfc-3 "},
	}

	got, changes := normalizeDocumentForFix(doc)
	if changes != 8 {
		t.Fatalf("expected 8 changes, got %d", changes)
	}
	if got.ID != "RFC-42" {
		t.Fatalf("expected canonical id, got %q", got.ID)
	}
	if got.Type != "Annex" {
		t.Fatalf("expected canonical type, got %q", got.Type)
	}
	if got.Status != "Accepted" {
		t.Fatalf("expected canonical status, got %q", got.Status)
	}
	if got.Quorum != "majority" {
		t.Fatalf("expected canonical quorum, got %q", got.Quorum)
	}
	if got.Path != "docs/rfc/registry.yaml" {
		t.Fatalf("expected canonical path, got %q", got.Path)
	}
	if len(got.LinkedAnnexes) != 1 || got.LinkedAnnexes[0] != "Annex-1" {
		t.Fatalf("expected canonical annex refs, got %#v", got.LinkedAnnexes)
	}
	if len(got.LinkedADRs) != 1 || got.LinkedADRs[0] != "ADR-2" {
		t.Fatalf("expected canonical ADR refs, got %#v", got.LinkedADRs)
	}
	if len(got.LinkedRFCs) != 1 || got.LinkedRFCs[0] != "RFC-3" {
		t.Fatalf("expected canonical RFC refs, got %#v", got.LinkedRFCs)
	}
}

func TestCanonicalizeRegistryPathRejectsTraversal(t *testing.T) {
	got, changed := canonicalizeRegistryPath(" ../docs/rfc/registry.yaml ")
	if got != "../docs/rfc/registry.yaml" {
		t.Fatalf("expected trimmed traversal path to remain unchanged, got %q", got)
	}
	if !changed {
		t.Fatalf("expected surrounding whitespace trim to count as a change")
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
		"    quorum: majority",
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
		Status: statusMap[statusDraftKey],
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
