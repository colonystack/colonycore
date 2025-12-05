package memory

import (
	"colonycore/pkg/domain"
	"colonycore/pkg/domain/extension"
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

// TestSnapshotAllEntities exercises snapshotFromMemoryState and memoryStateFromSnapshot with
// at least one entry in every entity collection to raise coverage on cloning loops.
func TestSnapshotAllEntities(t *testing.T) {
	store := NewStore(domain.NewRulesEngine())
	ctx := context.Background()
	var housing domain.HousingUnit
	var organism domain.Organism
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		facility, err := tx.CreateFacility(domain.Facility{Name: "Lab"})
		if err != nil {
			return err
		}
		// Create project
		if _, err := tx.CreateProject(domain.Project{Code: "P1", Title: "Project", FacilityIDs: []string{facility.ID}}); err != nil {
			return err
		}
		// Create housing
		h, err := tx.CreateHousingUnit(domain.HousingUnit{Name: "H1", FacilityID: facility.ID, Capacity: 2})
		if err != nil {
			return err
		}
		housing = h
		// Create protocol
		p, err := tx.CreateProtocol(domain.Protocol{Code: "PR", Title: "Proto", MaxSubjects: 5})
		if err != nil {
			return err
		}
		_ = p // protocol used implicitly via references
		// Create cohort
		if _, err := tx.CreateCohort(domain.Cohort{Name: "C1"}); err != nil {
			return err
		}
		// Create organism with attributes
		organismInput := domain.Organism{Name: "Spec", Species: "Frog"}
		if err := organismInput.SetCoreAttributes(map[string]any{"color": "green"}); err != nil {
			return err
		}
		o, err := tx.CreateOrganism(organismInput)
		if err != nil {
			return err
		}
		organism = o
		// Create breeding unit referencing organism
		if _, err := tx.CreateBreedingUnit(domain.BreedingUnit{Name: "Pair", FemaleIDs: []string{o.ID}, MaleIDs: []string{"M"}, HousingID: &h.ID, ProtocolID: &p.ID}); err != nil {
			return err
		}
		// Create procedure referencing organism
		if _, err := tx.CreateProcedure(domain.Procedure{Name: "Check", Status: domain.ProcedureStatusScheduled, ScheduledAt: time.Now().UTC(), ProtocolID: p.ID, OrganismIDs: []string{o.ID}}); err != nil {
			return err
		}
		return nil
	}); err != nil {
		t.Fatalf("seed transaction: %v", err)
	}
	snap := store.ExportState()
	if len(snap.Organisms) == 0 || len(snap.Cohorts) == 0 || len(snap.Housing) == 0 || len(snap.Breeding) == 0 || len(snap.Procedures) == 0 || len(snap.Protocols) == 0 || len(snap.Projects) == 0 {
		t.Fatalf("expected populated snapshot: %+v", snap)
	}
	// Clear then re-import to exercise memoryStateFromSnapshot cloning for all maps.
	store.ImportState(Snapshot{})
	if len(store.ListOrganisms()) != 0 {
		t.Fatalf("expected cleared state")
	}
	store.ImportState(snap)
	if len(store.ListOrganisms()) != 1 {
		t.Fatalf("expected restored organism")
	}
	// Update housing unit success branch (mutator returns nil and capacity positive)
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		_, err := tx.UpdateHousingUnit(housing.ID, func(h *domain.HousingUnit) error {
			h.Capacity = 3
			return nil
		})
		return err
	}); err != nil {
		t.Fatalf("update housing success: %v", err)
	}
	// Ensure organism attributes remain isolated (deep copy validated indirectly by modifying snapshot copy)
	snapOrg := snap.Organisms[organism.ID]
	attrs := snapOrg.CoreAttributes()
	attrs["color"] = "blue"
	if store.ListOrganisms()[0].CoreAttributes()["color"] != "green" {
		t.Fatalf("expected deep copy isolation")
	}
}

