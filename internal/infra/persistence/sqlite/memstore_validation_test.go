package sqlite

import (
	"context"
	"strings"
	"testing"
	"time"

	"colonycore/pkg/domain"
	entitymodel "colonycore/pkg/domain/entitymodel"
)

func TestMemStoreCloneOptionalFields(t *testing.T) {
	now := time.Now().UTC()
	const (
		descValue   = "desc"
		reasonValue = "reason"
	)
	desc := descValue
	reason := reasonValue

	line := Line{Line: entitymodel.Line{
		Description:       &desc,
		DeprecatedAt:      &now,
		DeprecationReason: &reason,
		GenotypeMarkerIDs: []string{"marker"},
	}}
	clonedLine := cloneLine(line)
	*line.Description = changedValue
	line.GenotypeMarkerIDs[0] = "other"
	if clonedLine.Description == nil || *clonedLine.Description != descValue {
		t.Fatalf("expected line description clone")
	}
	if clonedLine.DeprecatedAt == nil || !clonedLine.DeprecatedAt.Equal(now) {
		t.Fatalf("expected line deprecated timestamp clone")
	}
	if clonedLine.DeprecationReason == nil || *clonedLine.DeprecationReason != reasonValue {
		t.Fatalf("expected line deprecation reason clone")
	}
	if clonedLine.GenotypeMarkerIDs[0] != "marker" {
		t.Fatalf("expected line marker ids clone")
	}

	strainDesc := descValue
	gen := "G2"
	retired := now.Add(time.Hour)
	strain := Strain{Strain: entitymodel.Strain{
		Description:       &strainDesc,
		Generation:        &gen,
		RetiredAt:         &retired,
		RetirementReason:  &reason,
		GenotypeMarkerIDs: []string{"marker"},
	}}
	clonedStrain := cloneStrain(strain)
	*strain.Description = changedValue
	strain.GenotypeMarkerIDs[0] = "other"
	if clonedStrain.Description == nil || *clonedStrain.Description != descValue {
		t.Fatalf("expected strain description clone")
	}
	if clonedStrain.Generation == nil || *clonedStrain.Generation != "G2" {
		t.Fatalf("expected strain generation clone")
	}
	if clonedStrain.RetiredAt == nil || !clonedStrain.RetiredAt.Equal(retired) {
		t.Fatalf("expected strain retired timestamp clone")
	}
	if clonedStrain.RetirementReason == nil || *clonedStrain.RetirementReason != reasonValue {
		t.Fatalf("expected strain retirement reason clone")
	}
	if clonedStrain.GenotypeMarkerIDs[0] != "marker" {
		t.Fatalf("expected strain marker ids clone")
	}

	originalExpires := now.Add(2 * time.Hour)
	expires := originalExpires
	supply := SupplyItem{SupplyItem: entitymodel.SupplyItem{
		ExpiresAt:   &expires,
		FacilityIDs: []string{"facility"},
		ProjectIDs:  []string{"project"},
	}}
	clonedSupply := cloneSupplyItem(supply)
	expires = expires.Add(time.Hour)
	if clonedSupply.ExpiresAt == nil || !clonedSupply.ExpiresAt.Equal(originalExpires) {
		t.Fatalf("expected supply expiry clone to equal original value")
	}
	if clonedSupply.ExpiresAt.Equal(expires) {
		t.Fatalf("expected supply expiry clone to not equal modified value")
	}
	if clonedSupply.FacilityIDs[0] != "facility" || clonedSupply.ProjectIDs[0] != "project" {
		t.Fatalf("expected supply ids clone")
	}

	org := Organism{Organism: entitymodel.Organism{ParentIDs: []string{"parent"}}}
	clonedOrg := cloneOrganism(org)
	org.ParentIDs[0] = changedValue
	if clonedOrg.ParentIDs[0] != "parent" {
		t.Fatalf("expected organism parent ids clone")
	}
}

