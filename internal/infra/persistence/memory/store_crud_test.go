package memory_test

import (
	"colonycore/internal/infra/persistence/memory"
	"colonycore/pkg/domain"
	"context"
	"fmt"
	"strings"
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

func strPtr(v string) *string {
	return &v
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
		projectVal, err := tx.CreateProject(domain.Project{Code: "PRJ-1", Title: "domain.Project"})
		project := must(t, projectVal, err)
		ids.projectID = project.ID

		facilityInput := domain.Facility{
			Name:         "Vivarium",
			Zone:         "Zone-A",
			AccessPolicy: "badge-required",
			ProjectIDs:   []string{ids.projectID},
		}
		facilityInput.SetEnvironmentBaselines(map[string]any{"temperature": "22C"})
		facilityVal, err := tx.CreateFacility(facilityInput)
		facility := must(t, facilityVal, err)
		ids.facilityID = facility.ID

		if _, err := tx.CreateHousingUnit(domain.HousingUnit{Name: "Invalid", FacilityID: ids.facilityID, Capacity: 0}); err == nil {
			return fmt.Errorf("expected capacity validation error")
		}

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
		organismAInput := domain.Organism{
			Name:       "Alpha",
			Species:    "Test Frog",
			Stage:      domain.StageJuvenile,
			ProjectID:  &projectPtr,
			ProtocolID: &protocolPtr,
			CohortID:   &cohortPtr,
			HousingID:  &housingPtr,
		}
		organismAInput.SetAttributes(attrs)
		organismAVal, err := tx.CreateOrganism(organismAInput)
		organismA := must(t, organismAVal, err)
		ids.organismAID = organismA.ID

		attrs["skin_color_index"] = 9

		organismBInput := domain.Organism{
			Name:     "Beta",
			Species:  "Test Toad",
			Stage:    domain.StageAdult,
			CohortID: &cohortPtr,
		}
		organismBVal, err := tx.CreateOrganism(organismBInput)
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
			Status:      domain.ProcedureStatusScheduled,
			ScheduledAt: time.Now().Add(time.Minute),
			ProtocolID:  ids.protocolID,
			OrganismIDs: []string{ids.organismAID, ids.organismBID},
		})
		procedure := must(t, procedureVal, err)
		ids.procedureID = procedure.ID

		treatmentVal, err := tx.CreateTreatment(domain.Treatment{
			Name:              "Dose",
			Status:            domain.TreatmentStatusPlanned,
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
		observationInput := domain.Observation{
			ProcedureID: &ids.procedureID,
			OrganismID:  &ids.organismAID,
			RecordedAt:  recorded,
			Observer:    "tech",
			Notes:       strPtr("baseline"),
		}
		observationInput.SetData(map[string]any{"score": 5})
		observationVal, err := tx.CreateObservation(observationInput)
		observation := must(t, observationVal, err)
		ids.observationID = observation.ID

		foundObservation, ok := tx.FindObservation(ids.observationID)
		requireFound(t, foundObservation, ok, "expected to find observation")
		if foundObservation.Observer != "tech" {
			t.Fatalf("unexpected observation returned from lookup")
		}

		custody := []domain.SampleCustodyEvent{{Actor: "tech", Location: "bench", Timestamp: time.Now().UTC(), Notes: strPtr("collected")}}
		sampleInput := domain.Sample{
			Identifier:      "S-1",
			SourceType:      "blood",
			OrganismID:      &ids.organismAID,
			FacilityID:      ids.facilityID,
			CollectedAt:     time.Now().UTC(),
			Status:          domain.SampleStatusStored,
			StorageLocation: "freezer-1",
			AssayType:       "PCR",
			ChainOfCustody:  custody,
		}
		sampleInput.SetAttributes(map[string]any{"volume_ml": 1.5})
		sampleVal, err := tx.CreateSample(sampleInput)
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
			Notes:             strPtr("initial issuance"),
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
		supplyInput := domain.SupplyItem{
			SKU:            "SKU-1",
			Name:           "Diet Blocks",
			Description:    strPtr("nutrient feed"),
			QuantityOnHand: 100,
			Unit:           "grams",
			LotNumber:      strPtr("LOT-44"),
			ExpiresAt:      &expiry,
			FacilityIDs:    []string{ids.facilityID},
			ProjectIDs:     []string{ids.projectID},
			ReorderLevel:   20,
		}
		supplyInput.SetAttributes(map[string]any{"supplier": "Acme"})
		supplyVal, err := tx.CreateSupplyItem(supplyInput)
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
		attrs := organism.AttributesMap()
		if attrs["skin_color_index"].(int) != 5 {
			t.Fatalf("expected cloned attributes value 5, got %v", attrs["skin_color_index"])
		}
		attrs["skin_color_index"] = 1
		if organism.AttributesMap()["skin_color_index"].(int) != 5 {
			t.Fatalf("expected organism attributes clone to remain unchanged")
		}
		copyCheckDone = true
	}
	if !copyCheckDone {
		t.Fatalf("organism %s not found in list", ids.organismAID)
	}

	refreshedVal, ok := store.GetOrganism(ids.organismAID)
	refreshed := mustGet(t, refreshedVal, ok, "expected organism to exist")
	if refreshed.AttributesMap()["skin_color_index"].(int) != 5 {
		t.Fatalf("expected store attributes to remain 5, got %v", refreshed.AttributesMap()["skin_color_index"])
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
			p.Description = strPtr(updatedDesc)
			return nil
		})
		mustNoErr(t, err)
		_, err = tx.UpdateProtocol(ids.protocolID, func(p *domain.Protocol) error {
			p.Description = strPtr(updatedDesc)
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
			p.Status = domain.ProcedureStatusCompleted
			return nil
		})
		mustNoErr(t, err)
		_, err = tx.UpdateFacility(ids.facilityID, func(f *domain.Facility) error {
			f.AccessPolicy = "biosafety-training"
			baselines := f.EnvironmentBaselinesMap()
			if baselines == nil {
				baselines = map[string]any{}
			}
			baselines["humidity"] = "55%"
			f.SetEnvironmentBaselines(baselines)
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
			o.Notes = strPtr(updatedDesc)
			data := o.DataMap()
			if data == nil {
				data = map[string]any{}
			}
			data["score"] = 6
			o.SetData(data)
			return nil
		})
		mustNoErr(t, err)
		_, err = tx.UpdateSample(ids.sampleID, func(s *domain.Sample) error {
			s.Status = domain.SampleStatusConsumed
			s.ChainOfCustody = append(s.ChainOfCustody, domain.SampleCustodyEvent{Actor: "lab", Location: "analysis", Timestamp: time.Now().UTC()})
			attrs := s.AttributesMap()
			if attrs == nil {
				attrs = map[string]any{}
			}
			attrs["volume_ml"] = 1.0
			s.SetAttributes(attrs)
			return nil
		})
		mustNoErr(t, err)
		_, err = tx.UpdatePermit(ids.permitID, func(p *domain.Permit) error {
			p.Notes = strPtr(updatedDesc)
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
	if facility.EnvironmentBaselinesMap()["humidity"] != "55%" {
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
	if samples[0].Status != domain.SampleStatusConsumed {
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
		mustNoErr(t, tx.DeleteObservation(ids.observationID))
		mustNoErr(t, tx.DeleteTreatment(ids.treatmentID))
		mustNoErr(t, tx.DeleteProcedure(ids.procedureID))
		mustNoErr(t, tx.DeleteBreedingUnit(ids.breedingID))
		mustNoErr(t, tx.DeleteSample(ids.sampleID))
		mustNoErr(t, tx.DeleteOrganism(ids.organismAID))
		mustNoErr(t, tx.DeleteOrganism(ids.organismBID))
		mustNoErr(t, tx.DeleteCohort(ids.cohortID))
		mustNoErr(t, tx.DeleteSupplyItem(ids.supplyItemID))
		mustNoErr(t, tx.DeleteProject(ids.projectID))
		mustNoErr(t, tx.DeleteHousingUnit(ids.housingID))
		mustNoErr(t, tx.DeletePermit(ids.permitID))
		mustNoErr(t, tx.DeleteProtocol(ids.protocolID))
		mustNoErr(t, tx.DeleteFacility(ids.facilityID))
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

func TestCreateHousingUnitRequiresExistingFacility(t *testing.T) {
	store := memory.NewStore(nil)
	ctx := context.Background()

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
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

func TestUpdateHousingUnitMovesFacilityLinks(t *testing.T) {
	store := memory.NewStore(nil)
	ctx := context.Background()
	var facilityA, facilityB domain.Facility
	var housing domain.HousingUnit

	mustNoErr(t, func() error {
		_, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
			var err error
			facilityA, err = tx.CreateFacility(domain.Facility{Name: "Vivarium-A"})
			if err != nil {
				return err
			}
			facilityB, err = tx.CreateFacility(domain.Facility{Name: "Vivarium-B"})
			if err != nil {
				return err
			}
			housing, err = tx.CreateHousingUnit(domain.HousingUnit{Name: "Shared", FacilityID: facilityA.ID, Capacity: 2})
			return err
		})
		return err
	}())

	mustNoErr(t, func() error {
		_, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
			_, err := tx.UpdateHousingUnit(housing.ID, func(h *domain.HousingUnit) error {
				h.FacilityID = facilityB.ID
				h.Environment = "humid"
				return nil
			})
			return err
		})
		return err
	}())

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		_, err := tx.UpdateHousingUnit(housing.ID, func(h *domain.HousingUnit) error {
			h.FacilityID = "missing"
			return nil
		})
		if err == nil {
			t.Fatalf("expected update error when facility is missing")
		}
		return nil
	}); err != nil {
		t.Fatalf("transaction unexpected error: %v", err)
	}

	fA, ok := store.GetFacility(facilityA.ID)
	if !ok {
		t.Fatalf("expected facility A to exist")
	}
	if len(fA.HousingUnitIDs) != 0 {
		t.Fatalf("expected facility A to have no housing references, got %+v", fA.HousingUnitIDs)
	}
	fB, ok := store.GetFacility(facilityB.ID)
	if !ok {
		t.Fatalf("expected facility B to exist")
	}
	if len(fB.HousingUnitIDs) != 1 || fB.HousingUnitIDs[0] != housing.ID {
		t.Fatalf("expected facility B to reference housing, got %+v", fB.HousingUnitIDs)
	}
}

