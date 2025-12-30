package core

import (
	"colonycore/pkg/domain"
	entitymodel "colonycore/pkg/domain/entitymodel"
	"context"
	"testing"
)

func TestLineageIntegrityMissingParent(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore(NewRulesEngine())
	rule := LineageIntegrityRule()

	_, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		_, err := tx.CreateOrganism(domain.Organism{Organism: entitymodel.Organism{
			ID:        "child",
			Name:      "Child",
			Species:   "frog",
			Line:      "L1",
			Stage:     entitymodel.LifecycleStageJuvenile,
			ParentIDs: []string{"missing"},
		}})
		return err
	})
	if err != nil {
		t.Fatalf("create organism: %v", err)
	}

	_ = store.View(ctx, func(v domain.TransactionView) error {
		res, evalErr := rule.Evaluate(ctx, v, nil)
		if evalErr != nil {
			t.Fatalf("evaluate lineage rule: %v", evalErr)
		}
		if len(res.Violations) == 0 {
			t.Fatalf("expected lineage violation for missing parent")
		}
		return nil
	})
}

func TestLineageIntegrityBreedingSpeciesMismatch(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore(NewRulesEngine())
	rule := LineageIntegrityRule()

	var female, male domain.Organism
	_, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		f, err := tx.CreateOrganism(domain.Organism{Organism: entitymodel.Organism{
			ID:      "f",
			Name:    "F",
			Species: "frog",
			Line:    "L1",
			Stage:   entitymodel.LifecycleStageAdult,
		}})
		if err != nil {
			return err
		}
		m, err := tx.CreateOrganism(domain.Organism{Organism: entitymodel.Organism{
			ID:      "m",
			Name:    "M",
			Species: "mouse",
			Line:    "L1",
			Stage:   entitymodel.LifecycleStageAdult,
		}})
		if err != nil {
			return err
		}
		female, male = f, m
		return nil
	})
	if err != nil {
		t.Fatalf("create organisms: %v", err)
	}

	breeding := domain.BreedingUnit{BreedingUnit: entitymodel.BreedingUnit{
		ID:         "breeding-1",
		Name:       "Pair",
		Strategy:   "pair",
		FemaleIDs:  []string{female.ID},
		MaleIDs:    []string{male.ID},
		ProtocolID: nil,
	}}

	_ = store.View(ctx, func(v domain.TransactionView) error {
		res, evalErr := rule.Evaluate(ctx, v, []domain.Change{{Entity: domain.EntityBreeding, After: mustChangePayload(t, breeding)}})
		if evalErr != nil {
			t.Fatalf("evaluate lineage rule: %v", evalErr)
		}
		if len(res.Violations) == 0 {
			t.Fatalf("expected lineage violation for mixed species breeding unit")
		}
		return nil
	})
}

func TestLineageIntegrityParentSelfReference(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore(NewRulesEngine())
	rule := LineageIntegrityRule()

	_, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		_, err := tx.CreateOrganism(domain.Organism{Organism: entitymodel.Organism{
			ID:        "self",
			Name:      "Self",
			Species:   "frog",
			Line:      "L1",
			Stage:     entitymodel.LifecycleStageAdult,
			ParentIDs: []string{"self"},
		}})
		return err
	})
	if err != nil {
		t.Fatalf("create self organism: %v", err)
	}

	_ = store.View(ctx, func(v domain.TransactionView) error {
		res, evalErr := rule.Evaluate(ctx, v, nil)
		if evalErr != nil {
			t.Fatalf("evaluate lineage rule: %v", evalErr)
		}
		if len(res.Violations) == 0 {
			t.Fatalf("expected violation for self-referential parent")
		}
		return nil
	})
}

