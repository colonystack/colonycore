// Package validation provides hexagonal architecture pattern validation for plugin code
package validation

import (
	"bufio"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Error represents a hexagonal architecture violation found in code
type Error struct {
	File    string
	Line    int
	Message string
	Code    string
}

// ValidatePluginDirectory validates all Go files in a plugin directory for hexagonal architecture compliance
func ValidatePluginDirectory(dir string) []Error {
	var errors []Error

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
		// Add a validation error for walk failures instead of just logging
		errors = append(errors, Error{
			File:    dir,
			Line:    0,
			Message: "Failed to walk directory: " + err.Error(),
			Code:    "",
		})
	}

	return errors
}

func validatePluginFile(filePath string) []Error {
	var errors []Error

	// Text-based validation (catches string patterns)
	textErrors := validateFileText(filePath)
	errors = append(errors, textErrors...)

	// AST-based validation (catches structural patterns)
	astErrors := validateFileAST(filePath)
	errors = append(errors, astErrors...)

	return errors
}

func validateFileText(filePath string) []Error {
	var errors []Error

	file, err := os.Open(filepath.Clean(filePath))
	if err != nil {
		return append(errors, Error{
			File:    filePath,
			Line:    0,
			Message: "Failed to open file: " + err.Error(),
			Code:    "",
		})
	}
	defer func() {
		_ = file.Close() // Best effort close, ignore error
	}()

	// Anti-patterns to detect via regex
	antiPatterns := getAntiPatterns()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip comments and empty lines
		if strings.TrimSpace(line) == "" || isCommentLine(line) {
			continue
		}

		for pattern, message := range antiPatterns {
			if matched, _ := regexp.MatchString(pattern, line); matched {
				errors = append(errors, Error{
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

func getAntiPatterns() map[string]string {
	return map[string]string{
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
}

func validateFileAST(filePath string) []Error {
	var errors []Error

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

func validateCallExpr(fset *token.FileSet, call *ast.CallExpr) []Error {
	var errors []Error

	// Check for legacy NewViolation calls
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "pluginapi" {
			if sel.Sel.Name == "NewViolation" {
				pos := fset.Position(call.Pos())
				errors = append(errors, Error{
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

func validateBinaryExpr(fset *token.FileSet, binary *ast.BinaryExpr) []Error {
	var errors []Error

	// Check for entity type comparisons
	if binary.Op.String() == "==" {
		if isEntityConstant(binary.X) || isEntityConstant(binary.Y) {
			pos := fset.Position(binary.Pos())
			errors = append(errors, Error{
				File:    pos.Filename,
				Line:    pos.Line,
				Message: "Use contextual Equals() methods instead of direct entity comparisons",
				Code:    "entity == EntityType",
			})
		}
	}

	return errors
}

func validateIdentifier(fset *token.FileSet, ident *ast.Ident) []Error {
	var errors []Error

	// Check for forbidden constants
	forbiddenConstants := getForbiddenConstants()

	if suggestion, isForbidden := forbiddenConstants[ident.Name]; isForbidden {
		pos := fset.Position(ident.Pos())
		errors = append(errors, Error{
			File:    pos.Filename,
			Line:    pos.Line,
			Message: "Forbidden constant usage. " + suggestion,
			Code:    ident.Name,
		})
	}

	return errors
}

func getForbiddenConstants() map[string]string {
	return map[string]string{
		"EntityOrganism":       "Use entityContext.Organism()",
		"EntityHousingUnit":    "Use entityContext.Housing()",
		"SeverityWarn":         "Use severityContext.Warn()",
		"SeverityLog":          "Use severityContext.Log()",
		"SeverityBlock":        "Use severityContext.Block()",
		"ActionCreate":         "Use actionContext.Create()",
		"ActionUpdate":         "Use actionContext.Update()",
		"ActionDelete":         "Use actionContext.Delete()",
		"EnvironmentAquatic":   "Use housingContext.Aquatic()",
		"LifecycleStageAdult":  "Use lifecycleContext.Adult()",
		"ProtocolStatusActive": "Use protocolContext.Active()",
	}
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