func TestDeleteFacilityEnforcesReferences(t *testing.T) {
	store := memory.NewStore(nil)
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

	mustNoErr(t, func() error {
		_, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
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
				Status:            domain.PermitStatusActive,
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
		return err
	}())

	expectDeleteError := func(substr string) {
		t.Helper()
		if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
			return tx.DeleteFacility(facility.ID)
		}); err == nil || !strings.Contains(err.Error(), substr) {
			t.Fatalf("expected delete error containing %q, got %v", substr, err)
		}
	}

	expectDeleteError("housing unit")

	mustNoErr(t, func() error {
		_, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
			return tx.DeleteHousingUnit(housing.ID)
		})
		return err
	}())

	expectDeleteError("sample")

	mustNoErr(t, func() error {
		_, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
			return tx.DeleteSample(sample.ID)
		})
		return err
	}())

	expectDeleteError("project")

	mustNoErr(t, func() error {
		_, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
			return tx.DeleteProject(project.ID)
		})
		return err
	}())

	expectDeleteError("permit")

	mustNoErr(t, func() error {
		_, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
			return tx.DeletePermit(permit.ID)
		})
		return err
	}())

	expectDeleteError("supply item")

	mustNoErr(t, func() error {
		_, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
			return tx.DeleteSupplyItem(supply.ID)
		})
		return err
	}())

	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		return tx.DeleteFacility(facility.ID)
	}); err != nil {
		t.Fatalf("expected facility delete to succeed after removing references: %v", err)
	}
}

