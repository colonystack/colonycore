package sqlite

import (
	"colonycore/pkg/domain"
	entitymodel "colonycore/pkg/domain/entitymodel"
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFindersCoverSuccessAndFailure(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "finders.db"), domain.NewRulesEngine())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { _ = store.DB().Close() })

	ctx := context.Background()
	var organismID string
	var housingID string

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		facility, err := tx.CreateFacility(domain.Facility{Facility: entitymodel.Facility{
			Code:         "FAC-FIND",
			Name:         "Facility",
			Zone:         "Z",
			AccessPolicy: "all",
		}})
		if err != nil {
			return err
		}
		housing, err := tx.CreateHousingUnit(domain.HousingUnit{HousingUnit: entitymodel.HousingUnit{
			Name:       "Housing",
			FacilityID: facility.ID,
			Capacity:   1,
		}})
		if err != nil {
			return err
		}
		housingID = housing.ID
		organism, err := tx.CreateOrganism(domain.Organism{Organism: entitymodel.Organism{
			Name:      "Org",
			Species:   "species",
			Stage:     domain.StageAdult,
			HousingID: &housingID,
		}})
		if err != nil {
			return err
		}
		organismID = organism.ID
		return nil
	}); err != nil {
		t.Fatalf("seed transaction: %v", err)
	}

	if err := store.View(ctx, func(view domain.TransactionView) error {
		if _, ok := view.FindOrganism("missing"); ok {
			t.Fatalf("expected missing organism lookup to return false")
		}
		if _, ok := view.FindOrganism(organismID); !ok {
			t.Fatalf("expected stored organism lookup to succeed")
		}
		if _, ok := view.FindHousingUnit("missing"); ok {
			t.Fatalf("expected missing housing lookup to return false")
		}
		if _, ok := view.FindHousingUnit(housingID); !ok {
			t.Fatalf("expected stored housing lookup to succeed")
		}
		return nil
	}); err != nil {
		t.Fatalf("view validation: %v", err)
	}
}

func TestDeleteCohortCoversBranches(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "delete-cohort.db"), domain.NewRulesEngine())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { _ = store.DB().Close() })

	ctx := context.Background()

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		facility, err := tx.CreateFacility(domain.Facility{Facility: entitymodel.Facility{
			Code:         "FAC-DEL",
			Name:         "Facility",
			Zone:         "Z",
			AccessPolicy: "all",
		}})
		if err != nil {
			return err
		}
		project, err := tx.CreateProject(domain.Project{Project: entitymodel.Project{
			Code:        "PRJ-DEL",
			Title:       "Project",
			FacilityIDs: []string{facility.ID},
		}})
		if err != nil {
			return err
		}
		cohort, err := tx.CreateCohort(domain.Cohort{Cohort: entitymodel.Cohort{
			Name:      "Cohort",
			Purpose:   "purpose",
			ProjectID: &project.ID,
		}})
		if err != nil {
			return err
		}
		if err := tx.DeleteCohort("missing-cohort"); err == nil {
			return fmt.Errorf("expected missing cohort delete to error")
		}
		return tx.DeleteCohort(cohort.ID)
	}); err != nil {
		t.Fatalf("RunInTransaction: %v", err)
	}
}

func TestDeleteFacilityCoversBranches(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "delete-facility.db"), domain.NewRulesEngine())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { _ = store.DB().Close() })

	ctx := context.Background()

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		if err := tx.DeleteFacility("missing-facility"); err == nil {
			return fmt.Errorf("expected missing facility delete to error")
		}
		facility, err := tx.CreateFacility(domain.Facility{Facility: entitymodel.Facility{
			Code:         "FAC-DEL-2",
			Name:         "Facility",
			Zone:         "Z",
			AccessPolicy: "all",
		}})
		if err != nil {
			return err
		}
		return tx.DeleteFacility(facility.ID)
	}); err != nil {
		t.Fatalf("RunInTransaction: %v", err)
	}
}

func TestDeleteFacilityBlockedByHousing(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "delete-facility-blocked.db"), domain.NewRulesEngine())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { _ = store.DB().Close() })

	ctx := context.Background()
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		facility, err := tx.CreateFacility(domain.Facility{Facility: entitymodel.Facility{
			Code:         "FAC-BLOCK",
			Name:         "Facility",
			Zone:         "Z",
			AccessPolicy: "all",
		}})
		if err != nil {
			return err
		}
		if _, err := tx.CreateHousingUnit(domain.HousingUnit{HousingUnit: entitymodel.HousingUnit{
			Name:       "Housing",
			FacilityID: facility.ID,
			Capacity:   1,
		}}); err != nil {
			return err
		}
		if err := tx.DeleteFacility(facility.ID); err == nil || !strings.Contains(err.Error(), "housing") {
			return fmt.Errorf("expected delete facility to fail due to housing reference, got %v", err)
		}
		return nil
	}); err != nil {
		t.Fatalf("RunInTransaction: %v", err)
	}
}

func TestDeleteFacilityBlockedBySample(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "delete-facility-sample.db"), domain.NewRulesEngine())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { _ = store.DB().Close() })

	ctx := context.Background()
	now := time.Now().UTC()

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		facility, err := tx.CreateFacility(domain.Facility{Facility: entitymodel.Facility{
			Code:         "FAC-SAMP",
			Name:         "Facility",
			Zone:         "Z",
			AccessPolicy: "all",
		}})
		if err != nil {
			return err
		}
		organism, err := tx.CreateOrganism(domain.Organism{Organism: entitymodel.Organism{
			Name:    "Org",
			Species: "species",
			Stage:   domain.StageAdult,
		}})
		if err != nil {
			return err
		}
		if _, err := tx.CreateSample(domain.Sample{Sample: entitymodel.Sample{
			Identifier:      "SAMP-1",
			SourceType:      "blood",
			FacilityID:      facility.ID,
			OrganismID:      &organism.ID,
			CollectedAt:     now,
			StorageLocation: "cold",
			AssayType:       "assay",
			ChainOfCustody:  []domain.SampleCustodyEvent{{Actor: "tech", Location: "cold", Timestamp: now}},
		}}); err != nil {
			return err
		}
		if err := tx.DeleteFacility(facility.ID); err == nil || !strings.Contains(err.Error(), "sample") {
			return fmt.Errorf("expected delete facility to fail due to sample reference, got %v", err)
		}
		return nil
	}); err != nil {
		t.Fatalf("RunInTransaction: %v", err)
	}
}
