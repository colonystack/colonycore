package memory_test

import (
	"colonycore/internal/infra/persistence/memory"
	"colonycore/pkg/domain"
	"context"
	"fmt"
	"testing"
	"time"
)

type memoryIDs struct {
	projectID     string
	protocolID    string
	facilityID    string
	housingID     string
	cohortID      string
	breedingID    string
	procedureID   string
	treatmentID   string
	observationID string
	sampleID      string
	permitID      string
	supplyItemID  string
	organismAID   string
	organismBID   string
}

const permitNumberFixture = "PER-1"

func TestMemoryStoreCRUDAndQueries(t *testing.T) {
	store := memory.NewStore(nil)

	ids := seedMemoryStore(t, store)
	verifyMemoryStorePostCreate(t, store, ids)
	exerciseMemoryUpdates(t, store, ids)
	exerciseMemoryDeletes(t, store, ids)
	verifyMemoryStorePostDelete(t, store)
}

func seedMemoryStore(t *testing.T, store *memory.Store) memoryIDs {
	t.Helper()
	ctx := context.Background()

	var ids memoryIDs
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		if _, err := tx.CreateHousingUnit(domain.HousingUnit{Name: "Invalid", FacilityID: "Lab", Capacity: 0}); err == nil {
			return fmt.Errorf("expected capacity validation error")
		}

		projectVal, err := tx.CreateProject(domain.Project{Code: "PRJ-1", Title: "domain.Project"})
		project := must(t, projectVal, err)
		ids.projectID = project.ID

		facilityVal, err := tx.CreateFacility(domain.Facility{
			Name:                 "Vivarium",
			Zone:                 "Zone-A",
			AccessPolicy:         "badge-required",
			EnvironmentBaselines: map[string]any{"temperature": "22C"},
			ProjectIDs:           []string{ids.projectID},
		})
		facility := must(t, facilityVal, err)
		ids.facilityID = facility.ID

		foundFacility, ok := tx.FindFacility(ids.facilityID)
		requireFound(t, foundFacility, ok, "expected to find facility")
		if foundFacility.ID != ids.facilityID {
			t.Fatalf("unexpected facility returned from lookup")
		}
		_, ok = tx.FindFacility("missing-facility")
		requireMissing(t, ok, "unexpected facility lookup success")

		protocolVal, err := tx.CreateProtocol(domain.Protocol{Code: "PROT-1", Title: "domain.Protocol", MaxSubjects: 5})
		protocol := must(t, protocolVal, err)
		ids.protocolID = protocol.ID
		foundProtocol, ok := tx.FindProtocol(ids.protocolID)
		requireFound(t, foundProtocol, ok, "expected to find protocol")
		if foundProtocol.Code != "PROT-1" {
			t.Fatalf("unexpected protocol returned from lookup")
		}
		_, ok = tx.FindProtocol("missing-protocol")
		requireMissing(t, ok, "unexpected protocol lookup success")

		housingVal, err := tx.CreateHousingUnit(domain.HousingUnit{Name: "Tank", FacilityID: ids.facilityID, Capacity: 2, Environment: "arid"})
		housing := must(t, housingVal, err)
		ids.housingID = housing.ID
		_, err = tx.UpdateFacility(ids.facilityID, func(f *domain.Facility) error {
			f.HousingUnitIDs = append(f.HousingUnitIDs, ids.housingID)
			return nil
		})
		mustNoErr(t, err)
		_, ok = tx.FindTreatment("missing-treatment")
		requireMissing(t, ok, "unexpected treatment lookup success")

		projectPtr := ids.projectID
		housingPtr := ids.housingID
		protocolPtr := ids.protocolID

		cohortVal, err := tx.CreateCohort(domain.Cohort{
			Name:       "domain.Cohort",
			Purpose:    "Observation",
			ProjectID:  &projectPtr,
			HousingID:  &housingPtr,
			ProtocolID: &protocolPtr,
		})
		cohort := must(t, cohortVal, err)
		ids.cohortID = cohort.ID

		cohortPtr := ids.cohortID

		attrs := map[string]any{"skin_color_index": 5}
		organismAVal, err := tx.CreateOrganism(domain.Organism{
			Name:       "Alpha",
			Species:    "Test Frog",
			Stage:      domain.StageJuvenile,
			ProjectID:  &projectPtr,
			ProtocolID: &protocolPtr,
			CohortID:   &cohortPtr,
			HousingID:  &housingPtr,
			Attributes: attrs,
		})
		organismA := must(t, organismAVal, err)
		ids.organismAID = organismA.ID

		attrs["skin_color_index"] = 9

		organismBVal, err := tx.CreateOrganism(domain.Organism{
			Name:     "Beta",
			Species:  "Test Toad",
			Stage:    domain.StageAdult,
			CohortID: &cohortPtr,
		})
		organismB := must(t, organismBVal, err)
		ids.organismBID = organismB.ID

		if _, err := tx.CreateOrganism(domain.Organism{Base: domain.Base{ID: ids.organismAID}, Name: "Duplicate"}); err == nil {
			return fmt.Errorf("expected duplicate organism error")
		}

		breedingVal, err := tx.CreateBreedingUnit(domain.BreedingUnit{
			Name:       "Pair",
			Strategy:   "pair",
			HousingID:  &housingPtr,
			ProtocolID: &protocolPtr,
			FemaleIDs:  []string{ids.organismAID},
			MaleIDs:    []string{ids.organismBID},
		})
		breeding := must(t, breedingVal, err)
		ids.breedingID = breeding.ID

		procedureVal, err := tx.CreateProcedure(domain.Procedure{
			Name:        "Check",
			Status:      "scheduled",
			ScheduledAt: time.Now().Add(time.Minute),
			ProtocolID:  ids.protocolID,
			OrganismIDs: []string{ids.organismAID, ids.organismBID},
		})
		procedure := must(t, procedureVal, err)
		ids.procedureID = procedure.ID

		treatmentVal, err := tx.CreateTreatment(domain.Treatment{
			Name:              "Dose",
			ProcedureID:       ids.procedureID,
			OrganismIDs:       []string{ids.organismAID},
			CohortIDs:         []string{ids.cohortID},
			DosagePlan:        "10mg/kg",
			AdministrationLog: []string{"t0: administered"},
			AdverseEvents:     []string{},
		})
		treatment := must(t, treatmentVal, err)
		ids.treatmentID = treatment.ID

		foundTreatment, ok := tx.FindTreatment(ids.treatmentID)
		requireFound(t, foundTreatment, ok, "expected to find treatment")
		if foundTreatment.Name != "Dose" {
			t.Fatalf("unexpected treatment returned from lookup")
		}

		recorded := time.Now().UTC()
		observationVal, err := tx.CreateObservation(domain.Observation{
			ProcedureID: &ids.procedureID,
			OrganismID:  &ids.organismAID,
			RecordedAt:  recorded,
			Observer:    "tech",
			Data:        map[string]any{"score": 5},
			Notes:       "baseline",
		})
		observation := must(t, observationVal, err)
		ids.observationID = observation.ID

		foundObservation, ok := tx.FindObservation(ids.observationID)
		requireFound(t, foundObservation, ok, "expected to find observation")
		if foundObservation.Observer != "tech" {
			t.Fatalf("unexpected observation returned from lookup")
		}

		custody := []domain.SampleCustodyEvent{{Actor: "tech", Location: "bench", Timestamp: time.Now().UTC(), Notes: "collected"}}
		sampleVal, err := tx.CreateSample(domain.Sample{
			Identifier:      "S-1",
			SourceType:      "blood",
			OrganismID:      &ids.organismAID,
			FacilityID:      ids.facilityID,
			CollectedAt:     time.Now().UTC(),
			Status:          "stored",
			StorageLocation: "freezer-1",
			AssayType:       "PCR",
			ChainOfCustody:  custody,
			Attributes:      map[string]any{"volume_ml": 1.5},
		})
		sample := must(t, sampleVal, err)
		ids.sampleID = sample.ID

		foundSample, ok := tx.FindSample(ids.sampleID)
		requireFound(t, foundSample, ok, "expected to find sample")
		if foundSample.Identifier != "S-1" {
			t.Fatalf("unexpected sample returned from lookup")
		}
		_, ok = tx.FindSample("missing-sample")
		requireMissing(t, ok, "unexpected sample lookup success")

		permitVal, err := tx.CreatePermit(domain.Permit{
			PermitNumber:      permitNumberFixture,
			Authority:         "Agency",
			ValidFrom:         time.Now().Add(-time.Hour),
			ValidUntil:        time.Now().Add(24 * time.Hour),
			AllowedActivities: []string{"collect"},
			FacilityIDs:       []string{ids.facilityID},
			ProtocolIDs:       []string{ids.protocolID},
			Notes:             "initial issuance",
		})
		permit := must(t, permitVal, err)
		ids.permitID = permit.ID

		foundPermit, ok := tx.FindPermit(ids.permitID)
		requireFound(t, foundPermit, ok, "expected to find permit")
		if foundPermit.PermitNumber != permitNumberFixture {
			t.Fatalf("unexpected permit returned from lookup")
		}
		_, ok = tx.FindPermit("missing-permit")
		requireMissing(t, ok, "unexpected permit lookup success")

		expiry := time.Now().Add(48 * time.Hour)
		supplyVal, err := tx.CreateSupplyItem(domain.SupplyItem{
			SKU:            "SKU-1",
			Name:           "Diet Blocks",
			Description:    "nutrient feed",
			QuantityOnHand: 100,
			Unit:           "grams",
			LotNumber:      "LOT-44",
			ExpiresAt:      &expiry,
			FacilityIDs:    []string{ids.facilityID},
			ProjectIDs:     []string{ids.projectID},
			ReorderLevel:   20,
			Attributes:     map[string]any{"supplier": "Acme"},
		})
		supply := must(t, supplyVal, err)
		ids.supplyItemID = supply.ID

		foundSupply, ok := tx.FindSupplyItem(ids.supplyItemID)
		requireFound(t, foundSupply, ok, "expected to find supply item")
		if foundSupply.SKU != "SKU-1" {
			t.Fatalf("unexpected supply item returned from lookup")
		}
		_, ok = tx.FindSupplyItem("missing-supply")
		requireMissing(t, ok, "unexpected supply item lookup success")

		view := tx.Snapshot()
		requireLen(t, view.ListOrganisms(), 2, "view organisms count")
		_, ok = view.FindOrganism("missing")
		requireMissing(t, ok, "unexpected organism lookup success")
		_, ok = view.FindHousingUnit("missing")
		requireMissing(t, ok, "unexpected housing lookup success")

		requireLen(t, view.ListFacilities(), 1, "view facilities count")
		facilityView, ok := view.FindFacility(ids.facilityID)
		requireFound(t, facilityView, ok, "expected facility lookup success in view")
		if facilityView.ID != ids.facilityID {
			t.Fatalf("facility snapshot mismatch")
		}
		_, ok = view.FindFacility("missing")
		requireMissing(t, ok, "unexpected facility lookup success in view")

		requireLen(t, view.ListTreatments(), 1, "view treatments count")
		treatmentView, ok := view.FindTreatment(ids.treatmentID)
		requireFound(t, treatmentView, ok, "expected treatment lookup success in view")
		if treatmentView.Name != "Dose" {
			t.Fatalf("treatment snapshot mismatch")
		}

		requireLen(t, view.ListObservations(), 1, "view observations count")
		observationView, ok := view.FindObservation(ids.observationID)
		requireFound(t, observationView, ok, "expected observation lookup success in view")
		if observationView.Observer != "tech" {
			t.Fatalf("observation snapshot mismatch")
		}
		_, ok = view.FindObservation("missing")
		requireMissing(t, ok, "unexpected observation lookup success in view")

		requireLen(t, view.ListSamples(), 1, "view samples count")
		sampleView, ok := view.FindSample(ids.sampleID)
		requireFound(t, sampleView, ok, "expected sample lookup success in view")
		if sampleView.Identifier != "S-1" {
			t.Fatalf("sample snapshot mismatch")
		}
		_, ok = view.FindSample("missing")
		requireMissing(t, ok, "unexpected sample lookup success in view")

		requireLen(t, view.ListPermits(), 1, "view permits count")
		permitView, ok := view.FindPermit(ids.permitID)
		requireFound(t, permitView, ok, "expected permit lookup success in view")
		if permitView.PermitNumber != permitNumberFixture {
			t.Fatalf("permit snapshot mismatch")
		}

		requireLen(t, view.ListProtocols(), 1, "view protocols count")
		requireLen(t, view.ListProjects(), 1, "view projects count")
		requireLen(t, view.ListSupplyItems(), 1, "view supply items count")
		supplyView, ok := view.FindSupplyItem(ids.supplyItemID)
		requireFound(t, supplyView, ok, "expected supply item lookup success in view")
		if supplyView.SKU != "SKU-1" {
			t.Fatalf("supply item snapshot mismatch")
		}

		return nil
	}); err != nil {
		t.Fatalf("create transaction: %v", err)
	}

	return ids
}