func TestRelationshipValidations(t *testing.T) {
	store := memory.NewStore(nil)
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

		if _, err := tx.CreateTreatment(domain.Treatment{Name: "ValidTreatment", Status: domain.TreatmentStatusPlanned, ProcedureID: procedure.ID, OrganismIDs: []string{organism.ID}}); err != nil {
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
			Status:       domain.PermitStatusPending,
			ValidFrom:    now,
			ValidUntil:   now.Add(time.Hour),
		}); err == nil {
			t.Fatalf("expected permit with missing protocol to fail")
		}
		if _, err := tx.CreatePermit(domain.Permit{
			PermitNumber:      "PERM-OK",
			Authority:         "Gov",
			Status:            domain.PermitStatusPending,
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

func TestUpdateSupplyItemDedupe(t *testing.T) {
	store := memory.NewStore(nil)
	ctx := context.Background()
	var (
		facility domain.Facility
		project  domain.Project
		supply   domain.SupplyItem
	)

	mustNoErr(t, func() error {
		_, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
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
		return err
	}())

	mustNoErr(t, func() error {
		_, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
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
		return err
	}())

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

func TestUpdatePermitDedupe(t *testing.T) {
	store := memory.NewStore(nil)
	ctx := context.Background()
	var (
		facility domain.Facility
		protocol domain.Protocol
		permit   domain.Permit
	)

	mustNoErr(t, func() error {
		_, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
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
		return err
	}())

	mustNoErr(t, func() error {
		_, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
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
		return err
	}())

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

func TestMemoryStoreViewReadOnly(t *testing.T) {
	store := memory.NewStore(nil)
	ctx := context.Background()
	var housing domain.HousingUnit
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		facility, err := tx.CreateFacility(domain.Facility{Name: "Lab"})
		if err != nil {
			return err
		}
		housing, err = tx.CreateHousingUnit(domain.HousingUnit{Name: "Tank", FacilityID: facility.ID, Capacity: 1})
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
		facility, err := tx.CreateFacility(domain.Facility{Name: "Lab"})
		if err != nil {
			return err
		}
		housing, err = tx.CreateHousingUnit(domain.HousingUnit{Name: "Validated", FacilityID: facility.ID, Capacity: 2})
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
