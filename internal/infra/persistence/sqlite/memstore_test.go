package sqlite

import (
	"colonycore/pkg/domain"
	"context"
	"fmt"
	"testing"
	"time"
)

// Migrated minimal representative tests; original exhaustive tests remain at old path until cleanup.

func TestMemStoreBasicLifecycle(t *testing.T) {
	store := newMemStore(nil)
	ctx := context.Background()
	if store.NowFunc() == nil {
		t.Fatalf("expected NowFunc to be initialized")
	}
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		_, err := tx.CreateOrganism(domain.Organism{Name: "Specimen", Species: "Test"})
		return err
	}); err != nil {
		t.Fatalf("create organism: %v", err)
	}
	if len(store.ListOrganisms()) != 1 {
		t.Fatalf("expected 1 organism")
	}
	snapshot := store.ExportState()
	store.ImportState(Snapshot{})
	if len(store.ListOrganisms()) != 0 {
		t.Fatalf("expected cleared state")
	}
	store.ImportState(snapshot)
	if len(store.ListOrganisms()) != 1 {
		t.Fatalf("expected restored organism")
	}
}

func TestMemStoreRuleViolation(t *testing.T) {
	store := newMemStore(domain.NewRulesEngine())
	store.RulesEngine().Register(blockingRule{})
	if _, err := store.RunInTransaction(context.Background(), func(tx domain.Transaction) error {
		_, e := tx.CreateOrganism(domain.Organism{Name: "Fail"})
		return e
	}); err == nil {
		t.Fatalf("expected violation error")
	}
}

type blockingRule struct{}

func (blockingRule) Name() string { return "block" }
func (blockingRule) Evaluate(_ context.Context, _ domain.RuleView, _ []domain.Change) (domain.Result, error) {
	r := domain.Result{}
	r.Merge(domain.Result{Violations: []domain.Violation{{Rule: "block", Severity: domain.SeverityBlock}}})
	return r, nil
}

func TestMemStoreCRUDReduced(t *testing.T) {
	store := newMemStore(nil)
	ctx := context.Background()
	var projectID string
	const updatedDesc = "updated"
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		proj, err := tx.CreateProject(domain.Project{Code: "PRJ", Title: "Project"})
		if err != nil {
			return err
		}
		projectID = proj.ID
		if _, err := tx.CreateOrganism(domain.Organism{Name: "Alpha", Species: "Frog"}); err != nil {
			return err
		}
		return nil
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if got := len(store.ListProjects()); got != 1 {
		t.Fatalf("expected 1 project, got %d", got)
	}
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		if _, err := tx.UpdateProject(projectID, func(p *domain.Project) error {
			p.Description = strPtr(updatedDesc)
			return nil
		}); err != nil {
			return err
		}
		return tx.DeleteProject(projectID)
	}); err != nil {
		t.Fatalf("mutate: %v", err)
	}
}

func TestMemStoreProcedureLifecycleReduced(t *testing.T) {
	store := newMemStore(nil)
	ctx := context.Background()
	now := time.Now().UTC()
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		prot, err := tx.CreateProtocol(domain.Protocol{Code: "P", Title: "Proto", MaxSubjects: 5})
		if err != nil {
			return err
		}
		_, err = tx.CreateProcedure(domain.Procedure{Name: "Check", Status: domain.ProcedureStatusScheduled, ScheduledAt: now, ProtocolID: prot.ID})
		return err
	}); err != nil {
		t.Fatalf("create procedure: %v", err)
	}
	if got := len(store.ListProcedures()); got != 1 {
		t.Fatalf("expected one procedure, got %d", got)
	}
}

func TestMemStoreViewSnapshotReduced(t *testing.T) {
	store := newMemStore(nil)
	ctx := context.Background()
	if err := store.View(ctx, func(v domain.TransactionView) error {
		if len(v.ListOrganisms()) != 0 {
			return fmt.Errorf("expected empty")
		}
		return nil
	}); err != nil {
		t.Fatalf("view: %v", err)
	}
}

