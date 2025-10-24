package sqlite

import (
	"colonycore/pkg/domain"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func strPtr(v string) *string {
	return &v
}

// helper to run a transaction and fail fast
func runTx(t *testing.T, store *memStore, fn func(tx domain.Transaction) error) domain.Result {
	t.Helper()
	res, err := store.RunInTransaction(context.Background(), fn)
	if err != nil {
		t.Fatalf("transaction failed: %v", err)
	}
	return res
}

func TestMemStore_FullCRUDAndErrors(t *testing.T) { //nolint:gocyclo // exhaustive scenario coverage in a single integration-style test
	store := newMemStore(nil)
	ctx := context.Background()

	var (
		orgA, orgB  domain.Organism
		cohort      domain.Cohort
		housing     domain.HousingUnit
		breeding    domain.BreedingUnit
		protocol    domain.Protocol
		procedure   domain.Procedure
		treatment   domain.Treatment
		observation domain.Observation
		sample      domain.Sample
		permit      domain.Permit
		facility    domain.Facility
		supply      domain.SupplyItem
		project     domain.Project
	)

	// Create all entities
	runTx(t, store, func(tx domain.Transaction) error {
		o1, _ := tx.CreateOrganism(domain.Organism{Name: "Alpha", Species: "Frog", Attributes: map[string]any{"a": 1}})
		o2, _ := tx.CreateOrganism(domain.Organism{Name: "Beta", Species: "Frog"})
		orgA, orgB = o1, o2
		c, _ := tx.CreateCohort(domain.Cohort{Name: "C1", Purpose: "testing"})
		cohort = c
		pj, _ := tx.CreateProject(domain.Project{Code: "PRJ1", Title: "Proj"})
		project = pj
		f, _ := tx.CreateFacility(domain.Facility{
			Name:                 "Vivarium",
			Zone:                 "Zone-A",
			AccessPolicy:         "badge",
			EnvironmentBaselines: map[string]any{"temperature": "22C"},
			ProjectIDs:           []string{project.ID},
		})
		facility = f
		h, _ := tx.CreateHousingUnit(domain.HousingUnit{Name: "H1", Capacity: 2, Environment: "dry", FacilityID: facility.ID})
		housing = h
		_, _ = tx.UpdateFacility(facility.ID, func(fc *domain.Facility) error {
			fc.HousingUnitIDs = append(fc.HousingUnitIDs, housing.ID)
			return nil
		})
		b, _ := tx.CreateBreedingUnit(domain.BreedingUnit{Name: "B1", FemaleIDs: []string{o1.ID}, MaleIDs: []string{o2.ID}})
		breeding = b
		p, _ := tx.CreateProtocol(domain.Protocol{Code: "P1", Title: "Proto", MaxSubjects: 10})
		protocol = p
		pr, _ := tx.CreateProcedure(domain.Procedure{Name: "Proc", Status: domain.ProcedureStatusScheduled, ProtocolID: protocol.ID, OrganismIDs: []string{o1.ID}, ScheduledAt: time.Now().UTC()})
		procedure = pr
		t, _ := tx.CreateTreatment(domain.Treatment{Name: "Dose", Status: domain.TreatmentStatusPlanned, ProcedureID: procedure.ID, OrganismIDs: []string{o1.ID}, CohortIDs: []string{cohort.ID}, DosagePlan: "10mg"})
		treatment = t
		now := time.Now().UTC()
		ob, _ := tx.CreateObservation(domain.Observation{ProcedureID: &procedure.ID, OrganismID: &o1.ID, RecordedAt: now, Observer: "tech", Data: map[string]any{"score": 5}})
		observation = ob
		custody := []domain.SampleCustodyEvent{{Actor: "tech", Location: "bench", Timestamp: now}}
		sa, _ := tx.CreateSample(domain.Sample{
			Identifier:      "S1",
			SourceType:      "blood",
			OrganismID:      &o1.ID,
			FacilityID:      facility.ID,
			CollectedAt:     now,
			Status:          domain.SampleStatusStored,
			StorageLocation: "freezer",
			AssayType:       "PCR",
			ChainOfCustody:  custody,
			Attributes:      map[string]any{"volume_ml": 1.0},
		})
		sample = sa
		per, _ := tx.CreatePermit(domain.Permit{
			PermitNumber:      "PER-1",
			Authority:         "Agency",
			Status:            domain.PermitStatusActive,
			ValidFrom:         now.Add(-time.Hour),
			ValidUntil:        now.Add(time.Hour),
			AllowedActivities: []string{"collect"},
			FacilityIDs:       []string{facility.ID},
			ProtocolIDs:       []string{protocol.ID},
			Notes:             strPtr("issue"),
		})
		permit = per
		expiry := time.Now().Add(48 * time.Hour)
		s, _ := tx.CreateSupplyItem(domain.SupplyItem{
			SKU:            "SKU1",
			Name:           "Feed",
			Description:    strPtr("daily feed"),
			QuantityOnHand: 50,
			Unit:           "grams",
			LotNumber:      strPtr("LOT-1"),
			ExpiresAt:      &expiry,
			FacilityIDs:    []string{facility.ID},
			ProjectIDs:     []string{project.ID},
			ReorderLevel:   10,
			Attributes:     map[string]any{"supplier": "Acme"},
		})
		supply = s
		if _, ok := tx.FindFacility(facility.ID); !ok {
			return fmt.Errorf("expected facility lookup success")
		}
		if _, ok := tx.FindTreatment(treatment.ID); !ok {
			return fmt.Errorf("expected treatment lookup success")
		}
		if _, ok := tx.FindTreatment("missing-treatment"); ok {
			return fmt.Errorf("unexpected treatment lookup success")
		}
		if _, ok := tx.FindObservation(observation.ID); !ok {
			return fmt.Errorf("expected observation lookup success")
		}
		if _, ok := tx.FindObservation("missing-observation"); ok {
			return fmt.Errorf("unexpected observation lookup success")
		}
		if _, ok := tx.FindSample(sample.ID); !ok {
			return fmt.Errorf("expected sample lookup success")
		}
		if _, ok := tx.FindSample("missing-sample"); ok {
			return fmt.Errorf("unexpected sample lookup success")
		}
		if _, ok := tx.FindPermit(permit.ID); !ok {
			return fmt.Errorf("expected permit lookup success")
		}
		if _, ok := tx.FindPermit("missing-permit"); ok {
			return fmt.Errorf("unexpected permit lookup success")
		}
		if _, ok := tx.FindSupplyItem(supply.ID); !ok {
			return fmt.Errorf("expected supply item lookup success")
		}
		if _, ok := tx.FindSupplyItem("missing-supply"); ok {
			return fmt.Errorf("unexpected supply lookup success")
		}
		return nil
	})

	// Direct getters to cover new accessors
	if got, ok := store.GetOrganism(orgA.ID); !ok || got.Name != "Alpha" {
		t.Fatalf("GetOrganism mismatch")
	}
	if got, ok := store.GetHousingUnit(housing.ID); !ok || got.Name != "H1" {
		t.Fatalf("GetHousingUnit mismatch")
	}
	if _, ok := store.GetFacility(facility.ID); !ok {
		t.Fatalf("expected facility getter success")
	}
	if _, ok := store.GetPermit(permit.ID); !ok {
		t.Fatalf("expected permit getter success")
	}
	if len(store.ListProtocols()) != 1 || len(store.ListTreatments()) != 1 || len(store.ListObservations()) != 1 || len(store.ListSamples()) != 1 || len(store.ListFacilities()) != 1 || len(store.ListSupplyItems()) != 1 || len(store.ListPermits()) != 1 || len(store.ListProjects()) != 1 {
		t.Fatalf("unexpected list counts")
	}

	if err := store.View(ctx, func(view domain.TransactionView) error {
		if len(view.ListFacilities()) != 1 || len(view.ListTreatments()) != 1 || len(view.ListObservations()) != 1 || len(view.ListSamples()) != 1 || len(view.ListSupplyItems()) != 1 || len(view.ListPermits()) != 1 || len(view.ListProjects()) != 1 {
			return fmt.Errorf("unexpected view list counts")
		}
		if _, ok := view.FindFacility(facility.ID); !ok {
			return errors.New("facility not found in view")
		}
		if _, ok := view.FindFacility("missing"); ok {
			return errors.New("unexpected facility lookup success")
		}
		if _, ok := view.FindTreatment(treatment.ID); !ok {
			return errors.New("treatment not found in view")
		}
		if _, ok := view.FindTreatment("missing"); ok {
			return errors.New("unexpected treatment lookup success")
		}
		if _, ok := view.FindObservation(observation.ID); !ok {
			return errors.New("observation not found in view")
		}
		if _, ok := view.FindObservation("missing"); ok {
			return errors.New("unexpected observation lookup success")
		}
		if _, ok := view.FindSample(sample.ID); !ok {
			return errors.New("sample not found in view")
		}
		if _, ok := view.FindSample("missing"); ok {
			return errors.New("unexpected sample lookup success")
		}
		if _, ok := view.FindPermit(permit.ID); !ok {
			return errors.New("permit not found in view")
		}
		if _, ok := view.FindPermit("missing"); ok {
			return errors.New("unexpected permit lookup success")
		}
		if _, ok := view.FindSupplyItem(supply.ID); !ok {
			return errors.New("supply not found in view")
		}
		if _, ok := view.FindSupplyItem("missing"); ok {
			return errors.New("unexpected supply lookup success")
		}
		return nil
	}); err != nil {
		t.Fatalf("view validation: %v", err)
	}

	// Duplicate create errors
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error { _, e := tx.CreateOrganism(orgA); return e }); err == nil {
		t.Fatalf("expected duplicate organism error")
	}
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error { _, e := tx.CreateHousingUnit(housing); return e }); err == nil {
		t.Fatalf("expected duplicate housing error")
	}

	// Update & validation errors
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		_, e := tx.UpdateOrganism("missing", func(*domain.Organism) error { return nil })
		return e
	}); err == nil {
		t.Fatalf("expected missing organism update error")
	}
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		_, e := tx.UpdateHousingUnit(housing.ID, func(h *domain.HousingUnit) error { h.Capacity = 0; return nil })
		return e
	}); err == nil {
		t.Fatalf("expected capacity validation error")
	}

	// Successful updates
	runTx(t, store, func(tx domain.Transaction) error {
		_, _ = tx.UpdateOrganism(orgA.ID, func(o *domain.Organism) error { o.Name = "Alpha2"; o.Attributes["a"] = 2; return nil })
		_, _ = tx.UpdateCohort(cohort.ID, func(c *domain.Cohort) error { c.Purpose = "updated"; return nil })
		_, _ = tx.UpdateHousingUnit(housing.ID, func(h *domain.HousingUnit) error { h.Environment = "humid"; h.Capacity = 3; return nil })
		_, _ = tx.UpdateBreedingUnit(breeding.ID, func(b *domain.BreedingUnit) error { b.FemaleIDs = append(b.FemaleIDs, orgB.ID); return nil })
		_, _ = tx.UpdateProtocol(protocol.ID, func(p *domain.Protocol) error { p.Description = strPtr("desc"); return nil })
		_, _ = tx.UpdateProcedure(procedure.ID, func(p *domain.Procedure) error { p.Status = domain.ProcedureStatusCompleted; return nil })
		_, _ = tx.UpdateProject(project.ID, func(p *domain.Project) error { p.Description = strPtr("d"); return nil })
		_, _ = tx.UpdateFacility(facility.ID, func(f *domain.Facility) error {
			f.AccessPolicy = "training"
			f.EnvironmentBaselines["humidity"] = "55%"
			return nil
		})
		_, _ = tx.UpdateTreatment(treatment.ID, func(t *domain.Treatment) error {
			t.AdministrationLog = append(t.AdministrationLog, "follow-up")
			return nil
		})
		_, _ = tx.UpdateObservation(observation.ID, func(o *domain.Observation) error { o.Notes = strPtr("checked"); o.Data["score"] = 6; return nil })
		_, _ = tx.UpdateSample(sample.ID, func(s *domain.Sample) error { s.Status = domain.SampleStatusConsumed; return nil })
		_, _ = tx.UpdatePermit(permit.ID, func(p *domain.Permit) error { p.Notes = strPtr("updated"); return nil })
		_, _ = tx.UpdateSupplyItem(supply.ID, func(s *domain.SupplyItem) error { s.QuantityOnHand = 40; return nil })
		return nil
	})

	// Snapshot export/import consistency
	snap := store.ExportState()
	if len(snap.Organisms) != 2 || len(snap.Housing) != 1 || len(snap.Breeding) != 1 || len(snap.Procedures) != 1 || len(snap.Treatments) != 1 || len(snap.Samples) != 1 || len(snap.Facilities) != 1 {
		t.Fatalf("unexpected snapshot counts: %+v", snap)
	}
	store.ImportState(Snapshot{})
	if len(store.ListOrganisms()) != 0 {
		t.Fatalf("expected cleared state after import empty")
	}
	store.ImportState(snap)
	if len(store.ListOrganisms()) != 2 {
		t.Fatalf("expected restore after import")
	}

	// Deletions and missing delete errors
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error { return tx.DeleteOrganism("missing") }); err == nil {
		t.Fatalf("expected missing delete organism")
	}
	runTx(t, store, func(tx domain.Transaction) error {
		_ = tx.DeleteObservation(observation.ID)
		_ = tx.DeleteTreatment(treatment.ID)
		_ = tx.DeleteProcedure(procedure.ID)
		_ = tx.DeleteBreedingUnit(breeding.ID)
		_ = tx.DeleteSample(sample.ID)
		_ = tx.DeletePermit(permit.ID)
		_ = tx.DeleteSupplyItem(supply.ID)
		_ = tx.DeleteHousingUnit(housing.ID)
		_ = tx.DeleteProject(project.ID)
		_ = tx.DeleteProtocol(protocol.ID)
		_ = tx.DeleteFacility(facility.ID)
		_ = tx.DeleteCohort(cohort.ID)
		_ = tx.DeleteOrganism(orgB.ID)
		return nil
	})
	if len(store.ListOrganisms()) != 1 {
		t.Fatalf("expected 1 organism left")
	}
	if len(store.ListCohorts()) != 0 {
		t.Fatalf("expected no cohorts left")
	}
	if len(store.ListHousingUnits()) != 0 {
		t.Fatalf("expected no housing units left")
	}
	if len(store.ListFacilities()) != 0 {
		t.Fatalf("expected no facilities left")
	}
	if len(store.ListBreedingUnits()) != 0 {
		t.Fatalf("expected no breeding units left")
	}
	if len(store.ListProcedures()) != 0 {
		t.Fatalf("expected no procedures left")
	}
	if len(store.ListTreatments()) != 0 {
		t.Fatalf("expected no treatments left")
	}
	if len(store.ListObservations()) != 0 {
		t.Fatalf("expected no observations left")
	}
	if len(store.ListSamples()) != 0 {
		t.Fatalf("expected no samples left")
	}
	if len(store.ListPermits()) != 0 {
		t.Fatalf("expected no permits left")
	}
	if len(store.ListSupplyItems()) != 0 {
		t.Fatalf("expected no supplies left")
	}
}

