package sqlite

import (
	"colonycore/pkg/domain"
	entitymodel "colonycore/pkg/domain/entitymodel"
	"context"
	"fmt"
	"testing"
	"time"
)

func TestMemStoreDeleteStrainGuardsSQLite(t *testing.T) {
	store := newMemStore(nil)
	ctx := context.Background()

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		marker, err := tx.CreateGenotypeMarker(domain.GenotypeMarker{GenotypeMarker: entitymodel.GenotypeMarker{Name: "Marker", Locus: "loc", Alleles: []string{"A"}, AssayMethod: "PCR", Interpretation: "ctrl", Version: "v1"}})
		if err != nil {
			return err
		}
		line, err := tx.CreateLine(domain.Line{Line: entitymodel.Line{Code: "L-guard", Name: "Line", GenotypeMarkerIDs: []string{marker.ID}}})
		if err != nil {
			return err
		}
		strain, err := tx.CreateStrain(domain.Strain{Strain: entitymodel.Strain{Code: "S-guard", Name: "Strain", LineID: line.ID, GenotypeMarkerIDs: []string{marker.ID}}})
		if err != nil {
			return err
		}
		organism, err := tx.CreateOrganism(domain.Organism{Organism: entitymodel.Organism{Name: "Org", Species: "Spec", StrainID: &strain.ID}})
		if err != nil {
			return err
		}
		breedingPrimary, err := tx.CreateBreedingUnit(domain.BreedingUnit{BreedingUnit: entitymodel.BreedingUnit{Name: "Breed-primary", Strategy: "pair", StrainID: &strain.ID}})
		if err != nil {
			return err
		}
		breedingTarget, err := tx.CreateBreedingUnit(domain.BreedingUnit{BreedingUnit: entitymodel.BreedingUnit{Name: "Breed-target", Strategy: "pair", TargetStrainID: &strain.ID}})
		if err != nil {
			return err
		}

		if err := tx.DeleteStrain(strain.ID); err == nil {
			return fmt.Errorf("expected delete to fail while organism present")
		}
		if err := tx.DeleteOrganism(organism.ID); err != nil {
			return err
		}

		if err := tx.DeleteBreedingUnit(breedingTarget.ID); err != nil {
			return err
		}
		if err := tx.DeleteStrain(strain.ID); err == nil {
			return fmt.Errorf("expected delete to fail while breeding strain reference present")
		}

		targetOnly, err := tx.CreateBreedingUnit(domain.BreedingUnit{BreedingUnit: entitymodel.BreedingUnit{Name: "Breed-target-only", Strategy: "pair", TargetStrainID: &strain.ID}})
		if err != nil {
			return err
		}
		if err := tx.DeleteBreedingUnit(breedingPrimary.ID); err != nil {
			return err
		}
		if err := tx.DeleteStrain(strain.ID); err == nil {
			return fmt.Errorf("expected delete to fail while breeding target reference present")
		}

		if err := tx.DeleteBreedingUnit(targetOnly.ID); err != nil {
			return err
		}
		return tx.DeleteStrain(strain.ID)
	}); err != nil {
		t.Fatalf("guarded delete: %v", err)
	}
}

