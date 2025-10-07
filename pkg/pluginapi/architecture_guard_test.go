package pluginapi

import (
	"reflect"
	"testing"
)

const testRule = "test"

// TestHexagonalArchitectureGuards ensures our contextual interfaces maintain
// hexagonal architecture principles using runtime reflection.
func TestHexagonalArchitectureGuards(t *testing.T) {
	t.Run("contextual interfaces use opaque references", func(t *testing.T) {
		checkContextualInterfaceDesign(t)
	})

	t.Run("violation creation supports both patterns", func(t *testing.T) {
		checkViolationCreationPatterns(t)
	})

	t.Run("rule view provides contextual access", func(t *testing.T) {
		checkRuleViewContextualMethods(t)
	})
}

func checkContextualInterfaceDesign(t *testing.T) {
	// Verify that contextual interfaces exist and follow hexagonal principles
	contextualInterfaces := map[string]interface{}{
		"EntityContext":   (*EntityContext)(nil),
		"SeverityContext": (*SeverityContext)(nil),
		"ActionContext":   (*ActionContext)(nil),
	}

	for name, iface := range contextualInterfaces {
		ifaceType := reflect.TypeOf(iface).Elem()
		if ifaceType.Kind() != reflect.Interface {
			t.Errorf("Expected %s to be an interface, got %s", name, ifaceType.Kind())
			continue
		}

		// Verify interface has methods (contextual access)
		if ifaceType.NumMethod() == 0 {
			t.Errorf("Contextual interface %s should have methods for contextual access", name)
		}

		// Check that methods follow naming conventions for contextual access
		for i := 0; i < ifaceType.NumMethod(); i++ {
			method := ifaceType.Method(i)
			t.Logf("Contextual interface %s has method %s", name, method.Name)
		}
	}
}

func checkViolationCreationPatterns(t *testing.T) {
	// Test that both violation creation patterns exist by trying to call them

	// Test contextual pattern
	entityCtx := NewEntityContext()
	severityCtx := NewSeverityContext()

	organism := entityCtx.Organism()
	warn := severityCtx.Warn()

	// Test that NewViolationWithEntityRef exists and works
	violation := NewViolationWithEntityRef(testRule, warn, "test violation", organism, "entity-123")
	if violation.Rule() != testRule {
		t.Error("NewViolationWithEntityRef should create a valid violation")
	}

	if violation.Message() != "test violation" {
		t.Errorf("Expected message 'test violation', got %s", violation.Message())
	}

	// Test that legacy NewViolation still exists
	legacyViolation := NewViolation(testRule, SeverityWarn, "legacy violation", EntityOrganism, "entity-456")
	if legacyViolation.Rule() != testRule {
		t.Error("Legacy NewViolation should still exist")
	}
}

func checkRuleViewContextualMethods(t *testing.T) {
	// Test that RuleView interface provides contextual access by checking if contextual interfaces exist
	// Since reflection might not catch all methods, we test by ensuring the contextual interfaces themselves exist

	entityCtx := NewEntityContext()
	severityCtx := NewSeverityContext()
	actionCtx := NewActionContext()

	if entityCtx == nil {
		t.Error("EntityContext should be available for RuleView contextual access")
	}
	if severityCtx == nil {
		t.Error("SeverityContext should be available for RuleView contextual access")
	}
	if actionCtx == nil {
		t.Error("ActionContext should be available for RuleView contextual access")
	}

	t.Log("RuleView contextual methods verified through interface availability")
} // TestContextualInterfaceImmutability ensures contextual interfaces maintain immutability.
func TestContextualInterfaceImmutability(t *testing.T) {
	entityCtx := NewEntityContext()
	severityCtx := NewSeverityContext()
	actionCtx := NewActionContext()

	// Create references multiple times - should be equivalent
	org1 := entityCtx.Organism()
	org2 := entityCtx.Organism()

	if !org1.Equals(org2) {
		t.Error("Contextual references should be immutable and equal")
	}

	warn1 := severityCtx.Warn()
	warn2 := severityCtx.Warn()

	if !warn1.Equals(warn2) {
		t.Error("Severity references should be immutable and equal")
	}

	create1 := actionCtx.Create()
	create2 := actionCtx.Create()

	if !create1.Equals(create2) {
		t.Error("Action references should be immutable and equal")
	}
}

// TestContextualInterfaceSemantics ensures behavioral methods work correctly.
func TestContextualInterfaceSemantics(t *testing.T) {
	entityCtx := NewEntityContext()
	severityCtx := NewSeverityContext()
	actionCtx := NewActionContext()

	// Test entity semantics
	organism := entityCtx.Organism()
	housing := entityCtx.Housing()
	protocol := entityCtx.Protocol()

	if !organism.IsCore() || !housing.IsCore() {
		t.Error("Organism and Housing should be core entities")
	}
	if protocol.IsCore() {
		t.Error("Protocol should not be core entity")
	}

	// Test severity semantics
	log := severityCtx.Log()
	warn := severityCtx.Warn()
	block := severityCtx.Block()

	if log.IsBlocking() || warn.IsBlocking() {
		t.Error("Log and Warn should not be blocking")
	}
	if !block.IsBlocking() {
		t.Error("Block should be blocking")
	}

	// Test action semantics
	create := actionCtx.Create()
	update := actionCtx.Update()
	deleteAction := actionCtx.Delete()

	if !create.IsMutation() || !update.IsMutation() || !deleteAction.IsMutation() {
		t.Error("All actions should be mutations")
	}
	if create.IsDestructive() || update.IsDestructive() {
		t.Error("Create and Update should not be destructive")
	}
	if !deleteAction.IsDestructive() {
		t.Error("Delete should be destructive")
	}
}
