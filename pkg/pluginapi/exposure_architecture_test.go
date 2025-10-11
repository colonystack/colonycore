package pluginapi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReadOnlyEntityExposure enforces that plugin-facing entity exposures remain
// read-only via interfaces or DTO structs with unexported fields.
// Rules:
// 1. No exported struct type (except those suffixed with "Error") may declare exported fields.
// 2. No *View interface method may start with "Set" (mutators disallowed).
// 3. Core domain entity type names must not be re-declared here (Organism, Cohort, HousingUnit, BreedingUnit, Procedure, Protocol, Project).
func TestReadOnlyEntityExposure(t *testing.T) {
	// Domain entity names that must not appear as exported types in pluginapi.
	forbiddenTypeNames := map[string]struct{}{
		"Organism": {}, "Cohort": {}, "HousingUnit": {}, "BreedingUnit": {}, "Procedure": {}, "Protocol": {}, "Project": {},
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	entries, err := os.ReadDir(wd)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}

	var violations []string

	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") { // skip non-Go
			continue
		}
		// Skip generated files if any appear later.
		if strings.HasSuffix(name, "_generated.go") {
			continue
		}
		path := filepath.Join(wd, name)
		data, err := os.ReadFile(path) // #nosec G304 - constrained to package dir
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		lines := strings.Split(string(data), "\n")

		inStruct := false
		structName := ""
		inInterface := false
		interfaceName := ""

		for _, raw := range lines {
			line := strings.TrimSpace(raw)
			if line == "" || strings.HasPrefix(line, "//") {
				continue
			}

			// Detect start of type declarations.
			if strings.HasPrefix(line, "type ") {
				// form: type Name struct {  OR type Name interface {
				after := strings.TrimPrefix(line, "type ")
				// Extract identifier (up to first space or tab)
				ident := after
				if i := strings.IndexAny(ident, " \t"); i != -1 {
					ident = ident[:i]
				}
				if _, forbidden := forbiddenTypeNames[ident]; forbidden {
					violations = append(violations, "forbidden exported type redeclared: "+ident)
				}
				if strings.Contains(line, " struct") && strings.HasSuffix(line, "{") {
					inStruct = true
					structName = ident
					continue
				}
				if strings.Contains(line, " interface") && strings.HasSuffix(line, "{") {
					inInterface = true
					interfaceName = ident
					continue
				}
			}

			if inStruct {
				if line == "}" { // end of struct
					inStruct = false
					structName = ""
					continue
				}
				// Skip anonymous blank lines inside struct or comments already filtered.
				// Field line pattern: FieldName <rest>
				// We only care if first token starts with uppercase (exported) and struct not suffixed with Error.
				if structName != "" && !strings.HasSuffix(structName, "Error") && !strings.HasSuffix(name, "_test.go") {
					// get first token (up to space or tab)
					tok := line
					if i := strings.IndexAny(tok, " \t"); i != -1 {
						tok = tok[:i]
					}
					// Exclude embedded unexported fields (lowercase) and tags/backticks only lines.
					if tok != "" && isExportedIdent(tok) && !strings.HasPrefix(tok, "//") {
						// Defensive: ignore cases where token includes '(' which would indicate method decl (shouldn't happen in struct body)
						if !strings.Contains(tok, "(") {
							violations = append(violations, "exported field '"+tok+"' in struct '"+structName+"'")
						}
					}
				}
				continue
			}

			if inInterface {
				if line == "}" { // end of interface
					inInterface = false
					interfaceName = ""
					continue
				}
				if interfaceName != "" && strings.HasSuffix(interfaceName, "View") {
					// Method signature pattern: Name(...
					// Extract method ident up to '(' if present.
					if idx := strings.Index(line, "("); idx > 0 {
						mname := line[:idx]
						// Trim potential leading * or whitespace (should not exist inside interface)
						mname = strings.TrimSpace(strings.TrimPrefix(mname, "*"))
						if strings.HasPrefix(mname, "Set") && isExportedIdent(mname) {
							violations = append(violations, "mutator method '"+mname+"' in interface '"+interfaceName+"'")
						}
					}
				}
				continue
			}
		}
	}

	if len(violations) > 0 {
		t.Fatalf("read-only exposure contract violated:\n%s", strings.Join(violations, "\n"))
	}
}

func isExportedIdent(s string) bool {
	if s == "" {
		return false
	}
	r := rune(s[0])
	return r >= 'A' && r <= 'Z'
}
