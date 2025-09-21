package main

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
)

func writeTempRegistry(t *testing.T, content string) string {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), "registry-*.yaml")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	if _, err := file.WriteString(content); err != nil {
		if closeErr := file.Close(); closeErr != nil {
			t.Fatalf("close temp file after write failure: %v", closeErr)
		}
		t.Fatalf("write temp file: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close temp file: %v", err)
	}
	return file.Name()
}

func TestRunSuccess(t *testing.T) {
	content := strings.Join([]string{
		"documents:",
		"  - id: RFC-0001",
		"    type: RFC",
		"    title: Sample",
		"    status: Draft",
		"    path: docs/rfc/sample.md",
		"    authors:",
		"      - Alice",
		"    linked_adrs:",
		"      - ADR-0001",
		"",
	}, "\n")

	path := writeTempRegistry(t, content)
	if err := run(path); err != nil {
		t.Fatalf("run() returned error: %v", err)
	}
}

func TestRunMissingFile(t *testing.T) {
	if err := run("does-not-exist.yaml"); err == nil {
		t.Fatal("expected error when file is missing")
	}
}

func TestRunInvalidStatus(t *testing.T) {
	content := strings.Join([]string{
		"documents:",
		"  - id: RFC-0002",
		"    type: RFC",
		"    title: Invalid Status",
		"    status: Unknown",
		"    path: docs/rfc/invalid.md",
		"",
	}, "\n")

	path := writeTempRegistry(t, content)
	if err := run(path); err == nil || !strings.Contains(err.Error(), "invalid status") {
		t.Fatalf("expected invalid status error, got %v", err)
	}
}

func TestRunEmptyDocuments(t *testing.T) {
	content := "documents:\n"
	path := writeTempRegistry(t, content)
	if err := run(path); err == nil || !strings.Contains(err.Error(), "documents entry is empty") {
		t.Fatalf("expected empty documents error, got %v", err)
	}
}

func TestParseRegistryLists(t *testing.T) {
	content := strings.Join([]string{
		"documents:",
		"  - id: ADR-0001",
		"    type: ADR",
		"    title: Example ADR",
		"    status: Draft",
		"    path: docs/adr/0001-example.md",
		"    deciders:",
		"      - Team A",
		"    linked_rfcs:",
		"      - RFC-0001",
		"",
	}, "\n")
	path := writeTempRegistry(t, content)

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open temp file: %v", err)
	}
	t.Cleanup(func() {
		if err := file.Close(); err != nil {
			t.Fatalf("close temp file: %v", err)
		}
	})

	registry, err := parseRegistry(file)
	if err != nil {
		t.Fatalf("parseRegistry returned error: %v", err)
	}
	if len(registry.Documents) != 1 {
		t.Fatalf("expected 1 document, got %d", len(registry.Documents))
	}
	doc := registry.Documents[0]
	if len(doc.Deciders) != 1 || doc.Deciders[0] != "Team A" {
		t.Fatalf("expected deciders parsed, got %#v", doc.Deciders)
	}
	if len(doc.LinkedRFCs) != 1 || doc.LinkedRFCs[0] != "RFC-0001" {
		t.Fatalf("expected linked RFCs parsed, got %#v", doc.LinkedRFCs)
	}
}

func TestHelpers(t *testing.T) {
	if count := countLeadingSpaces("    four"); count != 4 {
		t.Fatalf("countLeadingSpaces expected 4, got %d", count)
	}

	key, value, err := splitKeyValue("status: Draft")
	if err != nil {
		t.Fatalf("splitKeyValue returned error: %v", err)
	}
	if key != "status" || value != "Draft" {
		t.Fatalf("unexpected key/value: %q %q", key, value)
	}

	if _, _, err := splitKeyValue("invalid"); err == nil {
		t.Fatal("expected error for missing delimiter")
	}
}