func TestMemStoreDeleteLineGuardsSQLite(t *testing.T) {
	store := newMemStore(nil)
	ctx := context.Background()

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		marker, err := tx.CreateGenotypeMarker(domain.GenotypeMarker{GenotypeMarker: entitymodel.GenotypeMarker{Name: "LineMarker", Locus: "loc", Alleles: []string{"A"}, AssayMethod: "PCR", Interpretation: "ctrl", Version: "v1"}})
		if err != nil {
			return err
		}
		line, err := tx.CreateLine(domain.Line{Line: entitymodel.Line{Code: "L-line-guard", Name: "Line", GenotypeMarkerIDs: []string{marker.ID}}})
		if err != nil {
			return err
		}
		breedingLine, err := tx.CreateBreedingUnit(domain.BreedingUnit{BreedingUnit: entitymodel.BreedingUnit{Name: "BreedLine", Strategy: "pair", LineID: &line.ID}})
		if err != nil {
			return err
		}
		if err := tx.DeleteLine(line.ID); err == nil {
			return fmt.Errorf("expected delete to fail while breeding references line")
		}
		if err := tx.DeleteBreedingUnit(breedingLine.ID); err != nil {
			return err
		}

		breedingTarget, err := tx.CreateBreedingUnit(domain.BreedingUnit{BreedingUnit: entitymodel.BreedingUnit{Name: "BreedTargetLine", Strategy: "pair", TargetLineID: &line.ID}})
		if err != nil {
			return err
		}
		if err := tx.DeleteLine(line.ID); err == nil {
			return fmt.Errorf("expected delete to fail while breeding target references line")
		}
		if err := tx.DeleteBreedingUnit(breedingTarget.ID); err != nil {
			return err
		}

		organism, err := tx.CreateOrganism(domain.Organism{Organism: entitymodel.Organism{Name: "Org", Species: "Spec", LineID: &line.ID}})
		if err != nil {
			return err
		}
		if err := tx.DeleteLine(line.ID); err == nil {
			return fmt.Errorf("expected delete to fail while organism references line")
		}
		if err := tx.DeleteOrganism(organism.ID); err != nil {
			return err
		}

		return tx.DeleteLine(line.ID)
	}); err != nil {
		t.Fatalf("line delete guards: %v", err)
	}
}

func TestMemStoreUpdateSampleGuardsSQLite(t *testing.T) {
	store := newMemStore(nil)
	ctx := context.Background()
	now := time.Now().UTC()

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		facility, err := tx.CreateFacility(domain.Facility{Facility: entitymodel.Facility{Name: "Lab"}})
		if err != nil {
			return err
		}
		organism, err := tx.CreateOrganism(domain.Organism{Organism: entitymodel.Organism{Name: "Subject", Species: "Spec"}})
		if err != nil {
			return err
		}
		sample, err := tx.CreateSample(domain.Sample{Sample: entitymodel.Sample{Identifier: "S-guard",
			SourceType:      "organism",
			OrganismID:      &organism.ID,
			FacilityID:      facility.ID,
			CollectedAt:     now,
			Status:          domain.SampleStatusStored,
			StorageLocation: "cold",
			ChainOfCustody: []domain.SampleCustodyEvent{{
				Actor:     "tech",
				Location:  "cold",
				Timestamp: now,
			}}},
		})
		if err != nil {
			return err
		}

		if _, err := tx.UpdateSample(sample.ID, func(s *domain.Sample) error {
			s.FacilityID = ""
			return nil
		}); err == nil {
			return fmt.Errorf("expected facility guard to trigger")
		}

		missingOrganism := "missing"
		if _, err := tx.UpdateSample(sample.ID, func(s *domain.Sample) error {
			s.FacilityID = facility.ID
			s.OrganismID = &missingOrganism
			return nil
		}); err == nil {
			return fmt.Errorf("expected missing organism guard to trigger")
		}

		if _, err := tx.UpdateSample(sample.ID, func(s *domain.Sample) error {
			s.FacilityID = facility.ID
			s.OrganismID = nil
			s.CohortID = nil
			return nil
		}); err == nil {
			return fmt.Errorf("expected subject guard to trigger")
		}

		if _, err := tx.UpdateSample(sample.ID, func(s *domain.Sample) error {
			s.FacilityID = "missing-facility"
			s.OrganismID = &organism.ID
			return nil
		}); err == nil {
			return fmt.Errorf("expected missing facility guard")
		}

		missingCohort := "missing-cohort"
		if _, err := tx.UpdateSample(sample.ID, func(s *domain.Sample) error {
			s.FacilityID = facility.ID
			s.OrganismID = nil
			s.CohortID = &missingCohort
			return nil
		}); err == nil {
			return fmt.Errorf("expected missing cohort guard")
		}

		_, err = tx.UpdateSample(sample.ID, func(s *domain.Sample) error {
			s.FacilityID = facility.ID
			s.OrganismID = &organism.ID
			s.StorageLocation = "ambient"
			return nil
		})
		return err
	}); err != nil {
		t.Fatalf("update sample guards: %v", err)
	}
}

