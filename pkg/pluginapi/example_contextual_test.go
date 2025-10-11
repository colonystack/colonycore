package pluginapi_test

import (
	"fmt"

	"colonycore/pkg/pluginapi"
)

// Example demonstrates hexagonal architecture violation creation using contextual interfaces
func ExampleNewViolationWithEntityRef() {
	// Get contextual interface instances
	entities := pluginapi.NewEntityContext()
	severities := pluginapi.NewSeverityContext()

	// Create violations using contextual interfaces instead of raw constants
	violation1 := pluginapi.NewViolationWithEntityRef(
		"habitat-rule",
		severities.Warn(),
		"Organism requires aquatic environment",
		entities.Organism(),
		"frog-123",
	)

	violation2 := pluginapi.NewViolationWithEntityRef(
		"capacity-rule",
		severities.Block(),
		"Housing unit exceeds maximum capacity",
		entities.Housing(),
		"tank-456",
	)

	// The violations work the same as those created with raw constants
	fmt.Printf("Violation 1: %s (%s)\n", violation1.Message(), violation1.Severity())
	fmt.Printf("Violation 2: %s (%s)\n", violation2.Message(), violation2.Severity())

	// Output:
	// Violation 1: Organism requires aquatic environment (warn)
	// Violation 2: Housing unit exceeds maximum capacity (block)
}

// Example demonstrates builder patterns for creating violations
func ExampleNewViolationBuilder() {
	// Builder pattern provides fluent interface and validation
	violation1, err := pluginapi.NewViolationBuilder().
		WithRule("habitat-rule").
		WithMessage("Organism requires aquatic environment").
		WithEntity(pluginapi.NewEntityContext().Organism()).
		WithEntityID("frog-123").
		BuildWarning()
	if err != nil {
		panic(err)
	}

	// Convenient methods for common severity levels
	violation2, err := pluginapi.NewViolationBuilder().
		WithRule("capacity-rule").
		WithMessage("Housing unit exceeds maximum capacity").
		WithEntity(pluginapi.NewEntityContext().Housing()).
		WithEntityID("tank-456").
		BuildBlocking()
	if err != nil {
		panic(err)
	}

	// Builder pattern for complex results
	result := pluginapi.NewResultBuilder().
		AddViolation(violation1).
		AddViolation(violation2).
		Build()

	fmt.Printf("Result has %d violations, blocking: %v\n", len(result.Violations()), result.HasBlocking())

	// Output:
	// Result has 2 violations, blocking: true
}

// Example shows how hexagonal architecture promotes testability
func Example_contextualInterfacesBenefits() {
	entities := pluginapi.NewEntityContext()
	severities := pluginapi.NewSeverityContext()

	// Plugin rules can work with behavioral methods instead of raw constants
	organism := entities.Organism()
	housing := entities.Housing()
	protocol := entities.Protocol()

	fmt.Printf("Organism is core entity: %t\n", organism.IsCore())
	fmt.Printf("Housing is core entity: %t\n", housing.IsCore())
	fmt.Printf("Protocol is core entity: %t\n", protocol.IsCore())

	warn := severities.Warn()
	block := severities.Block()

	fmt.Printf("Warn is blocking: %t\n", warn.IsBlocking())
	fmt.Printf("Block is blocking: %t\n", block.IsBlocking())

	// Output:
	// Organism is core entity: true
	// Housing is core entity: true
	// Protocol is core entity: false
	// Warn is blocking: false
	// Block is blocking: true
}