func TestParseRegistryErrors(t *testing.T) {
	cases := map[string]string{
		"missing header": "invalid:\n",
		"field before doc": strings.Join([]string{
			"documents:",
			"    id: bad",
		}, "\n"),
		"unknown list": strings.Join([]string{
			"documents:",
			"  - id: RFC-100",
			"    type: RFC",
			"    title: T",
			"    status: Draft",
			"    path: docs/rfc/100.md",
			"      - stray",
		}, "\n"),
		"unsupported structure": strings.Join([]string{
			"documents:",
			"  - id: RFC-101",
			"    type: RFC",
			"   title: T",
		}, "\n"),
	}

	for name, content := range cases {
		path := writeTempRegistry(t, content)
		file, err := os.Open(path)
		if err != nil {
			t.Fatalf("%s: open temp file: %v", name, err)
		}
		registry, err := parseRegistry(file)
		if closeErr := file.Close(); closeErr != nil {
			t.Fatalf("%s: close temp file: %v", name, closeErr)
		}
		if err == nil {
			t.Fatalf("%s: expected error, got registry %#v", name, registry)
		}
	}
}

func TestAssignScalarAndLists(t *testing.T) {
	doc := &Document{}
	if err := assignScalar(doc, "id", "RFC-10"); err != nil {
		t.Fatalf("assignScalar id error: %v", err)
	}
	if err := assignScalar(doc, "type", "RFC"); err != nil {
		t.Fatalf("assignScalar type error: %v", err)
	}
	if err := assignScalar(doc, "title", "Title"); err != nil {
		t.Fatalf("assignScalar title error: %v", err)
	}
	if err := assignScalar(doc, "status", "Draft"); err != nil {
		t.Fatalf("assignScalar status error: %v", err)
	}
	if err := assignScalar(doc, "created", "2025-01-01"); err != nil {
		t.Fatalf("assignScalar created error: %v", err)
	}
	if err := assignScalar(doc, "date", "2025-01-02"); err != nil {
		t.Fatalf("assignScalar date error: %v", err)
	}
	if err := assignScalar(doc, "last_updated", "2025-01-03"); err != nil {
		t.Fatalf("assignScalar last_updated error: %v", err)
	}
	if err := assignScalar(doc, "quorum", "3"); err != nil {
		t.Fatalf("assignScalar quorum error: %v", err)
	}
	if err := assignScalar(doc, "target_release", "v0.2.0"); err != nil {
		t.Fatalf("assignScalar target_release error: %v", err)
	}
	if err := assignScalar(doc, "path", "docs/rfc/10.md"); err != nil {
		t.Fatalf("assignScalar path error: %v", err)
	}

	resetList(doc, "authors")
	resetList(doc, "stakeholders")
	resetList(doc, "reviewers")
	resetList(doc, "owners")
	resetList(doc, "deciders")
	resetList(doc, "linked_annexes")
	resetList(doc, "linked_adrs")
	resetList(doc, "linked_rfcs")

	if err := appendList(doc, "authors", "Alice"); err != nil {
		t.Fatalf("appendList authors error: %v", err)
	}
	if err := appendList(doc, "stakeholders", "Stakeholder"); err != nil {
		t.Fatalf("appendList stakeholders error: %v", err)
	}
	if err := appendList(doc, "reviewers", "Reviewer"); err != nil {
		t.Fatalf("appendList reviewers error: %v", err)
	}
	if err := appendList(doc, "owners", "Owner"); err != nil {
		t.Fatalf("appendList owners error: %v", err)
	}
	if err := appendList(doc, "deciders", "Decider"); err != nil {
		t.Fatalf("appendList deciders error: %v", err)
	}
	if err := appendList(doc, "linked_annexes", "Annex-1"); err != nil {
		t.Fatalf("appendList linked_annexes error: %v", err)
	}
	if err := appendList(doc, "linked_adrs", "ADR-1"); err != nil {
		t.Fatalf("appendList linked_adrs error: %v", err)
	}
	if err := appendList(doc, "linked_rfcs", "RFC-1"); err != nil {
		t.Fatalf("appendList linked_rfcs error: %v", err)
	}

	if err := appendList(doc, "unknown", "value"); err == nil {
		t.Fatal("expected error for unknown list field")
	}
	if err := assignScalar(doc, "unknown", "value"); err == nil {
		t.Fatal("expected error for unknown scalar field")
	}
}

