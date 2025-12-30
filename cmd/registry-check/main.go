// Command registry-check validates the docs/rfc/registry.yaml file adheres to
// simple structural and semantic expectations enforced for governance.
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Document struct {
	ID            string
	Type          string
	Title         string
	Status        string
	Created       string
	Date          string
	LastUpdated   string
	Authors       []string
	Stakeholders  []string
	Reviewers     []string
	Quorum        string
	TargetRelease string
	Owners        []string
	Deciders      []string
	LinkedAnnexes []string
	LinkedADRs    []string
	LinkedRFCs    []string
	Path          string
}

type Registry struct {
	Documents []Document
}

var (
	allowedTypes  = map[string]struct{}{"RFC": {}, "Annex": {}, "ADR": {}}
	statusMap     = map[string]string{"draft": "Draft", "planned": "Planned", "accepted": "Accepted", "superseded": "Superseded", "archived": "Archived"}
	allowedStatus = buildAllowedStatus()
	exitFunc      = os.Exit
)

// buildAllowedStatus builds a set of canonical document status strings derived from statusMap.
// The returned map has canonical status values as keys and empty structs as values for efficient membership checks.
func buildAllowedStatus() map[string]struct{} {
	m := make(map[string]struct{}, len(statusMap))
	for _, canonical := range statusMap {
		m[canonical] = struct{}{}
	}
	return m
}

// main runs the command-line interface using the program arguments and exits
// the process with the status code returned by cli.
func main() {
	code := cli(os.Args[1:], os.Stdout, os.Stderr)
	exitFunc(code)
}

func cli(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("registry-check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var registryPath string
	fs.StringVar(&registryPath, "registry", "docs/rfc/registry.yaml", "path to registry yaml")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if err := run(registryPath); err != nil {
		if _, writeErr := fmt.Fprintf(stderr, "Registry validation failed: %v\n", err); writeErr != nil {
			return 1
		}
		return 1
	}
	if _, writeErr := fmt.Fprintln(stdout, "Registry validation passed."); writeErr != nil {
		return 1
	}
	return 0
}

// validatePath ensures the registry file path is within the repository tree and
// not an absolute or path-traversing reference. This mitigates G304 concerns
// around variable-based file inclusion.
func validatePath(p string) (string, error) {
	if strings.TrimSpace(p) == "" {
		return "", fmt.Errorf("empty path")
	}
	if filepath.IsAbs(p) {
		return "", fmt.Errorf("absolute paths not allowed: %s", p)
	}
	clean := filepath.Clean(p)
	if strings.Contains(clean, "..") { // prevents traversal outside working dir
		return "", fmt.Errorf("path traversal not allowed: %s", p)
	}
	return clean, nil
}

// run validates the given registry path, parses the registry file, and verifies each document and its recorded status.
//
// It validates the registry path, opens and parses the registry file, and ensures the registry contains at least one document.
// For each document it performs structural validation and verifies the document's declared status against the document file.
// Returns an error if path validation, file I/O, parsing, structural validation, status verification, or an empty documents entry occur;
// document-level errors are annotated with the document index (e.g., "documents[0]: ...").
func run(registryPath string) (err error) {
	safePath, vErr := validatePath(registryPath)
	if vErr != nil {
		return vErr
	}
	file, err := os.Open(safePath) // #nosec G304: path validated by validatePath
	if err != nil {
		return fmt.Errorf("read registry: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close registry: %w", cerr)
		}
	}()

	registry, err := parseRegistry(file)
	if err != nil {
		return fmt.Errorf("parse registry: %w", err)
	}

	if len(registry.Documents) == 0 {
		return errors.New("documents entry is empty")
	}

	for i, doc := range registry.Documents {
		if err := validateDocument(doc); err != nil {
			return fmt.Errorf("documents[%d]: %w", i, err)
		}
		if err := validateDocumentStatus(doc); err != nil {
			return fmt.Errorf("documents[%d]: %w", i, err)
		}
	}

	return nil
}

