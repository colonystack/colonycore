package sqlite

import (
	"colonycore/pkg/domain"
	"context"
	"testing"
	"time"
)

// Mirrors the memory store coverage walk to ensure sqlite memstore CRUD paths stay exercised.
func TestSQLiteMemStoreFullCrudCoverage(t *testing.T) {
	store := newMemStore(nil)
	ctx := context.Background()
	now := time.Now().UTC()

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		marker, err := tx.CreateGenotypeMarker(domain.GenotypeMarker{
			Base:           domain.Base{ID: "marker-full-sqlite"},
			Name:           "Marker",
			Locus:          "loc",
			Alleles:        []string{"A", "C", "C"},
			AssayMethod:    "PCR",
			Interpretation: "ctrl",
			Version:        "v1",
		})
		if err != nil {
			return err
		}

		line, err := tx.CreateLine(domain.Line{
			Base:              domain.Base{ID: "line-full-sqlite"},
			Code:              "L",
			Name:              "Line",
			Origin:            "field",
			GenotypeMarkerIDs: []string{marker.ID},
		})
		if err != nil {
			return err
		}
		strain, err := tx.CreateStrain(domain.Strain{
			Base:              domain.Base{ID: "strain-full-sqlite"},
			Code:              "S",
			Name:              "Strain",
			LineID:            line.ID,
			GenotypeMarkerIDs: []string{marker.ID},
		})
		if err != nil {
			return err
		}

		facility, err := tx.CreateFacility(domain.Facility{Base: domain.Base{ID: "facility-full-sqlite"}, Code: "FAC", Name: "Facility", Zone: "Z", AccessPolicy: "policy"})
		if err != nil {
			return err
		}

		project, err := tx.CreateProject(domain.Project{Base: domain.Base{ID: "project-full-sqlite"}, Code: "PRJ", Title: "Project", FacilityIDs: []string{facility.ID}})
		if err != nil {
			return err
		}

		housing, err := tx.CreateHousingUnit(domain.HousingUnit{Base: domain.Base{ID: "housing-full-sqlite"}, Name: "Housing", FacilityID: facility.ID, Capacity: 2, Environment: domain.HousingEnvironmentAquatic})
		if err != nil {
			return err
		}

		if _, err := tx.UpdateHousingUnit(housing.ID, func(h *domain.HousingUnit) error {
			h.Environment = domain.HousingEnvironmentTerrestrial
			return nil
		}); err != nil {
			return err
		}

		if _, err := tx.UpdateFacility(facility.ID, func(f *domain.Facility) error {
			f.Zone = "Z2"
			f.ProjectIDs = []string{project.ID}
			return nil
		}); err != nil {
			return err
		}

		protocol, err := tx.CreateProtocol(domain.Protocol{Base: domain.Base{ID: "protocol-full-sqlite"}, Code: "PROT", Title: "Protocol", MaxSubjects: 5})
		if err != nil {
			return err
		}

		if _, err := tx.UpdateProtocol(protocol.ID, func(p *domain.Protocol) error {
			p.Description = ptr("desc")
			return nil
		}); err != nil {
			return err
		}

		cohort, err := tx.CreateCohort(domain.Cohort{Base: domain.Base{ID: "cohort-full-sqlite"}, Name: "Cohort", Purpose: "Study", ProjectID: &project.ID, HousingID: &housing.ID, ProtocolID: &protocol.ID})
		if err != nil {
			return err
		}

		breeding, err := tx.CreateBreedingUnit(domain.BreedingUnit{Base: domain.Base{ID: "breeding-full-sqlite"}, Name: "Breeding", Strategy: "pair", LineID: &line.ID, StrainID: &strain.ID})
		if err != nil {
			return err
		}
		if _, err := tx.UpdateBreedingUnit(breeding.ID, func(b *domain.BreedingUnit) error {
			intent := "intent"
			b.PairingIntent = &intent
			return nil
		}); err != nil {
			return err
		}

		organism, err := tx.CreateOrganism(domain.Organism{Base: domain.Base{ID: "organism-full-sqlite"}, Name: "Org", Species: "Spec", LineID: &line.ID, StrainID: &strain.ID, CohortID: &cohort.ID, HousingID: &housing.ID})
		if err != nil {
			return err
		}

		if _, err := tx.UpdateOrganism(organism.ID, func(o *domain.Organism) error {
			o.ParentIDs = []string{"parent"}
			return nil
		}); err != nil {
			return err
		}

		procedure, err := tx.CreateProcedure(domain.Procedure{Base: domain.Base{ID: "procedure-full-sqlite"}, Name: "Proc", Status: domain.ProcedureStatusScheduled, ScheduledAt: now, ProtocolID: protocol.ID, CohortID: &cohort.ID, OrganismIDs: []string{organism.ID}})
		if err != nil {
			return err
		}

		if _, err := tx.UpdateProcedure(procedure.ID, func(p *domain.Procedure) error {
			p.ProjectID = &project.ID
			return nil
		}); err != nil {
			return err
		}

		treatment, err := tx.CreateTreatment(domain.Treatment{Base: domain.Base{ID: "treatment-full-sqlite"}, Name: "Treat", Status: domain.TreatmentStatusPlanned, ProcedureID: procedure.ID, OrganismIDs: []string{organism.ID}, CohortIDs: []string{cohort.ID}})
		if err != nil {
			return err
		}

		if _, err := tx.UpdateTreatment(treatment.ID, func(t *domain.Treatment) error {
			t.DosagePlan = "plan"
			return nil
		}); err != nil {
			return err
		}

		observation, err := tx.CreateObservation(domain.Observation{Base: domain.Base{ID: "observation-full-sqlite"}, ProcedureID: &procedure.ID, OrganismID: &organism.ID, RecordedAt: now, Observer: "tech"})
		if err != nil {
			return err
		}

		if _, err := tx.UpdateObservation(observation.ID, func(o *domain.Observation) error {
			o.Notes = ptr("updated")
			return nil
		}); err != nil {
			return err
		}

		sample, err := tx.CreateSample(domain.Sample{Base: domain.Base{ID: "sample-full-sqlite"}, Identifier: "S1", SourceType: "organism", OrganismID: &organism.ID, FacilityID: facility.ID, CollectedAt: now, Status: domain.SampleStatusStored, StorageLocation: "cold", AssayType: "type", ChainOfCustody: []domain.SampleCustodyEvent{{Actor: "tech", Location: "cold", Timestamp: now}}})
		if err != nil {
			return err
		}

		if _, err := tx.UpdateSample(sample.ID, func(s *domain.Sample) error {
			s.StorageLocation = "ambient"
			return nil
		}); err != nil {
			return err
		}

		permit, err := tx.CreatePermit(domain.Permit{Base: domain.Base{ID: "permit-full-sqlite"}, PermitNumber: "PERMIT", Authority: "Gov", ValidFrom: now, ValidUntil: now.Add(time.Hour), AllowedActivities: []string{"store"}, FacilityIDs: []string{facility.ID}, ProtocolIDs: []string{protocol.ID}})
		if err != nil {
			return err
		}

		if _, err := tx.UpdatePermit(permit.ID, func(p *domain.Permit) error {
			p.Status = domain.PermitStatusApproved
			return nil
		}); err != nil {
			return err
		}

		supply, err := tx.CreateSupplyItem(domain.SupplyItem{Base: domain.Base{ID: "supply-full-sqlite"}, SKU: "SKU", Name: "Item", QuantityOnHand: 1, Unit: "unit", FacilityIDs: []string{facility.ID}, ProjectIDs: []string{project.ID}})
		if err != nil {
			return err
		}

		if _, err := tx.UpdateSupplyItem(supply.ID, func(s *domain.SupplyItem) error {
			s.QuantityOnHand = 2
			return nil
		}); err != nil {
			return err
		}

		if _, err := tx.UpdateProject(project.ID, func(p *domain.Project) error {
			p.Description = ptr("updated")
			return nil
		}); err != nil {
			return err
		}

		for _, del := range []struct {
			fn func() error
		}{
			{fn: func() error { return tx.DeleteObservation(observation.ID) }},
			{fn: func() error { return tx.DeleteTreatment(treatment.ID) }},
			{fn: func() error { return tx.DeleteProcedure(procedure.ID) }},
			{fn: func() error { return tx.DeleteSample(sample.ID) }},
			{fn: func() error { return tx.DeletePermit(permit.ID) }},
			{fn: func() error { return tx.DeleteSupplyItem(supply.ID) }},
			{fn: func() error { return tx.DeleteOrganism(organism.ID) }},
			{fn: func() error { return tx.DeleteCohort(cohort.ID) }},
			{fn: func() error { return tx.DeleteHousingUnit(housing.ID) }},
			{fn: func() error { return tx.DeleteProject(project.ID) }},
			{fn: func() error { return tx.DeleteBreedingUnit(breeding.ID) }},
			{fn: func() error { return tx.DeleteStrain(strain.ID) }},
			{fn: func() error { return tx.DeleteLine(line.ID) }},
			{fn: func() error { return tx.DeleteGenotypeMarker(marker.ID) }},
			{fn: func() error { return tx.DeleteFacility(facility.ID) }},
		} {
			if err := del.fn(); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("full crud transaction: %v", err)
	}
}
