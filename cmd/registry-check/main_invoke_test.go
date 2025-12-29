package main

import (
	"os"
	"testing"
)

// TestMainFunctionCoversSuccessAndFailure invokes main with patched exitFunc.
func TestMainFunctionCoversSuccessAndFailure(t *testing.T) {
	// success registry file
	reg := "test_registry_main.yaml"
	docPath := "test_registry_main_doc.md"
	writeTestFile(t, docPath, "# Test\n- Status: Draft\n")
	content := "documents:\n  - id: RFC-10\n    type: RFC\n    title: Main\n    status: Draft\n    path: " + docPath + "\n"
	if err := os.WriteFile(reg, []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	defer func() { _ = os.Remove(reg) }()
	var codes []int
	old := exitFunc
	exitFunc = func(code int) { codes = append(codes, code) }
	defer func() { exitFunc = old }()
	os.Args = []string{"registry-check", "-registry", reg}
	main()
	// failure (nonexistent file)
	os.Args = []string{"registry-check", "-registry", "does-not-exist.yaml"}
	main()
	if len(codes) != 2 {
		t.Fatalf("expected two exit codes, got %v", codes)
	}
	if codes[0] != 0 || codes[1] == 0 {
		t.Fatalf("unexpected exit codes: %v", codes)
	}
}
