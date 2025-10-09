package datasetapi

import "testing"

func TestBreedingAndCohortContexts(t *testing.T) {
	t.Run("breeding context provides all strategy types", func(t *testing.T) {
		ctx := NewBreedingContext()

		// Test all strategy type methods exist and return proper types
		natural := ctx.Natural()
		artificial := ctx.Artificial()
		controlled := ctx.Controlled()
		selective := ctx.Selective()

		if natural.String() != "natural" {
			t.Errorf("Expected natural strategy, got %s", natural.String())
		}

		if artificial.String() != "artificial" {
			t.Errorf("Expected artificial strategy, got %s", artificial.String())
		}

		if controlled.String() != "controlled" {
			t.Errorf("Expected controlled strategy, got %s", controlled.String())
		}

		if selective.String() != "selective" {
			t.Errorf("Expected selective strategy, got %s", selective.String())
		}
	})

	t.Run("breeding strategy contextual methods work correctly", func(t *testing.T) {
		ctx := NewBreedingContext()

		natural := ctx.Natural()
		artificial := ctx.Artificial()

		// Test IsNatural behavior
		if !natural.IsNatural() {
			t.Error("Natural strategy should return true for IsNatural()")
		}

		if artificial.IsNatural() {
			t.Error("Artificial strategy should return false for IsNatural()")
		}

		// Test RequiresIntervention behavior
		if natural.RequiresIntervention() {
			t.Error("Natural strategy should return false for RequiresIntervention()")
		}

		if !artificial.RequiresIntervention() {
			t.Error("Artificial strategy should return true for RequiresIntervention()")
		}
	})

	t.Run("cohort context provides all purpose types", func(t *testing.T) {
		ctx := NewCohortContext()

		// Test all purpose type methods exist and return proper types
		research := ctx.Research()
		breeding := ctx.Breeding()
		teaching := ctx.Teaching()
		conservation := ctx.Conservation()
		production := ctx.Production()

		if research.String() != "research" {
			t.Errorf("Expected research purpose, got %s", research.String())
		}

		if breeding.String() != "breeding" {
			t.Errorf("Expected breeding purpose, got %s", breeding.String())
		}

		if teaching.String() != "teaching" {
			t.Errorf("Expected teaching purpose, got %s", teaching.String())
		}

		if conservation.String() != "conservation" {
			t.Errorf("Expected conservation purpose, got %s", conservation.String())
		}

		if production.String() != "production" {
			t.Errorf("Expected production purpose, got %s", production.String())
		}
	})

	t.Run("cohort purpose contextual methods work correctly", func(t *testing.T) {
		ctx := NewCohortContext()

		research := ctx.Research()
		breeding := ctx.Breeding()
		teaching := ctx.Teaching()

		// Test IsResearch behavior
		if !research.IsResearch() {
			t.Error("Research purpose should return true for IsResearch()")
		}

		if breeding.IsResearch() {
			t.Error("Breeding purpose should return false for IsResearch()")
		}

		// Test RequiresProtocol behavior
		if !research.RequiresProtocol() {
			t.Error("Research purpose should return true for RequiresProtocol()")
		}

		if !teaching.RequiresProtocol() {
			t.Error("Teaching purpose should return true for RequiresProtocol()")
		}

		if breeding.RequiresProtocol() {
			t.Error("Breeding purpose should return false for RequiresProtocol()")
		}
	})

	t.Run("breeding strategy equality works", func(t *testing.T) {
		ctx := NewBreedingContext()
		natural1 := ctx.Natural()
		natural2 := ctx.Natural()
		artificial := ctx.Artificial()

		if !natural1.Equals(natural2) {
			t.Error("Two natural strategy refs should be equal")
		}

		if natural1.Equals(artificial) {
			t.Error("Natural and artificial strategy refs should not be equal")
		}
	})

	t.Run("cohort purpose equality works", func(t *testing.T) {
		ctx := NewCohortContext()
		research1 := ctx.Research()
		research2 := ctx.Research()
		breeding := ctx.Breeding()

		if !research1.Equals(research2) {
			t.Error("Two research purpose refs should be equal")
		}

		if research1.Equals(breeding) {
			t.Error("Research and breeding purpose refs should not be equal")
		}
	})
}
