package core

import (
	"context"
	"fmt"
	"testing"
	"time"

	"colonycore/pkg/domain"
)

func TestMemoryStoreCRUDAndQueries(t *testing.T) {
	store := NewMemoryStore(nil)
	ctx := context.Background()

	var (
		projectID   string
		protocolID  string
		housingID   string
		cohortID    string
		breedingID  string
		procedureID string
		organismAID string
		organismBID string
	)

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		if _, err := tx.CreateHousingUnit(domain.HousingUnit{Name: "Invalid", Facility: "Lab", Capacity: 0}); err == nil {
			return fmt.Errorf("expected capacity validation error")
		}

		project, err := tx.CreateProject(domain.Project{Code: "PRJ-1", Title: "Project"})
		if err != nil {
			return err
		}
		projectID = project.ID

		protocol, err := tx.CreateProtocol(domain.Protocol{Code: "PROT-1", Title: "Protocol", MaxSubjects: 5})
		if err != nil {
			return err
		}
		protocolID = protocol.ID

		housing, err := tx.CreateHousingUnit(domain.HousingUnit{Name: "Tank", Facility: "Lab", Capacity: 2, Environment: "arid"})
		if err != nil {
			return err
		}
		housingID = housing.ID

		projectPtr := projectID
		housingPtr := housingID
		protocolPtr := protocolID

		cohort, err := tx.CreateCohort(domain.Cohort{Name: "Cohort", Purpose: "Observation", ProjectID: &projectPtr, HousingID: &housingPtr, ProtocolID: &protocolPtr})
		if err != nil {
			return err
		}
		cohortID = cohort.ID

		cohortPtr := cohortID

		attrs := map[string]any{"skin_color_index": 5}
		organismA, err := tx.CreateOrganism(domain.Organism{
			Name:       "Alpha",
			Species:    "Test Frog",
			Stage:      domain.StageJuvenile,
			ProjectID:  &projectPtr,
			ProtocolID: &protocolPtr,
			CohortID:   &cohortPtr,
			HousingID:  &housingPtr,
			Attributes: attrs,
		})
		if err != nil {
			return err
		}
		organismAID = organismA.ID

		attrs["skin_color_index"] = 9

		organismB, err := tx.CreateOrganism(domain.Organism{
			Name:     "Beta",
			Species:  "Test Toad",
			Stage:    domain.StageAdult,
			CohortID: &cohortPtr,
		})
		if err != nil {
			return err
		}
		organismBID = organismB.ID

		if _, err := tx.CreateOrganism(domain.Organism{Base: domain.Base{ID: organismAID}, Name: "Duplicate"}); err == nil {
			return fmt.Errorf("expected duplicate organism error")
		}

		breeding, err := tx.CreateBreedingUnit(domain.BreedingUnit{
			Name:       "Pair",
			Strategy:   "pair",
			HousingID:  &housingPtr,
			ProtocolID: &protocolPtr,
			FemaleIDs:  []string{organismAID},
			MaleIDs:    []string{organismBID},
		})
		if err != nil {
			return err
		}
		breedingID = breeding.ID

		procedure, err := tx.CreateProcedure(domain.Procedure{
			Name:        "Check",
			Status:      "scheduled",
			ScheduledAt: time.Now().Add(time.Minute),
			ProtocolID:  protocolID,
			OrganismIDs: []string{organismAID, organismBID},
		})
		if err != nil {
			return err
		}
		procedureID = procedure.ID

		view := tx.Snapshot()
		if got := len(view.ListOrganisms()); got != 2 {
			return fmt.Errorf("expected 2 organisms in view, got %d", got)
		}
		if _, ok := view.FindOrganism("missing"); ok {
			return fmt.Errorf("unexpected organism lookup success")
		}
		if _, ok := view.FindHousingUnit("missing"); ok {
			return fmt.Errorf("unexpected housing lookup success")
		}
		if got := len(view.ListProtocols()); got != 1 {
			return fmt.Errorf("expected 1 protocol in view, got %d", got)
		}
		return nil
	}); err != nil {
		t.Fatalf("create transaction: %v", err)
	}

	organisms := store.ListOrganisms()
	if len(organisms) != 2 {
		t.Fatalf("expected 2 organisms, got %d", len(organisms))
	}
	var copyCheckDone bool
	for _, organism := range organisms {
		if organism.ID != organismAID {
			continue
		}
		if organism.Attributes["skin_color_index"].(int) != 5 {
			t.Fatalf("expected cloned attributes value 5, got %v", organism.Attributes["skin_color_index"])
		}
		organism.Attributes["skin_color_index"] = 1
		copyCheckDone = true
	}
	if !copyCheckDone {
		t.Fatalf("organism %s not found in list", organismAID)
	}
	if refreshed, ok := store.GetOrganism(organismAID); !ok {
		t.Fatalf("expected organism %s to exist", organismAID)
	} else if refreshed.Attributes["skin_color_index"].(int) != 5 {
		t.Fatalf("expected store attributes to remain 5, got %v", refreshed.Attributes["skin_color_index"])
	}

	housingList := store.ListHousingUnits()
	if len(housingList) != 1 {
		t.Fatalf("expected 1 housing unit, got %d", len(housingList))
	}
	housingList[0].Environment = "modified"
	if stored, ok := store.GetHousingUnit(housingID); !ok {
		t.Fatalf("expected housing unit %s to exist", housingID)
	} else if stored.Environment != "arid" {
		t.Fatalf("expected environment to remain arid, got %s", stored.Environment)
	}

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		if _, err := tx.UpdateOrganism("missing", func(*domain.Organism) error { return nil }); err == nil {
			return fmt.Errorf("expected update error for missing organism")
		}
		if _, err := tx.UpdateHousingUnit(housingID, func(h *domain.HousingUnit) error {
			h.Capacity = 0
			return nil
		}); err == nil {
			return fmt.Errorf("expected housing capacity validation on update")
		}
		if _, err := tx.UpdateHousingUnit("missing", func(*domain.HousingUnit) error { return nil }); err == nil {
			return fmt.Errorf("expected missing housing update error")
		}
		const updatedDesc = "updated"
		if _, err := tx.UpdateProject(projectID, func(p *domain.Project) error {
			p.Description = updatedDesc
			return nil
		}); err != nil {
			return err
		}
		if _, err := tx.UpdateProtocol(protocolID, func(p *domain.Protocol) error {
			p.Description = updatedDesc
			return nil
		}); err != nil {
			return err
		}
		if _, err := tx.UpdateCohort(cohortID, func(c *domain.Cohort) error {
			c.Purpose = updatedDesc
			return nil
		}); err != nil {
			return err
		}
		if _, err := tx.UpdateBreedingUnit(breedingID, func(b *domain.BreedingUnit) error {
			b.Strategy = updatedDesc
			b.FemaleIDs = append(b.FemaleIDs, organismBID)
			return nil
		}); err != nil {
			return err
		}
		if _, err := tx.UpdateProcedure(procedureID, func(p *domain.Procedure) error {
			p.Status = "completed"
			return nil
		}); err != nil {
			return err
		}
		if _, err := tx.UpdateOrganism(organismBID, func(o *domain.Organism) error {
			o.Stage = domain.StageRetired
			return nil
		}); err != nil {
			return err
		}
		return nil
	}); err != nil {
		t.Fatalf("update transaction: %v", err)
	}

	if updated, ok := store.GetOrganism(organismBID); !ok {
		t.Fatalf("expected organism %s", organismBID)
	} else if updated.Stage != domain.StageRetired {
		t.Fatalf("expected stage to be retired, got %s", updated.Stage)
	}

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		if err := tx.DeleteProcedure(procedureID); err != nil {
			return err
		}
		if err := tx.DeleteBreedingUnit(breedingID); err != nil {
			return err
		}
		if err := tx.DeleteOrganism(organismAID); err != nil {
			return err
		}
		if err := tx.DeleteOrganism(organismBID); err != nil {
			return err
		}
		if err := tx.DeleteCohort(cohortID); err != nil {
			return err
		}
		if err := tx.DeleteHousingUnit(housingID); err != nil {
			return err
		}
		if err := tx.DeleteProtocol(protocolID); err != nil {
			return err
		}
		if err := tx.DeleteProject(projectID); err != nil {
			return err
		}
		if err := tx.DeleteOrganism(organismAID); err == nil {
			return fmt.Errorf("expected delete error for missing organism")
		}
		return nil
	}); err != nil {
		t.Fatalf("delete transaction: %v", err)
	}

	if len(store.ListOrganisms()) != 0 {
		t.Fatalf("expected no organisms after deletion")
	}
	if len(store.ListCohorts()) != 0 {
		t.Fatalf("expected no cohorts after deletion")
	}
	if len(store.ListHousingUnits()) != 0 {
		t.Fatalf("expected no housing units after deletion")
	}
	if len(store.ListProtocols()) != 0 {
		t.Fatalf("expected no protocols after deletion")
	}
	if len(store.ListProjects()) != 0 {
		t.Fatalf("expected no projects after deletion")
	}
	if len(store.ListBreedingUnits()) != 0 {
		t.Fatalf("expected no breeding units after deletion")
	}
	if len(store.ListProcedures()) != 0 {
		t.Fatalf("expected no procedures after deletion")
	}
}

