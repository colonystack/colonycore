package pluginapi

import (
	"testing"
)

func TestViolationBuilder(t *testing.T) {
	t.Run("successful build", func(t *testing.T) {
		entityCtx := NewEntityContext()
		severityCtx := NewSeverityContext()

		violation, err := NewViolationBuilder().
			WithRule("test-rule").
			WithSeverity(severityCtx.Warn()).
			WithMessage("test message").
			WithEntity(entityCtx.Organism()).
			WithEntityID("test-id").
			Build()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if violation.Rule() != "test-rule" {
			t.Errorf("expected rule 'test-rule', got '%s'", violation.Rule())
		}
		if violation.Message() != "test message" {
			t.Errorf("expected message 'test message', got '%s'", violation.Message())
		}
		if violation.EntityID() != "test-id" {
			t.Errorf("expected entity ID 'test-id', got '%s'", violation.EntityID())
		}
		if violation.Severity() != severityWarn {
			t.Errorf("expected severity %s, got %s", severityWarn, violation.Severity())
		}
		if violation.Entity() != entityOrganism {
			t.Errorf("expected entity %s, got %s", entityOrganism, violation.Entity())
		}
	})

	t.Run("build warning", func(t *testing.T) {
		entityCtx := NewEntityContext()

		violation, err := NewViolationBuilder().
			WithRule("warning-rule").
			WithMessage("warning message").
			WithEntity(entityCtx.Organism()).
			WithEntityID("warn-id").
			BuildWarning()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if violation.Severity() != severityWarn {
			t.Errorf("expected warning severity, got %s", violation.Severity())
		}
	})

	t.Run("build blocking", func(t *testing.T) {
		entityCtx := NewEntityContext()

		violation, err := NewViolationBuilder().
			WithRule("block-rule").
			WithMessage("block message").
			WithEntity(entityCtx.Housing()).
			WithEntityID("block-id").
			BuildBlocking()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if violation.Severity() != severityBlock {
			t.Errorf("expected blocking severity, got %s", violation.Severity())
		}
	})

	t.Run("build log", func(t *testing.T) {
		entityCtx := NewEntityContext()

		violation, err := NewViolationBuilder().
			WithRule("log-rule").
			WithMessage("log message").
			WithEntity(entityCtx.Protocol()).
			WithEntityID("log-id").
			BuildLog()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if violation.Severity() != severityLog {
			t.Errorf("expected log severity, got %s", violation.Severity())
		}
	})

	t.Run("validation errors", func(t *testing.T) {
		// Missing rule
		_, err := NewViolationBuilder().
			WithSeverity(NewSeverityContext().Warn()).
			WithEntity(NewEntityContext().Organism()).
			Build()
		if err == nil || err.Error() != "rule is required" {
			t.Errorf("expected 'rule is required' error, got %v", err)
		}

		// Missing severity
		_, err = NewViolationBuilder().
			WithRule("test").
			WithEntity(NewEntityContext().Organism()).
			Build()
		if err == nil || err.Error() != "severity is required" {
			t.Errorf("expected 'severity is required' error, got %v", err)
		}

		// Missing entity
		_, err = NewViolationBuilder().
			WithRule("test").
			WithSeverity(NewSeverityContext().Warn()).
			Build()
		if err == nil || err.Error() != "entity is required" {
			t.Errorf("expected 'entity is required' error, got %v", err)
		}
	})

	t.Run("builder reuse prevention", func(t *testing.T) {
		builder := NewViolationBuilder().
			WithRule("test").
			WithSeverity(NewSeverityContext().Warn()).
			WithEntity(NewEntityContext().Organism())

		// First build should work
		_, err := builder.Build()
		if err != nil {
			t.Fatalf("first build failed: %v", err)
		}

		// Second build should fail
		_, err = builder.Build()
		if err == nil || err.Error() != "builder already used" {
			t.Errorf("expected 'builder already used' error, got %v", err)
		}
	})
}

func TestChangeBuilder(t *testing.T) {
	t.Run("successful build", func(t *testing.T) {
		entityCtx := NewEntityContext()
		actionCtx := NewActionContext()

		before := map[string]any{"name": "old"}
		after := map[string]any{"name": "new"}

		change, err := NewChangeBuilder().
			WithEntity(entityCtx.Organism()).
			WithAction(actionCtx.Update()).
			WithBefore(before).
			WithAfter(after).
			Build()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if change.Entity() != entityOrganism {
			t.Errorf("expected entity %s, got %s", entityOrganism, change.Entity())
		}
		if change.Action() != actionUpdate {
			t.Errorf("expected action %s, got %s", actionUpdate, change.Action())
		}
	})

	t.Run("validation errors", func(t *testing.T) {
		// Missing entity
		_, err := NewChangeBuilder().
			WithAction(NewActionContext().Create()).
			Build()
		if err == nil || err.Error() != "entity is required" {
			t.Errorf("expected 'entity is required' error, got %v", err)
		}

		// Missing action
		_, err = NewChangeBuilder().
			WithEntity(NewEntityContext().Organism()).
			Build()
		if err == nil || err.Error() != "action is required" {
			t.Errorf("expected 'action is required' error, got %v", err)
		}
	})

	t.Run("builder reuse prevention", func(t *testing.T) {
		builder := NewChangeBuilder().
			WithEntity(NewEntityContext().Organism()).
			WithAction(NewActionContext().Create())

		// First build should work
		_, err := builder.Build()
		if err != nil {
			t.Fatalf("first build failed: %v", err)
		}

		// Second build should fail
		_, err = builder.Build()
		if err == nil || err.Error() != "builder already used" {
			t.Errorf("expected 'builder already used' error, got %v", err)
		}
	})
}

