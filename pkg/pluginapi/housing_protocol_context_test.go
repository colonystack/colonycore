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

	t.Run("housing state context provides lifecycle states", func(t *testing.T) {
		ctx := NewHousingStateContext()

		quarantine := ctx.Quarantine()
		active := ctx.Active()
		cleaning := ctx.Cleaning()
		decommissioned := ctx.Decommissioned()

		if quarantine.String() != "quarantine" {
			t.Errorf("Expected quarantine state, got %s", quarantine.String())
		}
		if active.String() != "active" {
			t.Errorf("Expected active state, got %s", active.String())
		}
		if cleaning.String() != "cleaning" {
			t.Errorf("Expected cleaning state, got %s", cleaning.String())
		}
		if decommissioned.String() != "decommissioned" {
			t.Errorf("Expected decommissioned state, got %s", decommissioned.String())
		}
	})

	t.Run("housing state contextual methods work correctly", func(t *testing.T) {
		ctx := NewHousingStateContext()

		active := ctx.Active()
		decommissioned := ctx.Decommissioned()

		if !active.IsActive() {
			t.Error("Active state should return true for IsActive()")
		}
		if active.IsDecommissioned() {
			t.Error("Active state should not be decommissioned")
		}
		if decommissioned.IsActive() {
			t.Error("Decommissioned state should not be active")
		}
		if !decommissioned.IsDecommissioned() {
			t.Error("Decommissioned state should be flagged terminal")
		}
	})
}

func TestProtocolContext(t *testing.T) {
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

		if approved.String() != "approved" {
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
		if draft.IsActive() {
			t.Error("Draft status should return false for IsActive()")
		}

		if !approved.IsActive() {
			t.Error("Approved status should return true for IsActive()")
		}

		if expired.IsActive() {
			t.Error("Expired status should return false for IsActive()")
		}

		// Test IsTerminal behavior
		if draft.IsTerminal() {
			t.Error("Draft status should return false for IsTerminal()")
		}

		if approved.IsTerminal() {
			t.Error("Approved status should return false for IsTerminal()")
		}

		if !expired.IsTerminal() {
			t.Error("Expired status should return true for IsTerminal()")
		}

		if !archived.IsTerminal() {
			t.Error("Archived status should return true for IsTerminal()")
		}
	})

	t.Run("protocol status equality works", func(t *testing.T) {
		ctx := NewProtocolContext()

		approved1 := ctx.Approved()
		approved2 := ctx.Approved()
		draft := ctx.Draft()

		if !approved1.Equals(approved2) {
			t.Error("Two approved references should be equal")
		}

		if approved1.Equals(draft) {
			t.Error("Approved and draft references should not be equal")
		}
	})
}

func TestPermitContext(t *testing.T) {
	t.Run("permit context provides all status types", func(t *testing.T) {
		statuses := NewPermitContext().Statuses()

		draft := statuses.Draft()
		submitted := statuses.Submitted()
		approved := statuses.Approved()
		onHold := statuses.OnHold()
		expired := statuses.Expired()
		archived := statuses.Archived()

		if draft.String() != "draft" {
			t.Errorf("Expected draft status, got %s", draft.String())
		}
		if submitted.String() != "submitted" {
			t.Errorf("Expected submitted status, got %s", submitted.String())
		}
		if approved.String() != "approved" {
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
		approved1 := ctx.Approved()
		approved2 := ctx.Approved()
		expired := ctx.Expired()

		// Test Equals method
		if !approved1.Equals(approved2) {
			t.Error("Two approved status refs should be equal")
		}

		if approved1.Equals(expired) {
			t.Error("Approved and expired status refs should not be equal")
		}

		// Test marker method exists (type safety)
		var _ = approved1
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
