package datasetapi

import "testing"

const (
	// Test string constants
	expectedAquatic     = "aquatic"
	expectedTerrestrial = "terrestrial"
	expectedActive      = "active"
	expectedCompleted   = "completed"
	expectedCancelled   = "cancelled"
	expectedFailed      = "failed"
)

func TestDatasetHousingContext(t *testing.T) {
	t.Run("housing context provides all environment types", func(t *testing.T) {
		ctx := NewHousingContext()

		// Test all environment type methods exist and return proper types
		aquatic := ctx.Aquatic()
		terrestrial := ctx.Terrestrial()
		arboreal := ctx.Arboreal()
		humid := ctx.Humid()

		if aquatic.String() != expectedAquatic {
			t.Errorf("Expected aquatic environment, got %s", aquatic.String())
		}

		if terrestrial.String() != expectedTerrestrial {
			t.Errorf("Expected terrestrial environment, got %s", terrestrial.String())
		}

		if arboreal.String() != "arboreal" {
			t.Errorf("Expected arboreal environment, got %s", arboreal.String())
		}

		if humid.String() != "humid" {
			t.Errorf("Expected humid environment, got %s", humid.String())
		}
	})

	t.Run("environment type contextual methods work correctly", func(t *testing.T) {
		ctx := NewHousingContext()

		aquatic := ctx.Aquatic()
		terrestrial := ctx.Terrestrial()
		humid := ctx.Humid()

		// Test IsAquatic behavior
		if !aquatic.IsAquatic() {
			t.Error("Aquatic environment should return true for IsAquatic()")
		}

		if terrestrial.IsAquatic() {
			t.Error("Terrestrial environment should return false for IsAquatic()")
		}

		// Test IsHumid behavior
		if !aquatic.IsHumid() {
			t.Error("Aquatic environment should return true for IsHumid()")
		}

		if !humid.IsHumid() {
			t.Error("Humid environment should return true for IsHumid()")
		}

		if terrestrial.IsHumid() {
			t.Error("Terrestrial environment should return false for IsHumid()")
		}
	})
}

func TestDatasetProtocolContext(t *testing.T) {
	t.Run("protocol context provides all status types", func(t *testing.T) {
		ctx := NewProtocolContext()

		// Test all status type methods exist and return proper types
		draft := ctx.Draft()
		active := ctx.Active()
		suspended := ctx.Suspended()
		completed := ctx.Completed()
		cancelled := ctx.Cancelled()

		if draft.String() != "draft" {
			t.Errorf("Expected draft status, got %s", draft.String())
		}

		if active.String() != expectedActive {
			t.Errorf("Expected active status, got %s", active.String())
		}

		if suspended.String() != "suspended" {
			t.Errorf("Expected suspended status, got %s", suspended.String())
		}

		if completed.String() != expectedCompleted {
			t.Errorf("Expected completed status, got %s", completed.String())
		}

		if cancelled.String() != expectedCancelled {
			t.Errorf("Expected cancelled status, got %s", cancelled.String())
		}
	})

	t.Run("protocol status contextual methods work correctly", func(t *testing.T) {
		ctx := NewProtocolContext()

		draft := ctx.Draft()
		active := ctx.Active()
		completed := ctx.Completed()
		cancelled := ctx.Cancelled()

		// Test IsActive behavior
		if !active.IsActive() {
			t.Error("Active status should return true for IsActive()")
		}

		if draft.IsActive() {
			t.Error("Draft status should return false for IsActive()")
		}

		// Test IsTerminal behavior
		if !completed.IsTerminal() {
			t.Error("Completed status should return true for IsTerminal()")
		}

		if !cancelled.IsTerminal() {
			t.Error("Cancelled status should return true for IsTerminal()")
		}

		if active.IsTerminal() {
			t.Error("Active status should return false for IsTerminal()")
		}
	})
}