func TestMemStoreUpdateTreatmentGuardsSQLite(t *testing.T) {
	store := newMemStore(nil)
	ctx := context.Background()
	now := time.Now().UTC()

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		protocol, err := tx.CreateProtocol(domain.Protocol{Protocol: entitymodel.Protocol{Code: "P-guard", Title: "Protocol", MaxSubjects: 1}})
		if err != nil {
			return err
		}
		procedure, err := tx.CreateProcedure(domain.Procedure{Procedure: entitymodel.Procedure{Name: "Proc", Status: domain.ProcedureStatusScheduled, ScheduledAt: now, ProtocolID: protocol.ID}})
		if err != nil {
			return err
		}
		organism, err := tx.CreateOrganism(domain.Organism{Organism: entitymodel.Organism{Name: "Org", Species: "Spec"}})
		if err != nil {
			return err
		}
		treatment, err := tx.CreateTreatment(domain.Treatment{Treatment: entitymodel.Treatment{ID: "treat-guard", Name: "Treat", Status: domain.TreatmentStatusPlanned, ProcedureID: procedure.ID, OrganismIDs: []string{organism.ID}}})
		if err != nil {
			return err
		}

		if _, err := tx.UpdateTreatment(treatment.ID, func(t *domain.Treatment) error {
			t.ProcedureID = ""
			return nil
		}); err == nil {
			return fmt.Errorf("expected missing procedure guard")
		}

		if _, err := tx.UpdateTreatment(treatment.ID, func(t *domain.Treatment) error {
			t.ProcedureID = procedure.ID
			t.OrganismIDs = []string{"missing"}
			return nil
		}); err == nil {
			return fmt.Errorf("expected missing organism guard")
		}

		if _, err := tx.UpdateTreatment(treatment.ID, func(t *domain.Treatment) error {
			t.ProcedureID = procedure.ID
			t.OrganismIDs = []string{organism.ID}
			t.CohortIDs = []string{"missing-cohort"}
			return nil
		}); err == nil {
			return fmt.Errorf("expected missing cohort guard")
		}

		_, err = tx.UpdateTreatment(treatment.ID, func(t *domain.Treatment) error {
			t.ProcedureID = procedure.ID
			t.OrganismIDs = []string{organism.ID}
			t.CohortIDs = nil
			return nil
		})
		return err
	}); err != nil {
		t.Fatalf("update treatment guards: %v", err)
	}
}

func TestMemStoreUpdateObservationGuardsSQLite(t *testing.T) {
	store := newMemStore(nil)
	ctx := context.Background()
	now := time.Now().UTC()

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		protocol, err := tx.CreateProtocol(domain.Protocol{Protocol: entitymodel.Protocol{Code: "OBS-PROT", Title: "Protocol", MaxSubjects: 1}})
		if err != nil {
			return err
		}
		procedure, err := tx.CreateProcedure(domain.Procedure{Procedure: entitymodel.Procedure{Name: "ObsProc", Status: domain.ProcedureStatusScheduled, ScheduledAt: now, ProtocolID: protocol.ID}})
		if err != nil {
			return err
		}
		organism, err := tx.CreateOrganism(domain.Organism{Organism: entitymodel.Organism{Name: "ObsOrg", Species: "Spec"}})
		if err != nil {
			return err
		}
		cohort, err := tx.CreateCohort(domain.Cohort{Cohort: entitymodel.Cohort{Name: "ObsCohort"}})
		if err != nil {
			return err
		}

		observation, err := tx.CreateObservation(domain.Observation{Observation: entitymodel.Observation{ID: "obs-guard", ProcedureID: &procedure.ID, RecordedAt: now, Observer: "tech"}})
		if err != nil {
			return err
		}

		if _, err := tx.UpdateObservation(observation.ID, func(o *domain.Observation) error {
			o.ProcedureID = nil
			o.OrganismID = nil
			o.CohortID = nil
			return nil
		}); err == nil {
			return fmt.Errorf("expected observation reference guard")
		}

		missingProc := "missing-proc"
		if _, err := tx.UpdateObservation(observation.ID, func(o *domain.Observation) error {
			o.ProcedureID = &missingProc
			return nil
		}); err == nil {
			return fmt.Errorf("expected missing procedure guard")
		}

		if _, err := tx.UpdateObservation(observation.ID, func(o *domain.Observation) error {
			o.ProcedureID = &procedure.ID
			o.OrganismID = ptr("missing-organism")
			return nil
		}); err == nil {
			return fmt.Errorf("expected missing organism guard")
		}

		if _, err := tx.UpdateObservation(observation.ID, func(o *domain.Observation) error {
			o.ProcedureID = &procedure.ID
			o.OrganismID = &organism.ID
			o.CohortID = &cohort.ID
			o.Notes = ptr("ok")
			return nil
		}); err != nil {
			return err
		}
		return nil
	}); err != nil {
		t.Fatalf("update observation guards: %v", err)
	}
}