func verifyMemoryStorePostCreate(t *testing.T, store *memory.Store, ids memoryIDs) {
	t.Helper()

	organisms := store.ListOrganisms()
	requireLen(t, organisms, 2, "organism list length")

	var copyCheckDone bool
	for _, organism := range organisms {
		if organism.ID != ids.organismAID {
			continue
		}
		if organism.Attributes["skin_color_index"].(int) != 5 {
			t.Fatalf("expected cloned attributes value 5, got %v", organism.Attributes["skin_color_index"])
		}
		organism.Attributes["skin_color_index"] = 1
		copyCheckDone = true
	}
	if !copyCheckDone {
		t.Fatalf("organism %s not found in list", ids.organismAID)
	}

	refreshedVal, ok := store.GetOrganism(ids.organismAID)
	refreshed := mustGet(t, refreshedVal, ok, "expected organism to exist")
	if refreshed.Attributes["skin_color_index"].(int) != 5 {
		t.Fatalf("expected store attributes to remain 5, got %v", refreshed.Attributes["skin_color_index"])
	}

	housingList := store.ListHousingUnits()
	requireLen(t, housingList, 1, "housing list length")
	housingList[0].Environment = "modified"

	storedHousingVal, ok := store.GetHousingUnit(ids.housingID)
	storedHousing := mustGet(t, storedHousingVal, ok, "expected housing unit to exist")
	if storedHousing.Environment != "arid" {
		t.Fatalf("expected environment to remain arid, got %s", storedHousing.Environment)
	}

	facilityVal, ok := store.GetFacility(ids.facilityID)
	facility := mustGet(t, facilityVal, ok, "expected facility to exist")
	if facility.AccessPolicy != "badge-required" {
		t.Fatalf("unexpected access policy %s", facility.AccessPolicy)
	}

	permitVal, ok := store.GetPermit(ids.permitID)
	permit := mustGet(t, permitVal, ok, "expected permit to exist")
	if permit.PermitNumber != permitNumberFixture {
		t.Fatalf("unexpected permit number %s", permit.PermitNumber)
	}

	projects := store.ListProjects()
	requireLen(t, projects, 1, "project list length")

	supplyItems := store.ListSupplyItems()
	requireLen(t, supplyItems, 1, "supply list length")
}