func TestLineageIntegrityParentLineMismatch(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore(NewRulesEngine())
	rule := LineageIntegrityRule()

	const (
		parentLine = "line-parent"
		childLine  = "line-child"
	)

	_, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		parent, err := tx.CreateOrganism(domain.Organism{Organism: entitymodel.Organism{
			ID:      "parent",
			Name:    "Parent",
			Species: "frog",
			Line:    parentLine,
			Stage:   entitymodel.LifecycleStageAdult,
			LineID:  stringPtr(parentLine),
		}})
		if err != nil {
			return err
		}
		_, err = tx.CreateOrganism(domain.Organism{Organism: entitymodel.Organism{
			ID:        "child",
			Name:      "Child",
			Species:   "frog",
			Line:      childLine,
			Stage:     entitymodel.LifecycleStageJuvenile,
			LineID:    stringPtr(childLine),
			ParentIDs: []string{parent.ID},
		}})
		return err
	})
	if err != nil {
		t.Fatalf("create organisms: %v", err)
	}

	_ = store.View(ctx, func(v domain.TransactionView) error {
		res, evalErr := rule.Evaluate(ctx, v, nil)
		if evalErr != nil {
			t.Fatalf("evaluate lineage rule: %v", evalErr)
		}
		if len(res.Violations) == 0 {
			t.Fatalf("expected violation for line mismatch")
		}
		return nil
	})
}

func TestLineageIntegrityBreedingLineAndStrainMismatch(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore(NewRulesEngine())
	rule := LineageIntegrityRule()

	var female domain.Organism
	_, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		f, err := tx.CreateOrganism(domain.Organism{Organism: entitymodel.Organism{
			ID:       "female-line",
			Name:     "Female",
			Species:  "frog",
			Line:     "line-a",
			Stage:    entitymodel.LifecycleStageAdult,
			LineID:   stringPtr("line-a"),
			StrainID: stringPtr("strain-a"),
		}})
		if err != nil {
			return err
		}
		female = f
		return nil
	})
	if err != nil {
		t.Fatalf("create female: %v", err)
	}

	breeding := domain.BreedingUnit{BreedingUnit: entitymodel.BreedingUnit{
		ID:        "breeding-line",
		Name:      "LineMismatch",
		Strategy:  "pair",
		LineID:    stringPtr("line-b"),
		StrainID:  stringPtr("strain-b"),
		FemaleIDs: []string{female.ID},
	}}

	_ = store.View(ctx, func(v domain.TransactionView) error {
		res, evalErr := rule.Evaluate(ctx, v, []domain.Change{{Entity: domain.EntityBreeding, After: mustChangePayload(t, breeding)}})
		if evalErr != nil {
			t.Fatalf("evaluate lineage rule: %v", evalErr)
		}
		if len(res.Violations) == 0 {
			t.Fatalf("expected violation for breeding line/strain mismatch")
		}
		return nil
	})
}

func TestLineageIntegrityBreedingDuplicateOrganism(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore(NewRulesEngine())
	rule := LineageIntegrityRule()

	var organism domain.Organism
	_, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		org, err := tx.CreateOrganism(domain.Organism{Organism: entitymodel.Organism{
			ID:      "duplicate",
			Name:    "Dup",
			Species: "frog",
			Line:    "line",
			Stage:   entitymodel.LifecycleStageAdult,
		}})
		if err != nil {
			return err
		}
		organism = org
		return nil
	})
	if err != nil {
		t.Fatalf("create organism: %v", err)
	}

	breeding := domain.BreedingUnit{BreedingUnit: entitymodel.BreedingUnit{
		ID:        "breeding-dup",
		Name:      "Dup",
		Strategy:  "pair",
		FemaleIDs: []string{organism.ID},
		MaleIDs:   []string{organism.ID},
	}}

	_ = store.View(ctx, func(v domain.TransactionView) error {
		res, evalErr := rule.Evaluate(ctx, v, []domain.Change{{Entity: domain.EntityBreeding, After: mustChangePayload(t, breeding)}})
		if evalErr != nil {
			t.Fatalf("evaluate lineage rule: %v", evalErr)
		}
		if len(res.Violations) == 0 {
			t.Fatalf("expected violation for duplicate organism use")
		}
		return nil
	})
}

func TestLineageIntegrityBreedingMissingOrganism(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore(NewRulesEngine())
	rule := LineageIntegrityRule()

	breeding := domain.BreedingUnit{BreedingUnit: entitymodel.BreedingUnit{
		ID:        "breeding-missing",
		Name:      "Missing",
		Strategy:  "colony",
		FemaleIDs: []string{"missing"},
	}}

	_ = store.View(ctx, func(v domain.TransactionView) error {
		res, evalErr := rule.Evaluate(ctx, v, []domain.Change{{Entity: domain.EntityBreeding, After: mustChangePayload(t, breeding)}})
		if evalErr != nil {
			t.Fatalf("evaluate lineage rule: %v", evalErr)
		}
		if len(res.Violations) == 0 {
			t.Fatalf("expected violation for missing breeding organism")
		}
		return nil
	})
}

