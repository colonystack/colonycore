package sqlite

import (
	"colonycore/pkg/domain"
	entitymodel "colonycore/pkg/domain/entitymodel"
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func TestSQLiteStoreNormalizesLifecycleStatuses(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "state.db"), domain.NewRulesEngine())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { _ = store.DB().Close() })

	now := time.Now().UTC()
	if _, err := store.RunInTransaction(context.Background(), func(tx domain.Transaction) error {
		facility, err := tx.CreateFacility(domain.Facility{Facility: entitymodel.Facility{
			Code:         "FAC-NORM",
			Name:         "Facility",
			Zone:         "Z",
			AccessPolicy: "all",
		}})
		if err != nil {
			return err
		}
		project, err := tx.CreateProject(domain.Project{Project: entitymodel.Project{
			Code:        "PRJ-NORM",
			Title:       "Project",
			FacilityIDs: []string{facility.ID},
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
		protocol, err := tx.CreateProtocol(domain.Protocol{Protocol: entitymodel.Protocol{
			Code:        "PROT-NORM",
			Title:       "Protocol",
			MaxSubjects: 1,
		}})
		if err != nil {
			return err
		}
		projectID := project.ID
		housingID := housing.ID
		protocolID := protocol.ID

		cohort, err := tx.CreateCohort(domain.Cohort{Cohort: entitymodel.Cohort{
			Name:       "Cohort",
			Purpose:    "purpose",
			ProjectID:  &projectID,
			HousingID:  &housingID,
			ProtocolID: &protocolID,
		}})
		if err != nil {
			return err
		}
		cohortID := cohort.ID

		organism, err := tx.CreateOrganism(domain.Organism{Organism: entitymodel.Organism{
			Name:       "Org",
			Species:    "species",
			Stage:      domain.StageJuvenile,
			HousingID:  &housingID,
			CohortID:   &cohortID,
			ProtocolID: &protocolID,
		}})
		if err != nil {
			return err
		}

		procedure, err := tx.CreateProcedure(domain.Procedure{Procedure: entitymodel.Procedure{
			Name:        "Procedure",
			ProtocolID:  protocol.ID,
			ScheduledAt: now,
		}})
		if err != nil {
			return err
		}
		if procedure.Status != domain.ProcedureStatusScheduled {
			return fmt.Errorf("expected procedure status defaulted, got %s", procedure.Status)
		}

		procedure, err = tx.UpdateProcedure(procedure.ID, func(p *domain.Procedure) error {
			p.Status = ""
			return nil
		})
		if err != nil {
			return err
		}
		if procedure.Status != domain.ProcedureStatusScheduled {
			return fmt.Errorf("expected procedure status to default on update, got %s", procedure.Status)
		}

		treatment, err := tx.CreateTreatment(domain.Treatment{Treatment: entitymodel.Treatment{
			Name:        "Treatment",
			ProcedureID: procedure.ID,
			DosagePlan:  "dose",
			OrganismIDs: []string{organism.ID},
		}})
		if err != nil {
			return err
		}
		if treatment.Status != domain.TreatmentStatusPlanned {
			return fmt.Errorf("expected treatment status defaulted, got %s", treatment.Status)
		}

		treatment, err = tx.UpdateTreatment(treatment.ID, func(t *domain.Treatment) error {
			t.Status = ""
			return nil
		})
		if err != nil {
			return err
		}
		if treatment.Status != domain.TreatmentStatusPlanned {
			return fmt.Errorf("expected treatment status to default on update, got %s", treatment.Status)
		}

		sample, err := tx.CreateSample(domain.Sample{Sample: entitymodel.Sample{
			Identifier:      "S-NORM",
			SourceType:      "blood",
			FacilityID:      facility.ID,
			OrganismID:      &organism.ID,
			CollectedAt:     now,
			StorageLocation: "cold",
			AssayType:       "assay",
			ChainOfCustody:  []domain.SampleCustodyEvent{{Actor: "tech", Location: "cold", Timestamp: now}},
		}})
		if err != nil {
			return err
		}
		if sample.Status != domain.SampleStatusStored {
			return fmt.Errorf("expected sample status defaulted, got %s", sample.Status)
		}

		sample, err = tx.UpdateSample(sample.ID, func(s *domain.Sample) error {
			s.Status = ""
			return nil
		})
		if err != nil {
			return err
		}
		if sample.Status != domain.SampleStatusStored {
			return fmt.Errorf("expected sample status to default on update, got %s", sample.Status)
		}
		return nil
	}); err != nil {
		t.Fatalf("RunInTransaction: %v", err)
	}
}

func TestSQLiteStoreRejectsInvalidLifecycleStatuses(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "invalid.db"), domain.NewRulesEngine())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { _ = store.DB().Close() })

	now := time.Now().UTC()
	if _, err := store.RunInTransaction(context.Background(), func(tx domain.Transaction) error {
		facility, err := tx.CreateFacility(domain.Facility{Facility: entitymodel.Facility{
			Code:         "FAC-INV",
			Name:         "Facility",
			Zone:         "Z",
			AccessPolicy: "all",
		}})
		if err != nil {
			return err
		}
		project, err := tx.CreateProject(domain.Project{Project: entitymodel.Project{
			Code:        "PRJ-INV",
			Title:       "Project",
			FacilityIDs: []string{facility.ID},
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
		protocol, err := tx.CreateProtocol(domain.Protocol{Protocol: entitymodel.Protocol{
			Code:        "PROT-INV",
			Title:       "Protocol",
			MaxSubjects: 1,
		}})
		if err != nil {
			return err
		}
		projectID := project.ID
		housingID := housing.ID
		protocolID := protocol.ID

		cohort, err := tx.CreateCohort(domain.Cohort{Cohort: entitymodel.Cohort{
			Name:       "Cohort",
			Purpose:    "purpose",
			ProjectID:  &projectID,
			HousingID:  &housingID,
			ProtocolID: &protocolID,
		}})
		if err != nil {
			return err
		}
		cohortID := cohort.ID

		organism, err := tx.CreateOrganism(domain.Organism{Organism: entitymodel.Organism{
			Name:       "Org",
			Species:    "species",
			Stage:      domain.StageAdult,
			HousingID:  &housingID,
			CohortID:   &cohortID,
			ProtocolID: &protocolID,
		}})
		if err != nil {
			return err
		}

		if _, err := tx.CreateProcedure(domain.Procedure{Procedure: entitymodel.Procedure{
			Name:        "InvalidProcedure",
			ProtocolID:  protocol.ID,
			ScheduledAt: now,
			Status:      domain.ProcedureStatus("invalid"),
		}}); err == nil {
			return fmt.Errorf("expected invalid procedure status to error")
		}

		procedure, err := tx.CreateProcedure(domain.Procedure{Procedure: entitymodel.Procedure{
			Name:        "Procedure",
			ProtocolID:  protocol.ID,
			ScheduledAt: now,
		}})
		if err != nil {
			return err
		}

		if _, err := tx.CreateTreatment(domain.Treatment{Treatment: entitymodel.Treatment{
			Name:        "InvalidTreatment",
			ProcedureID: procedure.ID,
			Status:      domain.TreatmentStatus("invalid"),
			DosagePlan:  "dose",
		}}); err == nil {
			return fmt.Errorf("expected invalid treatment status to error")
		}

		if _, err := tx.CreateSample(domain.Sample{Sample: entitymodel.Sample{
			Identifier:      "S-INV",
			SourceType:      "blood",
			FacilityID:      facility.ID,
			OrganismID:      &organism.ID,
			CollectedAt:     now,
			Status:          domain.SampleStatus("invalid"),
			StorageLocation: "cold",
			AssayType:       "assay",
			ChainOfCustody:  []domain.SampleCustodyEvent{{Actor: "tech", Location: "cold", Timestamp: now}},
		}}); err == nil {
			return fmt.Errorf("expected invalid sample status to error")
		}

		return nil
	}); err != nil {
		t.Fatalf("RunInTransaction: %v", err)
	}
}