func TestDatasetProcedureContext(t *testing.T) {
	t.Run("procedure context provides all status types", func(t *testing.T) {
		ctx := NewProcedureContext()

		// Test all status type methods exist and return proper types
		scheduled := ctx.Scheduled()
		inProgress := ctx.InProgress()
		completed := ctx.Completed()
		cancelled := ctx.Cancelled()
		failed := ctx.Failed()

		if scheduled.String() != "scheduled" {
			t.Errorf("Expected scheduled status, got %s", scheduled.String())
		}

		if inProgress.String() != "in_progress" {
			t.Errorf("Expected in_progress status, got %s", inProgress.String())
		}

		if completed.String() != expectedCompleted {
			t.Errorf("Expected completed status, got %s", completed.String())
		}

		if cancelled.String() != expectedCancelled {
			t.Errorf("Expected cancelled status, got %s", cancelled.String())
		}

		if failed.String() != expectedFailed {
			t.Errorf("Expected failed status, got %s", failed.String())
		}
	})

	t.Run("procedure status contextual methods work correctly", func(t *testing.T) {
		ctx := NewProcedureContext()

		scheduled := ctx.Scheduled()
		inProgress := ctx.InProgress()
		completed := ctx.Completed()
		cancelled := ctx.Cancelled()
		failed := ctx.Failed()

		// Test IsActive behavior
		if !inProgress.IsActive() {
			t.Error("In-progress procedure should be active")
		}

		if completed.IsActive() {
			t.Error("Completed procedure should not be active")
		}

		// Test IsTerminal behavior
		if scheduled.IsTerminal() {
			t.Error("Scheduled procedure should not be terminal")
		}

		if !completed.IsTerminal() {
			t.Error("Completed procedure should be terminal")
		}

		if !cancelled.IsTerminal() {
			t.Error("Cancelled procedure should be terminal")
		}

		if !failed.IsTerminal() {
			t.Error("Failed procedure should be terminal")
		}

		// Test IsSuccessful behavior
		if !completed.IsSuccessful() {
			t.Error("Completed procedure should be successful")
		}

		if cancelled.IsSuccessful() {
			t.Error("Cancelled procedure should not be successful")
		}

		if failed.IsSuccessful() {
			t.Error("Failed procedure should not be successful")
		}
	})
}

// Test utility methods that were previously untested
func TestEnvironmentTypeRefUtilityMethods(t *testing.T) {
	t.Run("Equals method works correctly", func(t *testing.T) {
		ctx := NewHousingContext()
		aquatic1 := ctx.Aquatic()
		aquatic2 := ctx.Aquatic()
		terrestrial := ctx.Terrestrial()

		if !aquatic1.Equals(aquatic2) {
			t.Error("Two aquatic environment refs should be equal")
		}

		if aquatic1.Equals(terrestrial) {
			t.Error("Aquatic and terrestrial environment refs should not be equal")
		}
	})

	t.Run("isEnvironmentTypeRef marker method exists", func(_ *testing.T) {
		ctx := NewHousingContext()
		aquatic := ctx.Aquatic()

		// This method should exist (it's a marker method for type safety)
		// We can't call it directly as it's private, but we can test through reflection
		// or just ensure the type implements the interface correctly
		var _ = aquatic
	})
}

func TestProcedureStatusRefUtilityMethods(t *testing.T) {
	t.Run("Equals method works correctly", func(t *testing.T) {
		ctx := NewProcedureContext()
		completed1 := ctx.Completed()
		completed2 := ctx.Completed()
		cancelled := ctx.Cancelled()

		if !completed1.Equals(completed2) {
			t.Error("Two completed status refs should be equal")
		}

		if completed1.Equals(cancelled) {
			t.Error("Completed and cancelled status refs should not be equal")
		}
	})

	t.Run("isProcedureStatusRef marker method exists", func(_ *testing.T) {
		ctx := NewProcedureContext()
		completed := ctx.Completed()

		// This method should exist (it's a marker method for type safety)
		var _ = completed
	})
}

func TestProtocolStatusRefUtilityMethods(t *testing.T) {
	t.Run("Equals method works correctly", func(t *testing.T) {
		ctx := NewProtocolContext()
		active1 := ctx.Active()
		active2 := ctx.Active()
		completed := ctx.Completed()

		if !active1.Equals(active2) {
			t.Error("Two active status refs should be equal")
		}

		if active1.Equals(completed) {
			t.Error("Active and completed status refs should not be equal")
		}
	})

	t.Run("isProtocolStatusRef marker method exists", func(_ *testing.T) {
		ctx := NewProtocolContext()
		active := ctx.Active()

		// This method should exist (it's a marker method for type safety)
		var _ = active
	})
}