func TestImportStateAppliesRelationshipMigrations(t *testing.T) {
	store := NewStore(nil)
	now := time.Now().UTC()
	const facilityKey = "fac-1"

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
			"house-1":       {Base: domain.Base{ID: "house-1"}, Name: "Housing", FacilityID: facilityKey, Capacity: 2},
			"house-invalid": {Base: domain.Base{ID: "house-invalid"}, Name: "Invalid", FacilityID: "missing", Capacity: 1},
		},
		Facilities: map[string]domain.Facility{
			facilityKey: {Base: domain.Base{ID: facilityKey}, Name: "Facility"},
		},
		Procedures: procedures,
		Treatments: map[string]domain.Treatment{
			"treat-1":   {Base: domain.Base{ID: "treat-1"}, Name: "Treat", ProcedureID: "proc-1", OrganismIDs: []string{"org-1", "missing", "org-1"}},
			"treat-bad": {Base: domain.Base{ID: "treat-bad"}, Name: "Bad", ProcedureID: "missing"},
		},
		Observations: map[string]domain.Observation{
			"obs-1": {Base: domain.Base{ID: "obs-1"}, ProcedureID: ptr("proc-1"), Observer: "Tech", RecordedAt: now},
			"obs-2": {Base: domain.Base{ID: "obs-2"}, ProcedureID: ptr("missing"), Observer: "Tech", RecordedAt: now},
		},
		Samples: map[string]domain.Sample{
			"sample-1": {Base: domain.Base{ID: "sample-1"}, Identifier: "S1", SourceType: "blood", FacilityID: facilityKey, OrganismID: ptr("org-1"), CollectedAt: now, Status: domain.SampleStatusStored, StorageLocation: "freezer"},
			"sample-2": {Base: domain.Base{ID: "sample-2"}, Identifier: "S2", SourceType: "blood", FacilityID: "missing", CollectedAt: now, Status: domain.SampleStatusStored, StorageLocation: "freezer"},
		},
		Protocols: protocols,
		Permits: map[string]domain.Permit{
			"permit-1": {Base: domain.Base{ID: "permit-1"}, PermitNumber: "P1", Authority: "Gov", Status: domain.PermitStatusApproved, ValidFrom: now, ValidUntil: now.AddDate(1, 0, 0), FacilityIDs: []string{facilityKey, "missing", facilityKey}, ProtocolIDs: []string{"prot-1", "missing"}},
		},
		Projects: map[string]domain.Project{
			"proj-1": {Base: domain.Base{ID: "proj-1"}, Code: "P1", Title: "Project", FacilityIDs: []string{facilityKey, facilityKey, "missing"}},
		},
		Supplies: map[string]domain.SupplyItem{
			"supply-1": {Base: domain.Base{ID: "supply-1"}, SKU: "SKU", Name: "Gloves", QuantityOnHand: 5, Unit: "box", FacilityIDs: []string{facilityKey, "missing"}, ProjectIDs: []string{"proj-1", "missing", "proj-1"}},
		},
	}

	store.ImportState(snapshot)

	if units := store.ListHousingUnits(); len(units) != 1 || units[0].ID != "house-1" {
		t.Fatalf("expected only valid housing to remain, got %+v", units)
	}

	facility, ok := store.GetFacility(facilityKey)
	if !ok {
		t.Fatalf("expected facility present")
	}
	if len(facility.HousingUnitIDs) != 1 || facility.HousingUnitIDs[0] != "house-1" {
		t.Fatalf("expected facility housing ids to be migrated, got %+v", facility.HousingUnitIDs)
	}
	if len(facility.ProjectIDs) != 1 || facility.ProjectIDs[0] != "proj-1" {
		t.Fatalf("expected facility project ids to be migrated, got %+v", facility.ProjectIDs)
	}

	if projects := store.ListProjects(); len(projects) != 1 || len(projects[0].FacilityIDs) != 1 || projects[0].FacilityIDs[0] != facilityKey {
		t.Fatalf("expected project facility ids filtered, got %+v", projects)
	}

	treatments := store.ListTreatments()
	if len(treatments) != 1 || len(treatments[0].OrganismIDs) != 1 || treatments[0].OrganismIDs[0] != "org-1" {
		t.Fatalf("expected valid treatment with deduped organism ids, got %+v", treatments)
	}

	observations := store.ListObservations()
	if len(observations) != 1 || observations[0].ObservationData() == nil {
		t.Fatalf("expected valid observation with data map initialised, got %+v", observations)
	}

	samples := store.ListSamples()
	if len(samples) != 1 || samples[0].FacilityID != facilityKey || samples[0].OrganismID == nil || *samples[0].OrganismID != "org-1" {
		t.Fatalf("expected only valid sample, got %+v", samples)
	}

	permits := store.ListPermits()
	if len(permits) != 1 || len(permits[0].FacilityIDs) != 1 || permits[0].FacilityIDs[0] != facilityKey || len(permits[0].ProtocolIDs) != 1 || permits[0].ProtocolIDs[0] != "prot-1" {
		t.Fatalf("expected permit lists filtered, got %+v", permits)
	}

	supplies := store.ListSupplyItems()
	if len(supplies) != 1 || len(supplies[0].FacilityIDs) != 1 || supplies[0].FacilityIDs[0] != facilityKey || len(supplies[0].ProjectIDs) != 1 || supplies[0].ProjectIDs[0] != "proj-1" {
		t.Fatalf("expected supply references filtered, got %+v", supplies)
	}
}

