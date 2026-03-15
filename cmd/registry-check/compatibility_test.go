package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestRegistryCompatibilityFixtures(t *testing.T) {
	paths := loadRegistryFixturePaths(t, "compat")

	for _, path := range paths {
		t.Run(strings.TrimSuffix(filepath.Base(path), ".yaml"), func(t *testing.T) {
			original, err := os.ReadFile(path) // #nosec G304 -- path is loaded from curated repository fixtures
			if err != nil {
				t.Fatalf("read compatibility fixture %s: %v", path, err)
			}
			if err := run(path); err != nil {
				t.Fatalf("expected compatibility fixture %s to validate unchanged, got %v", path, err)
			}

			tempRegistry := writeTestFile(t, filepath.Base(path), string(original))
			fixes, err := fixRegistryFile(tempRegistry)
			if err != nil {
				t.Fatalf("fix compatibility fixture %s: %v", path, err)
			}
			if fixes != 0 {
				t.Fatalf("expected canonical compatibility fixture %s to need no fixes, got %d", path, fixes)
			}

			rewritten, err := os.ReadFile(tempRegistry) // #nosec G304 -- tempRegistry is created by writeTestFile within the repo root
			if err != nil {
				t.Fatalf("read rewritten compatibility fixture %s: %v", tempRegistry, err)
			}
			if string(rewritten) != string(original) {
				t.Fatalf("expected compatibility fixture %s to remain byte-identical after fixer", path)
			}
		})
	}
}

func loadRegistryFixturePaths(t *testing.T, category string) []string {
	t.Helper()

	fixturesDir := filepath.Join(registryFixtureRoot, category)
	entries, err := os.ReadDir(fixturesDir)
	if err != nil {
		t.Fatalf("read fixture directory %s: %v", fixturesDir, err)
	}

	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		fixtureAbsPath := filepath.Join(fixturesDir, entry.Name())
		fixturePath, err := filepath.Rel(registryRepoRoot, fixtureAbsPath)
		if err != nil {
			t.Fatalf("resolve fixture path relative to repo root for %s: %v", fixtureAbsPath, err)
		}
		if strings.Contains(fixturePath, "..") {
			t.Fatalf("fixture path escapes repo root: %s", fixtureAbsPath)
		}
		paths = append(paths, filepath.ToSlash(filepath.Clean(fixturePath)))
	}

	if len(paths) == 0 {
		t.Fatalf("fixture category %q is empty at %s", category, fixturesDir)
	}

	sort.Strings(paths)
	return paths
}