func TestMemStore_ViewAndFinds(t *testing.T) {
	store := newMemStore(nil)
	runTx(t, store, func(tx domain.Transaction) error { _, _ = tx.CreateOrganism(domain.Organism{Name: "X"}); return nil })
	if err := store.View(context.Background(), func(v domain.TransactionView) error {
		if len(v.ListOrganisms()) != 1 {
			return fmt.Errorf("expected 1 organism in view")
		}
		if _, ok := v.FindOrganism(store.ListOrganisms()[0].ID); !ok {
			return errors.New("organism not found in view")
		}
		return nil
	}); err != nil {
		t.Fatalf("view: %v", err)
	}
}

func TestSQLiteStore_Persist_Reload_Full(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.db")
	engine := domain.NewRulesEngine()
	store, err := NewStore(path, engine)
	if err != nil {
		t.Skipf("sqlite unavailable: %v", err)
	}
	ctx := context.Background()
	var (
		projectID     string
		facilityID    string
		housingID     string
		cohortID      string
		protocolID    string
		procedureID   string
		organismID    string
		treatmentID   string
		observationID string
		sampleID      string
		permitID      string
		supplyID      string
	)
	now := time.Now().UTC()
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		project, err := tx.CreateProject(domain.Project{Code: "C", Title: "T"})
		if err != nil {
			return err
		}
		projectID = project.ID

		facility, err := tx.CreateFacility(domain.Facility{
			Name:                 "Vivarium",
			Zone:                 "Zone-A",
			AccessPolicy:         "badge",
			EnvironmentBaselines: map[string]any{"temperature": "22C"},
			ProjectIDs:           []string{project.ID},
		})
		if err != nil {
			return err
		}
		facilityID = facility.ID

		housing, err := tx.CreateHousingUnit(domain.HousingUnit{Name: "Tank", Capacity: 2, Environment: "arid", FacilityID: facility.ID})
		if err != nil {
			return err
		}
		housingID = housing.ID
		if _, err := tx.UpdateFacility(facility.ID, func(f *domain.Facility) error {
			f.HousingUnitIDs = append(f.HousingUnitIDs, housing.ID)
			return nil
		}); err != nil {
			return err
		}

		protocol, err := tx.CreateProtocol(domain.Protocol{Code: "P-1", Title: "Protocol", MaxSubjects: 5})
		if err != nil {
			return err
		}
		protocolID = protocol.ID

		housingPtr := housingID
		protocolPtr := protocolID
		projectPtr := projectID
		cohort, err := tx.CreateCohort(domain.Cohort{
			Name:       "C-1",
			Purpose:    "baseline",
			ProjectID:  &projectPtr,
			HousingID:  &housingPtr,
			ProtocolID: &protocolPtr,
		})
		if err != nil {
			return err
		}
		cohortID = cohort.ID

		cohortPtr := cohortID
		organism, err := tx.CreateOrganism(domain.Organism{
			Name:       "Persisted",
			Species:    "Test",
			Stage:      domain.StageJuvenile,
			CohortID:   &cohortPtr,
			HousingID:  &housingPtr,
			ProjectID:  &projectPtr,
			ProtocolID: &protocolPtr,
			Attributes: map[string]any{"tag": "alpha"},
		})
		if err != nil {
			return err
		}
		organismID = organism.ID

		procedure, err := tx.CreateProcedure(domain.Procedure{
			Name:        "Procedure",
			Status:      domain.ProcedureStatusScheduled,
			ScheduledAt: now,
			ProtocolID:  protocol.ID,
			OrganismIDs: []string{organism.ID},
		})
		if err != nil {
			return err
		}
		procedureID = procedure.ID

		treatment, err := tx.CreateTreatment(domain.Treatment{
			Name:        "Treatment",
			Status:      domain.TreatmentStatusPlanned,
			ProcedureID: procedure.ID,
			OrganismIDs: []string{organism.ID},
			CohortIDs:   []string{cohort.ID},
			DosagePlan:  "10mg/kg",
		})
		if err != nil {
			return err
		}
		treatmentID = treatment.ID

		observation, err := tx.CreateObservation(domain.Observation{
			ProcedureID: &procedure.ID,
			OrganismID:  &organism.ID,
			RecordedAt:  now,
			Observer:    "tech",
			Data:        map[string]any{"score": 5},
		})
		if err != nil {
			return err
		}
		observationID = observation.ID

		custody := []domain.SampleCustodyEvent{{Actor: "tech", Location: "bench", Timestamp: now}}
		sample, err := tx.CreateSample(domain.Sample{
			Identifier:      "S-1",
			SourceType:      "blood",
			OrganismID:      &organism.ID,
			FacilityID:      facility.ID,
			CollectedAt:     now,
			Status:          domain.SampleStatusStored,
			StorageLocation: "freezer",
			AssayType:       "PCR",
			ChainOfCustody:  custody,
			Attributes:      map[string]any{"volume_ml": 1.0},
		})
		if err != nil {
			return err
		}
		sampleID = sample.ID

		permit, err := tx.CreatePermit(domain.Permit{
			PermitNumber:      "PER-1",
			Authority:         "Agency",
			Status:            domain.PermitStatusActive,
			ValidFrom:         now.Add(-time.Hour),
			ValidUntil:        now.Add(time.Hour),
			AllowedActivities: []string{"collect"},
			FacilityIDs:       []string{facility.ID},
			ProtocolIDs:       []string{protocol.ID},
			Notes:             strPtr("initial issuance"),
		})
		if err != nil {
			return err
		}
		permitID = permit.ID

		expiry := now.Add(24 * time.Hour)
		supply, err := tx.CreateSupplyItem(domain.SupplyItem{
			SKU:            "SKU-1",
			Name:           "Diet Blocks",
			Description:    strPtr("nutrient feed"),
			QuantityOnHand: 100,
			Unit:           "grams",
			LotNumber:      strPtr("LOT-1"),
			ExpiresAt:      &expiry,
			FacilityIDs:    []string{facility.ID},
			ProjectIDs:     []string{project.ID},
			ReorderLevel:   25,
			Attributes:     map[string]any{"supplier": "Acme"},
		})
		if err != nil {
			return err
		}
		supplyID = supply.ID

		return nil
	}); err != nil {
		t.Fatalf("outer tx: %v", err)
	}
	reloaded, err := NewStore(path, domain.NewRulesEngine())
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(reloaded.ListOrganisms()) != 1 {
		t.Fatalf("expected organisms after reload")
	}
	if len(reloaded.ListFacilities()) != 1 || len(reloaded.ListHousingUnits()) != 1 || len(reloaded.ListCohorts()) != 1 {
		t.Fatalf("expected facility, housing, cohort counts after reload")
	}
	if len(reloaded.ListTreatments()) != 1 || len(reloaded.ListObservations()) != 1 || len(reloaded.ListSamples()) != 1 {
		t.Fatalf("expected treatment, observation, sample counts after reload")
	}
	if len(reloaded.ListPermits()) != 1 || len(reloaded.ListSupplyItems()) != 1 {
		t.Fatalf("expected permit and supply counts after reload")
	}
	if cohorts := reloaded.ListCohorts(); len(cohorts) != 1 || cohorts[0].ID != cohortID {
		t.Fatalf("expected cohort persisted")
	}
	if procs := reloaded.ListProcedures(); len(procs) != 1 || procs[0].ID != procedureID {
		t.Fatalf("expected procedure persisted")
	}
	if protocols := reloaded.ListProtocols(); len(protocols) != 1 || protocols[0].ID != protocolID {
		t.Fatalf("expected protocol persisted")
	}
	if supplies := reloaded.ListSupplyItems(); len(supplies) != 1 || supplies[0].ID != supplyID {
		t.Fatalf("expected supply persisted")
	}
	if got, ok := reloaded.GetFacility(facilityID); !ok || got.AccessPolicy != "badge" {
		t.Fatalf("expected facility persisted")
	}
	if got, ok := reloaded.GetHousingUnit(housingID); !ok || got.FacilityID != facilityID {
		t.Fatalf("expected housing persisted")
	}
	if got, ok := reloaded.GetPermit(permitID); !ok || got.PermitNumber != "PER-1" {
		t.Fatalf("expected permit persisted")
	}
	if got, ok := reloaded.GetOrganism(organismID); !ok || got.Attributes["tag"] != "alpha" {
		t.Fatalf("expected organism attributes persisted")
	}
	if got := reloaded.ListProjects(); len(got) != 1 || got[0].ID != projectID {
		t.Fatalf("expected project persisted")
	}
	if err := reloaded.View(ctx, func(view domain.TransactionView) error {
		if sample, ok := view.FindSample(sampleID); !ok || sample.Status != domain.SampleStatusStored {
			return fmt.Errorf("expected sample persisted via view")
		}
		if treatment, ok := view.FindTreatment(treatmentID); !ok || treatment.DosagePlan != "10mg/kg" {
			return fmt.Errorf("expected treatment persisted via view")
		}
		if observation, ok := view.FindObservation(observationID); !ok || observation.Observer != "tech" {
			return fmt.Errorf("expected observation persisted via view")
		}
		return nil
	}); err != nil {
		t.Fatalf("view validation after reload: %v", err)
	}
	if reloaded.Path() != path {
		t.Fatalf("expected path match")
	}
	if reloaded.DB() == nil {
		t.Fatalf("expected db handle")
	}
}