func ptr[T any](v T) *T {
	return &v
}

func TestMigrateSnapshotCleansDataVariants(t *testing.T) {
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
			"treatment-valid":  {Base: domain.Base{ID: "treatment-valid"}, Name: "Treat", Status: domain.TreatmentStatusPlanned, ProcedureID: "proc-keep", OrganismIDs: []string{"org-keep", "org-keep", "missing"}, CohortIDs: []string{"cohort-keep", "missing"}},
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
			"prot-keep": {Base: domain.Base{ID: "prot-keep"}, Code: "PR", Title: "Protocol", MaxSubjects: 5, Status: domain.ProtocolStatusApproved},
		},
		Permits: map[string]domain.Permit{
			"permit-valid": {Base: domain.Base{ID: "permit-valid"}, PermitNumber: "P", Authority: "Gov", Status: domain.PermitStatusApproved, ValidFrom: now, ValidUntil: now.Add(time.Hour), FacilityIDs: []string{facilityID, facilityID, "missing"}, ProtocolIDs: []string{"prot-keep", "missing"}},
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
	if obs := migrated.Observations["observation-valid"]; (&obs).ObservationData() == nil {
		t.Fatalf("expected observation data map to be initialised")
	}

	if len(migrated.Samples) != 1 {
		t.Fatalf("expected single valid sample, got %+v", migrated.Samples)
	}
	if sample := migrated.Samples["sample-valid"]; (&sample).SampleAttributes() == nil {
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

	if supply := migrated.Supplies["supply-valid"]; (&supply).SupplyAttributes() == nil {
		t.Fatalf("expected supply attributes map initialised")
	}
	if len(migrated.Supplies["supply-valid"].FacilityIDs) != 1 || migrated.Supplies["supply-valid"].FacilityIDs[0] != facilityID {
		t.Fatalf("expected supply facility IDs filtered")
	}
	if len(migrated.Supplies["supply-valid"].ProjectIDs) != 1 || migrated.Supplies["supply-valid"].ProjectIDs[0] != "project-valid" {
		t.Fatalf("expected supply project IDs filtered")
	}

	facility := migrated.Facilities[facilityID]
	if (&facility).EnvironmentBaselines() == nil {
		t.Fatalf("expected facility baselines map initialised")
	}
	if len(facility.HousingUnitIDs) != 1 || facility.HousingUnitIDs[0] != "housing-valid" {
		t.Fatalf("expected facility housing IDs populated, got %+v", facility.HousingUnitIDs)
	}
	if len(facility.ProjectIDs) != 1 || facility.ProjectIDs[0] != "project-valid" {
		t.Fatalf("expected facility project IDs populated, got %+v", facility.ProjectIDs)
	}
}

func TestObservationExtensionsSnapshotRoundTrip(t *testing.T) {
	container, err := extension.FromRaw(map[string]map[string]any{
		string(extension.HookObservationData): {
			extension.PluginCore.String(): map[string]any{"score": 3},
			"frog":                        map[string]any{"note": "warm"},
		},
	})
	if err != nil {
		t.Fatalf("build extension container: %v", err)
	}
	obs := domain.Observation{
		Base:     domain.Base{ID: "obs-1"},
		Observer: "Tech",
	}
	if err := obs.SetObservationExtensions(container); err != nil {
		t.Fatalf("seed observation extensions: %v", err)
	}
	snapshot := Snapshot{
		Procedures: map[string]domain.Procedure{
			"proc-1": {
				Base:        domain.Base{ID: "proc-1"},
				Name:        "Proc",
				Status:      domain.ProcedureStatusScheduled,
				ScheduledAt: time.Now().UTC(),
				ProtocolID:  "prot-1",
			},
		},
		Protocols: map[string]domain.Protocol{
			"prot-1": {
				Base:        domain.Base{ID: "prot-1"},
				Code:        "P1",
				Title:       "Protocol",
				MaxSubjects: 1,
				Status:      domain.ProtocolStatusApproved,
			},
		},
		Observations: map[string]domain.Observation{
			"obs-1": func(o domain.Observation) domain.Observation {
				o.ProcedureID = ptr("proc-1")
				return o
			}(obs),
		},
	}

	store := NewStore(nil)
	store.ImportState(snapshot)
	encoded, err := json.Marshal(store.ExportState())
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}

	var decoded Snapshot
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal snapshot: %v", err)
	}
	obs, ok := decoded.Observations["obs-1"]
	if !ok {
		t.Fatalf("expected observation present after round trip")
	}
	containerAfter, err := obs.ObservationExtensions()
	if err != nil {
		t.Fatalf("load observation extensions: %v", err)
	}
	hookPayload := containerAfter.Raw()[string(extension.HookObservationData)]
	if hookPayload == nil {
		t.Fatalf("expected hook payload preserved")
	}
	if frog, ok := hookPayload["frog"].(map[string]any); !ok || frog["note"] != "warm" {
		t.Fatalf("expected frog payload to survive, got %#v", frog)
	}
	if core, ok := hookPayload[extension.PluginCore.String()].(map[string]any); !ok || core["score"] != float64(3) {
		t.Fatalf("expected core payload to survive, got %#v", core)
	}
	if obs.ObservationData()["score"] != float64(3) {
		t.Fatalf("expected ObservationData to reflect core payload after round trip")
	}
}

