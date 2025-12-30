package main

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"colonycore/internal/validation"
)

func TestRunUsesDefaults(t *testing.T) {
	var gotAllowlist string
	var gotRoots []string
	var gotBase string
	exit := run([]string{"cmd"}, &bytes.Buffer{}, func(allowlistPath, baseDir string, roots []string) ([]validation.Error, error) {
		gotAllowlist = allowlistPath
		gotRoots = roots
		gotBase = baseDir
		return nil, nil
	})
	if exit != 0 {
		t.Fatalf("expected exit 0, got %d", exit)
	}
	if gotAllowlist != defaultAllowlistPath {
		t.Fatalf("expected allowlist %q, got %q", defaultAllowlistPath, gotAllowlist)
	}
	if strings.Join(gotRoots, ",") != defaultRoots {
		t.Fatalf("expected roots %q, got %q", defaultRoots, strings.Join(gotRoots, ","))
	}
	if gotBase == "" {
		t.Fatalf("expected base dir to be set")
	}
}

func TestMainUsesExitCode(t *testing.T) {
	originalExit := exitFunc
	originalValidate := validateFunc
	originalGetwd := getwd
	originalArgs := os.Args
	t.Cleanup(func() {
		exitFunc = originalExit
		validateFunc = originalValidate
		getwd = originalGetwd
		os.Args = originalArgs
	})
	var got int
	exitFunc = func(code int) { got = code }
	validateFunc = func(string, string, []string) ([]validation.Error, error) {
		return nil, nil
	}
	getwd = func() (string, error) { return t.TempDir(), nil }
	os.Args = []string{"cmd"}
	main()
	if got != 0 {
		t.Fatalf("expected exit code 0, got %d", got)
	}
}

func TestRunWithNoArgs(t *testing.T) {
	exit := run([]string{}, &bytes.Buffer{}, func(string, string, []string) ([]validation.Error, error) {
		return nil, nil
	})
	if exit != 1 {
		t.Fatalf("expected exit 1, got %d", exit)
	}
}

func TestRunFlagParseError(t *testing.T) {
	var stderr bytes.Buffer
	exit := run([]string{"cmd", "-unknown"}, &stderr, func(string, string, []string) ([]validation.Error, error) {
		return nil, nil
	})
	if exit != 1 {
		t.Fatalf("expected exit 1, got %d", exit)
	}
	if stderr.Len() == 0 {
		t.Fatalf("expected flag parse output")
	}
}

func TestRunGetwdFailure(t *testing.T) {
	originalGetwd := getwd
	getwd = func() (string, error) { return "", errors.New("nope") }
	t.Cleanup(func() { getwd = originalGetwd })
	var stderr bytes.Buffer
	exit := run([]string{"cmd"}, &stderr, func(string, string, []string) ([]validation.Error, error) {
		return nil, nil
	})
	if exit != 1 {
		t.Fatalf("expected exit 1, got %d", exit)
	}
	if !strings.Contains(stderr.String(), "resolve working directory") {
		t.Fatalf("expected getwd error, got %q", stderr.String())
	}
}

func TestRunReportsValidationError(t *testing.T) {
	var stderr bytes.Buffer
	exit := run([]string{"cmd"}, &stderr, func(string, string, []string) ([]validation.Error, error) {
		return nil, errors.New("boom")
	})
	if exit != 1 {
		t.Fatalf("expected exit 1, got %d", exit)
	}
	if !strings.Contains(stderr.String(), "any usage guard failed") {
		t.Fatalf("expected error message, got %q", stderr.String())
	}
}

func TestRunReportsViolations(t *testing.T) {
	var stderr bytes.Buffer
	exit := run([]string{"cmd"}, &stderr, func(string, string, []string) ([]validation.Error, error) {
		return []validation.Error{
			{File: "pkg/pluginapi/views.go", Line: 10, Message: "disallowed", Code: "type Foo map[string]any"},
		}, nil
	})
	if exit != 1 {
		t.Fatalf("expected exit 1, got %d", exit)
	}
	if !strings.Contains(stderr.String(), "Found 1 disallowed any usages") {
		t.Fatalf("expected violation header, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "pkg/pluginapi/views.go:10") {
		t.Fatalf("expected violation location, got %q", stderr.String())
	}
}

func TestRunRequiresRoots(t *testing.T) {
	var stderr bytes.Buffer
	exit := run([]string{"cmd", "-roots="}, &stderr, func(string, string, []string) ([]validation.Error, error) {
		return nil, nil
	})
	if exit != 1 {
		t.Fatalf("expected exit 1, got %d", exit)
	}
	if !strings.Contains(stderr.String(), "no roots provided") {
		t.Fatalf("expected roots error, got %q", stderr.String())
	}
}

func TestSplitRoots(t *testing.T) {
	roots := splitRoots(" pkg/pluginapi , pkg/datasetapi ,,")
	if len(roots) != 2 {
		t.Fatalf("expected 2 roots, got %d", len(roots))
	}
	if roots[0] != "pkg/pluginapi" || roots[1] != "pkg/datasetapi" {
		t.Fatalf("unexpected roots: %v", roots)
	}
}

func TestSplitRootsEmpty(t *testing.T) {
	if roots := splitRoots("   "); roots != nil {
		t.Fatalf("expected nil roots, got %v", roots)
	}
}
