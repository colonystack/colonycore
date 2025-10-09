// validate_plugin_patterns.go provides compile-time validation of plugin code
// to ensure adherence to hexagonal architecture and contextual accessor patterns.
package main

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type ValidationError struct {
	File    string
	Line    int
	Message string
	Code    string
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <plugin-directory>\n", os.Args[0])
		os.Exit(1)
	}

	pluginDir := os.Args[1]
	errors := validatePluginDirectory(pluginDir)

	if len(errors) > 0 {
		fmt.Fprintf(os.Stderr, "❌ Found %d hexagonal architecture violations:\n\n", len(errors))
		for _, err := range errors {
			fmt.Fprintf(os.Stderr, "🚨 %s:%d\n", err.File, err.Line)
			fmt.Fprintf(os.Stderr, "   %s\n", err.Message)
			fmt.Fprintf(os.Stderr, "   Code: %s\n\n", err.Code)
		}
		os.Exit(1)
	}
}

func validatePluginDirectory(dir string) []ValidationError {
	var errors []ValidationError

	err := filepath.Walk(dir, func(path string, _ os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fileErrors := validatePluginFile(path)
		errors = append(errors, fileErrors...)
		return nil
	})

	if err != nil {
		fmt.Printf("Warning: failed to walk directory %s: %v\n", dir, err)
	}

	return errors
}

func validatePluginFile(filePath string) []ValidationError {
	var errors []ValidationError

	// Text-based validation (catches string patterns)
	textErrors := validateFileText(filePath)
	errors = append(errors, textErrors...)

	// AST-based validation (catches structural patterns)
	astErrors := validateFileAST(filePath)
	errors = append(errors, astErrors...)

	return errors
}

func validateFileText(filePath string) []ValidationError {
	var errors []ValidationError

	file, err := os.Open(filepath.Clean(filePath))
	if err != nil {
		fmt.Printf("Warning: failed to open file %s: %v\n", filePath, err)
		return errors
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close file %s: %v\n", filePath, closeErr)
		}
	}()

	// Anti-patterns to detect via regex
	antiPatterns := map[string]string{
		`strings\.Contains\([^,]*\.Environment\(\)`:                 "Use contextual accessors like housing.IsAquaticEnvironment() instead of string manipulation",
		`strings\.ToLower\([^,]*\.Environment\(\)`:                  "Use contextual accessors instead of string case manipulation",
		`[^,]*\.Environment\(\)\s*==\s*"`:                           "Use housing.IsAquaticEnvironment() instead of direct string comparison",
		`[^,]*\.Stage\(\)\s*==\s*"`:                                 "Use organism.GetCurrentStage() contextual methods instead",
		`"(aquatic|humid|terrestrial|arboreal)"`:                    "Use housingContext.Aquatic() instead of raw string literals",
		`"(planned|larva|juvenile|adult|retired|deceased)"`:         "Use lifecycleContext.Adult() instead of raw string literals",
		`"(draft|active|suspended|completed|cancelled)"`:            "Use protocolContext.Active() instead of raw string literals",
		`\b(EntityOrganism|EntityHousingUnit|EntityProtocol)\b`:     "Use entityContext.Organism() instead of raw constants",
		`\b(SeverityLog|SeverityWarn|SeverityBlock)\b`:              "Use severityContext.Warn() instead of raw constants",
		`\b(ActionCreate|ActionUpdate|ActionDelete)\b`:              "Use actionContext.Create() instead of raw constants",
		`pluginapi\.(Environment|Protocol|Lifecycle)[A-Z][a-zA-Z]*`: "Use context providers instead of direct constant access",
	}

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip comments and string literals in certain contexts
		if strings.TrimSpace(line) == "" || isCommentLine(line) {
			continue
		}

		for pattern, message := range antiPatterns {
			if matched, _ := regexp.MatchString(pattern, line); matched {
				errors = append(errors, ValidationError{
					File:    filePath,
					Line:    lineNum,
					Message: message,
					Code:    strings.TrimSpace(line),
				})
			}
		}
	}

	return errors
}

func validateFileAST(filePath string) []ValidationError {
	var errors []ValidationError

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		// If we can't parse it, skip AST validation
		return errors
	}

	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			errors = append(errors, validateCallExpr(fset, node)...)
		case *ast.BinaryExpr:
			errors = append(errors, validateBinaryExpr(fset, node)...)
		case *ast.Ident:
			errors = append(errors, validateIdentifier(fset, node)...)
		}
		return true
	})

	return errors
}

func validateCallExpr(fset *token.FileSet, call *ast.CallExpr) []ValidationError {
	var errors []ValidationError

	// Check for legacy NewViolation calls
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "pluginapi" {
			if sel.Sel.Name == "NewViolation" {
				pos := fset.Position(call.Pos())
				errors = append(errors, ValidationError{
					File:    pos.Filename,
					Line:    pos.Line,
					Message: "Use NewViolationBuilder() for better hexagonal architecture compliance",
					Code:    "pluginapi.NewViolation(...)",
				})
			}
		}
	}

	return errors
}

func validateBinaryExpr(fset *token.FileSet, binary *ast.BinaryExpr) []ValidationError {
	var errors []ValidationError

	// Check for entity type comparisons
	if binary.Op.String() == "==" {
		if isEntityConstant(binary.X) || isEntityConstant(binary.Y) {
			pos := fset.Position(binary.Pos())
			errors = append(errors, ValidationError{
				File:    pos.Filename,
				Line:    pos.Line,
				Message: "Use contextual Equals() methods instead of direct entity comparisons",
				Code:    "entity == EntityType",
			})
		}
	}

	return errors
}

func validateIdentifier(fset *token.FileSet, ident *ast.Ident) []ValidationError {
	var errors []ValidationError

	// Check for forbidden constants
	forbiddenConstants := map[string]string{
		"EntityOrganism":       "Use entityContext.Organism()",
		"EntityHousingUnit":    "Use entityContext.Housing()",
		"SeverityWarn":         "Use severityContext.Warn()",
		"EnvironmentAquatic":   "Use housingContext.Aquatic()",
		"LifecycleStageAdult":  "Use lifecycleContext.Adult()",
		"ProtocolStatusActive": "Use protocolContext.Active()",
	}

	if suggestion, isForbidden := forbiddenConstants[ident.Name]; isForbidden {
		pos := fset.Position(ident.Pos())
		errors = append(errors, ValidationError{
			File:    pos.Filename,
			Line:    pos.Line,
			Message: fmt.Sprintf("Forbidden constant usage. %s", suggestion),
			Code:    ident.Name,
		})
	}

	return errors
}

func isCommentLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*")
}

func isEntityConstant(expr ast.Expr) bool {
	if ident, ok := expr.(*ast.Ident); ok {
		return strings.HasPrefix(ident.Name, "Entity") ||
			strings.HasPrefix(ident.Name, "Severity") ||
			strings.HasPrefix(ident.Name, "Action")
	}
	return false
}