func exerciseMemoryUpdates(t *testing.T, store *memory.Store, ids memoryIDs) {
	t.Helper()
	ctx := context.Background()
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		if _, err := tx.UpdateOrganism("missing", func(*domain.Organism) error { return nil }); err == nil {
			return fmt.Errorf("expected update error for missing organism")
		}
		if _, err := tx.UpdateHousingUnit(ids.housingID, func(h *domain.HousingUnit) error {
			h.Capacity = 0
			return nil
		}); err == nil {
			return fmt.Errorf("expected housing capacity validation on update")
		}
		if _, err := tx.UpdateHousingUnit("missing", func(*domain.HousingUnit) error { return nil }); err == nil {
			return fmt.Errorf("expected missing housing update error")
		}

		const updatedDesc = "updated"
		_, err := tx.UpdateProject(ids.projectID, func(p *domain.Project) error {
			p.Description = updatedDesc
			return nil
		})
		mustNoErr(t, err)
		_, err = tx.UpdateProtocol(ids.protocolID, func(p *domain.Protocol) error {
			p.Description = updatedDesc
			return nil
		})
		mustNoErr(t, err)
		_, err = tx.UpdateCohort(ids.cohortID, func(c *domain.Cohort) error {
			c.Purpose = updatedDesc
			return nil
		})
		mustNoErr(t, err)
		_, err = tx.UpdateBreedingUnit(ids.breedingID, func(b *domain.BreedingUnit) error {
			b.Strategy = updatedDesc
			b.FemaleIDs = append(b.FemaleIDs, ids.organismBID)
			return nil
		})
		mustNoErr(t, err)
		_, err = tx.UpdateProcedure(ids.procedureID, func(p *domain.Procedure) error {
			p.Status = "completed"
			return nil
		})
		mustNoErr(t, err)
		_, err = tx.UpdateFacility(ids.facilityID, func(f *domain.Facility) error {
			f.AccessPolicy = "biosafety-training"
			if f.EnvironmentBaselines == nil {
				f.EnvironmentBaselines = map[string]any{}
			}
			f.EnvironmentBaselines["humidity"] = "55%"
			return nil
		})
		mustNoErr(t, err)
		_, err = tx.UpdateTreatment(ids.treatmentID, func(tr *domain.Treatment) error {
			tr.AdministrationLog = append(tr.AdministrationLog, "t2: follow-up")
			tr.AdverseEvents = append(tr.AdverseEvents, "minor redness")
			return nil
		})
		mustNoErr(t, err)
		_, err = tx.UpdateObservation(ids.observationID, func(o *domain.Observation) error {
			o.Notes = updatedDesc
			if o.Data == nil {
				o.Data = map[string]any{}
			}
			o.Data["score"] = 6
			return nil
		})
		mustNoErr(t, err)
		_, err = tx.UpdateSample(ids.sampleID, func(s *domain.Sample) error {
			s.Status = "consumed"
			s.ChainOfCustody = append(s.ChainOfCustody, domain.SampleCustodyEvent{Actor: "lab", Location: "analysis", Timestamp: time.Now().UTC()})
			if s.Attributes == nil {
				s.Attributes = map[string]any{}
			}
			s.Attributes["volume_ml"] = 1.0
			return nil
		})
		mustNoErr(t, err)
		_, err = tx.UpdatePermit(ids.permitID, func(p *domain.Permit) error {
			p.Notes = updatedDesc
			p.AllowedActivities = append(p.AllowedActivities, "dispose")
			return nil
		})
		mustNoErr(t, err)
		_, err = tx.UpdateSupplyItem(ids.supplyItemID, func(s *domain.SupplyItem) error {
			s.QuantityOnHand = 80
			return nil
		})
		mustNoErr(t, err)
		_, err = tx.UpdateOrganism(ids.organismBID, func(o *domain.Organism) error {
			o.Stage = domain.StageRetired
			return nil
		})
		mustNoErr(t, err)
		return nil
	}); err != nil {
		t.Fatalf("update transaction: %v", err)
	}

	updatedOrganismVal, ok := store.GetOrganism(ids.organismBID)
	updatedOrganism := mustGet(t, updatedOrganismVal, ok, "expected organism after update")
	if updatedOrganism.Stage != domain.StageRetired {
		t.Fatalf("expected stage to be retired, got %s", updatedOrganism.Stage)
	}

	facilityVal, ok := store.GetFacility(ids.facilityID)
	facility := mustGet(t, facilityVal, ok, "expected facility after update")
	if facility.AccessPolicy != "biosafety-training" {
		t.Fatalf("expected updated access policy, got %s", facility.AccessPolicy)
	}
	if facility.EnvironmentBaselines["humidity"] != "55%" {
		t.Fatalf("expected humidity baseline to be updated")
	}

	permitVal, ok := store.GetPermit(ids.permitID)
	permit := mustGet(t, permitVal, ok, "expected permit after update")
	if len(permit.AllowedActivities) != 2 {
		t.Fatalf("expected permit activities to be extended")
	}

	treatments := store.ListTreatments()
	requireLen(t, treatments, 1, "treatment list length after update")
	if len(treatments[0].AdministrationLog) != 2 {
		t.Fatalf("expected treatment administration log to grow")
	}

	samples := store.ListSamples()
	requireLen(t, samples, 1, "sample list length after update")
	if samples[0].Status != "consumed" {
		t.Fatalf("expected sample status to be consumed, got %s", samples[0].Status)
	}

	supplyItems := store.ListSupplyItems()
	requireLen(t, supplyItems, 1, "supply list length after update")
	if supplyItems[0].QuantityOnHand != 80 {
		t.Fatalf("expected supply quantity to change, got %d", supplyItems[0].QuantityOnHand)
	}
}