func parseRegistry(file *os.File) (*Registry, error) {
	scanner := bufio.NewScanner(file)
	var registry Registry

	var currentDoc *Document
	var listField string

	for lineNum := 1; scanner.Scan(); lineNum++ {
		line := scanner.Text()
		if trimmed := strings.TrimSpace(line); trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		indent := countLeadingSpaces(line)
		trimmed := strings.TrimSpace(line)

		if indent == 0 {
			if trimmed != "documents:" {
				return nil, fmt.Errorf("line %d: expected 'documents:'", lineNum)
			}
			continue
		}

		if indent == 2 && strings.HasPrefix(trimmed, "- ") {
			if currentDoc != nil {
				registry.Documents = append(registry.Documents, *currentDoc)
			}
			currentDoc = &Document{}
			listField = ""

			key, value, err := splitKeyValue(trimmed[2:])
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			if err := assignScalar(currentDoc, key, value); err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			continue
		}

		if currentDoc == nil {
			return nil, fmt.Errorf("line %d: encountered field before any document", lineNum)
		}

		if indent == 4 {
			key, value, err := splitKeyValue(trimmed)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			if value == "" {
				listField = key
				resetList(currentDoc, key)
			} else {
				listField = ""
				if err := assignScalar(currentDoc, key, value); err != nil {
					return nil, fmt.Errorf("line %d: %w", lineNum, err)
				}
			}
			continue
		}

		if indent == 6 && strings.HasPrefix(trimmed, "- ") {
			if listField == "" {
				return nil, fmt.Errorf("line %d: list item without active list field", lineNum)
			}
			item := strings.TrimSpace(trimmed[2:])
			if err := appendList(currentDoc, listField, item); err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			continue
		}

		return nil, fmt.Errorf("line %d: unsupported structure", lineNum)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if currentDoc != nil {
		registry.Documents = append(registry.Documents, *currentDoc)
	}

	return &registry, nil
}

func countLeadingSpaces(s string) int {
	count := 0
	for _, r := range s {
		if r == ' ' {
			count++
		} else {
			break
		}
	}
	return count
}

func splitKeyValue(part string) (string, string, error) {
	idx := strings.Index(part, ":")
	if idx == -1 {
		return "", "", fmt.Errorf("missing ':' delimiter in %q", part)
	}
	key := strings.TrimSpace(part[:idx])
	value := strings.TrimSpace(part[idx+1:])
	return key, value, nil
}

func assignScalar(doc *Document, key, value string) error {
	switch key {
	case "id":
		doc.ID = value
	case "type":
		doc.Type = value
	case "title":
		doc.Title = value
	case "status":
		doc.Status = value
	case "created":
		doc.Created = value
	case "date":
		doc.Date = value
	case "last_updated":
		doc.LastUpdated = value
	case "quorum":
		doc.Quorum = value
	case "target_release":
		doc.TargetRelease = value
	case "path":
		doc.Path = value
	default:
		return fmt.Errorf("unsupported scalar field %q", key)
	}
	return nil
}

func resetList(doc *Document, key string) {
	switch key {
	case "authors":
		doc.Authors = nil
	case "stakeholders":
		doc.Stakeholders = nil
	case "reviewers":
		doc.Reviewers = nil
	case "owners":
		doc.Owners = nil
	case "deciders":
		doc.Deciders = nil
	case "linked_annexes":
		doc.LinkedAnnexes = nil
	case "linked_adrs":
		doc.LinkedADRs = nil
	case "linked_rfcs":
		doc.LinkedRFCs = nil
	default:
		// ignore unknown list keys until we encounter items where we can error
	}
}

func appendList(doc *Document, key, value string) error {
	switch key {
	case "authors":
		doc.Authors = append(doc.Authors, value)
	case "stakeholders":
		doc.Stakeholders = append(doc.Stakeholders, value)
	case "reviewers":
		doc.Reviewers = append(doc.Reviewers, value)
	case "owners":
		doc.Owners = append(doc.Owners, value)
	case "deciders":
		doc.Deciders = append(doc.Deciders, value)
	case "linked_annexes":
		doc.LinkedAnnexes = append(doc.LinkedAnnexes, value)
	case "linked_adrs":
		doc.LinkedADRs = append(doc.LinkedADRs, value)
	case "linked_rfcs":
		doc.LinkedRFCs = append(doc.LinkedRFCs, value)
	default:
		return fmt.Errorf("unsupported list field %q", key)
	}
	return nil
}

// validateDocument checks that a Document has all required fields and that any
// provided date fields are valid (YYYY-MM-DD). It returns an error describing
// the first problem found, such as a missing or invalid id, type, title, status,
// path, or a malformed created/date/last_updated value.
func validateDocument(doc Document) error {
	if doc.ID == "" {
		return errors.New("missing id")
	}
	if doc.Type == "" {
		return errors.New("missing type")
	}
	if _, ok := allowedTypes[doc.Type]; !ok {
		return fmt.Errorf("invalid type %q", doc.Type)
	}
	if doc.Title == "" {
		return errors.New("missing title")
	}
	if doc.Status == "" {
		return errors.New("missing status")
	}
	if _, ok := allowedStatus[doc.Status]; !ok {
		return fmt.Errorf("invalid status %q", doc.Status)
	}
	if doc.Path == "" {
		return errors.New("missing path")
	}

	if doc.Created != "" {
		if err := validateDate(doc.Created); err != nil {
			return fmt.Errorf("created: %w", err)
		}
	}
	if doc.Date != "" {
		if err := validateDate(doc.Date); err != nil {
			return fmt.Errorf("date: %w", err)
		}
	}
	if doc.LastUpdated != "" {
		if err := validateDate(doc.LastUpdated); err != nil {
			return fmt.Errorf("last_updated: %w", err)
		}
	}

	return nil
}

// validateDocumentStatus verifies that the status recorded in the registry for the given Document
// matches the canonical status read from the document file.
// It reads the document's status from the file at doc.Path and returns an error if reading fails
// or if the canonical status extracted from the file differs from doc.Status.
func validateDocumentStatus(doc Document) error {
	status, err := readDocumentStatus(doc.Path)
	if err != nil {
		return fmt.Errorf("status check for %s: %w", doc.ID, err)
	}
	if status != doc.Status {
		return fmt.Errorf("status mismatch for %s (%s): registry %q, doc %q", doc.ID, doc.Path, doc.Status, status)
	}
	return nil
}

// readDocumentStatus reads the document file at the given path and returns the document's canonical status.
//
// It validates the provided path, opens the file, and scans up to the first 120 non-empty lines to discover a status.
// Two discovery modes are supported:
// - A "## Status" header where the next non-empty line supplies the status value.
// - An inline "Status:" line anywhere that supplies the status value.
// The returned status is normalized to the registry's canonical form. If the file cannot be opened or scanned,
// if the status token is missing or invalid, or if a "## Status" header is present without a following value,
// an error is returned.
func readDocumentStatus(path string) (status string, err error) {
	const statusScanLimit = 120

	safePath, err := validatePath(path)
	if err != nil {
		return "", err
	}
	file, err := os.Open(safePath) // #nosec G304: path validated by validatePath
	if err != nil {
		return "", fmt.Errorf("read document %q: %w", safePath, err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close document: %w", cerr)
		}
	}()

	scanner := bufio.NewScanner(file)
	expectStatusLine := false
	for lineNum := 1; scanner.Scan(); lineNum++ {
		if lineNum > statusScanLimit {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if expectStatusLine {
			docStatus, statusErr := canonicalizeStatus(line)
			if statusErr != nil {
				return "", statusErr
			}
			return docStatus, nil
		}
		if line == "## Status" {
			expectStatusLine = true
			continue
		}
		docStatus, ok, statusErr := parseInlineStatus(line)
		if statusErr != nil {
			return "", statusErr
		}
		if ok {
			return docStatus, nil
		}
	}

	if scanErr := scanner.Err(); scanErr != nil {
		return "", scanErr
	}
	if expectStatusLine {
		return "", fmt.Errorf("status header without value in %s", path)
	}
	return "", fmt.Errorf("status not found in %s", path)
}

// parseInlineStatus examines a single line for an inline "Status:" token and, if present, returns the canonical status.
// If the line contains a status token the second return value is `true`; the first return is the canonical status and the third is a canonicalization error, if any.
func parseInlineStatus(line string) (string, bool, error) {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.TrimLeft(trimmed, "-* ")
	if !strings.HasPrefix(trimmed, "Status:") {
		return "", false, nil
	}
	raw := strings.TrimSpace(strings.TrimPrefix(trimmed, "Status:"))
	status, err := canonicalizeStatus(raw)
	if err != nil {
		return "", true, err
	}
	return status, true, nil
}

// canonicalizeStatus extracts the leading status token from value and returns the corresponding canonical status string.
// It returns an error if no token can be extracted or if the token is not recognized by the package's status mapping.
func canonicalizeStatus(value string) (string, error) {
	token := extractStatusToken(value)
	if token == "" {
		return "", fmt.Errorf("status value missing")
	}
	canonical, ok := statusMap[strings.ToLower(token)]
	if !ok {
		return "", fmt.Errorf("invalid status %q", token)
	}
	return canonical, nil
}

// extractStatusToken extracts the first whitespace-separated token from value and trims surrounding punctuation.
// If value is empty or contains no fields, it returns the empty string. The trimming removes common punctuation
// characters such as '(', ')', '.', ',', ';', ':' and '-'.
func extractStatusToken(value string) string {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return ""
	}
	return strings.Trim(fields[0], "().,;:-")
}

// validateDate checks that value is a date in YYYY-MM-DD format.
// It returns an error describing the invalid input when parsing fails.
func validateDate(value string) error {
	if _, err := time.Parse("2006-01-02", value); err != nil {
		return fmt.Errorf("invalid date %q", value)
	}
	return nil
}