func TestResultBuilder(t *testing.T) {
	t.Run("successful build empty", func(t *testing.T) {
		result := NewResultBuilder().Build()

		if result.HasBlocking() {
			t.Error("empty result should not have blocking violations")
		}
		if len(result.Violations()) != 0 {
			t.Errorf("expected 0 violations, got %d", len(result.Violations()))
		}
	})

	t.Run("successful build with violations", func(t *testing.T) {
		violation1, _ := NewViolationBuilder().
			WithRule("rule1").
			WithSeverity(NewSeverityContext().Warn()).
			WithMessage("message1").
			WithEntity(NewEntityContext().Organism()).
			WithEntityID("id1").
			Build()

		violation2, _ := NewViolationBuilder().
			WithRule("rule2").
			WithSeverity(NewSeverityContext().Block()).
			WithMessage("message2").
			WithEntity(NewEntityContext().Housing()).
			WithEntityID("id2").
			Build()

		result := NewResultBuilder().
			AddViolation(violation1).
			AddViolation(violation2).
			Build()

		if !result.HasBlocking() {
			t.Error("result should have blocking violations")
		}
		if len(result.Violations()) != 2 {
			t.Errorf("expected 2 violations, got %d", len(result.Violations()))
		}
	})

	t.Run("add multiple violations", func(t *testing.T) {
		violation1, _ := NewViolationBuilder().
			WithRule("rule1").
			WithSeverity(NewSeverityContext().Warn()).
			WithEntity(NewEntityContext().Organism()).
			Build()

		violation2, _ := NewViolationBuilder().
			WithRule("rule2").
			WithSeverity(NewSeverityContext().Log()).
			WithEntity(NewEntityContext().Protocol()).
			Build()

		result := NewResultBuilder().
			AddViolations(violation1, violation2).
			Build()

		if len(result.Violations()) != 2 {
			t.Errorf("expected 2 violations, got %d", len(result.Violations()))
		}
	})

	t.Run("from builder", func(t *testing.T) {
		vb := NewViolationBuilder().
			WithRule("from-builder").
			WithSeverity(NewSeverityContext().Warn()).
			WithEntity(NewEntityContext().Organism())

		result := NewResultBuilder().
			FromBuilder(vb).
			Build()

		if len(result.Violations()) != 1 {
			t.Errorf("expected 1 violation, got %d", len(result.Violations()))
		}
		if result.Violations()[0].Rule() != "from-builder" {
			t.Errorf("expected rule 'from-builder', got '%s'", result.Violations()[0].Rule())
		}
	})

	t.Run("merge result", func(t *testing.T) {
		violation, _ := NewViolationBuilder().
			WithRule("existing").
			WithSeverity(NewSeverityContext().Log()).
			WithEntity(NewEntityContext().Protocol()).
			Build()

		existingResult := NewResult(violation)

		newViolation, _ := NewViolationBuilder().
			WithRule("new").
			WithSeverity(NewSeverityContext().Warn()).
			WithEntity(NewEntityContext().Organism()).
			Build()

		result := NewResultBuilder().
			MergeResult(existingResult).
			AddViolation(newViolation).
			Build()

		if len(result.Violations()) != 2 {
			t.Errorf("expected 2 violations, got %d", len(result.Violations()))
		}
	})

	t.Run("builder reuse prevention", func(t *testing.T) {
		builder := NewResultBuilder()

		// First build should work
		_ = builder.Build()

		// Adding more violations should panic
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic when modifying used builder")
			}
		}()

		violation, _ := NewViolationBuilder().
			WithRule("test").
			WithSeverity(NewSeverityContext().Warn()).
			WithEntity(NewEntityContext().Organism()).
			Build()

		builder.AddViolation(violation)
	})
}