func TestLineageIntegrityParentSpeciesMismatch(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore(NewRulesEngine())
	rule := LineageIntegrityRule()

	_, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		parent, err := tx.CreateOrganism(domain.Organism{Organism: entitymodel.Organism{
			ID:      "parent-species",
			Name:    "Parent",
			Species: "frog",
			Stage:   entitymodel.LifecycleStageAdult,
		}})
		if err != nil {
			return err
		}
		_, err = tx.CreateOrganism(domain.Organism{Organism: entitymodel.Organism{
			ID:        "child-species",
			Name:      "Child",
			Species:   "mouse",
			Stage:     entitymodel.LifecycleStageJuvenile,
			ParentIDs: []string{parent.ID},
		}})
		return err
	})
	if err != nil {
		t.Fatalf("create organisms: %v", err)
	}

	_ = store.View(ctx, func(v domain.TransactionView) error {
		res, evalErr := rule.Evaluate(ctx, v, nil)
		if evalErr != nil {
			t.Fatalf("evaluate lineage rule: %v", evalErr)
		}
		if len(res.Violations) == 0 {
			t.Fatalf("expected species mismatch violation")
		}
		return nil
	})
}

func TestLineageIntegrityParentDuplicateReference(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore(NewRulesEngine())
	rule := LineageIntegrityRule()

	_, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		parent, err := tx.CreateOrganism(domain.Organism{Organism: entitymodel.Organism{
			ID:      "parent-dup",
			Name:    "Parent",
			Species: "frog",
			Stage:   entitymodel.LifecycleStageAdult,
		}})
		if err != nil {
			return err
		}
		_, err = tx.CreateOrganism(domain.Organism{Organism: entitymodel.Organism{
			ID:        "child-dup",
			Name:      "Child",
			Species:   "frog",
			Stage:     entitymodel.LifecycleStageJuvenile,
			ParentIDs: []string{parent.ID, parent.ID},
		}})
		return err
	})
	if err != nil {
		t.Fatalf("create organisms: %v", err)
	}

	_ = store.View(ctx, func(v domain.TransactionView) error {
		res, evalErr := rule.Evaluate(ctx, v, nil)
		if evalErr != nil {
			t.Fatalf("evaluate lineage rule: %v", evalErr)
		}
		if len(res.Violations) == 0 {
			t.Fatalf("expected duplicate parent violation")
		}
		return nil
	})
}

func TestLineageIntegrityParentStrainMismatch(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore(NewRulesEngine())
	rule := LineageIntegrityRule()

	_, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		parentStrain := "strain-parent"
		childStrain := "strain-child"
		parent, err := tx.CreateOrganism(domain.Organism{Organism: entitymodel.Organism{
			ID:       "parent-strain",
			Name:     "Parent",
			Species:  "frog",
			Stage:    entitymodel.LifecycleStageAdult,
			StrainID: stringPtr(parentStrain),
		}})
		if err != nil {
			return err
		}
		_, err = tx.CreateOrganism(domain.Organism{Organism: entitymodel.Organism{
			ID:        "child-strain",
			Name:      "Child",
			Species:   "frog",
			Stage:     entitymodel.LifecycleStageJuvenile,
			StrainID:  stringPtr(childStrain),
			ParentIDs: []string{parent.ID},
		}})
		return err
	})
	if err != nil {
		t.Fatalf("create organisms: %v", err)
	}

	_ = store.View(ctx, func(v domain.TransactionView) error {
		res, evalErr := rule.Evaluate(ctx, v, nil)
		if evalErr != nil {
			t.Fatalf("evaluate lineage rule: %v", evalErr)
		}
		if len(res.Violations) == 0 {
			t.Fatalf("expected strain mismatch violation")
		}
		return nil
	})
}

func stringPtr(v string) *string {
	return &v
}

func TestLineageIntegrityRuleName(t *testing.T) {
	if got := LineageIntegrityRule().Name(); got != "lineage_integrity" {
		t.Fatalf("unexpected rule name: %s", got)
	}
}