func TestMigrateSnapshotRelationships(t *testing.T) {
	store := newMemStore(nil)
	now := time.Now().UTC()

	organisms := map[string]domain.Organism{
		"org-1": {Base: domain.Base{ID: "org-1"}, Name: "Org", Species: "Spec"},
	}
	cohorts := map[string]domain.Cohort{
		"cohort-1": {Base: domain.Base{ID: "cohort-1"}, Name: "Cohort"},
	}
	protocols := map[string]domain.Protocol{
		"prot-1": {Base: domain.Base{ID: "prot-1"}, Code: "PR", Title: "Protocol", MaxSubjects: 10, Status: domain.ProtocolStatusApproved},
	}
	procedures := map[string]domain.Procedure{
		"proc-1": {Base: domain.Base{ID: "proc-1"}, Name: "Proc", Status: domain.ProcedureStatusScheduled, ScheduledAt: now, ProtocolID: "prot-1", OrganismIDs: []string{"org-1"}},
	}

	snapshot := Snapshot{
		Organisms: organisms,
		Cohorts:   cohorts,
		Housing: map[string]domain.HousingUnit{
			"house-1": {Base: domain.Base{ID: "house-1"}, Name: "Housing", FacilityID: "fac-1", Capacity: 2},
		},
		Facilities: map[string]domain.Facility{
			"fac-1": {Base: domain.Base{ID: "fac-1"}, Name: "Facility"},
		},
		Procedures: procedures,
		Treatments: map[string]domain.Treatment{
			"treat-1": {Base: domain.Base{ID: "treat-1"}, Name: "Treat", Status: domain.TreatmentStatusPlanned, ProcedureID: "proc-1", OrganismIDs: []string{"org-1", "org-1"}, CohortIDs: []string{"missing"}},
		},
		Observations: map[string]domain.Observation{
			"obs-1": {Base: domain.Base{ID: "obs-1"}, ProcedureID: ptr("proc-1"), Observer: "Tech", RecordedAt: now},
		},
		Samples: map[string]domain.Sample{
			"sample-1": {Base: domain.Base{ID: "sample-1"}, Identifier: "S1", SourceType: "blood", FacilityID: "fac-1", OrganismID: ptr("org-1"), CollectedAt: now, Status: domain.SampleStatusStored, StorageLocation: "freezer"},
		},
		Protocols: protocols,
		Permits: map[string]domain.Permit{
			"permit-1": {Base: domain.Base{ID: "permit-1"}, PermitNumber: "P1", Authority: "Gov", Status: domain.PermitStatusApproved, ValidFrom: now, ValidUntil: now.AddDate(1, 0, 0), FacilityIDs: []string{"fac-1", "fac-1"}, ProtocolIDs: []string{"prot-1"}},
		},
		Projects: map[string]domain.Project{
			"proj-1": {Base: domain.Base{ID: "proj-1"}, Code: "P1", Title: "Project", FacilityIDs: []string{"fac-1", "fac-1"}},
		},
		Supplies: map[string]domain.SupplyItem{
			"supply-1": {Base: domain.Base{ID: "supply-1"}, SKU: "SKU", Name: "Gloves", QuantityOnHand: 5, Unit: "box", FacilityIDs: []string{"fac-1"}, ProjectIDs: []string{"proj-1", "proj-1"}},
		},
	}

	store.ImportState(snapshot)

	facility, ok := store.GetFacility("fac-1")
	if !ok {
		t.Fatalf("expected facility present")
	}
	if len(facility.HousingUnitIDs) != 1 || facility.HousingUnitIDs[0] != "house-1" {
		t.Fatalf("expected facility housing ids migrated, got %+v", facility.HousingUnitIDs)
	}
	if len(facility.ProjectIDs) != 1 || facility.ProjectIDs[0] != "proj-1" {
		t.Fatalf("expected facility project ids migrated, got %+v", facility.ProjectIDs)
	}

	if treatments := store.ListTreatments(); len(treatments) != 1 || len(treatments[0].OrganismIDs) != 1 || treatments[0].OrganismIDs[0] != "org-1" {
		t.Fatalf("expected deduped treatment organism ids, got %+v", treatments)
	}

	if permits := store.ListPermits(); len(permits) != 1 || len(permits[0].FacilityIDs) != 1 || permits[0].FacilityIDs[0] != "fac-1" {
		t.Fatalf("expected permit facility ids filtered, got %+v", permits)
	}

	if supplies := store.ListSupplyItems(); len(supplies) != 1 || len(supplies[0].ProjectIDs) != 1 || supplies[0].ProjectIDs[0] != "proj-1" {
		t.Fatalf("expected supply project ids filtered, got %+v", supplies)
	}
}