func TestSQLiteStore_CorruptBucketHandling(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.db")
	store, err := NewStore(path, domain.NewRulesEngine())
	if err != nil {
		t.Skipf("sqlite unavailable: %v", err)
	}
	// insert corrupt row directly
	if _, err := store.DB().Exec(`INSERT INTO state(bucket, payload) VALUES('organisms', 'not-json')`); err != nil {
		t.Fatalf("insert corrupt: %v", err)
	}
	if _, err := NewStore(path, domain.NewRulesEngine()); err == nil {
		t.Fatalf("expected load error for corrupt json")
	}
}

func TestMemStore_RuleBlockingCoverage(t *testing.T) {
	store := newMemStore(domain.NewRulesEngine())
	store.RulesEngine().Register(blockingRule{})
	if _, err := store.RunInTransaction(context.Background(), func(tx domain.Transaction) error { _, e := tx.CreateOrganism(domain.Organism{Name: "Block"}); return e }); err == nil {
		t.Fatalf("expected blocking violation")
	}
}

// NowFunc already exercised indirectly in other tests via transactions; explicit test removed to satisfy lint (unused param warning).

// Covers transaction.Snapshot, transaction.FindHousingUnit/FindProtocol and
// transactionView.ListHousingUnits/ListProtocols/FindHousingUnit which were previously 0%.
func TestMemStore_TransactionViewFinds(t *testing.T) {
	store := newMemStore(nil)
	var housing domain.HousingUnit
	var protocol domain.Protocol
	runTx(t, store, func(tx domain.Transaction) error {
		f, err := tx.CreateFacility(domain.Facility{Name: "F"})
		if err != nil {
			return err
		}
		h, err := tx.CreateHousingUnit(domain.HousingUnit{Name: "T-H", Capacity: 1, Environment: "e", FacilityID: f.ID})
		if err != nil {
			return err
		}
		housing = h
		p, err := tx.CreateProtocol(domain.Protocol{Code: "TP", Title: "T", MaxSubjects: 1})
		if err != nil {
			return err
		}
		protocol = p
		// Direct transaction find methods
		if _, ok := tx.FindHousingUnit(housing.ID); !ok {
			t.Fatalf("tx.FindHousingUnit failed")
		}
		if _, ok := tx.FindProtocol(protocol.ID); !ok {
			t.Fatalf("tx.FindProtocol failed")
		}
		// Snapshot view usage
		v := tx.Snapshot()
		if len(v.ListHousingUnits()) != 1 {
			t.Fatalf("expected 1 housing in snapshot view")
		}
		if _, ok := v.FindHousingUnit(housing.ID); !ok {
			t.Fatalf("view.FindHousingUnit failed")
		}
		if len(v.ListProtocols()) != 1 {
			t.Fatalf("expected 1 protocol in snapshot view")
		}
		// Negative lookups for coverage of false branches
		if _, ok := tx.FindHousingUnit("missing-h"); ok {
			t.Fatalf("expected missing housing unit")
		}
		if _, ok := tx.FindProtocol("missing-p"); ok {
			t.Fatalf("expected missing protocol")
		}
		if _, ok := v.FindHousingUnit("missing-h"); ok {
			t.Fatalf("expected missing housing in view")
		}
		if _, ok := v.FindOrganism("missing-o"); ok {
			t.Fatalf("expected missing organism in view")
		}
		return nil
	})
}

