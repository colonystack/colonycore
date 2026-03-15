package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestCLIFixFailure(t *testing.T) {
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	code := cli([]string{"-registry", "does-not-exist.yaml", "-fix"}, out, errOut)
	if code == 0 {
		t.Fatalf("expected non-zero exit code for fix failure")
	}
	if !strings.Contains(errOut.String(), "Registry fix failed") {
		t.Fatalf("expected fixer failure output, got %s", errOut.String())
	}
}

func TestFixRegistryFileNoChanges(t *testing.T) {
	docPath := writeTestFile(t, "test_registry_fix_noop_doc.md", "# Test\n- Status: Draft\n")
	registryPath := writeTestFile(t, "test_registry_fix_noop.yaml", "documents:\n  - id: RFC-8\n    type: RFC\n    title: Noop\n    status: Draft\n    path: "+docPath+"\n")

	fixes, err := fixRegistryFile(registryPath)
	if err != nil {
		t.Fatalf("expected no-op fix success, got %v", err)
	}
	if fixes != 0 {
		t.Fatalf("expected no-op fixer, got %d changes", fixes)
	}
}

func TestFixRegistryFileParseError(t *testing.T) {
	registryPath := writeTestFile(t, "test_registry_fix_invalid.yaml", "documents:\n  - id missing-colon\n")

	if _, err := fixRegistryFile(registryPath); err == nil {
		t.Fatalf("expected parse error from fixer")
	}
}

func TestNormalizeRegistryForFixDoesNotMutateInput(t *testing.T) {
	original := Registry{
		Documents: []Document{
			{
				ID:     "rfc-11",
				Type:   "rfc",
				Title:  "Immutable Input",
				Status: "draft",
				Path:   "./docs\\rfc\\immutable.md",
			},
		},
	}

	fixed, changes := normalizeRegistryForFix(original)
	if changes == 0 {
		t.Fatalf("expected normalization changes")
	}
	if fixed.Documents[0].ID != "RFC-11" {
		t.Fatalf("expected fixed registry to be normalized, got %q", fixed.Documents[0].ID)
	}
	if original.Documents[0].ID != "rfc-11" {
		t.Fatalf("expected input registry to remain unchanged, got %q", original.Documents[0].ID)
	}
	if original.Documents[0].Path != "./docs\\rfc\\immutable.md" {
		t.Fatalf("expected input path to remain unchanged, got %q", original.Documents[0].Path)
	}
}

func TestWriteRegistryFile(t *testing.T) {
	registryPath := writeTestFile(t, "test_registry_write.yaml", "documents:\n  - id: RFC-9\n    type: RFC\n    title: Old\n    status: Draft\n    path: docs/rfc/old.md\n")

	info, err := os.Stat(registryPath)
	if err != nil {
		t.Fatalf("stat original registry: %v", err)
	}

	registry := Registry{
		Documents: []Document{
			{
				ID:     "RFC-9",
				Type:   "RFC",
				Title:  "Rewritten",
				Status: "Draft",
				Path:   "docs/rfc/new.md",
			},
		},
	}
	if err := writeRegistryFile(registryPath, registry); err != nil {
		t.Fatalf("writeRegistryFile failed: %v", err)
	}

	got, err := os.ReadFile(registryPath) // #nosec G304 -- registryPath is created by writeTestFile within the repo root
	if err != nil {
		t.Fatalf("read rewritten registry: %v", err)
	}
	want := "documents:\n  - id: RFC-9\n    type: RFC\n    title: Rewritten\n    status: Draft\n    path: docs/rfc/new.md\n"
	if string(got) != want {
		t.Fatalf("unexpected rewritten registry:\n%s", string(got))
	}

	rewrittenInfo, err := os.Stat(registryPath)
	if err != nil {
		t.Fatalf("stat rewritten registry: %v", err)
	}
	if rewrittenInfo.Mode().Perm() != info.Mode().Perm() {
		t.Fatalf("expected permissions %v, got %v", info.Mode().Perm(), rewrittenInfo.Mode().Perm())
	}
}