func TestMemStoreStateNormalization(t *testing.T) {
	store := newMemStore(nil)
	now := time.Now().UTC()
	if _, err := store.RunInTransaction(context.Background(), func(tx domain.Transaction) error {
		facility, err := tx.CreateFacility(domain.Facility{Name: "Lab"})
		if err != nil {
			return err
		}
		housing, err := tx.CreateHousingUnit(domain.HousingUnit{Name: "H", FacilityID: facility.ID, Capacity: 1})
		if err != nil {
			return err
		}
		if housing.State != domain.HousingStateQuarantine || housing.Environment != domain.HousingEnvironmentTerrestrial {
			return fmt.Errorf("expected housing defaults applied, got state=%q env=%q", housing.State, housing.Environment)
		}
		if _, err := tx.CreateHousingUnit(domain.HousingUnit{Name: "InvalidEnv", FacilityID: facility.ID, Capacity: 1, Environment: domain.HousingEnvironment("invalid")}); err == nil {
			return fmt.Errorf("expected invalid housing environment to error")
		}

		protocol, err := tx.CreateProtocol(domain.Protocol{Code: "P", Title: "Proto", MaxSubjects: 1})
		if err != nil {
			return err
		}
		if protocol.Status != domain.ProtocolStatusDraft {
			return fmt.Errorf("expected protocol status defaulted, got %q", protocol.Status)
		}
		if _, err := tx.CreateProtocol(domain.Protocol{Code: "P2", Title: "Invalid", MaxSubjects: 1, Status: domain.ProtocolStatus("invalid")}); err == nil {
			return fmt.Errorf("expected invalid protocol status to error")
		}

		permit, err := tx.CreatePermit(domain.Permit{
			PermitNumber:      "PER",
			Authority:         "Gov",
			ValidFrom:         now,
			ValidUntil:        now.Add(time.Hour),
			AllowedActivities: []string{"store"},
			FacilityIDs:       []string{facility.ID},
		})
		if err != nil {
			return err
		}
		if permit.Status != domain.PermitStatusDraft {
			return fmt.Errorf("expected permit status defaulted, got %q", permit.Status)
		}
		if _, err := tx.CreatePermit(domain.Permit{
			PermitNumber:      "PER-2",
			Authority:         "Gov",
			Status:            domain.PermitStatus("invalid"),
			ValidFrom:         now,
			ValidUntil:        now.Add(time.Hour),
			AllowedActivities: []string{"store"},
			FacilityIDs:       []string{facility.ID},
		}); err == nil {
			return fmt.Errorf("expected invalid permit status to error")
		}
		if _, err := tx.UpdateHousingUnit(housing.ID, func(h *domain.HousingUnit) error {
			h.Environment = domain.HousingEnvironment("invalid")
			return nil
		}); err == nil {
			return fmt.Errorf("expected invalid housing environment on update to error")
		}
		if _, err := tx.UpdateProtocol(protocol.ID, func(p *domain.Protocol) error {
			p.Status = domain.ProtocolStatus("invalid")
			return nil
		}); err == nil {
			return fmt.Errorf("expected invalid protocol status on update to error")
		}
		if _, err := tx.UpdatePermit(permit.ID, func(p *domain.Permit) error {
			p.Status = domain.PermitStatus("invalid")
			return nil
		}); err == nil {
			return fmt.Errorf("expected invalid permit status on update to error")
		}
		return nil
	}); err != nil {
		t.Fatalf("transaction error: %v", err)
	}
}