// Covers not found branches for updates/deletes across entities and duplicate creates.
func TestMemStore_ErrorBranchesAdditional(t *testing.T) {
	store := newMemStore(nil)
	ctx := context.Background()
	// Seed minimal entities for duplicates
	var prot domain.Protocol
	var proj domain.Project
	runTx(t, store, func(tx domain.Transaction) error {
		p, err := tx.CreateProtocol(domain.Protocol{Code: "DUP", Title: "Dup", MaxSubjects: 1})
		if err != nil {
			return err
		}
		prot = p
		pr, err := tx.CreateProject(domain.Project{Code: "DUPP", Title: "DupProj"})
		if err != nil {
			return err
		}
		proj = pr
		return nil
	})
	// Duplicate create errors
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error { _, e := tx.CreateProtocol(prot); return e }); err == nil {
		t.Fatalf("expected duplicate protocol error")
	}
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error { _, e := tx.CreateProject(proj); return e }); err == nil {
		t.Fatalf("expected duplicate project error")
	}
	// Not found updates
	notFoundUpdates := []struct {
		name string
		fn   func(domain.Transaction) error
	}{
		{"cohort", func(tx domain.Transaction) error {
			_, e := tx.UpdateCohort("missing", func(*domain.Cohort) error { return nil })
			return e
		}},
		{"breeding", func(tx domain.Transaction) error {
			_, e := tx.UpdateBreedingUnit("missing", func(*domain.BreedingUnit) error { return nil })
			return e
		}},
		{"procedure", func(tx domain.Transaction) error {
			_, e := tx.UpdateProcedure("missing", func(*domain.Procedure) error { return nil })
			return e
		}},
		{"protocol", func(tx domain.Transaction) error {
			_, e := tx.UpdateProtocol("missing", func(*domain.Protocol) error { return nil })
			return e
		}},
		{"project", func(tx domain.Transaction) error {
			_, e := tx.UpdateProject("missing", func(*domain.Project) error { return nil })
			return e
		}},
	}
	for _, tc := range notFoundUpdates {
		if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error { return tc.fn(tx) }); err == nil {
			t.Fatalf("expected not found update for %s", tc.name)
		}
	}
	// Not found deletes
	missingDeletes := []struct {
		name string
		fn   func(domain.Transaction) error
	}{
		{"cohort", func(tx domain.Transaction) error { return tx.DeleteCohort("missing") }},
		{"housing", func(tx domain.Transaction) error { return tx.DeleteHousingUnit("missing") }},
		{"breeding", func(tx domain.Transaction) error { return tx.DeleteBreedingUnit("missing") }},
		{"procedure", func(tx domain.Transaction) error { return tx.DeleteProcedure("missing") }},
		{"protocol", func(tx domain.Transaction) error { return tx.DeleteProtocol("missing") }},
		{"project", func(tx domain.Transaction) error { return tx.DeleteProject("missing") }},
	}
	for _, tc := range missingDeletes {
		if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error { return tc.fn(tx) }); err == nil {
			t.Fatalf("expected not found delete for %s", tc.name)
		}
	}

}

