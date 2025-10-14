package sqlite

import (
	"colonycore/pkg/domain"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

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
		pr, _ := tx.CreateProcedure(domain.Procedure{Name: "Proc", Status: "scheduled", ProtocolID: protocol.ID, OrganismIDs: []string{o1.ID}, ScheduledAt: time.Now().UTC()})
		procedure = pr
		t, _ := tx.CreateTreatment(domain.Treatment{Name: "Dose", ProcedureID: procedure.ID, OrganismIDs: []string{o1.ID}, CohortIDs: []string{cohort.ID}, DosagePlan: "10mg"})
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
			Status:          "stored",
			StorageLocation: "freezer",
			AssayType:       "PCR",
			ChainOfCustody:  custody,
			Attributes:      map[string]any{"volume_ml": 1.0},
		})
		sample = sa
		per, _ := tx.CreatePermit(domain.Permit{
			PermitNumber:      "PER-1",
			Authority:         "Agency",
			ValidFrom:         now.Add(-time.Hour),
			ValidUntil:        now.Add(time.Hour),
			AllowedActivities: []string{"collect"},
			FacilityIDs:       []string{facility.ID},
			ProtocolIDs:       []string{protocol.ID},
			Notes:             "issue",
		})
		permit = per
		expiry := time.Now().Add(48 * time.Hour)
		s, _ := tx.CreateSupplyItem(domain.SupplyItem{
			SKU:            "SKU1",
			Name:           "Feed",
			Description:    "daily feed",
			QuantityOnHand: 50,
			Unit:           "grams",
			LotNumber:      "LOT-1",
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
		_, _ = tx.UpdateProtocol(protocol.ID, func(p *domain.Protocol) error { p.Description = "desc"; return nil })
		_, _ = tx.UpdateProcedure(procedure.ID, func(p *domain.Procedure) error { p.Status = "complete"; return nil })
		_, _ = tx.UpdateProject(project.ID, func(p *domain.Project) error { p.Description = "d"; return nil })
		_, _ = tx.UpdateFacility(facility.ID, func(f *domain.Facility) error {
			f.AccessPolicy = "training"
			f.EnvironmentBaselines["humidity"] = "55%"
			return nil
		})
		_, _ = tx.UpdateTreatment(treatment.ID, func(t *domain.Treatment) error {
			t.AdministrationLog = append(t.AdministrationLog, "follow-up")
			return nil
		})
		_, _ = tx.UpdateObservation(observation.ID, func(o *domain.Observation) error { o.Notes = "checked"; o.Data["score"] = 6; return nil })
		_, _ = tx.UpdateSample(sample.ID, func(s *domain.Sample) error { s.Status = "consumed"; return nil })
		_, _ = tx.UpdatePermit(permit.ID, func(p *domain.Permit) error { p.Notes = "updated"; return nil })
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
		_ = tx.DeleteProcedure(procedure.ID)
		_ = tx.DeleteProtocol(protocol.ID)
		_ = tx.DeleteBreedingUnit(breeding.ID)
		_ = tx.DeleteTreatment(treatment.ID)
		_ = tx.DeleteObservation(observation.ID)
		_ = tx.DeleteSample(sample.ID)
		_ = tx.DeletePermit(permit.ID)
		_ = tx.DeleteSupplyItem(supply.ID)
		_ = tx.DeleteHousingUnit(housing.ID)
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
			Status:      "scheduled",
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
			Status:          "stored",
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
			ValidFrom:         now.Add(-time.Hour),
			ValidUntil:        now.Add(time.Hour),
			AllowedActivities: []string{"collect"},
			FacilityIDs:       []string{facility.ID},
			ProtocolIDs:       []string{protocol.ID},
			Notes:             "initial issuance",
		})
		if err != nil {
			return err
		}
		permitID = permit.ID

		expiry := now.Add(24 * time.Hour)
		supply, err := tx.CreateSupplyItem(domain.SupplyItem{
			SKU:            "SKU-1",
			Name:           "Diet Blocks",
			Description:    "nutrient feed",
			QuantityOnHand: 100,
			Unit:           "grams",
			LotNumber:      "LOT-1",
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
		if sample, ok := view.FindSample(sampleID); !ok || sample.Status != "stored" {
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
		h, err := tx.CreateHousingUnit(domain.HousingUnit{Name: "T-H", Capacity: 1, Environment: "e", FacilityID: "F"})
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
