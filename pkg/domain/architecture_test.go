package domain

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDomainDoesNotImportInternal enforces the architectural rule that the domain
// layer must not depend on any internal implementation packages. This is
// intentionally redundant with import-boss to give fast, local, intention‑revealing
// feedback close to the code when editing the domain layer.
func TestDomainDoesNotImportInternal(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("cannot get working dir: %v", err)
	}

	entries, err := os.ReadDir(wd)
	if err != nil {
		t.Fatalf("cannot read dir: %v", err)
	}

	violations := 0

	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		// Guard against any unexpected path traversal (shouldn't occur from ReadDir names)
		if strings.ContainsRune(name, os.PathSeparator) {
			continue
		}
		path := filepath.Join(wd, name)
		// #nosec G304 -- path is derived from controlled directory entries within the same package
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		lines := strings.Split(string(data), "\n")
		inBlock := false
		for _, raw := range lines {
			line := strings.TrimSpace(raw)
			if !inBlock {
				if strings.HasPrefix(line, "import (") {
					inBlock = true
					continue
				}
				if strings.HasPrefix(line, "import ") { // single-line import
					if q := extractQuoted(line); q != "" && strings.Contains(q, "/internal/") {
						violations++
						t.Errorf("domain package must not import internal packages: %s (%s)", q, name)
					}
				}
				continue
			}
			if line == ")" { // end of block
				inBlock = false
				continue
			}
			if q := extractQuoted(line); q != "" && strings.Contains(q, "/internal/") {
				violations++
				t.Errorf("domain package must not import internal packages: %s (%s)", q, name)
			}
		}
	}

	if violations > 0 {
		t.Fatalf("found %d forbidden internal imports in domain package", violations)
	}
}

// (Optional) Future enhancement idea: maintain a small allowlist of external dependencies
// (std lib + explicitly approved pure libraries) and fail if new deps are added here.
// Keeping scope minimal for now.

// extractQuoted returns the first double-quoted string literal in a line, or "".
func extractQuoted(line string) string {
	// crude but sufficient for import lines; avoids pulling in disallowed parser packages
	start := strings.Index(line, "\"")
	if start == -1 {
		return ""
	}
	end := strings.Index(line[start+1:], "\"")
	if end == -1 {
		return ""
	}
	return line[start+1 : start+1+end]
}
