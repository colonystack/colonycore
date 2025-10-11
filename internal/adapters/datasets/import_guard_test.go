package datasets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoInternalCoreImports ensures production code does not depend on internal/core.
func TestNoInternalCoreImports(t *testing.T) {
	const forbidden = "\"colonycore/internal/core\""
	err := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		// #nosec G304: paths provided by WalkDir within repository.
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("production file %s must not import colonycore/internal/core", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk imports: %v", err)
	}
}
