package sqlite

import (
	"colonycore/pkg/domain"
	"context"
	"strings"
	"testing"
	"time"
)

func TestLineRequiresGenotypeMarkersSQLite(t *testing.T) {
	store := newMemStore(nil)
	ctx := context.Background()
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		marker, err := tx.CreateGenotypeMarker(domain.GenotypeMarker{Name: "m1", Locus: "loc1", Alleles: []string{"A"}, AssayMethod: "pcr", Interpretation: "ok", Version: "v1"})
		if err != nil {
			return err
		}
		if _, err := tx.CreateLine(domain.Line{Code: "L-empty", Name: "Line", Origin: "lab"}); err == nil {
			t.Fatalf("expected error for missing genotype markers")
		}
		line, err := tx.CreateLine(domain.Line{Code: "L-one", Name: "Line", Origin: "lab", GenotypeMarkerIDs: []string{marker.ID}})
		if err != nil {
			return err
		}
		if _, err := tx.UpdateLine(line.ID, func(l *domain.Line) error {
			l.GenotypeMarkerIDs = nil
			return nil
		}); err == nil {
			t.Fatalf("expected update error when clearing genotype markers")
		}
		return nil
	}); err != nil {
		t.Fatalf("transaction: %v", err)
	}
}

func TestPermitRequiresRequiredArraysSQLite(t *testing.T) {
	store := newMemStore(nil)
	ctx := context.Background()
	now := time.Now()
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		facility, err := tx.CreateFacility(domain.Facility{Name: "Facility"})
		if err != nil {
			return err
		}
		protocol, err := tx.CreateProtocol(domain.Protocol{Code: "P-1", Title: "Protocol", MaxSubjects: 5})
		if err != nil {
			return err
		}
		if _, err := tx.CreatePermit(domain.Permit{PermitNumber: "PERM-0", Authority: "Auth", ValidFrom: now, ValidUntil: now.Add(time.Hour), FacilityIDs: []string{facility.ID}, ProtocolIDs: []string{protocol.ID}}); err == nil {
			t.Fatalf("expected error for missing allowed activities")
		}
		permit, err := tx.CreatePermit(domain.Permit{
			PermitNumber:      "PERM-1",
			Authority:         "Auth",
			ValidFrom:         now,
			ValidUntil:        now.Add(time.Hour),
			AllowedActivities: []string{"store"},
			FacilityIDs:       []string{facility.ID},
			ProtocolIDs:       []string{protocol.ID},
		})
		if err != nil {
			return err
		}
		if _, err := tx.UpdatePermit(permit.ID, func(p *domain.Permit) error {
			p.AllowedActivities = nil
			return nil
		}); err == nil || !strings.Contains(err.Error(), "allowed_activities") {
			t.Fatalf("expected allowed activities update error, got %v", err)
		}
		if _, err := tx.UpdatePermit(permit.ID, func(p *domain.Permit) error {
			p.FacilityIDs = nil
			return nil
		}); err == nil || !strings.Contains(err.Error(), "facility_ids") {
			t.Fatalf("expected facility_ids update error, got %v", err)
		}
		if _, err := tx.UpdatePermit(permit.ID, func(p *domain.Permit) error {
			p.ProtocolIDs = nil
			return nil
		}); err == nil || !strings.Contains(err.Error(), "protocol_ids") {
			t.Fatalf("expected protocol_ids update error, got %v", err)
		}
		return nil
	}); err != nil {
		t.Fatalf("transaction: %v", err)
	}
}

