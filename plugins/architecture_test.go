package plugins

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPluginsDoNotImportDomain enforces that plugin implementation packages do not
// import the internal domain model directly. Plugins must depend only on the
// stable facades in pkg/datasetapi or pkg/pluginapi. The test deliberately
// skips the test fixture helper package at plugins/testhelper which is an
// explicit escape hatch for building facade fixtures from domain entities.
func TestPluginsDoNotImportDomain(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("cannot get working dir: %v", err)
	}

	root := wd // this file lives in the plugins directory

	forbidden := "colonycore/pkg/domain"
	fixtureDir := filepath.Join(root, "testhelper")

	var violations []string

	walkErr := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error { //nolint:wrapcheck
		if err != nil { // propagate filesystem errors
			return err
		}
		// Skip the fixture helper subtree entirely
		if d.IsDir() && path == fixtureDir {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		// Ignore this test file itself just in case
		if path == filepath.Join(root, "architecture_test.go") {
			return nil
		}

		// #nosec G304 -- path comes from controlled WalkDir over the local repository tree,
		// restricted to .go source files under plugins (excluding fixture subtree); no external input.
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		lines := strings.Split(string(data), "\n")
		inImport := false
		for _, raw := range lines {
			line := strings.TrimSpace(raw)
			if !inImport {
				if strings.HasPrefix(line, "import (") {
					inImport = true
					continue
				}
				if strings.HasPrefix(line, "import ") { // single import form
					if q := extractQuoted(line); q == forbidden {
						violations = append(violations, path)
					}
				}
				continue
			}
			// inside import block
			if line == ")" {
				inImport = false
				continue
			}
			if q := extractQuoted(line); q == forbidden {
				violations = append(violations, path)
			}
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("walk plugins dir: %v", walkErr)
	}

	if len(violations) > 0 {
		for _, v := range violations {
			// Report each offending file for clarity
			// (Keep error format stable for grepping / future tooling.)
			//nolint:lll // readability > line length here
			t.Errorf("plugin file imports forbidden %s: %s", forbidden, v)
		}
		// Fail fast after listing all violations.
		// Using Fatalf would hide multiple offenders; we collect first.
		// So just mark the test failed here.
		// (t.FailNow not used to allow all errors to surface.)
	}
}

// extractQuoted mirrors the helper in pkg/domain/architecture_test.go but is
// duplicated locally to keep the test self-contained and avoid importing domain.
func extractQuoted(line string) string {
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
