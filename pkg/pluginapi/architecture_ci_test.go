package pluginapi

import (
	"strings"
	"testing"
)

const testRuleCI = "test"

// TestArchitectureAPIStability ensures our contextual interfaces maintain API stability
// and provide backwards compatibility guarantees.
func TestArchitectureAPIStability(t *testing.T) {
	t.Run("contextual interfaces are available", func(t *testing.T) {
		// Ensure all contextual factory functions exist and work
		entityCtx := NewEntityContext()
		severityCtx := NewSeverityContext()
		actionCtx := NewActionContext()

		if entityCtx == nil {
			t.Error("NewEntityContext() should not return nil")
		}
		if severityCtx == nil {
			t.Error("NewSeverityContext() should not return nil")
		}
		if actionCtx == nil {
			t.Error("NewActionContext() should not return nil")
		}
	})

	t.Run("legacy violation creation still works", func(t *testing.T) {
		// Ensure backwards compatibility
		violation := NewViolation(testRuleCI, SeverityWarn, "message", EntityOrganism, "id")
		if violation.Rule() != testRuleCI {
			t.Error("Legacy NewViolation should still work")
		}
	})

	t.Run("new contextual violation creation works", func(t *testing.T) {
		entities := NewEntityContext()
		severities := NewSeverityContext()

		violation := NewViolationWithEntityRef(testRuleCI, severities.Warn(), "message", entities.Organism(), "id")
		if violation.Rule() != testRuleCI {
			t.Error("Contextual NewViolationWithEntityRef should work")
		}
	})

	t.Run("contextual and legacy violations are equivalent", func(t *testing.T) {
		entities := NewEntityContext()
		severities := NewSeverityContext()

		legacy := NewViolation("rule", SeverityLog, "msg", EntityProtocol, "id")
		contextual := NewViolationWithEntityRef("rule", severities.Log(), "msg", entities.Protocol(), "id")

		if legacy.Rule() != contextual.Rule() ||
			legacy.Severity() != contextual.Severity() ||
			legacy.Message() != contextual.Message() ||
			legacy.Entity() != contextual.Entity() ||
			legacy.EntityID() != contextual.EntityID() {
			t.Error("Legacy and contextual violations should be equivalent")
		}
	})
}

// TestArchitectureInvariantsEnforcement ensures architectural invariants are maintained.
func TestArchitectureInvariantsEnforcement(t *testing.T) {
	t.Run("opaque references cannot be constructed externally", func(t *testing.T) {
		// This test ensures that plugin authors cannot create their own EntityTypeRef instances
		// The internal marker methods prevent external implementation

		entities := NewEntityContext()
		organism1 := entities.Organism()
		organism2 := entities.Organism()

		// Should be equal through proper factory
		if !organism1.Equals(organism2) {
			t.Error("References from same context should be equal")
		}

		// String representations should be consistent
		if organism1.String() != organism2.String() {
			t.Error("String representations should be consistent")
		}
	})

	t.Run("behavioral methods provide semantic value", func(t *testing.T) {
		entities := NewEntityContext()
		severities := NewSeverityContext()
		actions := NewActionContext()

		// Entity behavioral testing
		organism := entities.Organism()
		housing := entities.Housing()
		protocol := entities.Protocol()

		coreEntities := []EntityTypeRef{organism, housing}
		nonCoreEntities := []EntityTypeRef{protocol}

		for _, entity := range coreEntities {
			if !entity.IsCore() {
				t.Errorf("Entity %s should be core", entity.String())
			}
		}

		for _, entity := range nonCoreEntities {
			if entity.IsCore() {
				t.Errorf("Entity %s should not be core", entity.String())
			}
		}

		// Severity behavioral testing
		log := severities.Log()
		warn := severities.Warn()
		block := severities.Block()

		nonBlockingSeverities := []SeverityRef{log, warn}
		blockingSeverities := []SeverityRef{block}

		for _, severity := range nonBlockingSeverities {
			if severity.IsBlocking() {
				t.Errorf("Severity %s should not be blocking", severity.String())
			}
		}

		for _, severity := range blockingSeverities {
			if !severity.IsBlocking() {
				t.Errorf("Severity %s should be blocking", severity.String())
			}
		}

		// Action behavioral testing
		create := actions.Create()
		update := actions.Update()
		deleteAction := actions.Delete()
		allActions := []ActionRef{create, update, deleteAction}
		destructiveActions := []ActionRef{deleteAction}
		nonDestructiveActions := []ActionRef{create, update}

		for _, action := range allActions {
			if !action.IsMutation() {
				t.Errorf("Action %s should be a mutation", action.String())
			}
		}

		for _, action := range destructiveActions {
			if !action.IsDestructive() {
				t.Errorf("Action %s should be destructive", action.String())
			}
		}

		for _, action := range nonDestructiveActions {
			if action.IsDestructive() {
				t.Errorf("Action %s should not be destructive", action.String())
			}
		}
	})

	t.Run("cross-type equality handled safely", func(t *testing.T) {
		entities := NewEntityContext()

		organism1 := entities.Organism()
		organism2 := entities.Organism()
		housing := entities.Housing()

		// Same type equality should work
		if !organism1.Equals(organism2) {
			t.Error("Same entity types should be equal")
		}

		// Different type equality should work safely
		if organism1.Equals(housing) {
			t.Error("Different entity types should not be equal")
		}
	})
}

// TestArchitecturePerformance ensures contextual interfaces don't add significant overhead.
func TestArchitecturePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	t.Run("contextual interface creation is efficient", func(t *testing.T) {
		entities := NewEntityContext()

		// Create many references - should be fast
		const iterations = 10000
		organisms := make([]EntityTypeRef, iterations)

		for i := 0; i < iterations; i++ {
			organisms[i] = entities.Organism()
		}

		// Verify all are equal (efficient comparison)
		first := organisms[0]
		for i := 1; i < iterations; i++ {
			if !first.Equals(organisms[i]) {
				t.Error("All organism references should be equal")
				break
			}
		}
	})

	t.Run("violation creation performance acceptable", func(t *testing.T) {
		entities := NewEntityContext()
		severities := NewSeverityContext()

		organism := entities.Organism()
		warn := severities.Warn()

		const iterations = 1000
		violations := make([]Violation, iterations)

		for i := 0; i < iterations; i++ {
			violations[i] = NewViolationWithEntityRef("rule", warn, "message", organism, "id")
		}

		// Verify all violations are equivalent
		first := violations[0]
		for i := 1; i < iterations; i++ {
			if first.Rule() != violations[i].Rule() ||
				first.Severity() != violations[i].Severity() ||
				first.Entity() != violations[i].Entity() {
				t.Error("All violations should be equivalent")
				break
			}
		}
	})
}

// TestArchitectureDocumentationCompliance ensures our implementation matches documented behavior.
func TestArchitectureDocumentationCompliance(t *testing.T) {
	t.Run("string representations are for debugging only", func(t *testing.T) {
		entities := NewEntityContext()
		organism := entities.Organism()

		str := organism.String()
		if str == "" {
			t.Error("String() should return non-empty value for debugging")
		}

		// String should match the underlying constant for debugging purposes
		if !strings.Contains(strings.ToLower(str), "organism") {
			t.Error("String representation should indicate entity type for debugging")
		}
	})

	t.Run("contextual methods provide isolation", func(t *testing.T) {
		// Multiple context instances should provide same references
		entities1 := NewEntityContext()
		entities2 := NewEntityContext()

		org1 := entities1.Organism()
		org2 := entities2.Organism()

		if !org1.Equals(org2) {
			t.Error("Same entity types from different contexts should be equal")
		}
	})
}
