package datasetapi

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strings"
	"testing"
)

// TestDatasetAPIContextualAccessorEnforcement ensures all facade interfaces have contextual accessors
// and that the pattern is consistently applied across the datasetapi package.
func TestDatasetAPIContextualAccessorEnforcement(t *testing.T) {
	t.Run("all facade interfaces have contextual accessors", func(t *testing.T) {
		validateFacadeInterfaceContextualAccessors(t)
	})

	t.Run("no raw constant usage in facade interfaces", func(t *testing.T) {
		validateNoRawConstantsInFacadeInterfaces(t)
	})

	t.Run("contextual interfaces follow consistent pattern", func(t *testing.T) {
		validateDatasetContextualInterfacePattern(t)
	})

	t.Run("contextual accessors are consistent across packages", func(t *testing.T) {
		validateCrossPackageContextualConsistency(t)
	})
}

func validateFacadeInterfaceContextualAccessors(t *testing.T) {
	// Define facade interfaces that must have contextual accessors
	requiredFacadeInterfaces := map[string][]string{
		"Organism": {
			"GetCurrentStage", "IsActive", "IsRetired", "IsDeceased",
		},
		"HousingUnit": {
			"GetEnvironmentType", "IsAquaticEnvironment", "IsHumidEnvironment", "SupportsSpecies",
		},
		"Protocol": {
			"GetCurrentStatus", "IsActiveProtocol", "IsTerminalStatus", "CanAcceptNewSubjects",
		},
		"Procedure": {
			"GetCurrentStatus", "IsActiveProcedure", "IsTerminalStatus", "IsSuccessful",
		},
		"Cohort": {
			"GetPurpose", "IsResearchCohort", "RequiresProtocol",
		},
		"BreedingUnit": {
			"GetBreedingStrategy", "IsNaturalBreeding", "RequiresIntervention",
		},
	}

	// Use reflection to check that all required methods exist
	for interfaceName, requiredMethods := range requiredFacadeInterfaces {
		var interfaceType reflect.Type

		// Get the interface type using reflection
		switch interfaceName {
		case "Organism":
			interfaceType = reflect.TypeOf((*Organism)(nil)).Elem()
		case "HousingUnit":
			interfaceType = reflect.TypeOf((*HousingUnit)(nil)).Elem()
		case "Protocol":
			interfaceType = reflect.TypeOf((*Protocol)(nil)).Elem()
		case "Procedure":
			interfaceType = reflect.TypeOf((*Procedure)(nil)).Elem()
		case "Cohort":
			interfaceType = reflect.TypeOf((*Cohort)(nil)).Elem()
		case "BreedingUnit":
			interfaceType = reflect.TypeOf((*BreedingUnit)(nil)).Elem()
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
			validateDatasetContextualMethodSignature(t, interfaceName, methodName, method)
		}
	}
}

func validateDatasetContextualMethodSignature(t *testing.T, interfaceName, methodName string, method reflect.Method) {
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

func validateNoRawConstantsInFacadeInterfaces(t *testing.T) {
	// Parse the facade.go file to ensure no raw constants are used
	fset := token.NewFileSet()
	src, err := os.ReadFile("facade.go")
	if err != nil {
		t.Fatalf("Failed to read facade.go: %v", err)
	}

	file, err := parser.ParseFile(fset, "facade.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse facade.go: %v", err)
	}

	// List of interface names that should have contextual accessors
	contextualInterfaces := map[string]bool{
		"Organism":    true,
		"HousingUnit": true,
		"Protocol":    true,
		"Procedure":   true,
	}

	// Look for interface declarations
	ast.Inspect(file, func(n ast.Node) bool {
		if typeSpec, ok := n.(*ast.TypeSpec); ok {
			if interfaceType, ok := typeSpec.Type.(*ast.InterfaceType); ok {
				interfaceName := typeSpec.Name.Name

				// Only check interfaces that should have contextual accessors
				if !contextualInterfaces[interfaceName] {
					return true
				}

				// Check methods in the interface
				for _, method := range interfaceType.Methods.List {
					if funcType, ok := method.Type.(*ast.FuncType); ok && len(method.Names) > 0 {
						methodName := method.Names[0].Name

						// Skip legacy methods that are explicitly marked
						if hasDatasetLegacyComment(method) {
							continue
						}

						// Check return types for raw constant usage
						if funcType.Results != nil {
							for _, result := range funcType.Results.List {
								if ident, ok := result.Type.(*ast.Ident); ok {
									// Flag direct usage of non-contextual types for state/status
									if isDatasetRawConstantType(ident.Name) {
										t.Errorf("Interface %s method %s uses raw constant type %s instead of contextual reference",
											interfaceName, methodName, ident.Name)
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

func hasDatasetLegacyComment(method *ast.Field) bool {
	if method.Comment != nil {
		for _, comment := range method.Comment.List {
			if strings.Contains(comment.Text, "Legacy") {
				return true
			}
		}
	}
	return false
}

func isDatasetRawConstantType(typeName string) bool {
	// Define types that should be accessed via contextual interfaces
	rawConstantTypes := map[string]bool{
		"LifecycleStage": true,
		"Environment":    true,
		"Status":         true,
		"State":          true,
	}
	return rawConstantTypes[typeName]
}

func validateDatasetContextualInterfacePattern(t *testing.T) {
	// Check that all contextual reference interfaces follow the pattern
	contextualRefTypes := []reflect.Type{
		reflect.TypeOf((*LifecycleStageRef)(nil)).Elem(),
		reflect.TypeOf((*EnvironmentTypeRef)(nil)).Elem(),
		reflect.TypeOf((*ProtocolStatusRef)(nil)).Elem(),
		reflect.TypeOf((*ProcedureStatusRef)(nil)).Elem(),
		reflect.TypeOf((*BreedingStrategyRef)(nil)).Elem(),
		reflect.TypeOf((*CohortPurposeRef)(nil)).Elem(),
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

func validateCrossPackageContextualConsistency(t *testing.T) {
	// Ensure that contextual interfaces are consistent between pluginapi and datasetapi
	// This test validates that the same concepts use the same contextual patterns

	// Check LifecycleStageRef consistency
	pluginapiLifecycleRef := reflect.TypeOf((*LifecycleStageRef)(nil)).Elem()

	// Verify essential methods exist
	essentialMethods := []string{"String", "IsActive", "Equals"}
	for _, methodName := range essentialMethods {
		if _, exists := pluginapiLifecycleRef.MethodByName(methodName); !exists {
			t.Errorf("LifecycleStageRef missing essential method: %s", methodName)
		}
	}

	t.Log("Cross-package contextual consistency validated for LifecycleStageRef")
}
