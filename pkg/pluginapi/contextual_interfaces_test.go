package pluginapi

import (
	"testing"
)

func TestEntityContext(t *testing.T) {
	ctx := NewEntityContext()

	organism := ctx.Organism()
	housing := ctx.Housing()
	protocol := ctx.Protocol()

	if organism.Equals(housing) {
		t.Error("Organism and Housing should not be equal")
	}
	if housing.Equals(protocol) {
		t.Error("Housing and Protocol should not be equal")
	}
	if protocol.Equals(organism) {
		t.Error("Protocol and Organism should not be equal")
	}

	if organism.String() == "" {
		t.Error("Organism string representation should not be empty")
	}

	organism2 := ctx.Organism()
	if !organism.Equals(organism2) {
		t.Error("Same entity types should be equal")
	}

	if !organism.IsCore() {
		t.Error("Organism should be core entity")
	}
	if !housing.IsCore() {
		t.Error("Housing should be core entity")
	}
	if protocol.IsCore() {
		t.Error("Protocol should not be core entity")
	}
}

func TestSeverityContext(t *testing.T) {
	ctx := NewSeverityContext()

	log := ctx.Log()
	warn := ctx.Warn()
	block := ctx.Block()

	if log.Equals(warn) {
		t.Error("Log and Warn should not be equal")
	}
	if warn.Equals(block) {
		t.Error("Warn and Block should not be equal")
	}

	if log.String() == "" {
		t.Error("Log string representation should not be empty")
	}

	log2 := ctx.Log()
	if !log.Equals(log2) {
		t.Error("Same severity types should be equal")
	}

	if log.IsBlocking() {
		t.Error("Log should not be blocking")
	}
	if warn.IsBlocking() {
		t.Error("Warn should not be blocking")
	}
	if !block.IsBlocking() {
		t.Error("Block should be blocking")
	}
}

func TestActionContext(t *testing.T) {
	ctx := NewActionContext()

	create := ctx.Create()
	update := ctx.Update()
	deleteAction := ctx.Delete()

	if create.Equals(update) {
		t.Error("Create and Update should not be equal")
	}

	if create.String() == "" {
		t.Error("Create string representation should not be empty")
	}

	create2 := ctx.Create()
	if !create.Equals(create2) {
		t.Error("Same action types should be equal")
	}

	if !create.IsMutation() {
		t.Error("Create should be a mutation")
	}
	if !update.IsMutation() {
		t.Error("Update should be a mutation")
	}
	if !deleteAction.IsMutation() {
		t.Error("Delete should be a mutation")
	}

	if create.IsDestructive() {
		t.Error("Create should not be destructive")
	}
	if update.IsDestructive() {
		t.Error("Update should not be destructive")
	}
	if !deleteAction.IsDestructive() {
		t.Error("Delete should be destructive")
	}
}

func TestNewViolationWithEntityRef(t *testing.T) {
	entityCtx := NewEntityContext()
	severityCtx := NewSeverityContext()

	// Create violation using contextual interfaces
	organism := entityCtx.Organism()
	warn := severityCtx.Warn()

	violation := NewViolationWithEntityRef("test-rule", warn, "Test message", organism, "test-id")

	// Verify the violation was created correctly
	if violation.Rule() != "test-rule" {
		t.Errorf("Expected rule 'test-rule', got '%s'", violation.Rule())
	}
	if violation.Message() != "Test message" {
		t.Errorf("Expected message 'Test message', got '%s'", violation.Message())
	}
	if violation.EntityID() != "test-id" {
		t.Errorf("Expected entity ID 'test-id', got '%s'", violation.EntityID())
	}

	// Verify the underlying types were correctly extracted
	if violation.Severity() != severityWarn {
		t.Errorf("Expected severity %s, got %s", severityWarn, violation.Severity())
	}
	if violation.Entity() != entityOrganism {
		t.Errorf("Expected entity %s, got %s", entityOrganism, violation.Entity())
	}

	// Test with different contextual references
	housing := entityCtx.Housing()
	block := severityCtx.Block()

	violation2 := NewViolationWithEntityRef("test-rule-2", block, "Block message", housing, "housing-1")

	if violation2.Severity() != severityBlock {
		t.Errorf("Expected severity %s, got %s", severityBlock, violation2.Severity())
	}
	if violation2.Entity() != entityHousingUnit {
		t.Errorf("Expected entity %s, got %s", entityHousingUnit, violation2.Entity())
	}
}

func TestContextualViolationEquivalence(t *testing.T) {
	entityCtx := NewEntityContext()
	severityCtx := NewSeverityContext()

	// Create the same violation using both methods
	rawViolation := newViolationForTest("rule", severityLog, "message", entityProtocol, "id")
	contextualViolation := NewViolationWithEntityRef("rule", severityCtx.Log(), "message", entityCtx.Protocol(), "id")

	// They should be equivalent
	if rawViolation.Rule() != contextualViolation.Rule() {
		t.Error("Rules should be identical")
	}
	if rawViolation.Severity() != contextualViolation.Severity() {
		t.Error("Severities should be identical")
	}
	if rawViolation.Message() != contextualViolation.Message() {
		t.Error("Messages should be identical")
	}
	if rawViolation.Entity() != contextualViolation.Entity() {
		t.Error("Entities should be identical")
	}
	if rawViolation.EntityID() != contextualViolation.EntityID() {
		t.Error("Entity IDs should be identical")
	}
}

func TestInternalMarkerMethods(t *testing.T) {
	// Test entity marker method
	entityCtx := NewEntityContext()
	organism := entityCtx.Organism()
	organism.isEntityTypeRef() // Should not panic - if it panics, test will fail

	// Test action marker method
	actionCtx := NewActionContext()
	create := actionCtx.Create()
	create.isActionRef() // Should not panic - if it panics, test will fail

	// Test severity marker method
	severityCtx := NewSeverityContext()
	warn := severityCtx.Warn()
	warn.isSeverityRef() // Should not panic - if it panics, test will fail

	// If we reach here, all marker methods work correctly
	t.Log("All marker methods executed successfully")
}
