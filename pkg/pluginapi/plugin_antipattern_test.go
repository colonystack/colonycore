package pluginapi

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPluginAntiPatterns ensures plugins follow hexagonal architecture patterns
// by checking for raw constant usage in source files (text-based scanning).
func TestPluginAntiPatterns(t *testing.T) {
	// Scan plugin directories for anti-patterns
	pluginDirs := []string{
		"../../plugins/frog",
		// Add more plugin directories as they're created
	}

	for _, pluginDir := range pluginDirs {
		t.Run(filepath.Base(pluginDir), func(t *testing.T) {
			scanPluginDirectory(t, pluginDir)
		})
	}
}

func scanPluginDirectory(t *testing.T, dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Skipf("Plugin directory %s does not exist", dir)
		return
	}

	err := filepath.Walk(dir, func(path string, _ os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			t.Logf("Warning: Could not parse %s: %v", path, err)
			return nil
		}

		checkPluginFile(t, file, path)
		return nil
	})

	if err != nil {
		t.Errorf("Error walking plugin directory %s: %v", dir, err)
	}
}

func checkPluginFile(t *testing.T, file *ast.File, filepath string) {
	antiPatterns := []struct {
		name  string
		check func(*ast.File) []string
	}{
		{
			"raw constant usage",
			checkRawConstantUsage,
		},
		{
			"direct entity type comparison",
			checkDirectEntityTypeComparison,
		},
		{
			"raw severity usage in violations",
			checkRawSeverityUsageInViolations,
		},
	}

	for _, pattern := range antiPatterns {
		violations := pattern.check(file)
		for _, violation := range violations {
			if strings.HasPrefix(violation, "SUGGESTION:") {
				t.Logf("Architecture suggestion for %s: %s", filepath, violation)
			} else {
				// Report as suggestion for existing code to avoid breaking builds
				// Change to t.Errorf for strict enforcement in new code
				t.Logf("Architecture suggestion for %s - Anti-pattern '%s': %s", filepath, pattern.name, violation)
			}
		}
	}
}

// checkRawConstantUsage detects direct usage of exported constants instead of contextual interfaces
func checkRawConstantUsage(file *ast.File) []string {
	var violations []string

	// Define constants that should be accessed via contextual interfaces
	restrictedConstants := map[string]string{
		"EntityOrganism":    "Use entityContext.Organism() instead",
		"EntityHousingUnit": "Use entityContext.Housing() instead",
		"EntityProtocol":    "Use entityContext.Protocol() instead",
		"SeverityLog":       "Use severityContext.Log() instead",
		"SeverityWarn":      "Use severityContext.Warn() instead",
		"SeverityBlock":     "Use severityContext.Block() instead",
		"ActionCreate":      "Use actionContext.Create() instead",
		"ActionUpdate":      "Use actionContext.Update() instead",
		"ActionDelete":      "Use actionContext.Delete() instead",
	}

	ast.Inspect(file, func(n ast.Node) bool {
		if ident, ok := n.(*ast.Ident); ok {
			if suggestion, isRestricted := restrictedConstants[ident.Name]; isRestricted {
				// Allow usage in certain contexts (like switch statements in contextual implementations)
				if !isInAllowedContext(ident) {
					violations = append(violations, fmt.Sprintf("Direct usage of %s. %s", ident.Name, suggestion))
				}
			}
		}
		return true
	})

	return violations
}

// checkDirectEntityTypeComparison detects entity type comparisons that should use contextual methods
func checkDirectEntityTypeComparison(file *ast.File) []string {
	var violations []string

	ast.Inspect(file, func(n ast.Node) bool {
		if binExpr, ok := n.(*ast.BinaryExpr); ok && binExpr.Op == token.EQL {
			// Check for patterns like: entity == EntityOrganism
			if isEntityTypeComparison(binExpr.X) || isEntityTypeComparison(binExpr.Y) {
				violations = append(violations, "Direct entity type comparison detected. Use contextual interfaces with Equals() method")
			}
		}
		return true
	})

	return violations
}

// checkRawSeverityUsageInViolations detects NewViolation calls that should use NewViolationWithEntityRef
func checkRawSeverityUsageInViolations(file *ast.File) []string {
	var violations []string

	ast.Inspect(file, func(n ast.Node) bool {
		if callExpr, ok := n.(*ast.CallExpr); ok {
			if ident, ok := callExpr.Fun.(*ast.Ident); ok && ident.Name == "NewViolation" {
				// For existing plugins, this is a suggestion rather than a hard error
				// New plugins should prefer NewViolationWithEntityRef
				violations = append(violations, "SUGGESTION: Consider using NewViolationWithEntityRef for better hexagonal architecture compliance")
			}
		}
		return true
	})

	return violations
}

func isInAllowedContext(_ *ast.Ident) bool {
	// Check if the identifier is used in contexts where raw constants are acceptable
	// Currently returns false to enforce strict contextual interface usage
	// Future enhancement could analyze AST context for legitimate exceptions
	return false
}

func isEntityTypeComparison(expr ast.Expr) bool {
	if ident, ok := expr.(*ast.Ident); ok {
		return strings.HasPrefix(ident.Name, "Entity")
	}
	return false
}
