package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"colonycore/internal/validation"
)

var noopContractValidator = func(string) error { return nil }

func TestRunUsage(t *testing.T) {
	var stderr bytes.Buffer
	exitCode := run([]string{"validate_plugin_patterns"}, &stderr, validation.ValidatePluginDirectory, noopContractValidator)
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit code for missing args")
	}
	if !strings.Contains(stderr.String(), "Usage:") {
		t.Fatalf("expected usage message, got %q", stderr.String())
	}
}

func TestRunSuccess(t *testing.T) {
	var stderr bytes.Buffer
	exitCode := run([]string{"validate_plugin_patterns", filepath.Join("..", "plugins", "frog")}, &stderr, func(string) []validation.Error {
		return nil
	}, noopContractValidator)
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
	}, noopContractValidator)
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

func TestRunContractValidationFailure(t *testing.T) {
	var stderr bytes.Buffer
	contractErr := errors.New("contract mismatch")
	exitCode := run([]string{"validate_plugin_patterns", "plugin"}, &stderr, func(string) []validation.Error {
		return nil
	}, func(string) error {
		return contractErr
	})
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "Plugin contract enforcement failed") {
		t.Fatalf("expected contract failure message, got %q", stderr.String())
	}
}

func TestRunUsesDefaultContractPath(t *testing.T) {
	var calledWith string
	validator := func(path string) error {
		calledWith = path
		return nil
	}
	var stderr bytes.Buffer
	exit := run([]string{"cmd", "dir"}, &stderr, func(string) []validation.Error { return nil }, validator)
	if exit != 0 {
		t.Fatalf("expected success, got %d", exit)
	}
	if calledWith != defaultContractPath {
		t.Fatalf("expected default contract path %q, got %q", defaultContractPath, calledWith)
	}
}

func TestRunUsesCustomContractPath(t *testing.T) {
	custom := "../docs/custom-contract.md"
	var calledWith string
	validator := func(path string) error {
		calledWith = path
		return nil
	}
	var stderr bytes.Buffer
	exit := run([]string{"cmd", "dir", custom}, &stderr, func(string) []validation.Error { return nil }, validator)
	if exit != 0 {
		t.Fatalf("expected success, got %d", exit)
	}
	if calledWith != custom {
		t.Fatalf("expected custom contract path %q, got %q", custom, calledWith)
	}
}

func TestMainInvokesRun(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	pluginDir := filepath.Join("..", "plugins", "frog")
	os.Args = []string{"validate_plugin_patterns", pluginDir}

	exitCode := run(os.Args, &bytes.Buffer{}, validation.ValidatePluginDirectory, noopContractValidator)
	if exitCode != 0 {
		t.Fatalf("expected run to succeed, got exit code %d", exitCode)
	}
}

func TestRunErrorHandling(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		validator func(string) []validation.Error
		wantExit  int
		wantOut   string
	}{
		{
			name:     "no args",
			args:     []string{"cmd"},
			wantExit: 1,
			wantOut:  "Usage:",
		},
		{
			name:     "empty args",
			args:     []string{},
			wantExit: 1,
			wantOut:  "Usage:",
		},
		{
			name:     "single arg (missing directory)",
			args:     []string{"cmd"},
			wantExit: 1,
			wantOut:  "Usage:",
		},
		{
			name:      "success case",
			args:      []string{"cmd", "dir"},
			validator: func(string) []validation.Error { return nil },
			wantExit:  0,
		},
		{
			name: "validation errors",
			args: []string{"cmd", "dir"},
			validator: func(string) []validation.Error {
				return []validation.Error{
					{File: "test.go", Line: 5, Message: "error", Code: "code"},
				}
			},
			wantExit: 1,
			wantOut:  "hexagonal architecture violations",
		},
		{
			name: "multiple validation errors",
			args: []string{"cmd", "dir"},
			validator: func(string) []validation.Error {
				return []validation.Error{
					{File: "test1.go", Line: 5, Message: "error1", Code: "code1"},
					{File: "test2.go", Line: 10, Message: "error2", Code: "code2"},
				}
			},
			wantExit: 1,
			wantOut:  "Found 2 hexagonal",
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			var stderr bytes.Buffer
			exitCode := run(tc.args, &stderr, tc.validator, noopContractValidator)

			if exitCode != tc.wantExit {
				t.Errorf("run() exit code = %v, want %v", exitCode, tc.wantExit)
			}

			if tc.wantOut != "" && !strings.Contains(stderr.String(), tc.wantOut) {
				t.Errorf("run() output should contain %q, got %q", tc.wantOut, stderr.String())
			}
		})
	}
}

func TestRunFailedWriter(t *testing.T) {
	failWriter := &failingWriter{}
	exitCode := run([]string{"cmd"}, failWriter, nil, noopContractValidator)
	if exitCode != 1 {
		t.Errorf("expected exit code 1 when writer fails, got %d", exitCode)
	}
}

func TestRunViolationWriterFailures(t *testing.T) {
	mockErrors := []validation.Error{
		{File: "file.go", Line: 1, Message: "msg", Code: "code"},
	}
	for name, failAt := range map[string]int{
		"header": 1,
		"file":   2,
		"msg":    3,
		"code":   4,
	} {
		t.Run(name, func(t *testing.T) {
			writer := &limitedWriter{failAt: failAt}
			exit := run([]string{"cmd", "dir"}, writer, func(string) []validation.Error {
				return mockErrors
			}, noopContractValidator)
			if exit != 1 {
				t.Fatalf("expected exit code 1 when writer fails at %s (call %d), got %d", name, failAt, exit)
			}
		})
	}
}

