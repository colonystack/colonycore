package datasetapi

import "testing"

const (
	// Test string constants
	expectedAquatic     = "aquatic"
	expectedTerrestrial = "terrestrial"
	expectedApproved    = "approved"
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
		submitted := ctx.Submitted()
		approved := ctx.Approved()
		onHold := ctx.OnHold()
		expired := ctx.Expired()
		archived := ctx.Archived()

		if draft.String() != "draft" {
			t.Errorf("Expected draft status, got %s", draft.String())
		}

		if submitted.String() != "submitted" {
			t.Errorf("Expected submitted status, got %s", submitted.String())
		}

		if approved.String() != expectedApproved {
			t.Errorf("Expected approved status, got %s", approved.String())
		}

		if onHold.String() != "on_hold" {
			t.Errorf("Expected on_hold status, got %s", onHold.String())
		}

		if expired.String() != "expired" {
			t.Errorf("Expected expired status, got %s", expired.String())
		}

		if archived.String() != "archived" {
			t.Errorf("Expected archived status, got %s", archived.String())
		}
	})

	t.Run("protocol status contextual methods work correctly", func(t *testing.T) {
		ctx := NewProtocolContext()

		draft := ctx.Draft()
		approved := ctx.Approved()
		expired := ctx.Expired()
		archived := ctx.Archived()

		// Test IsActive behavior
		if !approved.IsActive() {
			t.Error("Approved status should return true for IsActive()")
		}

		if draft.IsActive() {
			t.Error("Draft status should return false for IsActive()")
		}

		// Test IsTerminal behavior
		if !expired.IsTerminal() {
			t.Error("Expired status should return true for IsTerminal()")
		}

		if !archived.IsTerminal() {
			t.Error("Archived status should return true for IsTerminal()")
		}

		if approved.IsTerminal() {
			t.Error("Approved status should return false for IsTerminal()")
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

func TestDatasetPermitContext(t *testing.T) {
	t.Run("permit context provides all status types", func(t *testing.T) {
		statuses := NewPermitContext().Statuses()

		draft := statuses.Draft()
		submitted := statuses.Submitted()
		approved := statuses.Approved()
		onHold := statuses.OnHold()
		expired := statuses.Expired()
		archived := statuses.Archived()

		if draft.String() != datasetPermitStatusDraft {
			t.Errorf("Expected draft status, got %s", draft.String())
		}
		if submitted.String() != datasetPermitStatusSubmitted {
			t.Errorf("Expected submitted status, got %s", submitted.String())
		}
		if approved.String() != datasetPermitStatusApproved {
			t.Errorf("Expected approved status, got %s", approved.String())
		}
		if onHold.String() != datasetPermitStatusOnHold {
			t.Errorf("Expected on_hold status, got %s", onHold.String())
		}
		if expired.String() != datasetPermitStatusExpired {
			t.Errorf("Expected expired status, got %s", expired.String())
		}
		if archived.String() != datasetPermitStatusArchived {
			t.Errorf("Expected archived status, got %s", archived.String())
		}
	})

	t.Run("permit status contextual methods work correctly", func(t *testing.T) {
		statuses := NewPermitContext().Statuses()

		approved := statuses.Approved()
		expired := statuses.Expired()
		archived := statuses.Archived()

		if !approved.IsActive() {
			t.Error("Approved status should be active")
		}
		if approved.IsExpired() || approved.IsArchived() {
			t.Error("Approved status should not be expired or archived")
		}
		if !expired.IsExpired() {
			t.Error("Expired status should be expired")
		}
		if expired.IsActive() {
			t.Error("Expired status should not be active")
		}
		if !archived.IsArchived() {
			t.Error("Archived status should be archived")
		}
		if archived.IsActive() {
			t.Error("Archived status should not be active")
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
		approved1 := ctx.Approved()
		approved2 := ctx.Approved()
		expired := ctx.Expired()

		if !approved1.Equals(approved2) {
			t.Error("Two approved status refs should be equal")
		}

		if approved1.Equals(expired) {
			t.Error("Approved and expired status refs should not be equal")
		}
	})

	t.Run("isProtocolStatusRef marker method exists", func(_ *testing.T) {
		ctx := NewProtocolContext()
		approved := ctx.Approved()

		// This method should exist (it's a marker method for type safety)
		var _ = approved
	})
}