func TestProjectRequiresFacilitiesSQLite(t *testing.T) {
	store := newMemStore(nil)
	ctx := context.Background()
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		if _, err := tx.CreateProject(domain.Project{Code: "PRJ-0", Title: "Empty"}); err == nil {
			t.Fatalf("expected error for missing facility_ids")
		}
		facility, err := tx.CreateFacility(domain.Facility{Name: "Facility"})
		if err != nil {
			return err
		}
		project, err := tx.CreateProject(domain.Project{Code: "PRJ-1", Title: "Project", FacilityIDs: []string{facility.ID}})
		if err != nil {
			return err
		}
		if _, err := tx.UpdateProject(project.ID, func(p *domain.Project) error {
			p.FacilityIDs = nil
			return nil
		}); err == nil {
			t.Fatalf("expected update error when clearing facility_ids")
		}
		return nil
	}); err != nil {
		t.Fatalf("transaction: %v", err)
	}
}

func TestSupplyRequiresFacilitiesAndProjectsSQLite(t *testing.T) {
	store := newMemStore(nil)
	ctx := context.Background()
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		facility, err := tx.CreateFacility(domain.Facility{Name: "Facility"})
		if err != nil {
			return err
		}
		project, err := tx.CreateProject(domain.Project{Code: "PRJ-S", Title: "Project", FacilityIDs: []string{facility.ID}})
		if err != nil {
			return err
		}
		if _, err := tx.CreateSupplyItem(domain.SupplyItem{SKU: "SKU-1", Name: "Item", QuantityOnHand: 1, Unit: "u", FacilityIDs: []string{facility.ID}}); err == nil {
			t.Fatalf("expected error for missing project_ids")
		}
		item, err := tx.CreateSupplyItem(domain.SupplyItem{
			SKU:            "SKU-2",
			Name:           "Item",
			QuantityOnHand: 1,
			Unit:           "u",
			FacilityIDs:    []string{facility.ID},
			ProjectIDs:     []string{project.ID},
			ReorderLevel:   1,
		})
		if err != nil {
			return err
		}
		if _, err := tx.UpdateSupplyItem(item.ID, func(s *domain.SupplyItem) error {
			s.ProjectIDs = nil
			return nil
		}); err == nil {
			t.Fatalf("expected update error when clearing project_ids")
		}
		if _, err := tx.UpdateSupplyItem(item.ID, func(s *domain.SupplyItem) error {
			s.FacilityIDs = nil
			return nil
		}); err == nil {
			t.Fatalf("expected update error when clearing facility_ids")
		}
		return nil
	}); err != nil {
		t.Fatalf("transaction: %v", err)
	}
}

func TestSampleRequiresChainOfCustodySQLite(t *testing.T) {
	store := newMemStore(nil)
	ctx := context.Background()
	now := time.Now()
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		facility, err := tx.CreateFacility(domain.Facility{Name: "Facility"})
		if err != nil {
			return err
		}
		org, err := tx.CreateOrganism(domain.Organism{Name: "Specimen", Species: "sp"})
		if err != nil {
			return err
		}
		if _, err := tx.CreateSample(domain.Sample{
			Identifier:      "S-0",
			SourceType:      "blood",
			OrganismID:      &org.ID,
			FacilityID:      facility.ID,
			CollectedAt:     now,
			Status:          domain.SampleStatusStored,
			StorageLocation: "loc",
			AssayType:       "assay",
		}); err == nil {
			t.Fatalf("expected error for missing chain of custody")
		}
		sample, err := tx.CreateSample(domain.Sample{
			Identifier:      "S-1",
			SourceType:      "blood",
			OrganismID:      &org.ID,
			FacilityID:      facility.ID,
			CollectedAt:     now,
			Status:          domain.SampleStatusStored,
			StorageLocation: "loc",
			AssayType:       "assay",
			ChainOfCustody: []domain.SampleCustodyEvent{{
				Actor:     "tech",
				Location:  "loc",
				Timestamp: now,
			}},
		})
		if err != nil {
			return err
		}
		if _, err := tx.UpdateSample(sample.ID, func(s *domain.Sample) error {
			s.ChainOfCustody = nil
			return nil
		}); err == nil {
			t.Fatalf("expected update error when clearing chain of custody")
		}
		return nil
	}); err != nil {
		t.Fatalf("transaction: %v", err)
	}
}