func TestMemoryStoreViewReadOnly(t *testing.T) {
	store := NewMemoryStore(nil)
	ctx := context.Background()
	var housing domain.HousingUnit
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		var err error
		housing, err = tx.CreateHousingUnit(domain.HousingUnit{Name: "Tank", Facility: "Lab", Capacity: 1})
		return err
	}); err != nil {
		t.Fatalf("create housing: %v", err)
	}

	if err := store.View(ctx, func(view domain.TransactionView) error {
		units := view.ListHousingUnits()
		if len(units) != 1 {
			t.Fatalf("expected single housing unit, got %d", len(units))
		}
		if _, ok := view.FindHousingUnit(housing.ID); !ok {
			t.Fatalf("expected to find housing unit %s", housing.ID)
		}
		return nil
	}); err != nil {
		t.Fatalf("view snapshot: %v", err)
	}
}

func TestUpdateHousingUnitValidation(t *testing.T) {
	store := NewMemoryStore(nil)
	ctx := context.Background()
	var housing domain.HousingUnit
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		var err error
		housing, err = tx.CreateHousingUnit(domain.HousingUnit{Name: "Validated", Facility: "Lab", Capacity: 2})
		return err
	}); err != nil {
		t.Fatalf("create housing: %v", err)
	}

	_, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		_, err := tx.UpdateHousingUnit(housing.ID, func(h *domain.HousingUnit) error {
			h.Capacity = 0
			return nil
		})
		return err
	})
	if err == nil {
		t.Fatalf("expected capacity validation error on update")
	}
}