func TestMemStoreCreateHousingUnitRequiresFacility(t *testing.T) {
	store := newMemStore(nil)
	if _, err := store.RunInTransaction(context.Background(), func(tx domain.Transaction) error {
		if _, err := tx.CreateHousingUnit(domain.HousingUnit{Name: "NoFacility", FacilityID: "missing", Capacity: 1}); err == nil {
			t.Fatalf("expected error when facility is missing")
		}
		facility, err := tx.CreateFacility(domain.Facility{Name: "Vivarium"})
		if err != nil {
			return err
		}
		if _, err := tx.CreateHousingUnit(domain.HousingUnit{Name: "Valid", FacilityID: facility.ID, Capacity: 2}); err != nil {
			t.Fatalf("expected housing creation to succeed: %v", err)
		}
		return nil
	}); err != nil {
		t.Fatalf("transaction unexpected error: %v", err)
	}
}

func TestMemStoreDeleteFacilityEnforcesReferences(t *testing.T) {
	store := newMemStore(nil)
	ctx := context.Background()
	now := time.Now().UTC()
	var (
		facility domain.Facility
		housing  domain.HousingUnit
		project  domain.Project
		protocol domain.Protocol
		permit   domain.Permit
		supply   domain.SupplyItem
		sample   domain.Sample
		org      domain.Organism
	)

	runTx(t, store, func(tx domain.Transaction) error {
		var err error
		facility, err = tx.CreateFacility(domain.Facility{Name: "Constraints"})
		if err != nil {
			return err
		}
		project, err = tx.CreateProject(domain.Project{Code: "PRJ", Title: "Project", FacilityIDs: []string{facility.ID}})
		if err != nil {
			return err
		}
		protocol, err = tx.CreateProtocol(domain.Protocol{Code: "PROTO", Title: "Protocol", MaxSubjects: 5})
		if err != nil {
			return err
		}
		housing, err = tx.CreateHousingUnit(domain.HousingUnit{Name: "Tank", FacilityID: facility.ID, Capacity: 2})
		if err != nil {
			return err
		}
		org, err = tx.CreateOrganism(domain.Organism{Name: "Specimen"})
		if err != nil {
			return err
		}
		sample, err = tx.CreateSample(domain.Sample{
			Identifier:      "S",
			SourceType:      "blood",
			FacilityID:      facility.ID,
			OrganismID:      &org.ID,
			CollectedAt:     now,
			Status:          domain.SampleStatusStored,
			StorageLocation: "room",
		})
		if err != nil {
			return err
		}
		permit, err = tx.CreatePermit(domain.Permit{
			PermitNumber:      "PERM",
			Authority:         "Gov",
			ValidFrom:         now,
			ValidUntil:        now.Add(time.Hour),
			AllowedActivities: []string{"collect"},
			FacilityIDs:       []string{facility.ID},
			ProtocolIDs:       []string{protocol.ID},
		})
		if err != nil {
			return err
		}
		supply, err = tx.CreateSupplyItem(domain.SupplyItem{
			SKU:            "SKU",
			Name:           "Gloves",
			QuantityOnHand: 5,
			Unit:           "box",
			FacilityIDs:    []string{facility.ID},
		})
		return err
	})

	expectDeleteError := func(substr string) {
		t.Helper()
		if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
			return tx.DeleteFacility(facility.ID)
		}); err == nil || !strings.Contains(err.Error(), substr) {
			t.Fatalf("expected delete error containing %q, got %v", substr, err)
		}
	}

	expectDeleteError("housing unit")

	runTx(t, store, func(tx domain.Transaction) error {
		return tx.DeleteHousingUnit(housing.ID)
	})

	expectDeleteError("sample")

	runTx(t, store, func(tx domain.Transaction) error {
		return tx.DeleteSample(sample.ID)
	})

	expectDeleteError("project")

	runTx(t, store, func(tx domain.Transaction) error {
		return tx.DeleteProject(project.ID)
	})

	expectDeleteError("permit")

	runTx(t, store, func(tx domain.Transaction) error {
		return tx.DeletePermit(permit.ID)
	})

	expectDeleteError("supply item")

	runTx(t, store, func(tx domain.Transaction) error {
		return tx.DeleteSupplyItem(supply.ID)
	})

	runTx(t, store, func(tx domain.Transaction) error {
		return tx.DeleteFacility(facility.ID)
	})
}

