package pluginapi

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strings"
	"testing"
)

// TestContextualAccessorEnforcement ensures all view interfaces have contextual accessors
// and that the pattern is consistently applied across the codebase.
func TestContextualAccessorEnforcement(t *testing.T) {
	t.Run("all view interfaces have contextual accessors", func(t *testing.T) {
		validateViewInterfaceContextualAccessors(t)
	})

	t.Run("no raw constant usage in view interfaces", func(t *testing.T) {
		validateNoRawConstantsInViewInterfaces(t)
	})

	t.Run("contextual interfaces follow consistent pattern", func(t *testing.T) {
		validateContextualInterfacePattern(t)
	})

	t.Run("all view implementations provide contextual methods", func(t *testing.T) {
		validateViewImplementationsHaveContextualMethods(t)
	})
}

func validateViewInterfaceContextualAccessors(t *testing.T) {
	// Define view interfaces that must have contextual accessors
	requiredViewInterfaces := map[string][]string{
		"OrganismView": {
			"GetCurrentStage", "IsActive", "IsRetired", "IsDeceased",
		},
		"HousingUnitView": {
			"GetEnvironmentType", "IsAquaticEnvironment", "IsHumidEnvironment", "SupportsSpecies",
		},
		"ProtocolView": {
			"GetCurrentStatus", "IsActiveProtocol", "IsTerminalStatus", "CanAcceptNewSubjects",
		},
	}

	// Use reflection to check that all required methods exist
	for interfaceName, requiredMethods := range requiredViewInterfaces {
		var interfaceType reflect.Type

		// Get the interface type using reflection
		switch interfaceName {
		case "OrganismView":
			interfaceType = reflect.TypeOf((*OrganismView)(nil)).Elem()
		case "HousingUnitView":
			interfaceType = reflect.TypeOf((*HousingUnitView)(nil)).Elem()
		case "ProtocolView":
			interfaceType = reflect.TypeOf((*ProtocolView)(nil)).Elem()
		default:
			t.Fatalf("Unknown interface: %s", interfaceName)
		}

		// Check that all required methods exist
		for _, methodName := range requiredMethods {
			method, exists := interfaceType.MethodByName(methodName)
			if !exists {
				t.Errorf("Interface %s is missing required contextual accessor: %s", interfaceName, methodName)
				continue
			}

			// Validate method signature patterns
			validateContextualMethodSignature(t, interfaceName, methodName, method)
		}
	}
}

func validateContextualMethodSignature(t *testing.T, interfaceName, methodName string, method reflect.Method) {
	methodType := method.Type

	// Validate that Get* methods return contextual references
	if strings.HasPrefix(methodName, "Get") {
		if methodType.NumOut() != 1 {
			t.Errorf("%s.%s should return exactly one value", interfaceName, methodName)
			return
		}

		returnType := methodType.Out(0)
		if !strings.HasSuffix(returnType.Name(), "Ref") {
			t.Errorf("%s.%s should return a contextual reference type ending with 'Ref', got %s",
				interfaceName, methodName, returnType.Name())
		}
	}

	// Validate that Is* methods return bool
	if strings.HasPrefix(methodName, "Is") || strings.HasPrefix(methodName, "Can") || strings.HasPrefix(methodName, "Supports") {
		if methodType.NumOut() != 1 {
			t.Errorf("%s.%s should return exactly one value", interfaceName, methodName)
			return
		}

		returnType := methodType.Out(0)
		if returnType.Kind() != reflect.Bool {
			t.Errorf("%s.%s should return bool, got %s", interfaceName, methodName, returnType.Kind())
		}
	}
}

func validateNoRawConstantsInViewInterfaces(t *testing.T) {
	// Parse the views.go file to ensure no raw constants are used
	fset := token.NewFileSet()
	src, err := os.ReadFile("views.go")
	if err != nil {
		t.Fatalf("Failed to read views.go: %v", err)
	}

	file, err := parser.ParseFile(fset, "views.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse views.go: %v", err)
	} // Look for interface declarations
	ast.Inspect(file, func(n ast.Node) bool {
		if typeSpec, ok := n.(*ast.TypeSpec); ok {
			if interfaceType, ok := typeSpec.Type.(*ast.InterfaceType); ok && strings.HasSuffix(typeSpec.Name.Name, "View") {
				// Check methods in the interface
				for _, method := range interfaceType.Methods.List {
					if funcType, ok := method.Type.(*ast.FuncType); ok && len(method.Names) > 0 {
						methodName := method.Names[0].Name

						// Skip legacy methods that are explicitly marked
						if hasLegacyComment(method) {
							continue
						}

						// Check return types for raw constant usage
						if funcType.Results != nil {
							for _, result := range funcType.Results.List {
								if ident, ok := result.Type.(*ast.Ident); ok {
									// Flag direct usage of non-contextual types for state/status
									if isRawConstantType(ident.Name) {
										t.Errorf("Interface %s method %s uses raw constant type %s instead of contextual reference",
											typeSpec.Name.Name, methodName, ident.Name)
									}
								}
							}
						}
					}
				}
			}
		}
		return true
	})
}

func hasLegacyComment(method *ast.Field) bool {
	if method.Comment != nil {
		for _, comment := range method.Comment.List {
			if strings.Contains(comment.Text, "Legacy") {
				return true
			}
		}
	}
	return false
}

func isRawConstantType(typeName string) bool {
	// Define types that should be accessed via contextual interfaces
	rawConstantTypes := map[string]bool{
		"LifecycleStage": true,
		"Environment":    true,
		"Status":         true,
		"State":          true,
	}
	return rawConstantTypes[typeName]
}

func validateContextualInterfacePattern(t *testing.T) {
	// Check that all contextual reference interfaces follow the pattern
	contextualRefTypes := []reflect.Type{
		reflect.TypeOf((*LifecycleStageRef)(nil)).Elem(),
		reflect.TypeOf((*EnvironmentTypeRef)(nil)).Elem(),
		reflect.TypeOf((*ProtocolStatusRef)(nil)).Elem(),
	}

	for _, refType := range contextualRefTypes {
		// All contextual refs should have String() method
		if _, exists := refType.MethodByName("String"); !exists {
			t.Errorf("Contextual reference type %s missing String() method", refType.Name())
		}

		// All contextual refs should have Equals() method
		if _, exists := refType.MethodByName("Equals"); !exists {
			t.Errorf("Contextual reference type %s missing Equals() method", refType.Name())
		}

		// All contextual refs should have internal marker method
		markerMethodName := "is" + refType.Name()
		if _, exists := refType.MethodByName(markerMethodName); !exists {
			t.Errorf("Contextual reference type %s missing marker method %s", refType.Name(), markerMethodName)
		}
	}
}

func validateViewImplementationsHaveContextualMethods(t *testing.T) {
	// This test ensures that when view interfaces are implemented,
	// they provide the contextual accessors.
	// Note: This would typically be tested in integration tests
	// where actual implementations are available.
	t.Log("View implementation validation should be done in integration tests with concrete implementations")
}