func TestStateNormalizationDefaultsAndValidation(t *testing.T) {
	store := NewStore(nil)
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
		if housing.State != domain.HousingStateQuarantine {
			return fmt.Errorf("expected housing state defaulted to quarantine, got %q", housing.State)
		}
		if housing.Environment != domain.HousingEnvironmentTerrestrial {
			return fmt.Errorf("expected housing environment defaulted to terrestrial, got %q", housing.Environment)
		}
		if _, err := tx.CreateHousingUnit(domain.HousingUnit{Name: "BadEnv", FacilityID: facility.ID, Capacity: 1, Environment: domain.HousingEnvironment("invalid")}); err == nil {
			return fmt.Errorf("expected invalid housing environment to error")
		}
		if _, err := tx.CreateHousingUnit(domain.HousingUnit{Name: "BadState", FacilityID: facility.ID, Capacity: 1, Environment: domain.HousingEnvironmentHumid, State: domain.HousingState("invalid")}); err == nil {
			return fmt.Errorf("expected invalid housing state to error")
		}

		protocol, err := tx.CreateProtocol(domain.Protocol{Code: "P", Title: "Proto", MaxSubjects: 1})
		if err != nil {
			return err
		}
		if protocol.Status != domain.ProtocolStatusDraft {
			return fmt.Errorf("expected protocol status defaulted to draft, got %q", protocol.Status)
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
			return fmt.Errorf("expected permit status defaulted to draft, got %q", permit.Status)
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

func TestProcedureObservationSampleLifecycle(t *testing.T) {
	store := NewStore(nil)
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