func TestMemStoreTransactionViewMissingFinders(t *testing.T) {
	store := newMemStore(nil)
	if err := store.View(context.Background(), func(v domain.TransactionView) error {
		if _, ok := v.FindOrganism("missing"); ok {
			t.Fatalf("expected missing organism")
		}
		if _, ok := v.FindTreatment("missing"); ok {
			t.Fatalf("expected missing treatment")
		}
		if _, ok := v.FindObservation("missing"); ok {
			t.Fatalf("expected missing observation")
		}
		if _, ok := v.FindPermit("missing"); ok {
			t.Fatalf("expected missing permit")
		}
		if _, ok := v.FindSupplyItem("missing"); ok {
			t.Fatalf("expected missing supply item")
		}
		if _, ok := v.FindHousingUnit("missing"); ok {
			t.Fatalf("expected missing housing unit")
		}
		return nil
	}); err != nil {
		t.Fatalf("view error: %v", err)
	}
}

func TestMemStoreProcedureObservationSampleLifecycle(t *testing.T) {
	store := newMemStore(nil)
	now := time.Now().UTC()
	if _, err := store.RunInTransaction(context.Background(), func(tx domain.Transaction) error {
		facility, err := tx.CreateFacility(domain.Facility{Name: "Lab-2"})
		if err != nil {
			return err
		}
		housing, err := tx.CreateHousingUnit(domain.HousingUnit{Name: "H2", FacilityID: facility.ID, Capacity: 1, Environment: domain.HousingEnvironmentTerrestrial})
		if err != nil {
			return err
		}
		cohort, err := tx.CreateCohort(domain.Cohort{Name: "C2"})
		if err != nil {
			return err
		}
		housingID := housing.ID
		organism, err := tx.CreateOrganism(domain.Organism{Name: "Org", Species: "Spec", HousingID: &housingID})
		if err != nil {
			return err
		}
		protocol, err := tx.CreateProtocol(domain.Protocol{Code: "PR-2", Title: "Protocol 2", MaxSubjects: 1, Status: domain.ProtocolStatusApproved})
		if err != nil {
			return err
		}
		procedure, err := tx.CreateProcedure(domain.Procedure{
			Name:        "Proc",
			Status:      domain.ProcedureStatusScheduled,
			ScheduledAt: now,
			ProtocolID:  protocol.ID,
			OrganismIDs: []string{organism.ID},
		})
		if err != nil {
			return err
		}
		treatment, err := tx.CreateTreatment(domain.Treatment{
			Name:              "Treat",
			Status:            domain.TreatmentStatusPlanned,
			ProcedureID:       procedure.ID,
			OrganismIDs:       []string{organism.ID},
			AdministrationLog: []string{},
			AdverseEvents:     []string{},
		})
		if err != nil {
			return err
		}
		if _, err := tx.UpdateTreatment(treatment.ID, func(t *domain.Treatment) error {
			t.Status = domain.TreatmentStatusCompleted
			return nil
		}); err != nil {
			return err
		}
		observation, err := tx.CreateObservation(domain.Observation{
			ProcedureID: &procedure.ID,
			OrganismID:  &organism.ID,
			CohortID:    &cohort.ID,
			Observer:    "Tech",
			RecordedAt:  now,
		})
		if err != nil {
			return err
		}
		if _, err := tx.UpdateObservation(observation.ID, func(o *domain.Observation) error {
			note := "updated"
			o.Notes = &note
			return nil
		}); err != nil {
			return err
		}
		sample, err := tx.CreateSample(domain.Sample{
			Identifier:      "S-1",
			SourceType:      "blood",
			FacilityID:      facility.ID,
			CollectedAt:     now,
			Status:          domain.SampleStatusStored,
			StorageLocation: "loc",
			OrganismID:      &organism.ID,
		})
		if err != nil {
			return err
		}
		if _, err := tx.UpdateSample(sample.ID, func(s *domain.Sample) error {
			s.Status = domain.SampleStatusInTransit
			return nil
		}); err != nil {
			return err
		}
		if err := tx.DeleteProcedure(procedure.ID); err == nil {
			return fmt.Errorf("expected delete procedure to fail while referenced")
		}
		if err := tx.DeleteTreatment(treatment.ID); err != nil {
			return err
		}
		if err := tx.DeleteObservation(observation.ID); err != nil {
			return err
		}
		if err := tx.DeleteProcedure(procedure.ID); err != nil {
			return err
		}
		return nil
	}); err != nil {
		t.Fatalf("transaction error: %v", err)
	}
}

func ptr[T any](v T) *T { return &v }
