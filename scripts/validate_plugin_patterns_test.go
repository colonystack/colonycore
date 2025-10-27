package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"colonycore/internal/validation"
)

func TestRunUsage(t *testing.T) {
	var stderr bytes.Buffer
	exitCode := run([]string{"validate_plugin_patterns"}, &stderr, validation.ValidatePluginDirectory)
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit code for missing args")
	}
	out := stderr.String()
	if !strings.Contains(out, "Usage:") {
		t.Fatalf("expected usage message, got %q", out)
	}
}

func TestRunSuccess(t *testing.T) {
	var stderr bytes.Buffer
	exitCode := run([]string{"validate_plugin_patterns", filepath.Join("..", "plugins", "frog")}, &stderr, func(string) []validation.Error {
		return nil
	})
	if exitCode != 0 {
		t.Fatalf("expected success exit code, got %d", exitCode)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
}

func TestRunWithViolations(t *testing.T) {
	var stderr bytes.Buffer
	mockErrors := []validation.Error{
		{File: "plugin/file.go", Line: 12, Message: "bad practice", Code: "strings.Contains(housing.Environment(), \"aquatic\")"},
	}
	exitCode := run([]string{"validate_plugin_patterns", "plugin"}, &stderr, func(string) []validation.Error {
		return mockErrors
	})
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit code when violations reported")
	}
	output := stderr.String()
	if !strings.Contains(output, "hexagonal architecture violations") {
		t.Fatalf("expected violation header, got %q", output)
	}
	if !strings.Contains(output, mockErrors[0].File) || !strings.Contains(output, mockErrors[0].Message) {
		t.Fatalf("expected error details in output, got %q", output)
	}
}

func TestMainInvokesRun(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	pluginDir := filepath.Join("..", "plugins", "frog")
	os.Args = []string{"validate_plugin_patterns", pluginDir}

	exitCode := run(os.Args, &bytes.Buffer{}, validation.ValidatePluginDirectory)
	if exitCode != 0 {
		t.Fatalf("expected run to succeed, got exit code %d", exitCode)
	}
}
