package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidatePluginDirectory(t *testing.T) {
	// Create a temporary test directory
	tempDir := t.TempDir()

	// Create a valid Go file
	validFile := filepath.Join(tempDir, "valid.go")
	validContent := `package test

import "colonycore/pkg/pluginapi"

func validExample() {
	entities := pluginapi.NewEntityContext()
	housing := entities.Housing()
}
`
	if err := os.WriteFile(validFile, []byte(validContent), 0600); err != nil {
		t.Fatalf("Failed to write valid file: %v", err)
	}

	// Create an invalid Go file with anti-patterns
	invalidFile := filepath.Join(tempDir, "invalid.go")
	invalidContent := `package test

import (
	"strings"
	"colonycore/pkg/pluginapi"
)

func invalidExample() {
	var housing pluginapi.HousingUnitView
	env := strings.ToLower(housing.Environment())
	if strings.Contains(env, "aquatic") {
		// bad pattern
	}
	if housing.Environment() == "terrestrial" {
		// another bad pattern
	}
}
`
	if err := os.WriteFile(invalidFile, []byte(invalidContent), 0600); err != nil {
		t.Fatalf("Failed to write invalid file: %v", err)
	}

	// Validate the directory
	errors := ValidatePluginDirectory(tempDir)

	// Should find multiple violations in invalid.go
	if len(errors) == 0 {
		t.Error("Expected validation errors but got none")
	}

	// Check that all errors are from invalid.go
	for _, err := range errors {
		if !strings.Contains(err.File, "invalid.go") {
			t.Errorf("Expected error from invalid.go, got error from %s", err.File)
		}
	}

	// Check specific error types
	foundStringManipulation := false
	foundStringComparison := false
	foundStringLiteral := false

	for _, err := range errors {
		if strings.Contains(err.Message, "string case manipulation") {
			foundStringManipulation = true
		}
		if strings.Contains(err.Message, "direct string comparison") {
			foundStringComparison = true
		}
		if strings.Contains(err.Message, "raw string literals") {
			foundStringLiteral = true
		}
	}

	if !foundStringManipulation {
		t.Error("Expected to find string manipulation error")
	}
	if !foundStringComparison {
		t.Error("Expected to find string comparison error")
	}
	if !foundStringLiteral {
		t.Error("Expected to find string literal error")
	}
}

func TestValidatePluginFileWithConstantUsage(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "constants.go")
	content := `package test

import "colonycore/pkg/pluginapi"

func badConstantUsage() {
	entity := EntityOrganism
	severity := SeverityWarn
	action := ActionCreate
}
`
	if err := os.WriteFile(testFile, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	errors := validatePluginFile(testFile)

	if len(errors) == 0 {
		t.Error("Expected validation errors for constant usage but got none")
	}

	// Should catch forbidden constants
	foundEntityConstant := false
	foundSeverityConstant := false
	foundActionConstant := false

	for _, err := range errors {
		if strings.Contains(err.Code, "EntityOrganism") {
			foundEntityConstant = true
		}
		if strings.Contains(err.Code, "SeverityWarn") {
			foundSeverityConstant = true
		}
		if strings.Contains(err.Code, "ActionCreate") {
			foundActionConstant = true
		}
	}

	if !foundEntityConstant {
		t.Error("Expected to find EntityOrganism constant usage")
	}
	if !foundSeverityConstant {
		t.Error("Expected to find SeverityWarn constant usage")
	}
	if !foundActionConstant {
		t.Error("Expected to find ActionCreate constant usage")
	}
}

func TestValidateFileTextWithComments(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "comments.go")
	content := `package test

// This is a comment with "aquatic" which should be ignored
/* Another comment with EntityOrganism */
func example() {
	// inline comment with "terrestrial"
	var actual = "aquatic" // This should be caught
}
`
	if err := os.WriteFile(testFile, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	errors := validateFileText(testFile)

	// Should only find one error (the actual string literal, not comments)
	if len(errors) != 1 {
		t.Errorf("Expected 1 error but got %d", len(errors))
	}

	if len(errors) > 0 && !strings.Contains(errors[0].Code, `var actual = "aquatic"`) {
		t.Errorf("Expected error on actual code line, got: %s", errors[0].Code)
	}
}

