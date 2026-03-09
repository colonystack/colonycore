package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"colonycore/pkg/datasetapi"
)

func TestCLIUnknownCommand(t *testing.T) {
	var stdout, stderr strings.Builder
	code := cli([]string{"unknown"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "Usage:") {
		t.Fatalf("expected usage output, got %q", stderr.String())
	}
}

func TestLintDatasetCLIRequiresPath(t *testing.T) {
	var stdout, stderr strings.Builder
	code := cli([]string{"lint", "dataset"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "provide at least one file or directory") {
		t.Fatalf("expected missing path message, got %q", stderr.String())
	}
}

func TestLintDatasetCLIValidFile(t *testing.T) {
	dir := t.TempDir()
	validPath := filepath.Join(dir, "valid.json")
	writeTemplateFile(t, validPath, validTemplateDescriptor())

	var stdout, stderr strings.Builder
	code := cli([]string{"lint", "dataset", validPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr=%q)", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "dataset lint passed") {
		t.Fatalf("expected pass summary, got %q", stdout.String())
	}
}

func TestLintDatasetCLIInvalidFile(t *testing.T) {
	dir := t.TempDir()
	invalidPath := filepath.Join(dir, "invalid.json")
	invalid := validTemplateDescriptor()
	invalid.Key = ""
	writeTemplateFile(t, invalidPath, invalid)

	var stdout, stderr strings.Builder
	code := cli([]string{"lint", "dataset", invalidPath}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "FAIL") || !strings.Contains(stderr.String(), "key") {
		t.Fatalf("expected field-level failure output, got %q", stderr.String())
	}
}

func TestLintDatasetCLIDirectoryTraversal(t *testing.T) {
	dir := t.TempDir()
	writeTemplateFile(t, filepath.Join(dir, "a.json"), validTemplateDescriptor())
	subDir := filepath.Join(dir, "nested")
	if err := os.MkdirAll(subDir, 0o750); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	writeTemplateFile(t, filepath.Join(subDir, "b.json"), validTemplateDescriptor())
	if err := os.WriteFile(filepath.Join(subDir, "ignore.txt"), []byte("x"), 0o600); err != nil {
		t.Fatalf("write text file: %v", err)
	}

	var stdout, stderr strings.Builder
	code := cli([]string{"lint", "dataset", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr=%q)", code, stderr.String())
	}
	if strings.Count(stdout.String(), ": OK") != 2 {
		t.Fatalf("expected two validated files, got output %q", stdout.String())
	}
}

func TestLintDatasetCLINonJSONFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "template.yaml")
	if err := os.WriteFile(path, []byte("dialect: sql"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	var stdout, stderr strings.Builder
	code := cli([]string{"lint", "dataset", path}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "expected .json file") {
		t.Fatalf("expected extension error, got %q", stderr.String())
	}
}

func TestLintTemplateFileJSONParseError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatalf("write bad file: %v", err)
	}
	err := lintTemplateFile(path)
	if err == nil || !strings.Contains(err.Error(), "parse JSON") {
		t.Fatalf("expected parse error, got %v", err)
	}
}

func TestLintTemplateFileUnknownField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "unknown.json")
	payload := `{
  "plugin": "frog",
  "key": "frog_population_snapshot",
  "version": "0.1.0",
  "title": "Frog Population Snapshot",
  "description": "fixture",
  "dialect": "dsl",
  "query": "REPORT frog_population_snapshot\nSELECT organism_id FROM organisms",
  "parameters": [],
  "columns": [{"name":"organism_id","type":"string"}],
  "output_formats": ["json"],
  "slug": "frog/frog_population_snapshot@0.1.0",
  "unknown_key": true
}`
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatalf("write unknown field file: %v", err)
	}
	err := lintTemplateFile(path)
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown field parse error, got %v", err)
	}
}

func TestCollectTemplateJSONFilesMissingPath(t *testing.T) {
	_, err := collectTemplateJSONFiles([]string{filepath.Join(t.TempDir(), "missing")})
	if err == nil {
		t.Fatalf("expected missing path error")
	}
}

func TestLintDatasetCLIFixtures(t *testing.T) {
	root := repositoryRoot(t)
	validDir := filepath.Join(root, "testutil", "fixtures", "dataset-templates", "valid")
	edgeDir := filepath.Join(root, "testutil", "fixtures", "dataset-templates", "edge")
	invalidDir := filepath.Join(root, "testutil", "fixtures", "dataset-templates", "invalid")
	for _, dir := range []string{validDir, edgeDir, invalidDir} {
		if _, err := os.Stat(dir); err != nil {
			if os.IsNotExist(err) {
				t.Skipf("fixture directories not present: %s", dir)
			}
			t.Fatalf("stat fixture directory %s: %v", dir, err)
		}
	}

	t.Run("valid and edge fixtures pass", func(t *testing.T) {
		var stdout, stderr strings.Builder
		code := cli([]string{"lint", "dataset", validDir, edgeDir}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("expected success for valid fixtures, got %d (stderr=%q)", code, stderr.String())
		}
	})

	t.Run("invalid fixtures fail", func(t *testing.T) {
		var stdout, stderr strings.Builder
		code := cli([]string{"lint", "dataset", invalidDir}, &stdout, &stderr)
		if code != 1 {
			t.Fatalf("expected failure for invalid fixtures, got %d", code)
		}
		if !strings.Contains(stderr.String(), "dataset lint failed") {
			t.Fatalf("expected summary failure message, got %q", stderr.String())
		}
	})
}

func validTemplateDescriptor() datasetapi.TemplateDescriptor {
	dialectProvider := datasetapi.GetDialectProvider()
	formatProvider := datasetapi.GetFormatProvider()
	return datasetapi.TemplateDescriptor{
		Plugin:        "frog",
		Key:           "frog_population_snapshot",
		Version:       "0.1.0",
		Title:         "Frog Population Snapshot",
		Description:   "fixture",
		Dialect:       dialectProvider.DSL(),
		Query:         "REPORT frog_population_snapshot\nSELECT organism_id FROM organisms",
		Parameters:    []datasetapi.Parameter{{Name: "limit", Type: "integer", Default: json.RawMessage("10")}},
		Columns:       []datasetapi.Column{{Name: "organism_id", Type: "string"}},
		OutputFormats: []datasetapi.Format{formatProvider.JSON(), formatProvider.CSV()},
		Slug:          "frog/frog_population_snapshot@0.1.0",
	}
}

func writeTemplateFile(t *testing.T, path string, descriptor datasetapi.TemplateDescriptor) {
	t.Helper()
	payload, err := json.MarshalIndent(descriptor, "", "  ")
	if err != nil {
		t.Fatalf("marshal template: %v", err)
	}
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime caller unavailable")
	}
	dir := filepath.Dir(currentFile)
	for {
		candidate := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(candidate); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("could not find repository root from %s", currentFile)
	return "" // unreachable
}