func TestViolationBuilderPanicPaths(t *testing.T) {
	t.Run("WithRule after build panics", func(t *testing.T) {
		builder := NewViolationBuilder().
			WithRule("test").
			WithSeverity(NewSeverityContext().Warn()).
			WithEntity(NewEntityContext().Organism())

		_, _ = builder.Build()

		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic when modifying used builder")
			}
		}()
		builder.WithRule("new-rule")
	})

	t.Run("WithSeverity after build panics", func(t *testing.T) {
		builder := NewViolationBuilder().
			WithRule("test").
			WithSeverity(NewSeverityContext().Warn()).
			WithEntity(NewEntityContext().Organism())

		_, _ = builder.Build()

		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic when modifying used builder")
			}
		}()
		builder.WithSeverity(NewSeverityContext().Log())
	})

	t.Run("WithMessage after build panics", func(t *testing.T) {
		builder := NewViolationBuilder().
			WithRule("test").
			WithSeverity(NewSeverityContext().Warn()).
			WithEntity(NewEntityContext().Organism())

		_, _ = builder.Build()

		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic when modifying used builder")
			}
		}()
		builder.WithMessage("new message")
	})

	t.Run("WithEntity after build panics", func(t *testing.T) {
		builder := NewViolationBuilder().
			WithRule("test").
			WithSeverity(NewSeverityContext().Warn()).
			WithEntity(NewEntityContext().Organism())

		_, _ = builder.Build()

		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic when modifying used builder")
			}
		}()
		builder.WithEntity(NewEntityContext().Protocol())
	})

	t.Run("WithEntityID after build panics", func(t *testing.T) {
		builder := NewViolationBuilder().
			WithRule("test").
			WithSeverity(NewSeverityContext().Warn()).
			WithEntity(NewEntityContext().Organism())

		_, _ = builder.Build()

		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic when modifying used builder")
			}
		}()
		builder.WithEntityID("new-id")
	})
}

func TestChangeBuilderPanicPaths(t *testing.T) {
	t.Run("WithEntity after build panics", func(t *testing.T) {
		builder := NewChangeBuilder().
			WithEntity(NewEntityContext().Organism()).
			WithAction(NewActionContext().Create())

		_, _ = builder.Build()

		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic when modifying used builder")
			}
		}()
		builder.WithEntity(NewEntityContext().Protocol())
	})

	t.Run("WithAction after build panics", func(t *testing.T) {
		builder := NewChangeBuilder().
			WithEntity(NewEntityContext().Organism()).
			WithAction(NewActionContext().Create())

		_, _ = builder.Build()

		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic when modifying used builder")
			}
		}()
		builder.WithAction(NewActionContext().Update())
	})

	t.Run("WithBefore after build panics", func(t *testing.T) {
		builder := NewChangeBuilder().
			WithEntity(NewEntityContext().Organism()).
			WithAction(NewActionContext().Create())

		_, _ = builder.Build()

		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic when modifying used builder")
			}
		}()
		builder.WithBefore(map[string]any{})
	})

	t.Run("WithAfter after build panics", func(t *testing.T) {
		builder := NewChangeBuilder().
			WithEntity(NewEntityContext().Organism()).
			WithAction(NewActionContext().Create())

		_, _ = builder.Build()

		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic when modifying used builder")
			}
		}()
		builder.WithAfter(map[string]any{})
	})
}

func TestResultBuilderPanicPaths(t *testing.T) {
	t.Run("AddViolations after build panics", func(t *testing.T) {
		builder := NewResultBuilder()
		_ = builder.Build()

		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic when modifying used builder")
			}
		}()

		violation, _ := NewViolationBuilder().
			WithRule("test").
			WithSeverity(NewSeverityContext().Warn()).
			WithEntity(NewEntityContext().Organism()).
			Build()

		builder.AddViolations(violation)
	})

	t.Run("MergeResult after build panics", func(t *testing.T) {
		builder := NewResultBuilder()
		_ = builder.Build()

		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic when modifying used builder")
			}
		}()

		builder.MergeResult(NewResult())
	})

	t.Run("FromBuilder after build panics", func(t *testing.T) {
		builder := NewResultBuilder()
		_ = builder.Build()

		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic when modifying used builder")
			}
		}()

		vb := NewViolationBuilder().
			WithRule("test").
			WithSeverity(NewSeverityContext().Warn()).
			WithEntity(NewEntityContext().Organism())

		builder.FromBuilder(vb)
	})
}

func TestConvertFunctions(t *testing.T) {
	t.Run("ConvertEntityType", func(t *testing.T) {
		converted := ConvertEntityType(entityOrganism)
		if converted.Value() != entityOrganism {
			t.Errorf("expected %s, got %s", entityOrganism, converted.Value())
		}
		if !converted.IsCore() {
			t.Error("organism should be core entity")
		}
	})

	t.Run("ConvertAction", func(t *testing.T) {
		converted := ConvertAction(actionUpdate)
		if converted.Value() != actionUpdate {
			t.Errorf("expected %s, got %s", actionUpdate, converted.Value())
		}
		if !converted.IsMutation() {
			t.Error("update should be a mutation")
		}
	})
}

func TestNewChange(t *testing.T) {
	t.Run("successful change creation", func(t *testing.T) {
		entityCtx := NewEntityContext()
		actionCtx := NewActionContext()

		before := map[string]any{"name": "old"}
		after := map[string]any{"name": "new"}

		change := NewChange(entityCtx.Organism(), actionCtx.Update(), before, after)

		if change.Entity() != entityOrganism {
			t.Errorf("expected entity %s, got %s", entityOrganism, change.Entity())
		}
		if change.Action() != actionUpdate {
			t.Errorf("expected action %s, got %s", actionUpdate, change.Action())
		}
	})
}