func TestMemStoreRelationshipValidations(t *testing.T) {
	store := newMemStore(nil)
	ctx := context.Background()
	now := time.Now().UTC()

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		if _, err := tx.CreateTreatment(domain.Treatment{Name: "NoProcedure"}); err == nil {
			t.Fatalf("expected treatment creation without procedure to fail")
		}

		facility, err := tx.CreateFacility(domain.Facility{Name: "Facility"})
		if err != nil {
			return err
		}
		protocol, err := tx.CreateProtocol(domain.Protocol{Code: "PROT", Title: "Protocol", MaxSubjects: 5})
		if err != nil {
			return err
		}

		if _, err := tx.CreateTreatment(domain.Treatment{Name: "MissingProcedureRef", ProcedureID: "missing"}); err == nil {
			t.Fatalf("expected treatment missing procedure to fail")
		}

		procedure, err := tx.CreateProcedure(domain.Procedure{Name: "Proc", Status: domain.ProcedureStatusScheduled, ScheduledAt: now, ProtocolID: protocol.ID})
		if err != nil {
			return err
		}

		if _, err := tx.CreateTreatment(domain.Treatment{Name: "MissingOrganism", ProcedureID: procedure.ID, OrganismIDs: []string{"missing"}}); err == nil {
			t.Fatalf("expected missing organism validation to fail")
		}

		organism, err := tx.CreateOrganism(domain.Organism{Name: "Org"})
		if err != nil {
			return err
		}

		if _, err := tx.CreateTreatment(domain.Treatment{Name: "ValidTreatment", ProcedureID: procedure.ID, OrganismIDs: []string{organism.ID}}); err != nil {
			t.Fatalf("expected treatment creation to succeed: %v", err)
		}

		if _, err := tx.CreateObservation(domain.Observation{Observer: "Tech", RecordedAt: now}); err == nil {
			t.Fatalf("expected observation without context to fail")
		}
		procID := procedure.ID
		if _, err := tx.CreateObservation(domain.Observation{ProcedureID: &procID, Observer: "Tech", RecordedAt: now}); err != nil {
			t.Fatalf("expected observation creation to succeed: %v", err)
		}

		if _, err := tx.CreateSample(domain.Sample{Identifier: "S0", SourceType: "blood", CollectedAt: now, Status: domain.SampleStatusStored, StorageLocation: "room"}); err == nil {
			t.Fatalf("expected sample without facility to fail")
		}
		if _, err := tx.CreateSample(domain.Sample{Identifier: "S1", SourceType: "blood", FacilityID: facility.ID, CollectedAt: now, Status: domain.SampleStatusStored, StorageLocation: "room"}); err == nil {
			t.Fatalf("expected sample without organism or cohort to fail")
		}
		if _, err := tx.CreateSample(domain.Sample{Identifier: "S2", SourceType: "blood", FacilityID: facility.ID, OrganismID: &organism.ID, CollectedAt: now, Status: domain.SampleStatusStored, StorageLocation: "room"}); err != nil {
			t.Fatalf("expected sample creation to succeed: %v", err)
		}

		if _, err := tx.CreatePermit(domain.Permit{PermitNumber: "PERM-FAIL", FacilityIDs: []string{"missing"}, ProtocolIDs: []string{"prot"}}); err == nil {
			t.Fatalf("expected permit with missing facility to fail")
		}
		if _, err := tx.CreatePermit(domain.Permit{
			PermitNumber: "PERM-FAIL2",
			FacilityIDs:  []string{facility.ID},
			ProtocolIDs:  []string{"missing"},
			ValidFrom:    now,
			ValidUntil:   now.Add(time.Hour),
		}); err == nil {
			t.Fatalf("expected permit with missing protocol to fail")
		}
		if _, err := tx.CreatePermit(domain.Permit{
			PermitNumber:      "PERM-OK",
			Authority:         "Gov",
			ValidFrom:         now,
			ValidUntil:        now.Add(time.Hour),
			AllowedActivities: []string{"collect"},
			FacilityIDs:       []string{facility.ID},
			ProtocolIDs:       []string{protocol.ID},
		}); err != nil {
			t.Fatalf("expected permit creation to succeed: %v", err)
		}

		if _, err := tx.CreateSupplyItem(domain.SupplyItem{SKU: "SKU-FAIL", Name: "Supply", FacilityIDs: []string{"missing"}}); err == nil {
			t.Fatalf("expected supply creation with missing facility to fail")
		}
		if _, err := tx.CreateSupplyItem(domain.SupplyItem{
			SKU:         "SKU-FAIL2",
			Name:        "Supply",
			FacilityIDs: []string{facility.ID},
			ProjectIDs:  []string{"missing"},
		}); err == nil {
			t.Fatalf("expected supply creation with missing project to fail")
		}

		project, err := tx.CreateProject(domain.Project{Code: "PRJ-REL", Title: "Project"})
		if err != nil {
			return err
		}
		if _, err := tx.CreateSupplyItem(domain.SupplyItem{
			SKU:            "SKU-OK",
			Name:           "Supply",
			QuantityOnHand: 10,
			Unit:           "box",
			FacilityIDs:    []string{facility.ID},
			ProjectIDs:     []string{project.ID},
		}); err != nil {
			t.Fatalf("expected supply creation to succeed: %v", err)
		}
		return nil
	}); err != nil {
		t.Fatalf("transaction unexpected error: %v", err)
	}
}