func TestWriteRegistryFileStatError(t *testing.T) {
	if err := writeRegistryFile("does-not-exist.yaml", Registry{}); err == nil {
		t.Fatalf("expected stat error for missing registry")
	}
}

func TestMarshalRegistryQuotesSpecialScalars(t *testing.T) {
	registry := Registry{
		Documents: []Document{
			{
				ID:      "RFC:10",
				Type:    "RFC",
				Title:   "Title: Example",
				Status:  "Draft",
				Authors: []string{`Ops:Team`, `Quoted "Reviewer"`, "true"},
				Path:    "docs/rfc/example.md",
			},
		},
	}

	marshaled := marshalRegistry(registry)
	if !strings.Contains(marshaled, `- id: "RFC:10"`) {
		t.Fatalf("expected id to be quoted, got:\n%s", marshaled)
	}
	if !strings.Contains(marshaled, `title: "Title: Example"`) {
		t.Fatalf("expected title to be quoted, got:\n%s", marshaled)
	}
	if !strings.Contains(marshaled, `- "Ops:Team"`) {
		t.Fatalf("expected colon-containing list item to be quoted, got:\n%s", marshaled)
	}
	if !strings.Contains(marshaled, `- "Quoted \"Reviewer\""`) {
		t.Fatalf("expected quote-containing list item to be escaped, got:\n%s", marshaled)
	}
	if !strings.Contains(marshaled, `- "true"`) {
		t.Fatalf("expected YAML reserved word to be quoted, got:\n%s", marshaled)
	}

	registryPath := writeTestFile(t, "test_registry_marshaled.yaml", marshaled)
	file, err := os.Open(registryPath) // #nosec G304 -- registryPath is created by writeTestFile within the repo root
	if err != nil {
		t.Fatalf("open marshaled registry: %v", err)
	}
	defer func() {
		_ = file.Close()
	}()

	parsed, err := parseRegistry(file)
	if err != nil {
		t.Fatalf("parse marshaled registry: %v", err)
	}
	if len(parsed.Documents) != 1 {
		t.Fatalf("expected one parsed document, got %d", len(parsed.Documents))
	}
	doc := parsed.Documents[0]
	if doc.Title != registry.Documents[0].Title {
		t.Fatalf("expected title %q, got %q", registry.Documents[0].Title, doc.Title)
	}
	if len(doc.Authors) != len(registry.Documents[0].Authors) {
		t.Fatalf("expected %d authors, got %d", len(registry.Documents[0].Authors), len(doc.Authors))
	}
}

func TestCanonicalizersNoChange(t *testing.T) {
	t.Run("reference", func(t *testing.T) {
		if got, changed := canonicalizeReferenceID("custom-1"); got != "custom-1" || changed {
			t.Fatalf("expected no change, got %q changed=%v", got, changed)
		}
	})
	t.Run("type", func(t *testing.T) {
		if got, changed := canonicalizeDocType("custom"); got != "custom" || changed {
			t.Fatalf("expected no change, got %q changed=%v", got, changed)
		}
	})
	t.Run("status", func(t *testing.T) {
		if got, changed := canonicalizeDocStatus("custom"); got != "custom" || changed {
			t.Fatalf("expected no change, got %q changed=%v", got, changed)
		}
	})
	t.Run("path", func(t *testing.T) {
		if got, changed := canonicalizeRegistryPath("../docs/rfc/registry.yaml"); got != "../docs/rfc/registry.yaml" || changed {
			t.Fatalf("expected traversal path to remain unchanged, got %q changed=%v", got, changed)
		}
	})
	t.Run("absolute-like", func(t *testing.T) {
		if got, changed := canonicalizeRegistryPath("/docs/rfc/registry.yaml"); got != "/docs/rfc/registry.yaml" || changed {
			t.Fatalf("expected leading slash path to remain unchanged, got %q changed=%v", got, changed)
		}
	})
	t.Run("dot", func(t *testing.T) {
		if got, changed := canonicalizeRegistryPath("."); got != "." || changed {
			t.Fatalf("expected dot path to remain unchanged, got %q changed=%v", got, changed)
		}
	})
}
