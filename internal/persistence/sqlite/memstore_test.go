package sqlite

import (
	"colonycore/pkg/domain"
	"context"
	"fmt"
	"testing"
	"time"
)

func TestMemStoreRunInTransactionAndSnapshots(t *testing.T) {
	store := newMemStore(nil)
	ctx := context.Background()
	_, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		if _, ok := tx.FindHousingUnit("missing"); ok {
			t.Fatalf("expected missing housing lookup")
		}
		created, err := tx.CreateOrganism(domain.Organism{Name: "Test", Species: "Frog"})
		if err != nil {
			return err
		}
		if created.ID == "" {
			t.Fatalf("expected generated ID")
		}
		view := tx.Snapshot()
		if len(view.ListOrganisms()) != 1 {
			t.Fatalf("snapshot mismatch")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("run transaction: %v", err)
	}
	if len(store.ListOrganisms()) != 1 {
		t.Fatalf("expected persisted organism")
	}
	snapshot := store.ExportState()
	store.ImportState(Snapshot{})
	if len(store.ListOrganisms()) != 0 {
		t.Fatalf("expected cleared state")
	}
	store.ImportState(snapshot)
	if len(store.ListOrganisms()) != 1 {
		t.Fatalf("expected restored state")
	}
	if err := store.View(ctx, func(view domain.TransactionView) error {
		if len(view.ListOrganisms()) != 1 {
			return fmt.Errorf("expected organism in view")
		}
		return nil
	}); err != nil {
		t.Fatalf("view: %v", err)
	}
	if store.RulesEngine() == nil {
		t.Fatalf("expected rules engine")
	}
	if store.NowFunc() == nil {
		t.Fatalf("expected now func")
	}
}

func TestMemStoreRuleViolation(t *testing.T) {
	store := newMemStore(domain.NewRulesEngine())
	store.RulesEngine().Register(blockingRule{})
	_, err := store.RunInTransaction(context.Background(), func(tx domain.Transaction) error {
		_, e := tx.CreateOrganism(domain.Organism{Name: "Fail"})
		return e
	})
	if err == nil {
		t.Fatalf("expected rule violation error")
	}
}

func TestMemStoreCRUDAndQueries(t *testing.T) {
	store := newMemStore(nil)
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
		if _, ok := tx.FindProtocol(protocolID); !ok {
			return fmt.Errorf("expected to find protocol %s", protocolID)
		}
		if _, ok := tx.FindProtocol("missing-protocol"); ok {
			return fmt.Errorf("unexpected protocol lookup success")
		}

		housing, err := tx.CreateHousingUnit(domain.HousingUnit{Name: "Tank", Facility: "Lab", Capacity: 2, Environment: "arid"})
		if err != nil {
			return err
		}
		housingID = housing.ID
		if _, ok := tx.FindHousingUnit(housingID); !ok {
			return fmt.Errorf("expected to find housing %s", housingID)
		}

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

		if _, err := tx.UpdateBreedingUnit(breedingID, func(unit *domain.BreedingUnit) error {
			unit.MaleIDs = append(unit.MaleIDs, "extra")
			return nil
		}); err != nil {
			return err
		}

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
	if got := len(store.ListProjects()); got != 1 {
		t.Fatalf("expected one project, got %d", got)
	}
	if got := len(store.ListProtocols()); got != 1 {
		t.Fatalf("expected one protocol, got %d", got)
	}
	if got := len(store.ListCohorts()); got != 1 {
		t.Fatalf("expected one cohort, got %d", got)
	}
	if got := len(store.ListBreedingUnits()); got != 1 {
		t.Fatalf("expected one breeding unit, got %d", got)
	}
	if got := len(store.ListProcedures()); got != 1 {
		t.Fatalf("expected one procedure, got %d", got)
	}
	if err := store.View(ctx, func(view domain.TransactionView) error {
		if len(view.ListHousingUnits()) != 1 {
			return fmt.Errorf("expected housing in view")
		}
		return nil
	}); err != nil {
		t.Fatalf("store view: %v", err)
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
		if _, err := tx.UpdateOrganism(organismAID, func(o *domain.Organism) error {
			o.Name = "Updated"
			return nil
		}); err != nil {
			return err
		}
		if _, err := tx.UpdateHousingUnit(housingID, func(h *domain.HousingUnit) error {
			h.Capacity = 0
			return nil
		}); err == nil {
			return fmt.Errorf("expected housing capacity validation on update")
		}
		if _, err := tx.UpdateHousingUnit(housingID, func(h *domain.HousingUnit) error {
			h.Environment = "humid"
			return nil
		}); err != nil {
			return err
		}
		if _, err := tx.UpdateHousingUnit("missing", func(*domain.HousingUnit) error { return nil }); err == nil {
			return fmt.Errorf("expected missing housing update error")
		}
		if _, err := tx.UpdateProject(projectID, func(p *domain.Project) error {
			p.Description = "updated"
			return nil
		}); err != nil {
			return err
		}
		if _, err := tx.UpdateProtocol(protocolID, func(p *domain.Protocol) error {
			p.Description = "updated"
			return nil
		}); err != nil {
			return err
		}
		if _, err := tx.UpdateCohort(cohortID, func(c *domain.Cohort) error {
			c.Purpose = "updated"
			return nil
		}); err != nil {
			return err
		}
		if _, err := tx.UpdateBreedingUnit("missing", func(*domain.BreedingUnit) error { return nil }); err == nil {
			return fmt.Errorf("expected missing breeding unit update error")
		}
		if _, err := tx.UpdateProcedure(procedureID, func(p *domain.Procedure) error {
			p.Status = "complete"
			return nil
		}); err != nil {
			return err
		}
		if err := tx.DeleteProcedure(procedureID); err != nil {
			return err
		}
		if err := tx.DeleteBreedingUnit(breedingID); err != nil {
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
		if err := tx.DeleteProject(projectID); err != nil {
			return err
		}
		if err := tx.DeleteProtocol(protocolID); err != nil {
			return err
		}
		return nil
	}); err != nil {
		t.Fatalf("update transaction: %v", err)
	}
}

type blockingRule struct{}

func (blockingRule) Name() string { return "block" }

func (blockingRule) Evaluate(ctx context.Context, view domain.RuleView, changes []domain.Change) (domain.Result, error) {
	res := domain.Result{}
	res.Merge(domain.Result{Violations: []domain.Violation{{Rule: "block", Severity: domain.SeverityBlock}}})
	return res, nil
}