func exerciseMemoryDeletes(t *testing.T, store *memory.Store, ids memoryIDs) {
	t.Helper()
	ctx := context.Background()
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		mustNoErr(t, tx.DeleteProcedure(ids.procedureID))
		mustNoErr(t, tx.DeleteBreedingUnit(ids.breedingID))
		mustNoErr(t, tx.DeleteOrganism(ids.organismAID))
		mustNoErr(t, tx.DeleteOrganism(ids.organismBID))
		mustNoErr(t, tx.DeleteCohort(ids.cohortID))
		mustNoErr(t, tx.DeleteHousingUnit(ids.housingID))
		mustNoErr(t, tx.DeleteFacility(ids.facilityID))
		mustNoErr(t, tx.DeleteProtocol(ids.protocolID))
		mustNoErr(t, tx.DeleteTreatment(ids.treatmentID))
		mustNoErr(t, tx.DeleteObservation(ids.observationID))
		mustNoErr(t, tx.DeleteSample(ids.sampleID))
		mustNoErr(t, tx.DeletePermit(ids.permitID))
		mustNoErr(t, tx.DeleteProject(ids.projectID))
		mustNoErr(t, tx.DeleteSupplyItem(ids.supplyItemID))
		if err := tx.DeleteOrganism(ids.organismAID); err == nil {
			return fmt.Errorf("expected delete error for missing organism")
		}
		return nil
	}); err != nil {
		t.Fatalf("delete transaction: %v", err)
	}
}