func TestResultMergeAndBlocking(t *testing.T) {
	res := domain.Result{}
	res.Merge(domain.Result{Violations: []domain.Violation{{Rule: "warn", Severity: domain.SeverityWarn}}})
	if res.HasBlocking() {
		t.Fatalf("expected no blocking violations yet")
	}
	res.Merge(domain.Result{Violations: []domain.Violation{{Rule: "block", Severity: domain.SeverityBlock}}})
	if !res.HasBlocking() {
		t.Fatalf("expected blocking violation")
	}
	err := domain.RuleViolationError{Result: res}
	if err.Error() == "" {
		t.Fatalf("expected error string")
	}
}

func TestRulesEngineAggregates(t *testing.T) {
	engine := NewRulesEngine()
	engine.Register(staticRule{"warn", domain.SeverityWarn})
	engine.Register(staticRule{"block", domain.SeverityBlock})

	store := NewMemoryStore(engine)
	ctx := context.Background()

	res, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		_, err := tx.CreateProject(domain.Project{Code: "P", Title: "Project"})
		return err
	})
	if err == nil {
		t.Fatalf("expected transaction to fail due to blocking rule")
	}
	if !res.HasBlocking() {
		t.Fatalf("expected blocking violation from rule engine")
	}
}

type staticRule struct {
	name     string
	severity domain.Severity
}

func (r staticRule) Name() string { return r.name }

func (r staticRule) Evaluate(_ context.Context, _ domain.RuleView, _ []domain.Change) (domain.Result, error) {
	return domain.Result{Violations: []domain.Violation{{Rule: r.name, Severity: r.severity}}}, nil
}
