package pluginapi

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// TestBreakingChangeEnforcement ensures that the hexagonal architecture breaking change
// is properly enforced by verifying old constants are not exported and plugins must
// use contextual interfaces.
func TestBreakingChangeEnforcement(t *testing.T) {
	t.Run("old constants are not exported", func(t *testing.T) {
		// These constants should no longer be accessible to external packages
		prohibitedConstants := []string{
			"SeverityWarn", "SeverityLog", "SeverityBlock",
			"EntityOrganism", "EntityProtocol", "EntityHousingUnit",
			"ActionCreate", "ActionUpdate", "ActionDelete",
		}

		// Parse the domain_aliases.go file to check exports
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "domain_aliases.go", nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("failed to parse domain_aliases.go: %v", err)
		}

		// Check that prohibited constants are not exported
		for _, decl := range file.Decls {
			if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.CONST {
				for _, spec := range genDecl.Specs {
					if valueSpec, ok := spec.(*ast.ValueSpec); ok {
						for _, name := range valueSpec.Names {
							for _, prohibited := range prohibitedConstants {
								if name.Name == prohibited {
									t.Errorf("Prohibited constant %s is still exported - breaking change not complete", prohibited)
								}
							}
						}
					}
				}
			}
		}
	})

	t.Run("contextual interfaces are required", func(t *testing.T) {
		// Verify that the new contextual interfaces exist and are callable
		// We test by actually calling them rather than parsing source

		// These should not panic and should return valid interfaces
		entityCtx := NewEntityContext()
		if entityCtx == nil {
			t.Error("NewEntityContext should return valid interface")
		}

		severityCtx := NewSeverityContext()
		if severityCtx == nil {
			t.Error("NewSeverityContext should return valid interface")
		}

		actionCtx := NewActionContext()
		if actionCtx == nil {
			t.Error("NewActionContext should return valid interface")
		}

		// Builders should be available
		vb := NewViolationBuilder()
		if vb == nil {
			t.Error("NewViolationBuilder should return valid builder")
		}

		cb := NewChangeBuilder()
		if cb == nil {
			t.Error("NewChangeBuilder should return valid builder")
		}

		rb := NewResultBuilder()
		if rb == nil {
			t.Error("NewResultBuilder should return valid builder")
		}
	})

	t.Run("builder validation works", func(t *testing.T) {
		// Test that builders properly validate and prevent invalid usage

		// ViolationBuilder validation
		_, err := NewViolationBuilder().Build()
		if err == nil || !strings.Contains(err.Error(), "rule is required") {
			t.Error("ViolationBuilder should validate required fields")
		}

		_, err = NewViolationBuilder().WithRule("test").Build()
		if err == nil || !strings.Contains(err.Error(), "severity is required") {
			t.Error("ViolationBuilder should validate severity requirement")
		}

		// ChangeBuilder validation
		_, err = NewChangeBuilder().Build()
		if err == nil || !strings.Contains(err.Error(), "entity is required") {
			t.Error("ChangeBuilder should validate required fields")
		}
	})

	t.Run("contextual interfaces provide behavioral methods", func(t *testing.T) {
		entities := NewEntityContext()
		severities := NewSeverityContext()
		actions := NewActionContext()

		// Test that contextual interfaces return opaque references with behavioral methods
		organism := entities.Organism()
		if organism == nil {
			t.Fatal("EntityContext.Organism() should return valid reference")
		}
		if !organism.IsCore() {
			t.Error("Organism should be identified as core entity via behavioral method")
		}

		warn := severities.Warn()
		if warn == nil {
			t.Fatal("SeverityContext.Warn() should return valid reference")
		}
		if warn.IsBlocking() {
			t.Error("Warn severity should not be blocking via behavioral method")
		}

		create := actions.Create()
		if create == nil {
			t.Fatal("ActionContext.Create() should return valid reference")
		}
		if !create.IsMutation() {
			t.Error("Create action should be identified as mutation via behavioral method")
		}
	})

	t.Run("anti-pattern detection catches violations", func(t *testing.T) {
		// Create a mock Go file that uses prohibited patterns
		mockGoCode := `
package test
import "colonycore/pkg/pluginapi"
func badExample() {
	_ = pluginapi.SeverityWarn
	_ = pluginapi.EntityOrganism
}`

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "mock.go", mockGoCode, parser.ParseComments)
		if err != nil {
			t.Fatalf("failed to parse mock code: %v", err)
		}

		// Use the anti-pattern detection from plugin_antipattern_test.go
		violations := checkRawConstantUsage(file)
		if len(violations) == 0 {
			t.Error("Anti-pattern detection should catch usage of prohibited constants")
		}

		// Verify specific violations are detected
		found := false
		for _, violation := range violations {
			if strings.Contains(violation, "SeverityWarn") && strings.Contains(violation, "severityContext.Warn()") {
				found = true
				break
			}
		}
		if !found {
			t.Error("Anti-pattern detection should specifically catch SeverityWarn usage")
		}
	})
}

// TestAPIStabilityContract ensures the API snapshot mechanism properly guards
// against accidental re-introduction of prohibited constants.
func TestAPIStabilityContract(t *testing.T) {
	t.Run("api snapshot excludes prohibited constants", func(t *testing.T) {
		currentAPI, err := currentAPISnapshot(t)
		if err != nil {
			t.Fatalf("failed to generate current API snapshot: %v", err)
		}

		apiContent := string(currentAPI)

		// Verify prohibited constants are not in the API surface
		prohibitedInAPI := []string{
			"CONST SeverityWarn", "CONST SeverityLog", "CONST SeverityBlock",
			"CONST EntityOrganism", "CONST EntityProtocol", "CONST EntityHousingUnit",
			"CONST ActionCreate", "CONST ActionUpdate", "CONST ActionDelete",
		}

		for _, prohibited := range prohibitedInAPI {
			if strings.Contains(apiContent, prohibited) {
				t.Errorf("API snapshot contains prohibited constant: %s", prohibited)
			}
		}
	})

	t.Run("api snapshot includes required interfaces", func(t *testing.T) {
		currentAPI, err := currentAPISnapshot(t)
		if err != nil {
			t.Fatalf("failed to generate current API snapshot: %v", err)
		}

		apiContent := string(currentAPI)

		// Verify required functions are in the API surface
		required := []string{
			"FUNC NewEntityContext()",
			"FUNC NewSeverityContext()",
			"FUNC NewActionContext()",
			"FUNC NewViolationBuilder()",
			"FUNC NewChangeBuilder()",
			"FUNC NewResultBuilder()",
		}

		for _, req := range required {
			if !strings.Contains(apiContent, req) {
				t.Errorf("API snapshot missing required function: %s", req)
			}
		}
	})
}