func verifyMemoryStorePostDelete(t *testing.T, store *memory.Store) {
	t.Helper()

	requireLen(t, store.ListOrganisms(), 0, "organisms after deletion")
	requireLen(t, store.ListCohorts(), 0, "cohorts after deletion")
	requireLen(t, store.ListHousingUnits(), 0, "housing units after deletion")
	requireLen(t, store.ListFacilities(), 0, "facilities after deletion")
	requireLen(t, store.ListProtocols(), 0, "protocols after deletion")
	requireLen(t, store.ListProjects(), 0, "projects after deletion")
	requireLen(t, store.ListBreedingUnits(), 0, "breeding units after deletion")
	requireLen(t, store.ListProcedures(), 0, "procedures after deletion")
	requireLen(t, store.ListTreatments(), 0, "treatments after deletion")
	requireLen(t, store.ListObservations(), 0, "observations after deletion")
	requireLen(t, store.ListSamples(), 0, "samples after deletion")
	requireLen(t, store.ListPermits(), 0, "permits after deletion")
	requireLen(t, store.ListSupplyItems(), 0, "supplies after deletion")
}

func TestMemoryStoreViewReadOnly(t *testing.T) {
	store := memory.NewStore(nil)
	ctx := context.Background()
	var housing domain.HousingUnit
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		var err error
		housing, err = tx.CreateHousingUnit(domain.HousingUnit{Name: "Tank", FacilityID: "Lab", Capacity: 1})
		return err
	}); err != nil {
		t.Fatalf("create housing: %v", err)
	}

	if err := store.View(ctx, func(view domain.TransactionView) error {
		units := view.ListHousingUnits()
		if len(units) != 1 {
			t.Fatalf("expected single housing unit, got %d", len(units))
		}
		if _, ok := view.FindHousingUnit(housing.ID); !ok {
			t.Fatalf("expected to find housing unit %s", housing.ID)
		}
		return nil
	}); err != nil {
		t.Fatalf("view snapshot: %v", err)
	}
}