func TestValidateFileASTNewViolationUsage(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "violation.go")
	content := `package test

import "colonycore/pkg/pluginapi"

func badViolationUsage() {
	violation := pluginapi.NewViolation("rule", severity, "message", entity, "id")
}
`
	if err := os.WriteFile(testFile, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	errors := validateFileAST(testFile)

	if len(errors) == 0 {
		t.Error("Expected validation error for NewViolation usage but got none")
	}

	found := false
	for _, err := range errors {
		if strings.Contains(err.Message, "NewViolationBuilder") {
			found = true
		}
	}

	if !found {
		t.Error("Expected error about using NewViolationBuilder")
	}
}

func TestGetAntiPatterns(t *testing.T) {
	patterns := getAntiPatterns()

	expectedPatterns := []string{
		`strings\.Contains\([^,]*\.Environment\(\)`,
		`strings\.ToLower\([^,]*\.Environment\(\)`,
		`\b(EntityOrganism|EntityHousingUnit|EntityProtocol)\b`,
		`\b(SeverityLog|SeverityWarn|SeverityBlock)\b`,
	}

	for _, expected := range expectedPatterns {
		if _, exists := patterns[expected]; !exists {
			t.Errorf("Expected anti-pattern %s not found", expected)
		}
	}

	if len(patterns) == 0 {
		t.Error("Expected anti-patterns but got none")
	}
}

func TestGetForbiddenConstants(t *testing.T) {
	constants := getForbiddenConstants()

	expectedConstants := map[string]string{
		"EntityOrganism":    "Use entityContext.Organism()",
		"EntityHousingUnit": "Use entityContext.Housing()",
		"SeverityWarn":      "Use severityContext.Warn()",
	}

	for constant, expectedMessage := range expectedConstants {
		if message, exists := constants[constant]; !exists {
			t.Errorf("Expected forbidden constant %s not found", constant)
		} else if message != expectedMessage {
			t.Errorf("Expected message %s for constant %s, got %s", expectedMessage, constant, message)
		}
	}
}

func TestIsCommentLine(t *testing.T) {
	tests := []struct {
		line     string
		expected bool
	}{
		{"// single line comment", true},
		{"   // indented comment", true},
		{"/* block comment */", true},
		{"  /* indented block comment", true},
		{"actual code", false},
		{"var x = 5 // inline comment should be false", false},
		{"", false},
		{"   ", false},
	}

	for _, test := range tests {
		result := isCommentLine(test.line)
		if result != test.expected {
			t.Errorf("isCommentLine(%q) = %v, expected %v", test.line, result, test.expected)
		}
	}
}

func TestIsEntityConstant(t *testing.T) {
	// This function requires AST nodes, so we test it indirectly through validateIdentifier
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "entity.go")
	content := `package test

func example() {
	var entity = EntityOrganism
	var severity = SeverityWarn
	var action = ActionCreate
	var other = SomeOtherThing
}
`
	if err := os.WriteFile(testFile, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	errors := validateFileAST(testFile)

	// Should find Entity, Severity, and Action constants but not SomeOtherThing
	foundEntity := false
	foundSeverity := false
	foundAction := false
	foundOther := false

	for _, err := range errors {
		if strings.Contains(err.Code, "EntityOrganism") {
			foundEntity = true
		}
		if strings.Contains(err.Code, "SeverityWarn") {
			foundSeverity = true
		}
		if strings.Contains(err.Code, "ActionCreate") {
			foundAction = true
		}
		if strings.Contains(err.Code, "SomeOtherThing") {
			foundOther = true
		}
	}

	if !foundEntity {
		t.Error("Expected to find EntityOrganism")
	}
	if !foundSeverity {
		t.Error("Expected to find SeverityWarn")
	}
	if !foundAction {
		t.Error("Expected to find ActionCreate")
	}
	if foundOther {
		t.Error("Should not find SomeOtherThing as forbidden constant")
	}
}

func TestValidatePluginDirectoryNonExistent(t *testing.T) {
	errors := ValidatePluginDirectory("/nonexistent/directory")

	if len(errors) == 0 {
		t.Error("Expected validation error for non-existent directory")
	}

	if len(errors) > 0 && !strings.Contains(errors[0].Message, "Failed to walk directory") {
		t.Errorf("Expected walk directory error, got: %s", errors[0].Message)
	}
}

func TestValidateFileTextUnreadableFile(t *testing.T) {
	errors := validateFileText("/nonexistent/file.go")

	if len(errors) == 0 {
		t.Error("Expected validation error for unreadable file")
	}

	if len(errors) > 0 && !strings.Contains(errors[0].Message, "Failed to open file") {
		t.Errorf("Expected file open error, got: %s", errors[0].Message)
	}
}
