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
	allowedStatus = map[string]struct{}{"Draft": {}, "Planned": {}, "Accepted": {}, "Superseded": {}, "Archived": {}}
	exitFunc      = os.Exit
)

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

func validateDate(value string) error {
	if _, err := time.Parse("2006-01-02", value); err != nil {
		return fmt.Errorf("invalid date %q", value)
	}
	return nil
}