func TestUpdateHousingUnitValidation(t *testing.T) {
	store := memory.NewStore(nil)
	ctx := context.Background()
	var housing domain.HousingUnit
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		var err error
		housing, err = tx.CreateHousingUnit(domain.HousingUnit{Name: "Validated", FacilityID: "Lab", Capacity: 2})
		return err
	}); err != nil {
		t.Fatalf("create housing: %v", err)
	}

	_, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		_, err := tx.UpdateHousingUnit(housing.ID, func(h *domain.HousingUnit) error {
			h.Capacity = 0
			return nil
		})
		return err
	})
	if err == nil {
		t.Fatalf("expected capacity validation error on update")
	}
}

func TestResultMergeAndBlocking(t *testing.T) {
	res := domain.Result{}
	res.Merge(domain.Result{Violations: []domain.Violation{{Rule: "warn", Severity: domain.SeverityWarn}}})
	if res.HasBlocking() {
		t.Fatalf("expected no blocking violations yet")
	}
	res.Merge(domain.Result{Violations: []domain.Violation{{Rule: "block", Severity: domain.SeverityBlock}}})
	if !res.HasBlocking() {
		t.Fatalf("expected blocking violation")
	}
	err := domain.RuleViolationError{Result: res}
	if err.Error() == "" {
		t.Fatalf("expected error string")
	}
}