func TestSQLiteMigrateSnapshotCleansDataVariants(t *testing.T) {
	const facilityID = "fac-clean"
	now := time.Now().UTC()
	snapshot := Snapshot{
		Organisms: map[string]domain.Organism{
			"org-keep": {Base: domain.Base{ID: "org-keep"}, Name: "Org", Species: "Spec"},
		},
		Cohorts: map[string]domain.Cohort{
			"cohort-keep": {Base: domain.Base{ID: "cohort-keep"}, Name: "Cohort"},
		},
		Facilities: map[string]domain.Facility{
			facilityID: {Base: domain.Base{ID: facilityID}},
		},
		Housing: map[string]domain.HousingUnit{
			"housing-valid":  {Base: domain.Base{ID: "housing-valid"}, Name: "HV", FacilityID: facilityID, Capacity: 0},
			"housing-remove": {Base: domain.Base{ID: "housing-remove"}, Name: "HR", FacilityID: "missing", Capacity: 2},
		},
		Procedures: map[string]domain.Procedure{
			"proc-keep": {Base: domain.Base{ID: "proc-keep"}, Name: "Proc", Status: domain.ProcedureStatusScheduled, ScheduledAt: now, ProtocolID: "prot-keep"},
		},
		Treatments: map[string]domain.Treatment{
			"treatment-valid":  {Base: domain.Base{ID: "treatment-valid"}, Name: "Treat", ProcedureID: "proc-keep", OrganismIDs: []string{"org-keep", "org-keep", "missing"}, CohortIDs: []string{"cohort-keep", "missing"}},
			"treatment-remove": {Base: domain.Base{ID: "treatment-remove"}, Name: "TreatBad", ProcedureID: "missing"},
		},
		Observations: map[string]domain.Observation{
			"observation-valid": {Base: domain.Base{ID: "observation-valid"}, ProcedureID: ptr("proc-keep"), Observer: "Tech", RecordedAt: now},
			"observation-drop":  {Base: domain.Base{ID: "observation-drop"}, ProcedureID: ptr("missing"), Observer: "Tech", RecordedAt: now},
		},
		Samples: map[string]domain.Sample{
			"sample-valid":            {Base: domain.Base{ID: "sample-valid"}, Identifier: "S", SourceType: "blood", FacilityID: facilityID, OrganismID: ptr("org-keep"), CollectedAt: now, Status: domain.SampleStatusStored, StorageLocation: "room"},
			"sample-drop":             {Base: domain.Base{ID: "sample-drop"}, Identifier: "S2", SourceType: "blood", FacilityID: facilityID, OrganismID: ptr("missing"), CollectedAt: now, Status: domain.SampleStatusStored, StorageLocation: "room"},
			"sample-missing-facility": {Base: domain.Base{ID: "sample-missing-facility"}, Identifier: "S3", SourceType: "blood", FacilityID: "missing", CollectedAt: now, Status: domain.SampleStatusStored, StorageLocation: "room"},
		},
		Protocols: map[string]domain.Protocol{
			"prot-keep": {Base: domain.Base{ID: "prot-keep"}, Code: "PR", Title: "Protocol", MaxSubjects: 5, Status: "active"},
		},
		Permits: map[string]domain.Permit{
			"permit-valid": {Base: domain.Base{ID: "permit-valid"}, PermitNumber: "P", Authority: "Gov", ValidFrom: now, ValidUntil: now.Add(time.Hour), FacilityIDs: []string{facilityID, facilityID, "missing"}, ProtocolIDs: []string{"prot-keep", "missing"}},
		},
		Projects: map[string]domain.Project{
			"project-valid": {Base: domain.Base{ID: "project-valid"}, Code: "PRJ", Title: "Project", FacilityIDs: []string{facilityID, facilityID, "missing"}},
		},
		Supplies: map[string]domain.SupplyItem{
			"supply-valid": {Base: domain.Base{ID: "supply-valid"}, SKU: "SKU", Name: "Supply", FacilityIDs: []string{facilityID, facilityID, "missing"}, ProjectIDs: []string{"project-valid", "missing"}},
		},
	}

	migrated := migrateSnapshot(snapshot)

	if len(migrated.Housing) != 1 {
		t.Fatalf("expected one housing unit to remain, got %+v", migrated.Housing)
	}
	if got := migrated.Housing["housing-valid"].Capacity; got != 1 {
		t.Fatalf("expected capacity to default to 1, got %d", got)
	}

	if len(migrated.Treatments) != 1 || len(migrated.Treatments["treatment-valid"].OrganismIDs) != 1 || migrated.Treatments["treatment-valid"].OrganismIDs[0] != "org-keep" {
		t.Fatalf("unexpected treatments after migration: %+v", migrated.Treatments)
	}
	if len(migrated.Treatments["treatment-valid"].CohortIDs) != 1 || migrated.Treatments["treatment-valid"].CohortIDs[0] != "cohort-keep" {
		t.Fatalf("expected cohort IDs to be deduped")
	}

	if len(migrated.Observations) != 1 {
		t.Fatalf("expected single observation, got %+v", migrated.Observations)
	}
	if migrated.Observations["observation-valid"].Data == nil {
		t.Fatalf("expected observation data map to be initialised")
	}

	if len(migrated.Samples) != 1 {
		t.Fatalf("expected single valid sample, got %+v", migrated.Samples)
	}
	if migrated.Samples["sample-valid"].Attributes == nil {
		t.Fatalf("expected sample attributes map to be initialised")
	}

	if len(migrated.Permits) != 1 || len(migrated.Permits["permit-valid"].FacilityIDs) != 1 || migrated.Permits["permit-valid"].FacilityIDs[0] != facilityID {
		t.Fatalf("expected permit facility IDs filtered, got %+v", migrated.Permits["permit-valid"].FacilityIDs)
	}
	if len(migrated.Permits["permit-valid"].ProtocolIDs) != 1 || migrated.Permits["permit-valid"].ProtocolIDs[0] != "prot-keep" {
		t.Fatalf("expected permit protocol IDs filtered")
	}

	if len(migrated.Projects["project-valid"].FacilityIDs) != 1 || migrated.Projects["project-valid"].FacilityIDs[0] != facilityID {
		t.Fatalf("expected project facility IDs filtered")
	}

	if migrated.Supplies["supply-valid"].Attributes == nil {
		t.Fatalf("expected supply attributes map initialised")
	}
	if len(migrated.Supplies["supply-valid"].FacilityIDs) != 1 || migrated.Supplies["supply-valid"].FacilityIDs[0] != facilityID {
		t.Fatalf("expected supply facility IDs filtered")
	}
	if len(migrated.Supplies["supply-valid"].ProjectIDs) != 1 || migrated.Supplies["supply-valid"].ProjectIDs[0] != "project-valid" {
		t.Fatalf("expected supply project IDs filtered")
	}

	facility := migrated.Facilities[facilityID]
	if facility.EnvironmentBaselines == nil {
		t.Fatalf("expected facility baselines map initialised")
	}
	if len(facility.HousingUnitIDs) != 1 || facility.HousingUnitIDs[0] != "housing-valid" {
		t.Fatalf("expected facility housing IDs populated, got %+v", facility.HousingUnitIDs)
	}
	if len(facility.ProjectIDs) != 1 || facility.ProjectIDs[0] != "project-valid" {
		t.Fatalf("expected facility project IDs populated, got %+v", facility.ProjectIDs)
	}
}

