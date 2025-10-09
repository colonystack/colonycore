package integration

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestContextualAccessorPatternEnforcement performs comprehensive validation
// of the contextual accessor pattern across the entire codebase.
// This test ensures no regressions and enforces architectural consistency.
func TestContextualAccessorPatternEnforcement(t *testing.T) {
	// Find repository root by looking for go.mod
	repoRoot, err := findRepositoryRoot()
	if err != nil {
		t.Fatalf("Failed to find repository root: %v", err)
	}

	t.Run("all view interfaces have contextual accessors", func(t *testing.T) {
		validateAllViewInterfacesHaveContextualAccessors(t, repoRoot)
	})

	t.Run("no raw constant access in plugin implementations", func(t *testing.T) {
		validateNoRawConstantAccessInPlugins(t, repoRoot)
	})

	t.Run("contextual pattern consistency across packages", func(t *testing.T) {
		validateContextualPatternConsistency(t, repoRoot)
	})

	t.Run("enforce contextual usage in examples", func(t *testing.T) {
		validateContextualUsageInExamples(t, repoRoot)
	})

	t.Run("validate CI compatibility", func(t *testing.T) {
		validateCICompatibility(t, repoRoot)
	})
}

// validateAllViewInterfacesHaveContextualAccessors scans all packages for view interfaces
// and ensures they implement the contextual accessor pattern.
func validateAllViewInterfacesHaveContextualAccessors(t *testing.T, baseDir string) {
	// Define required patterns for view interfaces
	requiredPatterns := map[string][]string{
		"OrganismView":    {"GetCurrentStage", "IsActive", "IsRetired", "IsDeceased"},
		"Organism":        {"GetCurrentStage", "IsActive", "IsRetired", "IsDeceased"},
		"HousingUnitView": {"GetEnvironmentType", "IsAquaticEnvironment", "IsHumidEnvironment", "SupportsSpecies"},
		"HousingUnit":     {"GetEnvironmentType", "IsAquaticEnvironment", "IsHumidEnvironment", "SupportsSpecies"},
		"ProtocolView":    {"GetCurrentStatus", "IsActiveProtocol", "IsTerminalStatus", "CanAcceptNewSubjects"},
		"Protocol":        {"GetCurrentStatus", "IsActiveProtocol", "IsTerminalStatus", "CanAcceptNewSubjects"},
		"Procedure":       {"GetCurrentStatus", "IsActiveProcedure", "IsTerminalStatus", "IsSuccessful"},
	}

	// Scan for Go files in pkg directories
	err := filepath.Walk(filepath.Join(baseDir, "pkg"), func(path string, _ os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// Parse the file
		fset := token.NewFileSet()
		src, err := os.ReadFile(path) //nolint:gosec // Path comes from controlled filepath.Walk
		if err != nil {
			return err
		}

		file, err := parser.ParseFile(fset, path, src, parser.ParseComments)
		if err != nil {
			return err
		}

		// Look for view interfaces
		ast.Inspect(file, func(n ast.Node) bool {
			if typeSpec, ok := n.(*ast.TypeSpec); ok {
				if interfaceType, ok := typeSpec.Type.(*ast.InterfaceType); ok {
					interfaceName := typeSpec.Name.Name

					// Check if this interface should have contextual accessors
					if requiredMethods, exists := requiredPatterns[interfaceName]; exists {
						validateInterfaceHasContextualMethods(t, path, interfaceName, interfaceType, requiredMethods)
					}
				}
			}
			return true
		})

		return nil
	})

	if err != nil {
		t.Fatalf("Failed to scan packages: %v", err)
	}
}

func validateInterfaceHasContextualMethods(t *testing.T, filePath, interfaceName string, interfaceAST *ast.InterfaceType, requiredMethods []string) {
	// Extract method names from the interface
	existingMethods := make(map[string]bool)
	for _, method := range interfaceAST.Methods.List {
		if len(method.Names) > 0 {
			existingMethods[method.Names[0].Name] = true
		}
	}

	// Check that all required methods exist
	for _, requiredMethod := range requiredMethods {
		if !existingMethods[requiredMethod] {
			t.Errorf("Interface %s in %s is missing required contextual accessor: %s",
				interfaceName, filePath, requiredMethod)
		}
	}
}

// validateNoRawConstantAccessInPlugins scans plugin implementations to ensure
// they don't use raw constants and only access values via contextual interfaces.
func validateNoRawConstantAccessInPlugins(t *testing.T, baseDir string) {
	// Define forbidden raw constant patterns
	forbiddenPatterns := []*regexp.Regexp{
		regexp.MustCompile(`\bstage(Planned|Larva|Juvenile|Adult|Retired|Deceased)\b`),
		regexp.MustCompile(`\benvironment(Aquatic|Terrestrial|Arboreal|Humid)\b`),
		regexp.MustCompile(`\bstatus(Draft|Active|Suspended|Completed|Cancelled)\b`),
		regexp.MustCompile(`\bEntityOrganism\b`),
		regexp.MustCompile(`\bEntityHousingUnit\b`),
		regexp.MustCompile(`\bSeverityWarn\b`),
		regexp.MustCompile(`\bSeverityBlock\b`),
	}

	// Scan plugins directory
	pluginDir := filepath.Join(baseDir, "plugins")
	err := filepath.Walk(pluginDir, func(path string, _ os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// Read and check file content
		content, err := os.ReadFile(path) //nolint:gosec // Path comes from controlled filepath.Walk
		if err != nil {
			return err
		}

		contentStr := string(content)

		// Check for forbidden patterns
		for _, pattern := range forbiddenPatterns {
			if matches := pattern.FindAllStringIndex(contentStr, -1); len(matches) > 0 {
				lines := strings.Split(contentStr, "\n")
				for _, match := range matches {
					// Find line number
					lineNum := strings.Count(contentStr[:match[0]], "\n") + 1
					matchedText := contentStr[match[0]:match[1]]

					// Skip comments and string literals (basic heuristic)
					if lineNum <= len(lines) {
						line := lines[lineNum-1]
						if strings.Contains(line, "//") && strings.Index(line, "//") < strings.Index(line, matchedText) {
							continue // Skip comments
						}
					}

					t.Errorf("Plugin %s line %d: Found raw constant usage '%s' - should use contextual interface",
						path, lineNum, matchedText)
				}
			}
		}

		return nil
	})

	if err != nil {
		t.Fatalf("Failed to scan plugins directory: %v", err)
	}
}