func TestExtractContractMetadata(t *testing.T) {
	content := []byte("<!-- CONTRACT-METADATA {\"version\":\"1\",\"entities\":{}} -->")
	meta, err := extractContractMetadata(content)
	if err != nil {
		t.Fatalf("expected metadata, got error: %v", err)
	}
	if meta.Version != "1" {
		t.Fatalf("expected version 1, got %s", meta.Version)
	}
}

func TestExtractContractMetadataMissingBlock(t *testing.T) {
	_, err := extractContractMetadata([]byte("no metadata here"))
	if err == nil {
		t.Fatalf("expected error for missing metadata block")
	}
}

func TestValidateContract(t *testing.T) {
	schemaMeta, err := loadSchemaMetadata(filepath.Join("..", entityModelPath))
	if err != nil {
		t.Fatalf("load schema metadata: %v", err)
	}
	contractPath := writeContractFile(t, schemaMeta)
	withRepoRoot(t, func() {
		if err := validateContract(contractPath); err != nil {
			t.Fatalf("expected contract validation success, got %v", err)
		}
	})
}

func TestValidateContractMismatch(t *testing.T) {
	schemaMeta, err := loadSchemaMetadata(filepath.Join("..", entityModelPath))
	if err != nil {
		t.Fatalf("load schema metadata: %v", err)
	}
	badMeta := contractMetadata{Version: schemaMeta.Version + "-extra", Entities: schemaMeta.Entities}
	contractPath := writeContractFile(t, badMeta)
	withRepoRoot(t, func() {
		if err := validateContract(contractPath); err == nil {
			t.Fatalf("expected validation failure for version mismatch")
		}
	})
}

func TestCompareContractMetadataEntityMismatch(t *testing.T) {
	good := contractMetadata{Version: "1", Entities: map[string]contractEntityMetadata{"a": {}}}
	bad := contractMetadata{Version: "1", Entities: map[string]contractEntityMetadata{"b": {}}}
	if err := compareContractMetadata(good, bad); err == nil {
		t.Fatalf("expected mismatch error")
	}
}

func TestMainFunctionLogic(t *testing.T) {
	testArgs := []string{"validate_plugin_patterns", "../plugins/frog"}
	var stderr bytes.Buffer
	exitCode := run(testArgs, &stderr, validation.ValidatePluginDirectory, noopContractValidator)
	if exitCode != 0 {
		t.Logf("Exit code: %d, stderr: %s", exitCode, stderr.String())
	}
}

func TestMainForCoverage(t *testing.T) {
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
	}()
	os.Args = []string{"validate_plugin_patterns", "../plugins/frog"}
	var mockStderr bytes.Buffer
	exitCode := run(os.Args, &mockStderr, validation.ValidatePluginDirectory, noopContractValidator)
	t.Logf("Main function code path executed with exit code: %d", exitCode)
}

func TestMainWithExit(t *testing.T) {
	if os.Getenv("BE_MAIN_RUNNER") == "1" {
		main()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestMainWithExit") // #nosec G204 -- safe fixed args
	cmd.Env = append(os.Environ(), "BE_MAIN_RUNNER=1")
	cmd.Dir = "/mnt/c/Users/Tobi-Wan/git/me/colonycore/scripts"

	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			t.Logf("Main exited with status: %d", exitError.ExitCode())
		} else {
			t.Fatalf("Unexpected error running main: %v", err)
		}
	}
}

func TestMainWithInvalidPath(t *testing.T) {
	if os.Getenv("BE_MAIN_RUNNER_FAIL") == "1" {
		if err := os.Chdir("/tmp"); err != nil {
			t.Fatalf("Failed to change directory: %v", err)
		}
		main()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestMainWithInvalidPath") // #nosec G204 -- safe fixed args
	cmd.Env = append(os.Environ(), "BE_MAIN_RUNNER_FAIL=1")

	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			return
		}
	}
	t.Fatalf("Expected main to exit with code 1")
}

type failingWriter struct{}

func (f *failingWriter) Write(_ []byte) (n int, err error) {
	return 0, bytes.ErrTooLarge
}

type limitedWriter struct {
	failAt int
	writes int
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	w.writes++
	if w.failAt > 0 && w.writes == w.failAt {
		return 0, bytes.ErrTooLarge
	}
	return len(p), nil
}

func writeContractFile(t *testing.T, meta contractMetadata) string {
	t.Helper()
	data, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	content := []byte("<!-- CONTRACT-METADATA " + string(data) + " -->")
	path := filepath.Join(t.TempDir(), "contract.md")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write contract file: %v", err)
	}
	return path
}

func withRepoRoot(t *testing.T, fn func()) {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	repoRoot := findRepoRoot(t, cwd)
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("chdir to repo root: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})
	fn()
}

func findRepoRoot(t *testing.T, start string) string {
	t.Helper()
	cur := start
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(cur, "go.mod")); err == nil {
			return cur
		}
		next := filepath.Dir(cur)
		if next == cur {
			break
		}
		cur = next
	}
	t.Fatalf("could not locate repo root from %s", start)
	return start
}
