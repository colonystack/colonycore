package pluginapi

import "testing"

const expectedAquatic = "aquatic"

func TestHousingContext(t *testing.T) {
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

		if terrestrial.String() != "terrestrial" {
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

	t.Run("environment type equality works", func(t *testing.T) {
		ctx := NewHousingContext()

		aquatic1 := ctx.Aquatic()
		aquatic2 := ctx.Aquatic()
		terrestrial := ctx.Terrestrial()

		if !aquatic1.Equals(aquatic2) {
			t.Error("Two aquatic references should be equal")
		}

		if aquatic1.Equals(terrestrial) {
			t.Error("Aquatic and terrestrial references should not be equal")
		}
	})
}

func TestProtocolContext(t *testing.T) {
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

		if active.String() != "active" {
			t.Errorf("Expected active status, got %s", active.String())
		}

		if suspended.String() != "suspended" {
			t.Errorf("Expected suspended status, got %s", suspended.String())
		}

		if completed.String() != "completed" {
			t.Errorf("Expected completed status, got %s", completed.String())
		}

		if cancelled.String() != "cancelled" {
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
		if draft.IsActive() {
			t.Error("Draft status should return false for IsActive()")
		}

		if !active.IsActive() {
			t.Error("Active status should return true for IsActive()")
		}

		if completed.IsActive() {
			t.Error("Completed status should return false for IsActive()")
		}

		// Test IsTerminal behavior
		if draft.IsTerminal() {
			t.Error("Draft status should return false for IsTerminal()")
		}

		if active.IsTerminal() {
			t.Error("Active status should return false for IsTerminal()")
		}

		if !completed.IsTerminal() {
			t.Error("Completed status should return true for IsTerminal()")
		}

		if !cancelled.IsTerminal() {
			t.Error("Cancelled status should return true for IsTerminal()")
		}
	})

	t.Run("protocol status equality works", func(t *testing.T) {
		ctx := NewProtocolContext()

		active1 := ctx.Active()
		active2 := ctx.Active()
		draft := ctx.Draft()

		if !active1.Equals(active2) {
			t.Error("Two active references should be equal")
		}

		if active1.Equals(draft) {
			t.Error("Active and draft references should not be equal")
		}
	})
}

// Test utility methods that were previously untested
func TestPluginAPIContextUtilityMethods(t *testing.T) {
	t.Run("EnvironmentTypeRef utility methods", func(t *testing.T) {
		ctx := NewHousingContext()
		aquatic1 := ctx.Aquatic()
		aquatic2 := ctx.Aquatic()
		terrestrial := ctx.Terrestrial()

		// Test Equals method
		if !aquatic1.Equals(aquatic2) {
			t.Error("Two aquatic environment refs should be equal")
		}

		if aquatic1.Equals(terrestrial) {
			t.Error("Aquatic and terrestrial environment refs should not be equal")
		}

		// Test marker method exists (type safety)
		var _ = aquatic1
	})

	t.Run("ProtocolStatusRef utility methods", func(t *testing.T) {
		ctx := NewProtocolContext()
		active1 := ctx.Active()
		active2 := ctx.Active()
		completed := ctx.Completed()

		// Test Equals method
		if !active1.Equals(active2) {
			t.Error("Two active status refs should be equal")
		}

		if active1.Equals(completed) {
			t.Error("Active and completed status refs should not be equal")
		}

		// Test marker method exists (type safety)
		var _ = active1
	})

	t.Run("ActionRef utility methods", func(t *testing.T) {
		ctx := NewActionContext()
		create1 := ctx.Create()
		create2 := ctx.Create()
		update := ctx.Update()

		// Test Equals method
		if !create1.Equals(create2) {
			t.Error("Two create action refs should be equal")
		}

		if create1.Equals(update) {
			t.Error("Create and update action refs should not be equal")
		}

		// Test marker method exists (type safety)
		var _ = create1
	})

	t.Run("EntityTypeRef utility methods", func(t *testing.T) {
		ctx := NewEntityContext()
		organism1 := ctx.Organism()
		organism2 := ctx.Organism()
		housing := ctx.Housing()

		// Test Equals method
		if !organism1.Equals(organism2) {
			t.Error("Two organism entity refs should be equal")
		}

		if organism1.Equals(housing) {
			t.Error("Organism and housing entity refs should not be equal")
		}

		// Test marker method exists (type safety)
		var _ = organism1
	})

	t.Run("SeverityRef utility methods", func(t *testing.T) {
		ctx := NewSeverityContext()
		warn1 := ctx.Warn()
		warn2 := ctx.Warn()
		block := ctx.Block()

		// Test Equals method
		if !warn1.Equals(warn2) {
			t.Error("Two warn severity refs should be equal")
		}

		if warn1.Equals(block) {
			t.Error("Warn and block severity refs should not be equal")
		}

		// Test marker method exists (type safety)
		var _ = warn1
	})

	t.Run("LifecycleStageRef utility methods", func(t *testing.T) {
		ctx := NewLifecycleStageContext()
		adult1 := ctx.Adult()
		adult2 := ctx.Adult()
		juvenile := ctx.Juvenile()

		// Test Equals method
		if !adult1.Equals(adult2) {
			t.Error("Two adult lifecycle refs should be equal")
		}

		if adult1.Equals(juvenile) {
			t.Error("Adult and juvenile lifecycle refs should not be equal")
		}

		// Test marker method exists (type safety)
		var _ = adult1
	})
}