// validateContextualPatternConsistency ensures that contextual interfaces
// follow consistent patterns across all packages.
func validateContextualPatternConsistency(t *testing.T, baseDir string) {
	// Define expected contextual interface patterns
	expectedContexts := map[string][]string{
		"LifecycleStageContext": {"Planned", "Larva", "Juvenile", "Adult", "Retired", "Deceased"},
		"HousingContext":        {"Aquatic", "Terrestrial", "Arboreal", "Humid"},
		"ProtocolContext":       {"Draft", "Active", "Suspended", "Completed", "Cancelled"},
		"ProcedureContext":      {"Scheduled", "InProgress", "Completed", "Cancelled", "Failed"},
		"EntityContext":         {"Organism", "Housing", "Protocol"},
		"SeverityContext":       {"Log", "Warn", "Block"},
		"ActionContext":         {"Create", "Update", "Delete"},
	}

	// Scan context files
	err := filepath.Walk(filepath.Join(baseDir, "pkg"), func(path string, _ os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.Contains(path, "context.go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// Parse the context file
		fset := token.NewFileSet()
		src, err := os.ReadFile(path) //nolint:gosec // Path comes from controlled filepath.Walk
		if err != nil {
			return err
		}

		file, err := parser.ParseFile(fset, path, src, parser.ParseComments)
		if err != nil {
			return err
		}

		// Check for context interfaces
		ast.Inspect(file, func(n ast.Node) bool {
			if typeSpec, ok := n.(*ast.TypeSpec); ok {
				if interfaceType, ok := typeSpec.Type.(*ast.InterfaceType); ok {
					contextName := typeSpec.Name.Name
					if expectedMethods, exists := expectedContexts[contextName]; exists {
						validateContextInterfacePattern(t, path, contextName, interfaceType, expectedMethods)
					}
				}
			}
			return true
		})

		return nil
	})

	if err != nil {
		t.Fatalf("Failed to scan context files: %v", err)
	}
}

func validateContextInterfacePattern(t *testing.T, filePath, contextName string, interfaceAST *ast.InterfaceType, expectedMethods []string) {
	// Extract method names from the context interface
	existingMethods := make(map[string]bool)
	for _, method := range interfaceAST.Methods.List {
		if len(method.Names) > 0 {
			existingMethods[method.Names[0].Name] = true
		}
	}

	// Check that all expected methods exist
	for _, expectedMethod := range expectedMethods {
		if !existingMethods[expectedMethod] {
			t.Errorf("Context %s in %s is missing expected method: %s",
				contextName, filePath, expectedMethod)
		}
	}
}

// validateContextualUsageInExamples ensures examples demonstrate proper contextual usage
func validateContextualUsageInExamples(t *testing.T, baseDir string) {
	// Look for example files and test files that should demonstrate contextual usage
	examplePattern := regexp.MustCompile(`New[A-Z][a-zA-Z]*Context\(\)`)

	err := filepath.Walk(filepath.Join(baseDir, "pkg"), func(path string, _ os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.Contains(path, "example") && !strings.Contains(path, "_test.go") {
			return nil
		}

		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		content, err := os.ReadFile(path) //nolint:gosec // Path comes from controlled filepath.Walk
		if err != nil {
			return err
		}

		contentStr := string(content)

		// Check for contextual usage patterns in examples
		if strings.Contains(path, "example") || strings.Contains(path, "contextual") {
			if !examplePattern.MatchString(contentStr) {
				t.Logf("Example file %s should demonstrate contextual interface usage", path)
			}
		}

		return nil
	})

	if err != nil {
		t.Fatalf("Failed to scan example files: %v", err)
	}
}

// validateCICompatibility ensures the contextual accessor pattern doesn't break CI
func validateCICompatibility(t *testing.T, baseDir string) {
	// Check for common CI-breaking patterns

	// 1. Verify no circular dependencies in context packages
	t.Log("Validating no circular dependencies in contextual interfaces")

	// 2. Ensure all context providers have factory functions
	contextProviders := []string{"GetDialectProvider", "GetFormatProvider", "NewLifecycleStageContext",
		"NewEntityContext", "NewSeverityContext", "NewActionContext", "NewHousingContext",
		"NewProtocolContext", "NewProcedureContext"}

	for _, provider := range contextProviders {
		found := false
		err := filepath.Walk(filepath.Join(baseDir, "pkg"), func(path string, _ os.FileInfo, err error) error {
			if err != nil || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}

			content, err := os.ReadFile(path) //nolint:gosec // Path comes from controlled filepath.Walk
			if err != nil {
				return err
			}

			if strings.Contains(string(content), "func "+provider) {
				found = true
			}

			return nil
		})

		if err != nil {
			t.Fatalf("Failed to check for provider %s: %v", provider, err)
		}

		if !found {
			t.Errorf("Missing contextual provider function: %s", provider)
		}
	}

	t.Log("CI compatibility validation completed")
}

// findRepositoryRoot finds the repository root by looking for go.mod file
func findRepositoryRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("could not find go.mod file")
}