func TestMemStoreValidationErrors(t *testing.T) {
	store := newMemStore(nil)
	ctx := context.Background()
	now := time.Now().UTC()

	var (
		facility  domain.Facility
		facility2 domain.Facility
		project   domain.Project
		protocol  domain.Protocol
		housing   domain.HousingUnit
		organism  domain.Organism
		sample    domain.Sample
	)

	runTx(t, store, func(tx domain.Transaction) error {
		f, err := tx.CreateFacility(domain.Facility{Facility: entitymodel.Facility{Name: "Facility", Zone: "Z", AccessPolicy: "policy"}})
		if err != nil {
			return err
		}
		facility = f
		f2, err := tx.CreateFacility(domain.Facility{Facility: entitymodel.Facility{Name: "Facility2", Zone: "Z2", AccessPolicy: "policy"}})
		if err != nil {
			return err
		}
		facility2 = f2
		p, err := tx.CreateProject(domain.Project{Project: entitymodel.Project{Code: "PRJ", Title: "Project", FacilityIDs: []string{facility.ID}}})
		if err != nil {
			return err
		}
		project = p
		pr, err := tx.CreateProtocol(domain.Protocol{Protocol: entitymodel.Protocol{Code: "P1", Title: "Protocol", MaxSubjects: 1}})
		if err != nil {
			return err
		}
		protocol = pr
		h, err := tx.CreateHousingUnit(domain.HousingUnit{HousingUnit: entitymodel.HousingUnit{Name: "H1", FacilityID: facility.ID, Capacity: 1, Environment: domain.HousingEnvironmentTerrestrial}})
		if err != nil {
			return err
		}
		housing = h
		o, err := tx.CreateOrganism(domain.Organism{Organism: entitymodel.Organism{Name: "Org", Species: "Spec"}})
		if err != nil {
			return err
		}
		organism = o
		return nil
	})

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		_, err := tx.UpdateHousingUnit(housing.ID, func(h *domain.HousingUnit) error {
			h.FacilityID = ""
			return nil
		})
		return err
	}); err == nil || !strings.Contains(err.Error(), "housing unit requires facility id") {
		t.Fatalf("expected facility id error, got %v", err)
	}

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		_, err := tx.UpdateHousingUnit(housing.ID, func(h *domain.HousingUnit) error {
			h.FacilityID = "missing"
			return nil
		})
		return err
	}); err == nil || !strings.Contains(err.Error(), "facility \"missing\" not found") {
		t.Fatalf("expected missing facility error, got %v", err)
	}

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		_, err := tx.UpdateHousingUnit(housing.ID, func(h *domain.HousingUnit) error {
			h.Capacity = 0
			return nil
		})
		return err
	}); err == nil || !strings.Contains(err.Error(), "housing capacity must be positive") {
		t.Fatalf("expected capacity error, got %v", err)
	}

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		_, err := tx.UpdateHousingUnit(housing.ID, func(h *domain.HousingUnit) error {
			h.Environment = "unknown"
			return nil
		})
		return err
	}); err == nil || !strings.Contains(err.Error(), "unsupported housing environment") {
		t.Fatalf("expected environment error, got %v", err)
	}

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		_, err := tx.CreateObservation(domain.Observation{Observation: entitymodel.Observation{Observer: "tech"}})
		return err
	}); err == nil || !strings.Contains(err.Error(), "observation requires procedure") {
		t.Fatalf("expected observation reference error, got %v", err)
	}

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		_, err := tx.CreateTreatment(domain.Treatment{Treatment: entitymodel.Treatment{Name: "Treat"}})
		return err
	}); err == nil || !strings.Contains(err.Error(), "treatment requires procedure id") {
		t.Fatalf("expected treatment procedure error, got %v", err)
	}

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		return tx.DeleteHousingUnit(housing.ID)
	}); err != nil {
		t.Fatalf("delete housing: %v", err)
	}

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		s, err := tx.CreateSample(domain.Sample{Sample: entitymodel.Sample{Identifier: "S1",
			SourceType:      "organism",
			OrganismID:      &organism.ID,
			FacilityID:      facility.ID,
			CollectedAt:     now,
			Status:          domain.SampleStatusStored,
			StorageLocation: "cold",
			AssayType:       "PCR",
			ChainOfCustody: []domain.SampleCustodyEvent{{
				Actor:     "tech",
				Location:  "bench",
				Timestamp: now,
			}}},
		})
		if err != nil {
			return err
		}
		sample = s
		return nil
	}); err != nil {
		t.Fatalf("create sample: %v", err)
	}

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		return tx.DeleteOrganism(organism.ID)
	}); err == nil || !strings.Contains(err.Error(), "still referenced by sample") {
		t.Fatalf("expected organism sample error, got %v", err)
	}

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		return tx.DeleteFacility(facility.ID)
	}); err == nil || !strings.Contains(err.Error(), "still referenced by sample") {
		t.Fatalf("expected facility sample error, got %v", err)
	}

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		return tx.DeleteSample(sample.ID)
	}); err != nil {
		t.Fatalf("delete sample: %v", err)
	}

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		return tx.DeleteFacility(facility.ID)
	}); err == nil || !strings.Contains(err.Error(), "still referenced by project") {
		t.Fatalf("expected facility project error, got %v", err)
	}

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		return tx.DeleteProject(project.ID)
	}); err != nil {
		t.Fatalf("delete project: %v", err)
	}

	var permit domain.Permit
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		p, err := tx.CreatePermit(domain.Permit{Permit: entitymodel.Permit{PermitNumber: "PER-1",
			Authority:         "Gov",
			Status:            domain.PermitStatusApproved,
			ValidFrom:         now,
			ValidUntil:        now.Add(time.Hour),
			AllowedActivities: []string{"use"},
			FacilityIDs:       []string{facility.ID},
			ProtocolIDs:       []string{protocol.ID}},
		})
		if err != nil {
			return err
		}
		permit = p
		return nil
	}); err != nil {
		t.Fatalf("create permit: %v", err)
	}

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		return tx.DeleteFacility(facility.ID)
	}); err == nil || !strings.Contains(err.Error(), "still referenced by permit") {
		t.Fatalf("expected facility permit error, got %v", err)
	}

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		return tx.DeletePermit(permit.ID)
	}); err != nil {
		t.Fatalf("delete permit: %v", err)
	}

	var project2 domain.Project
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		p, err := tx.CreateProject(domain.Project{Project: entitymodel.Project{Code: "PRJ2", Title: "Project2", FacilityIDs: []string{facility2.ID}}})
		if err != nil {
			return err
		}
		project2 = p
		return nil
	}); err != nil {
		t.Fatalf("create project2: %v", err)
	}

	var supply domain.SupplyItem
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		s, err := tx.CreateSupplyItem(domain.SupplyItem{SupplyItem: entitymodel.SupplyItem{
			SKU:            "SKU",
			Name:           "Item",
			QuantityOnHand: 1,
			Unit:           "unit",
			FacilityIDs:    []string{facility.ID},
			ProjectIDs:     []string{project2.ID},
		}})
		if err != nil {
			return err
		}
		supply = s
		return nil
	}); err != nil {
		t.Fatalf("create supply: %v", err)
	}

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		return tx.DeleteFacility(facility.ID)
	}); err == nil || !strings.Contains(err.Error(), "still referenced by supply item") {
		t.Fatalf("expected facility supply error, got %v", err)
	}

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		return tx.DeleteSupplyItem(supply.ID)
	}); err != nil {
		t.Fatalf("delete supply: %v", err)
	}
}