func TestRulesEngineAggregates(t *testing.T) {
	engine := domain.NewRulesEngine()
	engine.Register(staticRule{"warn", domain.SeverityWarn})
	engine.Register(staticRule{"block", domain.SeverityBlock})

	store := memory.NewStore(engine)
	ctx := context.Background()

	res, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		_, err := tx.CreateProject(domain.Project{Code: "P", Title: "domain.Project"})
		return err
	})
	if err == nil {
		t.Fatalf("expected transaction to fail due to blocking rule")
	}
	if !res.HasBlocking() {
		t.Fatalf("expected blocking violation from rule engine")
	}
}

type staticRule struct {
	name     string
	severity domain.Severity
}

func (r staticRule) Name() string { return r.name }

func (r staticRule) Evaluate(_ context.Context, _ domain.RuleView, _ []domain.Change) (domain.Result, error) {
	return domain.Result{Violations: []domain.Violation{{Rule: r.name, Severity: r.severity}}}, nil
}

func must[T any](t *testing.T, value T, err error) T {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return value
}

func mustNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func mustGet[T any](t *testing.T, value T, ok bool, msg string) T {
	return requireFound(t, value, ok, msg)
}

func requireFound[T any](t *testing.T, value T, ok bool, msg string) T {
	t.Helper()
	if !ok {
		t.Fatal(msg)
	}
	return value
}

func requireMissing(t *testing.T, ok bool, msg string) {
	t.Helper()
	if ok {
		t.Fatal(msg)
	}
}

func requireLen[T any](t *testing.T, items []T, expected int, msg string) {
	t.Helper()
	if len(items) != expected {
		t.Fatalf("%s: expected %d, got %d", msg, expected, len(items))
	}
}