func TestValidateDocument(t *testing.T) {
	valid := Document{ID: "ID", Type: "RFC", Title: "T", Status: "Draft", Path: "docs/rfc/file.md"}
	if err := validateDocument(valid); err != nil {
		t.Fatalf("expected valid document, got %v", err)
	}

	for name, doc := range map[string]Document{
		"missing id":     {},
		"missing type":   {ID: "ID"},
		"invalid type":   {ID: "ID", Type: "BAD"},
		"missing title":  {ID: "ID", Type: "RFC"},
		"missing status": {ID: "ID", Type: "RFC", Title: "T"},
		"invalid status": {ID: "ID", Type: "RFC", Title: "T", Status: "BAD"},
		"missing path":   {ID: "ID", Type: "RFC", Title: "T", Status: "Draft"},
	} {
		if err := validateDocument(doc); err == nil {
			t.Fatalf("expected error for %s", name)
		}
	}
	if err := validateDocument(Document{ID: "ID", Type: "RFC", Title: "T", Status: "Draft", Path: "docs/rfc/file.md", Created: "bad-date"}); err == nil {
		t.Fatal("expected date validation error")
	}
}

func TestValidateDate(t *testing.T) {
	if err := validateDate("2025-01-01"); err != nil {
		t.Fatalf("validateDate expected success: %v", err)
	}
	if err := validateDate("01-01-2025"); err == nil {
		t.Fatal("expected invalid date error")
	}
}

func TestCLI(t *testing.T) {
	content := strings.Join([]string{
		"documents:",
		"  - id: RFC-010",
		"    type: RFC",
		"    title: CLI",
		"    status: Draft",
		"    path: docs/rfc/cli.md",
		"",
	}, "\n")
	path := writeTempRegistry(t, content)
	var out, errBuf bytes.Buffer
	code := cli([]string{"-registry", path}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr=%s)", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "Registry validation passed") {
		t.Fatalf("expected success message, got %q", out.String())
	}

	code = cli([]string{"-registry", "missing.yaml"}, &out, &errBuf)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(errBuf.String(), "Registry validation failed") {
		t.Fatalf("expected failure message, got %q", errBuf.String())
	}

	errBuf.Reset()
	code = cli([]string{"--invalid-flag"}, &out, &errBuf)
	if code != 2 {
		t.Fatalf("expected exit code 2 for flag error, got %d", code)
	}
}

func TestMainFunction(t *testing.T) {
	content := strings.Join([]string{
		"documents:",
		"  - id: RFC-011",
		"    type: RFC",
		"    title: Main",
		"    status: Draft",
		"    path: docs/rfc/main.md",
		"",
	}, "\n")
	path := writeTempRegistry(t, content)
	origExit := exitFunc
	defer func() { exitFunc = origExit }()
	exitCode := -1
	exitFunc = func(code int) { exitCode = code }
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"registry-check", "-registry", path}
	main()
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
}

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) {
	return 0, w.err
}

func TestCLIWriteFailures(t *testing.T) {
	content := strings.Join([]string{
		"documents:",
		"  - id: RFC-012",
		"    type: RFC",
		"    title: Failure",
		"    status: Draft",
		"    path: docs/rfc/write.md",
		"",
	}, "\n")
	path := writeTempRegistry(t, content)

	stdoutFail := failingWriter{err: errors.New("write failure")}
	code := cli([]string{"-registry", path}, stdoutFail, &bytes.Buffer{})
	if code != 1 {
		t.Fatalf("expected exit code 1 when stdout write fails, got %d", code)
	}

	stderrFail := failingWriter{err: errors.New("write failure")}
	code = cli([]string{"-registry", "missing.yaml"}, &bytes.Buffer{}, stderrFail)
	if code != 1 {
		t.Fatalf("expected exit code 1 when stderr write fails, got %d", code)
	}
}