func TestMemStoreTransactionMissingFindersSQLite(t *testing.T) {
	store := newMemStore(nil)
	if _, err := store.RunInTransaction(context.Background(), func(tx domain.Transaction) error {
		if _, ok := tx.FindFacility("missing"); ok {
			return fmt.Errorf("expected missing facility")
		}
		if _, ok := tx.FindLine("missing"); ok {
			return fmt.Errorf("expected missing line")
		}
		if _, ok := tx.FindStrain("missing"); ok {
			return fmt.Errorf("expected missing strain")
		}
		if _, ok := tx.FindGenotypeMarker("missing"); ok {
			return fmt.Errorf("expected missing marker")
		}
		return nil
	}); err != nil {
		t.Fatalf("missing finders: %v", err)
	}
}

func TestMemStoreDeleteOrganismAndCohortGuardsSQLite(t *testing.T) {
	store := newMemStore(nil)
	ctx := context.Background()
	now := time.Now().UTC()

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		facility, err := tx.CreateFacility(domain.Facility{Facility: entitymodel.Facility{Name: "Lab"}})
		if err != nil {
			return err
		}
		cohort, err := tx.CreateCohort(domain.Cohort{Cohort: entitymodel.Cohort{Name: "Cohort"}})
		if err != nil {
			return err
		}
		organism, err := tx.CreateOrganism(domain.Organism{Organism: entitymodel.Organism{Name: "Org", Species: "Spec"}})
		if err != nil {
			return err
		}
		sample, err := tx.CreateSample(domain.Sample{Sample: entitymodel.Sample{Identifier: "S-org",
			SourceType:      "organism",
			OrganismID:      &organism.ID,
			CohortID:        &cohort.ID,
			FacilityID:      facility.ID,
			CollectedAt:     now,
			Status:          domain.SampleStatusStored,
			StorageLocation: "cold",
			ChainOfCustody: []domain.SampleCustodyEvent{{
				Actor:     "tech",
				Location:  "cold",
				Timestamp: now,
			}}},
		})
		if err != nil {
			return err
		}

		if err := tx.DeleteOrganism(organism.ID); err == nil {
			return fmt.Errorf("expected organism delete to fail while sample references it")
		}
		if err := tx.DeleteCohort(cohort.ID); err == nil {
			return fmt.Errorf("expected cohort delete to fail while sample references it")
		}

		if err := tx.DeleteSample(sample.ID); err != nil {
			return err
		}
		if err := tx.DeleteOrganism(organism.ID); err != nil {
			return err
		}
		return tx.DeleteCohort(cohort.ID)
	}); err != nil {
		t.Fatalf("organism/cohort delete guards: %v", err)
	}
}

func TestMemStoreCreateStrainGuardsSQLite(t *testing.T) {
	store := newMemStore(nil)
	if _, err := store.RunInTransaction(context.Background(), func(tx domain.Transaction) error {
		if _, err := tx.CreateStrain(domain.Strain{Strain: entitymodel.Strain{Code: "S-missing", Name: "Strain"}}); err == nil {
			return fmt.Errorf("expected missing line id error")
		}
		if _, err := tx.CreateStrain(domain.Strain{Strain: entitymodel.Strain{Code: "S-missing-line", Name: "Strain", LineID: "missing"}}); err == nil {
			return fmt.Errorf("expected missing line reference error")
		}
		return nil
	}); err != nil {
		t.Fatalf("create strain guards: %v", err)
	}
}