func TestMemStoreUpdateSupplyItemDedupe(t *testing.T) {
	store := newMemStore(nil)
	var (
		facility domain.Facility
		project  domain.Project
		supply   domain.SupplyItem
	)

	runTx(t, store, func(tx domain.Transaction) error {
		var err error
		facility, err = tx.CreateFacility(domain.Facility{Name: "Supply Facility"})
		if err != nil {
			return err
		}
		project, err = tx.CreateProject(domain.Project{Code: "SUP", Title: "Supply Project"})
		if err != nil {
			return err
		}
		supply, err = tx.CreateSupplyItem(domain.SupplyItem{
			SKU:            "SKU-DEDUP",
			Name:           "Gloves",
			QuantityOnHand: 5,
			Unit:           "box",
			FacilityIDs:    []string{facility.ID},
			ProjectIDs:     []string{project.ID},
		})
		return err
	})

	runTx(t, store, func(tx domain.Transaction) error {
		_, err := tx.UpdateSupplyItem(supply.ID, func(s *domain.SupplyItem) error {
			s.FacilityIDs = []string{facility.ID, facility.ID}
			s.ProjectIDs = []string{project.ID, project.ID}
			return nil
		})
		if err != nil {
			return err
		}
		if _, err := tx.UpdateSupplyItem(supply.ID, func(s *domain.SupplyItem) error {
			s.FacilityIDs = []string{"missing"}
			return nil
		}); err == nil {
			t.Fatalf("expected update error for missing facility reference")
		}
		return nil
	})

	items := store.ListSupplyItems()
	if len(items) != 1 {
		t.Fatalf("expected single supply item, got %d", len(items))
	}
	if len(items[0].FacilityIDs) != 1 || items[0].FacilityIDs[0] != facility.ID {
		t.Fatalf("expected deduped facility IDs, got %+v", items[0].FacilityIDs)
	}
	if len(items[0].ProjectIDs) != 1 || items[0].ProjectIDs[0] != project.ID {
		t.Fatalf("expected deduped project IDs, got %+v", items[0].ProjectIDs)
	}
}

func TestMemStoreUpdatePermitDedupe(t *testing.T) {
	store := newMemStore(nil)
	var (
		facility domain.Facility
		protocol domain.Protocol
		permit   domain.Permit
	)

	runTx(t, store, func(tx domain.Transaction) error {
		var err error
		facility, err = tx.CreateFacility(domain.Facility{Name: "Permit Facility"})
		if err != nil {
			return err
		}
		protocol, err = tx.CreateProtocol(domain.Protocol{Code: "PERM", Title: "Permit Proto", MaxSubjects: 2})
		if err != nil {
			return err
		}
		permit, err = tx.CreatePermit(domain.Permit{
			PermitNumber:      "PERM-DEDUP",
			Authority:         "Gov",
			ValidFrom:         time.Now().UTC(),
			ValidUntil:        time.Now().UTC().Add(time.Hour),
			AllowedActivities: []string{"collect"},
			FacilityIDs:       []string{facility.ID},
			ProtocolIDs:       []string{protocol.ID},
		})
		return err
	})

	runTx(t, store, func(tx domain.Transaction) error {
		_, err := tx.UpdatePermit(permit.ID, func(p *domain.Permit) error {
			p.FacilityIDs = []string{facility.ID, facility.ID}
			p.ProtocolIDs = []string{protocol.ID, protocol.ID}
			return nil
		})
		if err != nil {
			return err
		}
		if _, err := tx.UpdatePermit(permit.ID, func(p *domain.Permit) error {
			p.FacilityIDs = []string{"missing"}
			return nil
		}); err == nil {
			t.Fatalf("expected update error for missing facility reference")
		}
		return nil
	})

	permits := store.ListPermits()
	if len(permits) != 1 {
		t.Fatalf("expected single permit, got %d", len(permits))
	}
	if len(permits[0].FacilityIDs) != 1 || permits[0].FacilityIDs[0] != facility.ID {
		t.Fatalf("expected deduped facility IDs, got %+v", permits[0].FacilityIDs)
	}
	if len(permits[0].ProtocolIDs) != 1 || permits[0].ProtocolIDs[0] != protocol.ID {
		t.Fatalf("expected deduped protocol IDs, got %+v", permits[0].ProtocolIDs)
	}
}
